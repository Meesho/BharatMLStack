package sdk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// staticClient builds a Client backed by a fixed assignment (no etcd/DNS).
func staticClient(shardCount uint32, assignment map[uint32][]string) *Client {
	r := NewRouter(NewStaticResolver(assignment))
	r.SetShardCount(shardCount)
	return &Client{
		config: Config{ConnsPerPod: 4, TimeoutMs: 100},
		router: r,
		pool:   NewConnPool(4),
		cancel: func() {},
	}
}

// startFakeServer runs a fakeServer-backed TCP listener and returns its addr.
func startFakeServer(t *testing.T, data map[string][]byte) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go fakeServer(conn, data)
		}
	}()
	return ln.Addr().String()
}

func TestScatterGather_OrderPreserved(t *testing.T) {
	// 3 keys, single shard, single pod.
	k1, k2, k3 := key12("k1"), key12("k2"), key12("k3")
	addr := startFakeServer(t, map[string][]byte{
		string(k1): []byte("v1"),
		string(k3): []byte("v3"),
		// k2 is a miss
	})

	c := NewDirectClient(addr, 4)
	defer c.Close()

	results, err := c.BatchGet(context.Background(), [][]byte{k1, k2, k3})
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, []byte("v1"), results[0].Value)
	assert.Nil(t, results[1].Value) // miss
	assert.Equal(t, []byte("v3"), results[2].Value)
	// Keys preserved in order
	assert.Equal(t, k1, results[0].Key)
	assert.Equal(t, k2, results[1].Key)
	assert.Equal(t, k3, results[2].Key)
}

func TestScatterGather_MultiShard(t *testing.T) {
	// Two shards on two fake servers. Build the router manually.
	kA := key12("alpha")
	kB := key12("bravo")
	addr0 := startFakeServer(t, map[string][]byte{string(kA): []byte("va")})
	addr1 := startFakeServer(t, map[string][]byte{string(kB): []byte("vb")})

	c := staticClient(2, map[uint32][]string{0: {addr0}, 1: {addr1}})
	defer c.Close()

	// Route each key to its shard's server and confirm values come back.
	results, err := c.BatchGet(context.Background(), [][]byte{kA, kB})
	require.NoError(t, err)

	// Map back by key (order preserved, but shard assignment is hash-based).
	byKey := map[string][]byte{}
	for _, r := range results {
		require.NoError(t, r.Err)
		byKey[string(r.Key)] = r.Value
	}
	// Each key resolves on whichever shard crc32 picks; both servers only hold
	// their own key, so a value is returned only if routing matches storage.
	// We assert the lookups completed without error and order is intact.
	assert.Len(t, results, 2)
	assert.Equal(t, kA, results[0].Key)
	assert.Equal(t, kB, results[1].Key)
}

func TestScatterGather_PartialFailure_NoPodForShard(t *testing.T) {
	// Shard 0 has a pod; shard 1 has none. Keys hashing to shard 1 fail,
	// keys to shard 0 succeed.
	k := key12("k")
	addr := startFakeServer(t, map[string][]byte{string(k): []byte("v")})

	// Only shard 0 has a pod.
	c := staticClient(2, map[uint32][]string{0: {addr}})
	defer c.Close()

	// Build keys that deterministically hit each shard.
	var shard0Key, shard1Key []byte
	for i := 0; i < 1000 && (shard0Key == nil || shard1Key == nil); i++ {
		cand := key12(string(rune('a'+i%26)) + string(rune('0'+i/26)))
		switch c.router.ShardFor(cand) {
		case 0:
			if shard0Key == nil {
				shard0Key = cand
			}
		case 1:
			if shard1Key == nil {
				shard1Key = cand
			}
		}
	}
	require.NotNil(t, shard0Key)
	require.NotNil(t, shard1Key)

	results, err := c.BatchGet(context.Background(), [][]byte{shard0Key, shard1Key})
	require.Error(t, err) // first shard error surfaced
	assert.ErrorIs(t, err, ErrNoHealthyPod)

	// shard0 key succeeded, shard1 key carries the error.
	for _, r := range results {
		if string(r.Key) == string(shard1Key) {
			assert.ErrorIs(t, r.Err, ErrNoHealthyPod)
		} else {
			assert.NoError(t, r.Err)
		}
	}
}

