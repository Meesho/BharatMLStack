package coverage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// stubLister is a test double for PodLister.
type stubLister struct {
	pods map[string]model.PodData
	err  error
}

func (s *stubLister) ListPods(_ context.Context, _, _ string) (map[string]model.PodData, error) {
	return s.pods, s.err
}

func TestChecker_AllShardsWarm(t *testing.T) {
	lister := &stubLister{pods: map[string]model.PodData{
		"t-s-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		"t-s-shard-1-0": {PodIP: "10.0.0.2", WarmVersions: []string{"v1"}},
		"t-s-shard-2-0": {PodIP: "10.0.0.3", WarmVersions: []string{"v1"}},
	}}
	c := New(lister)
	result, err := c.Check(context.Background(), "t", "s", "v1", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 3, result.Warm)
	assert.Empty(t, result.Missing)
	assert.True(t, result.IsComplete())
	assert.Equal(t, 1.0, result.Ratio())
}

func TestChecker_NoPods(t *testing.T) {
	lister := &stubLister{pods: map[string]model.PodData{}}
	c := New(lister)
	result, err := c.Check(context.Background(), "t", "s", "v1", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 0, result.Warm)
	assert.Equal(t, []string{"0", "1"}, result.Missing)
	assert.False(t, result.IsComplete())
	assert.Equal(t, 0.0, result.Ratio())
}

func TestChecker_PartialCoverage(t *testing.T) {
	lister := &stubLister{pods: map[string]model.PodData{
		"t-s-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		// shard 1 missing
	}}
	c := New(lister)
	result, err := c.Check(context.Background(), "t", "s", "v1", 2)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Warm)
	assert.Equal(t, []string{"1"}, result.Missing)
	assert.False(t, result.IsComplete())
	assert.InDelta(t, 0.5, result.Ratio(), 0.001)
}

func TestChecker_ListerError(t *testing.T) {
	lister := &stubLister{err: errors.New("etcd timeout")}
	c := New(lister)
	_, err := c.Check(context.Background(), "t", "s", "v1", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing pods")
}

func TestChecker_ZeroShardCount(t *testing.T) {
	lister := &stubLister{pods: nil}
	c := New(lister)
	result, err := c.Check(context.Background(), "t", "s", "v1", 0)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Equal(t, 0, result.Warm)
	assert.True(t, result.IsComplete())
	assert.Equal(t, 1.0, result.Ratio())
}

func TestResult_MissingSorted(t *testing.T) {
	// Provide pods for shards 2 and 4 only, shards 0,1,3 are missing → sorted
	pods := map[string]model.PodData{
		"t-s-shard-2-0": {PodIP: "10.0.0.3", WarmVersions: []string{"v1"}},
		"t-s-shard-4-0": {PodIP: "10.0.0.5", WarmVersions: []string{"v1"}},
	}
	lister := &stubLister{pods: pods}
	c := New(lister)
	result, err := c.Check(context.Background(), "t", "s", "v1", 5)
	require.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "3"}, result.Missing)
}
