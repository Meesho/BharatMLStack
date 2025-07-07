package allocator

import (
	"reflect"
	"testing"
)

func TestNewSlabAllocator_DefaultSizeClasses(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{})

	sizeClasses := allocator.GetSizeClasses()
	expectedSizeClasses := []int{
		2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576,
	}

	if !reflect.DeepEqual(expectedSizeClasses, sizeClasses) {
		t.Errorf("Expected size classes %v, got %v", expectedSizeClasses, sizeClasses)
	}
	if len(expectedSizeClasses) != len(allocator.pools) {
		t.Errorf("Expected %d pools, got %d", len(expectedSizeClasses), len(allocator.pools))
	}
}

func TestNewSlabAllocator_CustomSizeClasses(t *testing.T) {
	customSizes := []int{1024, 4096, 16384, 65536}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: customSizes})

	sizeClasses := allocator.GetSizeClasses()
	if !reflect.DeepEqual(customSizes, sizeClasses) {
		t.Errorf("Expected size classes %v, got %v", customSizes, sizeClasses)
	}
	if len(customSizes) != len(allocator.pools) {
		t.Errorf("Expected %d pools, got %d", len(customSizes), len(allocator.pools))
	}
}

func TestNewSlabAllocator_UnsortedSizeClasses(t *testing.T) {
	unsortedSizes := []int{65536, 1024, 16384, 4096}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: unsortedSizes})

	expectedSorted := []int{1024, 4096, 16384, 65536}
	sizeClasses := allocator.GetSizeClasses()
	if !reflect.DeepEqual(expectedSorted, sizeClasses) {
		t.Errorf("Expected sorted size classes %v, got %v", expectedSorted, sizeClasses)
	}
}

func TestGetBuffer_ExactSizeMatch(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	buffer, actualSize := allocator.GetBuffer(1024)

	if len(buffer) != 1024 {
		t.Errorf("Expected buffer length 1024, got %d", len(buffer))
	}
	if actualSize != 1024 {
		t.Errorf("Expected actual size 1024, got %d", actualSize)
	}
	if cap(buffer) != 1024 {
		t.Errorf("Expected buffer capacity 1024, got %d", cap(buffer))
	}

	// Verify buffer is zeroed
	for i, b := range buffer {
		if b != 0 {
			t.Errorf("Buffer should be zeroed at index %d, got %d", i, b)
		}
	}
}

func TestGetBuffer_SmallerThanSizeClass(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	buffer, actualSize := allocator.GetBuffer(512)

	if len(buffer) != 512 {
		t.Errorf("Expected buffer length 512, got %d", len(buffer))
	}
	if actualSize != 1024 {
		t.Errorf("Expected actual size 1024 (next larger size class), got %d", actualSize)
	}
	if cap(buffer) != 1024 {
		t.Errorf("Expected buffer capacity 1024, got %d", cap(buffer))
	}
}

func TestGetBuffer_LargerThanAllSizeClasses(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	buffer, actualSize := allocator.GetBuffer(16384)

	if len(buffer) != 16384 {
		t.Errorf("Expected buffer length 16384, got %d", len(buffer))
	}
	if actualSize != 16384 {
		t.Errorf("Expected actual size 16384 (direct allocation), got %d", actualSize)
	}
	if cap(buffer) != 16384 {
		t.Errorf("Expected buffer capacity 16384, got %d", cap(buffer))
	}
}

func TestGetBuffer_ZeroSize(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	buffer, actualSize := allocator.GetBuffer(0)

	if len(buffer) != 0 {
		t.Errorf("Expected buffer length 0, got %d", len(buffer))
	}
	if actualSize != 1024 {
		t.Errorf("Expected actual size 1024 (smallest size class), got %d", actualSize)
	}
	if cap(buffer) != 1024 {
		t.Errorf("Expected buffer capacity 1024, got %d", cap(buffer))
	}
}

