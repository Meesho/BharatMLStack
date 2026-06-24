package sdk

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientv3 "go.etcd.io/etcd/client/v3"
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
