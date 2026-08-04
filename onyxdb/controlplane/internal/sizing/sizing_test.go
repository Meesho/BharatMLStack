package sizing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompute(t *testing.T) {
	tests := []struct {
		name              string
		input             Input
		wantShards        int
		wantReplicas      int
		wantTotalPods     int
		wantNVMeApprox    float64
		wantMemoryApprox  float64
	}{
		{
			name:           "small_dataset_under_1TB",
			input:          Input{DatasetSizeGB: 100, TargetRPS: 10000},
			wantShards:     1,
			wantReplicas:   2, // min
			wantTotalPods:  2,
			wantNVMeApprox: 250.0, // 100 * 2.5
			wantMemoryApprox: 102.0, // 100 + 2
		},
		{
			name:           "exactly_1TB",
			input:          Input{DatasetSizeGB: 1000, TargetRPS: 1},
			wantShards:     2,
			wantReplicas:   2,
			wantTotalPods:  4,
			wantNVMeApprox: 1250.0, // 500 * 2.5
			wantMemoryApprox: 502.0,
		},
		{
			name:           "large_10TB",
			input:          Input{DatasetSizeGB: 10000, TargetRPS: 500000},
			wantShards:     11,
			wantReplicas:   2, // ceil(500k/60k)=9 pods / 11 shards → 1, bumped to min 2
			wantTotalPods:  22,
			wantNVMeApprox: 10000.0 / 11.0 * 2.5, // ≈ 2272.73
			wantMemoryApprox: 10000.0/11.0 + 2.0,  // ≈ 911.09
		},
		{
			name:           "high_rps_forces_more_replicas",
			input:          Input{DatasetSizeGB: 100, TargetRPS: 1020000},
			wantShards:     1,
			wantReplicas:   17, // ceil(1020000/60000)=17 > min 2
			wantTotalPods:  17,
			wantNVMeApprox: 250.0,
			wantMemoryApprox: 102.0,
		},
		{
			name:           "zero_rps_uses_minimum_replicas",
			input:          Input{DatasetSizeGB: 50, TargetRPS: 0},
			wantShards:     1,
			wantReplicas:   2,
			wantTotalPods:  2,
			wantNVMeApprox: 125.0,
			wantMemoryApprox: 52.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.input)
			assert.Equal(t, tc.wantShards, got.ShardCount, "ShardCount")
			assert.Equal(t, tc.wantReplicas, got.ReplicaFactor, "ReplicaFactor")
			assert.Equal(t, cpuPerPod, got.CPUPerPod, "CPUPerPod")
			assert.Equal(t, tc.wantTotalPods, got.TotalPods, "TotalPods")
			assert.InDelta(t, tc.wantNVMeApprox, got.NVMePerPodGB, 0.01, "NVMePerPodGB")
			assert.InDelta(t, tc.wantMemoryApprox, got.MemoryPerPodGB, 0.01, "MemoryPerPodGB")
		})
	}
}

func TestCompute_ReplicaFactorNeverBelowTwo(t *testing.T) {
	out := Compute(Input{DatasetSizeGB: 500, TargetRPS: 1})
	assert.GreaterOrEqual(t, out.ReplicaFactor, minReplicaFactor)
}

func TestCompute_TotalPodsEqualsShardCountTimesReplica(t *testing.T) {
	out := Compute(Input{DatasetSizeGB: 300, TargetRPS: 120000})
	assert.Equal(t, out.ShardCount*out.ReplicaFactor, out.TotalPods)
}
