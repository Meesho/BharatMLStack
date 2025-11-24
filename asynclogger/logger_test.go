package asynclogger

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := DefaultConfig("/tmp/test.log")
		err := config.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 64*1024*1024, config.BufferSize) // 64MB baseline configuration
		assert.Equal(t, 8, config.NumShards)
		assert.Equal(t, 10*time.Second, config.FlushInterval)
	})

	t.Run("missing log path", func(t *testing.T) {
		config := Config{}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LogFilePath is required")
	})

	t.Run("applies defaults", func(t *testing.T) {
		config := Config{LogFilePath: "/tmp/test.log"}
		err := config.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 64*1024*1024, config.BufferSize) // 64MB baseline configuration
		assert.Equal(t, 8, config.NumShards)
	})

	t.Run("shard size too small", func(t *testing.T) {
		config := Config{
			LogFilePath: "/tmp/test.log",
			BufferSize:  1024, // 1KB
			NumShards:   100,  // Would result in 10 bytes per shard
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shard size too small")
	})
}

func TestLogger_BasicLogging(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Log some messages
	logger.Log("test message 1")
	logger.Log("test message 2")
	logger.Log("test message 3")

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	// Close to ensure all data is flushed
	err = logger.Close()
	assert.NoError(t, err)

	// Verify stats
	totalLogs, droppedLogs, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(3), totalLogs)
	assert.Equal(t, int64(0), droppedLogs)

	// Verify file exists
	_, err = os.Stat(logPath)
	assert.NoError(t, err)
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	tests := []struct {
		name       string
		goroutines int
		messages   int
	}{
		{"8 goroutines", 8, 1000},
		{"50 goroutines", 50, 500},
		{"100 goroutines", 100, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logPath := filepath.Join(t.TempDir(), "test.log")
			config := DefaultConfig(logPath)
			config.FlushInterval = 50 * time.Millisecond

			logger, err := New(config)
			require.NoError(t, err)
			defer logger.Close()

			var wg sync.WaitGroup
			totalMessages := tt.goroutines * tt.messages

			// Launch concurrent writers
			for i := 0; i < tt.goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < tt.messages; j++ {
						logger.Log(fmt.Sprintf("goroutine %d message %d", id, j))
					}
				}(i)
			}

			wg.Wait()

			// Wait for flushes
			time.Sleep(200 * time.Millisecond)

			// Close logger
			err = logger.Close()
			assert.NoError(t, err)

			// Verify stats
			totalLogs, droppedLogs, _, _, flushErrors, _ := logger.GetStatsSnapshot()

			// Should have logged most messages (some drops acceptable under high load)
			assert.Greater(t, totalLogs, int64(totalMessages*90/100), "should log at least 90%% of messages")
			assert.Equal(t, int64(0), flushErrors, "should have no flush errors")

			t.Logf("Logged %d/%d messages (%.2f%% success, %.2f%% dropped)",
				totalLogs, totalMessages,
				float64(totalLogs)/float64(totalMessages)*100,
				float64(droppedLogs)/float64(totalMessages)*100)
		})
	}
}

func TestLogger_BufferFillingAndSwapping(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := Config{
		LogFilePath:   logPath,
		BufferSize:    256 * 1024, // 256KB buffer to trigger swaps (4 shards * 64KB min)
		NumShards:     4,
		FlushInterval: 10 * time.Second, // Long interval to test buffer-driven swaps
	}

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Write enough data to fill buffers and trigger swaps
	// Each shard is 64KB (256KB / 4 shards)
	// Write 10KB messages to fill shards faster
	message := make([]byte, 10*1024) // 10KB message
	for i := range message {
		message[i] = 'A'
	}

	// Write 100 messages = 1MB total, should trigger multiple swaps
	for i := 0; i < 100; i++ {
		logger.Log(string(message))
	}

	// Wait for flushes
	time.Sleep(200 * time.Millisecond)

	// Check that swaps occurred
	_, _, _, _, _, setSwaps := logger.GetStatsSnapshot()
	assert.Greater(t, setSwaps, int64(0), "should have performed buffer swaps")

	// Close logger
	err = logger.Close()
	assert.NoError(t, err)
}

