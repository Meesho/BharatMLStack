package asynclogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerManager(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "test.log")
		config := DefaultConfig(logPath)

		lm, err := NewLoggerManager(config)
		require.NoError(t, err)
		assert.NotNil(t, lm)
		assert.Equal(t, filepath.Dir(logPath), lm.baseDir)
	})

	t.Run("invalid config", func(t *testing.T) {
		config := Config{} // Missing LogFilePath

		lm, err := NewLoggerManager(config)
		assert.Error(t, err)
		assert.Nil(t, lm)
		assert.Contains(t, err.Error(), "invalid config")
	})

	t.Run("extracts base directory correctly", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "subdir", "test.log")
		config := DefaultConfig(logPath)

		lm, err := NewLoggerManager(config)
		require.NoError(t, err)
		assert.Equal(t, filepath.Dir(logPath), lm.baseDir)
	})

	t.Run("handles current directory", func(t *testing.T) {
		config := DefaultConfig("test.log") // Relative path

		lm, err := NewLoggerManager(config)
		require.NoError(t, err)
		assert.Equal(t, ".", lm.baseDir)
	})
}

func TestSanitizeEventName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{"valid name", "payment", "payment", false},
		{"with spaces", "payment event", "payment_event", false},
		{"with invalid chars", "payment/event:test", "payment_event_test", false},
		{"multiple invalid chars", "payment*test?file", "payment_test_file", false},
		{"empty string", "", "", true},
		{"only spaces", "   ", "___", false},
		{"long name", strings.Repeat("a", 300), strings.Repeat("a", 255), false},
		{"with backslash", "payment\\event", "payment_event", false},
		{"with quotes", `payment"event`, "payment_event", false},
		{"with angle brackets", "payment<event>test", "payment_event_test", false},
		{"with pipe", "payment|event", "payment_event", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeEventName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestLoggerManager_LogBytesWithEvent(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("creates logger on first use", func(t *testing.T) {
		data := []byte("test message\n")
		lm.LogBytesWithEvent("payment", data)

		// Wait for flush
		time.Sleep(200 * time.Millisecond)

		// Verify logger was created
		assert.True(t, lm.HasEventLogger("payment"))

		// Verify file was created
		eventLogPath := filepath.Join(lm.baseDir, "payment.log")
		_, err := os.Stat(eventLogPath)
		assert.NoError(t, err)
	})

	t.Run("multiple events create separate files", func(t *testing.T) {
		lm.LogBytesWithEvent("payment", []byte("payment log\n"))
		lm.LogBytesWithEvent("login", []byte("login log\n"))

		time.Sleep(200 * time.Millisecond)

		// Verify both loggers exist
		assert.True(t, lm.HasEventLogger("payment"))
		assert.True(t, lm.HasEventLogger("login"))

		// Verify both files exist
		paymentLog := filepath.Join(lm.baseDir, "payment.log")
		loginLog := filepath.Join(lm.baseDir, "login.log")
		_, err1 := os.Stat(paymentLog)
		_, err2 := os.Stat(loginLog)
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})

	t.Run("invalid event name drops log", func(t *testing.T) {
		// Create a fresh manager for this test to avoid interference
		lm2, err := NewLoggerManager(config)
		require.NoError(t, err)
		defer lm2.Close()

		initialStats, _, _, _, _, _ := lm2.GetStatsSnapshot()

		// Empty string is truly invalid and will be dropped
		lm2.LogBytesWithEvent("", []byte("should be dropped\n"))

		// Note: "invalid/name" gets sanitized to "invalid_name" which is valid
		// So it will create a logger, not drop the log
		lm2.LogBytesWithEvent("invalid/name", []byte("will be sanitized\n"))

		time.Sleep(100 * time.Millisecond)
		finalStats, _, _, _, _, _ := lm2.GetStatsSnapshot()

		// Empty string should not create a logger
		assert.False(t, lm2.HasEventLogger(""))
		
		// "invalid/name" gets sanitized to "invalid_name" and creates a logger
		assert.True(t, lm2.HasEventLogger("invalid_name"))
		
		// Stats should reflect the sanitized logger creation
		assert.Greater(t, finalStats, initialStats, "stats should reflect sanitized logger")
	})
}

func TestLoggerManager_LogWithEvent(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	lm.LogWithEvent("server", "server log message")
	time.Sleep(200 * time.Millisecond)

	assert.True(t, lm.HasEventLogger("server"))

	serverLog := filepath.Join(lm.baseDir, "server.log")
	_, err = os.Stat(serverLog)
	assert.NoError(t, err)
}

func TestLoggerManager_InitializeEventLogger(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("creates logger successfully", func(t *testing.T) {
		err := lm.InitializeEventLogger("payment")
		assert.NoError(t, err)
		assert.True(t, lm.HasEventLogger("payment"))
	})

	t.Run("idempotent - can call multiple times", func(t *testing.T) {
		err1 := lm.InitializeEventLogger("login")
		assert.NoError(t, err1)

		err2 := lm.InitializeEventLogger("login")
		assert.NoError(t, err2)

		// Should still have only one logger
		events := lm.ListEventLoggers()
		loginCount := 0
		for _, e := range events {
			if e == "login" {
				loginCount++
			}
		}
		assert.Equal(t, 1, loginCount)
	})

	t.Run("invalid event name returns error", func(t *testing.T) {
		err := lm.InitializeEventLogger("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid event name")
	})

	t.Run("creates file on initialization", func(t *testing.T) {
		err := lm.InitializeEventLogger("order")
		require.NoError(t, err)

		orderLog := filepath.Join(lm.baseDir, "order.log")
		_, err = os.Stat(orderLog)
		assert.NoError(t, err)
	})
}

func TestLoggerManager_CloseEventLogger(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("closes existing logger", func(t *testing.T) {
		// Create logger and log some data
		lm.LogBytesWithEvent("payment", []byte("test message\n"))
		time.Sleep(200 * time.Millisecond)

		assert.True(t, lm.HasEventLogger("payment"))

		// Close the logger
		err := lm.CloseEventLogger("payment")
		assert.NoError(t, err)

		// Verify logger is removed
		assert.False(t, lm.HasEventLogger("payment"))
	})

	t.Run("returns error for non-existent logger", func(t *testing.T) {
		err := lm.CloseEventLogger("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event logger not found")
	})

	t.Run("invalid event name returns error", func(t *testing.T) {
		err := lm.CloseEventLogger("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid event name")
	})

	t.Run("flushes pending logs before closing", func(t *testing.T) {
		lm.LogBytesWithEvent("order", []byte("order message\n"))

		// Close immediately (before flush interval)
		err := lm.CloseEventLogger("order")
		assert.NoError(t, err)

		// Verify file has content (data was flushed)
		orderLog := filepath.Join(lm.baseDir, "order.log")
		info, err := os.Stat(orderLog)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})
}

func TestLoggerManager_HasEventLogger(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("returns false for non-existent logger", func(t *testing.T) {
		assert.False(t, lm.HasEventLogger("nonexistent"))
	})

	t.Run("returns true after logger creation", func(t *testing.T) {
		lm.LogBytesWithEvent("payment", []byte("test\n"))
		assert.True(t, lm.HasEventLogger("payment"))
	})

	t.Run("returns false for invalid event name", func(t *testing.T) {
		assert.False(t, lm.HasEventLogger(""))
		assert.False(t, lm.HasEventLogger("invalid/name"))
	})
}

func TestLoggerManager_ListEventLoggers(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("returns empty list initially", func(t *testing.T) {
		events := lm.ListEventLoggers()
		assert.Empty(t, events)
	})

	t.Run("returns all active loggers", func(t *testing.T) {
		lm.LogBytesWithEvent("payment", []byte("test\n"))
		lm.LogBytesWithEvent("login", []byte("test\n"))
		lm.LogBytesWithEvent("order", []byte("test\n"))

		events := lm.ListEventLoggers()
		assert.Len(t, events, 3)
		assert.Contains(t, events, "payment")
		assert.Contains(t, events, "login")
		assert.Contains(t, events, "order")
	})

	t.Run("excludes closed loggers", func(t *testing.T) {
		lm.LogBytesWithEvent("temp", []byte("test\n"))
		assert.Contains(t, lm.ListEventLoggers(), "temp")

		err := lm.CloseEventLogger("temp")
		require.NoError(t, err)

		assert.NotContains(t, lm.ListEventLoggers(), "temp")
	})
}

func TestLoggerManager_Close(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)

	t.Run("closes all loggers", func(t *testing.T) {
		lm.LogBytesWithEvent("payment", []byte("test\n"))
		lm.LogBytesWithEvent("login", []byte("test\n"))

		time.Sleep(200 * time.Millisecond)

		err := lm.Close()
		assert.NoError(t, err)

		// Verify all loggers are closed
		assert.Empty(t, lm.ListEventLoggers())
	})

	t.Run("flushes all pending logs", func(t *testing.T) {
		lm2, err := NewLoggerManager(config)
		require.NoError(t, err)

		lm2.LogBytesWithEvent("order", []byte("order message\n"))
		lm2.LogBytesWithEvent("shipment", []byte("shipment message\n"))

		// Close immediately (before flush interval)
		err = lm2.Close()
		assert.NoError(t, err)

		// Verify files have content
		orderLog := filepath.Join(lm2.baseDir, "order.log")
		shipmentLog := filepath.Join(lm2.baseDir, "shipment.log")

		orderInfo, err := os.Stat(orderLog)
		require.NoError(t, err)
		assert.Greater(t, orderInfo.Size(), int64(0))

		shipmentInfo, err := os.Stat(shipmentLog)
		require.NoError(t, err)
		assert.Greater(t, shipmentInfo.Size(), int64(0))
	})

	t.Run("idempotent - can close multiple times", func(t *testing.T) {
		lm3, err := NewLoggerManager(config)
		require.NoError(t, err)

		err = lm3.Close()
		assert.NoError(t, err)

		err = lm3.Close()
		assert.NoError(t, err)
	})
}

