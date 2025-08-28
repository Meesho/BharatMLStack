// Benchmark tests for Memtable operations optimized for single-threaded performance
// Uses 50GB max file size and 1GB memtable page size as specified
package memtables

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Meesho/BharatMLStack/flashring/external/fs"
)

const (
	// Configuration for single-threaded benchmarks
	BENCH_MAX_FILE_SIZE   = 50 * 1024 * 1024 * 1024 // 50GB max file size
	BENCH_PAGE_SIZE       = 1024 * 1024 * 1024      // 1GB memtable page size
	BENCH_PUNCH_HOLE_SIZE = 64 * 1024 * 1024        // 64MB punch hole size

	// Data sizes for single-threaded performance testing
	SMALL_DATA_SIZE      = 256         // 256 bytes - typical small record
	MEDIUM_DATA_SIZE     = 4096        // 4KB - typical medium record
	LARGE_DATA_SIZE      = 64 * 1024   // 64KB - large record
	VERY_LARGE_DATA_SIZE = 1024 * 1024 // 1MB - very large record
)

// Helper function to create benchmark file
func createBenchmarkFile(b *testing.B) *fs.WrapAppendFile {
	filename := filepath.Join("/media/a0d00kc/freedom/tmp/bench_memtable.dat")

	config := fs.FileConfig{
		Filename:          filename,
		MaxFileSize:       BENCH_MAX_FILE_SIZE,
		FilePunchHoleSize: BENCH_PUNCH_HOLE_SIZE,
		BlockSize:         fs.BLOCK_SIZE,
	}

	file, err := fs.NewWrapAppendFile(config)
	if err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}
	return file
}

// Helper function to create benchmark page
func createBenchmarkPage() *fs.AlignedPage {
	return fs.NewAlignedPage(BENCH_PAGE_SIZE)
}

// Helper function to create benchmark memtable
func createBenchmarkMemtable(b *testing.B) (*Memtable, *fs.WrapAppendFile, *fs.AlignedPage) {
	file := createBenchmarkFile(b)
	page := createBenchmarkPage()

	config := MemtableConfig{
		capacity: BENCH_PAGE_SIZE,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		cleanup(file, page)
		b.Fatalf("Failed to create benchmark memtable: %v", err)
	}

	return memtable, file, page
}