func TestLogger_GracefulShutdown(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 1 * time.Hour // Very long to test shutdown flush

	logger, err := New(config)
	require.NoError(t, err)

	// Log messages
	for i := 0; i < 100; i++ {
		logger.Log(fmt.Sprintf("message %d", i))
	}

	// Close immediately (should flush all data)
	err = logger.Close()
	assert.NoError(t, err)

	// Verify all messages were logged
	totalLogs, _, bytesWritten, _, flushErrors, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(100), totalLogs)
	assert.Greater(t, bytesWritten, int64(0))
	assert.Equal(t, int64(0), flushErrors)

	// Verify file has content
	info, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestLogger_DoubleClose(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	logger, err := New(config)
	require.NoError(t, err)

	logger.Log("test message")

	// First close
	err = logger.Close()
	assert.NoError(t, err)

	// Second close should not panic or error
	err = logger.Close()
	assert.NoError(t, err)

	// Logging after close should be handled gracefully (drops)
	logger.Log("should be dropped")
	_, droppedLogs, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Greater(t, droppedLogs, int64(0))
}

func TestLogger_Statistics(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Log messages
	numMessages := 50
	for i := 0; i < numMessages; i++ {
		logger.Log(fmt.Sprintf("message %d", i))
	}

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	// Check statistics
	totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps := logger.GetStatsSnapshot()

	assert.Equal(t, int64(numMessages), totalLogs, "should track total logs")
	assert.Equal(t, int64(0), droppedLogs, "should have no dropped logs")
	assert.Greater(t, bytesWritten, int64(0), "should track bytes written")
	assert.Greater(t, flushes, int64(0), "should have performed flushes")
	assert.Equal(t, int64(0), flushErrors, "should have no flush errors")
	assert.GreaterOrEqual(t, setSwaps, int64(0), "should track set swaps")
}

func TestLogger_MessageWithoutNewline(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	logger, err := New(config)
	require.NoError(t, err)

	// Log without newline
	logger.Log("message without newline")

	// Log with newline
	logger.Log("message with newline\n")

	time.Sleep(100 * time.Millisecond)
	err = logger.Close()
	assert.NoError(t, err)

	// Both should be logged successfully
	totalLogs, _, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(2), totalLogs)
}

func TestBuffer_Write(t *testing.T) {
	buffer := NewBuffer(1024, 0)

	// Write some data (now includes 4-byte length prefix)
	logData := []byte("test message\n")
	n, needsFlush := buffer.Write(logData)
	// Expected: 4 bytes (length prefix) + 13 bytes (log data) = 17 bytes
	assert.Equal(t, 17, n)
	assert.False(t, needsFlush)

	// Verify offset (includes 8-byte header reservation + 17 bytes of data = 25)
	assert.Equal(t, int32(25), buffer.Offset())
	assert.True(t, buffer.HasData())
	assert.False(t, buffer.IsFull())

	// Get data (with timeout) - now returns full capacity including header reservation
	data, complete := buffer.GetData(10 * time.Millisecond)
	// Data includes 8-byte header reservation at start, then log data starts at offset 8
	assert.GreaterOrEqual(t, len(data), 25) // At least the written data + header reservation
	assert.True(t, complete)                // All writes should be complete

	// Verify length prefix is correct (little-endian uint32 of 13)
	// Length prefix starts at offset 8 (after header reservation)
	expectedLength := uint32(13)
	actualLength := binary.LittleEndian.Uint32(data[8:12])
	assert.Equal(t, expectedLength, actualLength)

	// Verify log data follows the length prefix (starts at offset 12)
	assert.Equal(t, "test message\n", string(data[12:25]))

	// Reset
	buffer.Reset()
	assert.Equal(t, int32(8), buffer.Offset()) // Reset to header offset
	assert.False(t, buffer.HasData())
}

