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

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// ── mock etcd client ──────────────────────────────────────────────────────────

type mockEtcd struct {
	mu       sync.Mutex
	kv       map[string]string
	getErr   map[string]error
	watchChs []chan clientv3.WatchResponse
	watchIdx int
}

func newMockEtcd() *mockEtcd {
	return &mockEtcd{
		kv:     map[string]string{},
		getErr: map[string]error{},
		// Pre-allocate channels for two watches: activeVersion + podPrefix.
		watchChs: []chan clientv3.WatchResponse{
			make(chan clientv3.WatchResponse, 8),
			make(chan clientv3.WatchResponse, 8),
		},
	}
}

// activeVersionCh returns the channel for the first Watch call (activeVersion).
func (m *mockEtcd) activeVersionCh() chan clientv3.WatchResponse { return m.watchChs[0] }

// podCh returns the channel for the second Watch call (pod registrations).
func (m *mockEtcd) podCh() chan clientv3.WatchResponse { return m.watchChs[1] }

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
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.watchIdx
	m.watchIdx++
	if idx < len(m.watchChs) {
		return m.watchChs[idx]
	}
	// Fallback: return a never-firing channel.
	ch := make(chan clientv3.WatchResponse)
	return ch
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

func metaWithAssignmentJSON(t *testing.T, shardCount int, assignment map[string][]string) string {
	t.Helper()
	b, err := json.Marshal(model.VersionMeta{
		ShardCount: shardCount,
		Status:     model.StatusActive,
		Assignment: assignment,
	})
	require.NoError(t, err)
	return string(b)
}

// newWatcherWith builds a watcher over a mock etcd + a DNS resolver whose
// lookup is already stubbed (caller wraps in withLookup).
func newWatcherWith(m *mockEtcd) (*TopologyWatcher, *Router, *DNSResolver) {
	dnsRes := NewDNSResolver(dnsCfg(9091))
	assignRes := NewAssignmentResolver()
	router := NewRouter(NewFallbackResolver(assignRes, dnsRes))
	tw := NewTopologyWatcher(m, router, dnsRes, "recsys", "catalog")
	tw.SetAssignmentResolver(assignRes)
	return tw, router, dnsRes
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

// ── Assignment-aware routing ─────────────────────────────────────────────────

func TestReload_PushesAssignmentToResolver(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 2, map[string][]string{
				"0": {"10.0.1.10:9091", "10.0.1.11:9091"},
				"1": {"10.0.1.12:9091"},
			}))

		tw, r, _ := newWatcherWith(m)
		require.NoError(t, tw.reload(context.Background()))

		// Shard 0 should resolve to one of the assignment addrs, not DNS.
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		assert.Contains(t, []string{"10.0.1.10:9091", "10.0.1.11:9091"}, pod)

		// Shard 1 should resolve to the single assignment addr.
		pod, err = r.PodFor(1)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.12:9091", pod)
	})
}

func TestReload_EmptyAssignment_FallsBackToDNS(t *testing.T) {
	withLookup(okLookup("10.0.0.5"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		// No assignment in meta → DNS fallback.
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))

		tw, r, _ := newWatcherWith(m)
		require.NoError(t, tw.reload(context.Background()))

		pod, err := r.PodFor(0)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5:9091", pod)
	})
}

// ── Run ───────────────────────────────────────────────────────────────────────

func TestRun_InitialLoadThenReResolveOnFlip(t *testing.T) {
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
		m.activeVersionCh() <- putActiveVersion("v2")

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

		m.activeVersionCh() <- putActiveVersion("v1")
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

		m.activeVersionCh() <- putActiveVersion("ghost") // meta missing → logged error
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))
		m.activeVersionCh() <- putActiveVersion("v1")

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

		m.activeVersionCh() <- clientv3.WatchResponse{Events: []*clientv3.Event{
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
	close(m.activeVersionCh())
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after watch channel close")
	}
}

// ── Pod watch triggers re-read ──────────────────────────────────────────────

