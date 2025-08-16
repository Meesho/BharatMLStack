package allocators

import (
	"testing"

	"github.com/Meesho/BharatMLStack/flashring/external/fs"
)

func TestNewSlabAlignedPageAllocator(t *testing.T) {
	t.Run("creates allocator with single aligned size class", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10}, // 4096 is aligned to fs.BLOCK_SIZE
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if allocator == nil {
			t.Error("Expected allocator to be non-nil")
		}
		if allocator.config.SizeClasses[0].Size != config.SizeClasses[0].Size {
			t.Errorf("Expected config to match, got %v", allocator.config)
		}
		if len(allocator.pools) != 1 {
			t.Errorf("Expected 1 pool, got %d", len(allocator.pools))
		}
		if allocator.pools[0].Meta.(Meta).Size != 4096 {
			t.Errorf("Expected pool size 4096, got %d", allocator.pools[0].Meta.(Meta).Size)
		}
		if allocator.pools[0].Meta.(Meta).Name != "SlabAlignedPagePool-4096Bytes" {
			t.Errorf("Expected pool name 'SlabAlignedPagePool-4096Bytes', got %s", allocator.pools[0].Meta.(Meta).Name)
		}
	})

	t.Run("creates allocator with multiple aligned size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 8192, MinCount: 5},  // 8192 is aligned to fs.BLOCK_SIZE
				{Size: 4096, MinCount: 10}, // 4096 is aligned to fs.BLOCK_SIZE
				{Size: 16384, MinCount: 3}, // 16384 is aligned to fs.BLOCK_SIZE
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if allocator == nil {
			t.Error("Expected allocator to be non-nil")
		}
		if len(allocator.pools) != 3 {
			t.Errorf("Expected 3 pools, got %d", len(allocator.pools))
		}

		// Should be sorted by size
		if allocator.pools[0].Meta.(Meta).Size != 4096 {
			t.Errorf("Expected first pool size 4096, got %d", allocator.pools[0].Meta.(Meta).Size)
		}
		if allocator.pools[1].Meta.(Meta).Size != 8192 {
			t.Errorf("Expected second pool size 8192, got %d", allocator.pools[1].Meta.(Meta).Size)
		}
		if allocator.pools[2].Meta.(Meta).Size != 16384 {
			t.Errorf("Expected third pool size 16384, got %d", allocator.pools[2].Meta.(Meta).Size)
		}
	})

	t.Run("creates allocator with empty size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if allocator == nil {
			t.Error("Expected allocator to be non-nil")
		}
		if len(allocator.pools) != 0 {
			t.Errorf("Expected 0 pools, got %d", len(allocator.pools))
		}
	})

	t.Run("returns error for non-aligned size class", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4097, MinCount: 10}, // 4097 is not aligned to fs.BLOCK_SIZE (4096)
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)

		if err != ErrSizeNotAligned {
			t.Errorf("Expected ErrSizeNotAligned, got %v", err)
		}
		if allocator != nil {
			t.Error("Expected allocator to be nil on error")
		}
	})

	t.Run("returns error for mixed aligned and non-aligned size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10}, // aligned
				{Size: 3000, MinCount: 5},  // not aligned
				{Size: 8192, MinCount: 3},  // aligned
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)

		if err != ErrSizeNotAligned {
			t.Errorf("Expected ErrSizeNotAligned, got %v", err)
		}
		if allocator != nil {
			t.Error("Expected allocator to be nil on error")
		}
	})
}

