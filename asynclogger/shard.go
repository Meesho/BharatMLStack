package asynclogger

import (
	"sync"
	"time"
)

// Shard represents a single shard with its own buffer and mutex
type Shard struct {
	buffer *Buffer
	mu     sync.Mutex
}

// NewShard creates a new shard with the specified capacity
func NewShard(capacity int, id uint32) *Shard {
	return &Shard{
		buffer: NewBuffer(capacity, id),
	}
}

// Write writes data to the shard's buffer
// Returns the number of bytes written and whether a flush is needed
func (s *Shard) Write(p []byte) (int, bool) {
	return s.buffer.Write(p)
}

// GetData returns the current data in the shard's buffer
// Should only be called during flush operations
// Returns the data and whether all writes completed (false if timeout occurred)
func (s *Shard) GetData(timeout time.Duration) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buffer.GetData(timeout)
}

// Reset clears the shard's buffer for reuse
// Should only be called after flushing
func (s *Shard) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer.Reset()
}

// Offset returns the current write offset in the buffer
func (s *Shard) Offset() int32 {
	return s.buffer.Offset()
}

// Capacity returns the buffer capacity
func (s *Shard) Capacity() int32 {
	return s.buffer.Capacity()
}

// ID returns the shard identifier
func (s *Shard) ID() uint32 {
	return s.buffer.ID()
}

// IsFull returns true if the buffer is marked for flushing
func (s *Shard) IsFull() bool {
	return s.buffer.IsFull()
}

// HasData returns true if the buffer contains any data
func (s *Shard) HasData() bool {
	return s.buffer.HasData()
}