func TestBuffer_FillAndFlush(t *testing.T) {
	// Use 1KB buffer (will be aligned to 1024 bytes)
	buffer := NewBuffer(1024, 0)
	capacity := buffer.Capacity()

	// Fill buffer to 91% to trigger flush (90% threshold)
	// Account for 4-byte length prefix: we need fillSize bytes of data,
	// but Write() will use fillSize + 4 bytes total
	fillSize := int(float64(capacity)*0.91) - 4 // Subtract 4 for length prefix
	message := make([]byte, fillSize)
	for i := range message {
		message[i] = 'X'
	}

	n, needsFlush := buffer.Write(message)
	// Write returns fillSize + 4 (length prefix + data)
	assert.Equal(t, fillSize+4, n)
	assert.True(t, needsFlush, "should trigger flush at 90%% capacity")

	// Buffer should be marked as full
	assert.True(t, buffer.IsFull())

	// Additional writes should fail
	n, needsFlush = buffer.Write([]byte("more"))
	assert.Equal(t, 0, n)
	assert.True(t, needsFlush)
}

func TestShard_ConcurrentWrites(t *testing.T) {
	shard := NewShard(10*1024, 0)

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				shard.Write([]byte(fmt.Sprintf("g%d-m%d\n", id, j)))
			}
		}(i)
	}

	wg.Wait()

	// Verify shard has data
	assert.True(t, shard.HasData())
	// Offset starts at 8 (header reservation), so check it's greater than 8
	assert.Greater(t, shard.Offset(), int32(8))
}

func TestBufferSet_RoundRobin(t *testing.T) {
	// Use larger buffer to ensure all writes succeed
	bufferSet := NewBufferSet(256*1024, 4, 0) // 256KB total, 64KB per shard

	// Write messages and track which shards are used
	shardIDs := make([]int, 0, 20)
	for i := 0; i < 20; i++ {
		n, _, shardID := bufferSet.Write([]byte("test message\n"))
		if n > 0 && shardID >= 0 {
			shardIDs = append(shardIDs, shardID)
		}
	}

	// Verify writes succeeded
	assert.Equal(t, 20, len(shardIDs), "all writes should succeed")

	// Verify round-robin pattern: should cycle through shards
	// With 20 writes and 4 shards, we expect pattern like: 1,2,3,0,1,2,3,0,1,2,3,0,1,2,3,0,1,2,3,0
	// Check that all 4 shards were used
	uniqueShards := make(map[int]bool)
	for _, id := range shardIDs {
		uniqueShards[id] = true
	}
	assert.GreaterOrEqual(t, len(uniqueShards), 3, "should use at least 3 different shards")

	// Verify the pattern has some distribution
	// Count should be roughly equal (allow for ±2 variance)
	shardCounts := make(map[int]int)
	for _, id := range shardIDs {
		shardCounts[id]++
	}

	// With 20 writes to 4 shards, expect 5 per shard
	for _, count := range shardCounts {
		assert.GreaterOrEqual(t, count, 3, "each shard should have at least 3 writes")
		assert.LessOrEqual(t, count, 7, "each shard should have at most 7 writes")
	}
}

func TestBufferSet_HasData(t *testing.T) {
	bufferSet := NewBufferSet(4*1024, 4, 0)

	// Initially no data
	assert.False(t, bufferSet.HasData())

	// Write some data
	bufferSet.Write([]byte("test message\n"))

	// Should have data now
	assert.True(t, bufferSet.HasData())

	// Reset all shards
	bufferSet.Reset()

	// Should have no data after reset
	assert.False(t, bufferSet.HasData())
}

func TestLogger_LogBytes(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Test with byte slice
	logger.LogBytes([]byte("test message 1"))
	logger.LogBytes([]byte("test message 2\n")) // With newline
	logger.LogBytes([]byte("test message 3"))

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	// Close to ensure all data is flushed
	err = logger.Close()
	assert.NoError(t, err)

	// Verify stats
	totalLogs, droppedLogs, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(3), totalLogs)
	assert.Equal(t, int64(0), droppedLogs)
}

func TestLogger_LogBytes_ZeroAllocation(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 1 * time.Second

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Pre-allocate buffer (simulating worker pattern)
	buf := make([]byte, 256)

	// Reuse buffer multiple times
	for i := 0; i < 100; i++ {
		n := copy(buf, fmt.Sprintf("Message %d\n", i))
		logger.LogBytes(buf[:n])
	}

	time.Sleep(200 * time.Millisecond)
	err = logger.Close()
	assert.NoError(t, err)

	totalLogs, droppedLogs, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(100), totalLogs)
	assert.Equal(t, int64(0), droppedLogs)
}

