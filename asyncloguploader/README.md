# Async Log Uploader

A high-performance, asynchronous logging library with automatic Google Cloud Storage (GCS) upload support. Designed for high-throughput scenarios where applications need efficient event logging and cloud storage integration.

## Features

- 🚀 **High Performance**: Lock-free CAS operations, Direct I/O, and sharded double buffers
- 📊 **Event-Based Logging**: Multiple loggers per event type via `LoggerManager`
- 🔄 **Automatic Rotation**: Size-based file rotation with configurable thresholds
- ☁️ **GCS Upload**: Automatic upload of completed log files to GCS buckets
- ⚡ **Parallel Uploads**: Chunked parallel uploads with multi-level compose support
- 📈 **Comprehensive Metrics**: Detailed statistics for monitoring and debugging
- 🛡️ **Production Ready**: Error handling, retries, and graceful shutdown

## Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "time"
    
    "github.com/Meesho/BharatMLStack/asyncloguploader"
    "github.com/Meesho/go-core/metric"
)

func main() {
    // Create configuration
    config := asyncloguploader.DefaultConfig("/var/log/myapp/app.log")
    config.BufferSize = 64 * 1024 * 1024  // 64MB
    config.NumShards = 8
    config.MaxFileSize = 100 * 1024 * 1024  // 100MB rotation
    config.MetricTags = metric.BuildTag(
        metric.NewTag("service", "myapp"),
        metric.NewTag("env", "prod"),
    )
    
    // Optional: Configure GCS upload
    gcsConfig := asyncloguploader.DefaultGCSUploadConfig("my-gcs-bucket")
    gcsConfig.ObjectPrefix = "logs/myapp/"
    config.GCSUploadConfig = &gcsConfig
    
    // Create logger manager
    loggerManager, err := asyncloguploader.NewLoggerManager(config)
    if err != nil {
        log.Fatalf("Failed to create logger manager: %v", err)
    }
    defer loggerManager.Close()
    
    // Log events
    loggerManager.LogWithEvent("payment", `{"amount": 100, "currency": "USD"}`)
    loggerManager.LogWithEvent("login", `{"user": "alice", "timestamp": "2024-01-01T00:00:00Z"}`)
    loggerManager.LogBytesWithEvent("api", []byte("raw binary data"))
    
    // Give time for async operations
    time.Sleep(2 * time.Second)
    
    // Metrics (logBytes, logBytesDropped, logBytesSuccess, etc.) are emitted
    // via go-core/metric and can be viewed in Grafana with Config.MetricTags.
}
```

### Advanced Usage

#### Event Logger Management

```go
// Initialize logger for specific event
err := loggerManager.InitializeEventLogger("payment")
if err != nil {
    log.Fatal(err)
}

// Check if logger exists
if loggerManager.HasEventLogger("payment") {
    loggerManager.LogWithEvent("payment", "Payment processed")
}

// List all active loggers
events := loggerManager.ListEventLoggers()
log.Printf("Active events: %v", events)

// Close specific logger
err = loggerManager.CloseEventLogger("payment")
if err != nil {
    log.Printf("Error closing logger: %v", err)
}
```

#### Metrics and Tag Propagation

```go
// Pass application tags for Grafana filtering
config.MetricTags = metric.BuildTag(
    metric.NewTag("service", "myapp"),
    metric.NewTag("env", "prod"),
)

// Metrics are emitted via go-core/metric:
// - logBytes, logBytesDropped, logBytesSuccess, logBytesWritten
// - logBytesFlushAttempts, logBytesFlushSuccess, logBytesFlushFailure, logBytesFlushDuration
// - fileWriterWriteDuration, fileWriterRotationCount
// - uploadFile, uploadFileFailed, uploadFileDuration, uploadBytes
// Event name is automatically added as event_name tag for event-scoped metrics.
```

## Architecture

The module uses a sharded double-buffer architecture with lock-free operations:

1. **LoggerManager**: Manages multiple `Logger` instances (one per event)
2. **Logger**: Core async logger with sharded buffers
3. **ShardCollection**: Collection of shards for parallel writes
4. **Shard**: Double buffer with CAS-based swapping
5. **FileWriter**: Direct I/O writes with file rotation
6. **Uploader**: GCS upload with parallel chunk uploads

### Data Flow

```
Application → LoggerManager → Logger → ShardCollection → Shard (Buffer)
                                                              ↓
                                                         Flush Worker
                                                              ↓
                                                         FileWriter (.tmp → .log on rotation)
                                                              ↓
                                                         Uploader (scans dir for .log) → GCS
