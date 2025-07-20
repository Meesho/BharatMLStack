package indices

// Entry represents a 32-byte value. Adjust fields as needed.
type Entry [32]byte

// RingBuffer is a dynamically growing circular queue. It grows until
// maxCapacity and then starts overwriting the oldest entries. Add returns an
// absolute index which can be used with Get.
type RingBuffer struct {
	buf         []Entry
	head        int
	tail        int
	size        int
	nextIndex   int
	maxCapacity int
}

// NewRingBuffer creates a ring buffer with the given initial and maximum
// capacity. initial must be >0 and <= max.
func NewRingBuffer(initial, max int) *RingBuffer {
	if initial <= 0 || initial > max {
		panic("invalid capacity")
	}
	return &RingBuffer{
		buf:         make([]Entry, initial),
		maxCapacity: max,
	}
}

// Add inserts e into the buffer and returns its absolute index. When the buffer
// reaches maxCapacity it wraps around and overwrites the oldest entry.
func (rb *RingBuffer) Add(e *Entry) int {
	if rb.size == len(rb.buf) {
		if len(rb.buf) < rb.maxCapacity {
			// Grow to the lesser of double the size or maxCapacity.
			newCap := len(rb.buf) * 2
			if newCap > rb.maxCapacity {
				newCap = rb.maxCapacity
			}
			newBuf := make([]Entry, newCap)
			copy(newBuf, rb.toSlice())
			rb.buf = newBuf
			rb.head = 0
			rb.tail = rb.size
		} else {
			// At max capacity: overwrite oldest.
			rb.buf[rb.tail] = *e
			rb.head = (rb.head + 1) % len(rb.buf)
			idx := rb.nextIndex
			rb.nextIndex++
			rb.tail = (rb.tail + 1) % len(rb.buf)
			return idx
		}
	}
	rb.buf[rb.tail] = *e
	idx := rb.nextIndex
	rb.nextIndex++
	rb.tail = (rb.tail + 1) % len(rb.buf)
	rb.size++
	return idx
}

// Get retrieves an entry by its absolute index. The boolean return is false if
// the index is out of range (either overwritten or not yet added).
func (rb *RingBuffer) Get(index int) (*Entry, bool) {
	if index < rb.nextIndex-rb.size || index >= rb.nextIndex {
		return nil, false
	}
	pos := index % len(rb.buf)
	return &rb.buf[pos], true
}

// toSlice returns the current contents in order from oldest to newest.
func (rb *RingBuffer) toSlice() []Entry {
	if rb.size == 0 {
		return nil
	}
	if rb.head < rb.tail {
		return rb.buf[rb.head:rb.tail]
	}
	res := make([]Entry, rb.size)
	n := copy(res, rb.buf[rb.head:])
	copy(res[n:], rb.buf[:rb.tail])
	return res
}

// Delete removes the oldest entry from the buffer if it is not empty.
// It returns true when an element was removed.
func (rb *RingBuffer) Delete() (*Entry, *Entry, bool) {
	if rb.size == 0 {
		return nil, nil, false
	}
	deleted := rb.buf[rb.head]
	rb.head = (rb.head + 1) % len(rb.buf)
	rb.size--
	return &deleted, &rb.buf[rb.head], true
}

// TailIndex returns the absolute index that will be assigned to the next Add.
func (rb *RingBuffer) TailIndex() int {
	return rb.nextIndex
}
