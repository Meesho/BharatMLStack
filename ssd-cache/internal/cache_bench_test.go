package internal

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

const (
	KEY_SIZE    = 32                    // 32 byte keys
	VALUE_SIZE  = 16 * 1024             // 16KB values
	RECORD_SIZE = KEY_SIZE + VALUE_SIZE // Total record size
)

// Test capacities from 64KB to 1GB
var loadTestCapacities = []struct {
	name     string
	capacity int64
}{
	{"64KB", 64 * 1024},
	// {"128KB", 128 * 1024},
	// {"256KB", 256 * 1024},
	// {"512KB", 512 * 1024},
	// {"1MB", 1024 * 1024},
	// {"2MB", 2 * 1024 * 1024},
	// {"4MB", 4 * 1024 * 1024},
	// {"8MB", 8 * 1024 * 1024},
	// {"16MB", 16 * 1024 * 1024},
	// {"32MB", 32 * 1024 * 1024},
	// {"64MB", 64 * 1024 * 1024},
	// {"128MB", 128 * 1024 * 1024},
	// {"256MB", 256 * 1024 * 1024},
	// {"512MB", 512 * 1024 * 1024},
	// {"1GB", 1024 * 1024 * 1024},
}

// Generate a key of exactly KEY_SIZE bytes
func generateKey(index int) string {
	key := fmt.Sprintf("key_%010d", index)
	// Pad to exactly 32 bytes
	for len(key) < KEY_SIZE {
		key += "_"
	}
	if len(key) > KEY_SIZE {
		key = key[:KEY_SIZE]
	}
	return key
}

// Generate test value of exactly VALUE_SIZE bytes
func generateValue(seed int) []byte {
	value := make([]byte, VALUE_SIZE)
	for i := range value {
		value[i] = byte((seed + i) % 256)
	}
	return value
}

// Benchmark PUT operations with load testing
func BenchmarkCache_PUT_LoadTest(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	for _, tc := range loadTestCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			// Pin to OS thread for consistent performance
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)

			for i := 0; i < b.N; i++ {
				key := generateKey(i)
				value := generateValue(i)

				cache.Put(key, value)
				totalBytes += RECORD_SIZE
			}

			elapsed := time.Since(start)

			// Calculate performance metrics
			recordsPerSecond := float64(b.N) / elapsed.Seconds()
			mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
			avgLatencyNs := elapsed.Nanoseconds() / int64(b.N)

			b.ReportMetric(recordsPerSecond, "records/sec")
			b.ReportMetric(mbPerSecond, "MB/sec")
			b.ReportMetric(float64(avgLatencyNs), "ns/op")
			b.ReportMetric(float64(totalBytes)/(1024*1024), "total_MB")
		})
	}
}

// Benchmark PUT operations with sustained load (longer duration)
func BenchmarkCache_PUT_SustainedLoad(b *testing.B) {
	for _, tc := range loadTestCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// Run for a fixed duration to simulate sustained load
			duration := 10 * time.Second

			b.ResetTimer()

			start := time.Now()
			count := 0
			totalBytes := int64(0)

			for time.Since(start) < duration {
				key := generateKey(count)
				value := generateValue(count)

				cache.Put(key, value)
				totalBytes += RECORD_SIZE
				count++
			}

			elapsed := time.Since(start)

			// Calculate sustained performance metrics
			recordsPerSecond := float64(count) / elapsed.Seconds()
			mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
			avgLatencyNs := elapsed.Nanoseconds() / int64(count)

			b.ReportMetric(recordsPerSecond, "records/sec")
			b.ReportMetric(mbPerSecond, "MB/sec")
			b.ReportMetric(float64(avgLatencyNs), "ns/op")
			b.ReportMetric(float64(count), "total_records")
			b.ReportMetric(float64(totalBytes)/(1024*1024), "total_MB")
		})
	}
}

// Benchmark PUT operations with memory pressure simulation
func BenchmarkCache_PUT_MemoryPressure(b *testing.B) {
	for _, tc := range loadTestCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// Calculate how many records fit in one memtable
			recordsPerMemtable := tc.capacity / RECORD_SIZE
			if recordsPerMemtable == 0 {
				recordsPerMemtable = 1
			}

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)
			flushCount := 0

			for i := 0; i < b.N; i++ {
				key := generateKey(i)
				value := generateValue(i)

				// Track when we expect flushes to occur
				if i > 0 && int64(i)%recordsPerMemtable == 0 {
					flushCount++
				}

				cache.Put(key, value)
				totalBytes += RECORD_SIZE
			}

			elapsed := time.Since(start)

			// Performance metrics under memory pressure
			recordsPerSecond := float64(b.N) / elapsed.Seconds()
			mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
			avgLatencyNs := elapsed.Nanoseconds() / int64(b.N)

			b.ReportMetric(recordsPerSecond, "records/sec")
			b.ReportMetric(mbPerSecond, "MB/sec")
			b.ReportMetric(float64(avgLatencyNs), "ns/op")
			b.ReportMetric(float64(flushCount), "expected_flushes")
			b.ReportMetric(float64(recordsPerMemtable), "records/memtable")
		})
	}
}

