// Package sdk is the Go client for mNemo WORM data stores.
//
// Production usage — discovers topology from etcd and follows version flips:
//
//	client, err := sdk.NewClient(sdk.Config{
//	    EtcdEndpoints: []string{"localhost:2379"},
//	    Tenant:        "recsys",
//	    Store:         "catalog",
//	})
//	defer client.Close()
//
//	val, err := client.Get(ctx, key)
//	results, err := client.BatchGet(ctx, keys)
//
// The SDK watches etcd's activeVersion key and atomically rebuilds its
// shard→pod route table on every promote/rollback — etcd stays off the read
// hot path (routing uses the cached assignment).
package sdk

import (
	"context"
	"errors"
	"io"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ErrKeyNotFound is returned by Get when a key is absent from the store.
var ErrKeyNotFound = errors.New("mnemo: key not found")

// ErrNoHealthyPod is returned when a shard has no assigned pod (topology not
// loaded, or the version has incomplete coverage).
var ErrNoHealthyPod = errors.New("mnemo: no healthy pod for shard")

// ErrNoEndpoints is returned by NewClient when no etcd endpoints are configured.
var ErrNoEndpoints = errors.New("mnemo: no etcd endpoints configured")

// Config configures a mNemo client.
type Config struct {
	EtcdEndpoints []string // etcd endpoints for topology discovery (activeVersion watch)
	Tenant        string
	Store         string

	// Kubernetes headless-service DNS resolution. Each shard resolves to
	// {tenant}-{store}-shard-{N}.{Namespace}.svc.{DNSZone}:{Port}.
	Namespace          string        // K8s namespace (default "default")
	DNSZone            string        // cluster DNS zone (default "cluster.local")
	Port               int           // read server TCP port (default 9091)
	DNSRefreshInterval time.Duration // background re-resolve cadence (default 30s)

	ConnsPerPod int // TCP pool size per pod (default 4)
	TimeoutMs   int // per-request timeout (default 100ms)
}

func (c *Config) applyDefaults() {
	if c.ConnsPerPod == 0 {
		c.ConnsPerPod = 4
	}
	if c.TimeoutMs == 0 {
		c.TimeoutMs = 100
	}
	if c.Port == 0 {
		c.Port = 9091
	}
	if c.DNSZone == "" {
		c.DNSZone = "cluster.local"
	}
	if c.Namespace == "" {
		c.Namespace = "default"
	}
	if c.DNSRefreshInterval == 0 {
		c.DNSRefreshInterval = 30 * time.Second
	}
}

// Result is the outcome for a single key in a BatchGet.
type Result struct {
	Key   []byte
	Value []byte // nil if not found
	Err   error
}

// Client is the mNemo SDK entry point.
type Client struct {
	config     Config
	router     *Router
	pool       *ConnPool
	cancel     context.CancelFunc // stops the topology watcher
	etcdCloser io.Closer          // nil for a direct client
}

// newEtcdClient is the etcd-client constructor, indirected through a package
// var so tests can inject a dial failure.
var newEtcdClient = func(endpoints []string) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
}

// NewClient creates an etcd-backed client and starts the topology watcher.
func NewClient(config Config) (*Client, error) {
	if len(config.EtcdEndpoints) == 0 {
		return nil, ErrNoEndpoints
	}
	cli, err := newEtcdClient(config.EtcdEndpoints)
	if err != nil {
		return nil, err
	}
	return newClientWithEtcd(config, cli, cli)
}

// newClientWithEtcd wires the resolver, router, pool, and topology watcher
// around an injected etcd client. Separated from NewClient so tests can supply
// an embedded-etcd or mock client.
func newClientWithEtcd(config Config, etcd EtcdClient, closer io.Closer) (*Client, error) {
	config.applyDefaults()

	resolver := NewDNSResolver(DNSConfig{
		Tenant:    config.Tenant,
		Store:     config.Store,
		Namespace: config.Namespace,
		DNSZone:   config.DNSZone,
		Port:      config.Port,
		Interval:  config.DNSRefreshInterval,
	})
	router := NewRouter(resolver)
	pool := NewConnPool(config.ConnsPerPod)

	// After each DNS refresh: clear transient unhealthy marks and prune pools
	// for pods that dropped out of DNS (scale-down / no-longer-warm).
	resolver.OnRefresh(func() {
		router.ClearUnhealthy()
		pool.Prune(resolver.AllAddrs())
	})

	watcher := NewTopologyWatcher(etcd, router, resolver, config.Tenant, config.Store)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = watcher.Run(ctx) }()
	go resolver.Run(ctx)

	return &Client{
		config:     config,
		router:     router,
		pool:       pool,
		cancel:     cancel,
		etcdCloser: closer,
	}, nil
}

// NewDirectClient creates a client that talks to one endpoint with no etcd or
// DNS — for local development and testing. Single shard, single pod.
func NewDirectClient(addr string, connsPerPod int) *Client {
	cfg := Config{ConnsPerPod: connsPerPod}
	cfg.applyDefaults()

	router := NewRouter(NewStaticResolver(map[uint32][]string{0: {addr}}))
	router.SetShardCount(1)

	return &Client{
		config: cfg,
		router: router,
		pool:   NewConnPool(cfg.ConnsPerPod),
		cancel: func() {},
	}
}

func (c *Client) timeout() time.Duration {
	return time.Duration(c.config.TimeoutMs) * time.Millisecond
}

// Get performs a single-key point lookup.
func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	shardID := c.router.ShardFor(key)
	pod, err := c.router.PodFor(shardID)
	if err != nil {
		return nil, err
	}

	conn, err := c.pool.Get(pod)
	if err != nil {
		c.router.MarkUnhealthy(pod)
		return nil, err
	}

	val, err := conn.SingleLookup(ctx, key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			c.pool.Put(pod, conn) // healthy connection, just a miss
			return nil, err
		}
		_ = conn.Close() // broken connection — discard
		c.router.MarkUnhealthy(pod)
		return nil, err
	}
	c.pool.Put(pod, conn)
	return val, nil
}

// BatchGet performs a multi-key scatter-gather. Keys are grouped by shard,
// fanned out in parallel, and merged in input order.
func (c *Client) BatchGet(ctx context.Context, keys [][]byte) ([]Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	return scatterGather(ctx, c, keys)
}

// StringGet performs a single string-key point lookup using opcode 0x03.
// The key is a variable-length UTF-8 byte slice (e.g. BuildStringKey output).
func (c *Client) StringGet(ctx context.Context, key []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	shardID := c.router.ShardFor(key)
	pod, err := c.router.PodFor(shardID)
	if err != nil {
		return nil, err
	}

	conn, err := c.pool.Get(pod)
	if err != nil {
		c.router.MarkUnhealthy(pod)
		return nil, err
	}

	val, err := conn.StringSingleLookup(ctx, key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			c.pool.Put(pod, conn)
			return nil, err
		}
		_ = conn.Close()
		c.router.MarkUnhealthy(pod)
		return nil, err
	}
	c.pool.Put(pod, conn)
	return val, nil
}

// StringBatchGet performs a multi-key scatter-gather using string-key opcode 0x04.
// Keys are variable-length UTF-8 byte slices.
func (c *Client) StringBatchGet(ctx context.Context, keys [][]byte) ([]Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	return stringScatterGather(ctx, c, keys)
}

// Close stops the watcher, closes pooled connections, and releases etcd.
func (c *Client) Close() error {
	c.cancel()
	c.pool.Close()
	if c.etcdCloser != nil {
		return c.etcdCloser.Close()
	}
	return nil
}
