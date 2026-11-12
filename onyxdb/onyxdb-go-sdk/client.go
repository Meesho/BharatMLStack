// Package sdk is the Go client for OnyxDB WORM data stores.
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
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// ErrKeyNotFound is returned by Get when a key is absent from the store.
var ErrKeyNotFound = errors.New("onyxdb: key not found")

// ErrNoHealthyPod is returned when a shard has no assigned pod (topology not
// loaded, or the version has incomplete coverage).
var ErrNoHealthyPod = errors.New("onyxdb: no healthy pod for shard")

// ErrNoEndpoints is returned by NewClient when no etcd endpoints are configured.
var ErrNoEndpoints = errors.New("onyxdb: no etcd endpoints configured")

// ErrDialBackoff is returned by ConnPool.GetContext, and so by any lookup, when
// the pod has no idle connection and the pool will not dial it now: its last
// dial failed and it is backing off (PoolConfig.DialBackoffBase), or this
// client's dial rate to it is used up (PoolConfig.DialRatePerPod). The request
// fails at once; nothing is sent.
var ErrDialBackoff = errors.New("onyxdb: dial backoff")

// Config configures a OnyxDB client.
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

	// UnhealthyBackoffBase and UnhealthyBackoffMax bound how long a pod that
	// failed a request is kept out of routing: the base after the first
	// failure in a row, doubling per failure up to the max, ±20% jitter, ended
	// by a successful request (defaults 1s and 30s). Overridden by clientConfig
	// unhealthyBackoffBaseMs / unhealthyBackoffMaxMs.
	UnhealthyBackoffBase time.Duration
	UnhealthyBackoffMax  time.Duration
	// ShardFailFast fails a shard's keys at once with ErrShardUnavailable while
	// every pod of the shard is inside its unhealthy window, and sends a pod
	// past its window one probe at a time before it takes traffic again. nil
	// means true; false keeps the old best-effort routing to a marked pod.
	// Overridden by clientConfig shardFailFast.
	ShardFailFast *bool

	// Timing is an optional callback for latency metrics (nil-safe). The caller
	// provides the implementation (typically Datadog StatsD via pkg/metric).
	Timing func(name string, value time.Duration, tags []string)

	// Count is an optional callback for count metrics (nil-safe).
	Count func(name string, value int64, tags []string)

	// ext is the part of the etcd clientConfig document model.ClientConfig
	// cannot carry (clientConfigExt); set by NewClient's fetch only.
	ext *clientConfigExt
}

// clientConfigExt is the part of a store's etcd clientConfig document that
// controlplane/model.ClientConfig does not carry yet: the redial limits,
// unhealthy backoff and shard fail-fast
// (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3 B4). It is
// decoded here, from the same document, rather than through model.ClientConfig
// because the SDK is released on its own and builds against the last released
// control plane module, which predates these keys (as podRegistration in
// topology.go does for slow start). Zero or absent means the default. The JSON
// keys are the etcd contract; TestClientConfigExt_EtcdKeysArePinned pins them.
type clientConfigExt struct {
	// DialRatePerPod → PoolConfig.DialRatePerPod (negative: no rate limit).
	DialRatePerPod float64 `json:"dialRatePerPod,omitempty"`
	// DialBackoffBaseMs, DialBackoffMaxMs → PoolConfig.DialBackoffBase / Max
	// (a negative base turns dial backoff off).
	DialBackoffBaseMs int `json:"dialBackoffBaseMs,omitempty"`
	DialBackoffMaxMs  int `json:"dialBackoffMaxMs,omitempty"`
	// UnhealthyBackoffBaseMs, UnhealthyBackoffMaxMs →
	// Config.UnhealthyBackoffBase / Max.
	UnhealthyBackoffBaseMs int `json:"unhealthyBackoffBaseMs,omitempty"`
	UnhealthyBackoffMaxMs  int `json:"unhealthyBackoffMaxMs,omitempty"`
	// ShardFailFast → Config.ShardFailFast; absent means true.
	ShardFailFast *bool `json:"shardFailFast,omitempty"`
}