func TestLoggerManager_GetStatsSnapshot(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("aggregates stats from all loggers", func(t *testing.T) {
		lm.LogBytesWithEvent("payment", []byte("payment 1\n"))
		lm.LogBytesWithEvent("payment", []byte("payment 2\n"))
		lm.LogBytesWithEvent("login", []byte("login 1\n"))

		time.Sleep(200 * time.Millisecond)

		totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps := lm.GetStatsSnapshot()

		assert.Equal(t, int64(3), totalLogs)
		assert.Equal(t, int64(0), droppedLogs)
		assert.Greater(t, bytesWritten, int64(0))
		assert.Greater(t, flushes, int64(0))
		assert.Equal(t, int64(0), flushErrors)
		assert.GreaterOrEqual(t, setSwaps, int64(0))
	})

	t.Run("returns zero stats when no loggers", func(t *testing.T) {
		lm2, err := NewLoggerManager(config)
		require.NoError(t, err)
		defer lm2.Close()

		totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps := lm2.GetStatsSnapshot()

		assert.Equal(t, int64(0), totalLogs)
		assert.Equal(t, int64(0), droppedLogs)
		assert.Equal(t, int64(0), bytesWritten)
		assert.Equal(t, int64(0), flushes)
		assert.Equal(t, int64(0), flushErrors)
		assert.Equal(t, int64(0), setSwaps)
	})
}

