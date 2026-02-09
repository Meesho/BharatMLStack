package internal

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Configuration constants for benchmarks
const (
	defaultKeyCount   = 100000
	defaultValueSize  = 1024
	defaultTTLMinutes = 60
)

// setupBenchmarkCache creates a WrapCache instance for benchmarking
func setupBenchmarkCache(b *testing.B, numShards int, keysPerShard int) (*WrapCache, string) {
	b.Helper()

	tempDir, err := os.MkdirTemp("", "flashring_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}

	config := WrapCacheConfig{
		NumShards:             numShards,
		KeysPerShard:          keysPerShard,
		FileSize:              1 << 30,          // 1GB per shard
		MemtableSize:          64 * 1024 * 1024, // 64MB memtable
		ReWriteScoreThreshold: 0.5,
		GridSearchEpsilon:     0.01,
		SampleDuration:        time.Second,
		MountPoint:            tempDir,
		MaxConcurrentReads:    100,
	}

	cache, err := NewWrapCache(config, tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create WrapCache: %v", err)
	}

	return cache, tempDir
}

// cleanupBenchmarkCache removes the temporary directory and cleans up resources
func cleanupBenchmarkCache(tempDir string) {
	os.RemoveAll(tempDir)
}

// generateTestData pre-generates keys and values to avoid benchmark contamination
func generateTestData(keyCount int, valueSize int) ([]string, [][]byte) {
	keys := make([]string, keyCount)
	values := make([][]byte, keyCount)

	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("key_%016d", i)
		values[i] = make([]byte, valueSize)
		// Fill with deterministic but varied data
		for j := range values[i] {
			values[i][j] = byte((i + j) % 256)
		}
	}

	return keys, values
}

// generateZipfianIndex generates an index following Zipfian distribution
// (80% of requests go to 20% of keys)
func generateZipfianIndex(maxIndex int) int {
	if rand.Float64() < 0.8 {
		// 80% of requests go to top 20% of keys
		return rand.Intn(maxIndex / 5)
	}
	// 20% of requests go to remaining 80% of keys
	return maxIndex/5 + rand.Intn(maxIndex*4/5)
}

// LatencyRecorder tracks latency samples for percentile calculation
type LatencyRecorder struct {
	samples []time.Duration
	mu      sync.Mutex
}

func NewLatencyRecorder(capacity int) *LatencyRecorder {
	return &LatencyRecorder{
		samples: make([]time.Duration, 0, capacity),
	}
}

func (lr *LatencyRecorder) Record(d time.Duration) {
	lr.mu.Lock()
	lr.samples = append(lr.samples, d)
	lr.mu.Unlock()
}

func (lr *LatencyRecorder) Percentile(p float64) time.Duration {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if len(lr.samples) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(lr.samples))
	copy(sorted, lr.samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

// =============================================================================
// Basic Operation Benchmarks
// =============================================================================

// BenchmarkWrapCache_Put measures single Put latency and throughput
func BenchmarkWrapCache_Put(b *testing.B) {
	cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
	defer cleanupBenchmarkCache(tempDir)

	keys, values := generateTestData(b.N, defaultValueSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % len(keys)
		err := cache.Put(keys[idx], values[idx], defaultTTLMinutes)
		if err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
	b.ReportMetric(float64(b.N*defaultValueSize)/b.Elapsed().Seconds()/1024/1024, "MB/s")
}

// BenchmarkWrapCache_Get_Hit measures Get latency for cache hits
func BenchmarkWrapCache_Get_Hit(b *testing.B) {
	cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
	defer cleanupBenchmarkCache(tempDir)

	// Pre-populate cache
	keyCount := 10000
	keys, values := generateTestData(keyCount, defaultValueSize)

	for i := 0; i < keyCount; i++ {
		err := cache.Put(keys[i], values[i], defaultTTLMinutes)
		if err != nil {
			b.Fatalf("Pre-population Put failed: %v", err)
		}
	}

	var hits, misses int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % keyCount
		_, found, _ := cache.Get(keys[idx])
		if found {
			hits++
		} else {
			misses++
		}
	}

	b.StopTimer()
	hitRate := float64(hits) / float64(hits+misses) * 100
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
	b.ReportMetric(hitRate, "hit_rate_%")
}

// BenchmarkWrapCache_Get_Miss measures Get latency for cache misses
func BenchmarkWrapCache_Get_Miss(b *testing.B) {
	cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
	defer cleanupBenchmarkCache(tempDir)

	// Generate keys that are NOT in the cache
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("missing_key_%016d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get(keys[i])
	}

	b.StopTimer()
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// BenchmarkWrapCache_PutGet measures combined Put+Get operations
func BenchmarkWrapCache_PutGet(b *testing.B) {
	cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
	defer cleanupBenchmarkCache(tempDir)

	keys, values := generateTestData(defaultKeyCount, defaultValueSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % defaultKeyCount

		// Put
		err := cache.Put(keys[idx], values[idx], defaultTTLMinutes)
		if err != nil {
			b.Fatalf("Put failed: %v", err)
		}

		// Get
		cache.Get(keys[idx])
	}

	b.StopTimer()
	// Each iteration is 2 operations (put + get)
	b.ReportMetric(float64(b.N*2)/b.Elapsed().Seconds(), "ops/sec")
}

// =============================================================================
// Concurrency Benchmarks
// =============================================================================

// BenchmarkWrapCache_ParallelPut measures parallel Put performance
func BenchmarkWrapCache_ParallelPut(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8, 16, 32}

	for _, numGoroutines := range goroutineCounts {
		b.Run(fmt.Sprintf("goroutines_%d", numGoroutines), func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
			defer cleanupBenchmarkCache(tempDir)

			keys, values := generateTestData(defaultKeyCount, defaultValueSize)

			b.SetParallelism(numGoroutines / runtime.GOMAXPROCS(0))
			if b.N < numGoroutines {
				b.N = numGoroutines
			}

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				localIdx := 0
				for pb.Next() {
					idx := localIdx % defaultKeyCount
					cache.Put(keys[idx], values[idx], defaultTTLMinutes)
					localIdx++
				}
			})

			b.StopTimer()
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
		})
	}
}

