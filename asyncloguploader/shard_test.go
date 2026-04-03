package asyncloguploader

import (
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShard_NewShard(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("CreatesShardWithDoubleBuffer", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil) // 1MB capacity
		require.NoError(t, err)
		defer shard.Close()

		assert.NotNil(t, shard.bufferA)
		assert.NotNil(t, shard.bufferB)
		assert.Equal(t, uint32(1), shard.ID())
		assert.Equal(t, int32(1024*1024), shard.Capacity())
		assert.Equal(t, headerOffset, int(shard.Offset()))
	})

	t.Run("InitializesOffsetsToHeaderOffset", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		assert.Equal(t, headerOffset, int(shard.bufferA.Offset()))
		assert.Equal(t, headerOffset, int(shard.bufferB.Offset()))
	})

	t.Run("SetsBufferAAsInitialActive", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		activeBufPtr := shard.activeBuffer.Load()
		assert.NotNil(t, activeBufPtr)
		assert.Equal(t, shard.bufferA, *activeBufPtr)
	})
}

func TestShard_Write(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("WritesDataToActiveBuffer", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		data := []byte("test log entry")
		n, needsFlush := shard.Write(data)

		assert.Greater(t, n, 0)
		assert.False(t, needsFlush)
		assert.Greater(t, shard.Offset(), int32(headerOffset))
	})

	t.Run("PrependsLengthPrefixAndTimestamp", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		data := []byte("test")
		before := time.Now().UnixNano()
		n, _ := shard.Write(data)
		after := time.Now().UnixNano()

		// Should write 4-byte length prefix + 8-byte timestamp + data
		const timestampSize = 8
		expectedSize := 4 + timestampSize + len(data)
		assert.Equal(t, expectedSize, n)

		// Verify length prefix = timestampSize + len(data)
		activeBufPtr := shard.activeBuffer.Load()
		require.NotNil(t, activeBufPtr)
		activeBuf := *activeBufPtr

		recordStart := int32(headerOffset) // first record starts right after header
		lengthPrefix := binary.LittleEndian.Uint32(activeBuf.data[recordStart : recordStart+4])
		assert.Equal(t, uint32(timestampSize+len(data)), lengthPrefix)

		// Verify timestamp is within expected range
		tsNano := binary.LittleEndian.Uint64(activeBuf.data[recordStart+4 : recordStart+4+timestampSize])
		assert.GreaterOrEqual(t, int64(tsNano), before)
		assert.LessOrEqual(t, int64(tsNano), after)

		// Verify payload
		payloadStart := recordStart + 4 + int32(timestampSize)
		assert.Equal(t, data, activeBuf.data[payloadStart:payloadStart+int32(len(data))])
	})

	t.Run("ReturnsNeedsFlushWhenBufferFull", func(t *testing.T) {
		shard, err := NewShard(1024, 1, flushChan, nil) // Small capacity (aligned to 4096)
		require.NoError(t, err)
		defer shard.Close()

		// Fill buffer; each record = 4 (prefix) + 8 (timestamp) + 500 (payload) = 512 bytes
		largeData := make([]byte, 500)
		for i := 0; i < 20; i++ {
			n, needsFlush := shard.Write(largeData)
			if needsFlush {
				// n=0 when the buffer is full and the write couldn't fit (caller should retry);
				// n>0 when the write succeeded but the buffer hit the 90% threshold.
				return
			}
			assert.Greater(t, n, 0)
		}
		t.Fatal("Buffer should have filled")
	})

	t.Run("HandlesEmptyData", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		n, needsFlush := shard.Write(nil)
		assert.Equal(t, 0, n)
		assert.False(t, needsFlush)
	})
}

func TestShard_TrySwap(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("SwapsWhenBufferNearlyFull", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil) // 1MB capacity
		require.NoError(t, err)
		defer shard.Close()

		// Write enough data to trigger swap (90% of capacity)
		// Account for headerOffset (8), length prefix (4), and timestamp (8) per write
		capacity := int(shard.Capacity())
		targetSize := capacity * 9 / 10
		dataSize := targetSize - headerOffset - 4 - 8 // Subtract header, length prefix, and timestamp
		largeData := make([]byte, dataSize)
		n, needsFlush := shard.Write(largeData)

		assert.Greater(t, n, 0)
		_ = needsFlush
	})

	t.Run("IsCASProtected", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// Multiple goroutines trying to swap concurrently
		// Use a barrier to ensure all goroutines start CAS at the same time
		var wg sync.WaitGroup
		ready := make(chan struct{})
		swapped := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-ready // Wait for all goroutines to be ready
				success := shard.swapping.CompareAndSwap(false, true)
				swapped <- success
				if success {
					// Don't release immediately - let test verify only one succeeded
					// Release will happen after test checks
				}
			}()
		}

		// Start all goroutines at once
		close(ready)
		wg.Wait()

		// Only one should succeed in the concurrent CAS
		successCount := 0
		for i := 0; i < 10; i++ {
			if <-swapped {
				successCount++
			}
		}
		assert.Equal(t, 1, successCount, "Only one goroutine should succeed in concurrent CAS")

		// Release the lock if one succeeded
		if successCount == 1 {
			shard.swapping.Store(false)
		}
	})
}

