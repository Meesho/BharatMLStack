package sdk

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// Tests for re-establishing an ended etcd watch. A watch channel closes when
// etcd cancels the watch — most often a client that reconnects after
// compaction has passed the revision it resumes from — not only on shutdown.

// waitRecorder replaces the watcher's backoff sleep: it records each wait and
// returns at once (after a token pause, so a retry loop cannot spin).
type waitRecorder struct {
	mu    sync.Mutex
	waits []time.Duration
}

func (w *waitRecorder) sleep(ctx context.Context, d time.Duration) bool {
	w.mu.Lock()
	w.waits = append(w.waits, d)
	w.mu.Unlock()
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Millisecond):
		return true
	}
}

func (w *waitRecorder) all() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]time.Duration(nil), w.waits...)
}

// fakeClock is a settable clock for the watcher's healthy-watch check.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// startRun runs tw in the background and returns a channel with Run's result.
func startRun(t *testing.T, tw *TopologyWatcher) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- tw.Run(ctx) }()
	t.Cleanup(cancel)
	return cancel, done
}

func requireWatchCalls(t *testing.T, m *mockEtcd, n int) []watchCall {
	t.Helper()
	require.Eventually(t, func() bool { return len(m.watchCalls()) >= n }, 2*time.Second, time.Millisecond,
		"expected %d Watch calls", n)
	return m.watchCalls()
}

func requireRunning(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("Run returned (%v) while its context was live", err)
	default:
	}
}

func restartCounts(mc *metricCollector) []metricRecord {
	return mc.findAll(MetricTopologyWatchRestart)
}

func TestRun_ActiveWatchClosed_ResyncsAndRewatches(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))
		m.put(model.VersionPrefix("recsys", "catalog", "v2"), metaJSON(t, 4))
		m.put(model.VersionPrefix("recsys", "catalog", "v3"), metaJSON(t, 2))
		second := m.addWatchPair()

		tw, r, _ := newWatcherWith(m)
		wr := &waitRecorder{}
		tw.sleep = wr.sleep
		mc := &metricCollector{}
		tw.SetMetrics(mc.timing, mc.count, []string{"tenant:recsys", "store:catalog"})
		_, done := startRun(t, tw)

		require.Eventually(t, func() bool { return r.ShardCount() == 1 }, time.Second, time.Millisecond)
		first := requireWatchCalls(t, m, 2)

		// Promoted while the watch is down: no event will ever announce it.
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v2")
		close(m.activeVersionCh())

		// The resync reads it.
		require.Eventually(t, func() bool { return r.ShardCount() == 4 }, 2*time.Second, time.Millisecond)
		calls := requireWatchCalls(t, m, 4)
		assert.Equal(t, model.ActiveVersionPath("recsys", "catalog"), calls[2].key)
		assert.Equal(t, model.PodWatchPrefix("recsys", "catalog"), calls[3].key)
		assert.True(t, calls[3].prefix, "the pod watch is a prefix watch")
		select {
		case <-first[1].ctx.Done():
		default:
			t.Fatal("the first attempt's pod watch was left running")
		}

		// The new watches are live.
		second.active <- putActiveVersion("v3")
		require.Eventually(t, func() bool { return r.ShardCount() == 2 }, time.Second, time.Millisecond)

		requireRunning(t, done)
		restarts := restartCounts(mc)
		require.Len(t, restarts, 1)
		assert.Equal(t, []string{"tenant:recsys", "store:catalog", "reason:closed"}, restarts[0].Tags)
		assert.Len(t, wr.all(), 1, "one backoff wait before the resync")
	})
}

func TestRun_PodWatchClosed_ResyncsAndRewatches(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.10:9091"}}))
		second := m.addWatchPair()

		tw, r, _ := newWatcherWith(m)
		tw.sleep = (&waitRecorder{}).sleep
		_, done := startRun(t, tw)

		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.1.10:9091"
		}, time.Second, time.Millisecond)
		first := requireWatchCalls(t, m, 2)

		close(m.podCh())
		requireWatchCalls(t, m, 4)
		select {
		case <-first[0].ctx.Done():
		default:
			t.Fatal("the first attempt's activeVersion watch was left running")
		}

		// A pod event on the new watch is followed.
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.20:9091"}}))
		second.pods <- podPutEvent()
		require.Eventually(t, func() bool {
			pod, err := r.PodFor(0)
			return err == nil && pod == "10.0.1.20:9091"
		}, time.Second, time.Millisecond)
		requireRunning(t, done)
	})
}