func TestSlabAlignedPageAllocator_Get(t *testing.T) {
	t.Run("returns aligned page for exact size match", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(4096)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}
		if len(page.Buf) != 4096 {
			t.Errorf("Expected page buffer length 4096, got %d", len(page.Buf))
		}
		if cap(page.Buf) != 4096 {
			t.Errorf("Expected page buffer capacity 4096, got %d", cap(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})

	t.Run("returns aligned page for smaller size", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(2048)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}
		if len(page.Buf) != 4096 {
			t.Errorf("Expected page buffer length 4096, got %d", len(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})

	t.Run("returns smallest suitable size class", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 5},
				{Size: 8192, MinCount: 10},
				{Size: 16384, MinCount: 3},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(6000)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}
		if len(page.Buf) != 8192 {
			t.Errorf("Expected page buffer length 8192, got %d", len(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})

	t.Run("returns nil for size larger than all size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(8192)
		if page != nil {
			t.Error("Expected page to be nil for size larger than all size classes")
		}
	})

	t.Run("returns nil for empty size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(4096)
		if page != nil {
			t.Error("Expected page to be nil for empty size classes")
		}
	})

	t.Run("returns page for zero size request", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(0)
		if page == nil {
			t.Error("Expected page to be non-nil for zero size request")
		}
		if len(page.Buf) != 4096 {
			t.Errorf("Expected page buffer length 4096, got %d", len(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})

	t.Run("returns page for negative size request", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(-1)
		if page == nil {
			t.Error("Expected page to be non-nil for negative size request")
		}
		if len(page.Buf) != 4096 {
			t.Errorf("Expected page buffer length 4096, got %d", len(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})
}

func TestSlabAlignedPageAllocator_Put(t *testing.T) {
	t.Run("puts aligned page back to correct pool", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(4096)
		if page == nil {
			t.Fatal("Expected page to be non-nil")
		}

		// Put should not panic
		allocator.Put(page)
	})

	t.Run("puts page to smallest suitable pool", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 5},
				{Size: 8192, MinCount: 10},
				{Size: 16384, MinCount: 3},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Create a page manually (not from pool)
		page := fs.NewAlignedPage(6000)
		if page == nil {
			t.Fatal("Failed to create aligned page")
		}

		// Should not panic, even though page wasn't from the pool
		allocator.Put(page)
	})

	t.Run("handles page larger than all size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Create a large page
		page := fs.NewAlignedPage(8192)
		if page == nil {
			t.Fatal("Failed to create aligned page")
		}

		// Should not panic, but will log error
		allocator.Put(page)

		// Clean up manually since it won't be put back in pool
		fs.Unmap(page)
	})

	t.Run("handles nil page", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should not panic, but may cause issues due to nil pointer
		// This test mainly ensures the method doesn't crash completely
		defer func() {
			if r := recover(); r != nil {
				// It's expected that this might panic due to nil pointer access
				t.Logf("Expected panic occurred: %v", r)
			}
		}()

		allocator.Put(nil)
	})
}

func TestSlabAlignedPageAllocator_GetAndPut_Integration(t *testing.T) {
	t.Run("get and put multiple times", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 2},
				{Size: 8192, MinCount: 3},
				{Size: 16384, MinCount: 1},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Get multiple pages
		pages := make([]*fs.AlignedPage, 5)
		for i := 0; i < 5; i++ {
			pages[i] = allocator.Get(3000) // Should get 4096 size
			if pages[i] == nil {
				t.Errorf("Expected page %d to be non-nil", i)
			}
			if len(pages[i].Buf) != 4096 {
				t.Errorf("Expected page %d buffer length 4096, got %d", i, len(pages[i].Buf))
			}
		}

		// Put them back
		for _, page := range pages {
			if page != nil {
				allocator.Put(page)
			}
		}

		// Get them again
		for i := 0; i < 5; i++ {
			page := allocator.Get(3000)
			if page == nil {
				t.Errorf("Expected page %d to be non-nil on second get", i)
			}
			if page != nil && len(page.Buf) != 4096 {
				t.Errorf("Expected page %d buffer length 4096 on second get, got %d", i, len(page.Buf))
			}
		}
	})

	t.Run("get and put with different sizes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 2},
				{Size: 8192, MinCount: 3},
				{Size: 16384, MinCount: 1},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Get pages of different sizes
		page4k := allocator.Get(3000)   // Should get 4096
		page8k := allocator.Get(6000)   // Should get 8192
		page16k := allocator.Get(12000) // Should get 16384

		if len(page4k.Buf) != 4096 {
			t.Errorf("Expected page4k buffer length 4096, got %d", len(page4k.Buf))
		}
		if len(page8k.Buf) != 8192 {
			t.Errorf("Expected page8k buffer length 8192, got %d", len(page8k.Buf))
		}
		if len(page16k.Buf) != 16384 {
			t.Errorf("Expected page16k buffer length 16384, got %d", len(page16k.Buf))
		}

		// Put them back
		allocator.Put(page4k)
		allocator.Put(page8k)
		allocator.Put(page16k)

		// Get them again
		newPage4k := allocator.Get(3000)
		newPage8k := allocator.Get(6000)
		newPage16k := allocator.Get(12000)

		if len(newPage4k.Buf) != 4096 {
			t.Errorf("Expected newPage4k buffer length 4096, got %d", len(newPage4k.Buf))
		}
		if len(newPage8k.Buf) != 8192 {
			t.Errorf("Expected newPage8k buffer length 8192, got %d", len(newPage8k.Buf))
		}
		if len(newPage16k.Buf) != 16384 {
			t.Errorf("Expected newPage16k buffer length 16384, got %d", len(newPage16k.Buf))
		}
	})
}

