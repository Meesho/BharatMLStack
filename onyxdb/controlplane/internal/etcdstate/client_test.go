package etcdstate

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// freePort returns a TCP port that the OS has confirmed is not yet bound.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// startEmbeddedEtcd starts an in-process etcd server on free ports.
// Returns the client URL and a teardown func.
func startEmbeddedEtcd(t *testing.T) (clientURL string, teardown func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "etcd-onyxdb-test-*")
	require.NoError(t, err)

	clientPort := freePort(t)
	peerPort := freePort(t)
	cu, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", clientPort))
	pu, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", peerPort))

	cfg := embed.NewConfig()
	cfg.Dir = dir
	cfg.LogLevel = "error"
	cfg.ListenClientUrls = []url.URL{*cu}
	cfg.AdvertiseClientUrls = []url.URL{*cu}
	cfg.ListenPeerUrls = []url.URL{*pu}
	cfg.AdvertisePeerUrls = []url.URL{*pu}
	cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, pu.String())

	e, err := embed.StartEtcd(cfg)
	require.NoError(t, err)

	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		e.Server.Stop()
		t.Fatal("embedded etcd did not start in time")
	}

	return cu.String(), func() {
		e.Server.Stop()
		_ = os.RemoveAll(dir)
	}
}

// ── NewEtcdStateClient ────────────────────────────────────────────────────────

func TestNewEtcdStateClient_Success(t *testing.T) {
	clientURL, teardown := startEmbeddedEtcd(t)
	defer teardown()

	sc, err := NewEtcdStateClient([]string{clientURL})
	require.NoError(t, err)
	assert.NotNil(t, sc)
	assert.NoError(t, sc.Close())
}

func TestNewEtcdStateClient_EmptyEndpoints(t *testing.T) {
	_, err := NewEtcdStateClient([]string{})
	assert.Error(t, err)
}

// ── etcdKVOps ────────────────────────────────────────────────────────────────

func TestEtcdKVOps(t *testing.T) {
	clientURL, teardown := startEmbeddedEtcd(t)
	defer teardown()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{clientURL},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	defer cli.Close()

	ops := &etcdKVOps{kv: cli.KV}
	ctx := context.Background()

	t.Run("get_not_found", func(t *testing.T) {
		val, rev, found, err := ops.get(ctx, "/onyxdb-test/missing")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, val)
		assert.Zero(t, rev)
	})

	t.Run("put_and_get", func(t *testing.T) {
		require.NoError(t, ops.put(ctx, "/onyxdb-test/key1", "hello"))
		val, rev, found, err := ops.get(ctx, "/onyxdb-test/key1")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "hello", val)
		assert.Positive(t, rev)
	})

	t.Run("getPrefix", func(t *testing.T) {
		require.NoError(t, ops.put(ctx, "/onyxdb-test/prefix/a", "1"))
		require.NoError(t, ops.put(ctx, "/onyxdb-test/prefix/b", "2"))
		result, err := ops.getPrefix(ctx, "/onyxdb-test/prefix/")
		require.NoError(t, err)
		assert.Equal(t, "1", result["/onyxdb-test/prefix/a"])
		assert.Equal(t, "2", result["/onyxdb-test/prefix/b"])
	})

	t.Run("atomicCreate_success", func(t *testing.T) {
		err := ops.atomicCreate(ctx, "/onyxdb-test/new-key", map[string]string{
			"/onyxdb-test/new-key": "value",
			"/onyxdb-test/other":   "other-value",
		})
		require.NoError(t, err)
		val, _, found, _ := ops.get(ctx, "/onyxdb-test/new-key")
		assert.True(t, found)
		assert.Equal(t, "value", val)
	})

	t.Run("atomicCreate_already_exists", func(t *testing.T) {
		// Put the key first so atomicCreate finds it
		require.NoError(t, ops.put(ctx, "/onyxdb-test/exists", "v"))
		err := ops.atomicCreate(ctx, "/onyxdb-test/exists", map[string]string{"/onyxdb-test/exists": "v2"})
		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("atomicSwap_success", func(t *testing.T) {
		require.NoError(t, ops.put(ctx, "/onyxdb-test/swap-key", "initial"))
		_, rev, _, _ := ops.get(ctx, "/onyxdb-test/swap-key")

		ok, err := ops.atomicSwap(ctx, "/onyxdb-test/swap-key", rev, map[string]string{
			"/onyxdb-test/swap-key":    "updated",
			"/onyxdb-test/swap-other":  "side-effect",
		})
		require.NoError(t, err)
		assert.True(t, ok)

		val, _, _, _ := ops.get(ctx, "/onyxdb-test/swap-key")
		assert.Equal(t, "updated", val)
	})

	t.Run("atomicSwap_cas_fails_on_stale_rev", func(t *testing.T) {
		require.NoError(t, ops.put(ctx, "/onyxdb-test/cas-key", "v1"))
		_, rev, _, _ := ops.get(ctx, "/onyxdb-test/cas-key")

		// Advance the key so rev is stale
		require.NoError(t, ops.put(ctx, "/onyxdb-test/cas-key", "v2"))

		ok, err := ops.atomicSwap(ctx, "/onyxdb-test/cas-key", rev, map[string]string{
			"/onyxdb-test/cas-key": "v3",
		})
		require.NoError(t, err)
		assert.False(t, ok)

		// Key should still be "v2", not "v3"
		val, _, _, _ := ops.get(ctx, "/onyxdb-test/cas-key")
		assert.Equal(t, "v2", val)
	})
}

// ── Concurrent safety ────────────────────────────────────────────────────────

func TestEtcdKVOps_ConcurrentPuts(t *testing.T) {
	clientURL, teardown := startEmbeddedEtcd(t)
	defer teardown()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{clientURL},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	defer cli.Close()

	ops := &etcdKVOps{kv: cli.KV}
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("/onyxdb-test/concurrent/%d", n)
			assert.NoError(t, ops.put(ctx, key, fmt.Sprintf("v%d", n)))
		}(i)
	}
	wg.Wait()

	result, err := ops.getPrefix(ctx, "/onyxdb-test/concurrent/")
	require.NoError(t, err)
	assert.Len(t, result, 20)
}

// ── Error paths via cancelled context ────────────────────────────────────────

func TestEtcdKVOps_CancelledContext(t *testing.T) {
	clientURL, teardown := startEmbeddedEtcd(t)
	defer teardown()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{clientURL},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	defer cli.Close()

	ops := &etcdKVOps{kv: cli.KV}

	// A pre-cancelled context makes all etcd RPCs fail immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("get_error", func(t *testing.T) {
		_, _, _, err := ops.get(ctx, "/onyxdb-test/any")
		assert.Error(t, err)
	})

	t.Run("getPrefix_error", func(t *testing.T) {
		_, err := ops.getPrefix(ctx, "/onyxdb-test/")
		assert.Error(t, err)
	})

	t.Run("atomicCreate_error", func(t *testing.T) {
		err := ops.atomicCreate(ctx, "/onyxdb-test/k", map[string]string{"/onyxdb-test/k": "v"})
		assert.Error(t, err)
	})

	t.Run("atomicSwap_error", func(t *testing.T) {
		_, err := ops.atomicSwap(ctx, "/onyxdb-test/k", 0, map[string]string{"/onyxdb-test/k": "v"})
		assert.Error(t, err)
	})
}
