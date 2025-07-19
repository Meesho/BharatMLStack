package indices

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestNewHBM24L4(t *testing.T) {
	t.Run("LazyAllocation", func(t *testing.T) {
		hbm := NewHBM24L4(true)
		if hbm == nil {
			t.Fatal("Expected non-nil HBM24L4")
		}
		if hbm.root == nil {
			t.Fatal("Expected non-nil root")
		}
		if !hbm.lazyAlloc {
			t.Error("Expected lazyAlloc to be true")
		}

		// With lazy allocation, nodes should be nil initially
		for i := 0; i < 64; i++ {
			if hbm.root.Nodes[i] != nil {
				t.Errorf("Expected node %d to be nil with lazy allocation", i)
			}
		}
	})

	t.Run("EagerAllocation", func(t *testing.T) {
		hbm := NewHBM24L4(false)
		if hbm == nil {
			t.Fatal("Expected non-nil HBM24L4")
		}
		if hbm.root == nil {
			t.Fatal("Expected non-nil root")
		}
		if hbm.lazyAlloc {
			t.Error("Expected lazyAlloc to be false")
		}

		// With eager allocation, all nodes should be pre-allocated
		for i := 0; i < 64; i++ {
			if hbm.root.Nodes[i] == nil {
				t.Errorf("Expected node %d to be non-nil with eager allocation", i)
			}
			for j := 0; j < 64; j++ {
				if hbm.root.Nodes[i].Nodes[j] == nil {
					t.Errorf("Expected level1 node [%d][%d] to be non-nil with eager allocation", i, j)
				}
			}
		}
	})
}

func TestExtractIndices(t *testing.T) {
	tests := []struct {
		hash        uint32
		expectedI0  uint8
		expectedI1  uint8
		expectedI2  uint8
		expectedI3  uint8
		description string
	}{
		{0x000000, 0, 0, 0, 0, "all zeros"},
		{0xFFFFFF, 63, 63, 63, 63, "all ones (24-bit)"},
		{0xFC0000, 63, 0, 0, 0, "max i0"},
		{0x03F000, 0, 63, 0, 0, "max i1"},
		{0x000FC0, 0, 0, 63, 0, "max i2"},
		{0x00003F, 0, 0, 0, 63, "max i3"},
		{0x123456, 0x04, 0x23, 0x11, 0x16, "mixed values"},
		{0xABCDEF, 0x2A, 0x3C, 0x37, 0x2F, "mixed high values"},
		{0x1000000, 0, 0, 0, 0, "bit 25 set (should be masked out)"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			i0, i1, i2, i3 := extractIndices24(test.hash)
			if i0 != test.expectedI0 {
				t.Errorf("i0: expected %d, got %d", test.expectedI0, i0)
			}
			if i1 != test.expectedI1 {
				t.Errorf("i1: expected %d, got %d", test.expectedI1, i1)
			}
			if i2 != test.expectedI2 {
				t.Errorf("i2: expected %d, got %d", test.expectedI2, i2)
			}
			if i3 != test.expectedI3 {
				t.Errorf("i3: expected %d, got %d", test.expectedI3, i3)
			}
		})
	}
}

func TestHBM24L4_BasicOperations(t *testing.T) {
	testCases := []bool{true, false} // Test both lazy and eager allocation

	for _, lazyAlloc := range testCases {
		t.Run(func() string {
			if lazyAlloc {
				return "LazyAllocation"
			}
			return "EagerAllocation"
		}(), func(t *testing.T) {
			hbm := NewHBM24L4(lazyAlloc)

			// Test empty bitmap
			if hbm.Get(0) {
				t.Error("Expected Get(0) to return false for empty bitmap")
			}
			if hbm.Get(0xFFFFFF) {
				t.Error("Expected Get(0xFFFFFF) to return false for empty bitmap")
			}

			// Test single put/get
			hash := uint32(12345)
			hbm.Put(hash)
			if !hbm.Get(hash) {
				t.Errorf("Expected Get(%d) to return true after Put", hash)
			}

			// Test that other values are still false
			if hbm.Get(hash + 1) {
				t.Errorf("Expected Get(%d) to return false", hash+1)
			}

			// Test remove
			hbm.Remove(hash)
			if hbm.Get(hash) {
				t.Errorf("Expected Get(%d) to return false after Remove", hash)
			}
		})
	}
}

