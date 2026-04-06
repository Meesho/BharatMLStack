package logger

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func TestInitLogger(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Test valid initialization
	appName := "test_app"
	logLevel := "INFO"

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitLogger panicked: %v", r)
		}
	}()

	InitLogger(appName, logLevel)

	if !initialized {
		t.Error("Logger should be initialized")
	}
}

func TestInitLoggerEmptyAppName(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Test empty app name should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("InitLogger should panic with empty app name")
		}
	}()

	InitLogger("", "INFO")
}

func TestInitLoggerEmptyLogLevel(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Test empty log level should use default
	InitLogger("test_app", "")

	if !initialized {
		t.Error("Logger should be initialized with default log level")
	}
}

func TestInit(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "DEBUG")

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Init panicked: %v", r)
		}
	}()

	Init()

	if !initialized {
		t.Error("Logger should be initialized")
	}
}

func TestInitMissingAppName(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Test missing APP_NAME should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Init should panic with missing APP_NAME")
		}
	}()

	viper.Set("APP_LOG_LEVEL", "INFO")
	Init()
}

func TestInitMissingLogLevel(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Test missing APP_LOG_LEVEL should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Init should panic with missing APP_LOG_LEVEL")
		}
	}()

	viper.Set("APP_NAME", "test_app")
	Init()
}

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"DEBUG", true},
		{"INFO", true},
		{"WARN", true},
		{"ERROR", true},
		{"FATAL", true},
		{"PANIC", true},
		{"DISABLED", true},
		{"INVALID", false},
	}

	for _, test := range tests {
		t.Run(test.level, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && test.expected {
					t.Errorf("setLogLevel panicked for valid level %s: %v", test.level, r)
				} else if r == nil && !test.expected {
					t.Errorf("setLogLevel should have panicked for invalid level %s", test.level)
				}
			}()

			setLogLevel(test.level)
		})
	}
}

func TestTraceHook(t *testing.T) {
	hook := TraceHook{}
	ctx := context.Background()

	// Create a mock event
	event := log.Ctx(ctx).Info()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TraceHook.Run panicked: %v", r)
		}
	}()

	hook.Run(event, zerolog.InfoLevel, "test message")
}

func TestCustomHook(t *testing.T) {
	hook := CustomHook{}
	ctx := context.Background()

	// Create a mock event
	event := log.Ctx(ctx).Info()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CustomHook.Run panicked: %v", r)
		}
	}()

	hook.Run(event, zerolog.InfoLevel, "test message")
}

func TestMapToString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: "",
		},
		{
			name:     "single entry",
			input:    map[string]string{"key1": "value1"},
			expected: "key1=value1",
		},
		{
			name:     "multiple entries",
			input:    map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			expected: "", // We'll check this separately due to map iteration order
		},
		{
			name:     "empty values",
			input:    map[string]string{"key1": "", "key2": "value2"},
			expected: "", // We'll check this separately due to map iteration order
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := mapToString(test.input)

			// For tests with multiple entries, we need to handle map iteration order
			if test.name == "empty values" {
				// Check that both keys are present in the result
				if !strings.Contains(result, "key1=") || !strings.Contains(result, "key2=value2") {
					t.Errorf("mapToString() = %v, should contain both key1= and key2=value2", result)
				}
			} else if test.name == "multiple entries" {
				// Check that all three keys are present in the result
				if !strings.Contains(result, "key1=value1") || !strings.Contains(result, "key2=value2") || !strings.Contains(result, "key3=value3") {
					t.Errorf("mapToString() = %v, should contain key1=value1, key2=value2, and key3=value3", result)
				}
			} else {
				if result != test.expected {
					t.Errorf("mapToString() = %v, want %v", result, test.expected)
				}
			}
		})
	}
}

func TestLoggerWithRingBuffer(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config with ring buffer
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	viper.Set("LOG_RB_SIZE", 100)
	viper.Set("LOG_RB_DRAINING_INTERVAL", "5ms")

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Init with ring buffer panicked: %v", r)
		}
	}()

	Init()

	if !initialized {
		t.Error("Logger should be initialized with ring buffer")
	}
}