// shardFailFast resolves Config.ShardFailFast: unset means on.
func (c *Config) shardFailFast() bool {
	return c.ShardFailFast == nil || *c.ShardFailFast
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
	c.overlayClientConfigExt(&pc)

	pc.applyDefaults()
	return pc
}

// overlayClientConfigExt applies the clientConfig keys model.ClientConfig does
// not carry (clientConfigExt): set keys win, as every clientConfig key does. A
// negative dial key turns that limit off; a non-positive unhealthy key is
// ignored.
func (c *Config) overlayClientConfigExt(pc *PoolConfig) {
	x := c.ext
	if x == nil {
		return
	}
	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }
	if x.DialRatePerPod != 0 {
		pc.DialRatePerPod = x.DialRatePerPod
	}
	if x.DialBackoffBaseMs != 0 {
		pc.DialBackoffBase = ms(x.DialBackoffBaseMs)
	}
	if x.DialBackoffMaxMs > 0 {
		pc.DialBackoffMax = ms(x.DialBackoffMaxMs)
	}
	if x.UnhealthyBackoffBaseMs > 0 {
		c.UnhealthyBackoffBase = ms(x.UnhealthyBackoffBaseMs)
	}
	if x.UnhealthyBackoffMaxMs > 0 {
		c.UnhealthyBackoffMax = ms(x.UnhealthyBackoffMaxMs)
	}
	if x.ShardFailFast != nil {
		on := *x.ShardFailFast
		c.ShardFailFast = &on
	}
}

// Result is the outcome for a single key in a BatchGet.
type Result struct {
	Key   []byte
	Value []byte // nil if not found
	Err   error
}

// Client is the OnyxDB SDK entry point.
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

	// Best-effort: fetch ClientConfig from control plane.
	if config.ClientConfig == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cc, ext := fetchClientConfig(ctx, etcd, config.Tenant, config.Store)
		cancel()
		config.ClientConfig = cc // nil is fine — all defaults
		config.ext = ext
	}

	poolCfg := config.buildPoolConfig()
	pool := NewConnPoolWithConfig(poolCfg)

	assignRes := NewAssignmentResolver()

	dnsResolver := NewDNSResolver(DNSConfig{
		Tenant:    config.Tenant,
		Store:     config.Store,
		Namespace: config.Namespace,
		DNSZone:   config.DNSZone,
		Port:      config.Port,
		Interval:  config.DNSRefreshInterval,
		// DNS is the fallback: only resolve shards the assignment map does not
		// already cover. Keeps the refresh tick (and the ClearUnhealthy/Prune
		// it drives) running without probing a Service that need not exist.
		Skip: func(shardID uint32) bool { return len(assignRes.Resolve(shardID)) > 0 },
	})

	// The router uses the assignment resolver as primary (works for K8s + VM).
	// DNS resolver is kept for backwards-compat: when assignment map is empty
	// for a shard, the router falls back to DNS.
	router := NewRouter(NewFallbackResolver(assignRes, dnsResolver))
	config.configureRouter(router)

	// After each DNS refresh (the DNS tick, and every topology reload): drop
	// unhealthy marks whose window has passed or whose pod left, and prune
	// pools for pods that dropped out (scale-down / no-longer-warm).
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
	router.SetMetrics(config.Count, bt)
	if count := config.Count; count != nil {
		router.SetRampPickHook(func() { count(MetricSlowStartPick, 1, bt) })
	}

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

// configureRouter applies the unhealthy backoff and shard fail-fast settings
// to r. Call after buildPoolConfig, which overlays them from clientConfig and
// settles the request timeout: a probe counts as in flight for the request
// timeout plus a second, so its outcome always lands inside the lease.
func (c *Config) configureRouter(r *Router) {
	r.SetUnhealthyBackoff(c.UnhealthyBackoffBase, c.UnhealthyBackoffMax)
	r.SetShardFailFast(c.shardFailFast())
	r.mu.Lock()
	r.probeLease = time.Duration(c.TimeoutMs)*time.Millisecond + time.Second
	r.mu.Unlock()
}

