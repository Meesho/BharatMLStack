package indicesv2

import "sync/atomic"

// Entry represents a 32-byte value. Adjust fields as needed.
type Entry [16]byte
type HashNextPrev [3]uint64

// RingBuffer is a fixed-size circular queue that wraps around when full.
// It maintains a sliding window of the most recent entries. Add returns an
// absolute index which can be used with Get.
type RingBuffer struct {
	nextIndex int64  // atomic; must be first for 64-bit alignment on 32-bit platforms
	wrapped   uint32 // atomic; 0 = false, 1 = true (one-time transition)
	buf       []Entry
	hashTable []HashNextPrev
	head      int
	tail      int
	size      int
	capacity  int
}

// NewRingBuffer creates a ring buffer with the given initial and maximum
// capacity. Since we use a fixed-size buffer, initial and max should be the same.
func NewRingBuffer(initial, max int) *RingBuffer {
	if initial <= 0 || initial > max {
		panic("invalid capacity")
	}
	// Use max capacity for fixed-size buffer (initial = max in practice)
	capacity := max
	return &RingBuffer{
		buf:       make([]Entry, capacity),
		hashTable: make([]HashNextPrev, capacity),
		capacity:  capacity,
	}
}

// Add inserts e into the buffer and returns its absolute index. When the buffer
// is full it wraps around and overwrites the oldest entry.
func (rb *RingBuffer) Add(e *Entry) int {
	raw := atomic.AddInt64(&rb.nextIndex, 1) - 1
	idx := int(raw) % rb.capacity
	rb.buf[idx] = *e
	nextIdx := int(raw+1) % rb.capacity
	if nextIdx == rb.head {
		rb.head = (rb.head + 1) % rb.capacity
	}
	return idx
}

func (rb *RingBuffer) NextAddNeedsDelete() bool {
	nextIdx := int(atomic.LoadInt64(&rb.nextIndex)) % rb.capacity
	return nextIdx == rb.head && atomic.LoadUint32(&rb.wrapped) == 1
}

func (rb *RingBuffer) GetNextFreeSlot() (*Entry, *HashNextPrev, int, bool) {
	raw := atomic.AddInt64(&rb.nextIndex, 1) - 1
	idx := int(raw) % rb.capacity

	shouldDelete := false
	nextIdx := int(raw+1) % rb.capacity
	if nextIdx == rb.head {
		atomic.StoreUint32(&rb.wrapped, 1)
		shouldDelete = true
	}
	return &rb.buf[idx], &rb.hashTable[idx], idx, shouldDelete
}

// Get retrieves an entry by its absolute index. The boolean return is false if
// the index is out of range (either overwritten or not yet added).
func (rb *RingBuffer) Get(index int) (*Entry, *HashNextPrev, bool) {
	// Calculate the valid window based on current state
	if index > rb.capacity {
		return nil, nil, false
	}
	return &rb.buf[index], &rb.hashTable[index], true
}

// Delete removes the oldest entry from the buffer if it is not empty.
// For a fixed-size ring buffer, this only decreases size if not at capacity.
func (rb *RingBuffer) Delete() (*Entry, *HashNextPrev, int, *Entry) {
	deletedIdx := rb.head
	deleted := rb.buf[rb.head]
	deletedHashNextPrev := rb.hashTable[rb.head]
	rb.head = (rb.head + 1) % rb.capacity
	return &deleted, &deletedHashNextPrev, deletedIdx, &rb.buf[rb.head]
}

// TailIndex returns the slot index that will be assigned to the next Add.
func (rb *RingBuffer) TailIndex() int {
	return int(atomic.LoadInt64(&rb.nextIndex)) % rb.capacity
}

func (rb *RingBuffer) ActiveEntries() int {
	next := int(atomic.LoadInt64(&rb.nextIndex)) % rb.capacity
	return (next - rb.head + rb.capacity) % rb.capacity
}