func TestHBM24L4_MultipleValues(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Test multiple puts
	hashes := []uint32{0, 1, 100, 1000, 10000, 100000, 1000000, 0xFFFFFF}

	// Put all hashes
	for _, hash := range hashes {
		hbm.Put(hash)
	}

	// Verify all hashes are present
	for _, hash := range hashes {
		if !hbm.Get(hash) {
			t.Errorf("Expected Get(%d) to return true", hash)
		}
	}

	// Remove some hashes
	toRemove := []uint32{1, 1000, 1000000}
	for _, hash := range toRemove {
		hbm.Remove(hash)
	}

	// Verify removed hashes are gone
	for _, hash := range toRemove {
		if hbm.Get(hash) {
			t.Errorf("Expected Get(%d) to return false after removal", hash)
		}
	}

	// Verify remaining hashes are still present
	remaining := []uint32{0, 100, 10000, 100000, 0xFFFFFF}
	for _, hash := range remaining {
		if !hbm.Get(hash) {
			t.Errorf("Expected Get(%d) to return true after partial removal", hash)
		}
	}
}

func TestHBM24L4_BoundaryConditions(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Test boundary values
	boundaries := []uint32{
		0,        // minimum
		0xFFFFFF, // maximum 24-bit value
		0x3F,     // max single index
		0xFC0,    // i2 boundary
		0x3F000,  // i1 boundary
		0xFC0000, // i0 boundary
	}

	for _, hash := range boundaries {
		t.Run(func() string { return fmt.Sprintf("boundary_%x", hash) }(), func(t *testing.T) {
			// Initially should be false
			if hbm.Get(hash) {
				t.Errorf("Expected Get(0x%x) to be false initially", hash)
			}

			// Put and verify
			hbm.Put(hash)
			if !hbm.Get(hash) {
				t.Errorf("Expected Get(0x%x) to be true after Put", hash)
			}

			// Remove and verify
			hbm.Remove(hash)
			if hbm.Get(hash) {
				t.Errorf("Expected Get(0x%x) to be false after Remove", hash)
			}
		})
	}
}

func TestHBM24L4_SummaryBits(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Test that summary bits are set correctly
	hash := uint32(0x123456) // This will map to specific indices
	i0, i1, i2, i3 := extractIndices24(hash)

	hbm.Put(hash)

	// Check that summary bits are set at each level
	if hbm.root.Sum&bitIndex[i0] == 0 {
		t.Error("Expected level 2 summary bit to be set")
	}

	l1 := hbm.root.Nodes[i0]
	if l1.Sum&bitIndex[i1] == 0 {
		t.Error("Expected level 1 summary bit to be set")
	}

	l0 := l1.Nodes[i1]
	if l0.Sum&bitIndex[i2] == 0 {
		t.Error("Expected level 0 summary bit to be set")
	}

	// Check the actual bit
	if l0.Leafs[i2]&bitIndex[i3] == 0 {
		t.Error("Expected leaf bit to be set")
	}

	// Remove and check summary bits are cleared appropriately
	hbm.Remove(hash)

	// The leaf bit should be cleared
	if l0.Leafs[i2]&bitIndex[i3] != 0 {
		t.Error("Expected leaf bit to be cleared after removal")
	}
}

func TestHBM24L4_DuplicatePuts(t *testing.T) {
	hbm := NewHBM24L4(true)

	hash := uint32(12345)

	// Put the same hash multiple times
	for i := 0; i < 10; i++ {
		hbm.Put(hash)
		if !hbm.Get(hash) {
			t.Errorf("Expected Get(%d) to return true after Put iteration %d", hash, i)
		}
	}

	// Should still be true after multiple puts
	if !hbm.Get(hash) {
		t.Errorf("Expected Get(%d) to return true after multiple Puts", hash)
	}

	// Single remove should clear it
	hbm.Remove(hash)
	if hbm.Get(hash) {
		t.Errorf("Expected Get(%d) to return false after single Remove", hash)
	}
}