func TestRun_PodRegistrationChange_TriggersReload(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{
				"0": {"10.0.1.10:9091"},
			}))

		tw, r, _ := newWatcherWith(m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)

		// Wait for initial load.
		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.1.10:9091"
		}, time.Second, 5*time.Millisecond)

		// Simulate a scale-up: update the version meta with a new pod.
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{
				"0": {"10.0.1.10:9091", "10.0.1.20:9091"},
			}))

		// Fire a pod registration event.
		m.podCh() <- clientv3.WatchResponse{Events: []*clientv3.Event{
			{Type: mvccpb.PUT, Kv: &mvccpb.KeyValue{
				Key:   []byte("/config/mnemo-cluster-manager/recsys/catalog/new-pod"),
				Value: []byte("{}"),
			}},
		}}

		// Router should now see both pods for shard 0.
		require.Eventually(t, func() bool {
			// After reload, both addrs should be reachable via round-robin.
			seen := make(map[string]bool)
			for i := 0; i < 10; i++ {
				pod, _ := r.PodFor(0)
				seen[pod] = true
			}
			return seen["10.0.1.10:9091"] && seen["10.0.1.20:9091"]
		}, time.Second, 10*time.Millisecond)
	})
}

// ── AssignmentResolver ──────────────────────────────────────────────────────

func TestAssignmentResolver_SwapAndResolve(t *testing.T) {
	ar := NewAssignmentResolver()
	newAddrs := ar.SwapAssignment(map[string][]string{
		"0": {"10.0.0.1:9091"},
		"1": {"10.0.0.2:9091", "10.0.0.3:9091"},
	})
	// First swap: all addrs are new.
	assert.Len(t, newAddrs, 3)
	assert.Equal(t, []string{"10.0.0.1:9091"}, ar.Resolve(0))
	assert.Len(t, ar.Resolve(1), 2)
	assert.Nil(t, ar.Resolve(99)) // unknown shard
}

func TestAssignmentResolver_SwapDetectsNewAddrs(t *testing.T) {
	ar := NewAssignmentResolver()
	ar.SwapAssignment(map[string][]string{
		"0": {"10.0.0.1:9091"},
	})
	newAddrs := ar.SwapAssignment(map[string][]string{
		"0": {"10.0.0.1:9091", "10.0.0.2:9091"}, // 10.0.0.2 is new
	})
	assert.Equal(t, []string{"10.0.0.2:9091"}, newAddrs)
}

func TestAssignmentResolver_AllAddrs(t *testing.T) {
	ar := NewAssignmentResolver()
	ar.SwapAssignment(map[string][]string{
		"0": {"a:1", "b:2"},
		"1": {"b:2", "c:3"},
	})
	all := ar.AllAddrs()
	assert.Len(t, all, 3)
}

// ── FallbackResolver ─────────────────────────────────────────────────────────

func TestFallbackResolver_PrimaryWins(t *testing.T) {
	primary := NewStaticResolver(map[uint32][]string{0: {"primary:1"}})
	secondary := NewStaticResolver(map[uint32][]string{0: {"secondary:1"}})
	fb := NewFallbackResolver(primary, secondary)
	assert.Equal(t, []string{"primary:1"}, fb.Resolve(0))
}

func TestFallbackResolver_FallsBackWhenPrimaryEmpty(t *testing.T) {
	primary := NewStaticResolver(map[uint32][]string{})
	secondary := NewStaticResolver(map[uint32][]string{0: {"secondary:1"}})
	fb := NewFallbackResolver(primary, secondary)
	assert.Equal(t, []string{"secondary:1"}, fb.Resolve(0))
}

// ── helpers ─────────────────────────────────────────────────────────────────

// okLookup returns a stub that always resolves to a single IP.
func okLookup(ip string) func(context.Context, string) ([]string, error) {
	return func(_ context.Context, _ string) ([]string, error) {
		return []string{ip}, nil
	}
}