func TestScatterGather_PoolGetError(t *testing.T) {
	// Pod address that refuses connection → pool.Get (Dial) fails.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	deadAddr := ln.Addr().String()
	ln.Close()

	c := staticClient(1, map[uint32][]string{0: {deadAddr}})
	defer c.Close()

	results, err := c.BatchGet(context.Background(), [][]byte{key12("k")})
	require.Error(t, err)
	assert.Error(t, results[0].Err)
}

func TestScatterGather_BatchLookupError(t *testing.T) {
	// Server accepts then immediately closes → BatchLookup read fails.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close() // close before responding
		}
	}()

	c := staticClient(1, map[uint32][]string{0: {ln.Addr().String()}})
	defer c.Close()

	results, err := c.BatchGet(context.Background(), [][]byte{key12("k")})
	require.Error(t, err)
	assert.Error(t, results[0].Err)
}

// ── Client.Get (unit, no etcd) ──────────────────────────────────────────────

func TestClientGet_Hit(t *testing.T) {
	k := key12("hello")
	addr := startFakeServer(t, map[string][]byte{string(k): []byte("world")})
	c := NewDirectClient(addr, 4)
	defer c.Close()

	val, err := c.Get(context.Background(), k)
	require.NoError(t, err)
	assert.Equal(t, []byte("world"), val)
}

func TestClientGet_Miss(t *testing.T) {
	addr := startFakeServer(t, map[string][]byte{})
	c := NewDirectClient(addr, 4)
	defer c.Close()

	_, err := c.Get(context.Background(), key12("nope"))
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClientGet_NoHealthyPod(t *testing.T) {
	c := staticClient(1, nil) // no pods for shard 0
	defer c.Close()
	_, err := c.Get(context.Background(), key12("k"))
	assert.ErrorIs(t, err, ErrNoHealthyPod)
}

func TestClientGet_DialError(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	c := NewDirectClient(addr, 4)
	defer c.Close()
	_, err := c.Get(context.Background(), key12("k"))
	assert.Error(t, err)
}

func TestClientGet_BrokenConn(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close() // close immediately
		}
	}()
	c := NewDirectClient(ln.Addr().String(), 4)
	defer c.Close()
	_, err := c.Get(context.Background(), key12("k"))
	assert.Error(t, err)
}

// ── Client.StringGet (unit, no etcd) ────────────────────────────────────────

func TestClientStringGet_Hit(t *testing.T) {
	k := []byte("entity:123|456")
	addr := startFakeServer(t, map[string][]byte{string(k): []byte("val")})
	c := NewDirectClient(addr, 4)
	defer c.Close()

	val, err := c.StringGet(context.Background(), k)
	require.NoError(t, err)
	assert.Equal(t, []byte("val"), val)
}

func TestClientStringGet_Miss(t *testing.T) {
	addr := startFakeServer(t, map[string][]byte{})
	c := NewDirectClient(addr, 4)
	defer c.Close()

	_, err := c.StringGet(context.Background(), []byte("no:key"))
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClientStringGet_NoHealthyPod(t *testing.T) {
	c := staticClient(1, nil)
	defer c.Close()
	_, err := c.StringGet(context.Background(), []byte("k"))
	assert.ErrorIs(t, err, ErrNoHealthyPod)
}

func TestClientStringGet_DialError(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	c := NewDirectClient(addr, 4)
	defer c.Close()
	_, err := c.StringGet(context.Background(), []byte("k"))
	assert.Error(t, err)
}

func TestClientStringGet_BrokenConn(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	c := NewDirectClient(ln.Addr().String(), 4)
	defer c.Close()
	_, err := c.StringGet(context.Background(), []byte("k"))
	assert.Error(t, err)
}

// ── Client.StringBatchGet (unit, no etcd) ───────────────────────────────────

func TestClientStringBatchGet_HitAndMiss(t *testing.T) {
	k1 := []byte("entity:1|2")
	k2 := []byte("entity:3|4")
	addr := startFakeServer(t, map[string][]byte{string(k1): []byte("v1")})
	c := NewDirectClient(addr, 4)
	defer c.Close()

	results, err := c.StringBatchGet(context.Background(), [][]byte{k1, k2})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, []byte("v1"), results[0].Value)
	assert.Nil(t, results[1].Value)
}

