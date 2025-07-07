package allocator

import (
	"runtime"
	"sync"
	"testing"
)

// Benchmark basic allocation and deallocation patterns
func BenchmarkSlabAllocator_BasicAllocation(b *testing.B) {
	sizes := []int{1024, 4096, 16384, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizes})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark direct allocation for comparison
func BenchmarkDirectAllocation(b *testing.B) {
	sizes := []int{1024, 4096, 16384, 65536}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buffer := make([]byte, size)
		_ = buffer // Prevent optimization
	}
}

// Benchmark small buffer allocations
func BenchmarkSlabAllocator_SmallBuffers(b *testing.B) {
	sizes := []int{64, 128, 256, 512}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizes})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark large buffer allocations
func BenchmarkSlabAllocator_LargeBuffers(b *testing.B) {
	sizes := []int{65536, 131072, 262144, 524288}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizes})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark allocation of exact size class matches
func BenchmarkSlabAllocator_ExactSizeMatch(b *testing.B) {
	sizeClasses := []int{1024, 4096, 16384, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizeClasses[i%len(sizeClasses)]
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark allocation of sizes that don't match exactly (need next larger size class)
func BenchmarkSlabAllocator_InexactSizeMatch(b *testing.B) {
	sizeClasses := []int{1024, 4096, 16384, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	// Request sizes that fall between size classes
	requestSizes := []int{512, 2048, 8192, 32768}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := requestSizes[i%len(requestSizes)]
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark allocation of sizes larger than all size classes (direct allocation)
func BenchmarkSlabAllocator_OversizedAllocation(b *testing.B) {
	sizeClasses := []int{1024, 4096, 16384, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	// Request sizes larger than the largest size class
	oversizedSizes := []int{131072, 262144, 524288, 1048576}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := oversizedSizes[i%len(oversizedSizes)]
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark high-frequency allocation pattern
func BenchmarkSlabAllocator_HighFrequency(b *testing.B) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024}})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buffer, _ := allocator.GetBuffer(1024)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark memory reuse pattern (get many, put many back)
func BenchmarkSlabAllocator_BatchReuse(b *testing.B) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096}})
	batchSize := 100

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Allocate a batch of buffers
		buffers := make([][]byte, batchSize)
		for j := 0; j < batchSize; j++ {
			size := 1024
			if j%2 == 0 {
				size = 4096
			}
			buffers[j], _ = allocator.GetBuffer(size)
		}

		// Use the buffers (simulate work)
		for j := 0; j < batchSize; j++ {
			if len(buffers[j]) > 0 {
				buffers[j][0] = byte(j)
			}
		}

		// Return all buffers
		for j := 0; j < batchSize; j++ {
			allocator.PutBuffer(buffers[j])
		}
	}
}

// Benchmark allocation pattern with mixed sizes
func BenchmarkSlabAllocator_MixedSizes(b *testing.B) {
	sizeClasses := []int{64, 256, 1024, 4096, 16384, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Allocate different sizes in a realistic pattern
		// Smaller sizes are more common
		var size int
		switch i % 10 {
		case 0, 1, 2, 3, 4: // 50% small
			size = sizeClasses[i%2]
		case 5, 6, 7: // 30% medium
			size = sizeClasses[2+i%2]
		case 8, 9: // 20% large
			size = sizeClasses[4+i%2]
		}

		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark findSizeClass performance
func BenchmarkSlabAllocator_FindSizeClass(b *testing.B) {
	sizeClasses := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	testSizes := []int{32, 100, 300, 700, 1500, 3000, 6000, 12000, 25000, 50000, 100000}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := testSizes[i%len(testSizes)]
		_ = allocator.findSizeClass(size)
	}
}

// Benchmark findSizeClassExact performance (O(1) with map)
func BenchmarkSlabAllocator_FindSizeClassExact(b *testing.B) {
	sizeClasses := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizeClasses[i%len(sizeClasses)]
		_ = allocator.findSizeClassExact(size)
	}
}

// Benchmark default configuration performance
func BenchmarkSlabAllocator_DefaultConfig(b *testing.B) {
	allocator := NewSlabAllocator(SlabConfig{}) // Use default size classes

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Test with common request sizes
		var size int
		switch i % 8 {
		case 0:
			size = 64
		case 1:
			size = 256
		case 2:
			size = 1024
		case 3:
			size = 4096
		case 4:
			size = 8192
		case 5:
			size = 16384
		case 6:
			size = 65536
		case 7:
			size = 262144
		}

		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

// Benchmark memory pressure scenario
func BenchmarkSlabAllocator_MemoryPressure(b *testing.B) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 16384}})

	// Force garbage collection before starting
	runtime.GC()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buffers := make([][]byte, 50)

		// Allocate many buffers
		for j := 0; j < 50; j++ {
			size := 1024 << (j % 3) // 1KB, 2KB, 4KB pattern
			buffers[j], _ = allocator.GetBuffer(size)
		}

		// Use buffers
		for j := 0; j < 50; j++ {
			if len(buffers[j]) > 10 {
				buffers[j][j%len(buffers[j])] = byte(j)
			}
		}

		// Return some buffers (simulate partial cleanup)
		for j := 0; j < 25; j++ {
			allocator.PutBuffer(buffers[j])
		}

		// Force periodic garbage collection
		if i%100 == 0 {
			runtime.GC()
		}

		// Return remaining buffers
		for j := 25; j < 50; j++ {
			allocator.PutBuffer(buffers[j])
		}
	}
}

// Benchmark comparison: SlabAllocator vs sync.Pool
func BenchmarkComparison_SyncPool(b *testing.B) {
	pool1024 := &sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}
	pool4096 := &sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}

	pools := []*sync.Pool{pool1024, pool4096}
	sizes := []int{1024, 4096}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % 2
		buffer := pools[idx].Get().([]byte)

		// Clear buffer (similar to what SlabAllocator does)
		for j := range buffer[:sizes[idx]] {
			buffer[j] = 0
		}

		pools[idx].Put(buffer)
	}
}

// Benchmark for getting statistics
func BenchmarkSlabAllocator_GetStats(b *testing.B) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 16384, 65536}})

	// Allocate some buffers to have meaningful stats
	buffers := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		size := 1024 << (i % 4)
		buffers[i], _ = allocator.GetBuffer(size)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = allocator.GetStats()
	}

	// Clean up
	for i := 0; i < 100; i++ {
		allocator.PutBuffer(buffers[i])
	}
}
