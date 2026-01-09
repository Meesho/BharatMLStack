package asyncloguploader

import (
	"math/rand/v2"
)

// ShardCollection represents a collection of shards with individual double buffers
// Each shard manages its own double buffer and swaps independently
type ShardCollection struct {
	shards    []*Shard
	numShards int
	threshold int32          // Threshold count (25% of numShards) - used by flush worker
	flushChan chan<- *Buffer // Channel to send buffers for flush (set by Logger)
}

// NewShardCollection creates a new collection of shards with individual double buffers
// totalCapacity is divided evenly among numShards
// Threshold is fixed at 25% of numShards
// flushChan is required - buffers will be sent to it on swap
func NewShardCollection(totalCapacity, numShards int, flushChan chan<- *Buffer) (*ShardCollection, error) {
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
		shard, err := NewShard(shardCapacity, uint32(i), flushChan)
		if err != nil {
			// Cleanup already created shards on error
			for j := 0; j < i; j++ {
				shards[j].Close()
			}
			return nil, err
		}
		shards[i] = shard
	}

	// Calculate threshold: 25% of numShards
	threshold := int32((numShards * 25) / 100)
	if threshold == 0 {
		threshold = 1 // At least 1 shard
	}

	return &ShardCollection{
		shards:    shards,
		numShards: numShards,
		threshold: threshold,
		flushChan: flushChan,
	}, nil
}

// Write writes data to a shard using random selection for better load distribution
// Returns bytes written, whether flush is needed, and which shard was written to
func (sc *ShardCollection) Write(p []byte) (n int, needsFlush bool, shardID int) {
	if len(p) == 0 {
		return 0, false, -1
	}

	// Random selection for better load distribution across shards
	shardIdx := rand.IntN(sc.numShards)
	shard := sc.shards[shardIdx]

	n, needsFlush = shard.Write(p)

	// Note: Buffer is already pushed to flush channel by trySwap() in shard.Write()
	// The flush worker tracks threshold by counting unique shards in the buffer list

	return n, needsFlush, shardIdx
}

// GetShard returns a specific shard by index
func (sc *ShardCollection) GetShard(idx int) *Shard {
	if idx < 0 || idx >= sc.numShards {
		return nil
	}
	return sc.shards[idx]
}

// NumShards returns the number of shards
func (sc *ShardCollection) NumShards() int {
	return sc.numShards
}

// Shards returns all shards for iteration
func (sc *ShardCollection) Shards() []*Shard {
	return sc.shards
}

// GetReadyShards returns all shards that are ready for flush
func (sc *ShardCollection) GetReadyShards() []*Shard {
	ready := make([]*Shard, 0, sc.numShards)
	for _, shard := range sc.shards {
		if shard.IsFull() {
			ready = append(ready, shard)
		}
	}
	return ready
}

// HasData returns true if any shard has data
func (sc *ShardCollection) HasData() bool {
	for _, shard := range sc.shards {
		if shard.HasData() {
			return true
		}
	}
	return false
}

// TotalBytes returns the total bytes currently in all shards (excluding header reservations)
func (sc *ShardCollection) TotalBytes() int64 {
	var total int64
	for _, shard := range sc.shards {
		// Offset includes the 8-byte header reservation, so subtract it for actual data size
		total += int64(shard.Offset() - headerOffset)
	}
	return total
}

// Close releases all resources associated with the shard collection
func (sc *ShardCollection) Close() {
	for _, shard := range sc.shards {
		shard.Close()
	}
}
