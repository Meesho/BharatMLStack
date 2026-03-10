package asyncloguploader

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_NewLogger(t *testing.T) {
	t.Run("CreatesLoggerWithValidConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024
		config.NumShards = 4
		config.FlushInterval = 100 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		assert.NotNil(t, logger.shardCollection)
		assert.NotNil(t, logger.fileWriter)
		assert.NotNil(t, logger.flushChan)
		assert.NotNil(t, logger.semaphore)
	})

	t.Run("ReturnsErrorForInvalidConfig", func(t *testing.T) {
		config := Config{} // Invalid: missing LogFilePath

		logger, err := NewLogger(config, "test")
		assert.Error(t, err)
		assert.Nil(t, logger)
	})
}

func TestLogger_LogBytes(t *testing.T) {
	t.Run("WritesLogSuccessfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024
		config.NumShards = 4
		config.FlushInterval = 100 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		data := []byte("test log entry")
		logger.LogBytes(data)

		time.Sleep(200 * time.Millisecond)

		// Verify data was written to file
		err = logger.Close()
		require.NoError(t, err)
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.log"))
		require.NotEmpty(t, matches)
		info, err := os.Stat(matches[0])
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	t.Run("DropsLogWhenClosed", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024
		config.NumShards = 4

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)

		logger.Close()
		// LogBytes after close should not panic (drops silently)
		logger.LogBytes([]byte("test"))
	})

	t.Run("HandlesConcurrentWrites", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 10 * 1024 * 1024
		config.NumShards = 8
		config.FlushInterval = 100 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		const numGoroutines = 10
		const writesPerGoroutine = 100
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < writesPerGoroutine; j++ {
					data := []byte{byte(id), byte(j)}
					logger.LogBytes(data)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		err = logger.Close()
		require.NoError(t, err)
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.log"))
		require.NotEmpty(t, matches)
		info, err := os.Stat(matches[0])
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})
}

func TestLogger_SwapCoordination(t *testing.T) {
	t.Run("CoordinatesSwapWithSemaphore", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024 // Small buffer
		config.NumShards = 1            // Single shard to test swap
		config.FlushInterval = 100 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		// Fill buffer to trigger swap
		largeData := make([]byte, 512*1024) // 512KB
		for i := 0; i < 10; i++ {
			logger.LogBytes(largeData)
		}

		time.Sleep(200 * time.Millisecond)

		err = logger.Close()
		require.NoError(t, err)
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.log"))
		require.NotEmpty(t, matches)
		info, err := os.Stat(matches[0])
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	t.Run("RetriesAfterSwap", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024
		config.NumShards = 1
		config.FlushInterval = 100 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		// Fill buffer
		largeData := make([]byte, 512*1024)
		for i := 0; i < 5; i++ {
			logger.LogBytes(largeData)
		}

		logger.LogBytes([]byte("after fill"))

		time.Sleep(200 * time.Millisecond)

		err = logger.Close()
		require.NoError(t, err)
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.log"))
		require.NotEmpty(t, matches)
		info, err := os.Stat(matches[0])
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})
}

func TestLogger_Flush(t *testing.T) {
	t.Run("FlushesWhenThresholdReached", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 8 * 1024 * 1024
		config.NumShards = 8
		config.FlushInterval = 100 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		// Fill shards to reach threshold (2 out of 8)
		// Need to write enough data to fill multiple shards
		largeData := make([]byte, 512*1024) // 512KB per write
		for i := 0; i < 50; i++ {
			logger.LogBytes(largeData)
		}

		time.Sleep(500 * time.Millisecond)

		err = logger.Close()
		require.NoError(t, err)
	})

	t.Run("FlushesOnInterval", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 8 * 1024 * 1024
		config.NumShards = 8
		config.FlushInterval = 50 * time.Millisecond

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		logger.LogBytes([]byte("test"))

		time.Sleep(100 * time.Millisecond)

		err = logger.Close()
		require.NoError(t, err)
	})
}

func TestLogger_Close(t *testing.T) {
	t.Run("FlushesRemainingDataOnClose", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 8 * 1024 * 1024
		config.NumShards = 8

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)

		logger.LogBytes([]byte("test data"))
		err = logger.Close()
		assert.NoError(t, err)

		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.log"))
		require.NotEmpty(t, matches)
		info, err := os.Stat(matches[0])
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	t.Run("HandlesDoubleClose", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024
		config.NumShards = 4

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)

		err1 := logger.Close()
		err2 := logger.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

func TestLogger_Log(t *testing.T) {
	t.Run("ConvertsStringToBytes", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.BufferSize = 1024 * 1024
		config.NumShards = 4

		logger, err := NewLogger(config, "test")
		require.NoError(t, err)
		defer logger.Close()

		logger.Log("test message")
		time.Sleep(200 * time.Millisecond)

		err = logger.Close()
		require.NoError(t, err)
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.log"))
		require.NotEmpty(t, matches)
		info, err := os.Stat(matches[0])
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})
}