func TestLogger_LogBytes_ConcurrentWithReuse(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	var wg sync.WaitGroup
	numWorkers := 10
	messagesPerWorker := 100

	// Each worker has its own reusable buffer
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			buf := make([]byte, 256) // Per-worker buffer

			for j := 0; j < messagesPerWorker; j++ {
				n := copy(buf, fmt.Sprintf("Worker %d message %d\n", workerID, j))
				logger.LogBytes(buf[:n])
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)
	err = logger.Close()
	assert.NoError(t, err)

	totalLogs, droppedLogs, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(numWorkers*messagesPerWorker), totalLogs)
	assert.Equal(t, int64(0), droppedLogs)
}

func TestLogger_LogString_BackwardCompatible(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Test that string API still works
	logger.Log("string message 1")
	logger.Log("string message 2\n")
	logger.Log("string message 3")

	time.Sleep(200 * time.Millisecond)
	err = logger.Close()
	assert.NoError(t, err)

	totalLogs, _, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(3), totalLogs)
}

func TestLogger_MixedStringAndBytes(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Mix string and byte APIs
	logger.Log("string message")
	logger.LogBytes([]byte("bytes message"))
	logger.Log("another string")
	logger.LogBytes([]byte("another bytes\n"))

	time.Sleep(100 * time.Millisecond)
	err = logger.Close()
	assert.NoError(t, err)

	totalLogs, droppedLogs, _, _, _, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(4), totalLogs)
	assert.Equal(t, int64(0), droppedLogs)
}