func TestLoggerManager_GetEventStats(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 100 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("returns stats for specific event", func(t *testing.T) {
		lm.LogBytesWithEvent("payment", []byte("payment message\n"))
		lm.LogBytesWithEvent("payment", []byte("payment message 2\n"))

		time.Sleep(200 * time.Millisecond)

		totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps, err := lm.GetEventStats("payment")
		require.NoError(t, err)

		assert.Equal(t, int64(2), totalLogs)
		assert.Equal(t, int64(0), droppedLogs)
		assert.Greater(t, bytesWritten, int64(0))
		assert.Greater(t, flushes, int64(0))
		assert.Equal(t, int64(0), flushErrors)
		assert.GreaterOrEqual(t, setSwaps, int64(0))
	})

	t.Run("returns error for non-existent event", func(t *testing.T) {
		_, _, _, _, _, _, err := lm.GetEventStats("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event logger not found")
	})

	t.Run("returns error for invalid event name", func(t *testing.T) {
		_, _, _, _, _, _, err := lm.GetEventStats("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid event name")
	})
}

func TestLoggerManager_ConcurrentAccess(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)
	config.FlushInterval = 50 * time.Millisecond

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("concurrent logger creation", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50
		eventsPerGoroutine := 10

		// Concurrently create loggers and log messages
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				eventName := fmt.Sprintf("event_%d", id%5) // 5 unique events
				for j := 0; j < eventsPerGoroutine; j++ {
					lm.LogBytesWithEvent(eventName, []byte(fmt.Sprintf("message %d\n", j)))
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Verify all events were created
		events := lm.ListEventLoggers()
		assert.Len(t, events, 5)

		// Verify stats
		totalLogs, droppedLogs, _, _, flushErrors, _ := lm.GetStatsSnapshot()
		expectedLogs := int64(numGoroutines * eventsPerGoroutine)
		assert.Equal(t, expectedLogs, totalLogs)
		assert.Equal(t, int64(0), droppedLogs)
		assert.Equal(t, int64(0), flushErrors)
	})

	t.Run("concurrent initialize and log", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 20

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				eventName := fmt.Sprintf("concurrent_%d", id%3)

				// Some initialize, some just log
				if id%2 == 0 {
					lm.InitializeEventLogger(eventName)
				}
				lm.LogBytesWithEvent(eventName, []byte("concurrent message\n"))
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Verify no errors occurred
		_, droppedLogs, _, _, flushErrors, _ := lm.GetStatsSnapshot()
		assert.Equal(t, int64(0), droppedLogs)
		assert.Equal(t, int64(0), flushErrors)
	})

	t.Run("concurrent close and log", func(t *testing.T) {
		lm2, err := NewLoggerManager(config)
		require.NoError(t, err)

		// Create some loggers
		lm2.LogBytesWithEvent("temp1", []byte("test\n"))
		lm2.LogBytesWithEvent("temp2", []byte("test\n"))
		time.Sleep(100 * time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(3)

		// Concurrently close one logger and log to others
		go func() {
			defer wg.Done()
			lm2.CloseEventLogger("temp1")
		}()

		go func() {
			defer wg.Done()
			lm2.LogBytesWithEvent("temp2", []byte("test\n"))
		}()

		go func() {
			defer wg.Done()
			lm2.LogBytesWithEvent("temp3", []byte("test\n"))
		}()

		wg.Wait()
		time.Sleep(100 * time.Millisecond)

		// temp1 should be closed, temp2 and temp3 should exist
		assert.False(t, lm2.HasEventLogger("temp1"))
		assert.True(t, lm2.HasEventLogger("temp2"))
		assert.True(t, lm2.HasEventLogger("temp3"))

		lm2.Close()
	})
}

