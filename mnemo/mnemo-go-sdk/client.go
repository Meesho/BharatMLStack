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
// The SDK watches etcd's activeVersion key and pod registrations, atomically
// rebuilding its shard→pod route table on every promote/rollback/scale event.
// Routing uses the assignment map from VersionMeta (works on K8s and VM
// deployments); DNS resolution is a fallback for K8s.
package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
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

	// Connection pool settings. These are overridden by ClientConfig from the
	// control plane when available (zero = use control plane value or default).
	ConnsPerPod int // TCP pool ceiling per pod (default 4) — legacy alias for PoolConfig.MaxPerPod
	TimeoutMs   int // per-request timeout (default 100ms) — legacy alias for PoolConfig.RequestTimeoutMs

	// PoolConfig allows full control over connection pool tuning. When set,
	// ConnsPerPod/TimeoutMs are ignored. When nil, the SDK builds a PoolConfig
	// from ConnsPerPod + TimeoutMs + any ClientConfig fetched from the control plane.
	Pool *PoolConfig

	// ClientConfig can be provided explicitly to skip the control plane fetch.
	// When nil, the SDK reads it from etcd on init (best-effort — missing config
	// means all defaults).
	ClientConfig *model.ClientConfig

	// Timing is an optional callback for latency metrics (nil-safe). The caller
	// provides the implementation (typically Datadog StatsD via pkg/metric).
	Timing func(name string, value time.Duration, tags []string)

	// Count is an optional callback for count metrics (nil-safe).
	Count func(name string, value int64, tags []string)
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

// buildPoolConfig merges Config + ClientConfig into a PoolConfig.
func (c *Config) buildPoolConfig() PoolConfig {
	if c.Pool != nil {
		pc := *c.Pool
		pc.applyDefaults()
		return pc
	}

	pc := PoolConfig{
		MaxPerPod: c.ConnsPerPod,
	}

	// Overlay ClientConfig from control plane (non-zero fields win).
	if cc := c.ClientConfig; cc != nil {
		// Per-request deadline: the control plane value wins over the default.
		// Applied by Get/BatchGet via c.timeout(); for BatchGet it bounds the
		// whole scatter-gather, so size it to cover the largest expected batch.
		if cc.RequestTimeoutMs > 0 {
			c.TimeoutMs = cc.RequestTimeoutMs
		}
		if cc.ConnectTimeoutMs > 0 {
			pc.DialTimeout = time.Duration(cc.ConnectTimeoutMs) * time.Millisecond
		}
		if cc.KeepAliveIntervalMs > 0 {
			pc.KeepAliveInterval = time.Duration(cc.KeepAliveIntervalMs) * time.Millisecond
		}
		if cc.KeepAliveTimeoutMs > 0 {
			pc.KeepAliveTimeout = time.Duration(cc.KeepAliveTimeoutMs) * time.Millisecond
		}
		if cc.IdleTimeoutMs > 0 {
			pc.IdleTimeout = time.Duration(cc.IdleTimeoutMs) * time.Millisecond
		}
		if cc.IdleCheckIntervalMs > 0 {
			pc.IdleCheckInterval = time.Duration(cc.IdleCheckIntervalMs) * time.Millisecond
		}
		if cc.MinConnsPerPod > 0 {
			pc.MinPerPod = cc.MinConnsPerPod
		}
		if cc.MaxConnsPerPod > 0 {
			pc.MaxPerPod = cc.MaxConnsPerPod
		}
		if cc.DNSRefreshIntervalMs > 0 {
			c.DNSRefreshInterval = time.Duration(cc.DNSRefreshIntervalMs) * time.Millisecond
		}
	}

	pc.applyDefaults()
	return pc
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

// EtcdKeepAliveDefaults are used when ClientConfig doesn't specify etcd
// keepalive values. 60s is safe for most etcd servers / proxies (default
// server MinTime is 5s, but LBs often enforce higher).
const (
	defaultEtcdKeepAliveTime    = 60 * time.Second
	defaultEtcdKeepAliveTimeout = 10 * time.Second
)

// newEtcdClient is the etcd-client constructor, indirected through a package
// var so tests can inject a dial failure.
var newEtcdClient = func(endpoints []string, keepAliveTime, keepAliveTimeout time.Duration) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    keepAliveTime,
		DialKeepAliveTimeout: keepAliveTimeout,
	})
}