func TestHBM24L4_RemoveNonExistent(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Remove from empty bitmap should not panic
	hbm.Remove(12345)

	// Add a value, then remove a different value
	hbm.Put(100)
	hbm.Remove(200)

	// Original value should still be there
	if !hbm.Get(100) {
		t.Error("Expected Get(100) to return true after removing non-existent value")
	}
}

func TestHBM24L4_HighBitsIgnored(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Test that bits above 24 are ignored
	hash1 := uint32(0x123456)
	hash2 := uint32(0x1123456)  // Same lower 24 bits, but with bit 24 set
	hash3 := uint32(0xFF123456) // Same lower 24 bits, but with high bits set

	hbm.Put(hash1)

	// All should be considered the same due to 24-bit masking
	if !hbm.Get(hash2) {
		t.Error("Expected hash with bit 24 set to be considered same as original")
	}
	if !hbm.Get(hash3) {
		t.Error("Expected hash with high bits set to be considered same as original")
	}

	// Removing any of them should remove the entry
	hbm.Remove(hash2)
	if hbm.Get(hash1) {
		t.Error("Expected original hash to be removed when removing equivalent hash")
	}
}

func TestHBM24L4_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	hbm := NewHBM24L4(true)
	rand.Seed(time.Now().UnixNano())

	const numOperations = 10000
	hashes := make(map[uint32]bool)

	// Random puts and gets
	for i := 0; i < numOperations; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		operation := rand.Float32()

		if operation < 0.7 { // 70% chance to put
			hbm.Put(hash)
			hashes[hash] = true
		} else if operation < 0.85 { // 15% chance to remove (85% - 70%)
			hbm.Remove(hash)
			delete(hashes, hash)
		}
		// 15% chance to do nothing (just query)

		// Verify consistency every 1000 operations
		if i%1000 == 999 {
			for h := range hashes {
				if !hbm.Get(h) {
					t.Errorf("Expected hash %d to be present", h)
				}
			}
		}
	}

	// Final consistency check
	for hash := range hashes {
		if !hbm.Get(hash) {
			t.Errorf("Final check: Expected hash %d to be present", hash)
		}
	}
}

func TestHBM24L4_AllBitsInWord(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Test setting all bits in a single Level0 word
	baseHash := uint32(0x123400) // Same i0, i1, i2, varying i3

	// Set all 64 bits in the word
	for i3 := uint8(0); i3 < 64; i3++ {
		hash := baseHash | uint32(i3)
		hbm.Put(hash)
	}

	// Verify all bits are set
	for i3 := uint8(0); i3 < 64; i3++ {
		hash := baseHash | uint32(i3)
		if !hbm.Get(hash) {
			t.Errorf("Expected bit %d to be set", i3)
		}
	}

	// Remove all bits
	for i3 := uint8(0); i3 < 64; i3++ {
		hash := baseHash | uint32(i3)
		hbm.Remove(hash)
	}

	// Verify all bits are cleared and summary bits are updated
	for i3 := uint8(0); i3 < 64; i3++ {
		hash := baseHash | uint32(i3)
		if hbm.Get(hash) {
			t.Errorf("Expected bit %d to be cleared", i3)
		}
	}
}