func TestLoggerManager_EventNameSanitizationInFileNames(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("sanitized event names create valid file names", func(t *testing.T) {
		lm.LogBytesWithEvent("payment/event", []byte("test\n"))
		lm.LogBytesWithEvent("login event", []byte("test\n"))
		lm.LogBytesWithEvent("order*test", []byte("test\n"))

		time.Sleep(100 * time.Millisecond)

		// Verify files exist with sanitized names
		paymentLog := filepath.Join(lm.baseDir, "payment_event.log")
		loginLog := filepath.Join(lm.baseDir, "login_event.log")
		orderLog := filepath.Join(lm.baseDir, "order_test.log")

		_, err1 := os.Stat(paymentLog)
		_, err2 := os.Stat(loginLog)
		_, err3 := os.Stat(orderLog)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
	})
}

func TestLoggerManager_LoadOrStoreRaceCondition(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	config := DefaultConfig(logPath)

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	t.Run("handles concurrent creation of same logger", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// All goroutines try to create the same logger simultaneously
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				lm.LogBytesWithEvent("race_test", []byte("test\n"))
			}()
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)

		// Should have only one logger instance
		events := lm.ListEventLoggers()
		raceTestCount := 0
		for _, e := range events {
			if e == "race_test" {
				raceTestCount++
			}
		}
		assert.Equal(t, 1, raceTestCount)

		// Verify stats
		totalLogs, _, _, _, _, _ := lm.GetStatsSnapshot()
		assert.Equal(t, int64(numGoroutines), totalLogs)
	})
}

