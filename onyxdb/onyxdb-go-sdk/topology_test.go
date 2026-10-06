package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
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

func (m *mockEtcd) Get(_ context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.getErr[key]; err != nil {
		return nil, err
	}
	resp := &clientv3.GetResponse{}
	if len(clientv3.OpGet(key, opts...).RangeBytes()) > 0 { // WithPrefix: every key under key, in key order
		keys := make([]string, 0, len(m.kv))
		for k := range m.kv {
			if strings.HasPrefix(k, key) {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			resp.Kvs = append(resp.Kvs, &mvccpb.KeyValue{Key: []byte(k), Value: []byte(m.kv[k])})
		}
		return resp, nil
	}
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

// failLookup is a DNS stub that never resolves, keeping tests off real DNS.
func failLookup(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("no such host")
}

// reloadRecords splits the MetricTopologyReload emissions into counts and timings.
func reloadRecords(mc *metricCollector) (counts, timings []metricRecord) {
	for _, r := range mc.findAll(MetricTopologyReload) {
		if _, isTiming := r.Value.(time.Duration); isTiming {
			timings = append(timings, r)
		} else {
			counts = append(counts, r)
		}
	}
	return counts, timings
}

func podCount(p *ConnPool, addr string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pools[addr])
}

func hasPool(p *ConnPool, addr string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.pools[addr]
	return ok
}

// ── reload metrics ───────────────────────────────────────────────────────────

func TestReloadVersion_EmitsReloadMetric(t *testing.T) {
	versionKey := model.VersionPrefix("recsys", "catalog", "v1")
	tests := []struct {
		name       string
		setup      func(t *testing.T, m *mockEtcd)
		wantStatus string
		wantTiming bool
	}{
		{"success", func(t *testing.T, m *mockEtcd) { m.put(versionKey, metaJSON(t, 1)) }, "status:ok", true},
		{"get error", func(_ *testing.T, m *mockEtcd) { m.setGetErr(versionKey, errors.New("timeout")) }, "status:error", false},
		{"meta missing", func(_ *testing.T, _ *mockEtcd) {}, "status:error", false},
		{"corrupt meta", func(_ *testing.T, m *mockEtcd) { m.put(versionKey, "not-json") }, "status:error", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withLookup(failLookup, func() {
				m := newMockEtcd()
				tc.setup(t, m)
				tw, _, _ := newWatcherWith(m)
				mc := &metricCollector{}
				tw.SetMetrics(mc.timing, mc.count, []string{"tenant:recsys", "store:catalog"})

				_ = tw.reloadVersion(context.Background(), "v1")

				counts, timings := reloadRecords(mc)
				require.Len(t, counts, 1)
				assert.Equal(t, int64(1), counts[0].Value)
				assert.Equal(t, []string{"tenant:recsys", "store:catalog", tc.wantStatus}, counts[0].Tags)
				if tc.wantTiming {
					require.Len(t, timings, 1)
					assert.Equal(t, []string{"tenant:recsys", "store:catalog", "status:ok"}, timings[0].Tags)
				} else {
					assert.Empty(t, timings)
				}
			})
		})
	}
}

// ── pod watch ────────────────────────────────────────────────────────────────

func podPutEvent() clientv3.WatchResponse {
	return clientv3.WatchResponse{Events: []*clientv3.Event{
		{Type: mvccpb.PUT, Kv: &mvccpb.KeyValue{Key: []byte("pod-a"), Value: []byte("{}")}},
	}}
}

func TestHandlePodWatch_BeforeFirstActiveVersion_DoesNothing(t *testing.T) {
	m := newMockEtcd()
	m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 3))
	tw, r, _ := newWatcherWith(m)
	mc := &metricCollector{}
	tw.SetMetrics(mc.timing, mc.count, nil)

	tw.handlePodWatch(context.Background(), podPutEvent())

	assert.Equal(t, uint32(0), r.ShardCount())
	assert.Empty(t, mc.findAll(MetricTopologyReload), "no reload without an active version")
}