func TestPutBuffer_ValidSizeClass(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	// Get a buffer first
	buffer, actualSize := allocator.GetBuffer(1024)

	// Modify the buffer to ensure it gets cleared on next get
	buffer[0] = 42
	buffer[100] = 99

	// Put the buffer back
	allocator.PutBuffer(buffer)

	// Get another buffer of the same size
	newBuffer, newActualSize := allocator.GetBuffer(1024)

	if actualSize != newActualSize {
		t.Errorf("Expected actual size %d, got %d", actualSize, newActualSize)
	}
	if newBuffer[0] != 0 {
		t.Errorf("Buffer should be zeroed at index 0, got %d", newBuffer[0])
	}
	if newBuffer[100] != 0 {
		t.Errorf("Buffer should be zeroed at index 100, got %d", newBuffer[100])
	}
}

func TestPutBuffer_InvalidSizeClass(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	// Create a buffer that doesn't match any size class
	buffer := make([]byte, 2048)

	// This should not panic and should handle gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PutBuffer with invalid size class should not panic, got: %v", r)
		}
	}()
	allocator.PutBuffer(buffer)
}

func TestPutBuffer_DirectlyAllocatedBuffer(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	// Get a buffer larger than all size classes
	buffer, _ := allocator.GetBuffer(16384)

	// Put it back (should handle gracefully since it wasn't from a pool)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PutBuffer with directly allocated buffer should not panic, got: %v", r)
		}
	}()
	allocator.PutBuffer(buffer)
}

func TestFindSizeClass(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	testCases := []struct {
		size     int
		expected int
	}{
		{512, 0},    // Should use 1024 (index 0)
		{1024, 0},   // Exact match for 1024 (index 0)
		{2048, 1},   // Should use 4096 (index 1)
		{4096, 1},   // Exact match for 4096 (index 1)
		{8192, 2},   // Exact match for 8192 (index 2)
		{16384, -1}, // Larger than all size classes
	}

	for _, tc := range testCases {
		result := allocator.findSizeClass(tc.size)
		if result != tc.expected {
			t.Errorf("Size %d should map to index %d, got %d", tc.size, tc.expected, result)
		}
	}
}

func TestFindSizeClassExact(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	testCases := []struct {
		size     int
		expected int
	}{
		{1024, 0},   // Exact match for 1024 (index 0)
		{4096, 1},   // Exact match for 4096 (index 1)
		{8192, 2},   // Exact match for 8192 (index 2)
		{512, -1},   // No exact match
		{2048, -1},  // No exact match
		{16384, -1}, // No exact match
	}

	for _, tc := range testCases {
		result := allocator.findSizeClassExact(tc.size)
		if result != tc.expected {
			t.Errorf("Size %d should map to exact index %d, got %d", tc.size, tc.expected, result)
		}
	}
}

func TestGetSizeClasses(t *testing.T) {
	sizeClasses := []int{1024, 4096, 8192}
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: sizeClasses})

	result := allocator.GetSizeClasses()

	// Should return a copy, not the original slice
	if !reflect.DeepEqual(sizeClasses, result) {
		t.Errorf("Expected %v, got %v", sizeClasses, result)
	}

	// Modifying the returned slice should not affect the original
	result[0] = 9999
	originalSizeClasses := allocator.GetSizeClasses()
	if originalSizeClasses[0] != 1024 {
		t.Errorf("Modifying returned slice should not affect original, expected 1024, got %d", originalSizeClasses[0])
	}
}

func TestGetStats(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	// Get initial stats
	stats := allocator.GetStats()

	expectedSizeClasses := []int{1024, 4096, 8192}
	if !reflect.DeepEqual(expectedSizeClasses, stats.SizeClasses) {
		t.Errorf("Expected size classes %v, got %v", expectedSizeClasses, stats.SizeClasses)
	}
	if len(stats.PoolStats) != 3 {
		t.Errorf("Expected 3 pool stats, got %d", len(stats.PoolStats))
	}

	// All pools should start with 0 objects in use
	for i, count := range stats.PoolStats {
		if count != 0 {
			t.Errorf("Pool %d should start with 0 objects in use, got count %d", i, count)
		}
	}
}