// BenchmarkWrapCache_ParallelGet measures parallel Get performance
func BenchmarkWrapCache_ParallelGet(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8, 16, 32}

	for _, numGoroutines := range goroutineCounts {
		b.Run(fmt.Sprintf("goroutines_%d", numGoroutines), func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
			defer cleanupBenchmarkCache(tempDir)

			// Pre-populate cache
			keyCount := 10000
			keys, values := generateTestData(keyCount, defaultValueSize)
			for i := 0; i < keyCount; i++ {
				cache.Put(keys[i], values[i], defaultTTLMinutes)
			}

			var hits atomic.Int64
			var misses atomic.Int64

			b.SetParallelism(numGoroutines / runtime.GOMAXPROCS(0))

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				localIdx := 0
				for pb.Next() {
					idx := localIdx % keyCount
					_, found, _ := cache.Get(keys[idx])
					if found {
						hits.Add(1)
					} else {
						misses.Add(1)
					}
					localIdx++
				}
			})

			b.StopTimer()
			totalHits := hits.Load()
			totalMisses := misses.Load()
			hitRate := float64(totalHits) / float64(totalHits+totalMisses) * 100
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			b.ReportMetric(hitRate, "hit_rate_%")
		})
	}
}

// BenchmarkWrapCache_ParallelMixed measures mixed read/write workload (80% reads, 20% writes)
func BenchmarkWrapCache_ParallelMixed(b *testing.B) {
	goroutineCounts := []int{1, 4, 8, 16, 32}

	for _, numGoroutines := range goroutineCounts {
		b.Run(fmt.Sprintf("goroutines_%d", numGoroutines), func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
			defer cleanupBenchmarkCache(tempDir)

			keyCount := 10000
			keys, values := generateTestData(keyCount, defaultValueSize)

			// Pre-populate with half the keys
			for i := 0; i < keyCount/2; i++ {
				cache.Put(keys[i], values[i], defaultTTLMinutes)
			}

			var reads, writes atomic.Int64
			var hits, misses atomic.Int64

			b.SetParallelism(numGoroutines / runtime.GOMAXPROCS(0))

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
				localIdx := 0

				for pb.Next() {
					idx := localIdx % keyCount

					if localRand.Float64() < 0.8 {
						// 80% reads
						_, found, _ := cache.Get(keys[idx])
						reads.Add(1)
						if found {
							hits.Add(1)
						} else {
							misses.Add(1)
						}
					} else {
						// 20% writes
						cache.Put(keys[idx], values[idx], defaultTTLMinutes)
						writes.Add(1)
					}
					localIdx++
				}
			})

			b.StopTimer()
			totalHits := hits.Load()
			totalMisses := misses.Load()
			hitRate := float64(totalHits) / float64(totalHits+totalMisses) * 100
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			b.ReportMetric(hitRate, "hit_rate_%")
			b.ReportMetric(float64(reads.Load())/float64(b.N)*100, "read_%")
			b.ReportMetric(float64(writes.Load())/float64(b.N)*100, "write_%")
		})
	}
}

// =============================================================================
// Value Size Benchmarks
// =============================================================================

