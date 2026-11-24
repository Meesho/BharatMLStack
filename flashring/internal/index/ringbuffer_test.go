package index

import (
	"testing"
)

func TestNewRingBuffer_ValidParams_PreallocatesCapacity(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	if rb.capacity != 4 {
		t.Fatalf("capacity = %d, want 4", rb.capacity)
	}
	if len(rb.buf) != 4 {
		t.Fatalf("len(buf) = %d, want 4", len(rb.buf))
	}
	if len(rb.hashTable) != 4 {
		t.Fatalf("len(hashTable) = %d, want 4", len(rb.hashTable))
	}
	if rb.head != 0 || rb.tail != 0 || rb.nextIndex != 0 {
		t.Fatalf("head/tail/nextIndex non-zero on empty RB: head=%d tail=%d next=%d",
			rb.head, rb.tail, rb.nextIndex)
	}
	if rb.wrapped {
		t.Fatal("wrapped = true on empty RB")
	}
}

func TestNewRingBuffer_InvalidParams_Panic(t *testing.T) {
	for _, tc := range []struct {
		name             string
		initial, maxSize int
	}{
		{"zero_initial", 0, 4},
		{"negative_initial", -1, 4},
		{"initial_gt_max", 8, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("NewRingBuffer(%d, %d) did not panic", tc.initial, tc.maxSize)
				}
			}()
			_ = NewRingBuffer(tc.initial, tc.maxSize)
		})
	}
}

func TestGetNextFreeSlot_AdvancesIndex(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	_, _, idx0, shouldDelete := rb.GetNextFreeSlot()
	if idx0 != 0 {
		t.Fatalf("first slot idx = %d, want 0", idx0)
	}
	if shouldDelete {
		t.Fatal("shouldDelete = true on first slot of empty RB")
	}
	if rb.nextIndex != 1 {
		t.Fatalf("nextIndex = %d, want 1", rb.nextIndex)
	}

	_, _, idx1, _ := rb.GetNextFreeSlot()
	if idx1 != 1 {
		t.Fatalf("second slot idx = %d, want 1", idx1)
	}
}

func TestGetNextFreeSlot_WrapsAndFlagsDelete(t *testing.T) {
	rb := NewRingBuffer(1, 3)

	// Fill three slots — third call advances nextIndex to 0 == head, which
	// flips wrapped + shouldDelete=true.
	rb.GetNextFreeSlot()
	rb.GetNextFreeSlot()
	_, _, _, shouldDelete := rb.GetNextFreeSlot()

	if !rb.wrapped {
		t.Fatal("wrapped = false after filling RB")
	}
	if !shouldDelete {
		t.Fatal("shouldDelete = false when wrapping")
	}
	if rb.nextIndex != 0 {
		t.Fatalf("nextIndex = %d, want 0 after wrap", rb.nextIndex)
	}
}

func TestNextAddNeedsDelete_Tracks(t *testing.T) {
	rb := NewRingBuffer(1, 2)

	if rb.NextAddNeedsDelete() {
		t.Fatal("NextAddNeedsDelete = true on empty RB")
	}

	rb.GetNextFreeSlot()
	rb.GetNextFreeSlot() // wraps → nextIndex back to 0 == head

	if !rb.NextAddNeedsDelete() {
		t.Fatal("NextAddNeedsDelete = false after wrap")
	}
}

func TestGet_WithinRange_ReturnsEntry(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	entry, hnp, ok := rb.Get(2)
	if !ok {
		t.Fatal("ok = false for valid index")
	}
	if entry == nil || hnp == nil {
		t.Fatal("nil pointers returned")
	}
}

func TestGet_OutOfRange_ReturnsFalse(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	entry, hnp, ok := rb.Get(99)
	if ok {
		t.Fatal("ok = true for out-of-range index")
	}
	if entry != nil || hnp != nil {
		t.Fatalf("expected nil pointers, got entry=%v hnp=%v", entry, hnp)
	}
}

func TestDelete_AdvancesHeadAndReturnsNeighbours(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	// Stamp identifiable bytes so we can distinguish slots.
	rb.buf[0][0] = 0xAA
	rb.buf[1][0] = 0xBB

	deleted, _, deletedIdx, next := rb.Delete()
	if deletedIdx != 0 {
		t.Fatalf("deletedIdx = %d, want 0", deletedIdx)
	}
	if deleted[0] != 0xAA {
		t.Fatalf("deleted[0] = %x, want 0xAA", deleted[0])
	}
	if next[0] != 0xBB {
		t.Fatalf("next[0] = %x, want 0xBB", next[0])
	}
	if rb.head != 1 {
		t.Fatalf("head = %d, want 1", rb.head)
	}
}

func TestTailIndex_ReturnsNextIndex(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	if got := rb.TailIndex(); got != 0 {
		t.Fatalf("TailIndex on empty = %d, want 0", got)
	}

	rb.GetNextFreeSlot()
	if got := rb.TailIndex(); got != 1 {
		t.Fatalf("TailIndex after one Put = %d, want 1", got)
	}
}

func TestActiveEntries_TracksGrowthAndShrinkage(t *testing.T) {
	rb := NewRingBuffer(2, 4)

	if got := rb.ActiveEntries(); got != 0 {
		t.Fatalf("ActiveEntries on empty = %d, want 0", got)
	}

	rb.GetNextFreeSlot()
	rb.GetNextFreeSlot()
	if got := rb.ActiveEntries(); got != 2 {
		t.Fatalf("ActiveEntries after 2 inserts = %d, want 2", got)
	}

	rb.Delete()
	if got := rb.ActiveEntries(); got != 1 {
		t.Fatalf("ActiveEntries after 1 delete = %d, want 1", got)
	}
}

// TestRingBuffer_Get_BoundsGuard locks down the fix for the off-by-one in
// RingBuffer.Get (PR #2377). Prior code used `index > rb.capacity` (strict),
// allowing index == capacity to slip past the guard and panic on slice
// access. Guard is now `< 0 || >= capacity`.
func TestRingBuffer_Get_BoundsGuard(t *testing.T) {
	rb := NewRingBuffer(4, 4)
	tests := []struct {
		name   string
		index  int
		wantOK bool
	}{
		{"negative", -1, false},
		{"zero", 0, true},
		{"middle", 2, true},
		{"last_valid", 3, true},
		{"equals_capacity", 4, false},
		{"way_over", 10000, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Get(%d) panicked: %v", tc.index, r)
				}
			}()
			entry, hnp, ok := rb.Get(tc.index)
			if ok != tc.wantOK {
				t.Fatalf("Get(%d) ok = %v, want %v", tc.index, ok, tc.wantOK)
			}
			if !tc.wantOK && (entry != nil || hnp != nil) {
				t.Fatalf("Get(%d) returned non-nil pointers with ok=false: entry=%v hnp=%v", tc.index, entry, hnp)
			}
		})
	}
}
