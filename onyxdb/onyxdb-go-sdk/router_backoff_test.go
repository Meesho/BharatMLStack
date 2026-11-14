package sdk

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B2 and B3 (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3).
// An unhealthy mark used to last until the next refresh, which in production fires
// every few seconds (every pod-key re-put and every topology reload), and a
// shard whose replicas were all marked still got every read. A mark now keeps
// the pod out for a backoff window that grows while it keeps failing and ends
// on its first success. With ShardFailFast, a shard whose replicas are all
// inside their windows fails at once with ErrShardUnavailable, and one probe
// per shard at a time finds out whether a pod is back.

// backoffRouter routes shard 0 over pods on a fake clock, with waits pinned to
// their nominal value (no jitter) and metrics collected.
func backoffRouter(pods ...string) (*Router, *fakeClock, *metricCollector) {
	r := routerWith(1, map[uint32][]string{0: pods})
	clk := &fakeClock{t: redialT0}
	r.now = clk.now
	r.float = func() float64 { return 0.5 }
	mc := &metricCollector{}
	r.SetMetrics(mc.count, []string{"tenant:t", "store:s"})
	return r, clk, mc
}

// window is how long pod is still kept out at the router's current time; 0
// when it is not marked or its window has passed.
func window(r *Router, pod string) time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h := r.unhealthy[pod]
	if h == nil {
		return 0
	}
	return max(0, h.until.Sub(r.now()))
}

func TestMarkUnhealthy_WindowDoublesWhileThePodKeepsFailing(t *testing.T) {
	r, clk, _ := backoffRouter("a", "b")
	for _, want := range []time.Duration{
		time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second,
		30 * time.Second, 30 * time.Second,
	} {
		r.MarkUnhealthy("a")
		require.Equal(t, want, window(r, "a"))

		// More failures inside the window are the same episode: requests that
		// were in flight when the pod stalled must not stretch it.
		clk.advance(want / 2)
		r.MarkUnhealthy("a")
		require.Equal(t, want-want/2, window(r, "a"))
		clk.advance(want - want/2)
	}
}

func TestMarkUnhealthy_WindowIsJittered(t *testing.T) {
	cases := []struct {
		u    float64
		want time.Duration
	}{{0, 800 * time.Millisecond}, {0.5, time.Second}, {0.99999, 1199996 * time.Microsecond}}
	for _, tc := range cases {
		r, _, _ := backoffRouter("a")
		r.float = func() float64 { return tc.u }
		r.MarkUnhealthy("a")
		assert.InDelta(t, float64(tc.want), float64(window(r, "a")), float64(time.Microsecond))
	}
}

func TestSetUnhealthyBackoff(t *testing.T) {
	r, clk, _ := backoffRouter("a")
	r.SetUnhealthyBackoff(200*time.Millisecond, 500*time.Millisecond)
	for _, want := range []time.Duration{200 * time.Millisecond, 400 * time.Millisecond, 500 * time.Millisecond} {
		r.MarkUnhealthy("a")
		require.Equal(t, want, window(r, "a"))
		clk.advance(want)
	}

	r.SetUnhealthyBackoff(0, 0) // zero: the defaults
	r.MarkHealthy("a")
	r.MarkUnhealthy("a")
	assert.Equal(t, time.Second, window(r, "a"))
	assert.Equal(t, 30*time.Second, r.unhealthyMax)

	r.SetUnhealthyBackoff(5*time.Second, time.Second) // a cap below the base is raised to it
	assert.Equal(t, 5*time.Second, r.unhealthyMax)
}

func TestMarkHealthy_EndsTheBackoff(t *testing.T) {
	r, clk, _ := backoffRouter("a", "b")
	for i := 0; i < 3; i++ {
		r.MarkUnhealthy("a")
		clk.advance(window(r, "a"))
	}
	r.MarkUnhealthy("a")
	require.Equal(t, 8*time.Second, window(r, "a"))

	r.MarkHealthy("a")
	assert.False(t, isUnhealthy(r, "a"))
	assert.Equal(t, int32(0), r.marked.Load())
	r.MarkUnhealthy("a")
	assert.Equal(t, time.Second, window(r, "a"), "a success starts the schedule over")

	r.MarkHealthy("never-marked") // no-op
	assert.Equal(t, int32(1), r.marked.Load())
}

func TestPodFor_ShardFailFast_EveryReplicaInItsWindowFailsAtOnce(t *testing.T) {
	r, _, mc := backoffRouter("10.0.1.10:9091", "10.0.1.11:9091")
	r.MarkUnhealthy("10.0.1.10:9091")
	r.MarkUnhealthy("10.0.1.11:9091")

	for i := 0; i < 5; i++ {
		_, err := r.PodFor(0)
		require.ErrorIs(t, err, ErrShardUnavailable)
	}
	got := mc.findAll(MetricRouterShardUnavailable)
	require.Len(t, got, 5)
	assert.Equal(t, int64(1), got[0].Value)
	assert.Equal(t, []string{"tenant:t", "store:s"}, got[0].Tags, "no pod address in the tags")
}

