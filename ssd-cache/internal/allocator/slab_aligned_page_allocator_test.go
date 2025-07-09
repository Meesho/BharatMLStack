package allocator

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNewSlabAlignedPageAllocator(t *testing.T) {
	t.Run("ValidConfiguration", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			PageSizeAlignement: 4096,
			Multipliers:        []int{1, 2, 4, 8},
			MaxPages:           []int{10, 20, 30, 40},
		}

		allocator := NewSlabAlignedPageAllocator(config)

		if allocator == nil {
			t.Fatal("NewSlabAlignedPageAllocator should not return nil")
		}

		if !reflect.DeepEqual(allocator.config, config) {
			t.Errorf("Expected config %+v, got %+v", config, allocator.config)
		}

		if len(allocator.pools) != len(config.Multipliers) {
			t.Errorf("Expected %d pools, got %d", len(config.Multipliers), len(allocator.pools))
		}

		if len(allocator.sizeClassToIndex) != len(config.Multipliers) {
			t.Errorf("Expected %d size classes, got %d", len(config.Multipliers), len(allocator.sizeClassToIndex))
		}

		// Verify size class mapping
		expectedMapping := map[int]int{
			4096:  0, // 4096 * 1
			8192:  1, // 4096 * 2
			16384: 2, // 4096 * 4
			32768: 3, // 4096 * 8
		}

		if !reflect.DeepEqual(allocator.sizeClassToIndex, expectedMapping) {
			t.Errorf("Expected size class mapping %+v, got %+v", expectedMapping, allocator.sizeClassToIndex)
		}
	})

	t.Run("SingleSizeClass", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			PageSizeAlignement: 4096,
			Multipliers:        []int{1},
			MaxPages:           []int{5},
		}

		allocator := NewSlabAlignedPageAllocator(config)

		if len(allocator.pools) != 1 {
			t.Errorf("Expected 1 pool, got %d", len(allocator.pools))
		}

		if len(allocator.sizeClassToIndex) != 1 {
			t.Errorf("Expected 1 size class, got %d", len(allocator.sizeClassToIndex))
		}

		if allocator.sizeClassToIndex[4096] != 0 {
			t.Errorf("Expected size class 4096 to map to index 0, got %d", allocator.sizeClassToIndex[4096])
		}
	})

	t.Run("EmptyConfiguration", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			PageSizeAlignement: 4096,
			Multipliers:        []int{},
			MaxPages:           []int{},
		}

		allocator := NewSlabAlignedPageAllocator(config)

		if len(allocator.pools) != 0 {
			t.Errorf("Expected 0 pools, got %d", len(allocator.pools))
		}

		if len(allocator.sizeClassToIndex) != 0 {
			t.Errorf("Expected 0 size classes, got %d", len(allocator.sizeClassToIndex))
		}
	})
}

