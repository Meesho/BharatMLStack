package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// ── mock etcd client ──────────────────────────────────────────────────────────

type mockEtcd struct {
	mu      sync.Mutex
	kv      map[string]string
	getErr  map[string]error
	watchCh chan clientv3.WatchResponse
}

func newMockEtcd() *mockEtcd {
	return &mockEtcd{
		kv:      map[string]string{},
		getErr:  map[string]error{},
		watchCh: make(chan clientv3.WatchResponse, 8),
	}
}

func (m *mockEtcd) put(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kv[key] = value
}

func (m *mockEtcd) setGetErr(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErr[key] = err
}

func (m *mockEtcd) Get(_ context.Context, key string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.getErr[key]; err != nil {
		return nil, err
	}
	resp := &clientv3.GetResponse{}
	if v, ok := m.kv[key]; ok {
		resp.Kvs = []*mvccpb.KeyValue{{Key: []byte(key), Value: []byte(v)}}
	}
	return resp, nil
}

func (m *mockEtcd) Watch(_ context.Context, _ string, _ ...clientv3.OpOption) clientv3.WatchChan {
	return m.watchCh
}

func putActiveVersion(version string) clientv3.WatchResponse {
	return clientv3.WatchResponse{
		Events: []*clientv3.Event{
			{Type: mvccpb.PUT, Kv: &mvccpb.KeyValue{Value: []byte(version)}},
		},
	}
}

func metaJSON(t *testing.T, shardCount int) string {
	t.Helper()
	b, err := json.Marshal(model.VersionMeta{ShardCount: shardCount, Status: model.StatusActive})
	require.NoError(t, err)
	return string(b)
}

// newWatcherWith builds a watcher over a mock etcd + a DNS resolver whose
// lookup is already stubbed (caller wraps in withLookup).
func newWatcherWith(m *mockEtcd) (*TopologyWatcher, *Router, *DNSResolver) {
	resolver := NewDNSResolver(dnsCfg(9091))
	router := NewRouter(resolver)
	tw := NewTopologyWatcher(m, router, resolver, "recsys", "catalog")
	return tw, router, resolver
}

// ── reload ────────────────────────────────────────────────────────────────────

func TestReload_NoActiveVersion(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		tw, r, _ := newWatcherWith(m)
		require.NoError(t, tw.reload(context.Background()))
		assert.Equal(t, uint32(0), r.ShardCount())
	})
}

func TestReload_EmptyActiveVersionValue(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "")
		tw, r, _ := newWatcherWith(m)
		require.NoError(t, tw.reload(context.Background()))
		assert.Equal(t, uint32(0), r.ShardCount())
	})
}

func TestReload_Success_SetsShardCountAndResolves(t *testing.T) {
	withLookup(okLookup("10.0.0.7"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 2))
		tw, r, _ := newWatcherWith(m)

		require.NoError(t, tw.reload(context.Background()))
		assert.Equal(t, uint32(2), r.ShardCount())
		// Resolver was refreshed → router resolves shard 1 to the looked-up IP.
		pod, err := r.PodFor(1)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.7:9091", pod)
	})
}

func TestReload_GetActiveVersionError(t *testing.T) {
	m := newMockEtcd()
	m.setGetErr(model.ActiveVersionPath("recsys", "catalog"), errors.New("etcd down"))
	tw, _, _ := newWatcherWith(m)
	assert.ErrorContains(t, tw.reload(context.Background()), "get activeVersion")
}

func TestReloadVersion_GetMetaError(t *testing.T) {
	m := newMockEtcd()
	m.setGetErr(model.VersionPrefix("recsys", "catalog", "v1"), errors.New("timeout"))
	tw, _, _ := newWatcherWith(m)
	assert.ErrorContains(t, tw.reloadVersion(context.Background(), "v1"), "get version meta")
}

func TestReloadVersion_MetaNotFound(t *testing.T) {
	m := newMockEtcd()
	tw, _, _ := newWatcherWith(m)
	assert.ErrorContains(t, tw.reloadVersion(context.Background(), "ghost"), "not found")
}

func TestReloadVersion_CorruptJSON(t *testing.T) {
	m := newMockEtcd()
	m.put(model.VersionPrefix("recsys", "catalog", "v1"), "not-json")
	tw, _, _ := newWatcherWith(m)
	assert.ErrorContains(t, tw.reloadVersion(context.Background(), "v1"), "parse version meta")
}

// ── Run ───────────────────────────────────────────────────────────────────────

func TestRun_InitialLoadThenReResolveOnFlip(t *testing.T) {
	// lookup returns different IPs before/after the flip — mirrors DNS endpoints
	// changing as readiness probes re-evaluate post-promote.
	var mu sync.Mutex
	current := "10.0.0.1"
	withLookup(func(_ context.Context, _ string) ([]string, error) {
		mu.Lock()
		defer mu.Unlock()
		return []string{current}, nil
	}, func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))
		m.put(model.VersionPrefix("recsys", "catalog", "v2"), metaJSON(t, 1))

		tw, r, _ := newWatcherWith(m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)

		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.0.1:9091"
		}, time.Second, 5*time.Millisecond)

		// Promote v2; DNS now returns a different pod IP.
		mu.Lock()
		current = "10.0.0.2"
		mu.Unlock()
		m.watchCh <- putActiveVersion("v2")

		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.0.2:9091"
		}, time.Second, 5*time.Millisecond)
	})
}

func TestRun_InitialLoadFailsButWatchStillRuns(t *testing.T) {
	withLookup(okLookup("10.0.0.9"), func() {
		m := newMockEtcd()
		m.setGetErr(model.ActiveVersionPath("recsys", "catalog"), errors.New("etcd unavailable"))
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))

		tw, r, _ := newWatcherWith(m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)

		m.watchCh <- putActiveVersion("v1")
		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.0.9:9091"
		}, time.Second, 5*time.Millisecond)
	})
}

func TestRun_WatchReloadError_IsLoggedNotFatal(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		tw, r, _ := newWatcherWith(m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)

		m.watchCh <- putActiveVersion("ghost") // meta missing → logged error
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))
		m.watchCh <- putActiveVersion("v1")

		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.0.1:9091"
		}, time.Second, 5*time.Millisecond)
	})
}

func TestRun_IgnoresNonPutAndEmptyEvents(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		tw, r, _ := newWatcherWith(m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)

		m.watchCh <- clientv3.WatchResponse{Events: []*clientv3.Event{
			{Type: mvccpb.DELETE, Kv: &mvccpb.KeyValue{Value: []byte("v1")}},
			{Type: mvccpb.PUT, Kv: &mvccpb.KeyValue{Value: []byte("")}},
		}}
		time.Sleep(30 * time.Millisecond)
		assert.Equal(t, uint32(0), r.ShardCount())
	})
}

func TestRun_ContextCancel_Returns(t *testing.T) {
	m := newMockEtcd()
	tw, _, _ := newWatcherWith(m)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- tw.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestRun_WatchChannelClosed_Returns(t *testing.T) {
	m := newMockEtcd()
	tw, _, _ := newWatcherWith(m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- tw.Run(ctx) }()
	close(m.watchCh)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after watch channel close")
	}
}

// okLookup returns a stub that always resolves to a single IP.
func okLookup(ip string) func(context.Context, string) ([]string, error) {
	return func(_ context.Context, _ string) ([]string, error) {
		return []string{ip}, nil
	}
}
