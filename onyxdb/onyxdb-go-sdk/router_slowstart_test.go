package sdk

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Slow start (docs/design/onyxdb-pod-slow-start.md): a pod that has just
// started serving from an empty block cache gets a weight that ramps from 1%
// to a full share over its window.

var rampT0 = time.Date(2026, 10, 6, 15, 0, 0, 0, time.UTC)

// rampRouter routes shard 0 over pods, with a fixed clock and a seeded
// random source so splits are reproducible.
func rampRouter(pods []string, now time.Time) *Router {
	r := routerWith(1, map[uint32][]string{0: pods})
	r.now = func() time.Time { return now }
	r.float = rand.New(rand.NewPCG(1, 2)).Float64
	return r
}

func share(t *testing.T, r *Router, pod string, n int) float64 {
	t.Helper()
	hits := 0
	for i := 0; i < n; i++ {
		got, err := r.PodFor(0)
		require.NoError(t, err)
		if got == pod {
			hits++
		}
	}
	return float64(hits) / float64(n)
}

func TestSlowStartWeight(t *testing.T) {
	w := 300 * time.Second
	cases := []struct {
		name string
		s    SlowStart
		now  time.Time
		want float64
	}{
		{"no window: full weight", SlowStart{Since: rampT0}, rampT0, 1},
		{"at the start: the 1% floor", SlowStart{Since: rampT0, Window: w}, rampT0, minSlowStartWeight},
		{"just after the start: still the floor", SlowStart{Since: rampT0, Window: w}, rampT0.Add(time.Second), minSlowStartWeight},
		{"a quarter in", SlowStart{Since: rampT0, Window: w}, rampT0.Add(75 * time.Second), 0.25},
		{"half way", SlowStart{Since: rampT0, Window: w}, rampT0.Add(150 * time.Second), 0.5},
		{"window over", SlowStart{Since: rampT0, Window: w}, rampT0.Add(w), 1},
		{"long after", SlowStart{Since: rampT0, Window: w}, rampT0.Add(time.Hour), 1},
		{"since in the future (clock skew): the floor", SlowStart{Since: rampT0.Add(time.Minute), Window: w}, rampT0, minSlowStartWeight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.want, tc.s.weight(tc.now), 1e-9)
		})
	}
}

func TestPodFor_SlowStartSplitFollowsTheWeight(t *testing.T) {
	const n = 100_000
	window := 300 * time.Second
	cases := []struct {
		name    string
		pods    []string
		elapsed time.Duration
		want    float64 // the ramping pod's share: w / (w + number of full pods)
	}{
		{"1 old + 1 new at the start", []string{"old", "new"}, 0, 0.01 / 1.01},
		{"1 old + 1 new a quarter in", []string{"old", "new"}, 75 * time.Second, 0.25 / 1.25},
		{"1 old + 1 new half way", []string{"old", "new"}, 150 * time.Second, 0.5 / 1.5},
		{"2 old + 1 new half way", []string{"old1", "new", "old2"}, 150 * time.Second, 0.5 / 2.5},
		{"window over: an equal share", []string{"old", "new"}, window, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := rampRouter(tc.pods, rampT0.Add(tc.elapsed))
			r.SetSlowStart(map[string]SlowStart{"new": {Since: rampT0, Window: window}})
			assert.InDelta(t, tc.want, share(t, r, "new", n), 0.01)
		})
	}
}

// With nothing ramping, or only finished ramps, PodFor is the plain
// round-robin it was before slow start: same pods in the same order.
func TestPodFor_NoActiveRampIsPlainRoundRobin(t *testing.T) {
	pods := []string{"a", "b", "c"}
	cases := map[string]map[string]SlowStart{
		"no ramps":                       nil,
		"finished ramp":                  {"b": {Since: rampT0.Add(-time.Hour), Window: 300 * time.Second}},
		"ramp of a pod not in the shard": {"zzz": {Since: rampT0, Window: 300 * time.Second}},
	}
	for name, ramps := range cases {
		t.Run(name, func(t *testing.T) {
			ref := routerWith(1, map[uint32][]string{0: pods})
			r := rampRouter(pods, rampT0)
			r.SetSlowStart(ramps)
			for i := 0; i < 12; i++ {
				want, _ := ref.PodFor(0)
				got, _ := r.PodFor(0)
				assert.Equal(t, want, got, "pick %d", i)
			}
		})
	}
}

func TestPodFor_SlowStartRespectsUnhealthyMarks(t *testing.T) {
	ramp := map[string]SlowStart{"new": {Since: rampT0, Window: 300 * time.Second}}

	t.Run("old pod marked down: every read goes to the ramping pod", func(t *testing.T) {
		r := rampRouter([]string{"old", "new"}, rampT0)
		r.SetSlowStart(ramp)
		r.MarkUnhealthy("old")
		assert.Equal(t, 1.0, share(t, r, "new", 1000))
	})
	t.Run("ramping pod marked down: none go to it", func(t *testing.T) {
		r := rampRouter([]string{"old", "new"}, rampT0.Add(150*time.Second))
		r.SetSlowStart(ramp)
		r.MarkUnhealthy("new")
		assert.Equal(t, 0.0, share(t, r, "new", 1000))
	})
	t.Run("every pod marked down: best-effort pick, no error", func(t *testing.T) {
		r := rampRouter([]string{"old", "new"}, rampT0)
		r.SetSlowStart(ramp)
		r.MarkUnhealthy("old")
		r.MarkUnhealthy("new")
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		assert.Contains(t, []string{"old", "new"}, pod)
	})
}

// Weights are relative: when every pod of a shard is ramping (a 1-replica
// shard whose old pod is already gone, a new store), they split evenly.
func TestPodFor_AllPodsRampingSplitEvenly(t *testing.T) {
	r := rampRouter([]string{"a", "b"}, rampT0.Add(10*time.Second))
	r.SetSlowStart(map[string]SlowStart{
		"a": {Since: rampT0, Window: 300 * time.Second},
		"b": {Since: rampT0, Window: 300 * time.Second},
	})
	assert.InDelta(t, 0.5, share(t, r, "a", 100_000), 0.01)
}

// A draw at the very top of the range must still land on a healthy pod.
func TestPodFor_SlowStartDrawAtTheEdgeLandsOnAHealthyPod(t *testing.T) {
	r := rampRouter([]string{"new", "old"}, rampT0.Add(150*time.Second))
	r.SetSlowStart(map[string]SlowStart{"new": {Since: rampT0, Window: 300 * time.Second}})
	r.MarkUnhealthy("old")
	r.float = func() float64 { return 1 } // past [0,1): models float rounding at the edge
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Equal(t, "new", pod)
}

func TestPodFor_RampPickHookCountsPicksOnRampingPods(t *testing.T) {
	r := rampRouter([]string{"old", "new"}, rampT0.Add(150*time.Second))
	r.SetSlowStart(map[string]SlowStart{"new": {Since: rampT0, Window: 300 * time.Second}})
	hooked := 0
	r.SetRampPickHook(func() { hooked++ })

	toNew := 0
	for i := 0; i < 1000; i++ {
		if pod, _ := r.PodFor(0); pod == "new" {
			toNew++
		}
	}
	assert.Positive(t, toNew)
	assert.Equal(t, toNew, hooked, "one hook call per pick on the ramping pod, none for the full pod")
}