// NewClient creates an etcd-backed client and starts the topology watcher.
func NewClient(config Config) (*Client, error) {
	if len(config.EtcdEndpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Resolve etcd gRPC keepalive: explicit ClientConfig > defaults (60s).
	kaTime := defaultEtcdKeepAliveTime
	kaTimeout := defaultEtcdKeepAliveTimeout
	if cc := config.ClientConfig; cc != nil {
		if cc.EtcdKeepAliveTimeMs > 0 {
			kaTime = time.Duration(cc.EtcdKeepAliveTimeMs) * time.Millisecond
		}
		if cc.EtcdKeepAliveTimeoutMs > 0 {
			kaTimeout = time.Duration(cc.EtcdKeepAliveTimeoutMs) * time.Millisecond
		}
	}

	cli, err := newEtcdClient(config.EtcdEndpoints, kaTime, kaTimeout)
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

	// Best-effort: fetch ClientConfig from control plane.
	if config.ClientConfig == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cc := fetchClientConfig(ctx, etcd, config.Tenant, config.Store)
		cancel()
		config.ClientConfig = cc // nil is fine — all defaults
	}

	poolCfg := config.buildPoolConfig()
	pool := NewConnPoolWithConfig(poolCfg)

	dnsResolver := NewDNSResolver(DNSConfig{
		Tenant:    config.Tenant,
		Store:     config.Store,
		Namespace: config.Namespace,
		DNSZone:   config.DNSZone,
		Port:      config.Port,
		Interval:  config.DNSRefreshInterval,
	})

	assignRes := NewAssignmentResolver()

	// The router uses the assignment resolver as primary (works for K8s + VM).
	// DNS resolver is kept for backwards-compat: when assignment map is empty
	// for a shard, the router falls back to DNS.
	router := NewRouter(NewFallbackResolver(assignRes, dnsResolver))

	// After each DNS refresh: clear transient unhealthy marks and prune pools
	// for pods that dropped out of DNS (scale-down / no-longer-warm).
	dnsResolver.OnRefresh(func() {
		router.ClearUnhealthy()
		pool.Prune(assignRes.AllAddrs())
	})

	watcher := NewTopologyWatcher(etcd, router, dnsResolver, config.Tenant, config.Store)
	watcher.SetAssignmentResolver(assignRes)

	// Wire metrics callbacks into pool and topology watcher.
	bt := []string{"tenant:" + config.Tenant, "store:" + config.Store}
	pool.SetMetrics(config.Timing, config.Count, bt)
	watcher.SetMetrics(config.Timing, config.Count, bt)

	// Determine warm-up connection count.
	warmUp := poolCfg.MinPerPod
	if config.ClientConfig != nil && config.ClientConfig.WarmUpOnTopologyChange != nil && !*config.ClientConfig.WarmUpOnTopologyChange {
		warmUp = 0
	}
	watcher.SetPoolForWarmUp(pool, warmUp)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = watcher.Run(ctx) }()
	go dnsResolver.Run(ctx)

	return &Client{
		config:     config,
		router:     router,
		pool:       pool,
		cancel:     cancel,
		etcdCloser: closer,
	}, nil
}

// fetchClientConfig reads the ClientConfig from etcd. Returns nil on any error.
func fetchClientConfig(ctx context.Context, etcd EtcdClient, tenant, store string) *model.ClientConfig {
	resp, err := etcd.Get(ctx, model.ClientConfigPath(tenant, store))
	if err != nil || len(resp.Kvs) == 0 {
		return nil
	}
	var cc model.ClientConfig
	if err := json.Unmarshal(resp.Kvs[0].Value, &cc); err != nil {
		return nil
	}
	return &cc
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
	start := time.Now()
	status := "error"
	defer func() {
		tags := c.opTags("single", status)
		c.emitTiming(MetricRequestLatency, time.Since(start), tags)
		c.emitCount(MetricRequestCount, 1, tags)
	}()

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
			status = "miss"
			return nil, err
		}
		_ = conn.Close() // broken connection — discard
		c.router.MarkUnhealthy(pod)
		return nil, err
	}
	c.pool.Put(pod, conn)
	status = "hit"
	return val, nil
}

// BatchGet performs a multi-key scatter-gather. Keys are grouped by shard,
// fanned out in parallel, and merged in input order.
func (c *Client) BatchGet(ctx context.Context, keys [][]byte) ([]Result, error) {
	start := time.Now()
	status := "ok"
	defer func() {
		tags := c.opTags("batch", status)
		c.emitTiming(MetricRequestLatency, time.Since(start), tags)
		c.emitCount(MetricRequestCount, 1, tags)
		c.emitCount(MetricBatchKeys, int64(len(keys)), c.baseTags())
	}()

	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	results, err := scatterGather(ctx, c, keys)
	if err != nil {
		status = "error"
	}
	return results, err
}

// StringGet performs a single string-key point lookup using opcode 0x03.
// The key is a variable-length UTF-8 byte slice (e.g. BuildStringKey output).
func (c *Client) StringGet(ctx context.Context, key []byte) ([]byte, error) {
	start := time.Now()
	status := "error"
	defer func() {
		tags := c.opTags("string_single", status)
		c.emitTiming(MetricRequestLatency, time.Since(start), tags)
		c.emitCount(MetricRequestCount, 1, tags)
	}()

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
			status = "miss"
			return nil, err
		}
		_ = conn.Close()
		c.router.MarkUnhealthy(pod)
		return nil, err
	}
	c.pool.Put(pod, conn)
	status = "hit"
	return val, nil
}

// StringBatchGet performs a multi-key scatter-gather using string-key opcode 0x04.
// Keys are variable-length UTF-8 byte slices.
func (c *Client) StringBatchGet(ctx context.Context, keys [][]byte) ([]Result, error) {
	start := time.Now()
	status := "ok"
	defer func() {
		tags := c.opTags("string_batch", status)
		c.emitTiming(MetricRequestLatency, time.Since(start), tags)
		c.emitCount(MetricRequestCount, 1, tags)
		c.emitCount(MetricBatchKeys, int64(len(keys)), c.baseTags())
	}()

	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	results, err := stringScatterGather(ctx, c, keys)
	if err != nil {
		status = "error"
	}
	return results, err
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
