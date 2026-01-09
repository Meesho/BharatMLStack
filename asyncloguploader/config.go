package asyncloguploader

import (
	"fmt"
	"time"
)

// Config holds the configuration for the async logger
type Config struct {
	// Buffer configuration
	BufferSize int // Total buffer size in bytes (default: 64MB)
	NumShards  int // Number of shards (default: 8)

	// File configuration
	LogFilePath         string // Path to log file (required)
	MaxFileSize         int64  // Maximum file size before rotation (0 = disabled)
	PreallocateFileSize int64  // Size to preallocate using fallocate (0 = disabled)

	// Flush timing
	FlushInterval time.Duration // Periodic flush trigger (default: 10s)
	FlushTimeout  time.Duration // Wait for write completion before flush (default: 10ms)

	// Upload configuration
	UploadChannel   chan<- string    // Optional: channel for completed files
	GCSUploadConfig *GCSUploadConfig // Optional: GCS upload configuration
}

// GCSUploadConfig holds configuration for GCS uploader
type GCSUploadConfig struct {
	Bucket              string        // GCS bucket name (required)
	ObjectPrefix        string        // Object prefix (e.g., "logs/event1/")
	ChunkSize           int           // Chunk size for parallel upload (default: 32MB)
	MaxChunksPerCompose int           // Maximum chunks per compose (default: 32)
	MaxRetries          int           // Max retry attempts (default: 3)
	RetryDelay          time.Duration // Delay between retries (default: 5s)
	GRPCPoolSize        int           // gRPC connection pool size (default: 64)
	ChannelBufferSize   int           // Upload channel buffer size (default: 100)
}

// DefaultConfig returns a configuration with baseline defaults
func DefaultConfig(logPath string) Config {
	return Config{
		BufferSize:          64 * 1024 * 1024, // 64MB
		NumShards:           8,                // 8 shards
		LogFilePath:         logPath,
		MaxFileSize:         0, // Disabled by default
		PreallocateFileSize: 0, // Disabled by default
		FlushInterval:       10 * time.Second,
		FlushTimeout:        10 * time.Millisecond,
		UploadChannel:       nil, // Optional
		GCSUploadConfig:     nil, // Optional
	}
}

// DefaultGCSUploadConfig returns a GCS upload configuration with defaults
func DefaultGCSUploadConfig(bucket string) GCSUploadConfig {
	return GCSUploadConfig{
		Bucket:              bucket,
		ObjectPrefix:        "",
		ChunkSize:           32 * 1024 * 1024, // 32MB
		MaxChunksPerCompose: 32,               // GCS limit
		MaxRetries:          3,
		RetryDelay:          5 * time.Second,
		GRPCPoolSize:        64,
		ChannelBufferSize:   100,
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

	// Ensure minimum shard size
	shardSize := c.BufferSize / c.NumShards
	if shardSize < 64*1024 {
		return fmt.Errorf("shard size too small (%d bytes), increase BufferSize or decrease NumShards", shardSize)
	}

	if c.FlushInterval <= 0 {
		c.FlushInterval = 10 * time.Second
	}

	if c.FlushTimeout <= 0 {
		c.FlushTimeout = 10 * time.Millisecond
	}

	// Validate GCS config if provided
	if c.GCSUploadConfig != nil {
		if err := c.GCSUploadConfig.Validate(); err != nil {
			return fmt.Errorf("GCSUploadConfig validation failed: %w", err)
		}
	}

	return nil
}

// Validate checks if the GCS upload configuration is valid
func (g *GCSUploadConfig) Validate() error {
	if g.Bucket == "" {
		return fmt.Errorf("bucket name is required")
	}

	if g.ChunkSize <= 0 {
		g.ChunkSize = 32 * 1024 * 1024 // 32MB default
	}

	if g.MaxChunksPerCompose <= 0 {
		g.MaxChunksPerCompose = 32 // GCS limit
	}

	if g.MaxRetries <= 0 {
		g.MaxRetries = 3
	}

	if g.RetryDelay <= 0 {
		g.RetryDelay = 5 * time.Second
	}

	if g.GRPCPoolSize <= 0 {
		g.GRPCPoolSize = 64
	}

	if g.ChannelBufferSize <= 0 {
		g.ChannelBufferSize = 100
	}

	return nil
}