// BenchmarkWrapCache_ValueSizes tests performance with different value sizes
func BenchmarkWrapCache_ValueSizes(b *testing.B) {
	valueSizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, valueSize := range valueSizes {
		b.Run(fmt.Sprintf("size_%dB", valueSize), func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
			defer cleanupBenchmarkCache(tempDir)

			keyCount := 10000
			keys, values := generateTestData(keyCount, valueSize)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				idx := i % keyCount
				err := cache.Put(keys[idx], values[idx], defaultTTLMinutes)
				if err != nil {
					b.Fatalf("Put failed: %v", err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			b.ReportMetric(float64(b.N*valueSize)/b.Elapsed().Seconds()/1024/1024, "MB/s")
		})
	}
}

// BenchmarkWrapCache_ValueSizes_Get tests Get performance with different value sizes
func BenchmarkWrapCache_ValueSizes_Get(b *testing.B) {
	valueSizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, valueSize := range valueSizes {
		b.Run(fmt.Sprintf("size_%dB", valueSize), func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
			defer cleanupBenchmarkCache(tempDir)

			keyCount := 10000
			keys, values := generateTestData(keyCount, valueSize)

			// Pre-populate
			for i := 0; i < keyCount; i++ {
				cache.Put(keys[i], values[i], defaultTTLMinutes)
			}

			var hits atomic.Int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				idx := i % keyCount
				_, found, _ := cache.Get(keys[idx])
				if found {
					hits.Add(1)
				}
			}

			b.StopTimer()
			hitRate := float64(hits.Load()) / float64(b.N) * 100
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			b.ReportMetric(float64(b.N*valueSize)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			b.ReportMetric(hitRate, "hit_rate_%")
		})
	}
}

// =============================================================================
// Throughput Benchmarks
// =============================================================================

// BenchmarkWrapCache_Throughput measures sustained throughput
func BenchmarkWrapCache_Throughput(b *testing.B) {
	cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
	defer cleanupBenchmarkCache(tempDir)

	keyCount := defaultKeyCount
	keys, values := generateTestData(keyCount, defaultValueSize)

	// Pre-populate half
	for i := 0; i < keyCount/2; i++ {
		cache.Put(keys[i], values[i], defaultTTLMinutes)
	}

	var totalOps atomic.Int64
	var totalBytes atomic.Int64

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		localIdx := 0
		localOps := int64(0)
		localBytes := int64(0)

		for pb.Next() {
			// Use Zipfian distribution for realistic workload
			idx := generateZipfianIndex(keyCount)

			if localRand.Float64() < 0.8 {
				// 80% reads
				val, found, _ := cache.Get(keys[idx])
				if found {
					localBytes += int64(len(val))
				}
			} else {
				// 20% writes
				cache.Put(keys[idx], values[idx], defaultTTLMinutes)
				localBytes += int64(len(values[idx]))
			}
			localOps++
			localIdx++
		}

		totalOps.Add(localOps)
		totalBytes.Add(localBytes)
	})

	b.StopTimer()
	elapsed := b.Elapsed().Seconds()
	b.ReportMetric(float64(totalOps.Load())/elapsed, "ops/sec")
	b.ReportMetric(float64(totalBytes.Load())/elapsed/1024/1024, "MB/s")
}

// =============================================================================
// Latency Percentile Benchmarks
// =============================================================================

