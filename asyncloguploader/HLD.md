# High-Level Design: Async Log Uploader Module

## 1. Overview

The `asyncloguploader` module provides a high-performance, asynchronous logging system with automatic upload to Google Cloud Storage (GCS). It is designed for high-throughput scenarios where applications need to log events efficiently and upload completed log files to cloud storage.

### 1.1 Key Features

- **Asynchronous Logging**: Non-blocking write operations using sharded double buffers
- **High Performance**: Lock-free CAS operations on hot path, Direct I/O for disk writes
- **Event-Based Logging**: Multiple loggers per event type (managed by `LoggerManager`)
- **Automatic File Rotation**: Size-based rotation with configurable thresholds
- **GCS Upload**: Automatic upload of completed log files to GCS buckets
- **Parallel Uploads**: Chunked parallel uploads with multi-level compose support
- **Production Ready**: Comprehensive error handling, retries, and statistics

### 1.2 Architecture Principles

- **Lock-Free Hot Path**: Write operations use atomic CAS operations, avoiding mutex contention
- **Sharded Buffers**: Multiple shards reduce contention and improve parallelism
- **Double Buffering**: Each shard maintains two buffers for continuous writes during flushes
- **Threshold-Based Flushing**: Flushes triggered when 25% of shards are ready
- **Direct I/O**: Bypasses OS page cache for predictable performance
- **Graceful Shutdown**: Ensures all data is flushed before shutdown

## 2. System Architecture

### 2.1 Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           LoggerManager (Entry Point)                 │  │
│  │  - Manages multiple Logger instances                  │  │
│  │  - One logger per event type                          │  │
│  │  - Event name sanitization                            │  │
│  │  - Shared upload channel                              │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Logger (Per Event)                      │  │
│  │  - ShardCollection (multiple shards)                 │  │
│  │  - FileWriter (Direct I/O, rotation)                 │  │
│  │  - Flush worker (threshold-based)                    │  │
│  │  - Statistics tracking                               │  │
│  └──────────────────────────────────────────────────────┘  │
│         │                    │                              │
│         ▼                    ▼                              │
│  ┌──────────────┐   ┌──────────────────┐                   │
│  │ Shard        │   │ FileWriter       │                   │
│  │ Collection   │   │ (Linux/Direct)   │                   │
│  │              │   │                  │                   │
│  │ - Shard 0    │   │ - Direct I/O     │                   │
│  │ - Shard 1    │   │ - File rotation  │                   │
│  │ - Shard N    │   │ - Upload channel │                   │
│  └──────────────┘   └──────────────────┘                   │
│         │                    │                              │
│         │                    ▼                              │
│         │            ┌──────────────────┐                   │
│         │            │ Upload Channel    │                   │
│         │            │ (chan string)     │                   │
│         │            └──────────────────┘                   │
│         │                    │                              │
└─────────┼────────────────────┼──────────────────────────────┘
           │                    │
           │                    ▼
           │            ┌──────────────────┐
           │            │   Uploader       │
           │            │                  │
           │            │ - GCS Client     │
           │            │ - Chunk Manager   │
           │            │ - Retry Logic     │
           │            │ - Statistics      │
           │            └──────────────────┘
           │                    │
           │                    ▼
           │            ┌──────────────────┐
           │            │  Google Cloud     │
           │            │  Storage (GCS)    │
           │            └──────────────────┘
           │
           ▼
    ┌──────────────┐
    │   Shard      │
    │              │
    │ - Buffer A   │
    │ - Buffer B   │
    │ - CAS Swap   │
    │ - mmap       │
    └──────────────┘