// Benchmark PUT operations with mixed workload patterns
func BenchmarkCache_PUT_MixedWorkload(b *testing.B) {
	for _, tc := range loadTestCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)
			putCount := 0
			getCount := 0

			for i := 0; i < b.N; i++ {
				if i%10 < 8 { // 80% PUT operations
					key := generateKey(i)
					value := generateValue(i)
					cache.Put(key, value)
					totalBytes += RECORD_SIZE
					putCount++
				} else { // 20% GET operations
					key := generateKey(i - (i % 10)) // Get existing keys
					_ = cache.Get(key)
					getCount++
				}
			}

			elapsed := time.Since(start)

			// Mixed workload metrics
			opsPerSecond := float64(b.N) / elapsed.Seconds()
			mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
			avgLatencyNs := elapsed.Nanoseconds() / int64(b.N)

			b.ReportMetric(opsPerSecond, "ops/sec")
			b.ReportMetric(mbPerSecond, "MB/sec")
			b.ReportMetric(float64(avgLatencyNs), "ns/op")
			b.ReportMetric(float64(putCount), "put_count")
			b.ReportMetric(float64(getCount), "get_count")
		})
	}
}

// Benchmark PUT latency distribution
func BenchmarkCache_PUT_LatencyDistribution(b *testing.B) {
	for _, tc := range loadTestCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// Collect latency samples
			latencies := make([]time.Duration, 0, b.N)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := generateKey(i)
				value := generateValue(i)

				start := time.Now()
				cache.Put(key, value)
				latency := time.Since(start)

				latencies = append(latencies, latency)
			}

			// Calculate latency statistics
			if len(latencies) > 0 {
				// Sort latencies for percentile calculation
				// Simple insertion sort for small datasets
				for i := 1; i < len(latencies); i++ {
					key := latencies[i]
					j := i - 1
					for j >= 0 && latencies[j] > key {
						latencies[j+1] = latencies[j]
						j--
					}
					latencies[j+1] = key
				}

				// Calculate percentiles
				p50 := latencies[len(latencies)*50/100]
				p95 := latencies[len(latencies)*95/100]
				p99 := latencies[len(latencies)*99/100]
				max := latencies[len(latencies)-1]

				b.ReportMetric(float64(p50.Nanoseconds()), "p50_ns")
				b.ReportMetric(float64(p95.Nanoseconds()), "p95_ns")
				b.ReportMetric(float64(p99.Nanoseconds()), "p99_ns")
				b.ReportMetric(float64(max.Nanoseconds()), "max_ns")
			}
		})
	}
}

// Benchmark PUT operations with capacity scaling analysis
func BenchmarkCache_PUT_CapacityScaling(b *testing.B) {
	for _, tc := range loadTestCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// Calculate theoretical limits
			recordsPerMemtable := tc.capacity / RECORD_SIZE
			if recordsPerMemtable == 0 {
				recordsPerMemtable = 1
			}

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)

			for i := 0; i < b.N; i++ {
				key := generateKey(i)
				value := generateValue(i)
				cache.Put(key, value)
				totalBytes += RECORD_SIZE
			}

			elapsed := time.Since(start)

			// Scaling analysis metrics
			recordsPerSecond := float64(b.N) / elapsed.Seconds()
			mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
			throughputPerMB := recordsPerSecond / (float64(tc.capacity) / (1024 * 1024))
			efficiencyRatio := float64(b.N) / float64(recordsPerMemtable)

			b.ReportMetric(recordsPerSecond, "records/sec")
			b.ReportMetric(mbPerSecond, "MB/sec")
			b.ReportMetric(throughputPerMB, "records/sec/MB_capacity")
			b.ReportMetric(efficiencyRatio, "efficiency_ratio")
			b.ReportMetric(float64(recordsPerMemtable), "records/memtable")
			b.ReportMetric(float64(tc.capacity)/(1024*1024), "capacity_MB")
		})
	}
}

