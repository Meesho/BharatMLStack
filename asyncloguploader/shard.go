package asyncloguploader

import (
	"encoding/binary"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// headerOffset is the number of bytes reserved at the start of each buffer for the shard header
const headerOffset = 8

// Buffer represents a single buffer with its own state and operations
type Buffer struct {
	// mmap'd memory
	data []byte

	// Write state
	offset   atomic.Int32 // Current write offset
	inflight atomic.Int64 // Number of concurrent writes in progress

	// Buffer metadata
	capacity int32
	shardID  uint32

	// Mutex for flush operations (shared with Shard)
	mu *sync.Mutex
}

// GetData returns the buffer data, waiting for all inflight writes to complete
// Returns the full capacity slice and whether all writes completed
func (b *Buffer) GetData(timeout time.Duration) ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.data == nil {
		return nil, false
	}

	// Wait for all inflight writes to complete
	deadline := time.Now().Add(timeout)
	const checkInterval = 50 * time.Microsecond

	for time.Now().Before(deadline) {
		if b.inflight.Load() == 0 {
			// All writes have completed
			return b.data[:b.capacity], true
		}

		// Writes still in progress, yield CPU
		runtime.Gosched()
		time.Sleep(checkInterval)
	}

	// Timeout expired: flush anyway (may contain incomplete last write)
	return b.data[:b.capacity], false
}

// Reset clears the buffer after flush
func (b *Buffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.offset.Store(headerOffset)
	b.inflight.Store(0)
}

// HasData returns true if the buffer has data (offset > headerOffset)
func (b *Buffer) HasData() bool {
	return b.offset.Load() > headerOffset
}

// Offset returns the current write offset
func (b *Buffer) Offset() int32 {
	return b.offset.Load()
}

// Capacity returns the buffer capacity
func (b *Buffer) Capacity() int32 {
	return b.capacity
}

// ShardID returns the shard ID this buffer belongs to
func (b *Buffer) ShardID() uint32 {
	return b.shardID
}

// Shard represents a single shard with double buffer
type Shard struct {
	// Double buffer: two Buffer instances
	bufferA *Buffer
	bufferB *Buffer

	// Active buffer pointer (atomically swapped)
	activeBuffer atomic.Pointer[*Buffer]

	// Capacity (same for both buffers, includes headerOffset)
	capacity int32

	// Mutex for flush operations (shared with buffers)
	mu sync.Mutex

	// Shard identifier
	id uint32

	// Swap coordination
	swapping      atomic.Bool
	readyForFlush atomic.Bool
	swapSemaphore chan struct{} // Per-shard semaphore for swap coordination (buffer size 1)

	// Flush channel for pushing buffer references
	flushChan chan<- *Buffer

	// Cleanup functions for mmap (called on Close)
	cleanupA func()
	cleanupB func()
}

// NewShard creates a new shard with double buffer using anonymous mmap
func NewShard(capacity int, id uint32, flushChan chan<- *Buffer) (*Shard, error) {
	alignedCap := alignSize(capacity)

	// Allocate bufferA via anonymous mmap
	dataA, cleanupA, err := allocMmapBuffer(alignedCap)
	if err != nil {
		return nil, err
	}

	// Allocate bufferB via anonymous mmap
	dataB, cleanupB, err := allocMmapBuffer(alignedCap)
	if err != nil {
		cleanupA()
		unix.Munmap(dataA)
		return nil, err
	}

	s := &Shard{
		capacity:      int32(alignedCap),
		id:            id,
		cleanupA:      cleanupA,
		cleanupB:      cleanupB,
		swapSemaphore: make(chan struct{}, 1), // Per-shard semaphore (buffer size 1)
		flushChan:     flushChan,
	}

	// Create Buffer instances
	s.bufferA = &Buffer{
		data:     dataA,
		capacity: int32(alignedCap),
		shardID:  id,
		mu:       &s.mu,
	}
	s.bufferA.offset.Store(headerOffset)

	s.bufferB = &Buffer{
		data:     dataB,
		capacity: int32(alignedCap),
		shardID:  id,
		mu:       &s.mu,
	}
	s.bufferB.offset.Store(headerOffset)

	// Set bufferA as initial active buffer
	s.activeBuffer.Store(&s.bufferA)

	// Set finalizer on Shard struct (not on individual buffers)
	// This ensures buffers are only unmapped when Shard is garbage collected
	runtime.SetFinalizer(s, func(shard *Shard) {
		if shard != nil {
			// Acquire mutex to ensure no concurrent access during cleanup
			shard.mu.Lock()
			defer shard.mu.Unlock()

			// Unmap buffers if they still exist
			if shard.bufferA != nil && len(shard.bufferA.data) > 0 {
				unix.Munmap(shard.bufferA.data)
				shard.bufferA.data = nil
			}
			if shard.bufferB != nil && len(shard.bufferB.data) > 0 {
				unix.Munmap(shard.bufferB.data)
				shard.bufferB.data = nil
			}
		}
	})

	return s, nil
}