```

### 2.2 Data Flow

1. **Write Path**:
   ```
   Application → LoggerManager.LogWithEvent() 
              → Logger.LogBytes() 
              → ShardCollection.Write() (random shard selection)
              → Shard.Write() (CAS atomic write)
              → Buffer (mmap'd memory)
   ```

2. **Flush Path**:
   ```
   Buffer full → Shard.trySwap() 
              → Buffer enqueued to flushChan
              → FlushWorker accumulates buffers
              → Threshold reached (25% shards)
              → FileWriter.WriteVectored() (Pwritev syscall)
              → Direct I/O write to disk
   ```

3. **Upload Path**:
   ```
   File rotation → FileWriter.swapFiles()
                → File path sent to uploadChannel
                → Uploader.uploadWorker()
                → ChunkManager (parallel upload)
                → GCS Compose API
                → Local file deleted
   ```

## 3. Core Components

### 3.1 LoggerManager

**Purpose**: Manages multiple `Logger` instances, one per event type.

**Key Responsibilities**:
- Event name sanitization (filesystem-safe names)
- Logger lifecycle management (create, retrieve, close)
- Shared upload channel coordination
- Optional internal GCS uploader creation

**Key Design Decisions**:
- Uses `sync.Map` for concurrent logger access
- `LoadOrStore` pattern prevents race conditions during logger creation
- Base directory extracted from `LogFilePath` config
- Each event gets its own log file: `{baseDir}/{eventName}.log`

**API**:
```go
type LoggerManager struct {
    LogWithEvent(eventName string, message string)
    LogBytesWithEvent(eventName string, data []byte)
    InitializeEventLogger(eventName string) error
    CloseEventLogger(eventName string) error
    HasEventLogger(eventName string) bool
    ListEventLoggers() []string
    Close() error
    GetAggregatedStats() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64)
    GetUploadStats() *Stats
}
```

### 3.2 Logger

**Purpose**: Core async logger with sharded double buffers.

**Key Responsibilities**:
- Shard collection management
- Flush worker coordination
- Statistics tracking
- Graceful shutdown

**Key Design Decisions**:
- Threshold-based flushing (25% of shards must be ready)
- Single flush semaphore prevents concurrent flushes
- Batch flush using `Pwritev` for multiple buffers
- Lock-free write path using atomic CAS operations

**Internal Structure**:
- `ShardCollection`: Manages multiple shards
- `FileWriter`: Handles Direct I/O writes and rotation
- `flushChan`: Buffers queued for flush
- `flushWorker`: Background goroutine accumulating buffers

**Flush Threshold Logic**:
- Flush worker accumulates buffers in a list
- Tracks unique shards (by shard ID)
- When unique shard count >= threshold (25% of total shards), triggers flush
- Prevents flushing too frequently while ensuring timely writes

### 3.3 ShardCollection

**Purpose**: Manages a collection of shards for parallel writes.

**Key Responsibilities**:
- Shard creation and lifecycle
- Random shard selection for load distribution
- Threshold calculation (25% of shards)

**Key Design Decisions**:
- Random shard selection reduces contention hotspots
- Each shard has independent double buffers
- Threshold fixed at 25% of shards (configurable via code)

**Shard Selection**:
```go
shardIdx := rand.IntN(sc.numShards)  // Random selection
shard := sc.shards[shardIdx]
```

### 3.4 Shard

**Purpose**: Single shard with double buffer for lock-free writes.

**Key Responsibilities**:
- Double buffer management (Buffer A and Buffer B)
- Atomic buffer swapping using CAS
- Lock-free write operations
- Inflight write tracking

**Key Design Decisions**:
- **Double Buffering**: While one buffer is being flushed, the other accepts writes
- **CAS Swap**: Atomic pointer swap prevents race conditions
- **mmap Allocation**: Anonymous mmap for zero-copy buffer access
- **Per-Shard Semaphore**: Coordinates swap operations (prevents multiple concurrent swaps)

**Buffer Structure**:
- 8-byte header: `[capacity (4 bytes)][validDataBytes (4 bytes)]`
- Data section: Variable-length log entries with 4-byte length prefix

**Write Flow**:
1. Get active buffer pointer (atomic load)
2. Reserve space using CAS on offset
3. Write length prefix + data
4. Check if buffer is 90% full → trigger swap
5. Swap pushes inactive buffer to flush channel

### 3.5 FileWriter

**Purpose**: Handles Direct I/O writes and file rotation.

**Key Responsibilities**:
- Direct I/O file operations (`O_DIRECT`, `O_DSYNC`)
- Size-based file rotation
- File preallocation using `fallocate`
- Upload channel notification

**Key Design Decisions**:
- **Direct I/O**: Bypasses OS page cache for predictable latency
- **Proactive Rotation**: Creates next file at 90% capacity
- **Vectored I/O**: Uses `Pwritev` for batch writes
- **File Naming**: Timestamped files: `{baseName}_{YYYY-MM-DD_HH-MM-SS}.log`

**Rotation Logic**:
- Rotation triggered when file size >= `MaxFileSize`
- Proactive creation at 90% capacity
- Atomic swap using mutex
- Completed files sent to upload channel

**Direct I/O Requirements**:
- Buffer alignment: 4096 bytes (4KB)
- File offset alignment: 4096 bytes
- Uses `unix.Pwritev` for vectored writes

### 3.6 Uploader

**Purpose**: Uploads completed log files to GCS.

**Key Responsibilities**:
- GCS client management
- Parallel chunk uploads
- Multi-level compose (handles >32 chunks)
- Retry logic with exponential backoff
- Statistics tracking

**Key Design Decisions**:
- **Parallel Chunks**: Uploads chunks concurrently
- **Chunk Manager**: Handles GCS 32-chunk compose limit
- **Retry Logic**: Configurable retries with delay
- **gRPC Pool**: Connection pooling for better performance
- **Cleanup**: Deletes local files after successful upload

**Upload Flow**:
1. Read entire file into memory
2. Split into chunks (default 32MB)
3. Upload chunks in parallel
4. Compose chunks into final object (handles >32 chunks)
5. Verify final object size
6. Delete local file

**Chunk Compose Strategy**:
- If chunks <= 32: Single compose operation
- If chunks > 32: Multi-level compose
  - Compose groups of 32 into intermediate objects
  - Recursively compose intermediate objects
  - Cleanup intermediate objects after success

### 3.7 ChunkManager

**Purpose**: Manages GCS compose operations with 32-chunk limit handling.

**Key Responsibilities**:
- Single-level compose (≤32 chunks)
- Multi-level compose (>32 chunks)
- Intermediate object cleanup

**Key Design Decisions**:
- GCS compose API limit: 32 chunks per operation
- Multi-level compose for large files
- Cleanup on error to prevent orphaned chunks

## 4. Concurrency Model

### 4.1 Write Path (Hot Path)

**Lock-Free Design**:
- Uses atomic CAS operations for offset updates
- No mutexes on write path
- Per-shard semaphore only for swap coordination

**Concurrency Guarantees**:
- Multiple goroutines can write concurrently
- CAS ensures atomic offset updates
- Inflight counter tracks concurrent writes

### 4.2 Flush Path

**Coordination**:
- Single flush semaphore prevents concurrent flushes
- Flush worker accumulates buffers before batch write
- Threshold-based triggering reduces flush frequency

**Buffer Safety**:
- Mutex protects buffer data access during flush
- Inflight counter ensures writes complete before flush
- Timeout mechanism prevents indefinite waits

### 4.3 Swap Coordination

**Per-Shard Semaphore**:
- Each shard has its own semaphore (buffer size 1)
- Only one swap operation per shard at a time
- CAS on `swapping` flag provides additional protection

**Swap Flow**:
1. Check `swapping` flag (CAS)
2. Acquire per-shard semaphore
3. Atomically swap active buffer pointer
4. Push inactive buffer to flush channel
5. Release semaphore

## 5. Performance Optimizations

### 5.1 Lock-Free Operations

- **Atomic CAS**: Offset updates use `CompareAndSwap`
- **Atomic Pointers**: Buffer pointer swaps are atomic
- **Zero Mutexes**: Hot path has no mutex contention

### 5.2 Memory Management

- **mmap Buffers**: Anonymous mmap for zero-copy access
- **Zero Allocation**: String-to-bytes conversion uses `unsafe`
- **Buffer Reuse**: Double buffers are reused, not reallocated

### 5.3 I/O Optimizations

- **Direct I/O**: Bypasses OS page cache
- **Vectored I/O**: Batch writes using `Pwritev`
- **File Preallocation**: `fallocate` reduces fragmentation
- **Proactive Rotation**: Next file created before rotation

### 5.4 Upload Optimizations

- **Parallel Chunks**: Concurrent chunk uploads
- **gRPC Pool**: Connection pooling
- **Chunked Compose**: Handles large files efficiently

## 6. Configuration

### 6.1 Logger Configuration

```go
type Config struct {
    BufferSize          int           // Total buffer size (default: 64MB)
    NumShards           int           // Number of shards (default: 8)
    LogFilePath         string        // Base log file path (required)
    MaxFileSize         int64         // Max file size before rotation (0 = disabled)
    PreallocateFileSize int64         // Preallocation size (0 = disabled)
    FlushInterval       time.Duration // Periodic flush interval (default: 10s)
    FlushTimeout        time.Duration // Write completion timeout (default: 10ms)
    UploadChannel       chan<- string // Optional: custom upload channel
    GCSUploadConfig     *GCSUploadConfig // Optional: GCS upload config
}
```

### 6.2 GCS Upload Configuration

```go
type GCSUploadConfig struct {
    Bucket              string        // GCS bucket name (required)
    ObjectPrefix        string        // Object prefix (e.g., "logs/event1/")
    ChunkSize           int           // Chunk size (default: 32MB)
    MaxChunksPerCompose int           // Max chunks per compose (default: 32)
    MaxRetries          int           // Max retry attempts (default: 3)
    RetryDelay          time.Duration // Retry delay (default: 5s)
    GRPCPoolSize        int           // gRPC connection pool size (default: 64)
    ChannelBufferSize   int           // Upload channel buffer (default: 100)
}
```

## 7. Error Handling

### 7.1 Write Errors

- **Buffer Full**: Logs are dropped, `DroppedLogs` counter incremented
- **Logger Closed**: Logs are dropped gracefully
- **Write Failures**: Retry mechanism with timeout

### 7.2 Flush Errors

- **Flush Failures**: Tracked in `FlushErrors` counter
- **Buffer Reset**: Buffers are reset even on error to prevent deadlock
- **Error Logging**: Errors logged with context

### 7.3 Upload Errors

- **Retry Logic**: Configurable retries with exponential backoff
- **Failed Uploads**: Tracked in `Failed` counter
- **Cleanup**: Temporary chunks cleaned up on error
- **Non-Blocking**: Upload failures don't block logging

## 8. Statistics and Monitoring

### 8.1 Logger Statistics

- `TotalLogs`: Total log attempts
- `DroppedLogs`: Logs dropped (buffer full, closed, etc.)
- `BytesWritten`: Total bytes written to buffers
- `Flushes`: Number of flush operations
- `FlushErrors`: Number of flush failures
- `FlushQueueDepth`: Current flush queue depth
- `BlockedSwaps`: Swaps that blocked waiting for flush

### 8.2 Flush Metrics

- `AvgFlushDuration`: Average flush duration
- `MaxFlushDuration`: Maximum flush duration
- `AvgWriteDuration`: Average write duration
- `MaxWriteDuration`: Maximum write duration
- `AvgPwritevDuration`: Average Pwritev syscall duration
- `MaxPwritevDuration`: Maximum Pwritev duration
- `WritePercent`: Write duration as % of flush duration
- `PwritevPercent`: Pwritev duration as % of flush duration

### 8.3 Upload Statistics

- `TotalFiles`: Total files processed
- `Successful`: Successful uploads
- `Failed`: Failed uploads
- `TotalBytes`: Total bytes uploaded
- `TotalDuration`: Total upload duration
- `LastUploadTime`: Last successful upload time
- `MinUploadDuration`: Minimum upload duration
- `MaxUploadDuration`: Maximum upload duration
- `AvgUploadDuration`: Average upload duration

## 9. File Format

### 9.1 Log File Structure

Each log file contains multiple shard buffers, each with:

```
[Shard Header (8 bytes)]
├── Capacity (4 bytes, little-endian uint32)
├── Valid Data Bytes (4 bytes, little-endian uint32)
└── [Data Section]
    ├── Entry 1: [Length (4 bytes)][Data (N bytes)]
    ├── Entry 2: [Length (4 bytes)][Data (M bytes)]
    └── ...