func TestHBM24L4_RemoveLogic(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Test removing the only element should clear all summary bits
	hash := uint32(0x123456)
	i0, i1, i2, i3 := extractIndices24(hash)

	// Put the hash
	hbm.Put(hash)

	// Verify it's there
	if !hbm.Get(hash) {
		t.Fatal("Hash should be present after Put")
	}

	// Check summary bits are set
	l2 := hbm.root
	l1 := l2.Nodes[i0]
	l0 := l1.Nodes[i1]

	if l2.Sum&bitIndex[i0] == 0 {
		t.Error("L2 summary bit should be set")
	}
	if l1.Sum&bitIndex[i1] == 0 {
		t.Error("L1 summary bit should be set")
	}
	if l0.Sum&bitIndex[i2] == 0 {
		t.Error("L0 summary bit should be set")
	}
	if l0.Leafs[i2]&bitIndex[i3] == 0 {
		t.Error("Leaf bit should be set")
	}

	// Remove the hash
	hbm.Remove(hash)

	// Verify it's gone
	if hbm.Get(hash) {
		t.Error("Hash should not be present after Remove")
	}

	// Check that leaf bit is cleared
	if l0.Leafs[i2]&bitIndex[i3] != 0 {
		t.Error("Leaf bit should be cleared")
	}

	// If the leaf word is now empty, summary bit should be cleared
	if l0.Leafs[i2] == 0 {
		if l0.Sum&bitIndex[i2] != 0 {
			t.Error("L0 summary bit should be cleared when leaf word is empty")
		}

		// If L0 is now empty, L1 summary should be cleared
		if l0.Sum == 0 {
			if l1.Sum&bitIndex[i1] != 0 {
				t.Error("L1 summary bit should be cleared when L0 is empty")
			}

			// If L1 is now empty, L2 summary should be cleared
			if l1.Sum == 0 {
				if l2.Sum&bitIndex[i0] != 0 {
					t.Error("L2 summary bit should be cleared when L1 is empty")
				}
			}
		}
	}
}

func TestHBM24L4_RemovePartialWord(t *testing.T) {
	hbm := NewHBM24L4(true)

	// Put two hashes that map to the same leaf word but different bits
	baseHash := uint32(0x123400) // Same i0, i1, i2
	hash1 := baseHash | 5        // i3 = 5
	hash2 := baseHash | 10       // i3 = 10

	hbm.Put(hash1)
	hbm.Put(hash2)

	// Both should be present
	if !hbm.Get(hash1) {
		t.Error("Hash1 should be present")
	}
	if !hbm.Get(hash2) {
		t.Error("Hash2 should be present")
	}

	// Remove one hash
	hbm.Remove(hash1)

	// First should be gone, second should remain
	if hbm.Get(hash1) {
		t.Error("Hash1 should be removed")
	}
	if !hbm.Get(hash2) {
		t.Error("Hash2 should still be present")
	}

	// Summary bits should still be set because hash2 is still there
	i0, i1, i2, _ := extractIndices24(hash2)
	l2 := hbm.root
	l1 := l2.Nodes[i0]
	l0 := l1.Nodes[i1]

	if l2.Sum&bitIndex[i0] == 0 {
		t.Error("L2 summary bit should still be set")
	}
	if l1.Sum&bitIndex[i1] == 0 {
		t.Error("L1 summary bit should still be set")
	}
	if l0.Sum&bitIndex[i2] == 0 {
		t.Error("L0 summary bit should still be set")
	}
}

// Benchmark tests
func BenchmarkHBM24L4_Put(b *testing.B) {
	hbm := NewHBM24L4(true)
	hashes := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		hashes[i] = uint32(i) & _24BIT_MASK
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Put(hashes[i])
	}
}

func BenchmarkHBM24L4_Get(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Pre-populate with some data
	for i := 0; i < 10000; i++ {
		hbm.Put(uint32(i))
	}

	hashes := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		hashes[i] = uint32(i % 20000) // Some hits, some misses
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Get(hashes[i])
	}
}

func BenchmarkHBM24L4_Remove(b *testing.B) {
	hbm := NewHBM24L4(true)
	hashes := make([]uint32, b.N)

	// Pre-populate
	for i := 0; i < b.N; i++ {
		hashes[i] = uint32(i) & _24BIT_MASK
		hbm.Put(hashes[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Remove(hashes[i])
	}
}

func BenchmarkHBM24L4_LazyVsEager(b *testing.B) {
	b.Run("Lazy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(true)
			hbm.Put(uint32(i))
			hbm.Get(uint32(i))
		}
	})

	b.Run("Eager", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(false)
			hbm.Put(uint32(i))
			hbm.Get(uint32(i))
		}
	})
}