func TestClientStringBatchGet_NoPod(t *testing.T) {
	c := staticClient(1, nil)
	defer c.Close()
	results, err := c.StringBatchGet(context.Background(), [][]byte{[]byte("k")})
	assert.Error(t, err)
	assert.ErrorIs(t, results[0].Err, ErrNoHealthyPod)
}

// ── Client.Close ────────────────────────────────────────────────────────────

func TestClientClose_DirectClient(t *testing.T) {
	c := NewDirectClient("127.0.0.1:1", 4)
	assert.NoError(t, c.Close())
}

// ── Config.applyDefaults full coverage ──────────────────────────────────────

func TestConfigApplyDefaults_AllZero(t *testing.T) {
	c := Config{}
	c.applyDefaults()
	assert.Equal(t, 4, c.ConnsPerPod)
	assert.Equal(t, 100, c.TimeoutMs)
	assert.Equal(t, 9091, c.Port)
	assert.Equal(t, "cluster.local", c.DNSZone)
	assert.Equal(t, "default", c.Namespace)
	assert.Equal(t, 30*time.Second, c.DNSRefreshInterval)
}

func TestConfigApplyDefaults_CustomValues(t *testing.T) {
	c := Config{
		ConnsPerPod:        8,
		TimeoutMs:          200,
		Port:               1234,
		DNSZone:            "custom.zone",
		Namespace:          "prod",
		DNSRefreshInterval: 10 * time.Second,
	}
	c.applyDefaults()
	assert.Equal(t, 8, c.ConnsPerPod)
	assert.Equal(t, 200, c.TimeoutMs)
	assert.Equal(t, 1234, c.Port)
	assert.Equal(t, "custom.zone", c.DNSZone)
	assert.Equal(t, "prod", c.Namespace)
	assert.Equal(t, 10*time.Second, c.DNSRefreshInterval)
}

// ── NewClient error paths (unit, no etcd) ───────────────────────────────────

func TestNewClient_NoEndpoints(t *testing.T) {
	_, err := NewClient(Config{Tenant: "t", Store: "s"})
	assert.ErrorIs(t, err, ErrNoEndpoints)
}

func TestNewClient_EtcdDialError(t *testing.T) {
	orig := newEtcdClient
	defer func() { newEtcdClient = orig }()
	newEtcdClient = func(_ []string) (*clientv3.Client, error) {
		return nil, assert.AnError
	}
	_, err := NewClient(Config{EtcdEndpoints: []string{"x"}, Tenant: "t", Store: "s"})
	assert.Error(t, err)
}

// ── scatter-gather: string variant failures + edge shapes ───────────────────

func isUnhealthy(r *Router, pod string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, bad := r.unhealthy[pod]
	return bad
}

// oneKeyPerShard returns one key routed to each of the router's shards, in shard order.
func oneKeyPerShard(t *testing.T, r *Router, shards uint32) [][]byte {
	t.Helper()
	keys := make([][]byte, shards)
	found := uint32(0)
	for i := 0; i < 10000 && found < shards; i++ {
		k := key12(fmt.Sprintf("k%d", i))
		if s := r.ShardFor(k); keys[s] == nil {
			keys[s] = k
			found++
		}
	}
	require.Equal(t, shards, found, "could not find a key for every shard")
	return keys
}

func TestStringScatterGather_PoolGetError_MarksPodUnhealthy(t *testing.T) {
	dead := refusedAddr(t)
	c := staticClient(1, map[uint32][]string{0: {dead}})
	defer c.Close()
	keys := [][]byte{[]byte("e:1"), []byte("e:2")}

	results, err := c.StringBatchGet(context.Background(), keys)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "onyxdb dial "+dead)
	require.Len(t, results, 2)
	for i, r := range results {
		assert.Equal(t, keys[i], r.Key)
		assert.Equal(t, err, r.Err)
		assert.Nil(t, r.Value)
	}
	assert.True(t, isUnhealthy(c.router, dead))
}

func TestStringScatterGather_BatchLookupError_DiscardsConnAndMarksPodUnhealthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close() // close before responding
		}
	}()
	addr := ln.Addr().String()
	c := staticClient(1, map[uint32][]string{0: {addr}})
	defer c.Close()

	results, err := c.StringBatchGet(context.Background(), [][]byte{[]byte("e:1")})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "string batch")
	require.Len(t, results, 1)
	assert.Equal(t, err, results[0].Err)
	assert.True(t, isUnhealthy(c.router, addr))
	assert.Equal(t, 0, podCount(c.pool, addr), "a broken connection must not be returned to the pool")
}