// allocMmapBuffer allocates a buffer using anonymous mmap
// Returns the buffer, cleanup function, and error
func allocMmapBuffer(size int) ([]byte, func(), error) {
	// Round up to page size alignment
	alignedSize := alignSize(size)

	// Create anonymous private mapping
	data, err := unix.Mmap(
		-1, 0,
		alignedSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_PRIVATE|unix.MAP_ANONYMOUS,
	)
	if err != nil {
		return nil, nil, err
	}

	// Cleanup function keeps buffer alive during use
	cleanup := func() {
		runtime.KeepAlive(data)
		// Don't unmap during normal operation - reuse buffers
	}

	// NOTE: Finalizer is NOT set here - it's set on the Shard struct instead
	// This prevents premature unmapping while the buffer is still in use

	return data, cleanup, nil
}

// alignSize rounds up size to the nearest alignment boundary (4096 bytes)
func alignSize(size int) int {
	const alignmentSize = 4096
	return ((size + alignmentSize - 1) / alignmentSize) * alignmentSize
}

// Write writes data to the active buffer (lock-free hot path)
// Prepends a 4-byte length prefix (little-endian) before the log data
// Returns the number of bytes written (including length prefix) and whether the buffer needs flushing
func (s *Shard) Write(p []byte) (n int, needsFlush bool) {
	if len(p) == 0 {
		return 0, false
	}

	// Get active buffer
	activeBufPtr := s.activeBuffer.Load()
	if activeBufPtr == nil {
		// Active buffer is nil - shard may be in invalid state
		return 0, true
	}
	activeBuf := *activeBufPtr

	// Reserve space for: 4-byte length prefix + log data
	const lengthPrefixSize = 4
	totalSize := lengthPrefixSize + len(p)

	// Try to reserve space in the buffer (starting after the 8-byte header)
	currentOffset := activeBuf.offset.Load()
	newOffset := currentOffset + int32(totalSize)

	// Check if we have enough space in the active buffer
	// IMPORTANT: Check buffer space BEFORE checking readyForFlush
	// This allows writes to the new active buffer after swap, even if readyForFlush is still true
	if newOffset >= s.capacity {
		// Active buffer is full - swap so it becomes inactive and can be flushed
		s.trySwap()
		return 0, true
	}

	// If readyForFlush is true but active buffer has space, it means:
	// - A swap just happened and the inactive buffer is being flushed
	// - The new active buffer is empty and ready for writes
	// - We should allow writes to proceed
	// Note: readyForFlush only prevents writes when BOTH buffers are full

	// Try to atomically update the offset (CAS)
	if !activeBuf.offset.CompareAndSwap(currentOffset, newOffset) {
		// Another goroutine updated the offset, retry
		return s.Write(p)
	}

	// Increment inflight counter
	activeBuf.inflight.Add(1)

	// Write 4-byte length prefix (little-endian uint32)
	binary.LittleEndian.PutUint32(activeBuf.data[currentOffset:currentOffset+lengthPrefixSize], uint32(len(p)))

	// Use copy() for data copy - Go's copy() is already highly optimized and safe
	copy(activeBuf.data[currentOffset+lengthPrefixSize:newOffset], p)

	// Decrement inflight counter: write completed
	activeBuf.inflight.Add(-1)

	// Check if buffer is now full or nearly full (within 10%)
	if newOffset >= s.capacity*9/10 {
		// CRITICAL: Force swap immediately so inactive buffer has the data
		s.trySwap()
		return totalSize, true
	}

	return totalSize, false
}

