package asynclogger

import (
	"fmt"
	"time"
)

// Config holds the configuration for the async logger
type Config struct {
	// LogFilePath is the path to the log file (required)
	LogFilePath string

	// BufferSize is the total buffer size in bytes (default: 64MB)
	BufferSize int

	// NumShards is the number of shards (default: 8)
	NumShards int

	// FlushInterval is the time-based flush trigger (default: 10s)
	FlushInterval time.Duration

	// FlushTimeout is the maximum time to wait for writes to complete before flushing (default: 10ms)
	// If timeout expires, flush proceeds anyway (may result in one corrupted log line)
	FlushTimeout time.Duration
}

// DefaultConfig returns a configuration with baseline defaults
// logPath is required - the path where logs will be written
func DefaultConfig(logPath string) Config {
	return Config{
		LogFilePath:   logPath,
		BufferSize:    64 * 1024 * 1024,      // 64MB (baseline configuration)
		NumShards:     8,                     // 8 shards
		FlushInterval: 10 * time.Second,      // 10 seconds
		FlushTimeout:  10 * time.Millisecond, // 10ms timeout for write completion
	}
}

// Validate checks if the configuration is valid and applies defaults where needed
func (c *Config) Validate() error {
	if c.LogFilePath == "" {
		return fmt.Errorf("LogFilePath is required")
	}

	if c.BufferSize <= 0 {
		c.BufferSize = 64 * 1024 * 1024 // 64MB default
	}

	if c.NumShards <= 0 {
		c.NumShards = 8 // 8 shards default
	}

	if c.FlushInterval <= 0 {
		c.FlushInterval = 10 * time.Second
	}

	if c.FlushTimeout <= 0 {
		c.FlushTimeout = 10 * time.Millisecond
	}

	// Ensure minimum shard size
	shardSize := c.BufferSize / c.NumShards
	if shardSize < 64*1024 {
		return fmt.Errorf("shard size too small (%d bytes), increase BufferSize or decrease NumShards", shardSize)
	}

	return nil
}
