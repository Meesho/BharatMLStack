package indices

// Entry represents a 32-byte value. Adjust fields as needed.
type Entry [32]byte

// RingBuffer is a fixed-size circular queue that wraps around when full.
// It maintains a sliding window of the most recent entries. Add returns an
// absolute index which can be used with Get.
type RingBuffer struct {
	buf       []Entry
	head      int
	tail      int
	size      int
	nextIndex int
	capacity  int // Fixed capacity (initial = max)
	wrapped   bool
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
		buf:      make([]Entry, capacity),
		capacity: capacity,
		wrapped:  false,
	}
}

// Add inserts e into the buffer and returns its absolute index. When the buffer
// is full it wraps around and overwrites the oldest entry.
func (rb *RingBuffer) Add(e *Entry) int {
	// Store the entry at current tail position
	rb.buf[rb.nextIndex] = *e
	idx := rb.nextIndex
	rb.nextIndex = (rb.nextIndex + 1) % rb.capacity
	if rb.nextIndex == rb.head {
		rb.head = (rb.head + 1) % rb.capacity
	}

	return idx
}

func (rb *RingBuffer) NextAddNeedsDelete() bool {
	return rb.nextIndex == rb.head && rb.wrapped
}

func (rb *RingBuffer) GetEntry() (*Entry, int, bool) {
	idx := rb.nextIndex
	rb.nextIndex = (rb.nextIndex + 1) % rb.capacity
	shouldDelete := false
	if rb.nextIndex == rb.head {
		// rb.head = (rb.head + 1) % rb.capacity
		rb.wrapped = true
		shouldDelete = true

	}

	return &rb.buf[idx], idx, shouldDelete
}

// Get retrieves an entry by its absolute index. The boolean return is false if
// the index is out of range (either overwritten or not yet added).
func (rb *RingBuffer) Get(index int) (*Entry, bool) {
	// Calculate the valid window based on current state
	if index > rb.capacity {
		return nil, false
	}
	return &rb.buf[index], true
}

// Delete removes the oldest entry from the buffer if it is not empty.
// For a fixed-size ring buffer, this only decreases size if not at capacity.
func (rb *RingBuffer) Delete() (*Entry, *Entry) {
	deleted := rb.buf[rb.head]
	rb.head = (rb.head + 1) % rb.capacity
	return &deleted, &rb.buf[rb.head]
}

// TailIndex returns the absolute index that will be assigned to the next Add.
func (rb *RingBuffer) TailIndex() int {
	return rb.nextIndex
}