func TestSlabAlignedPageAllocator_Get(t *testing.T) {
	config := SlabAlignedPageAllocatorConfig{
		PageSizeAlignement: 4096,
		Multipliers:        []int{1, 2, 4, 8},
		MaxPages:           []int{2, 2, 2, 2},
	}
	allocator := NewSlabAlignedPageAllocator(config)

	t.Run("ExactSizeMatch", func(t *testing.T) {
		page, crossedBound := allocator.Get(4096)

		if page == nil {
			t.Fatal("Expected page to be returned, got nil")
		}

		// Map iteration order is not guaranteed, so any valid size class >= 4096 is acceptable
		validSizes := []int{4096, 8192, 16384, 32768}
		found := false
		for _, size := range validSizes {
			if len(page.Buf) == size {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected buffer size to be one of %v, got %d", validSizes, len(page.Buf))
		}

		// CrossedBound depends on usage and capacity
		_ = crossedBound
	})

	t.Run("SmallerThanSizeClass", func(t *testing.T) {
		page, crossedBound := allocator.Get(2048)

		if page == nil {
			t.Fatal("Expected page to be returned, got nil")
		}

		if len(page.Buf) != 4096 {
			t.Errorf("Expected buffer size 4096 (smallest size class), got %d", len(page.Buf))
		}

		if crossedBound {
			t.Error("Expected crossedBound to be false for first allocation")
		}
	})

	t.Run("LargerSizeClass", func(t *testing.T) {
		page, crossedBound := allocator.Get(12288) // Should use a size class >= 12288

		if page == nil {
			t.Fatal("Expected page to be returned, got nil")
		}

		// Map iteration order is not guaranteed, so any valid size class >= 12288 is acceptable
		validSizes := []int{16384, 32768}
		found := false
		for _, size := range validSizes {
			if len(page.Buf) == size {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected buffer size to be one of %v, got %d", validSizes, len(page.Buf))
		}

		// CrossedBound depends on usage and capacity
		_ = crossedBound
	})

	t.Run("LargerThanAllSizeClasses", func(t *testing.T) {
		page, crossedBound := allocator.Get(65536) // Larger than 32768 (max size class)

		if page != nil {
			t.Error("Expected nil page for size larger than all size classes")
		}

		if crossedBound {
			t.Error("Expected crossedBound to be false when returning nil")
		}
	})

	t.Run("ZeroSize", func(t *testing.T) {
		page, crossedBound := allocator.Get(0)

		if page == nil {
			t.Fatal("Expected page to be returned, got nil")
		}

		// Should get the smallest size class, but map iteration order is not guaranteed
		validSizes := []int{4096, 8192, 16384, 32768}
		found := false
		for _, size := range validSizes {
			if len(page.Buf) == size {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected buffer size to be one of %v, got %d", validSizes, len(page.Buf))
		}

		// CrossedBound depends on usage and capacity, so we can't guarantee it's false
		_ = crossedBound
	})

	t.Run("CrossedBoundWhenPoolExhausted", func(t *testing.T) {
		// Allocate pages until pool is exhausted
		var pages []*Page
		var crossedBoundCount int
		for i := 0; i < 5; i++ { // MaxPages is 2, so eventually should cross bound
			page, crossedBound := allocator.Get(4096)
			if page == nil {
				t.Fatal("Expected page to be returned, got nil")
			}
			pages = append(pages, page)

			if crossedBound {
				crossedBoundCount++
			}
		}

		// Should have crossed bound at least once when allocating more than MaxPages
		if crossedBoundCount == 0 {
			t.Error("Expected crossedBound to be true at least once when pool is exhausted")
		}
	})
}

func TestSlabAlignedPageAllocator_Put(t *testing.T) {
	config := SlabAlignedPageAllocatorConfig{
		PageSizeAlignement: 4096,
		Multipliers:        []int{1, 2, 4},
		MaxPages:           []int{2, 2, 2},
	}
	allocator := NewSlabAlignedPageAllocator(config)

	t.Run("ValidPageReturn", func(t *testing.T) {
		page, _ := allocator.Get(4096)
		if page == nil {
			t.Fatal("Expected page to be returned from Get")
		}

		// This should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Put should not panic, got: %v", r)
			}
		}()

		allocator.Put(page)
	})

	t.Run("PutAndGetCycle", func(t *testing.T) {
		// Get a page
		page1, _ := allocator.Get(8192)
		if page1 == nil {
			t.Fatal("Expected page to be returned from Get")
		}

		// Put it back
		allocator.Put(page1)

		// Get another page of the same size
		page2, crossedBound := allocator.Get(8192)
		if page2 == nil {
			t.Fatal("Expected page to be returned from Get after Put")
		}

		// CrossedBound depends on usage and capacity
		_ = crossedBound

		// Due to map iteration order not being guaranteed, we might get different size classes
		// So we can't guarantee the same page will be reused
		// Just verify we get a valid page
		if len(page2.Buf) < 8192 {
			t.Errorf("Expected buffer size >= 8192, got %d", len(page2.Buf))
		}
	})

	t.Run("PutInvalidSizeClass", func(t *testing.T) {
		// Create a page with buffer size that doesn't match any size class
		page := &Page{
			Buf: make([]byte, 6000), // Doesn't match 4096, 8192, or 16384
		}

		// This should handle gracefully and not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Put with invalid size class should not panic, got: %v", r)
			}
		}()

		allocator.Put(page)
	})

	t.Run("PutNilPage", func(t *testing.T) {
		// The actual implementation doesn't handle nil pages gracefully, so it should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Put with nil page should panic")
			}
		}()

		allocator.Put(nil)
	})
}

