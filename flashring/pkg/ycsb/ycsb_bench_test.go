package ycsb

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

// YCSB Workload configurations based on standard YCSB workloads
type WorkloadConfig struct {
	Name                string
	ReadProportion      float64
	UpdateProportion    float64
	InsertProportion    float64
	ScanProportion      float64
	ReadModifyWriteProp float64
	RequestDistribution string // "uniform", "zipfian", "latest"
	Description         string
}

// Standard YCSB Workloads
var (
	WorkloadA = WorkloadConfig{
		Name:                "WorkloadA",
		ReadProportion:      0.5,
		UpdateProportion:    0.5,
		InsertProportion:    0.0,
		ScanProportion:      0.0,
		ReadModifyWriteProp: 0.0,
		RequestDistribution: "zipfian",
		Description:         "Read/Update heavy (50%/50%) - Update heavy workload",
	}

	WorkloadB = WorkloadConfig{
		Name:                "WorkloadB",
		ReadProportion:      0.95,
		UpdateProportion:    0.05,
		InsertProportion:    0.0,
		ScanProportion:      0.0,
		ReadModifyWriteProp: 0.0,
		RequestDistribution: "zipfian",
		Description:         "Read heavy (95%/5%) - Read mostly workload",
	}

	WorkloadC = WorkloadConfig{
		Name:                "WorkloadC",
		ReadProportion:      1.0,
		UpdateProportion:    0.0,
		InsertProportion:    0.0,
		ScanProportion:      0.0,
		ReadModifyWriteProp: 0.0,
		RequestDistribution: "zipfian",
		Description:         "Read only (100%) - Read only workload",
	}

	WorkloadD = WorkloadConfig{
		Name:                "WorkloadD",
		ReadProportion:      0.95,
		UpdateProportion:    0.0,
		InsertProportion:    0.05,
		ScanProportion:      0.0,
		ReadModifyWriteProp: 0.0,
		RequestDistribution: "latest",
		Description:         "Read latest (95%/5%) - Read latest workload",
	}

	WorkloadF = WorkloadConfig{
		Name:                "WorkloadF",
		ReadProportion:      0.5,
		UpdateProportion:    0.0,
		InsertProportion:    0.0,
		ScanProportion:      0.0,
		ReadModifyWriteProp: 0.5,
		RequestDistribution: "zipfian",
		Description:         "Read-modify-write (50%/50%) - Transaction workload",
	}
)

// BenchmarkYCSB_AllWorkloads runs all standard YCSB workloads
func BenchmarkYCSB_AllWorkloads(b *testing.B) {
	workloads := []WorkloadConfig{WorkloadA, WorkloadB, WorkloadC, WorkloadD, WorkloadF}

	for _, workload := range workloads {
		b.Run(workload.Name, func(b *testing.B) {
			benchmarkYCSBWorkload(b, workload)
		})
	}
}

// BenchmarkYCSB_WorkloadA tests read/update heavy workload
func BenchmarkYCSB_WorkloadA(b *testing.B) {
	benchmarkYCSBWorkload(b, WorkloadA)
}

// BenchmarkYCSB_WorkloadB tests read heavy workload
func BenchmarkYCSB_WorkloadB(b *testing.B) {
	benchmarkYCSBWorkload(b, WorkloadB)
}

// BenchmarkYCSB_WorkloadC tests read only workload
func BenchmarkYCSB_WorkloadC(b *testing.B) {
	benchmarkYCSBWorkload(b, WorkloadC)
}

