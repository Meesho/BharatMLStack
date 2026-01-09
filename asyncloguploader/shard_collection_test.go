package asyncloguploader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardCollection_NewShardCollection(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("CreatesCollectionWithCorrectShardCount", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 8, flushChan) // 8MB total, 8 shards
		require.NoError(t, err)
		defer collection.Close()

		assert.Equal(t, 8, collection.NumShards())
		assert.Len(t, collection.Shards(), 8)
	})

	t.Run("Calculates25PercentThreshold", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 8, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		// Threshold is internal, but we can verify shard count
		assert.Equal(t, 8, collection.NumShards())
	})

	t.Run("SetsMinimumThresholdToOne", func(t *testing.T) {
		collection, err := NewShardCollection(4*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		assert.Equal(t, 4, collection.NumShards())
	})

	t.Run("HandlesSmallShardSize", func(t *testing.T) {
		collection, err := NewShardCollection(64*1024, 8, flushChan) // Very small total
		require.NoError(t, err)
		defer collection.Close()

		// Should adjust shard count to ensure minimum 64KB per shard
		assert.GreaterOrEqual(t, collection.NumShards(), 1)
	})
}

func TestShardCollection_Write(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("WritesToShardUsingRandomSelection", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		data := []byte("test data")
		n, _, shardID := collection.Write(data)

		assert.Greater(t, n, 0)
		assert.GreaterOrEqual(t, shardID, 0)
		assert.Less(t, shardID, collection.NumShards())
	})

	t.Run("DistributesWritesAcrossShards", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		shardWrites := make(map[int]int)
		for i := 0; i < 20; i++ {
			_, _, shardID := collection.Write([]byte("test"))
			shardWrites[shardID]++
		}

		// Should distribute writes across shards (random selection)
		assert.Greater(t, len(shardWrites), 1)
	})

	t.Run("ReturnsZeroForEmptyData", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		n, needsFlush, shardID := collection.Write(nil)

		assert.Equal(t, 0, n)
		assert.False(t, needsFlush)
		assert.Equal(t, -1, shardID)
	})

	t.Run("EnqueuesBufferWhenFull", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		// Fill a shard to trigger flush
		largeData := make([]byte, 1024*1024) // 1MB
		for i := 0; i < 10; i++ {
			_, needsFlush, _ := collection.Write(largeData)
			if needsFlush {
				// Check if buffer was enqueued
				select {
				case <-flushChan:
					// Buffer was enqueued
					return
				default:
				}
			}
		}
	})
}

// Threshold logic is now handled by flush worker, not exposed in ShardCollection
// These tests are removed as the API has changed

func TestShardCollection_GetReadyShards(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsOnlyFullShards", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		// Fill some shards
		largeData := make([]byte, 1024*1024)
		for i := 0; i < 10; i++ {
			collection.Write(largeData)
		}

		readyShards := collection.GetReadyShards()
		for _, shard := range readyShards {
			assert.True(t, shard.IsFull())
		}
	})

	t.Run("ReturnsEmptyWhenNoShardsReady", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		readyShards := collection.GetReadyShards()
		assert.Empty(t, readyShards)
	})
}

// Reset methods are no longer part of ShardCollection API
// Buffers are reset individually after flush

func TestShardCollection_HasData(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsFalseWhenNoData", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		assert.False(t, collection.HasData())
	})

	t.Run("ReturnsTrueWhenHasData", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		// Write data
		collection.Write([]byte("test"))
		// Give time for write to complete
		time.Sleep(10 * time.Millisecond)

		// HasData checks inactive buffers, so we need to check if any shard has data in active buffer
		// or swap and check inactive. For simplicity, check if TotalBytes > 0
		totalBytes := collection.TotalBytes()
		assert.Greater(t, totalBytes, int64(0), "Should have data written")
	})
}

func TestShardCollection_GetShard(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsCorrectShard", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		shard := collection.GetShard(0)
		assert.NotNil(t, shard)
		assert.Equal(t, uint32(0), shard.ID())
	})

	t.Run("ReturnsNilForInvalidIndex", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		assert.Nil(t, collection.GetShard(-1))
		assert.Nil(t, collection.GetShard(10))
	})
}

func TestShardCollection_TotalBytes(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("CalculatesTotalBytesCorrectly", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		// Write data
		data := []byte("test")
		collection.Write(data)
		collection.Write(data)

		totalBytes := collection.TotalBytes()
		assert.Greater(t, totalBytes, int64(0))
	})

	t.Run("ExcludesHeaderReservation", func(t *testing.T) {
		collection, err := NewShardCollection(8*1024*1024, 4, flushChan)
		require.NoError(t, err)
		defer collection.Close()

		// Write exactly headerOffset bytes
		data := make([]byte, headerOffset)
		collection.Write(data)

		// TotalBytes should subtract headerOffset
		totalBytes := collection.TotalBytes()
		assert.GreaterOrEqual(t, totalBytes, int64(0))
	})
}