func TestHandlePodWatch_ReloadFailure_KeepsLastKnownTopology(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		versionKey := model.VersionPrefix("recsys", "catalog", "v1")
		m.put(versionKey, metaJSON(t, 2))
		tw, r, _ := newWatcherWith(m)
		require.NoError(t, tw.reloadVersion(context.Background(), "v1"))

		m.setGetErr(versionKey, errors.New("etcd timeout"))
		mc := &metricCollector{}
		tw.SetMetrics(mc.timing, mc.count, nil)
		tw.handlePodWatch(context.Background(), podPutEvent())

		assert.Equal(t, uint32(2), r.ShardCount())
		assert.Equal(t, "v1", tw.activeVID)
		counts, _ := reloadRecords(mc)
		require.Len(t, counts, 1)
		assert.Equal(t, []string{"status:error"}, counts[0].Tags)
	})
}

func TestRun_PodWatchChannelClosed_ReturnsNil(t *testing.T) {
	m := newMockEtcd()
	tw, _, _ := newWatcherWith(m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- tw.Run(ctx) }()

	close(m.podCh())

	select {
	case err := <-done:
		assert.NoError(t, err, "a closed watch with a live context returns ctx.Err() == nil")
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after the pod watch channel closed")
	}
}

// ── pool prune + warm-up ─────────────────────────────────────────────────────

func TestReloadVersion_PrunesPoolToAssignment(t *testing.T) {
	tests := []struct {
		name          string
		wireAssign    bool
		wantStaleKept bool
	}{
		{"assignment resolver wired: departed pod pruned", true, false},
		{"no assignment resolver: pool left untouched", false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withLookup(failLookup, func() {
				m := newMockEtcd()
				m.put(model.VersionPrefix("recsys", "catalog", "v1"),
					metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.10:9091"}}))
				var tw *TopologyWatcher
				if tc.wireAssign {
					tw, _, _ = newWatcherWith(m)
				} else {
					dnsRes := NewDNSResolver(dnsCfg(9091))
					tw = NewTopologyWatcher(m, NewRouter(dnsRes), dnsRes, "recsys", "catalog")
				}
				pool := quietPool(1)
				defer pool.Close()
				stale := idleConn(t, time.Now())
				pool.Put("10.0.9.9:9091", stale)
				tw.SetPoolForWarmUp(pool, 0) // pool wired, warm-up disabled

				require.NoError(t, tw.reloadVersion(context.Background(), "v1"))

				assert.Equal(t, tc.wantStaleKept, hasPool(pool, "10.0.9.9:9091"))
				assert.Equal(t, !tc.wantStaleKept, isClosed(stale))
			})
		})
	}
}

func TestReloadVersion_WarmsUpNewlyAssignedPods(t *testing.T) {
	addr := holdingListener(t)
	withLookup(failLookup, func() {
		m := newMockEtcd()
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{"0": {addr}}))
		tw, _, _ := newWatcherWith(m)
		pool := quietPool(1)
		defer pool.Close()
		tw.SetPoolForWarmUp(pool, 2)

		require.NoError(t, tw.reloadVersion(context.Background(), "v1"))
		require.Eventually(t, func() bool { return podCount(pool, addr) == 2 },
			3*time.Second, 5*time.Millisecond, "warm-up should pre-dial 2 connections")

		// Same assignment again: no newly-added pods, so nothing more is dialled.
		require.NoError(t, tw.reloadVersion(context.Background(), "v1"))
		assert.Equal(t, 2, podCount(pool, addr))
	})
}

func TestWarmUp_UnreachablePodSkippedOthersWarmed(t *testing.T) {
	live := holdingListener(t)
	dead := refusedAddr(t)
	pool := NewConnPoolWithConfig(PoolConfig{MaxPerPod: 4, DialTimeout: time.Second, IdleCheckInterval: time.Hour})
	defer pool.Close()
	tw := NewTopologyWatcher(newMockEtcd(), nil, nil, "recsys", "catalog")
	tw.SetPoolForWarmUp(pool, 2)

	tw.warmUp([]string{dead, live})

	assert.False(t, hasPool(pool, dead), "a failed dial puts nothing in the pool")
	assert.Equal(t, 2, podCount(pool, live))
}

func TestHandleActiveVersionWatch_EventlessResponse_DoesNothing(t *testing.T) {
	m := newMockEtcd()
	m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 3))
	tw, r, _ := newWatcherWith(m)

	tw.handleActiveVersionWatch(context.Background(), clientv3.WatchResponse{}) // e.g. a progress notification

	assert.Equal(t, uint32(0), r.ShardCount())
	assert.Equal(t, "", tw.activeVID)
}