```

### 9.2 Entry Format

- **Length Prefix**: 4-byte little-endian uint32 indicating data length
- **Data**: Raw log data (no encoding, can be JSON, text, binary, etc.)

## 10. Deployment Considerations

### 10.1 Prerequisites

- **Linux**: Direct I/O requires Linux (uses `O_DIRECT` flag)
- **Filesystem**: Ext4 or similar (4096-byte alignment)
- **GCS Credentials**: Application Default Credentials or service account
- **Disk Space**: Sufficient space for log files and buffers

### 10.2 Resource Requirements

- **Memory**: Buffer size × 2 (double buffering) + overhead
- **Disk**: Log files + rotation space
- **Network**: Bandwidth for GCS uploads
- **CPU**: Minimal (lock-free design)

### 10.3 Production Recommendations

- **Buffer Size**: 64MB - 256MB depending on throughput
- **Num Shards**: 8-16 shards for optimal parallelism
- **Max File Size**: 100MB - 1GB depending on rotation frequency
- **GCS Chunk Size**: 32MB for optimal upload performance
- **gRPC Pool Size**: 64-128 connections for high throughput

## 11. Limitations and Future Enhancements

### 11.1 Current Limitations

- **Linux Only**: Direct I/O requires Linux-specific syscalls
- **Single Process**: Not designed for multi-process scenarios
- **Memory Usage**: Entire file loaded into memory for upload
- **No Compression**: Logs are not compressed before upload

### 11.2 Future Enhancements

- **Compression**: Add gzip compression before upload
- **Multi-Process**: Support for shared memory buffers
- **Streaming Upload**: Stream large files instead of loading into memory
- **Metrics Export**: Prometheus/OpenTelemetry integration
- **Configurable Threshold**: Make flush threshold configurable
- **Windows Support**: Fallback to standard I/O on Windows

## 12. Security Considerations

### 12.1 Input Validation

- **Event Names**: Sanitized to prevent path traversal
- **File Paths**: Validated and sanitized
- **GCS Credentials**: Stored securely (not in code)

### 12.2 Access Control

- **File Permissions**: Log files created with 0644 permissions
- **GCS IAM**: Proper IAM roles for bucket access
- **Network Security**: TLS for GCS communication

## 13. Testing Strategy

### 13.1 Unit Tests

- Shard operations (write, swap, reset)
- ShardCollection operations
- Logger operations
- FileWriter operations
- ChunkManager compose logic

### 13.2 Integration Tests

- End-to-end logging flow
- File rotation integrity
- GCS upload verification
- Concurrent write scenarios
- File format verification

### 13.3 Performance Tests

- Throughput benchmarks
- Latency measurements
- Concurrent write stress tests
- Flush performance analysis

---

**Document Version**: 1.0  
**Last Updated**: 2024  
**Author**: BharatMLStack Team
