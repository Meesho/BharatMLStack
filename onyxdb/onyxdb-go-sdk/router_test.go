package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResolver is a test ShardResolver backed by a fixed map.
type fakeResolver struct{ m map[uint32][]string }

func (f *fakeResolver) Resolve(shardID uint32) []string { return f.m[shardID] }

func routerWith(shardCount uint32, m map[uint32][]string) *Router {
	r := NewRouter(&fakeResolver{m: m})
	r.SetShardCount(shardCount)
	return r
}

func TestShardFor_ZeroShardCount(t *testing.T) {
	r := NewRouter(&fakeResolver{})
	assert.Equal(t, uint32(0), r.ShardFor([]byte("anything")))
}

func TestShardFor_Deterministic(t *testing.T) {
	r := routerWith(10, nil)
	key := []byte("catalog_id|123")
	assert.Equal(t, r.ShardFor(key), r.ShardFor(key))
	assert.Less(t, r.ShardFor(key), uint32(10))
}

func TestShardFor_CRC32Golden(t *testing.T) {
	// crc32.ChecksumIEEE("hello") = 907060870; % 16 = 6.
	r := routerWith(16, nil)
	assert.Equal(t, uint32(907060870%16), r.ShardFor([]byte("hello")))
}

func TestShardCount(t *testing.T) {
	r := routerWith(7, nil)
	assert.Equal(t, uint32(7), r.ShardCount())
}

func TestPodFor_NoPods(t *testing.T) {
	r := routerWith(1, nil) // resolver returns nil for shard 0
	_, err := r.PodFor(0)
	assert.ErrorIs(t, err, ErrNoHealthyPod)
}

func TestPodFor_SinglePod(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"10.0.0.1:9091"}})
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1:9091", pod)
}

func TestPodFor_RoundRobin(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b", "c"}})
	seen := map[string]int{}
	for i := 0; i < 30; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		seen[pod]++
	}
	assert.Equal(t, 3, len(seen))
	for _, count := range seen {
		assert.Equal(t, 10, count, "round-robin should be even")
	}
}

func TestPodFor_SkipsUnhealthy(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b"}})
	r.MarkUnhealthy("a")
	for i := 0; i < 10; i++ {
		pod, err := r.PodFor(0)
		require.NoError(t, err)
		assert.Equal(t, "b", pod)
	}
}

// pickCounts makes n PodFor picks for shardID and counts them by pod.
func pickCounts(t *testing.T, r *Router, shardID uint32, n int) map[string]int {
	t.Helper()
	seen := map[string]int{}
	for i := 0; i < n; i++ {
		pod, err := r.PodFor(shardID)
		require.NoError(t, err)
		seen[pod]++
	}
	return seen
}

// A marked pod's share spreads evenly over the shard's routable pods. Every
// client resolves a shard's pods in the same order, so handing the share to
// the marked pod's successor in that list would double one pod's load across
// every client at once.
func TestPodFor_SpreadsAMarkedPodsShareEvenly(t *testing.T) {
	pods := []string{"a", "b", "c", "d", "e", "f"}
	tests := []struct {
		name   string
		marked []string
		want   map[string]int
	}{
		{
			name:   "one pod marked",
			marked: []string{"c"},
			want:   map[string]int{"a": 1200, "b": 1200, "d": 1200, "e": 1200, "f": 1200},
		},
		{
			name:   "the last pod marked: its share does not all wrap to the first",
			marked: []string{"f"},
			want:   map[string]int{"a": 1200, "b": 1200, "c": 1200, "d": 1200, "e": 1200},
		},
		{
			name:   "two pods marked side by side",
			marked: []string{"c", "d"},
			want:   map[string]int{"a": 1500, "b": 1500, "e": 1500, "f": 1500},
		},
		{
			name:   "two pods marked apart",
			marked: []string{"b", "e"},
			want:   map[string]int{"a": 1500, "c": 1500, "d": 1500, "f": 1500},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := routerWith(1, map[uint32][]string{0: pods})
			r.now = (&fakeClock{t: redialT0}).now // the windows outlast the picks: no probe comes due
			for _, p := range tc.marked {
				r.MarkUnhealthy(p)
			}

			got := pickCounts(t, r, 0, 6000)

			assert.Equal(t, tc.want, got)
		})
	}
}

// A mark on another shard's pod leaves this shard's round-robin even.
func TestPodFor_AMarkOnAnotherShardLeavesThisShardEven(t *testing.T) {
	r := routerWith(2, map[uint32][]string{0: {"a", "b", "c"}, 1: {"x", "y"}})
	r.now = (&fakeClock{t: redialT0}).now
	r.MarkUnhealthy("x")

	got := pickCounts(t, r, 0, 30)

	assert.Equal(t, map[string]int{"a": 10, "b": 10, "c": 10}, got)
}

// With ShardFailFast off, a shard whose pods are all marked still gets one of
// them (best effort), as before fail-fast existed. With it on (the default)
// see router_backoff_test.go.
func TestPodFor_AllUnhealthy_FallsBackWithShardFailFastOff(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b"}})
	r.SetShardFailFast(false)
	r.MarkUnhealthy("a")
	r.MarkUnhealthy("b")
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Contains(t, []string{"a", "b"}, pod)
}

func TestPodFor_AllUnhealthy_FailsFastByDefault(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b"}})
	r.MarkUnhealthy("a")
	r.MarkUnhealthy("b")
	_, err := r.PodFor(0)
	assert.ErrorIs(t, err, ErrShardUnavailable)
}

// ClearUnhealthy makes a pod routable again once its window has passed (and
// keeps it out while the window lasts: see TestClearUnhealthy_KeepsActiveBackoffs).
func TestClearUnhealthy(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b"}})
	clk := &fakeClock{t: redialT0}
	r.now = clk.now
	r.MarkUnhealthy("a")
	clk.advance(2 * time.Second) // past the jittered 1 s window
	r.ClearUnhealthy()
	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		pod, _ := r.PodFor(0)
		seen[pod] = true
	}
	assert.True(t, seen["a"], "a should be healthy again after ClearUnhealthy")
}

func TestSetShardCount(t *testing.T) {
	r := routerWith(1, nil)
	r.SetShardCount(8)
	assert.Equal(t, uint32(8), r.ShardCount())
}