```

## Configuration

### Logger Configuration

```go
type Config struct {
    BufferSize          int           // Total buffer size (default: 64MB)
    NumShards           int           // Number of shards (default: 8)
    LogFilePath         string        // Base log file path (required)
    MaxFileSize         int64         // Max file size before rotation (0 = disabled)
    PreallocateFileSize int64         // Preallocation size (0 = disabled)
    FlushInterval       time.Duration // Periodic flush interval (default: 10s)
    FlushTimeout        time.Duration // Write completion timeout (default: 10ms)
    GCSUploadConfig     *GCSUploadConfig // Optional: GCS upload config (scans log dir)
}
```

### GCS Upload Configuration

```go
type GCSUploadConfig struct {
    Bucket              string        // GCS bucket name (required)
    ObjectPrefix        string        // Object prefix (e.g., "logs/event1/")
    ChunkSize           int           // Chunk size (default: 32MB)
    MaxChunksPerCompose int           // Max chunks per compose (default: 32)
    MaxRetries          int           // Max retry attempts (default: 3)
    RetryDelay          time.Duration // Retry delay (default: 5s)
    GRPCPoolSize        int           // gRPC connection pool size (default: 64)
    ScanInterval        time.Duration // Scan log dir for .log files (default: 10s)
}
```

### Example Configuration

```go
config := asyncloguploader.Config{
    BufferSize:          128 * 1024 * 1024,  // 128MB
    NumShards:           16,                  // 16 shards
    LogFilePath:         "/var/log/myapp/app.log",
    MaxFileSize:         500 * 1024 * 1024,  // 500MB rotation
    PreallocateFileSize: 100 * 1024 * 1024,  // 100MB preallocation
    FlushInterval:       5 * time.Second,
    FlushTimeout:        50 * time.Millisecond,
    GCSUploadConfig: &asyncloguploader.GCSUploadConfig{
        Bucket:              "my-logs-bucket",
        ObjectPrefix:        "logs/myapp/",
        ChunkSize:           32 * 1024 * 1024,  // 32MB chunks
        MaxChunksPerCompose: 32,
        MaxRetries:          5,
        RetryDelay:          10 * time.Second,
        GRPCPoolSize:        128,
        ScanInterval:        10 * time.Second,
    },
}
```

## Performance Tuning

### Buffer Size

- **Small (32MB)**: Lower memory usage, more frequent flushes
- **Medium (64MB)**: Balanced performance (default)
- **Large (128MB+)**: Higher throughput, less frequent flushes

### Number of Shards

- **Few (4-8)**: Lower parallelism, simpler coordination
- **Medium (8-16)**: Balanced parallelism (default: 8)
- **Many (16+)**: Higher parallelism, more overhead

### Flush Threshold

The flush threshold is fixed at 25% of shards. When 25% of shards are ready, a batch flush is triggered. This balances:
- **Lower threshold**: More frequent flushes, lower latency
- **Higher threshold**: Less frequent flushes, higher throughput

### File Rotation

- **Small files (100MB)**: More frequent uploads, easier processing
- **Medium files (500MB)**: Balanced (recommended)
- **Large files (1GB+)**: Less frequent uploads, longer processing time

## File Format

Each log file contains shard buffers with the following format:

```
[Shard Header (8 bytes)]
├── Capacity (4 bytes, little-endian uint32)
├── Valid Data Bytes (4 bytes, little-endian uint32)
└── [Data Section]
    ├── Entry 1: [Length (4 bytes)][Data (N bytes)]
    ├── Entry 2: [Length (4 bytes)][Data (M bytes)]
    └── ...
```

### Entry Format

- **Length Prefix**: 4-byte little-endian uint32 indicating data length
- **Data**: Raw log data (JSON, text, binary, etc.)

## GCS Upload

### Automatic Upload

When `GCSUploadConfig` is provided, completed log files are automatically uploaded to GCS:

1. File rotation triggers upload
2. File is read into memory
3. Split into chunks (default: 32MB)
4. Chunks uploaded in parallel
5. Chunks composed into final object
6. Local file deleted after successful upload

### Chunk Compose

The module handles GCS's 32-chunk compose limit:
- **≤32 chunks**: Single compose operation
- **>32 chunks**: Multi-level compose (intermediate objects)

### GCS Credentials

The module uses Google Application Default Credentials (ADC):

```bash
# Set environment variable
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"