// Once the earliest window has passed, exactly one request goes through to
// that pod, however many race for it; the rest keep failing fast until the
// probe's outcome is in.
func TestPodFor_ShardFailFast_LetsExactlyOneProbeThrough(t *testing.T) {
	r, clk, mc := backoffRouter("a", "b")
	r.MarkUnhealthy("a") // until t0+1s
	clk.advance(500 * time.Millisecond)
	r.MarkUnhealthy("b") // until t0+1.5s
	clk.advance(500 * time.Millisecond)

	var mu sync.Mutex
	picked := map[string]int{}
	unavailable := 0
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pod, err := r.PodFor(0)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				assert.ErrorIs(t, err, ErrShardUnavailable)
				unavailable++
				return
			}
			picked[pod]++
		}()
	}
	wg.Wait()
	assert.Equal(t, map[string]int{"a": 1}, picked, "one probe, to the pod whose window ended first")
	assert.Equal(t, 63, unavailable)

	r.MarkHealthy("a") // the probe answered
	assert.Equal(t, []string{"tenant:t", "store:s", "result:ok"}, mc.findAll(MetricRouterProbe)[0].Tags)
	for i := 0; i < 10; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		assert.Equal(t, "a", pod, "b is still inside its window")
	}
}

func TestPodFor_ShardFailFast_AFailedProbeDoublesThePodsWindow(t *testing.T) {
	r, clk, mc := backoffRouter("a")
	r.MarkUnhealthy("a")
	clk.advance(time.Second)
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	require.Equal(t, "a", pod)

	r.MarkUnhealthy("a") // the probe failed
	assert.Equal(t, 2*time.Second, window(r, "a"))
	assert.Equal(t, int64(1), countTotal(mc, MetricRouterProbe, "result:fail"))
	_, err = r.PodFor(0)
	require.ErrorIs(t, err, ErrShardUnavailable)

	clk.advance(2 * time.Second)
	pod, err = r.PodFor(0)
	require.NoError(t, err, "the next probe is due when the doubled window ends")
	assert.Equal(t, "a", pod)
}

// A probe whose outcome never comes back (a code path that forgot to report)
// does not hold the shard shut: after the probe lease another probe goes out.
func TestPodFor_ShardFailFast_AProbeWithoutAnOutcomeLapses(t *testing.T) {
	r, clk, _ := backoffRouter("a")
	r.probeLease = 300 * time.Millisecond
	r.MarkUnhealthy("a")
	clk.advance(time.Second)
	_, err := r.PodFor(0)
	require.NoError(t, err)

	clk.advance(299 * time.Millisecond)
	_, err = r.PodFor(0)
	require.ErrorIs(t, err, ErrShardUnavailable)
	clk.advance(time.Millisecond)
	_, err = r.PodFor(0)
	assert.NoError(t, err)
}

// A probe that never reached the pod (this client's own dial rate refused it)
// ends without a verdict: the window is unchanged and the next request probes.
func TestEndProbe_FreesTheProbeWithoutAVerdict(t *testing.T) {
	r, clk, mc := backoffRouter("a")
	r.MarkUnhealthy("a")
	clk.advance(time.Second)
	_, err := r.PodFor(0)
	require.NoError(t, err)

	r.endProbe("a")
	r.endProbe("not-marked")
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Equal(t, "a", pod)
	r.mu.RLock()
	assert.Equal(t, 1, r.unhealthy["a"].failures)
	r.mu.RUnlock()
	assert.Empty(t, mc.findAll(MetricRouterProbe))

	r.MarkHealthy("a")
	r.endProbe("a") // nothing marked: no-op
	assert.False(t, isUnhealthy(r, "a"))
}

// A pod past its window that still has a healthy peer comes back through a
// probe too, not with its full share at once.
func TestPodFor_ShardFailFast_APodWithAHealthyPeerComesBackThroughAProbe(t *testing.T) {
	r, clk, _ := backoffRouter("a", "b")
	r.MarkUnhealthy("a")
	for i := 0; i < 10; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		require.Equal(t, "b", pod, "inside its window a gets nothing")
	}

	clk.advance(time.Second)
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Equal(t, "a", pod, "the first read after the window is the probe")
	for i := 0; i < 10; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		require.Equal(t, "b", pod, "while the probe is out, the rest go to the healthy peer")
	}
	r.MarkHealthy("a")
	seen := map[string]int{}
	for i := 0; i < 10; i++ {
		pod, _ := r.PodFor(0)
		seen[pod]++
	}
	assert.Equal(t, map[string]int{"a": 5, "b": 5}, seen)
}