func TestRun_CancelledWatch_RestartsWithReason(t *testing.T) {
	tests := []struct {
		name   string
		end    func(m *mockEtcd)
		reason string
	}{
		{"compacted", func(m *mockEtcd) {
			m.activeVersionCh() <- clientv3.WatchResponse{Canceled: true, CompactRevision: 12}
		}, "reason:compacted"},
		{"cancelled by the server", func(m *mockEtcd) {
			m.podCh() <- clientv3.WatchResponse{Canceled: true}
		}, "reason:canceled"},
		{"channel closed", func(m *mockEtcd) { close(m.podCh()) }, "reason:closed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withLookup(failLookup, func() {
				m := newMockEtcd()
				m.addWatchPair()
				tw, _, _ := newWatcherWith(m)
				tw.sleep = (&waitRecorder{}).sleep
				mc := &metricCollector{}
				tw.SetMetrics(mc.timing, mc.count, nil)
				_, done := startRun(t, tw)
				requireWatchCalls(t, m, 2)

				tc.end(m)

				requireWatchCalls(t, m, 4)
				require.Eventually(t, func() bool { return len(restartCounts(mc)) == 1 }, time.Second, time.Millisecond)
				assert.Equal(t, []string{tc.reason}, restartCounts(mc)[0].Tags)
				requireRunning(t, done)
			})
		})
	}
}

func TestRun_WatchesStartRightAfterTheRevisionRead(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))
		m.setRev(41)
		m.addWatchPair()

		tw, _, _ := newWatcherWith(m)
		tw.sleep = (&waitRecorder{}).sleep
		startRun(t, tw)

		calls := requireWatchCalls(t, m, 2)
		assert.Equal(t, int64(42), calls[0].rev, "activeVersion watch starts after the initial read")
		assert.Equal(t, int64(42), calls[1].rev, "pod watch starts after the initial read")
		assert.True(t, calls[1].prefix)

		m.setRev(57)
		close(m.activeVersionCh())
		calls = requireWatchCalls(t, m, 4)
		assert.Equal(t, int64(58), calls[2].rev)
		assert.Equal(t, int64(58), calls[3].rev)
		assert.True(t, calls[3].prefix)
	})
}

func TestRun_NoRevisionRead_WatchesFromNow(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd() // no header revision, no active version
		tw, _, _ := newWatcherWith(m)
		startRun(t, tw)
		calls := requireWatchCalls(t, m, 2)
		assert.Equal(t, int64(0), calls[0].rev)
		assert.Equal(t, int64(0), calls[1].rev)
	})
}

func TestRun_RestartBackoff_DoublesCapsAndJitters(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd()
		for i := 0; i < 4; i++ {
			m.addWatchPair()
		}
		tw, _, _ := newWatcherWith(m)
		wr := &waitRecorder{}
		tw.sleep = wr.sleep
		clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
		tw.now = clock.now // the clock never moves: no watch counts as healthy
		tw.restartMin = 100 * time.Millisecond
		tw.restartMax = 400 * time.Millisecond
		startRun(t, tw)

		for round := 0; round < 4; round++ {
			requireWatchCalls(t, m, 2*(round+1))
			close(m.watchChs[2*round])
		}
		requireWatchCalls(t, m, 10)

		waits := wr.all()
		require.Len(t, waits, 4)
		for i, ceiling := range []time.Duration{100, 200, 400, 400} {
			ceiling *= time.Millisecond
			assert.GreaterOrEqual(t, waits[i], ceiling/2, "wait %d at least half its backoff", i)
			assert.LessOrEqual(t, waits[i], ceiling, "wait %d at most its backoff", i)
		}
	})
}