func TestBatchGets_EveryShardFails_EachKeyCarriesError(t *testing.T) {
	tests := []struct {
		name string
		get  func(c *Client, keys [][]byte) ([]Result, error)
	}{
		{"BatchGet", func(c *Client, keys [][]byte) ([]Result, error) { return c.BatchGet(context.Background(), keys) }},
		{"StringBatchGet", func(c *Client, keys [][]byte) ([]Result, error) { return c.StringBatchGet(context.Background(), keys) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := staticClient(2, nil) // neither shard has a pod
			defer c.Close()
			keys := oneKeyPerShard(t, c.router, 2)

			results, err := tc.get(c, keys)

			assert.ErrorIs(t, err, ErrNoHealthyPod)
			require.Len(t, results, 2)
			for i, r := range results {
				assert.Equal(t, keys[i], r.Key)
				assert.ErrorIs(t, r.Err, ErrNoHealthyPod)
				assert.Nil(t, r.Value)
			}
		})
	}
}

func TestBatchGets_NoKeys_ReturnEmptyWithoutDialing(t *testing.T) {
	tests := []struct {
		name string
		get  func(c *Client) ([]Result, error)
	}{
		{"BatchGet", func(c *Client) ([]Result, error) { return c.BatchGet(context.Background(), nil) }},
		{"StringBatchGet", func(c *Client) ([]Result, error) { return c.StringBatchGet(context.Background(), nil) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dead := refusedAddr(t) // any dial would fail
			c := staticClient(1, map[uint32][]string{0: {dead}})
			defer c.Close()

			results, err := tc.get(c)

			require.NoError(t, err)
			assert.Equal(t, []Result{}, results)
			assert.False(t, isUnhealthy(c.router, dead))
		})
	}
}

// ── Config.buildPoolConfig ──────────────────────────────────────────────────

func boolPtr(b bool) *bool { return &b }

func TestConfigBuildPoolConfig(t *testing.T) {
	defaults := PoolConfig{
		MinPerPod: 1, MaxPerPod: 4, DialTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second,
		IdleCheckInterval: 10 * time.Second, KeepAliveInterval: 15 * time.Second, KeepAliveTimeout: 5 * time.Second,
	}
	withMax := func(pc PoolConfig, max int) PoolConfig { pc.MaxPerPod = max; return pc }
	tests := []struct {
		name        string
		cfg         Config
		wantPool    PoolConfig
		wantTimeout int
		wantDNS     time.Duration
	}{
		{
			name: "explicit Pool wins and ClientConfig is ignored",
			cfg: Config{
				ConnsPerPod: 4, TimeoutMs: 100, DNSRefreshInterval: 30 * time.Second,
				Pool:         &PoolConfig{MinPerPod: 2, MaxPerPod: 6},
				ClientConfig: &model.ClientConfig{RequestTimeoutMs: 900, MaxConnsPerPod: 9, DNSRefreshIntervalMs: 1000},
			},
			wantPool:    func() PoolConfig { pc := withMax(defaults, 6); pc.MinPerPod = 2; return pc }(),
			wantTimeout: 100,
			wantDNS:     30 * time.Second,
		},
		{
			name:        "no ClientConfig: ConnsPerPod is the ceiling, rest defaulted",
			cfg:         Config{ConnsPerPod: 8, TimeoutMs: 100, DNSRefreshInterval: 30 * time.Second},
			wantPool:    withMax(defaults, 8),
			wantTimeout: 100,
			wantDNS:     30 * time.Second,
		},
		{
			name: "ClientConfig with zero fields keeps defaults",
			cfg: Config{
				ConnsPerPod: 4, TimeoutMs: 100, DNSRefreshInterval: 30 * time.Second,
				ClientConfig: &model.ClientConfig{},
			},
			wantPool:    defaults,
			wantTimeout: 100,
			wantDNS:     30 * time.Second,
		},
		{
			name: "ClientConfig with every field set overlays pool and request settings",
			cfg: Config{
				ConnsPerPod: 4, TimeoutMs: 100, DNSRefreshInterval: 30 * time.Second,
				ClientConfig: &model.ClientConfig{
					ConnectTimeoutMs: 700, RequestTimeoutMs: 250, KeepAliveIntervalMs: 2000,
					KeepAliveTimeoutMs: 3000, IdleTimeoutMs: 40000, IdleCheckIntervalMs: 500,
					MinConnsPerPod: 3, MaxConnsPerPod: 12, DNSRefreshIntervalMs: 5000,
				},
			},
			wantPool: PoolConfig{
				MinPerPod: 3, MaxPerPod: 12, DialTimeout: 700 * time.Millisecond, IdleTimeout: 40 * time.Second,
				IdleCheckInterval: 500 * time.Millisecond, KeepAliveInterval: 2 * time.Second, KeepAliveTimeout: 3 * time.Second,
			},
			wantTimeout: 250,
			wantDNS:     5 * time.Second,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var callerPool PoolConfig
			if tc.cfg.Pool != nil {
				callerPool = *tc.cfg.Pool
			}

			got := tc.cfg.buildPoolConfig()

			assert.Equal(t, tc.wantPool, got)
			assert.Equal(t, tc.wantTimeout, tc.cfg.TimeoutMs)
			assert.Equal(t, tc.wantDNS, tc.cfg.DNSRefreshInterval)
			if tc.cfg.Pool != nil {
				assert.Equal(t, callerPool, *tc.cfg.Pool, "the caller's PoolConfig must not be mutated")
			}
		})
	}
}