// ShardFailFast=false keeps the old routing: a shard whose replicas are all
// marked still gets a pod (best effort), and a pod past its window takes
// traffic again without a probe.
func TestPodFor_ShardFailFastOff_KeepsTheOldFallback(t *testing.T) {
	r, clk, mc := backoffRouter("a", "b")
	r.SetShardFailFast(false)
	r.MarkUnhealthy("a")
	r.MarkUnhealthy("b")

	seen := map[string]int{}
	for i := 0; i < 10; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		seen[pod]++
	}
	assert.Equal(t, map[string]int{"a": 5, "b": 5}, seen)
	assert.Empty(t, mc.findAll(MetricRouterShardUnavailable))

	r.MarkHealthy("b")
	clk.advance(time.Second) // a's window is over: it routes like any pod, no probe
	seen = map[string]int{}
	for i := 0; i < 10; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		seen[pod]++
	}
	assert.Equal(t, map[string]int{"a": 5, "b": 5}, seen)

	r.MarkUnhealthy("a") // a failure after the window still escalates
	assert.Equal(t, 2*time.Second, window(r, "a"))
}

// A refresh (every DNS tick, topology reload and pod-key re-put) drops only
// marks whose window has passed and marks of pods no longer routed to; it
// keeps active backoffs, and a pod being probed.
func TestClearUnhealthy_KeepsActiveBackoffs(t *testing.T) {
	r := routerWith(2, map[uint32][]string{0: {"active", "expired"}, 1: {"probing"}})
	clk := &fakeClock{t: redialT0}
	r.now = clk.now
	r.float = func() float64 { return 0.5 }

	r.MarkUnhealthy("expired")
	r.MarkUnhealthy("probing")
	clk.advance(time.Second)
	pod, err := r.PodFor(1)
	require.NoError(t, err)
	require.Equal(t, "probing", pod)
	r.MarkUnhealthy("active")
	r.MarkUnhealthy("unassigned")

	r.ClearUnhealthy()

	assert.True(t, isUnhealthy(r, "active"), "inside its window: kept")
	assert.True(t, isUnhealthy(r, "probing"), "probe in flight: kept")
	assert.False(t, isUnhealthy(r, "expired"), "window over: dropped")
	assert.False(t, isUnhealthy(r, "unassigned"), "no longer routed to: dropped")
	assert.Equal(t, int32(2), r.marked.Load())

	clk.advance(time.Second) // the probe lease and every window are over
	r.ClearUnhealthy()
	assert.Equal(t, int32(0), r.marked.Load())
	r.ClearUnhealthy() // nothing marked: no-op
}

func TestClearUnhealthy_DropsEveryMarkBeforeTheTopologyLoads(t *testing.T) {
	r := routerWith(0, nil)
	r.MarkUnhealthy("a")
	r.ClearUnhealthy()
	assert.False(t, isUnhealthy(r, "a"))
}

func TestRouter_RedialDefaults(t *testing.T) {
	r := NewRouter(&fakeResolver{})
	assert.True(t, r.failFast)
	assert.Equal(t, time.Second, r.unhealthyBase)
	assert.Equal(t, 30*time.Second, r.unhealthyMax)
	assert.Equal(t, time.Second, r.probeLease)
}

// A shard with every pod inside its window and a ramp in progress fails fast
// too; slow-start weighting only applies among pods that may be routed to.
func TestPodFor_ShardFailFast_WithASlowStartRamp(t *testing.T) {
	r, clk, _ := backoffRouter("old", "new")
	r.SetSlowStart(map[string]SlowStart{"new": {Since: redialT0, Window: 300 * time.Second}})
	r.MarkUnhealthy("old")
	r.MarkUnhealthy("new")
	_, err := r.PodFor(0)
	require.ErrorIs(t, err, ErrShardUnavailable)

	clk.advance(time.Second)
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Contains(t, []string{"old", "new"}, pod, "the probe")
	_, err = r.PodFor(0)
	assert.ErrorIs(t, err, ErrShardUnavailable)
}

// Between PodFor's read-locked pass and the write lock a pod can heal (a
// request in flight succeeded): it is routed to rather than failing the shard.
func TestProbeOrFail_RoutesToAPodThatHealedInBetween(t *testing.T) {
	r, _, mc := backoffRouter("a", "b")
	r.MarkUnhealthy("a")
	pod, err := r.probeOrFail([]string{"a", "b"}, 0)
	require.NoError(t, err)
	assert.Equal(t, "b", pod)
	assert.Empty(t, mc.findAll(MetricRouterShardUnavailable))
}

// probeOrFail's fallback spreads like PodFor: over the routable pods evenly,
// not onto the marked pod's successor.
func TestProbeOrFail_SpreadsOverTheRoutablePods(t *testing.T) {
	pods := []string{"a", "b", "c", "d"}
	r, _, mc := backoffRouter(pods...)
	r.MarkUnhealthy("b")

	seen := map[string]int{}
	for start := 0; start < 300; start++ {
		pod, err := r.probeOrFail(pods, start)
		require.NoError(t, err)
		seen[pod]++
	}

	assert.Equal(t, map[string]int{"a": 100, "c": 100, "d": 100}, seen)
	assert.Empty(t, mc.findAll(MetricRouterShardUnavailable))
}
