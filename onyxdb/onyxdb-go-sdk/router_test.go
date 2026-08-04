package sdk

import (
	"testing"

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

func TestPodFor_AllUnhealthy_FallsBack(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b"}})
	r.MarkUnhealthy("a")
	r.MarkUnhealthy("b")
	pod, err := r.PodFor(0)
	require.NoError(t, err)
	assert.Contains(t, []string{"a", "b"}, pod)
}

func TestClearUnhealthy(t *testing.T) {
	r := routerWith(1, map[uint32][]string{0: {"a", "b"}})
	r.MarkUnhealthy("a")
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