func benchmarkYCSBWorkload(b *testing.B, workload WorkloadConfig) {
	const (
		recordCount    = 1000000 // 1M records for load phase
		operationCount = 500000  // 500K operations for run phase
		fieldLength    = 100     // 100 bytes per field
		fieldCount     = 10      // 10 fields per record
	)

	// Create YCSB configuration
	config := YCSBConfig{
		Capacity:          500000, // 500K capacity (half of record count)
		EvictionThreshold: 0.7,    // 70% eviction threshold
		SlabSizes:         []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384},
	}

	// Create test data
	testValue := make([]byte, fieldLength*fieldCount)
	for i := range testValue {
		testValue[i] = byte(i % 256)
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		b.StopTimer()

		// Create fresh database for each iteration
		db, err := NewLRUCacheDB(config)
		if err != nil {
			b.Fatalf("Failed to create LRU cache DB: %v", err)
		}

		var memStatsBefore, memStatsAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStatsBefore)

		// Counters for operation tracking
		var readOps, updateOps, insertOps, rmwOps int64
		var readHits, readMisses int64

		b.StartTimer()
		startTime := time.Now()

		// Load phase: Insert initial records
		ctx := context.Background()
		for i := 0; i < recordCount; i++ {
			key := fmt.Sprintf("user%010d", i)
			values := map[string][]byte{
				"field0": testValue,
			}
			err := db.Insert(ctx, "usertable", key, values)
			if err != nil {
				b.Fatalf("Insert failed: %v", err)
			}
		}

		loadDuration := time.Since(startTime)

		// Run phase: Execute workload operations
		runStartTime := time.Now()
		for i := 0; i < operationCount; i++ {
			key := generateKey(i, recordCount, workload.RequestDistribution)
			operation := selectOperation(workload)

			switch operation {
			case "read":
				_, err := db.Read(ctx, "usertable", key, []string{"field0"})
				if err != nil {
					readMisses++
				} else {
					readHits++
				}
				readOps++

			case "update":
				values := map[string][]byte{
					"field0": testValue,
				}
				err := db.Update(ctx, "usertable", key, values)
				if err != nil {
					b.Errorf("Update failed: %v", err)
				}
				updateOps++

			case "insert":
				// For insert operations, use a new key
				newKey := fmt.Sprintf("user%010d", recordCount+i)
				values := map[string][]byte{
					"field0": testValue,
				}
				err := db.Insert(ctx, "usertable", newKey, values)
				if err != nil {
					b.Errorf("Insert failed: %v", err)
				}
				insertOps++

			case "readmodifywrite":
				// Read-modify-write operation
				_, err := db.Read(ctx, "usertable", key, []string{"field0"})
				if err != nil {
					readMisses++
				} else {
					readHits++
					// Modify and write back
					values := map[string][]byte{
						"field0": testValue,
					}
					err = db.Update(ctx, "usertable", key, values)
					if err != nil {
						b.Errorf("Read-modify-write update failed: %v", err)
					}
				}
				rmwOps++
			}
		}

		runDuration := time.Since(runStartTime)
		totalDuration := time.Since(startTime)

		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&memStatsAfter)

		// Get cache statistics
		stats := db.GetStats()

		// Calculate metrics
		totalOps := readOps + updateOps + insertOps + rmwOps
		throughput := float64(totalOps) / runDuration.Seconds()
		loadThroughput := float64(recordCount) / loadDuration.Seconds()

		// Calculate hit rates
		cacheHitRate := float64(stats.HitCount) / float64(stats.HitCount+stats.MissCount) * 100
		workloadHitRate := float64(readHits) / float64(readHits+readMisses) * 100

		// Calculate memory metrics
		allocsPerOp := float64(memStatsAfter.Mallocs-memStatsBefore.Mallocs) / float64(totalOps+recordCount)
		bytesPerOp := float64(memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc) / float64(totalOps+recordCount)

		// Report benchmark metrics
		b.ReportMetric(throughput, "ops/sec")
		b.ReportMetric(float64(runDuration.Nanoseconds())/float64(totalOps), "ns/op")
		b.ReportMetric(workloadHitRate, "hit_rate_%")
		b.ReportMetric(allocsPerOp, "allocs/op")
		b.ReportMetric(bytesPerOp, "B/op")

		// Log detailed stats on first iteration
		if n == 0 {
			b.Logf("\n=== YCSB %s Benchmark Results ===", workload.Name)
			b.Logf("Description: %s", workload.Description)
			b.Logf("\n--- Workload Configuration ---")
			b.Logf("Read Proportion: %.1f%%", workload.ReadProportion*100)
			b.Logf("Update Proportion: %.1f%%", workload.UpdateProportion*100)
			b.Logf("Insert Proportion: %.1f%%", workload.InsertProportion*100)
			b.Logf("Read-Modify-Write Proportion: %.1f%%", workload.ReadModifyWriteProp*100)
			b.Logf("Request Distribution: %s", workload.RequestDistribution)

			b.Logf("\n--- Performance Metrics ---")
			b.Logf("Load Throughput: %.2f ops/sec", loadThroughput)
			b.Logf("Run Throughput: %.2f ops/sec", throughput)
			b.Logf("Average Latency: %.2f ns/op", float64(runDuration.Nanoseconds())/float64(totalOps))

			b.Logf("\n--- Operation Breakdown ---")
			b.Logf("Read Operations: %d (%.1f%%)", readOps, float64(readOps)/float64(totalOps)*100)
			b.Logf("Update Operations: %d (%.1f%%)", updateOps, float64(updateOps)/float64(totalOps)*100)
			b.Logf("Insert Operations: %d (%.1f%%)", insertOps, float64(insertOps)/float64(totalOps)*100)
			b.Logf("Read-Modify-Write Operations: %d (%.1f%%)", rmwOps, float64(rmwOps)/float64(totalOps)*100)

			b.Logf("\n--- Cache Statistics ---")
			b.Logf("Cache Hit Rate: %.2f%% (%d/%d)", cacheHitRate, stats.HitCount, stats.HitCount+stats.MissCount)
			b.Logf("Workload Hit Rate: %.2f%% (%d/%d)", workloadHitRate, readHits, readHits+readMisses)
			b.Logf("Final Cache Size: %d", stats.Size)
			b.Logf("Cache Capacity: %d", stats.Capacity)
			b.Logf("Eviction Events: %d", stats.EvictCount)
			b.Logf("Total Items Evicted: %d", stats.EvictItemCount)

			b.Logf("\n--- Timing Breakdown ---")
			b.Logf("Load Phase Duration: %v", loadDuration)
			b.Logf("Run Phase Duration: %v", runDuration)
			b.Logf("Total Duration: %v", totalDuration)

			b.Logf("\n--- Memory Metrics ---")
			b.Logf("Allocations per Operation: %.2f", allocsPerOp)
			b.Logf("Bytes per Operation: %.2f", bytesPerOp)
		}
	}
}

