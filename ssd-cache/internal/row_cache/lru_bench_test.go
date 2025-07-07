package rowcache

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func BenchmarkLRUCache_ComprehensivePerformance(b *testing.B) {
	// Test with different value sizes: 1B, 10B, 100B, 1KB, 10KB
	valueSizes := []int{1, 10, 100, 1024, 10240}

	for _, valueSize := range valueSizes {
		b.Run(fmt.Sprintf("ValueSize_%dB", valueSize), func(b *testing.B) {
			benchmarkLRUCacheWithValueSize(b, valueSize)
		})
	}
}

func benchmarkLRUCacheWithValueSize(b *testing.B, valueSize int) {
	const (
		capacity       = 1_000_000  // 1M capacity
		evictionThresh = 0.7        // 70% eviction threshold
		totalOps       = 11_000_000 // Total operations per iteration
		setKeyStart    = 1_000_000  // Set operations use keys starting from 1M
		setKeyEnd      = 10_000_000 // Set operations use keys up to 10M
		getKeyStart    = 0          // Get operations use keys starting from 0
		getKeyEnd      = 500_000    // Get operations use keys up to 1M
		setProbability = 0.4        // 40% probability for Set operations
	)

	// Global variables outside the loop for key range tracking
	var setKeyIndex int64 = setKeyStart
	var getKeyIndex int64 = getKeyStart

	// Create cache configuration
	config := LRUCacheConfig{
		Capacity:          capacity,
		EvictionThreshold: evictionThresh,
		SlabSizes:         []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384},
	}

	// Create test value of specified size
	testValue := make([]byte, valueSize)
	for i := range testValue {
		testValue[i] = byte(i % 256)
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		b.StopTimer()

		// Create fresh cache for each iteration
		cache := NewLRUCache(config)

		// Reset key indices for each iteration
		setKeyIndex = setKeyStart
		getKeyIndex = getKeyStart

		var memStatsBefore, memStatsAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStatsBefore)

		// Counters for tracking operation types
		var setOpsCount, getOpsCount, getHitCount, getMissCount int64

		b.StartTimer()
		startTime := time.Now()

		// Perform mixed operations
		for i := 0; i < totalOps; i++ {
			if rand.Float64() < setProbability {
				// 40% probability: Set operation using key from [1M..10M] range
				if setKeyIndex >= setKeyEnd {
					setKeyIndex = setKeyStart // Wrap around if we exceed range
				}
				key := fmt.Sprintf("key_%d", setKeyIndex)
				cache.Set(key, testValue)
				setKeyIndex++
				setOpsCount++
			} else {
				// 60% probability: Get operation (and if not present, Set) using key from [0..1M] range
				if getKeyIndex >= getKeyEnd {
					getKeyIndex = getKeyStart // Wrap around if we exceed range
				}
				key := fmt.Sprintf("key_%d", getKeyIndex)
				value := cache.Get(key)
				if value == nil {
					// Not present, so Set it
					cache.Set(key, testValue)
					getMissCount++
				} else {
					getHitCount++
				}
				getKeyIndex++
				getOpsCount++
			}
		}

		totalDuration := time.Since(startTime)

		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&memStatsAfter)

		// Calculate metrics
		totalBytes := (setOpsCount+getMissCount)*int64(valueSize) + getHitCount*int64(valueSize)

		// Calculate throughput
		overallThroughputMBps := float64(totalBytes) / totalDuration.Seconds() / (1024 * 1024)

		// Calculate hit rate from both cache stats and our tracking
		cacheHitRate := float64(cache.stats.HitCount) / float64(cache.stats.HitCount+cache.stats.MissCount) * 100
		workloadHitRate := float64(getHitCount) / float64(getOpsCount) * 100

		// Calculate allocations
		allocsPerOp := float64(memStatsAfter.Mallocs-memStatsBefore.Mallocs) / float64(totalOps)
		bytesPerOp := float64(memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc) / float64(totalOps)

		// Report metrics
		b.ReportMetric(overallThroughputMBps, "MB/s")
		b.ReportMetric(float64(totalDuration.Nanoseconds())/float64(totalOps), "ns/op")
		b.ReportMetric(allocsPerOp, "allocs/op")
		b.ReportMetric(bytesPerOp, "B/op")
		b.ReportMetric(workloadHitRate, "workload_hit_rate_%")
		b.ReportMetric(cacheHitRate, "cache_hit_rate_%")

		// Log detailed stats
		if n == 0 { // Only log on first iteration to avoid spam
			b.Logf("\n=== Mixed Workload Performance Report for Value Size %d bytes ===", valueSize)
			b.Logf("Total Operations: %d", totalOps)
			b.Logf("Set Operations: %d (%.1f%%)", setOpsCount, float64(setOpsCount)/float64(totalOps)*100)
			b.Logf("Get Operations: %d (%.1f%%)", getOpsCount, float64(getOpsCount)/float64(totalOps)*100)
			b.Logf("Cache Capacity: %d", capacity)
			b.Logf("Eviction Threshold: %.0f%%", evictionThresh*100)
			b.Logf("\n--- Key Range Usage ---")
			b.Logf("Set Key Range: [%d..%d] (Final index: %d)", setKeyStart, setKeyEnd-1, setKeyIndex-1)
			b.Logf("Get Key Range: [%d..%d] (Final index: %d)", getKeyStart, getKeyEnd-1, getKeyIndex-1)
			b.Logf("\n--- Throughput Metrics ---")
			b.Logf("Overall Throughput: %.2f MB/s", overallThroughputMBps)
			b.Logf("Average Latency: %.2f ns/op", float64(totalDuration.Nanoseconds())/float64(totalOps))
			b.Logf("\n--- Cache Statistics ---")
			b.Logf("Workload Hit Rate: %.2f%% (%d hits / %d gets)", workloadHitRate, getHitCount, getOpsCount)
			b.Logf("Cache Hit Rate: %.2f%% (%d hits / %d total)", cacheHitRate, cache.stats.HitCount, cache.stats.HitCount+cache.stats.MissCount)
			b.Logf("Get Hits: %d", getHitCount)
			b.Logf("Get Misses (followed by Set): %d", getMissCount)
			b.Logf("Direct Set Operations: %d", setOpsCount)
			b.Logf("Total Cache Set Operations: %d", cache.stats.SetCount)
			b.Logf("Eviction Events: %d", cache.stats.EvictCount)
			b.Logf("Total Items Evicted: %d", cache.stats.EvictItemCount)
			b.Logf("Final Cache Size: %d", cache.size)
			b.Logf("\n--- Memory Metrics ---")
			b.Logf("Allocations per Operation: %.2f", allocsPerOp)
			b.Logf("Bytes per Operation: %.2f", bytesPerOp)
			b.Logf("Total Memory Allocated: %d bytes", memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc)
			if memStatsAfter.TotalAlloc > memStatsBefore.TotalAlloc {
				b.Logf("Memory Efficiency: %.2f%% (data/total)",
					float64(totalBytes)/float64(memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc)*100)
			}
			b.Logf("\n--- Operation Mix ---")
			b.Logf("Actual Set Probability: %.2f%% (target: %.0f%%)", float64(setOpsCount)/float64(totalOps)*100, setProbability*100)
			b.Logf("Actual Get Probability: %.2f%% (target: %.0f%%)", float64(getOpsCount)/float64(totalOps)*100, (1-setProbability)*100)
		}
	}
}