// Helper function to generate random data
func generateRandomData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// Benchmark Put operations with different data sizes
func BenchmarkMemtable_Put_Small(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(SMALL_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(SMALL_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		if memtable.readyForFlush {
			// Reset memtable for continued benchmarking
			memtable.currentOffset = 0
			memtable.readyForFlush = false
		}

		_, _, readyForFlush := memtable.Put(data)
		if readyForFlush {
			// Don't count flush operations in this benchmark
			b.StopTimer()
			memtable.currentOffset = 0
			memtable.readyForFlush = false
			b.StartTimer()
		}
	}
}

func BenchmarkMemtable_Put_Medium(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(MEDIUM_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(MEDIUM_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		if memtable.readyForFlush {
			memtable.currentOffset = 0
			memtable.readyForFlush = false
		}

		_, _, readyForFlush := memtable.Put(data)
		if readyForFlush {
			b.StopTimer()
			memtable.currentOffset = 0
			memtable.readyForFlush = false
			b.StartTimer()
		}
	}
}

func BenchmarkMemtable_Put_Large(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(LARGE_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(LARGE_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		if memtable.readyForFlush {
			memtable.currentOffset = 0
			memtable.readyForFlush = false
		}

		_, _, readyForFlush := memtable.Put(data)
		if readyForFlush {
			b.StopTimer()
			memtable.currentOffset = 0
			memtable.readyForFlush = false
			b.StartTimer()
		}
	}
}

func BenchmarkMemtable_Put_VeryLarge(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(VERY_LARGE_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(VERY_LARGE_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		if memtable.readyForFlush {
			memtable.currentOffset = 0
			memtable.readyForFlush = false
		}

		_, _, readyForFlush := memtable.Put(data)
		if readyForFlush {
			b.StopTimer()
			memtable.currentOffset = 0
			memtable.readyForFlush = false
			b.StartTimer()
		}
	}
}

// Benchmark Get operations
func BenchmarkMemtable_Get_Small(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	// Pre-populate memtable with data
	data := generateRandomData(SMALL_DATA_SIZE)
	numEntries := BENCH_PAGE_SIZE / SMALL_DATA_SIZE / 2 // Fill half the memtable

	offsets := make([]int, numEntries)
	lengths := make([]uint16, numEntries)

	for i := 0; i < numEntries; i++ {
		offset, length, _ := memtable.Put(data)
		offsets[i] = offset
		lengths[i] = length
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(SMALL_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		idx := i % numEntries
		_, err := memtable.Get(offsets[idx], lengths[idx])
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

func BenchmarkMemtable_Get_Medium(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(MEDIUM_DATA_SIZE)
	numEntries := BENCH_PAGE_SIZE / MEDIUM_DATA_SIZE / 2

	offsets := make([]int, numEntries)
	lengths := make([]uint16, numEntries)

	for i := 0; i < numEntries; i++ {
		offset, length, _ := memtable.Put(data)
		offsets[i] = offset
		lengths[i] = length
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(MEDIUM_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		idx := i % numEntries
		_, err := memtable.Get(offsets[idx], lengths[idx])
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

func BenchmarkMemtable_Get_Large(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(LARGE_DATA_SIZE)
	numEntries := BENCH_PAGE_SIZE / LARGE_DATA_SIZE / 2

	offsets := make([]int, numEntries)
	lengths := make([]uint16, numEntries)

	for i := 0; i < numEntries; i++ {
		offset, length, _ := memtable.Put(data)
		offsets[i] = offset
		lengths[i] = length
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(LARGE_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		idx := i % numEntries
		_, err := memtable.Get(offsets[idx], lengths[idx])
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// Benchmark Flush operations
func BenchmarkMemtable_Flush(b *testing.B) {
	file := createBenchmarkFile(b)
	defer cleanup(file, nil)

	// Create fresh memtable for each iteration
	page := createBenchmarkPage()
	config := MemtableConfig{
		capacity: BENCH_PAGE_SIZE,
		id:       uint32(0),
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		b.Fatalf("Failed to create memtable: %v", err)
	}

	// Fill memtable to near capacity then trigger flush with overflow
	fillData := generateRandomData(BENCH_PAGE_SIZE - 1000)
	memtable.Put(fillData)

	// Now add data that will exceed capacity to trigger flush
	overflowData := generateRandomData(2000) // This will exceed capacity
	_, _, readyForFlush := memtable.Put(overflowData)
	if !readyForFlush {
		b.Fatalf("Failed to trigger flush - memtable should be ready for flush")
	}
	b.ReportAllocs()
	b.SetBytes(BENCH_PAGE_SIZE)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		_, _, err = memtable.Flush()
		if err != nil {
			b.Fatalf("Flush failed: %v", err)
		}
		// Force re-flush same data in each iteration
		memtable.readyForFlush = true
	}
	fs.Unmap(page)
}

// Benchmark mixed operations (realistic usage pattern)
func BenchmarkMemtable_MixedOperations(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	// Pre-populate with some data
	initialData := generateRandomData(MEDIUM_DATA_SIZE)
	numInitial := 1000
	offsets := make([]int, numInitial)
	lengths := make([]uint16, numInitial)

	for i := 0; i < numInitial; i++ {
		offset, length, readyForFlush := memtable.Put(initialData)
		if readyForFlush {
			break
		}
		offsets[i] = offset
		lengths[i] = length
	}

	putData := generateRandomData(SMALL_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Mix of operations: 70% gets, 30% puts
		if i%10 < 7 {
			// Get operation
			idx := i % len(offsets)
			if idx < len(offsets) && lengths[idx] > 0 {
				_, err := memtable.Get(offsets[idx], lengths[idx])
				if err != nil && err != ErrOffsetOutOfBounds {
					b.Fatalf("Get failed: %v", err)
				}
			}
		} else {
			// Put operation
			if memtable.readyForFlush {
				// Reset for continued benchmarking
				memtable.currentOffset = 0
				memtable.readyForFlush = false
			}
			memtable.Put(putData)
		}
	}
}

// Benchmark sequential writes to measure throughput
func BenchmarkMemtable_SequentialWrites(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	data := generateRandomData(MEDIUM_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(MEDIUM_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		if memtable.readyForFlush {
			memtable.currentOffset = 0
			memtable.readyForFlush = false
		}

		_, _, readyForFlush := memtable.Put(data)
		if readyForFlush {
			b.StopTimer()
			memtable.currentOffset = 0
			memtable.readyForFlush = false
			b.StartTimer()
		}
	}
}

// Benchmark random access patterns
func BenchmarkMemtable_RandomAccess(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	// Pre-populate memtable
	data := generateRandomData(SMALL_DATA_SIZE)
	numEntries := BENCH_PAGE_SIZE / SMALL_DATA_SIZE / 4 // Fill quarter of memtable

	offsets := make([]int, numEntries)
	lengths := make([]uint16, numEntries)

	for i := 0; i < numEntries; i++ {
		offset, length, _ := memtable.Put(data)
		offsets[i] = offset
		lengths[i] = length
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Random access pattern
		idx := (i * 7919) % numEntries // Use prime number for better distribution
		_, err := memtable.Get(offsets[idx], lengths[idx])
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// Benchmark memory copying efficiency
func BenchmarkMemtable_MemoryCopy(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	// Test different copy sizes
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateRandomData(size)

			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				if memtable.readyForFlush {
					memtable.currentOffset = 0
					memtable.readyForFlush = false
				}

				_, _, readyForFlush := memtable.Put(data)
				if readyForFlush {
					b.StopTimer()
					memtable.currentOffset = 0
					memtable.readyForFlush = false
					b.StartTimer()
				}
			}
		})
	}
}

// Benchmark full memtable lifecycle
func BenchmarkMemtable_FullLifecycle(b *testing.B) {
	file := createBenchmarkFile(b)
	defer cleanup(file, nil)

	entrySize := MEDIUM_DATA_SIZE
	entriesPerMemtable := BENCH_PAGE_SIZE / entrySize

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(entriesPerMemtable * entrySize))

	for i := 0; i < b.N; i++ {
		// Create memtable
		page := createBenchmarkPage()
		config := MemtableConfig{
			capacity: BENCH_PAGE_SIZE,
			id:       uint32(i),
			page:     page,
			file:     file,
		}

		memtable, err := NewMemtable(config)
		if err != nil {
			b.Fatalf("Failed to create memtable: %v", err)
		}

		// Fill memtable to near capacity then trigger flush with overflow
		fillData := generateRandomData(BENCH_PAGE_SIZE - 1000)
		memtable.Put(fillData)

		// Add data that will exceed capacity to trigger flush
		overflowData := generateRandomData(2000)
		_, _, readyForFlush := memtable.Put(overflowData)
		if !readyForFlush {
			b.Fatalf("Failed to trigger flush in lifecycle test")
		}

		// Flush
		_, _, err = memtable.Flush()
		if err != nil {
			b.Fatalf("Flush failed: %v", err)
		}

		// Cleanup
		memtable.Discard()
		fs.Unmap(page)
	}
}

// Benchmark single-threaded workload patterns (read-heavy, write-heavy, mixed)
func BenchmarkMemtable_SingleThreadedWorkload(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	// Pre-populate with test data
	data := generateRandomData(SMALL_DATA_SIZE)
	numEntries := 10000
	offsets := make([]int, numEntries)
	lengths := make([]uint16, numEntries)
	validEntries := 0

	for i := 0; i < numEntries; i++ {
		offset, length, readyForFlush := memtable.Put(data)
		if readyForFlush {
			break
		}
		offsets[validEntries] = offset
		lengths[validEntries] = length
		validEntries++
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Single-threaded workload pattern: 80% reads, 20% writes
		if i%5 < 4 {
			// Read operation (80%)
			if validEntries > 0 {
				idx := i % validEntries
				memtable.Get(offsets[idx], lengths[idx])
			}
		} else {
			// Write operation (20%) - only if space available
			if !memtable.readyForFlush {
				memtable.Put(data)
			}
		}
	}
}

// Benchmark CPU-intensive single-threaded operations
func BenchmarkMemtable_CPUIntensive(b *testing.B) {
	memtable, file, page := createBenchmarkMemtable(b)
	defer cleanup(file, page)

	// Use medium-sized data for CPU-intensive operations
	data := generateRandomData(MEDIUM_DATA_SIZE)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(MEDIUM_DATA_SIZE)

	for i := 0; i < b.N; i++ {
		if memtable.readyForFlush {
			// Reset for continued benchmarking
			memtable.currentOffset = 0
			memtable.readyForFlush = false
		}

		// Perform put operation
		offset, length, readyForFlush := memtable.Put(data)
		if !readyForFlush {
			// Immediately read back the data to stress CPU
			_, err := memtable.Get(offset, length)
			if err != nil {
				b.Fatalf("Get failed: %v", err)
			}
		}
	}
}