# Or use gcloud
gcloud auth application-default login
```

## Error Handling

### Write Errors

- **Buffer Full**: Logs are dropped, `DroppedLogs` counter incremented
- **Logger Closed**: Logs are dropped gracefully
- **Write Failures**: Retry mechanism with timeout

### Flush Errors

- **Flush Failures**: Tracked in `FlushErrors` counter
- **Buffer Reset**: Buffers reset even on error to prevent deadlock
- **Error Logging**: Errors logged with context

### Upload Errors

- **Retry Logic**: Configurable retries with exponential backoff
- **Failed Uploads**: Tracked in `Failed` counter
- **Cleanup**: Temporary chunks cleaned up on error
- **Non-Blocking**: Upload failures don't block logging

## Metrics

Metrics are emitted via `github.com/Meesho/go-core/metric`. Use `Config.MetricTags` to pass application tags (e.g., service, env) for Grafana filtering. Event name is automatically added as `event_name` tag.

### Logger Metrics

- `logBytes`: Total log attempts
- `logBytesDropped`: Logs dropped (buffer full, closed, etc.)
- `logBytesSuccess`: Successful writes
- `logBytesWritten`: Bytes written (Count metric)
- `logBytesFlushAttempts`: Flush operations started
- `logBytesFlushSuccess`: Successful flushes
- `logBytesFlushFailure`: Failed flushes
- `logBytesFlushDuration`: Flush duration (Timing)

### FileWriter Metrics

- `fileWriterWriteDuration`: Write duration per WriteVectored call
- `fileWriterRotationCount`: File rotations completed

### Upload Metrics

- `uploadFile`: Upload attempts
- `uploadFileFailed`: Failed uploads after retries
- `uploadFileDuration`: Upload duration (Timing)
- `uploadBytes`: Bytes uploaded (Count)

## Requirements

### System Requirements

- **OS**: Linux (Direct I/O requires Linux-specific syscalls)
- **Go**: 1.24.0 or later
- **Filesystem**: Ext4 or similar (4096-byte alignment)

### Dependencies

- `cloud.google.com/go/storage`: GCS client
- `golang.org/x/sys`: System calls (Direct I/O)
- `google.golang.org/api`: Google API client

## Installation

```bash
go get github.com/neehar-mavuduru/logger-double-buffer/asyncloguploader
```

## Examples

### Example 1: Simple Logging

```go
config := asyncloguploader.DefaultConfig("/var/log/myapp/app.log")
loggerManager, _ := asyncloguploader.NewLoggerManager(config)
defer loggerManager.Close()

loggerManager.LogWithEvent("api", `{"method": "GET", "path": "/users"}`)
```

### Example 2: With GCS Upload

```go
config := asyncloguploader.DefaultConfig("/var/log/myapp/app.log")
gcsConfig := asyncloguploader.DefaultGCSUploadConfig("my-bucket")
gcsConfig.ObjectPrefix = "logs/myapp/"
config.GCSUploadConfig = &gcsConfig

loggerManager, _ := asyncloguploader.NewLoggerManager(config)
defer loggerManager.Close()

loggerManager.LogWithEvent("payment", `{"amount": 100}`)
```

### Example 3: Custom Configuration

```go
config := asyncloguploader.Config{
    BufferSize:   128 * 1024 * 1024,
    NumShards:    16,
    LogFilePath:  "/var/log/myapp/app.log",
    MaxFileSize:  500 * 1024 * 1024,
    FlushInterval: 5 * time.Second,
}

loggerManager, _ := asyncloguploader.NewLoggerManager(config)
defer loggerManager.Close()
```

## Testing

Run tests:

```bash
go test ./asyncloguploader/...
```

Run with coverage:

```bash
go test -cover ./asyncloguploader/...
```

See [TESTING.md](./TESTING.md) for detailed testing information.

## Limitations

- **Linux Only**: Direct I/O requires Linux-specific syscalls
- **Single Process**: Not designed for multi-process scenarios
- **Memory Usage**: Entire file loaded into memory for upload
- **No Compression**: Logs are not compressed before upload

## Contributing

Contributions are welcome! Please see the main repository's contributing guidelines.

## License

See the main repository's LICENSE file.

## Related Documentation

- [High-Level Design (HLD)](./HLD.md): Detailed architecture and design decisions
- [Testing Guide](./TESTING.md): Testing strategies and examples

---

**Version**: 1.0  
**Last Updated**: 2024