// selectOperation selects an operation based on workload proportions
func selectOperation(workload WorkloadConfig) string {
	r := rand.Float64()

	if r < workload.ReadProportion {
		return "read"
	}
	r -= workload.ReadProportion

	if r < workload.UpdateProportion {
		return "update"
	}
	r -= workload.UpdateProportion

	if r < workload.InsertProportion {
		return "insert"
	}
	r -= workload.InsertProportion

	if r < workload.ReadModifyWriteProp {
		return "readmodifywrite"
	}

	// Default to read if something goes wrong
	return "read"
}

// generateKey generates a key based on the request distribution
func generateKey(operationIndex, recordCount int, distribution string) string {
	var keyIndex int

	switch distribution {
	case "uniform":
		keyIndex = rand.Intn(recordCount)
	case "zipfian":
		// Simplified Zipfian: 80% of requests go to 20% of keys
		if rand.Float64() < 0.8 {
			keyIndex = rand.Intn(recordCount / 5) // Top 20% of keys
		} else {
			keyIndex = recordCount/5 + rand.Intn(recordCount*4/5) // Bottom 80% of keys
		}
	case "latest":
		// Latest distribution: favor recently inserted keys
		if rand.Float64() < 0.8 {
			// 80% chance to access the most recent 10% of keys
			keyIndex = recordCount*9/10 + rand.Intn(recordCount/10)
		} else {
			keyIndex = rand.Intn(recordCount * 9 / 10)
		}
	default:
		keyIndex = rand.Intn(recordCount)
	}

	return fmt.Sprintf("user%010d", keyIndex)
}
