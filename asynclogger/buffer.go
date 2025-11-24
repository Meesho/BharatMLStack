package asynclogger

import (
	"encoding/binary"
	"sync/atomic"
	"time"
)

// headerOffset is the number of bytes reserved at the start of each buffer for the shard header
const headerOffset = 8

// Buffer represents a single buffer for log entries with 512-byte alignment for Direct I/O
type Buffer struct {
	// data is the pre-allocated byte slice (512-byte aligned)
	// First 8 bytes are reserved for shard header (capacity + validDataBytes)
	data []byte

	// offset tracks the current write position (must use atomic operations)
	// Starts at headerOffset (8) to skip the reserved header space
	offset atomic.Int32

	// capacity is the maximum buffer size (includes the 8-byte header reservation)
	capacity int32

	// id is the buffer identifier for tracking and debugging
	id uint32

	// readyForFlush indicates the buffer is full and needs flushing
	readyForFlush atomic.Bool

	// writeCount tracks the number of writes to this buffer for statistics
	writeCount atomic.Int64

	// writesStarted tracks the number of writes that have reserved space (CAS succeeded)
	writesStarted atomic.Int64

	// writesCompleted tracks the number of writes that have completed copying data
	writesCompleted atomic.Int64
}

// NewBuffer creates a new buffer with the given capacity and ID
// The buffer is automatically aligned to 512-byte boundaries for Direct I/O
// First 8 bytes are reserved for shard header (capacity + validDataBytes)
func NewBuffer(capacity int, id uint32) *Buffer {
	// Reserve 8 bytes for header, then round total capacity to 512-byte alignment
	// This ensures the buffer is aligned and header space is reserved
	totalCapacity := capacity + 8 // Add header space
	alignedCap := alignSize(totalCapacity)

	buf := &Buffer{
		data:     allocAlignedBuffer(alignedCap),
		offset:   atomic.Int32{},
		capacity: int32(alignedCap),
		id:       id,
	}

	// Initialize offset to skip the 8-byte header reservation
	buf.offset.Store(8)

	return buf
}

// Write appends data to the buffer using atomic CAS for thread safety
// Prepends a 4-byte length prefix (little-endian) before the log data
// Returns the number of bytes written (including length prefix) and whether the buffer needs flushing
func (b *Buffer) Write(p []byte) (n int, needsFlush bool) {
	if len(p) == 0 {
		return 0, false
	}

	// Check if buffer is already full
	if b.readyForFlush.Load() {
		return 0, true
	}

	// Reserve space for: 4-byte length prefix + log data
	const lengthPrefixSize = 4
	totalSize := lengthPrefixSize + len(p)

	// Try to reserve space in the buffer (starting after the 8-byte header)
	currentOffset := b.offset.Load()
	newOffset := currentOffset + int32(totalSize)

	// Check if we have enough space (capacity includes the 8-byte header)
	// Use >= to handle the edge case where newOffset exactly equals capacity
	if newOffset >= b.capacity {
		b.readyForFlush.Store(true)
		return 0, true
	}

	// Try to atomically update the offset (CAS)
	if !b.offset.CompareAndSwap(currentOffset, newOffset) {
		// Another goroutine updated the offset, retry
		return b.Write(p)
	}

	// Write started: space reserved (atomic operations provide memory barriers)
	b.writesStarted.Add(1)

	// Write 4-byte length prefix (little-endian uint32)
	binary.LittleEndian.PutUint32(b.data[currentOffset:currentOffset+lengthPrefixSize], uint32(len(p)))

	// Copy log data after the length prefix
	copy(b.data[currentOffset+lengthPrefixSize:newOffset], p)

	// Write completed: copy finished (atomic operations provide memory barriers)
	b.writesCompleted.Add(1)

	// Increment write count for statistics
	b.writeCount.Add(1)

	// Check if buffer is now full or nearly full (within 10%)
	if newOffset >= b.capacity*9/10 {
		b.readyForFlush.Store(true)
		return totalSize, true
	}

	return totalSize, false
}

// GetData returns the entire buffer capacity (including invalid space at the end)
// This should only be called when the buffer is being flushed
// Waits for writesStarted == writesCompleted (all writes completed) or timeout expires
// Returns the full capacity slice and whether all writes completed (false if timeout occurred)
func (b *Buffer) GetData(timeout time.Duration) ([]byte, bool) {
	deadline := time.Now().Add(timeout)
	const checkInterval = 50 * time.Microsecond

	for time.Now().Before(deadline) {
		started := b.writesStarted.Load()
		completed := b.writesCompleted.Load()

		if started == completed {
			// All writes that started have completed
			// Return full capacity to handle invalid space at the end
			// Shard Header contains the capacity(4 bytes) and the valid data bytes(4 bytes)
			return b.data[:b.capacity], true
		}

		// Writes still in progress, wait a bit before retrying
		time.Sleep(checkInterval)
	}

	// Timeout expired: flush anyway (may contain incomplete last write)
	// Return full capacity to handle invalid space at the end
	return b.data[:b.capacity], false
}

// Reset clears the buffer for reuse
func (b *Buffer) Reset() {
	b.offset.Store(8) // Reset to header offset (skip 8-byte header reservation)
	b.readyForFlush.Store(false)
	b.writesStarted.Store(0)
	b.writesCompleted.Store(0)
}

// Offset returns the current write offset
// This includes the 8-byte header reservation, so actual data size is Offset() - 8
func (b *Buffer) Offset() int32 {
	return b.offset.Load()
}

// DataSize returns the size of actual data written (excluding header reservation)
// Returns 0 if offset is less than headerOffset (defensive check)
func (b *Buffer) DataSize() int32 {
	offset := b.offset.Load()
	if offset <= headerOffset {
		return 0
	}
	return offset - headerOffset
}

// Capacity returns the buffer capacity
func (b *Buffer) Capacity() int32 {
	return b.capacity
}

// ID returns the buffer identifier
func (b *Buffer) ID() uint32 {
	return b.id
}

// IsFull returns true if the buffer is marked for flushing
func (b *Buffer) IsFull() bool {
	return b.readyForFlush.Load()
}

// HasData returns true if the buffer contains any data
func (b *Buffer) HasData() bool {
	return b.offset.Load() > 8 // Data starts after the 8-byte header reservation
}

// WriteCount returns the total number of writes to this buffer
func (b *Buffer) WriteCount() int64 {
	return b.writeCount.Load()
}

// ResetWriteCount resets the write count to zero
func (b *Buffer) ResetWriteCount() {
	b.writeCount.Store(0)
}