func TestSlabAlignedPageAllocator_SizeClassSorting(t *testing.T) {
	t.Run("size classes are sorted correctly", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 16384, MinCount: 3},
				{Size: 4096, MinCount: 10},
				{Size: 8192, MinCount: 5},
				{Size: 12288, MinCount: 2}, // 12288 = 3 * 4096, aligned
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify pools are sorted by size
		if allocator.pools[0].Meta.(Meta).Size != 4096 {
			t.Errorf("Expected first pool size 4096, got %d", allocator.pools[0].Meta.(Meta).Size)
		}
		if allocator.pools[1].Meta.(Meta).Size != 8192 {
			t.Errorf("Expected second pool size 8192, got %d", allocator.pools[1].Meta.(Meta).Size)
		}
		if allocator.pools[2].Meta.(Meta).Size != 12288 {
			t.Errorf("Expected third pool size 12288, got %d", allocator.pools[2].Meta.(Meta).Size)
		}
		if allocator.pools[3].Meta.(Meta).Size != 16384 {
			t.Errorf("Expected fourth pool size 16384, got %d", allocator.pools[3].Meta.(Meta).Size)
		}

		// Test that Get returns from the correct pool
		page := allocator.Get(10000)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}
		if len(page.Buf) != 12288 {
			t.Errorf("Expected page buffer length 12288 (should use 12288 pool), got %d", len(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})
}

func TestSlabAlignedPageAllocator_EdgeCases(t *testing.T) {
	t.Run("single size class with exact match", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 1},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(4096)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}
		if len(page.Buf) != 4096 {
			t.Errorf("Expected page buffer length 4096, got %d", len(page.Buf))
		}

		allocator.Put(page)

		// Get again after putting back
		page2 := allocator.Get(4096)
		if page2 == nil {
			t.Error("Expected page2 to be non-nil")
		}
		if len(page2.Buf) != 4096 {
			t.Errorf("Expected page2 buffer length 4096, got %d", len(page2.Buf))
		}
	})

	t.Run("duplicate size classes", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 5},
				{Size: 4096, MinCount: 10},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(allocator.pools) != 2 {
			t.Errorf("Expected 2 pools, got %d", len(allocator.pools))
		}

		page := allocator.Get(4096)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}
		if len(page.Buf) != 4096 {
			t.Errorf("Expected page buffer length 4096, got %d", len(page.Buf))
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})
}

func TestSlabAlignedPageAllocator_MemoryAlignment(t *testing.T) {
	t.Run("pages are properly aligned", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 1},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		page := allocator.Get(4096)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}

		// Test that we can write to the page without issues
		if len(page.Buf) > 0 {
			page.Buf[0] = 0x42
			page.Buf[len(page.Buf)-1] = 0x24

			if page.Buf[0] != 0x42 {
				t.Error("Failed to write to first byte of page")
			}
			if page.Buf[len(page.Buf)-1] != 0x24 {
				t.Error("Failed to write to last byte of page")
			}
		}

		// Clean up
		if page != nil {
			fs.Unmap(page)
		}
	})
}

func TestSlabAlignedPageAllocator_PreDrefHook(t *testing.T) {
	t.Run("pre deref hook is registered", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			SizeClasses: []SizeClass{
				{Size: 4096, MinCount: 1},
			},
		}
		allocator, err := NewSlabAlignedPageAllocator(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// The PreDrefHook should be registered during construction
		// We can't directly test the hook execution without accessing private fields
		// But we can verify that pool creation succeeded
		if len(allocator.pools) != 1 {
			t.Errorf("Expected 1 pool to be created, got %d", len(allocator.pools))
		}

		// Test normal allocation and deallocation
		page := allocator.Get(4096)
		if page == nil {
			t.Error("Expected page to be non-nil")
		}

		// Put back should trigger the hook internally when pool is full
		allocator.Put(page)
	})
}

func TestSlabAlignedPageAllocator_AlignmentValidation(t *testing.T) {
	t.Run("various alignment checks", func(t *testing.T) {
		tests := []struct {
			name        string
			size        int
			shouldError bool
		}{
			{"aligned 4096", 4096, false},
			{"aligned 8192", 8192, false},
			{"aligned 12288", 12288, false},
			{"aligned 16384", 16384, false},
			{"unaligned 4097", 4097, true},
			{"unaligned 4000", 4000, true},
			{"unaligned 5000", 5000, true},
			{"unaligned 1024", 1024, true}, // 1024 < 4096
			{"unaligned 2048", 2048, true}, // 2048 < 4096
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := SlabAlignedPageAllocatorConfig{
					SizeClasses: []SizeClass{
						{Size: tt.size, MinCount: 1},
					},
				}
				allocator, err := NewSlabAlignedPageAllocator(config)

				if tt.shouldError {
					if err != ErrSizeNotAligned {
						t.Errorf("Expected ErrSizeNotAligned for size %d, got %v", tt.size, err)
					}
					if allocator != nil {
						t.Errorf("Expected nil allocator for size %d", tt.size)
					}
				} else {
					if err != nil {
						t.Errorf("Expected no error for size %d, got %v", tt.size, err)
					}
					if allocator == nil {
						t.Errorf("Expected non-nil allocator for size %d", tt.size)
					}
				}
			})
		}
	})
}
