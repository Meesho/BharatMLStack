package placement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

func TestExtractShardID(t *testing.T) {
	tests := []struct {
		podID string
		want  string
	}{
		{"fs-features-shard-0-0", "0"},
		{"fs-features-shard-5-1", "5"},
		{"recsys-catalog-shard-12-0", "12"},
		{"t-s-shard-3-0", "3"},              // minimal valid form with tenant-store prefix
		{"no-marker-at-all", ""},           // no "-shard-" marker anywhere
		{"prefix-shard-", ""},              // marker present but empty rest
		{"prefix-shard-42", "42"},          // no trailing replica index
		{"", ""},                           // empty string
	}

	for _, tc := range tests {
		t.Run(tc.podID, func(t *testing.T) {
			assert.Equal(t, tc.want, ExtractShardID(tc.podID))
		})
	}
}

func TestDeriveAssignment_EmptyPods(t *testing.T) {
	result := DeriveAssignment(3, nil, "v1")
	require.Len(t, result, 3)
	assert.Empty(t, result["0"])
	assert.Empty(t, result["1"])
	assert.Empty(t, result["2"])
}

func TestDeriveAssignment_AllShardsWarm(t *testing.T) {
	pods := map[string]model.PodData{
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		"fs-features-shard-1-0": {PodIP: "10.0.0.2", WarmVersions: []string{"v1"}},
		"fs-features-shard-2-0": {PodIP: "10.0.0.3", WarmVersions: []string{"v1"}},
	}

	result := DeriveAssignment(3, pods, "v1")
	assert.Equal(t, []string{"10.0.0.1:9091"}, result["0"])
	assert.Equal(t, []string{"10.0.0.2:9091"}, result["1"])
	assert.Equal(t, []string{"10.0.0.3:9091"}, result["2"])
}

func TestDeriveAssignment_VersionMismatch(t *testing.T) {
	pods := map[string]model.PodData{
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v2"}},
	}

	result := DeriveAssignment(1, pods, "v1")
	assert.Empty(t, result["0"])
}

func TestDeriveAssignment_MultipleReplicasPerShard(t *testing.T) {
	pods := map[string]model.PodData{
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		"fs-features-shard-0-1": {PodIP: "10.0.0.2", WarmVersions: []string{"v1"}},
	}

	result := DeriveAssignment(1, pods, "v1")
	assert.Len(t, result["0"], 2)
	assert.Contains(t, result["0"], "10.0.0.1:9091")
	assert.Contains(t, result["0"], "10.0.0.2:9091")
}

func TestDeriveAssignment_CoLocatedPodsUseReportedPort(t *testing.T) {
	// Two shards co-located on one host (host networking) — same IP, distinct
	// ports. The reported Port must be honored so clients reach the right server.
	pods := map[string]model.PodData{
		"test-store-shard-0-1": {PodIP: "10.0.0.1", Port: 9092, WarmVersions: []string{"v1"}},
		"test-store-shard-1-1": {PodIP: "10.0.0.1", Port: 9093, WarmVersions: []string{"v1"}},
	}

	result := DeriveAssignment(2, pods, "v1")
	assert.Equal(t, []string{"10.0.0.1:9092"}, result["0"])
	assert.Equal(t, []string{"10.0.0.1:9093"}, result["1"])
}

func TestDeriveAssignment_ZeroPortFallsBackToDefault(t *testing.T) {
	// Legacy pod without a Port → falls back to 9091 (K8s pod-per-IP model).
	pods := map[string]model.PodData{
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
	}

	result := DeriveAssignment(1, pods, "v1")
	assert.Equal(t, []string{"10.0.0.1:9091"}, result["0"])
}

func TestDeriveAssignment_PodWithInvalidName(t *testing.T) {
	// "totally-unrelated" has no "-shard-" substring → ExtractShardID returns "" → triggers sid=="" branch
	pods := map[string]model.PodData{
		"totally-unrelated": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
	}

	result := DeriveAssignment(1, pods, "v1")
	assert.Empty(t, result["0"]) // invalid pod is skipped
}

func TestDeriveAssignment_ShardOutOfRange(t *testing.T) {
	// Pod claims shard 99 but shardCount is 1 (shards 0..0 only)
	pods := map[string]model.PodData{
		"fs-features-shard-99-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
	}

	result := DeriveAssignment(1, pods, "v1")
	assert.Len(t, result, 1)     // only shard "0" in result
	assert.Empty(t, result["0"]) // shard 99 skipped
}

func TestDeriveAssignment_ZeroShardCount(t *testing.T) {
	result := DeriveAssignment(0, nil, "v1")
	assert.Empty(t, result)
}

func TestDeriveAssignment_PartialCoverage(t *testing.T) {
	pods := map[string]model.PodData{
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		// shard 1 has no warm pods
	}

	result := DeriveAssignment(2, pods, "v1")
	assert.NotEmpty(t, result["0"])
	assert.Empty(t, result["1"])
}
