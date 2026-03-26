package logger

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// setupLoggerWithoutDiode configures logger without diode (ring buffer)
func setupLoggerWithoutDiode(b *testing.B) {
	b.Helper()

	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Configure viper without ring buffer
	viper.Set("APP_NAME", "benchmark_test")
	viper.Set("APP_LOG_LEVEL", "INFO")
	// Don't set LOG_RB_SIZE to avoid diode

	// Initialize logger
	Init()
}

// setupLoggerWithDiode configures logger with diode (ring buffer)
func setupLoggerWithDiode(b *testing.B, rbSize int) {
	b.Helper()

	// Reset viper and logger state
	viper.Reset()
	initialized = false
	once = sync.Once{}

	// Configure viper with ring buffer
	viper.Set("APP_NAME", "benchmark_test")
	viper.Set("APP_LOG_LEVEL", "INFO")
	viper.Set("LOG_RB_SIZE", rbSize)
	viper.Set("LOG_RB_DRAINING_INTERVAL", 5*time.Millisecond)

	// Initialize logger
	Init()
}

// redirectStdout redirects stdout to discard output for cleaner benchmarks
func redirectStdout(b *testing.B) func() {
	b.Helper()

	originalStdout := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open /dev/null: %v", err)
	}

	os.Stdout = devNull

	return func() {
		devNull.Close()
		os.Stdout = originalStdout
	}
}

// BenchmarkLoggerWithoutDiode benchmarks logging performance without diode
func BenchmarkLoggerWithoutDiode(b *testing.B) {
	setupLoggerWithoutDiode(b)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().Msgf("Log message without diode %d", i)
			i++
		}
	})
}

// BenchmarkLoggerWithDiode benchmarks logging performance with diode
func BenchmarkLoggerWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 1000) // Ring buffer size of 1000
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().Msgf("Log message with diode %d", i)
			i++
		}
	})
}

// BenchmarkLoggerWithDiodeSmallBuffer benchmarks with smaller ring buffer
func BenchmarkLoggerWithDiodeSmallBuffer(b *testing.B) {
	setupLoggerWithDiode(b, 100) // Smaller ring buffer
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().Msgf("Log message with small diode %d", i)
			i++
		}
	})
}

// BenchmarkLoggerWithDiodeLargeBuffer benchmarks with larger ring buffer
func BenchmarkLoggerWithDiodeLargeBuffer(b *testing.B) {
	setupLoggerWithDiode(b, 10000) // Larger ring buffer
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().Msgf("Log message with large diode %d", i)
			i++
		}
	})
}

// BenchmarkLoggerHighConcurrencyWithoutDiode tests high concurrency without diode
func BenchmarkLoggerHighConcurrencyWithoutDiode(b *testing.B) {
	setupLoggerWithoutDiode(b)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()
	const numGoroutines = 100

	b.ResetTimer()
	b.SetParallelism(numGoroutines)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().
				Int("goroutine_id", i%numGoroutines).
				Int("iteration", i).
				Str("message", "High concurrency test without diode").
				Msg("benchmark")
			i++
		}
	})
}

// BenchmarkLoggerHighConcurrencyWithDiode tests high concurrency with diode
func BenchmarkLoggerHighConcurrencyWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 5000) // Larger buffer for high concurrency
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()
	const numGoroutines = 100

	b.ResetTimer()
	b.SetParallelism(numGoroutines)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().
				Int("goroutine_id", i%numGoroutines).
				Int("iteration", i).
				Str("message", "High concurrency test with diode").
				Msg("benchmark")
			i++
		}
	})
}

// BenchmarkLoggerExtremeConcurrencyWithDiode tests extreme concurrency with diode
func BenchmarkLoggerExtremeConcurrencyWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 10000) // Large buffer for extreme concurrency
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()
	const numGoroutines = 500

	b.ResetTimer()
	b.SetParallelism(numGoroutines)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().
				Int("goroutine_id", i%numGoroutines).
				Int("iteration", i).
				Str("component", "benchmark").
				Str("test_type", "extreme_concurrency").
				Int64("timestamp", time.Now().UnixNano()).
				Msg("Extreme concurrency logging test")
			i++
		}
	})
}

// BenchmarkLoggerDifferentLogLevelsWithDiode tests different log levels with diode
func BenchmarkLoggerDifferentLogLevelsWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 2000)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 5 {
			case 0:
				log.Ctx(ctx).Debug().Msgf("Debug message %d", i)
			case 1:
				log.Ctx(ctx).Info().Msgf("Info message %d", i)
			case 2:
				log.Ctx(ctx).Warn().Msgf("Warning message %d", i)
			case 3:
				log.Ctx(ctx).Error().Msgf("Error message %d", i)
			case 4:
				log.Ctx(ctx).Info().
					Str("key1", "value1").
					Int("key2", i).
					Bool("key3", true).
					Msgf("Structured message %d", i)
			}
			i++
		}
	})
}

// BenchmarkLoggerDifferentLogLevelsWithoutDiode tests different log levels without diode
func BenchmarkLoggerDifferentLogLevelsWithoutDiode(b *testing.B) {
	setupLoggerWithoutDiode(b)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 5 {
			case 0:
				log.Ctx(ctx).Debug().Msgf("Debug message %d", i)
			case 1:
				log.Ctx(ctx).Info().Msgf("Info message %d", i)
			case 2:
				log.Ctx(ctx).Warn().Msgf("Warning message %d", i)
			case 3:
				log.Ctx(ctx).Error().Msgf("Error message %d", i)
			case 4:
				log.Ctx(ctx).Info().
					Str("key1", "value1").
					Int("key2", i).
					Bool("key3", true).
					Msgf("Structured message %d", i)
			}
			i++
		}
	})
}

// BenchmarkLoggerStructuredLoggingWithDiode tests structured logging with diode
func BenchmarkLoggerStructuredLoggingWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 3000)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().
				Str("service", "benchmark_service").
				Str("method", "test_method").
				Int("request_id", i).
				Int64("latency_ms", int64(i%1000)).
				Float64("cpu_usage", float64(i%100)/100.0).
				Bool("success", i%2 == 0).
				Str("user_id", fmt.Sprintf("user_%d", i%1000)).
				Str("session_id", fmt.Sprintf("session_%d", i%500)).
				Msg("Structured logging benchmark with many fields")
			i++
		}
	})
}

// BenchmarkLoggerLargeMessagesWithDiode tests large messages with diode
func BenchmarkLoggerLargeMessagesWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 5000)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()

	// Create a large message payload
	largeMessage := make([]byte, 1024) // 1KB message
	for i := range largeMessage {
		largeMessage[i] = byte(65 + (i % 26)) // Fill with A-Z
	}
	largeMessageStr := string(largeMessage)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Info().
				Int("sequence", i).
				Str("payload", largeMessageStr).
				Msg("Large message benchmark")
			i++
		}
	})
}

// BenchmarkLoggerErrorWithStackTraceWithDiode tests error logging with stack trace
func BenchmarkLoggerErrorWithStackTraceWithDiode(b *testing.B) {
	setupLoggerWithDiode(b, 2000)
	restore := redirectStdout(b)
	defer restore()

	ctx := context.Background()
	testErr := fmt.Errorf("benchmark test error %d", 12345)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Ctx(ctx).Error().
				Err(testErr).
				Int("error_code", 500).
				Str("component", "benchmark").
				Msgf("Error occurred during benchmark iteration %d", i)
			i++
		}
	})
}
