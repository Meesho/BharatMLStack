// Package sizing computes resource requirements for a OnyxDB store.
//
// Given dataset size, target RPS, and p99 target, it produces
// a sizing recommendation: shard count, replica factor, and pod spec.
package sizing

// Input is the sizing request.
type Input struct {
	DatasetSizeGB  float64 // D — total dataset size
	TargetRPS      int     // desired queries per second
	P99TargetMs    float64 // p99 latency target in milliseconds
	AvgValueSizeKB float64 // average value size
}

// Output is the computed resource plan.
type Output struct {
	ShardCount     int     // S
	ReplicaFactor  int     // R
	CPUPerPod      float64 // vCPU
	MemoryPerPodGB float64 // block cache + overhead
	NVMePerPodGB   float64 // local SSD per pod
	TotalPods      int     // S × R
}

const (
	maxShardSizeGB   = 1000.0 // max 1 TB per shard
	memoryOverheadGB = 2.0    // OS + process overhead
	nvmeMultiplier   = 2.5    // 2× for V=2 versioning + headroom
	cpuPerPod        = 4.0    // 4 vCPU per pod (benchmark baseline)
	rpsPerPod        = 60000  // conservative single-pod RPS (4 vCPU, 8GB cache)
	minReplicaFactor = 2      // minimum R=2 for availability
)

// Compute calculates the sizing for a store.
//
// Heuristics derived from POC benchmarks:
//   - Single pod handles ~60K RPS on 4 vCPU with 8 GB block cache
//   - RPS scales linearly with replicas
//   - Block cache >= shard size gives p99 < 1 ms (cache-resident reads)
//   - NVMe = 2.5× shard size accommodates V=2 versioning + headroom
func Compute(input Input) Output {
	shardCount := int(input.DatasetSizeGB/maxShardSizeGB) + 1

	shardSizeGB := input.DatasetSizeGB / float64(shardCount)
	memoryPerPod := shardSizeGB + memoryOverheadGB

	totalPodsForRPS := 1
	if input.TargetRPS > 0 {
		totalPodsForRPS = (input.TargetRPS + rpsPerPod - 1) / rpsPerPod
	}

	replicaFactor := (totalPodsForRPS + shardCount - 1) / shardCount
	if replicaFactor < minReplicaFactor {
		replicaFactor = minReplicaFactor
	}

	return Output{
		ShardCount:     shardCount,
		ReplicaFactor:  replicaFactor,
		CPUPerPod:      cpuPerPod,
		MemoryPerPodGB: memoryPerPod,
		NVMePerPodGB:   shardSizeGB * nvmeMultiplier,
		TotalPods:      shardCount * replicaFactor,
	}
}