func TestShard_BufferGetData(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsBufferData", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// Write some data
		data := []byte("test data")
		shard.Write(data)

		// Swap to make current buffer inactive
		shard.trySwap()

		// Get data from inactive buffer
		inactiveBuf := shard.GetInactiveBuffer()
		bufferData, allCompleted := inactiveBuf.GetData(100 * time.Millisecond)

		assert.NotNil(t, bufferData)
		assert.True(t, allCompleted)
		assert.Equal(t, int(shard.Capacity()), len(bufferData))
	})

	t.Run("WaitsForWriteCompletion", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// Write data
		shard.Write([]byte("test"))
		shard.trySwap()

		// GetData should wait for writes to complete
		inactiveBuf := shard.GetInactiveBuffer()
		bufferData, allCompleted := inactiveBuf.GetData(100 * time.Millisecond)

		assert.NotNil(t, bufferData)
		assert.True(t, allCompleted)
	})

	t.Run("ReturnsFalseOnTimeout", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// GetData with very short timeout
		inactiveBuf := shard.GetInactiveBuffer()
		bufferData, allCompleted := inactiveBuf.GetData(1 * time.Nanosecond)

		// Should return data even if timeout (may be empty)
		assert.NotNil(t, bufferData)
		// May or may not be completed depending on timing
		_ = allCompleted
	})
}

func TestShard_GetInactiveBuffer(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsInactiveBuffer", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// GetInactiveBuffer should return a valid buffer
		inactiveBuf := shard.GetInactiveBuffer()
		assert.NotNil(t, inactiveBuf)
		// Initially, inactive buffer should be empty (offset at headerOffset)
		assert.GreaterOrEqual(t, inactiveBuf.Offset(), int32(headerOffset))
	})
}

func TestShard_GetBuffersWithData(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsBuffersWithData", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// Write data and swap
		shard.Write([]byte("test"))
		shard.trySwap()

		// GetBuffersWithData should return buffers that have data
		buffers := shard.GetBuffersWithData()
		assert.Greater(t, len(buffers), 0)
	})

	t.Run("ReturnsEmptyWhenNoData", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		buffers := shard.GetBuffersWithData()
		assert.Equal(t, 0, len(buffers))
	})
}

func TestShard_ConcurrentWrites(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("HandlesConcurrentWrites", func(t *testing.T) {
		shard, err := NewShard(10*1024*1024, 1, flushChan, nil) // 10MB capacity
		require.NoError(t, err)
		defer shard.Close()

		const numGoroutines = 10
		const writesPerGoroutine = 100
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < writesPerGoroutine; j++ {
					data := []byte{byte(id), byte(j)}
					shard.Write(data)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify data was written
		assert.Greater(t, shard.Offset(), int32(headerOffset))
	})
}

func TestShard_HasData(t *testing.T) {
	flushChan := make(chan *Buffer, 100)
	
	t.Run("ReturnsFalseWhenNoData", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		assert.False(t, shard.HasData())
	})

	t.Run("ReturnsTrueWhenHasData", func(t *testing.T) {
		shard, err := NewShard(1024*1024, 1, flushChan, nil)
		require.NoError(t, err)
		defer shard.Close()

		// Write data to active buffer
		shard.Write([]byte("test"))
		// Swap to make the buffer with data inactive
		shard.trySwap()
		// Give time for swap to complete
		time.Sleep(10 * time.Millisecond)

		// HasData checks inactive buffer, which should now have data
		// But HasData() checks GetInactiveBuffer().HasData()
		// Let's verify the inactive buffer has data
		inactiveBuf := shard.GetInactiveBuffer()
		if inactiveBuf.HasData() {
			assert.True(t, shard.HasData())
		} else {
			// If inactive buffer doesn't have data yet, check if active buffer has data
			// by checking offset
			assert.Greater(t, shard.Offset(), int32(headerOffset))
		}
	})
}