// BenchmarkWrapCache_LatencyPercentiles measures latency percentiles
func BenchmarkWrapCache_LatencyPercentiles(b *testing.B) {
	cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
	defer cleanupBenchmarkCache(tempDir)

	keyCount := 10000
	keys, values := generateTestData(keyCount, defaultValueSize)

	// Pre-populate
	for i := 0; i < keyCount; i++ {
		cache.Put(keys[i], values[i], defaultTTLMinutes)
	}

	// Limit sample collection to avoid memory issues
	maxSamples := 100000
	if b.N < maxSamples {
		maxSamples = b.N
	}
	recorder := NewLatencyRecorder(maxSamples)

	b.ResetTimer()

	sampleRate := 1
	if b.N > maxSamples {
		sampleRate = b.N / maxSamples
	}

	for i := 0; i < b.N; i++ {
		idx := i % keyCount

		if i%sampleRate == 0 {
			start := time.Now()
			cache.Get(keys[idx])
			recorder.Record(time.Since(start))
		} else {
			cache.Get(keys[idx])
		}
	}

	b.StopTimer()

	p50 := recorder.Percentile(0.50)
	p99 := recorder.Percentile(0.99)
	p999 := recorder.Percentile(0.999)

	b.ReportMetric(float64(p50.Microseconds()), "p50_us")
	b.ReportMetric(float64(p99.Microseconds()), "p99_us")
	b.ReportMetric(float64(p999.Microseconds()), "p999_us")
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

// BenchmarkWrapCache_MemoryAllocs tracks memory allocations per operation
func BenchmarkWrapCache_MemoryAllocs(b *testing.B) {
	b.Run("Put", func(b *testing.B) {
		cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
		defer cleanupBenchmarkCache(tempDir)

		keys, values := generateTestData(defaultKeyCount, defaultValueSize)

		var memStatsBefore, memStatsAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStatsBefore)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			idx := i % defaultKeyCount
			cache.Put(keys[idx], values[idx], defaultTTLMinutes)
		}

		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&memStatsAfter)

		allocsPerOp := float64(memStatsAfter.Mallocs-memStatsBefore.Mallocs) / float64(b.N)
		bytesPerOp := float64(memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc) / float64(b.N)

		b.ReportMetric(allocsPerOp, "heap_allocs/op")
		b.ReportMetric(bytesPerOp, "heap_B/op")
	})

	b.Run("Get", func(b *testing.B) {
		cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
		defer cleanupBenchmarkCache(tempDir)

		keyCount := 10000
		keys, values := generateTestData(keyCount, defaultValueSize)

		// Pre-populate
		for i := 0; i < keyCount; i++ {
			cache.Put(keys[i], values[i], defaultTTLMinutes)
		}

		var memStatsBefore, memStatsAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStatsBefore)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			idx := i % keyCount
			cache.Get(keys[idx])
		}

		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&memStatsAfter)

		allocsPerOp := float64(memStatsAfter.Mallocs-memStatsBefore.Mallocs) / float64(b.N)
		bytesPerOp := float64(memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc) / float64(b.N)

		b.ReportMetric(allocsPerOp, "heap_allocs/op")
		b.ReportMetric(bytesPerOp, "heap_B/op")
	})
}

// =============================================================================
// Shard Scalability Benchmarks
// =============================================================================

// BenchmarkWrapCache_ShardCount tests performance with different shard counts
func BenchmarkWrapCache_ShardCount(b *testing.B) {
	shardCounts := []int{1, 4, 8, 16, 32, 64}

	for _, numShards := range shardCounts {
		b.Run(fmt.Sprintf("shards_%d", numShards), func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, numShards, 1<<18)
			defer cleanupBenchmarkCache(tempDir)

			keyCount := 10000
			keys, values := generateTestData(keyCount, defaultValueSize)

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
				localIdx := 0

				for pb.Next() {
					idx := localIdx % keyCount

					if localRand.Float64() < 0.8 {
						cache.Get(keys[idx])
					} else {
						cache.Put(keys[idx], values[idx], defaultTTLMinutes)
					}
					localIdx++
				}
			})

			b.StopTimer()
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
		})
	}
}

// =============================================================================
// Distribution Pattern Benchmarks
// =============================================================================

// BenchmarkWrapCache_AccessPatterns tests different access patterns
func BenchmarkWrapCache_AccessPatterns(b *testing.B) {
	patterns := []struct {
		name string
		gen  func(maxIndex int) int
	}{
		{"uniform", func(maxIndex int) int { return rand.Intn(maxIndex) }},
		{"zipfian", generateZipfianIndex},
		{"sequential", func(maxIndex int) int { return 0 }}, // Will be overridden
		{"hotspot", func(maxIndex int) int {
			if rand.Float64() < 0.9 {
				return rand.Intn(maxIndex / 100) // 90% to 1% of keys
			}
			return rand.Intn(maxIndex)
		}},
	}

	for _, pattern := range patterns {
		b.Run(pattern.name, func(b *testing.B) {
			cache, tempDir := setupBenchmarkCache(b, 16, 1<<20)
			defer cleanupBenchmarkCache(tempDir)

			keyCount := 10000
			keys, values := generateTestData(keyCount, defaultValueSize)

			// Pre-populate
			for i := 0; i < keyCount; i++ {
				cache.Put(keys[i], values[i], defaultTTLMinutes)
			}

			var hits atomic.Int64

			b.ResetTimer()
			b.ReportAllocs()

			seqIdx := 0
			for i := 0; i < b.N; i++ {
				var idx int
				if pattern.name == "sequential" {
					idx = seqIdx % keyCount
					seqIdx++
				} else {
					idx = pattern.gen(keyCount)
				}

				_, found, _ := cache.Get(keys[idx])
				if found {
					hits.Add(1)
				}
			}

			b.StopTimer()
			hitRate := float64(hits.Load()) / float64(b.N) * 100
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			b.ReportMetric(hitRate, "hit_rate_%")
		})
	}
}