// fetchClientConfig reads the store's clientConfig document from etcd once and
// decodes it twice: into model.ClientConfig without the redial keys, and into
// clientConfigExt for the redial keys. Both are nil when the document is
// absent, unreadable or not JSON. A redial key of the wrong type costs only the
// redial keys (ext nil: their defaults apply): the control plane's
// model.ClientConfig carries them too, and decoding them there would fail the
// whole document.
func fetchClientConfig(ctx context.Context, etcd EtcdClient, tenant, store string) (*model.ClientConfig, *clientConfigExt) {
	resp, err := etcd.Get(ctx, model.ClientConfigPath(tenant, store))
	if err != nil || len(resp.Kvs) == 0 {
		return nil, nil
	}
	var cc model.ClientConfig
	if err := json.Unmarshal(withoutRedialKeys(resp.Kvs[0].Value), &cc); err != nil {
		return nil, nil
	}
	var ext clientConfigExt
	if err := json.Unmarshal(resp.Kvs[0].Value, &ext); err != nil {
		log.Warn().Err(err).Str("tenant", tenant).Str("store", store).
			Msg("clientConfig: redial keys do not parse, using their defaults")
		return &cc, nil
	}
	return &cc, &ext
}

// redialKeys are clientConfigExt's JSON keys, read from its tags so the two
// cannot drift.
var redialKeys = func() []string {
	t := reflect.TypeOf(clientConfigExt{})
	keys := make([]string, t.NumField())
	for i := range keys {
		keys[i], _, _ = strings.Cut(t.Field(i).Tag.Get("json"), ",")
	}
	return keys
}()

// withoutRedialKeys returns the clientConfig document with the redial keys
// removed, matched case-insensitively as encoding/json matches them. A document
// that is not a JSON object is returned as is, for the caller's decode to
// reject.
func withoutRedialKeys(doc []byte) []byte {
	var fields map[string]json.RawMessage
	if json.Unmarshal(doc, &fields) != nil {
		return doc
	}
	for k := range fields {
		if slices.ContainsFunc(redialKeys, func(r string) bool { return strings.EqualFold(k, r) }) {
			delete(fields, k)
		}
	}
	out, _ := json.Marshal(fields) // re-marshalling decoded RawMessages cannot fail
	return out
}

// NewDirectClient creates a client that talks to one endpoint with no etcd or
// DNS — for local development and testing. Single shard, single pod.
func NewDirectClient(addr string, connsPerPod int) *Client {
	cfg := Config{ConnsPerPod: connsPerPod}
	cfg.applyDefaults()

	router := NewRouter(NewStaticResolver(map[uint32][]string{0: {addr}}))
	router.SetShardCount(1)
	cfg.configureRouter(router)

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

// checkoutFailed reports a request that got no connection to pod. The pod is
// marked unhealthy, unless only this client's own dial rate refused the dial:
// that says nothing about the pod, so it only ends a probe without a verdict.
func (c *Client) checkoutFailed(pod string, err error) {
	if errors.Is(err, errDialRateLimited) {
		c.router.endProbe(pod)
		return
	}
	c.router.MarkUnhealthy(pod)
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

	conn, err := c.pool.GetContext(ctx, pod)
	if err != nil {
		c.checkoutFailed(pod, err)
		return nil, err
	}

	val, err := conn.SingleLookup(ctx, key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			c.pool.Put(pod, conn) // healthy connection, just a miss
			c.router.MarkHealthy(pod)
			status = "miss"
			return nil, err
		}
		c.pool.Recycle(pod, conn) // drained for reuse if it only timed out, else closed
		c.router.MarkUnhealthy(pod)
		return nil, err
	}
	c.pool.Put(pod, conn)
	c.router.MarkHealthy(pod)
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

	conn, err := c.pool.GetContext(ctx, pod)
	if err != nil {
		c.checkoutFailed(pod, err)
		return nil, err
	}

	val, err := conn.StringSingleLookup(ctx, key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			c.pool.Put(pod, conn)
			c.router.MarkHealthy(pod)
			status = "miss"
			return nil, err
		}
		c.pool.Recycle(pod, conn)
		c.router.MarkUnhealthy(pod)
		return nil, err
	}
	c.pool.Put(pod, conn)
	c.router.MarkHealthy(pod)
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