// ── fetchClientConfig ───────────────────────────────────────────────────────

func TestFetchClientConfig(t *testing.T) {
	path := model.ClientConfigPath("recsys", "catalog")
	tests := []struct {
		name  string
		setup func(m *mockEtcd)
		want  *model.ClientConfig
	}{
		{"etcd error yields nil", func(m *mockEtcd) { m.setGetErr(path, errors.New("etcd down")) }, nil},
		{"absent key yields nil", func(*mockEtcd) {}, nil},
		{"corrupt JSON yields nil", func(m *mockEtcd) { m.put(path, "{not json") }, nil},
		{
			"valid JSON is decoded",
			func(m *mockEtcd) {
				m.put(path, `{"requestTimeoutMs":250,"maxConnsPerPod":8,"warmUpOnTopologyChange":false}`)
			},
			&model.ClientConfig{RequestTimeoutMs: 250, MaxConnsPerPod: 8, WarmUpOnTopologyChange: boolPtr(false)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockEtcd()
			tc.setup(m)
			assert.Equal(t, tc.want, fetchClientConfig(context.Background(), m, "recsys", "catalog"))
		})
	}
}

// ── NewClient / newClientWithEtcd ───────────────────────────────────────────

// closerStub records Close calls on the client's etcd handle.
type closerStub struct {
	calls int
	err   error
}

func (c *closerStub) Close() error {
	c.calls++
	return c.err
}

// etcdKVStub and etcdWatcherStub adapt mockEtcd to clientv3's KV / Watcher
// interfaces, so a connectionless *clientv3.Client (NewCtxClient) can stand in
// for a dialled one.
type etcdKVStub struct {
	clientv3.KV
	m *mockEtcd
}

func (s etcdKVStub) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return s.m.Get(ctx, key, opts...)
}

type etcdWatcherStub struct {
	clientv3.Watcher
	m *mockEtcd
}

func (s etcdWatcherStub) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return s.m.Watch(ctx, key, opts...)
}

func (etcdWatcherStub) Close() error { return nil }

// putActiveV1 publishes v1 as the active version with the given meta JSON.
func putActiveV1(m *mockEtcd, meta string) {
	m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
	m.put(model.VersionPrefix("recsys", "catalog", "v1"), meta)
}

func reloadedOK(mc *metricCollector) bool {
	counts, _ := reloadRecords(mc)
	for _, r := range counts {
		if hasTag(r.Tags, "status:ok") {
			return true
		}
	}
	return false
}

func TestNewEtcdClient_RejectsEmptyEndpointList(t *testing.T) {
	cli, err := newEtcdClient(nil)
	assert.Nil(t, cli)
	assert.ErrorIs(t, err, clientv3.ErrNoAvailableEndpoints)
}