func TestRun_RestartBackoff_ResetsAfterAHealthyWatch(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd()
		m.addWatchPair()
		m.addWatchPair()
		tw, _, _ := newWatcherWith(m)
		wr := &waitRecorder{}
		tw.sleep = wr.sleep
		clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
		tw.now = clock.now
		tw.restartMin = 100 * time.Millisecond
		tw.restartMax = 400 * time.Millisecond
		startRun(t, tw)

		requireWatchCalls(t, m, 2)
		close(m.watchChs[0]) // ends at once: the next backoff doubles
		requireWatchCalls(t, m, 4)
		clock.advance(400 * time.Millisecond) // this watch lives restartMax
		close(m.watchChs[2])
		requireWatchCalls(t, m, 6)

		waits := wr.all()
		require.Len(t, waits, 2)
		assert.LessOrEqual(t, waits[1], 100*time.Millisecond, "a healthy watch starts the backoff over")
	})
}

func TestRun_ResyncReadFails_RetriesBeforeWatching(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd()
		m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 1))
		m.addWatchPair()
		tw, r, _ := newWatcherWith(m)
		wr := &waitRecorder{}
		tw.sleep = wr.sleep
		startRun(t, tw)
		requireWatchCalls(t, m, 2)
		require.Eventually(t, func() bool { return r.ShardCount() == 1 }, time.Second, time.Millisecond)

		m.setGetErr(model.ActiveVersionPath("recsys", "catalog"), errors.New("etcd unreachable"))
		close(m.activeVersionCh())

		// Retries the read; opens no watch it could not start from a revision.
		require.Eventually(t, func() bool { return len(wr.all()) >= 3 }, time.Second, time.Millisecond)
		assert.Len(t, m.watchCalls(), 2)

		m.setGetErr(model.ActiveVersionPath("recsys", "catalog"), nil)
		m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaJSON(t, 3))
		requireWatchCalls(t, m, 4)
		require.Eventually(t, func() bool { return r.ShardCount() == 3 }, time.Second, time.Millisecond)
	})
}

func TestRun_CancelledDuringBackoff_Returns(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd()
		tw, _, _ := newWatcherWith(m)
		entered := make(chan struct{})
		tw.sleep = func(ctx context.Context, _ time.Duration) bool {
			close(entered)
			<-ctx.Done()
			return false
		}
		cancel, done := startRun(t, tw)
		requireWatchCalls(t, m, 2)
		close(m.activeVersionCh())
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("no backoff after the watch ended")
		}
		cancel()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(time.Second):
			t.Fatal("Run did not return when cancelled during its backoff")
		}
		assert.Len(t, m.watchCalls(), 2, "no watch opened after cancellation")
	})
}

func TestRun_Shutdown_ClosedWatchesAreNotRestarted(t *testing.T) {
	withLookup(failLookup, func() {
		m := newMockEtcd()
		m.closeOnCancel = true // clientv3 closes every watch channel on cancel
		tw, _, _ := newWatcherWith(m)
		mc := &metricCollector{}
		tw.SetMetrics(mc.timing, mc.count, nil)
		cancel, done := startRun(t, tw)
		requireWatchCalls(t, m, 2)

		cancel()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(time.Second):
			t.Fatal("Run did not return on shutdown")
		}
		assert.Empty(t, restartCounts(mc), "shutdown is not a restart")
		assert.Len(t, m.watchCalls(), 2)
	})
}

func TestRealSleep_WaitsOrStopsOnCancel(t *testing.T) {
	assert.True(t, sleepCtx(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.False(t, sleepCtx(ctx, time.Hour))
}

func TestJitter_StaysWithinHalfToFullBackoff(t *testing.T) {
	for i := 0; i < 200; i++ {
		d := jitter(100 * time.Millisecond)
		assert.GreaterOrEqual(t, d, 50*time.Millisecond)
		assert.LessOrEqual(t, d, 100*time.Millisecond)
	}
	assert.Equal(t, time.Duration(0), jitter(0))
}
