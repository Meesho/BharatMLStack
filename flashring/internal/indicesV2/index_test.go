package indicesv2

import (
	"fmt"
	"testing"
)

func TestIndexAddRbMax(t *testing.T) {
	loadByteOrder()

	// Use equal initial and max capacity for the fixed-size ring buffer.
	rbMax := 1000_000
	rbInitial := rbMax
	hashBits := 16
	idx := NewIndex(hashBits, rbInitial, rbMax, 1)

	// Insert exactly rbMax distinct keys
	for i := 0; i < rbMax; i++ {
		key := fmt.Sprintf("k%d", i)
		length := uint16(100 + i)
		ttlMinutes := uint16(120) // ensure no expiry during test
		memID := uint32(1000 + i)
		offset := uint32(2000 + i)
		idx.Put(key, length, ttlMinutes, memID, offset)
	}

	// All keys should be present in the reverse map
	if got := len(idx.rm); got != rbMax {
		t.Fatalf("expected %d keys in index map, got %d", rbMax, got)
	}

	// After filling to capacity, next add should require delete (ring wrapped)
	if !idx.rb.NextAddNeedsDelete() {
		t.Fatalf("expected ring buffer to report NextAddNeedsDelete == true after %d inserts", rbMax)
	}

	// Verify we can Get every inserted key and fields match
	for i := 0; i < rbMax; i++ {
		key := fmt.Sprintf("k%d", i)
		expLength := uint16(100 + i)
		expMemID := uint32(1000 + i)
		expOffset := uint32(2000 + i)

		length, _, _, _, memID, offset, status := idx.Get(key)
		if status != StatusOK {
			t.Fatalf("Get(%q) status = %v, want %v", key, status, StatusOK)
		}
		if length != expLength {
			t.Fatalf("Get(%q) length = %d, want %d", key, length, expLength)
		}
		if memID != expMemID {
			t.Fatalf("Get(%q) memID = %d, want %d", key, memID, expMemID)
		}
		if offset != expOffset {
			t.Fatalf("Get(%q) offset = %d, want %d", key, offset, expOffset)
		}
	}
}

func TestIndexDeleteAndGet(t *testing.T) {
	loadByteOrder()

	// Keep this small and fast
	rbMax := 99
	rbInitial := rbMax
	hashBits := 16
	idx := NewIndex(hashBits, rbInitial, rbMax, 1)

	// Insert exactly rbMax distinct keys in order
	for i := 0; i < 33; i++ {
		key := fmt.Sprintf("k%d", i)
		length := uint16(100 + i)
		ttlMinutes := uint16(120)
		memID := uint32(1)
		offset := uint32(2000 + i)
		idx.Put(key, length, ttlMinutes, memID, offset)
	}

	for i := 33; i < 66; i++ {
		key := fmt.Sprintf("k%d", i)
		length := uint16(100 + i)
		ttlMinutes := uint16(120)
		memID := uint32(2)
		offset := uint32(2000 + i)
		idx.Put(key, length, ttlMinutes, memID, offset)
	}
	for i := 66; i < 99; i++ {
		key := fmt.Sprintf("k%d", i)
		length := uint16(100 + i)
		ttlMinutes := uint16(120)
		memID := uint32(3)
		offset := uint32(2000 + i)
		idx.Put(key, length, ttlMinutes, memID, offset)
	}

	if len(idx.rm) != rbMax {
		t.Fatalf("expected %d keys after fill, got %d", rbMax, len(idx.rm))
	}

	// Ensure buffer is in the full state (next add would need delete)
	if !idx.rb.NextAddNeedsDelete() {
		t.Fatalf("expected NextAddNeedsDelete() to be true after fill")
	}

	for i := 0; i < 99; i++ {
		key := fmt.Sprintf("k%d", i)
		_, _, _, _, _, _, st := idx.Get(key)
		if st != StatusOK {
			t.Fatalf("Get(%q) status=%v, want %v", key, st, StatusOK)
		}
	}
	// Delete oldest entries one-by-one and verify via Get
	toDelete := 33
	idx.Delete(toDelete)

	if len(idx.rm) != rbMax-toDelete {
		t.Fatalf("expected map size %d after deletes, got %d", rbMax-toDelete, len(idx.rm))
	}

	for i := 0; i < toDelete; i++ {
		key := fmt.Sprintf("k%d", i)
		_, _, _, _, _, _, st := idx.Get(key)
		if st != StatusNotFound {
			t.Fatalf("Get(%q) status=%v, want %v", key, st, StatusNotFound)
		}
	}

	for i := toDelete; i < 99; i++ {
		key := fmt.Sprintf("k%d", i)
		_, _, _, _, _, _, st := idx.Get(key)
		if st != StatusOK {
			t.Fatalf("Get(%q) status=%v, want %v", key, st, StatusOK)
		}
	}
}