func TestNewClient_Success_RoutesViaEtcdTopologyAndClosesEtcd(t *testing.T) {
	k := key12("hello")
	addr := startFakeServer(t, map[string][]byte{string(k): []byte("world")})
	m := newMockEtcd()
	putActiveV1(m, metaWithAssignmentJSON(t, 1, map[string][]string{"0": {addr}}))

	var gotEndpoints []string
	orig := newEtcdClient
	t.Cleanup(func() { newEtcdClient = orig })
	newEtcdClient = func(endpoints []string) (*clientv3.Client, error) {
		gotEndpoints = endpoints
		cli := clientv3.NewCtxClient(context.Background())
		cli.KV = etcdKVStub{m: m}
		cli.Watcher = etcdWatcherStub{m: m}
		return cli, nil
	}

	withLookup(failLookup, func() {
		c, err := NewClient(Config{
			EtcdEndpoints: []string{"etcd-a:2379", "etcd-b:2379"},
			Tenant:        "recsys", Store: "catalog", DNSRefreshInterval: time.Hour,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"etcd-a:2379", "etcd-b:2379"}, gotEndpoints)

		require.Eventually(t, func() bool { return c.router.ShardCount() == 1 }, 3*time.Second, 5*time.Millisecond)
		val, err := c.Get(context.Background(), k)
		require.NoError(t, err)
		assert.Equal(t, []byte("world"), val)

		// Close releases the etcd client; a connectionless clientv3 client
		// reports its own cancelled context from Close.
		assert.ErrorIs(t, c.Close(), context.Canceled)
	})
}

func TestNewClientWithEtcd_ClientConfigSource(t *testing.T) {
	ccPath := model.ClientConfigPath("recsys", "catalog")
	supplied := &model.ClientConfig{RequestTimeoutMs: 400}
	tests := []struct {
		name        string
		etcdCC      string // "" = absent
		supplied    *model.ClientConfig
		wantCC      *model.ClientConfig
		wantTimeout int
		wantMin     int
		wantMax     int
		wantDNS     time.Duration
	}{
		{
			name:        "fetched from etcd when not supplied",
			etcdCC:      `{"requestTimeoutMs":250,"minConnsPerPod":2,"maxConnsPerPod":7,"dnsRefreshIntervalMs":3600000}`,
			wantCC:      &model.ClientConfig{RequestTimeoutMs: 250, MinConnsPerPod: 2, MaxConnsPerPod: 7, DNSRefreshIntervalMs: 3600000},
			wantTimeout: 250, wantMin: 2, wantMax: 7, wantDNS: time.Hour,
		},
		{
			name:        "supplied config skips the etcd fetch",
			etcdCC:      `{"requestTimeoutMs":250}`,
			supplied:    supplied,
			wantCC:      supplied,
			wantTimeout: 400, wantMin: 1, wantMax: 4, wantDNS: 30 * time.Second,
		},
		{
			name:        "absent in etcd: all defaults",
			wantTimeout: 100, wantMin: 1, wantMax: 4, wantDNS: 30 * time.Second,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockEtcd()
			if tc.etcdCC != "" {
				m.put(ccPath, tc.etcdCC)
			}
			withLookup(failLookup, func() {
				c, err := newClientWithEtcd(Config{Tenant: "recsys", Store: "catalog", ClientConfig: tc.supplied}, m, nil)
				require.NoError(t, err)
				defer c.Close()

				assert.Equal(t, tc.wantCC, c.config.ClientConfig)
				assert.Equal(t, tc.wantTimeout, c.config.TimeoutMs)
				assert.Equal(t, tc.wantDNS, c.config.DNSRefreshInterval)
				assert.Equal(t, tc.wantMin, c.pool.cfg.MinPerPod)
				assert.Equal(t, tc.wantMax, c.pool.cfg.MaxPerPod)
			})
		})
	}
}

