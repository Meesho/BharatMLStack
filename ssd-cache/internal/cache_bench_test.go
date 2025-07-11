package internal

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

const (
	KEY_SIZE             = 32                    // 32 byte keys
	VALUE_SIZE           = 16 * 1024             // 16KB values
	RECORD_SIZE          = KEY_SIZE + VALUE_SIZE // Total record size
	PREGENERATED_RECORDS = 1 * 1024 * 1024       // 1M records
)

// Pre-generated key-value pairs for benchmarking
type KeyValuePair struct {
	Key   string
	Value []byte
}

var pregeneratedData []KeyValuePair

// Initialize pre-generated data once
func init() {
	fmt.Printf("Pre-generating %d key-value pairs for benchmarking...\n", PREGENERATED_RECORDS)
	pregeneratedData = make([]KeyValuePair, PREGENERATED_RECORDS)

	for i := 0; i < PREGENERATED_RECORDS; i++ {
		pregeneratedData[i] = KeyValuePair{
			Key:   generateKey(i),
			Value: generateValue(pregeneratedData[i].Key),
		}
	}
	fmt.Printf("Pre-generation complete. Ready for benchmarking.\n")
}

// Test capacities from 64KB to 1GB
var loadTestCapacities = []struct {
	name     string
	capacity int64
}{
	// {"64KB", 64 * 1024},
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
	{"1GB", 1024 * 1024 * 1024},
	// {"2GB", 2 * 1024 * 1024 * 1024},
	// {"4GB", 4 * 1024 * 1024 * 1024},
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
func generateValue(key string) []byte {
	value := make([]byte, VALUE_SIZE)
	offset := 0
	for i := 0; i < VALUE_SIZE; i++ {
		copy(value[offset:], []byte(key))
		offset += len(key)
		if offset >= VALUE_SIZE {
			break
		}
	}
	return value
}

// Benchmark PUT operations with load testing
func BenchmarkCache_PUT_LoadTest(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	ch := make(chan *Cache)
	go func() {
		for {
			cache := <-ch
			cache.Discard()
		}
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	time.Sleep(1 * time.Second)
	for _, tc := range loadTestCapacities {
		cache := NewCacheV2(CacheConfig{
			MemtableCapacity:     tc.capacity,
			BlockSizeMultipliers: []int{1, 2, 4, 5},
			LRUCacheSize:         21024 * 1024 * 1024,
			FilePunchHoleSize:    1024 * 1024 * 1024,
			FileMaxSize:          tc.capacity * 2,
		})
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)

			for i := 0; i < b.N; i++ {
				// Use pre-generated data, cycling through if needed
				dataIndex := i % PREGENERATED_RECORDS
				kv := pregeneratedData[dataIndex]

				cache.Put(kv.Key, kv.Value)
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
			var stat unix.Stat_t
			if err := unix.Stat(cache.writeFile.Name(), &stat); err != nil {
				fmt.Printf("Failed to stat file: %v\n", err)
				return
			}
			allocatedSize := stat.Blocks * 512
			b.ReportMetric(float64(allocatedSize)/(1024*1024), "allocated_MB")
		})
		ch <- cache
	}
	close(ch)
}

// Benchmark GET operations with load testing
func BenchmarkCache_GET_LoadTest(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	ch := make(chan *Cache)
	go func() {
		for {
			cache := <-ch
			cache.Discard()
		}
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	time.Sleep(1 * time.Second)

	for _, tc := range loadTestCapacities {
		cache := NewCacheV2(CacheConfig{
			MemtableCapacity:     tc.capacity,
			BlockSizeMultipliers: []int{1, 2, 4, 5},
			LRUCacheSize:         21024 * 1024 * 1024,
			FilePunchHoleSize:    1024 * 1024 * 1024,
			FileMaxSize:          tc.capacity * 15,
		})

		// Pre-populate cache with data for GET testing
		recordsToStore := PREGENERATED_RECORDS

		for i := 0; i < recordsToStore; i++ {
			kv := pregeneratedData[i]
			cache.Put(kv.Key, kv.Value)
		}

		stat, err := cache.writeFile.Stat()
		if err != nil {
			b.Fatalf("Failed to get file size: %v", err)
		}
		fmt.Printf("File size: %d\n", stat.Size())

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalBytes := int64(0)
			hitCount := 0
			missCount := 0
			for i := 0; i < b.N; i++ {
				// Use keys from stored records, cycling through if needed
				dataIndex := int(time.Now().UnixNano() % int64(recordsToStore))
				key := pregeneratedData[dataIndex].Key

				value := cache.Get(key)
				//b.Logf("key: %s, value: %s", key, string(value))
				if value != nil && bytes.Equal(value, pregeneratedData[dataIndex].Value) {
					hitCount++
					totalBytes += int64(len(value))
				} else {
					missCount++
				}
			}

			elapsed := time.Since(start)

			// Calculate performance metrics
			recordsPerSecond := float64(b.N) / elapsed.Seconds()
			mbPerSecond := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
			avgLatencyNs := elapsed.Nanoseconds() / int64(b.N)
			hitRate := float64(hitCount) / float64(b.N) * 100
			missRate := float64(missCount) / float64(b.N) * 100
			b.ReportMetric(recordsPerSecond, "records/sec")
			b.ReportMetric(mbPerSecond, "MB/sec")
			b.ReportMetric(float64(avgLatencyNs), "ns/op")
			b.ReportMetric(hitRate, "hit_rate_%")
			b.ReportMetric(missRate, "miss_rate_%")
			b.ReportMetric(float64(recordsToStore), "stored_records")
			b.ReportMetric(float64(cache.fromMemtableCount), "from_memtable_count")
			b.ReportMetric(float64(cache.fromDiskCount), "from_disk_count")
			b.ReportMetric(float64(cache.fromLruCacheCount), "from_lru_cache_count")
		})
		ch <- cache
	}
	close(ch)
}