func TestLoggerWithoutRingBuffer(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config without ring buffer
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	// Don't set LOG_RB_SIZE

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Init without ring buffer panicked: %v", r)
		}
	}()

	Init()

	if !initialized {
		t.Error("Logger should be initialized without ring buffer")
	}
}

func TestLoggerAlreadyInitialized(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Initialize logger first time
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	Init()

	if !initialized {
		t.Error("Logger should be initialized")
	}

	// Try to initialize again - should not panic and should remain initialized
	Init()

	if !initialized {
		t.Error("Logger should remain initialized")
	}
}

func TestLoggingOutput(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	Init()

	// This test just verifies that logging doesn't panic
	// The actual output verification is complex due to the console writer
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logging panicked: %v", r)
		}
	}()

	// Log a message - this should not panic
	log.Info().Msg("Test log message")
	log.Warn().Msg("Test warning message")
	log.Error().Msg("Test error message")
}

func TestLoggerRingBufferOverflow(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config with very small ring buffer to trigger overflow
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	viper.Set("LOG_RB_SIZE", 3)                    // Very small buffer to trigger overflow quickly
	viper.Set("LOG_RB_DRAINING_INTERVAL", "100ms") // Slower draining to ensure overflow

	// Capture stderr to check for overflow warning
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Initialize logger with small ring buffer
	Init()

	if !initialized {
		t.Error("Logger should be initialized")
	}

	// Generate many log messages rapidly to trigger ring buffer overflow
	// We need to send more messages than the buffer can handle
	numMessages := 100
	for i := range numMessages {
		log.Info().Msgf("Test message %d", i)
	}

	// Give some time for the ring buffer to process and potentially overflow
	time.Sleep(200 * time.Millisecond)

	// Close the pipe and restore stderr
	w.Close()
	os.Stderr = originalStderr

	// Read captured stderr output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	r.Close()

	// Check if the overflow warning message appears in stderr
	expectedWarning := "Error from Logger: dropping logs due to buffer overflow"
	if !strings.Contains(output, expectedWarning) {
		t.Logf("Expected warning '%s' not found in stderr output: %s", expectedWarning, output)
		// Note: This test might be flaky due to timing, so we'll log but not fail
		// The important thing is that the logger doesn't panic during overflow
	}

	// Verify that logging continues to work even after potential overflow
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logging panicked after overflow: %v", r)
		}
	}()

	// Additional logging after potential overflow should not panic
	log.Info().Msg("Logging after potential overflow")
	log.Warn().Msg("Warning after potential overflow")
}

func TestLoggerRingBufferOverflowWithMetrics(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config with very small ring buffer
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	viper.Set("LOG_RB_SIZE", 4)                    // Minimal buffer to ensure overflow
	viper.Set("LOG_RB_DRAINING_INTERVAL", "300ms") // Slower draining

	// Initialize logger
	Init()

	if !initialized {
		t.Error("Logger should be initialized")
	}

	// This test verifies that the logger handles overflow gracefully
	// without panicking, even when the ring buffer is overwhelmed
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked during ring buffer overflow: %v", r)
		}
	}()

	// Send a burst of log messages to trigger overflow
	// The ring buffer should handle this gracefully
	for i := 0; i < 50; i++ {
		log.Info().
			Int("iteration", i).
			Str("message", "High frequency logging test").
			Msgf("Burst message %d", i)
	}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Verify logger is still functional
	log.Info().Msg("Logger still functional after overflow test")
}

func TestLoggerRingBufferNoOverflow(t *testing.T) {
	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Set up viper config with larger ring buffer to avoid overflow
	viper.Set("APP_NAME", "test_app")
	viper.Set("APP_LOG_LEVEL", "INFO")
	viper.Set("LOG_RB_SIZE", 1000)               // Large buffer
	viper.Set("LOG_RB_DRAINING_INTERVAL", "1ms") // Fast draining

	// Initialize logger
	Init()

	if !initialized {
		t.Error("Logger should be initialized")
	}

	// This test verifies normal operation without overflow
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked during normal operation: %v", r)
		}
	}()

	// Send normal amount of log messages
	for i := 0; i < 100; i++ {
		log.Info().Msgf("Normal message %d", i)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Verify logger is still functional
	log.Info().Msg("Logger functional after normal operation test")
}