// Simple robust load test that avoids flush timing issues
func BenchmarkCache_PUT_SimpleLoad(b *testing.B) {
	// Use a subset of capacities that are more stable
	simpleCapacities := []struct {
		name     string
		capacity int64
	}{
		{"64KB", 64 * 1024},
		{"128KB", 128 * 1024},
		{"256KB", 256 * 1024},
		{"512KB", 512 * 1024},
		{"1MB", 1024 * 1024},
		{"2MB", 2 * 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
		{"8MB", 8 * 1024 * 1024},
	}

	for _, tc := range simpleCapacities {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// Calculate records per memtable for analysis
			recordsPerMemtable := tc.capacity / RECORD_SIZE
			if recordsPerMemtable == 0 {
				recordsPerMemtable = 1
			}

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			successfulPuts := 0
			totalBytes := int64(0)

			for i := 0; i < b.N; i++ {
				key := generateKey(i)
				value := generateValue(i)

				// Simple PUT without complex error handling
				cache.Put(key, value)
				successfulPuts++
				totalBytes += RECORD_SIZE
			}

			elapsed := time.Since(start)

			// Calculate metrics
			if elapsed.Seconds() > 0 {
				recordsPerSecond := float64(successfulPuts) / elapsed.Seconds()
				mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
				avgLatencyNs := elapsed.Nanoseconds() / int64(successfulPuts)

				b.ReportMetric(recordsPerSecond, "records/sec")
				b.ReportMetric(mbPerSecond, "MB/sec")
				b.ReportMetric(float64(avgLatencyNs), "ns/op")
				b.ReportMetric(float64(recordsPerMemtable), "records/memtable")
				b.ReportMetric(float64(tc.capacity)/(1024*1024), "capacity_MB")
			}
		})
	}
}

// Benchmark GET operations to complement PUT tests
func BenchmarkCache_GET_LoadTest(b *testing.B) {
	for _, tc := range []struct {
		name     string
		capacity int64
	}{
		{"64KB", 64 * 1024},
		{"128KB", 128 * 1024},
		{"256KB", 256 * 1024},
		{"512KB", 512 * 1024},
		{"1MB", 1024 * 1024},
	} {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			// Pre-populate cache with some data
			numKeys := 10
			keys := make([]string, numKeys)
			for i := 0; i < numKeys; i++ {
				key := generateKey(i)
				value := generateValue(i)
				cache.Put(key, value)
				keys[i] = key
			}

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			hits := 0
			misses := 0

			for i := 0; i < b.N; i++ {
				key := keys[i%numKeys] // Cycle through existing keys
				result := cache.Get(key)
				if result != nil {
					hits++
				} else {
					misses++
				}
			}

			elapsed := time.Since(start)

			// Calculate GET metrics
			if elapsed.Seconds() > 0 {
				getsPerSecond := float64(b.N) / elapsed.Seconds()
				hitRate := float64(hits) / float64(b.N) * 100
				avgLatencyNs := elapsed.Nanoseconds() / int64(b.N)

				b.ReportMetric(getsPerSecond, "gets/sec")
				b.ReportMetric(hitRate, "hit_rate_%")
				b.ReportMetric(float64(avgLatencyNs), "ns/op")
				b.ReportMetric(float64(hits), "hits")
				b.ReportMetric(float64(misses), "misses")
			}
		})
	}
}

// Throughput test for specific record sizes and capacities
func BenchmarkCache_Throughput_Analysis(b *testing.B) {
	testCases := []struct {
		name      string
		capacity  int64
		keySize   int
		valueSize int
	}{
		{"64KB_32B_16KB", 64 * 1024, 32, 16 * 1024},
		{"128KB_32B_16KB", 128 * 1024, 32, 16 * 1024},
		{"256KB_32B_16KB", 256 * 1024, 32, 16 * 1024},
		{"512KB_32B_16KB", 512 * 1024, 32, 16 * 1024},
		{"1MB_32B_16KB", 1024 * 1024, 32, 16 * 1024},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			cache := NewCache(tc.capacity)
			defer cache.Discard()

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// Custom key and value for this test case
			recordSize := tc.keySize + tc.valueSize
			recordsPerMemtable := tc.capacity / int64(recordSize)
			if recordsPerMemtable == 0 {
				recordsPerMemtable = 1
			}

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)

			for i := 0; i < b.N; i++ {
				// Generate key of specified size
				key := fmt.Sprintf("key_%010d", i)
				for len(key) < tc.keySize {
					key += "_"
				}
				if len(key) > tc.keySize {
					key = key[:tc.keySize]
				}

				// Generate value of specified size
				value := make([]byte, tc.valueSize)
				for j := range value {
					value[j] = byte((i + j) % 256)
				}

				cache.Put(key, value)
				totalBytes += int64(recordSize)
			}

			elapsed := time.Since(start)

			// Throughput analysis metrics
			if elapsed.Seconds() > 0 {
				recordsPerSecond := float64(b.N) / elapsed.Seconds()
				mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
				capacityUtilization := float64(totalBytes) / float64(tc.capacity) * 100
				avgLatencyUs := elapsed.Microseconds() / int64(b.N)

				b.ReportMetric(recordsPerSecond, "records/sec")
				b.ReportMetric(mbPerSecond, "MB/sec")
				b.ReportMetric(float64(avgLatencyUs), "us/op")
				b.ReportMetric(capacityUtilization, "capacity_util_%")
				b.ReportMetric(float64(recordsPerMemtable), "records/memtable")
			}
		})
	}
}
