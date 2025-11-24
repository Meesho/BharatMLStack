package asynclogger

import (
	"sync/atomic"
)

// BufferSet represents a set of shards that can be swapped atomically
type BufferSet struct {
	shards    []*Shard
	numShards int
	id        uint32
	counter   atomic.Uint64 // For round-robin shard selection
}

// NewBufferSet creates a new set of shards
// totalCapacity is divided evenly among numShards
func NewBufferSet(totalCapacity, numShards int, setID uint32) *BufferSet {
	if numShards <= 0 {
		numShards = 8 // Default
	}

	shardCapacity := totalCapacity / numShards
	if shardCapacity < 64*1024 {
		// Ensure minimum 64KB per shard
		shardCapacity = 64 * 1024
		numShards = totalCapacity / shardCapacity
		if numShards == 0 {
			numShards = 1
		}
	}

	shards := make([]*Shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = NewShard(shardCapacity, uint32(i))
	}

	return &BufferSet{
		shards:    shards,
		numShards: numShards,
		id:        setID,
	}
}

// Write writes data to a shard using round-robin selection
// Returns bytes written, whether flush is needed, and which shard was written to
func (bs *BufferSet) Write(p []byte) (n int, needsFlush bool, shardID int) {
	if len(p) == 0 {
		return 0, false, -1
	}

	// Round-robin shard selection
	counterVal := bs.counter.Add(1)
	shardIdx := int(counterVal % uint64(bs.numShards))
	shard := bs.shards[shardIdx]

	n, needsFlush = shard.Write(p)
	return n, needsFlush, shardIdx
}

// GetShard returns a specific shard by index
func (bs *BufferSet) GetShard(idx int) *Shard {
	if idx < 0 || idx >= bs.numShards {
		return nil
	}
	return bs.shards[idx]
}

// NumShards returns the number of shards
func (bs *BufferSet) NumShards() int {
	return bs.numShards
}

// Shards returns all shards for iteration
func (bs *BufferSet) Shards() []*Shard {
	return bs.shards
}

// ID returns the set identifier
func (bs *BufferSet) ID() uint32 {
	return bs.id
}

// SetID updates the set identifier
func (bs *BufferSet) SetID(id uint32) {
	bs.id = id
}

// HasData returns true if any shard has data
func (bs *BufferSet) HasData() bool {
	for _, shard := range bs.shards {
		if shard.HasData() {
			return true
		}
	}
	return false
}

// AnyShardFull returns true if any shard is marked for flush
func (bs *BufferSet) AnyShardFull() bool {
	for _, shard := range bs.shards {
		if shard.IsFull() {
			return true
		}
	}
	return false
}

// Reset resets all shards in the set
func (bs *BufferSet) Reset() {
	for _, shard := range bs.shards {
		shard.Reset()
	}
}

// TotalBytes returns the total bytes currently in all shards (excluding header reservations)
func (bs *BufferSet) TotalBytes() int64 {
	var total int64
	for _, shard := range bs.shards {
		// Offset includes the 8-byte header reservation, so subtract it for actual data size
		total += int64(shard.Offset() - 8)
	}
	return total
}
