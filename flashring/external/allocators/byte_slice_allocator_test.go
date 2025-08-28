package allocators

import (
	"testing"
)

func TestNewByteSliceAllocator(t *testing.T) {
	t.Run("creates allocator with single size class", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		if allocator == nil {
			t.Error("Expected allocator to be non-nil")
		}
		if allocator.config.SizeClasses[0].Size != config.SizeClasses[0].Size {
			t.Errorf("Expected config to match, got %v", allocator.config)
		}
		if len(allocator.pools) != 1 {
			t.Errorf("Expected 1 pool, got %d", len(allocator.pools))
		}
		if allocator.pools[0].Meta.(Meta).Size != 1024 {
			t.Errorf("Expected pool size 1024, got %d", allocator.pools[0].Meta.(Meta).Size)
		}
		if allocator.pools[0].Meta.(Meta).Name != "ByteSlicePool-1024Bytes" {
			t.Errorf("Expected pool name 'ByteSlicePool-1024Bytes', got %s", allocator.pools[0].Meta.(Meta).Name)
		}
	})

	t.Run("creates allocator with multiple size classes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 512, MinCount: 5},
				{Size: 1024, MinCount: 10},
				{Size: 256, MinCount: 15},
			},
		}
		allocator := NewByteSliceAllocator(config)

		if allocator == nil {
			t.Error("Expected allocator to be non-nil")
		}
		if len(allocator.pools) != 3 {
			t.Errorf("Expected 3 pools, got %d", len(allocator.pools))
		}

		// Should be sorted by size
		if allocator.pools[0].Meta.(Meta).Size != 256 {
			t.Errorf("Expected first pool size 256, got %d", allocator.pools[0].Meta.(Meta).Size)
		}
		if allocator.pools[1].Meta.(Meta).Size != 512 {
			t.Errorf("Expected second pool size 512, got %d", allocator.pools[1].Meta.(Meta).Size)
		}
		if allocator.pools[2].Meta.(Meta).Size != 1024 {
			t.Errorf("Expected third pool size 1024, got %d", allocator.pools[2].Meta.(Meta).Size)
		}
	})

	t.Run("creates allocator with empty size classes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{},
		}
		allocator := NewByteSliceAllocator(config)

		if allocator == nil {
			t.Error("Expected allocator to be non-nil")
		}
		if len(allocator.pools) != 0 {
			t.Errorf("Expected 0 pools, got %d", len(allocator.pools))
		}
	})
}

func TestByteSliceAllocator_Get(t *testing.T) {
	t.Run("returns byte slice for exact size match", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(1024)
		if slice == nil {
			t.Error("Expected slice to be non-nil")
		}
		if cap(slice) != 1024 {
			t.Errorf("Expected slice capacity 1024, got %d", cap(slice))
		}
		if len(slice) != 1024 {
			t.Errorf("Expected slice length 1024, got %d", len(slice))
		}
	})

	t.Run("returns byte slice for smaller size", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(512)
		if slice == nil {
			t.Error("Expected slice to be non-nil")
		}
		if cap(slice) != 1024 {
			t.Errorf("Expected slice capacity 1024, got %d", cap(slice))
		}
		if len(slice) != 1024 {
			t.Errorf("Expected slice length 1024, got %d", len(slice))
		}
	})

	t.Run("returns smallest suitable size class", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 256, MinCount: 5},
				{Size: 512, MinCount: 10},
				{Size: 1024, MinCount: 15},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(300)
		if slice == nil {
			t.Error("Expected slice to be non-nil")
		}
		if cap(slice) != 512 {
			t.Errorf("Expected slice capacity 512, got %d", cap(slice))
		}
		if len(slice) != 512 {
			t.Errorf("Expected slice length 512, got %d", len(slice))
		}
	})

	t.Run("returns nil for size larger than all size classes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(2048)
		if slice != nil {
			t.Error("Expected slice to be nil for size larger than all size classes")
		}
	})

	t.Run("returns nil for empty size classes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(1024)
		if slice != nil {
			t.Error("Expected slice to be nil for empty size classes")
		}
	})

	t.Run("returns slice for zero size request", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(0)
		if slice == nil {
			t.Error("Expected slice to be non-nil for zero size request")
		}
		if cap(slice) != 1024 {
			t.Errorf("Expected slice capacity 1024, got %d", cap(slice))
		}
	})

	t.Run("returns slice for negative size request", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(-1)
		if slice == nil {
			t.Error("Expected slice to be non-nil for negative size request")
		}
		if cap(slice) != 1024 {
			t.Errorf("Expected slice capacity 1024, got %d", cap(slice))
		}
	})
}

func TestByteSliceAllocator_Put(t *testing.T) {
	t.Run("puts byte slice back to correct pool", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(1024)
		if slice == nil {
			t.Fatal("Expected slice to be non-nil")
		}

		// Put should not panic
		allocator.Put(slice)
	})

	t.Run("puts byte slice to smallest suitable pool", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 256, MinCount: 5},
				{Size: 512, MinCount: 10},
				{Size: 1024, MinCount: 15},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := make([]byte, 300)
		allocator.Put(slice)
		// Should not panic, even though slice wasn't from the pool
	})

	t.Run("handles slice larger than all size classes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := make([]byte, 2048)
		// Should not panic, but will log error
		allocator.Put(slice)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := make([]byte, 0)
		allocator.Put(slice)
		// Should not panic
	})

	t.Run("handles nil slice", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		// Should not panic
		allocator.Put(nil)
	})
}

