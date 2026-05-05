package index

// Entry represents a 16-byte index entry.
type Entry [16]byte

// HashNextPrev stores the dual hash (for collision detection) and linked-list
// pointers for chaining entries that share the same hash-lo bucket.
type HashNextPrev [3]uint64

// RingBuffer is a fixed-size circular queue. It maintains a sliding window
// of the most recent entries and wraps around when full, overwriting the oldest.
type RingBuffer struct {
	buf       []Entry
	hashTable []HashNextPrev
	head      int
	tail      int
	size      int
	nextIndex int
	capacity  int
	wrapped   bool
}

func NewRingBuffer(initial, max int) *RingBuffer {
	if initial <= 0 || initial > max {
		panic("invalid capacity")
	}
	capacity := max
	return &RingBuffer{
		buf:       make([]Entry, capacity),
		hashTable: make([]HashNextPrev, capacity),
		capacity:  capacity,
	}
}

func (rb *RingBuffer) NextAddNeedsDelete() bool {
	return rb.nextIndex == rb.head && rb.wrapped
}

func (rb *RingBuffer) GetNextFreeSlot() (*Entry, *HashNextPrev, int, bool) {
	idx := rb.nextIndex
	rb.nextIndex = (rb.nextIndex + 1) % rb.capacity
	shouldDelete := false
	if rb.nextIndex == rb.head {
		rb.wrapped = true
		shouldDelete = true
	}
	return &rb.buf[idx], &rb.hashTable[idx], idx, shouldDelete
}

func (rb *RingBuffer) Get(index int) (*Entry, *HashNextPrev, bool) {
	if index > rb.capacity {
		return nil, nil, false
	}
	return &rb.buf[index], &rb.hashTable[index], true
}

func (rb *RingBuffer) Delete() (*Entry, *HashNextPrev, int, *Entry) {
	deletedIdx := rb.head
	deleted := rb.buf[rb.head]
	deletedHashNextPrev := rb.hashTable[rb.head]
	rb.head = (rb.head + 1) % rb.capacity
	return &deleted, &deletedHashNextPrev, deletedIdx, &rb.buf[rb.head]
}

func (rb *RingBuffer) TailIndex() int {
	return rb.nextIndex
}

func (rb *RingBuffer) ActiveEntries() int {
	return (rb.nextIndex - rb.head + rb.capacity) % rb.capacity
}