func TestGetStats_AfterBufferOperations(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096}})

	// Get some buffers
	buffer1, _ := allocator.GetBuffer(1024)
	buffer2, _ := allocator.GetBuffer(1024)
	buffer3, _ := allocator.GetBuffer(4096)

	// Put one buffer back
	allocator.PutBuffer(buffer1)

	// Check stats - Count() returns objects currently in use (checked out)
	stats := allocator.GetStats()

	// First pool should have 1 buffer in use (we got 2, put 1 back, so 1 still in use)
	if stats.PoolStats[0] != 1 {
		t.Errorf("Expected first pool to have 1 buffer in use, got %d", stats.PoolStats[0])
	}
	// Second pool should have 1 buffer in use (we got 1 but didn't put it back)
	if stats.PoolStats[1] != 1 {
		t.Errorf("Expected second pool to have 1 buffer in use, got %d", stats.PoolStats[1])
	}

	// Put remaining buffers back
	allocator.PutBuffer(buffer2)
	allocator.PutBuffer(buffer3)

	// Check final stats - all buffers should be returned, so 0 in use
	finalStats := allocator.GetStats()
	if finalStats.PoolStats[0] != 0 {
		t.Errorf("Expected first pool to have 0 buffers in use, got %d", finalStats.PoolStats[0])
	}
	if finalStats.PoolStats[1] != 0 {
		t.Errorf("Expected second pool to have 0 buffers in use, got %d", finalStats.PoolStats[1])
	}
}

func TestString(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 8192}})

	result := allocator.String()
	expected := "SlabAllocator{SizeClasses: [1024 4096 8192]}"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestPrintStats(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096}})

	// Get and put back some buffers to have some stats
	buffer1, _ := allocator.GetBuffer(1024)
	buffer2, _ := allocator.GetBuffer(4096)
	allocator.PutBuffer(buffer1)
	allocator.PutBuffer(buffer2)

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintStats should not panic, got: %v", r)
		}
	}()
	allocator.PrintStats()
}

func TestBufferReuseAndZeroing(t *testing.T) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024}})

	// Get a buffer and fill it with data
	buffer1, _ := allocator.GetBuffer(1024)
	for i := range buffer1 {
		buffer1[i] = byte(i % 256)
	}

	// Put it back
	allocator.PutBuffer(buffer1)

	// Get a new buffer of the same size
	buffer2, _ := allocator.GetBuffer(1024)

	// Should be zeroed
	for i, b := range buffer2 {
		if b != 0 {
			t.Errorf("Buffer should be zeroed at index %d, got %d", i, b)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("EmptyConfig", func(t *testing.T) {
		allocator := NewSlabAllocator(SlabConfig{})
		if allocator == nil {
			t.Error("NewSlabAllocator should not return nil")
		}
		if len(allocator.GetSizeClasses()) == 0 {
			t.Error("Empty config should result in default size classes")
		}
	})

	t.Run("SingleSizeClass", func(t *testing.T) {
		allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024}})

		buffer, actualSize := allocator.GetBuffer(512)
		if len(buffer) != 512 {
			t.Errorf("Expected buffer length 512, got %d", len(buffer))
		}
		if actualSize != 1024 {
			t.Errorf("Expected actual size 1024, got %d", actualSize)
		}

		buffer2, actualSize2 := allocator.GetBuffer(2048)
		if len(buffer2) != 2048 {
			t.Errorf("Expected buffer length 2048, got %d", len(buffer2))
		}
		if actualSize2 != 2048 {
			t.Errorf("Expected actual size 2048 (direct allocation), got %d", actualSize2)
		}
	})

	t.Run("NegativeSize", func(t *testing.T) {
		allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024}})

		// This should panic due to negative slice bounds
		defer func() {
			if r := recover(); r == nil {
				t.Error("GetBuffer with negative size should panic")
			}
		}()

		allocator.GetBuffer(-100)
	})
}

func BenchmarkGetBuffer(b *testing.B) {
	allocator := NewSlabAllocator(SlabConfig{SizeClasses: []int{1024, 4096, 16384, 65536}})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		size := 1024 + (i%4)*1024
		buffer, _ := allocator.GetBuffer(size)
		allocator.PutBuffer(buffer)
	}
}

func BenchmarkGetBufferDirect(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		size := 1024 + (i%4)*1024
		buffer := make([]byte, size)
		_ = buffer
	}
}