func TestNewClientWithEtcd_WarmUpOnTopologyChange(t *testing.T) {
	tests := []struct {
		name      string
		cc        *model.ClientConfig
		wantConns int
	}{
		{"no ClientConfig: warm the default floor of 1", nil, 1},
		{"flag unset: warm MinConnsPerPod", &model.ClientConfig{MinConnsPerPod: 2}, 2},
		{"flag true: warm MinConnsPerPod", &model.ClientConfig{MinConnsPerPod: 2, WarmUpOnTopologyChange: boolPtr(true)}, 2},
		{"flag false: no warm-up", &model.ClientConfig{MinConnsPerPod: 2, WarmUpOnTopologyChange: boolPtr(false)}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr := holdingListener(t)
			m := newMockEtcd()
			putActiveV1(m, metaWithAssignmentJSON(t, 1, map[string][]string{"0": {addr}}))
			mc := &metricCollector{}
			withLookup(failLookup, func() {
				c, err := newClientWithEtcd(Config{
					Tenant: "recsys", Store: "catalog", Count: mc.count, Timing: mc.timing, ClientConfig: tc.cc,
				}, m, nil)
				require.NoError(t, err)
				defer c.Close()

				require.Eventually(t, func() bool { return reloadedOK(mc) }, 3*time.Second, 5*time.Millisecond)
				counts, _ := reloadRecords(mc)
				assert.Equal(t, []string{"tenant:recsys", "store:catalog", "status:ok"}, counts[0].Tags)

				if tc.wantConns == 0 {
					assert.Equal(t, 0, podCount(c.pool, addr), "warm-up disabled: nothing pre-dialled")
					return
				}
				require.Eventually(t, func() bool { return podCount(c.pool, addr) == tc.wantConns },
					3*time.Second, 5*time.Millisecond)
			})
		})
	}
}

// DNS is only the fallback: shards the assignment covers are never looked up.
func TestNewClientWithEtcd_DNSResolvesOnlyUnassignedShards(t *testing.T) {
	var mu sync.Mutex
	var looked []string
	lookup := func(_ context.Context, host string) ([]string, error) {
		mu.Lock()
		looked = append(looked, host)
		mu.Unlock()
		return []string{"127.0.0.1"}, nil
	}
	m := newMockEtcd()
	putActiveV1(m, metaWithAssignmentJSON(t, 2, map[string][]string{"0": {"10.0.1.10:9091"}}))
	withLookup(lookup, func() {
		c, err := newClientWithEtcd(Config{
			Tenant: "recsys", Store: "catalog", Namespace: "onyxdb", Port: 9555, DNSRefreshInterval: time.Hour,
			ClientConfig: &model.ClientConfig{WarmUpOnTopologyChange: boolPtr(false)},
		}, m, nil)
		require.NoError(t, err)
		defer c.Close()

		require.Eventually(t, func() bool {
			pod, err := c.router.PodFor(1)
			return err == nil && pod == "127.0.0.1:9555"
		}, 3*time.Second, 5*time.Millisecond)
		pod, err := c.router.PodFor(0)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.10:9091", pod)
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"recsys-catalog-shard-1.onyxdb.svc.cluster.local"}, looked)
}

// Every DNS refresh clears local unhealthy marks and prunes pools for pods that
// left the topology.
func TestNewClientWithEtcd_RefreshClearsUnhealthyAndPrunesPool(t *testing.T) {
	m := newMockEtcd()
	putActiveV1(m, metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.10:9091"}}))
	mc := &metricCollector{}
	withLookup(failLookup, func() {
		c, err := newClientWithEtcd(Config{
			Tenant: "recsys", Store: "catalog", Count: mc.count, DNSRefreshInterval: time.Hour,
			ClientConfig: &model.ClientConfig{WarmUpOnTopologyChange: boolPtr(false)},
		}, m, nil)
		require.NoError(t, err)
		defer c.Close()
		require.Eventually(t, func() bool { return reloadedOK(mc) }, 3*time.Second, 5*time.Millisecond)

		c.router.MarkUnhealthy("10.0.1.10:9091")
		stale := idleConn(t, time.Now())
		c.pool.Put("10.0.9.9:9091", stale)
		m.podCh() <- podPutEvent() // pod registration change → reload → DNS refresh

		require.Eventually(t, func() bool {
			return !isUnhealthy(c.router, "10.0.1.10:9091") && !hasPool(c.pool, "10.0.9.9:9091")
		}, 3*time.Second, 5*time.Millisecond)
		assert.True(t, isClosed(stale))
	})
}

func TestClientClose_ClosesEtcdAndReturnsItsError(t *testing.T) {
	c := staticClient(1, nil)
	cancelled := false
	c.cancel = func() { cancelled = true }
	closer := &closerStub{err: errors.New("etcd close failed")}
	c.etcdCloser = closer

	err := c.Close()

	assert.EqualError(t, err, "etcd close failed")
	assert.Equal(t, 1, closer.calls)
	assert.True(t, cancelled, "Close must stop the topology watcher")
	c.pool.mu.Lock()
	defer c.pool.mu.Unlock()
	assert.True(t, c.pool.closed)
}