func TestByteSliceAllocator_GetAndPut_Integration(t *testing.T) {
	t.Run("get and put multiple times", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 256, MinCount: 2},
				{Size: 512, MinCount: 3},
				{Size: 1024, MinCount: 5},
			},
		}
		allocator := NewByteSliceAllocator(config)

		// Get multiple slices
		slices := make([][]byte, 5)
		for i := 0; i < 5; i++ {
			slices[i] = allocator.Get(200)
			if slices[i] == nil {
				t.Errorf("Expected slice %d to be non-nil", i)
			}
			if len(slices[i]) != 256 {
				t.Errorf("Expected slice %d length 256, got %d", i, len(slices[i]))
			}
		}

		// Put them back
		for _, slice := range slices {
			allocator.Put(slice)
		}

		// Get them again
		for i := 0; i < 5; i++ {
			slice := allocator.Get(200)
			if slice == nil {
				t.Errorf("Expected slice %d to be non-nil on second get", i)
			}
			if len(slice) != 256 {
				t.Errorf("Expected slice %d length 256 on second get, got %d", i, len(slice))
			}
		}
	})

	t.Run("get and put with different sizes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 256, MinCount: 2},
				{Size: 512, MinCount: 3},
				{Size: 1024, MinCount: 5},
			},
		}
		allocator := NewByteSliceAllocator(config)

		// Get slices of different sizes
		slice256 := allocator.Get(200)
		slice512 := allocator.Get(400)
		slice1024 := allocator.Get(800)

		if len(slice256) != 256 {
			t.Errorf("Expected slice256 length 256, got %d", len(slice256))
		}
		if len(slice512) != 512 {
			t.Errorf("Expected slice512 length 512, got %d", len(slice512))
		}
		if len(slice1024) != 1024 {
			t.Errorf("Expected slice1024 length 1024, got %d", len(slice1024))
		}

		// Put them back
		allocator.Put(slice256)
		allocator.Put(slice512)
		allocator.Put(slice1024)

		// Get them again
		newSlice256 := allocator.Get(200)
		newSlice512 := allocator.Get(400)
		newSlice1024 := allocator.Get(800)

		if len(newSlice256) != 256 {
			t.Errorf("Expected newSlice256 length 256, got %d", len(newSlice256))
		}
		if len(newSlice512) != 512 {
			t.Errorf("Expected newSlice512 length 512, got %d", len(newSlice512))
		}
		if len(newSlice1024) != 1024 {
			t.Errorf("Expected newSlice1024 length 1024, got %d", len(newSlice1024))
		}
	})
}

func TestByteSliceAllocator_SizeClassSorting(t *testing.T) {
	t.Run("size classes are sorted correctly", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 1024, MinCount: 10},
				{Size: 256, MinCount: 5},
				{Size: 512, MinCount: 15},
				{Size: 128, MinCount: 20},
			},
		}
		allocator := NewByteSliceAllocator(config)

		// Verify pools are sorted by size
		if allocator.pools[0].Meta.(Meta).Size != 128 {
			t.Errorf("Expected first pool size 128, got %d", allocator.pools[0].Meta.(Meta).Size)
		}
		if allocator.pools[1].Meta.(Meta).Size != 256 {
			t.Errorf("Expected second pool size 256, got %d", allocator.pools[1].Meta.(Meta).Size)
		}
		if allocator.pools[2].Meta.(Meta).Size != 512 {
			t.Errorf("Expected third pool size 512, got %d", allocator.pools[2].Meta.(Meta).Size)
		}
		if allocator.pools[3].Meta.(Meta).Size != 1024 {
			t.Errorf("Expected fourth pool size 1024, got %d", allocator.pools[3].Meta.(Meta).Size)
		}

		// Test that Get returns from the correct pool
		slice := allocator.Get(200)
		if slice == nil {
			t.Error("Expected slice to be non-nil")
		}
		if len(slice) != 256 {
			t.Errorf("Expected slice length 256 (should use 256 pool, not 128), got %d", len(slice))
		}
	})
}

func TestByteSliceAllocator_EdgeCases(t *testing.T) {
	t.Run("single size class with exact match", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 512, MinCount: 1},
			},
		}
		allocator := NewByteSliceAllocator(config)

		slice := allocator.Get(512)
		if slice == nil {
			t.Error("Expected slice to be non-nil")
		}
		if len(slice) != 512 {
			t.Errorf("Expected slice length 512, got %d", len(slice))
		}

		allocator.Put(slice)

		// Get again after putting back
		slice2 := allocator.Get(512)
		if slice2 == nil {
			t.Error("Expected slice2 to be non-nil")
		}
		if len(slice2) != 512 {
			t.Errorf("Expected slice2 length 512, got %d", len(slice2))
		}
	})

	t.Run("duplicate size classes", func(t *testing.T) {
		config := ByteSliceAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 512, MinCount: 5},
				{Size: 512, MinCount: 10},
			},
		}
		allocator := NewByteSliceAllocator(config)

		if len(allocator.pools) != 2 {
			t.Errorf("Expected 2 pools, got %d", len(allocator.pools))
		}

		slice := allocator.Get(512)
		if slice == nil {
			t.Error("Expected slice to be non-nil")
		}
		if len(slice) != 512 {
			t.Errorf("Expected slice length 512, got %d", len(slice))
		}
	})
}