// trySwap attempts to swap the active buffer (CAS-protected)
// Pushes the now-inactive buffer to the flush channel
func (s *Shard) trySwap() {
	// Check if already swapping
	if !s.swapping.CompareAndSwap(false, true) {
		return // Another goroutine is swapping
	}
	defer s.swapping.Store(false)

	// Get current active buffer
	currentBufPtr := s.activeBuffer.Load()
	if currentBufPtr == nil {
		return
	}
	currentBuf := *currentBufPtr

	// Determine next buffer
	var nextBuf *Buffer
	if currentBuf == s.bufferA {
		nextBuf = s.bufferB
	} else {
		nextBuf = s.bufferA
	}

	// Atomically swap active buffer
	nextBufPtr := &nextBuf
	if !s.activeBuffer.CompareAndSwap(&currentBuf, nextBufPtr) {
		// Swap failed, another goroutine beat us
		return
	}

	// Push the now-inactive buffer to flush channel
	// This buffer was just swapped out and needs to be flushed
	if s.flushChan != nil {
		select {
		case s.flushChan <- currentBuf:
			// Successfully queued for flush
		default:
			// Channel full, will be picked up by periodic flush
		}
	}

	// Mark shard as ready for flush
	s.readyForFlush.Store(true)
}

// GetInactiveBuffer returns the currently inactive buffer
func (s *Shard) GetInactiveBuffer() *Buffer {
	activeBufPtr := s.activeBuffer.Load()
	if activeBufPtr == nil {
		return s.bufferB // Default to B if no active buffer
	}
	activeBuf := *activeBufPtr
	if activeBuf == s.bufferA {
		return s.bufferB
	}
	return s.bufferA
}

// GetBuffersWithData returns all buffers that have data
func (s *Shard) GetBuffersWithData() []*Buffer {
	buffers := make([]*Buffer, 0, 2)
	if s.bufferA.HasData() {
		buffers = append(buffers, s.bufferA)
	}
	if s.bufferB.HasData() {
		buffers = append(buffers, s.bufferB)
	}
	return buffers
}

// ID returns the shard identifier
func (s *Shard) ID() uint32 {
	return s.id
}

// IsFull returns true if the shard is ready for flush
func (s *Shard) IsFull() bool {
	return s.readyForFlush.Load()
}

// HasData returns true if the inactive buffer has data
func (s *Shard) HasData() bool {
	return s.GetInactiveBuffer().HasData()
}

// Offset returns the offset of the active buffer
func (s *Shard) Offset() int32 {
	activeBufPtr := s.activeBuffer.Load()
	if activeBufPtr == nil {
		return headerOffset
	}
	return (*activeBufPtr).Offset()
}

// Capacity returns the capacity of buffers
func (s *Shard) Capacity() int32 {
	return s.capacity
}

// Close releases resources associated with the shard
func (s *Shard) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear finalizer before manual cleanup to prevent double-unmap
	runtime.SetFinalizer(s, nil)

	if s.cleanupA != nil {
		s.cleanupA()
	}
	if s.cleanupB != nil {
		s.cleanupB()
	}

	// Unmap buffers
	if s.bufferA != nil && len(s.bufferA.data) > 0 {
		unix.Munmap(s.bufferA.data)
		s.bufferA.data = nil
	}
	if s.bufferB != nil && len(s.bufferB.data) > 0 {
		unix.Munmap(s.bufferB.data)
		s.bufferB.data = nil
	}
}