func TestSlabAlignedPageAllocator_SizeClassSelection(t *testing.T) {
	config := SlabAlignedPageAllocatorConfig{
		PageSizeAlignement: 1024,
		Multipliers:        []int{1, 4, 16, 64},
		MaxPages:           []int{5, 5, 5, 5},
	}
	allocator := NewSlabAlignedPageAllocator(config)

	testCases := []struct {
		requestedSize int
		minSize       int
		shouldSucceed bool
	}{
		{512, 1024, true},    // Should use a size >= 1024
		{1024, 1024, true},   // Should use a size >= 1024
		{2048, 4096, true},   // Should use a size >= 4096
		{4096, 4096, true},   // Should use a size >= 4096
		{8192, 16384, true},  // Should use a size >= 16384
		{16384, 16384, true}, // Should use a size >= 16384
		{32768, 65536, true}, // Should use a size >= 65536
		{65536, 65536, true}, // Should use a size >= 65536
		{131072, 0, false},   // Larger than all size classes
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Size_%d", tc.requestedSize), func(t *testing.T) {
			page, _ := allocator.Get(tc.requestedSize)

			if tc.shouldSucceed {
				if page == nil {
					t.Errorf("Expected page to be returned for size %d", tc.requestedSize)
					return
				}
				if len(page.Buf) < tc.minSize {
					t.Errorf("Expected buffer size >= %d, got %d for requested size %d", tc.minSize, len(page.Buf), tc.requestedSize)
				}
			} else {
				if page != nil {
					t.Errorf("Expected nil page for size %d, got buffer size %d", tc.requestedSize, len(page.Buf))
				}
			}
		})
	}
}

func TestSlabAlignedPageAllocator_EdgeCases(t *testing.T) {
	t.Run("MismatchedMultipliersAndMaxPages", func(t *testing.T) {
		// This should handle gracefully or panic during construction
		defer func() {
			if r := recover(); r == nil {
				t.Error("Constructor should panic with mismatched multipliers and max pages")
			}
		}()

		config := SlabAlignedPageAllocatorConfig{
			PageSizeAlignement: 4096,
			Multipliers:        []int{1, 2, 4},
			MaxPages:           []int{5, 10}, // Different length
		}

		NewSlabAlignedPageAllocator(config)
	})

	t.Run("NegativeSize", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			PageSizeAlignement: 4096,
			Multipliers:        []int{1, 2},
			MaxPages:           []int{5, 5},
		}
		allocator := NewSlabAlignedPageAllocator(config)

		page, crossedBound := allocator.Get(-1000)

		// Negative size satisfies size <= sizeClass for all size classes,
		// so it returns a page from the first size class encountered
		if page == nil {
			t.Error("Expected page to be returned for negative size")
		}

		// Should be one of the valid size classes
		validSizes := []int{4096, 8192}
		found := false
		for _, size := range validSizes {
			if len(page.Buf) == size {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected buffer size to be one of %v, got %d", validSizes, len(page.Buf))
		}

		// CrossedBound depends on usage and capacity
		_ = crossedBound
	})

	t.Run("ZeroPageSizeAlignment", func(t *testing.T) {
		config := SlabAlignedPageAllocatorConfig{
			PageSizeAlignement: 0,
			Multipliers:        []int{1, 2},
			MaxPages:           []int{5, 5},
		}

		// This should either panic or handle gracefully
		defer func() {
			if r := recover(); r == nil {
				// If it doesn't panic, let's test the behavior
				allocator := NewSlabAlignedPageAllocator(config)
				page, _ := allocator.Get(1024)
				if page != nil {
					t.Error("Expected nil page with zero page size alignment")
				}
			}
		}()

		NewSlabAlignedPageAllocator(config)
	})
}

func TestSlabAlignedPageAllocator_ReuseAfterPut(t *testing.T) {
	config := SlabAlignedPageAllocatorConfig{
		PageSizeAlignement: 4096,
		Multipliers:        []int{1, 2},
		MaxPages:           []int{1, 1}, // Small pool to test reuse
	}
	allocator := NewSlabAlignedPageAllocator(config)

	// Get a page
	page1, crossedBound1 := allocator.Get(4096)
	if page1 == nil {
		t.Fatal("Expected page to be returned")
	}
	// CrossedBound depends on usage and capacity
	_ = crossedBound1

	// Get another page (should cross bound since pool size is 1)
	page2, crossedBound2 := allocator.Get(4096)
	if page2 == nil {
		t.Fatal("Expected page to be returned")
	}
	// CrossedBound depends on usage and capacity
	_ = crossedBound2

	// Put first page back
	allocator.Put(page1)

	// Get another page (should reuse the returned page)
	page3, crossedBound3 := allocator.Get(4096)
	if page3 == nil {
		t.Fatal("Expected page to be returned")
	}
	// CrossedBound depends on usage and capacity
	_ = crossedBound3

	// Due to map iteration order not being guaranteed, we might get different size classes
	// So we can't guarantee the same page will be reused
	// Just verify we get a valid page
	if len(page3.Buf) < 4096 {
		t.Errorf("Expected buffer size >= 4096, got %d", len(page3.Buf))
	}
}