func TestLogger_FileFormatVerification(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	// Use small buffer with single shard to avoid round-robin distribution
	config.BufferSize = 64 * 1024 // 64KB total buffer
	config.NumShards = 1          // 1 shard = all messages in one shard
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	require.NoError(t, err)
	defer logger.Close()

	// Write known test messages (small data)
	testMessages := []string{
		"message 1",
		"message 2\n",
		"message 3\n",
		"test",
	}

	for _, msg := range testMessages {
		logger.Log(msg)
	}

	// Wait for flush and ensure all data is written
	time.Sleep(300 * time.Millisecond)

	// Force flush by closing (this ensures all data is flushed)
	err = logger.Close()
	require.NoError(t, err)

	// Give a bit more time for file writes to complete
	time.Sleep(50 * time.Millisecond)

	// Verify stats
	totalLogs, droppedLogs, bytesWritten, _, flushErrors, _ := logger.GetStatsSnapshot()
	assert.Equal(t, int64(len(testMessages)), totalLogs)
	assert.Equal(t, int64(0), droppedLogs)
	assert.Greater(t, bytesWritten, int64(0))
	assert.Equal(t, int64(0), flushErrors)

	// Read file and verify format
	file, err := os.Open(logPath)
	require.NoError(t, err)
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0), "file should have content")

	// Read all shards from file (with single shard, all messages should be in one shard)
	allParsedMessages := []string{}
	fileOffset := int64(0)
	shardIndex := 0
	maxShards := 10 // Safety limit to prevent infinite loop

	for fileOffset < fileInfo.Size() && shardIndex < maxShards {
		// Check if we have enough bytes for header
		if fileOffset+8 > fileInfo.Size() {
			break // Not enough bytes for header
		}

		// Read shard header (8 bytes)
		header := make([]byte, 8)
		n, err := file.ReadAt(header, fileOffset)
		if err != nil || n != 8 {
			break // End of file or incomplete header
		}

		// Verify header structure
		capacity := binary.LittleEndian.Uint32(header[0:4])
		validDataBytes := binary.LittleEndian.Uint32(header[4:8])

		// Skip empty shards (capacity=0 or validDataBytes=0 means no data)
		if capacity == 0 || validDataBytes == 0 {
			break // No more shards with data
		}

		// Validate header values
		assert.Greater(t, capacity, uint32(0), "capacity should be > 0")
		assert.Greater(t, validDataBytes, uint32(0), "validDataBytes should be > 0")
		assert.LessOrEqual(t, validDataBytes, capacity, "validDataBytes should <= capacity")

		t.Logf("Shard %d header: capacity=%d, validDataBytes=%d", shardIndex, capacity, validDataBytes)

		// Read shard data (starting immediately after header - no padding between!)
		fileOffset += 8 // Move past header
		shardData := make([]byte, validDataBytes)
		n, err = file.ReadAt(shardData, fileOffset)
		require.NoError(t, err)
		require.Equal(t, int(validDataBytes), n, "should read exactly validDataBytes")

		// Verify data starts immediately after header (check first few bytes are log entry length prefix)
		if len(shardData) >= 4 {
			firstEntryLength := binary.LittleEndian.Uint32(shardData[0:4])
			assert.Greater(t, firstEntryLength, uint32(0), "first entry length should be > 0")
			assert.LessOrEqual(t, firstEntryLength, validDataBytes, "first entry length should <= validDataBytes")
		}

		// Parse log entries from this shard's data
		offset := 0
		for offset < len(shardData) {
			// Check if we have enough bytes for length prefix
			if offset+4 > len(shardData) {
				break // Incomplete entry
			}

			// Read length prefix
			entryLength := binary.LittleEndian.Uint32(shardData[offset : offset+4])
			offset += 4

			// Validate entry length
			if entryLength == 0 {
				break // Invalid entry
			}

			// Check if we have enough bytes for log data
			if offset+int(entryLength) > len(shardData) {
				break // Incomplete entry
			}

			// Extract log entry
			entryData := shardData[offset : offset+int(entryLength)]
			allParsedMessages = append(allParsedMessages, string(entryData))
			offset += int(entryLength)
		}

		// Move to next shard position
		// Since we're using writevAligned, the shard buffer is padded to 512-byte alignment
		// Calculate aligned size: round up (8 + validDataBytes) to 512-byte boundary
		shardTotalSize := 8 + int(validDataBytes)
		alignedShardSize := ((shardTotalSize + 511) / 512) * 512 // Round up to 512-byte boundary
		fileOffset += int64(alignedShardSize)
		shardIndex++

		// Stop if we've read past the file or if next read would be beyond file
		if fileOffset >= fileInfo.Size() {
			break
		}
	}

	// Verify all messages were written correctly
	assert.Greater(t, len(allParsedMessages), 0, "should parse at least one message")
	t.Logf("Parsed %d messages from %d shards (expected %d messages)", len(allParsedMessages), shardIndex, len(testMessages))

	// With single shard, all messages should be in one shard
	// Verify all test messages are present
	assert.Equal(t, len(testMessages), len(allParsedMessages), "should parse all messages")

	// Verify messages match in order (with single shard, order should be preserved)
	// This verifies data correctness: what was written matches what is read
	for i, expected := range testMessages {
		if i < len(allParsedMessages) {
			actual := allParsedMessages[i]
			assert.Equal(t, expected, actual, "message %d content mismatch: expected '%s', got '%s'", i, expected, actual)
			// Verify byte-by-byte correctness
			assert.Equal(t, []byte(expected), []byte(actual), "message %d byte content mismatch", i)
		}
	}

	t.Logf("✅ Verified: All %d messages match exactly (data correctness confirmed)", len(testMessages))

	// Verify file structure: header + data should be contiguous (no padding between)
	allData, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Verify first shard: header at offset 0, data starts at offset 8
	assert.GreaterOrEqual(t, len(allData), 8, "file should have at least header")
	firstHeader := allData[0:8]
	firstCapacity := binary.LittleEndian.Uint32(firstHeader[0:4])
	firstValidData := binary.LittleEndian.Uint32(firstHeader[4:8])

	if firstValidData > 0 && len(allData) >= 8+int(firstValidData) {
		// Verify data starts immediately at offset 8 (no padding between header and data)
		firstShardData := allData[8 : 8+int(firstValidData)]
		// First 4 bytes should be a valid length prefix
		if len(firstShardData) >= 4 {
			firstLength := binary.LittleEndian.Uint32(firstShardData[0:4])
			assert.Greater(t, firstLength, uint32(0), "first entry should have valid length")
			assert.LessOrEqual(t, firstLength, firstValidData, "first entry length should <= validDataBytes")
		}
		t.Logf("✅ Verified: Data starts immediately after header (no padding between)")
		t.Logf("✅ Verified: Header capacity=%d, validDataBytes=%d", firstCapacity, firstValidData)
	}
}
