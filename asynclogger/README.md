# Async Logger

Async logger with Direct I/O support for Go applications.

> **⚠️ Important: File Rotation Not Yet Implemented**
> 
> **File rotation is not implemented in the current version.** Log files will grow indefinitely until manually rotated or the application is restarted. **must implement file rotation before production deployment** to prevent disk space issues.

## Overview

This package provides a lock-free async logger using a Sharded Double Buffer CAS (Compare-and-Swap) architecture with Linux Direct I/O for predictable write performance.

## Features

- **Zero-Allocation API**: `LogBytes()` accepts pre-allocated buffers to reduce GC pressure
- **Direct I/O**: Bypasses OS page cache for predictable writes (Linux only)
- **Lock-Free Writes**: Uses atomic CAS for contention-free logging
- **Default Configuration**: 64MB buffer, 8 shards
- **Thread-Safe**: Supports concurrent logging from multiple goroutines
- **Graceful Shutdown**: Ensures all logs are flushed before exit
- **Statistics**: Built-in metrics for monitoring
- **Dual API**: Choose between convenience (`Log`) and low-allocation (`LogBytes`)
- **Multi-Event Logging**: `LoggerManager` manages multiple event-specific loggers with automatic file separation

## Architecture

### Sharded Double Buffer CAS

```
┌─────────────────────────────────────────────┐
│           Active Buffer Set                 │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐  │
│  │Shard 0│ │Shard 1│ │Shard 2│ │Shard 3│  │
│  └───────┘ └───────┘ └───────┘ └───────┘  │
│         ... (8 shards total) ...            │
└─────────────────────────────────────────────┘
                    ↕ Atomic CAS Swap
┌─────────────────────────────────────────────┐
│          Flushing Buffer Set                │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐  │
│  │Shard 0│ │Shard 1│ │Shard 2│ │Shard 3│  │
│  └───────┘ └───────┘ └───────┘ └───────┘  │
│         ... (8 shards total) ...            │
└─────────────────────────────────────────────┘
                    ↓
            ┌───────────────┐
            │  Direct I/O   │
            │  (O_DIRECT)   │
            └───────────────┘
                    ↓
            ┌───────────────┐
            │   Log File    │
            └───────────────┘
```

### Key Components

1. **Buffer**: 512-byte aligned buffer for Direct I/O compatibility
2. **Shard**: Independent buffer with mutex protection
3. **BufferSet**: Collection of shards for parallel writes
4. **Logger**: Orchestrates double buffering, swapping, and flushing

## Log File Structure

The logger writes logs to disk with a two-level header system for data integrity and recovery:

### 1. Shard Header (8 bytes)

Each shard's data is prefixed with an 8-byte header:

```
┌──────────────────┬──────────────────────┐
│ Capacity (4 bytes)│ Valid Data (4 bytes)│
│   uint32 LE       │   uint32 LE         │
└──────────────────┴──────────────────────┘
```

- **Bytes 0-3**: Shard buffer capacity (total allocated size)
- **Bytes 4-7**: Valid data bytes (actual data written)

**Purpose:**
- Boundary validation (distinguish valid data from padding)
- Recovery from incomplete flushes
- Direct I/O alignment handling (buffers are 512-byte aligned)

### 2. Log Entry Header (4 bytes)

Each log entry is prefixed with a 4-byte length header:

```
┌─────────────────┬──────────────────────┐
│ Length (4 bytes) │ Log Data (N bytes)   │
│  uint32 LE      │  Actual log message  │
└─────────────────┴──────────────────────┘
```

- **Bytes 0-3**: Length of log data (little-endian uint32)
- **Bytes 4+**: Actual log message content

**Purpose:**
- Boundary detection (know where each entry starts/ends)
- Corruption detection (invalid length values)
- Recovery support (identify complete vs incomplete entries)

### Complete File Layout Example

```
File Structure:
┌─────────────────────────────────────────────────────────────┐
│ Shard 0 Combined Buffer (8 bytes header + 512KB data + padding) │
│ ├─ Header (8 bytes):                                        │
│ │  ├─ Capacity: 1MB                                         │
│ │  └─ Valid Data: 512KB                                     │
│ ├─ Data (512KB valid):                                      │
│ │  ├─ [4 bytes: length] "Log entry 1\n"                    │
│ │  ├─ [4 bytes: length] "Log entry 2\n"                    │
│ │  └─ ...                                                   │
│ └─ Padding (to 512-byte alignment, ignored by readers)    │
├─────────────────────────────────────────────────────────────┤
│ Shard 1 Combined Buffer (8 bytes header + 256KB data + padding) │
│ ├─ Header (8 bytes):                                        │
│ │  ├─ Capacity: 1MB                                         │
│ │  └─ Valid Data: 256KB                                     │
│ ├─ Data (256KB valid):                                      │
│ │  └─ [4 bytes: length] "Log entry N\n"                    │
│ └─ Padding (to 512-byte alignment, ignored by readers)     │
└─────────────────────────────────────────────────────────────┘
```

**Key Points:**
- Headers and shard data are written together as a single aligned buffer
- Padding occurs only at the END of each combined buffer (after shard data)
- Headers and data are contiguous - no padding between them
- Readers use `validDataBytes` from the header to know where valid data ends

**Reading Process:**
1. Read 8-byte shard header → extract capacity and valid data size
2. Read shard data starting immediately after header, up to `validDataBytes`
   - Padding at the end is automatically ignored since readers stop at `validDataBytes`
3. Parse log entries:
   - Read 4-byte length prefix
   - Read `length` bytes of log data
   - Repeat until end of valid data
4. Skip to next 512-byte aligned boundary (if needed for Direct I/O) and read next shard header

## Installation

### Using Go Modules

Add the module to your `go.mod`:

```bash
go get github.com/Meesho/BharatMLStack/asynclogger
```

Or add to your `go.mod` file:

```go
require github.com/Meesho/BharatMLStack/asynclogger v0.0.0
```

Then import in your code:

```go
import "github.com/Meesho/BharatMLStack/asynclogger"
```

### Local Development

If using the module locally without publishing:

```bash
# In your project's go.mod, use a replace directive
replace github.com/Meesho/BharatMLStack/asynclogger => ../asynclogger
```

## Usage

### Complete Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/Meesho/BharatMLStack/asynclogger"
)

func main() {
    // Create logger configuration
    config := asynclogger.DefaultConfig("/var/log/myapp.log")
    
    // Optionally customize configuration
    config.BufferSize = 64 * 1024 * 1024  // 64MB
    config.NumShards = 8
    config.FlushInterval = 10 * time.Second
    config.FlushTimeout = 10 * time.Millisecond

    // Create logger
    logger, err := asynclogger.New(config)
    if err != nil {
        log.Fatalf("Failed to create logger: %v", err)
    }
    defer func() {
        if err := logger.Close(); err != nil {
            log.Printf("Error closing logger: %v", err)
        }
    }()

    // Start monitoring goroutine
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go monitorLogger(ctx, logger)

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Log some messages
    logger.Log("Application started")
    
    // Simulate application work
    for i := 0; i < 100; i++ {
        logger.Log("Processing request")
        time.Sleep(100 * time.Millisecond)
    }

    // Wait for shutdown signal
    <-sigChan
    log.Println("Shutting down...")
}

func monitorLogger(ctx context.Context, logger *asynclogger.Logger) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            total, dropped, bytesWritten, flushes, flushErrors, _ := logger.GetStatsSnapshot()
            if total > 0 {
                dropRate := float64(dropped) / float64(total) * 100
                log.Printf("Logger stats: total=%d, dropped=%d (%.4f%%), bytes=%d, flushes=%d, errors=%d",
                    total, dropped, dropRate, bytesWritten, flushes, flushErrors)
                
                if dropRate > 0.01 {
                    log.Printf("WARNING: High drop rate detected: %.4f%%", dropRate)
                }
            }
        }
    }
}
```

### Basic Usage

```go
package main

import (
    "log"
    "github.com/Meesho/BharatMLStack/asynclogger"
)

func main() {
    // Create logger with optimal defaults
    config := asynclogger.DefaultConfig("/var/log/app.log")
    logger, err := asynclogger.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close() // Ensures all logs are flushed

    // Log messages using convenience API
    logger.Log("Application started")
    logger.Log("Processing request")
    logger.Log("Request completed")
}
```

### Low-Allocation Logging with LogBytes

To reduce allocations and GC pressure, use the `LogBytes` API with reusable buffers:

```go
import (
    "strconv"
    "github.com/Meesho/BharatMLStack/asynclogger"
)

// Worker goroutine with reusable buffer (zero allocations)
type Worker struct {
    logger *asynclogger.Logger
    msgBuf [256]byte  // Reused for every log
}

func (w *Worker) ProcessRequest(userID int, status string) {
    // Format message directly into buffer
    n := copy(w.msgBuf[:], "Processing request for user ")
    n += copy(w.msgBuf[n:], strconv.Itoa(userID))
    n += copy(w.msgBuf[n:], " status=")
    n += copy(w.msgBuf[n:], status)
    w.msgBuf[n] = '\n'
    
    // Log with minimal allocations
    w.logger.LogBytes(w.msgBuf[:n+1])
}
```

**Note:** The `LogBytes` API avoids string-to-byte conversions when you provide a reusable buffer. The `Log` API uses `unsafe` to avoid allocations but still requires string creation.

### Using sync.Pool for Message Buffers

```go
import (
    "fmt"
    "sync"
    "github.com/Meesho/BharatMLStack/asynclogger"
)

var msgPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 256)
        return &buf
    },
}

func logRequest(logger *asynclogger.Logger, userID int) {
    bufPtr := msgPool.Get().(*[]byte)
    buf := *bufPtr
    defer msgPool.Put(bufPtr)
    
    // Format into pooled buffer
    n := copy(buf, fmt.Sprintf("Request for user %d\n", userID))
    logger.LogBytes(buf[:n])
}
```

### Monitoring Statistics

```go
// Get snapshot of current statistics
totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps := logger.GetStatsSnapshot()

fmt.Printf("Total Logs: %d\n", totalLogs)
fmt.Printf("Dropped Logs: %d (%.4f%%)\n", droppedLogs, float64(droppedLogs)/float64(totalLogs)*100)
fmt.Printf("Bytes Written: %d\n", bytesWritten)
fmt.Printf("Flushes: %d\n", flushes)
fmt.Printf("Flush Errors: %d\n", flushErrors)
fmt.Printf("Buffer Swaps: %d\n", setSwaps)

// Get detailed flush metrics
flushMetrics := logger.GetFlushMetrics()
fmt.Printf("Avg Flush Time: %.2fms\n", float64(flushMetrics.AvgFlushDuration.Microseconds())/1000.0)
fmt.Printf("Max Flush Time: %.2fms\n", float64(flushMetrics.MaxFlushDuration.Microseconds())/1000.0)

// Get per-shard statistics
shardStats := logger.GetShardStats()
for _, stat := range shardStats {
    fmt.Printf("Shard %d: %.1f%% utilized (%d writes)\n", 
        stat.ShardID, stat.UtilizationPct, stat.WriteCount)
}
```

### Periodic Monitoring Example

```go
// Monitor drop rate periodically
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        total, dropped, _, _, _, _ := logger.GetStatsSnapshot()
        if total > 0 {
            dropRate := float64(dropped) / float64(total) * 100
            if dropRate > 0.01 {
                log.Printf("WARNING: High drop rate: %.4f%%", dropRate)
            }
        }
    }
}()
```

## LoggerManager: Multi-Event Logging

The `LoggerManager` provides a convenient way to manage multiple event-specific loggers, where each event type writes to its own log file. This is useful for applications that need to separate logs by event type (e.g., `payment.log`, `login.log`, `order.log`).

### Features

- **Automatic Logger Creation**: Loggers are created lazily on first use
- **Event Name Sanitization**: Invalid filesystem characters are automatically sanitized
- **Thread-Safe**: Uses `sync.Map` for lock-free concurrent access
- **Per-Event Statistics**: Get statistics for individual events or aggregated across all events
- **Dynamic Management**: Initialize and close event loggers at runtime

### Basic Usage

```go
package main

import (
    "log"
    "github.com/Meesho/BharatMLStack/asynclogger"
)

func main() {
    // Create a LoggerManager with base configuration
    // The base directory is extracted from LogFilePath
    config := asynclogger.DefaultConfig("/var/log/myapp/base.log")
    manager, err := asynclogger.NewLoggerManager(config)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close() // Closes all event loggers

    // Log to different events (creates loggers automatically)
    manager.LogWithEvent("payment", "Payment processed: $100")
    manager.LogWithEvent("login", "User logged in: user123")
    manager.LogWithEvent("order", "Order created: #12345")

    // Files created:
    // - /var/log/myapp/payment.log
    // - /var/log/myapp/login.log
    // - /var/log/myapp/order.log
}
```

### Zero-Allocation Logging

```go
// Use LogBytesWithEvent for high-performance logging
data := []byte("Payment processed: $100\n")
manager.LogBytesWithEvent("payment", data)
```

### Event Logger Management

```go
// Initialize a logger explicitly (useful for webhook-driven setup)
err := manager.InitializeEventLogger("payment")
if err != nil {
    log.Printf("Failed to initialize payment logger: %v", err)
}

// Check if an event logger exists
if manager.HasEventLogger("payment") {
    log.Println("Payment logger is active")
}

// List all active event loggers
events := manager.ListEventLoggers()
log.Printf("Active events: %v", events)

// Close a specific event logger
err = manager.CloseEventLogger("payment")
if err != nil {
    log.Printf("Failed to close payment logger: %v", err)
}
```

### Statistics and Monitoring

```go
// Get aggregated statistics across all events
totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps := manager.GetStatsSnapshot()
log.Printf("Total logs: %d, Dropped: %d, Bytes: %d", totalLogs, droppedLogs, bytesWritten)

// Get statistics for a specific event
total, dropped, bytes, flushes, errors, swaps, err := manager.GetEventStats("payment")
if err != nil {
    log.Printf("Error getting payment stats: %v", err)
} else {
    log.Printf("Payment logs: %d, Dropped: %d", total, dropped)
}

// Get aggregated flush metrics
metrics := manager.GetAggregatedFlushMetrics()
log.Printf("Avg flush duration: %v", metrics.AvgFlushDuration)
```

### Event Name Sanitization

Event names are automatically sanitized to ensure valid filenames:

- Invalid characters (`/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`) are replaced with `_`
- Spaces are replaced with `_`
- Names are truncated to 255 characters
- Empty names are rejected

```go
// These event names are automatically sanitized:
manager.LogWithEvent("payment/event", "test")  // Creates: payment_event.log
manager.LogWithEvent("login event", "test")    // Creates: login_event.log
manager.LogWithEvent("order*test", "test")      // Creates: order_test.log
```

### Complete Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/Meesho/BharatMLStack/asynclogger"
)

func main() {
    // Create LoggerManager
    config := asynclogger.DefaultConfig("/var/log/myapp/base.log")
    manager, err := asynclogger.NewLoggerManager(config)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    // Initialize event loggers
    manager.InitializeEventLogger("payment")
    manager.InitializeEventLogger("login")

    // Start monitoring
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go monitorManager(ctx, manager)

    // Log events
    manager.LogWithEvent("payment", "Payment $100 processed")
    manager.LogWithEvent("login", "User alice logged in")
    manager.LogWithEvent("order", "Order #12345 created")

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
}

func monitorManager(ctx context.Context, manager *asynclogger.LoggerManager) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            total, dropped, bytesWritten, _, _, _ := manager.GetStatsSnapshot()
            if total > 0 {
                dropRate := float64(dropped) / float64(total) * 100
                log.Printf("Manager stats: total=%d, dropped=%d (%.4f%%), bytes=%d",
                    total, dropped, dropRate, bytesWritten)
            }

            // List active events
            events := manager.ListEventLoggers()
            log.Printf("Active events: %v", events)
        }
    }
}
```

## Configuration Guide

### Default Configuration

```go
config := asynclogger.DefaultConfig("/var/log/app.log")
// BufferSize:    64MB  (baseline configuration)
// NumShards:     8     (optimal thread-to-shard ratio 1:1)
// FlushInterval: 10s   (balance between latency and throughput)
// FlushTimeout:  10ms  (wait for writes to complete)
```

### Custom Configuration

```go
import (
    "time"
    "github.com/Meesho/BharatMLStack/asynclogger"
)

config := asynclogger.Config{
    LogFilePath:   "/var/log/myapp.log",
    BufferSize:    64 * 1024 * 1024,  // 64MB
    NumShards:     8,
    FlushInterval: 10 * time.Second,
    FlushTimeout:  10 * time.Millisecond,
}

logger, err := asynclogger.New(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

## Configuration Tuning Guide

This guide helps you choose optimal configuration values based on your workload characteristics.

### Step 1: Determine Your Workload Characteristics

Before tuning, understand your logging patterns:

- **Logging Volume**: How many log messages per second?
- **Message Size**: Average and maximum message size?
- **Concurrency**: How many goroutines log simultaneously?
- **Latency Requirements**: How quickly must logs be on disk?
- **Available Memory**: How much memory can be dedicated to logging?

### Step 2: Choose BufferSize

**Formula**: `BufferSize = (LoggingRate × AvgMessageSize × FlushInterval) × SafetyFactor`

**Guidelines**:

| Workload | Logging Rate | Recommended BufferSize | Reasoning |
|----------|--------------|------------------------|-----------|
| Low volume | < 100 msg/sec | 8-16 MB | Small buffer sufficient, low memory usage |
| Medium volume | 100-1K msg/sec | 32-64 MB | Default works well for most applications |
| High volume | 1K-10K msg/sec | 64-128 MB | Larger buffer reduces swap frequency |
| Very high volume | > 10K msg/sec | 128-256 MB | Maximum buffer to minimize drops |

**Memory Consideration**: Remember that total memory usage is `BufferSize × 2` (double buffering).

**Example Calculation**:
```go
// Example: 5000 messages/sec, 200 bytes avg, 10s flush interval
// BufferSize = (5000 × 200 × 10) × 1.5 (safety factor)
//            = 10MB × 1.5 = 15MB
// Round up to 16MB for alignment

config.BufferSize = 16 * 1024 * 1024 // 16MB
```

**Tuning Tips**:
- Start with default (64MB) and monitor drop rate
- If drop rate > 0.01%, increase BufferSize
- If memory is constrained, decrease BufferSize but expect more frequent flushes
- Minimum practical size: 4MB (below this, frequent swaps hurt performance)

### Step 3: Choose NumShards

We noticed during tests **10-15 concurrent writers per shard** is giving optimal results

**Tuning Tips**:
- Too few shards: High contention, increased CAS retries, dropped logs
- Too many shards: Wasted memory, smaller per-shard buffers
- Monitor shard utilization: `GetShardStats()` should show balanced distribution
- Round-robin distributes evenly, but 1:1 ratio minimizes contention


### Step 4: Choose FlushInterval

In our usecase flush is dominated by buffer filling up than interval

**Tuning Tips**:
- Shorter intervals: More frequent flushes, lower latency, higher I/O overhead
- Longer intervals: Fewer flushes, better throughput, higher memory usage
- Buffer fills may trigger flushes before interval expires
- Monitor flush frequency: `GetStatsSnapshot()` shows actual flush count

### Step 5: Choose FlushTimeout

This functionality may be removed

**Tuning Tips**:
- Too short: Incomplete writes may be flushed (last entry corrupted)
- Too long: Delays flush operations, increases memory usage
- Monitor flush metrics: `GetFlushMetrics()` shows if timeouts occur
- If `writesCompleted < writesStarted` frequently, increase timeout

## Direct I/O

### What is Direct I/O?

Direct I/O (`O_DIRECT`) bypasses the operating system's page cache, writing directly to disk. This provides:

- **Predictable write latency**: No cache eviction surprises
- **Consistent performance**: Not affected by other processes' I/O
- **Lower memory pressure**: No double-buffering in OS cache

### Platform Selection

The logger automatically selects the appropriate I/O implementation using Go build tags:

- **Linux**: Uses `directio_linux.go` with `O_DIRECT` and `O_DSYNC` flags for Direct I/O
- **Non-Linux** (macOS, Windows): Uses `directio_default.go` with standard file I/O (for testing)

The selection happens at compile time via build tags (`//go:build linux` and `//go:build !linux`), so no runtime checks are needed. Both implementations provide the same API, ensuring code compatibility across platforms.

### Requirements

- **Linux only**: Uses `syscall.O_DIRECT` (for production Direct I/O)
- **512-byte alignment**: All buffers and writes are automatically aligned
- **Block device**: Works best with physical disks and SSDs

### Trade-offs

✅ **Pros**:
- Predictable performance
- No cache pollution
- Lower system memory usage

⚠️ **Cons**:
- Higher write latency compared to buffered I/O
- Linux-specific (no macOS/Windows support)

## When to Use This Logger

### Suitable Use Cases

- **High-volume logging**: Applications that generate many log messages per second
- **Predictable I/O**: Applications requiring consistent write latency
- **Linux production environments**: Where Direct I/O is beneficial
- **Low-allocation requirements**: Applications sensitive to GC pressure
- **Concurrent logging**: Multiple goroutines logging simultaneously

### Not Suitable For

- **Low-volume logging**: Standard logging libraries may be simpler
- **Cross-platform applications**: Requires Linux for production Direct I/O
- **Applications needing log rotation**: Must implement externally
- **Structured logging**: This logger writes raw bytes/strings only
- **Log levels/filtering**: No built-in log level support

## Common Pitfalls

### 1. Forgetting to Close the Logger

**Problem**: Logs may not be flushed on application exit.

**Solution**: Always use `defer logger.Close()` immediately after creating the logger.

```go
logger, err := asynclogger.New(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close() // ✅ Always close
```

### 2. Not Monitoring Drop Rate

**Problem**: Logs may be silently dropped without notice.

**Solution**: Periodically check statistics and alert on high drop rates.

```go
total, dropped, _, _, _, _ := logger.GetStatsSnapshot()
if total > 0 && float64(dropped)/float64(total) > 0.01 {
    // Alert: More than 1% of logs are being dropped
}
```

### 3. Using Log() in Hot Paths

**Problem**: String allocations on every log call can cause GC pressure.

**Solution**: Use `LogBytes()` with reusable buffers for high-frequency logging.

```go
// ❌ Avoid in hot paths
logger.Log("frequent message")

// ✅ Better for hot paths
var buf [256]byte
n := copy(buf[:], "frequent message")
logger.LogBytes(buf[:n])
```

### 4. Incorrect Shard Count

**Problem**: Too few shards cause contention; too many waste memory.

**Solution**: Match shard count to expected concurrency (1:1 ratio is optimal).

```go
// For 8 concurrent writers
config.NumShards = 8 // ✅ Good

// For 100 concurrent writers
config.NumShards = 8 // ❌ Too few, will cause contention
config.NumShards = 16 // ✅ Better
```

### 5. Buffer Size Too Small

**Problem**: Frequent buffer swaps and dropped logs.

**Solution**: Size buffers based on logging volume and available memory.

```go
// ❌ Too small for high-volume logging
config.BufferSize = 1 * 1024 * 1024 // 1MB

// ✅ Appropriate for most applications
config.BufferSize = 64 * 1024 * 1024 // 64MB
```

### 6. Not Handling File Rotation

**Problem**: Log files grow indefinitely, filling disk space.

**Solution**: Implement external rotation (logrotate) or application-level rotation before production.

```bash
# Example logrotate configuration
/var/log/myapp.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 app app
    postrotate
        # Signal application to reopen log file (requires implementation)
    endscript
}
```

## Best Practices

### 1. Always Close the Logger

```go
logger, err := asynclogger.New(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()  // Ensures all logs are flushed
```

### 2. Monitor Drop Rate

```go
// Periodically check drop rate
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        total, dropped, _, _, _, _ := logger.GetStatsSnapshot()
        if total > 0 {
            dropRate := float64(dropped) / float64(total) * 100
            if dropRate > 0.01 {
                log.Printf("WARNING: High drop rate: %.4f%%", dropRate)
            }
        }
    }
}()
```

### 3. Size Buffers Appropriately

- **8MB** for most applications (8-20 concurrent writers)
- **16MB** for high-capacity systems (50+ writers)
- **4MB** for resource-constrained environments

### 4. Match Shards to Concurrency

```
Thread-to-Shard Ratio Guidelines:
- 1:1 ratio is optimal (e.g., 8 threads → 8 shards)
- Keep ratio ≤ 2:1 for best performance
- Avoid ratio > 12:1 (causes high contention)
```

### 5. Use LogBytes for Hot Paths

For high-frequency logging paths, use `LogBytes()` with reusable buffers to eliminate GC pressure:

```go
// Good: Zero allocations
var buf [256]byte
n := copy(buf[:], "message\n")
logger.LogBytes(buf[:n])

// Avoid: Allocations on every call
logger.Log("message")  // Creates string allocation
```

## Troubleshooting

### High Drop Rate (>0.01%)

**Symptoms**: Messages being discarded

**Solutions**:
1. Increase buffer size (8MB → 16MB)
2. Increase shard count (8 → 16)
3. Reduce concurrent writers if possible
4. Increase flush interval to reduce overhead

### Performance Issues

**Symptoms**: Slow logging, high latency

**Solutions**:
1. Verify Direct I/O is working (`cat /proc/<pid>/fdinfo/<fd>`)
2. Check disk I/O with `iostat -x 1`
3. Ensure SSD or fast disk is used
4. Monitor CPU usage and system resources

### Flush Errors

**Symptoms**: `flushErrors > 0` in statistics

**Solutions**:
1. Check disk space: `df -h`
2. Verify file permissions: `ls -l /var/log/app.log`
3. Check for disk errors: `dmesg | grep -i error`
4. Ensure Direct I/O is supported on filesystem (ext4, xfs)

## Testing

### Running Tests

Run comprehensive tests:

```bash
cd asynclogger
go test -v -race -cover
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

### Test Coverage

The test suite includes comprehensive coverage for both `Logger` and `LoggerManager`:

#### Configuration Tests
- **`TestConfig_Validate`** (4 sub-tests)
  - Valid config verification
  - Missing log path validation
  - Default value application
  - Shard size validation

#### Logger Basic Functionality Tests
- **`TestLogger_BasicLogging`** - Basic logging with file creation
- **`TestLogger_ConcurrentWrites`** - High concurrency (8, 50, 100 goroutines)
- **`TestLogger_BufferFillingAndSwapping`** - Buffer swap mechanism
- **`TestLogger_GracefulShutdown`** - Shutdown flush verification
- **`TestLogger_DoubleClose`** - Error handling (double close)
- **`TestLogger_Statistics`** - Statistics tracking verification
- **`TestLogger_MessageWithoutNewline`** - Message format handling

#### Buffer-Level Tests
- **`TestBuffer_Write`** - Buffer write with header reservation
- **`TestBuffer_FillAndFlush`** - Buffer fill threshold (90%)

#### Shard-Level Tests
- **`TestShard_ConcurrentWrites`** - Shard-level thread safety

#### BufferSet-Level Tests
- **`TestBufferSet_RoundRobin`** - Round-robin shard selection
- **`TestBufferSet_HasData`** - Data presence detection

#### API Tests
- **`TestLogger_LogBytes`** - LogBytes API functionality
- **`TestLogger_LogBytes_ZeroAllocation`** - Zero-allocation usage pattern
- **`TestLogger_LogBytes_ConcurrentWithReuse`** - Concurrent LogBytes with buffer reuse
- **`TestLogger_LogString_BackwardCompatible`** - String API compatibility
- **`TestLogger_MixedStringAndBytes`** - Mixed API usage

#### File Format Verification Tests
- **`TestLogger_FileFormatVerification`** - Comprehensive file format validation
  - ✅ 8-byte shard header structure verification
  - ✅ Data starts immediately after header (no padding)
  - ✅ Log entries have correct 4-byte length prefixes
  - ✅ **Data correctness** (what is written matches what is read)
  - ✅ End-to-end file format validation

#### LoggerManager Tests
- **`TestNewLoggerManager`** (4 sub-tests) - Manager creation and configuration
- **`TestSanitizeEventName`** (11 sub-tests) - Event name sanitization and validation
- **`TestLoggerManager_LogBytesWithEvent`** (3 sub-tests) - Logging with event separation
- **`TestLoggerManager_LogWithEvent`** - String-based event logging
- **`TestLoggerManager_InitializeEventLogger`** (4 sub-tests) - Explicit logger initialization
- **`TestLoggerManager_CloseEventLogger`** (4 sub-tests) - Event logger closing
- **`TestLoggerManager_HasEventLogger`** (3 sub-tests) - Logger existence checks
- **`TestLoggerManager_ListEventLoggers`** (3 sub-tests) - Listing active loggers
- **`TestLoggerManager_Close`** (3 sub-tests) - Manager shutdown
- **`TestLoggerManager_GetStatsSnapshot`** (2 sub-tests) - Aggregated statistics
- **`TestLoggerManager_GetEventStats`** (3 sub-tests) - Per-event statistics
- **`TestLoggerManager_ConcurrentAccess`** (3 sub-tests) - Concurrent operations
- **`TestLoggerManager_EventNameSanitizationInFileNames`** - File name validation
- **`TestLoggerManager_LoadOrStoreRaceCondition`** - Race condition handling

### Test Coverage Areas

- ✅ **Configuration validation** - All config scenarios tested
- ✅ **Basic logging functionality** - Core logging operations
- ✅ **Concurrent writes** - 8, 50, and 100 goroutines
- ✅ **Buffer management** - Filling, swapping, flushing
- ✅ **Graceful shutdown** - Data flush on close
- ✅ **Error handling** - Double close, invalid config
- ✅ **Statistics tracking** - All metrics verified
- ✅ **Buffer operations** - Write, fill, flush
- ✅ **Shard operations** - Concurrent writes, thread safety
- ✅ **BufferSet operations** - Round-robin, data detection
- ✅ **API compatibility** - Log vs LogBytes, mixed usage
- ✅ **File format correctness** - Header structure, data layout
- ✅ **Data integrity** - Round-trip verification (write → read)
- ✅ **LoggerManager functionality** - Multi-event logging, lazy initialization, event management
- ✅ **Event name sanitization** - Invalid character handling, filename validation
- ✅ **Concurrent LoggerManager access** - Thread-safe operations, race condition handling
- ✅ **Per-event statistics** - Individual and aggregated metrics

## Platform Support

- ✅ **Linux**: Full support with Direct I/O
- ❌ **macOS**: Not supported (requires different Direct I/O implementation)
- ❌ **Windows**: Not supported

## Reading Log Files

The logger writes logs in a binary format. Here's an example of how to read and parse log files:

```go
package main

import (
    "encoding/binary"
    "fmt"
    "io"
    "os"
)

func ReadLogFile(path string) ([]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var messages []string
    fileOffset := int64(0)
    const alignmentSize = 512

    for {
        // Read shard header (8 bytes)
        header := make([]byte, 8)
        n, err := file.ReadAt(header, fileOffset)
        if err == io.EOF || n < 8 {
            break
        }
        if err != nil {
            return nil, err
        }

        // Parse header
        capacity := binary.LittleEndian.Uint32(header[0:4])
        validDataBytes := binary.LittleEndian.Uint32(header[4:8])

        // Skip empty shards
        if capacity == 0 || validDataBytes == 0 {
            break
        }

        // Read shard data (starts immediately after header)
        fileOffset += 8
        shardData := make([]byte, validDataBytes)
        n, err = file.ReadAt(shardData, fileOffset)
        if err != nil && err != io.EOF {
            return nil, err
        }
        if n == 0 {
            break
        }

        // Parse log entries from shard data
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
            if entryLength == 0 || offset+int(entryLength) > len(shardData) {
                break // Invalid or incomplete entry
            }

            // Extract log entry
            entryData := shardData[offset : offset+int(entryLength)]
            messages = append(messages, string(entryData))
            offset += int(entryLength)
        }

        // Move to next shard (aligned to 512-byte boundary)
        shardTotalSize := 8 + int(validDataBytes)
        alignedShardSize := ((shardTotalSize + alignmentSize - 1) / alignmentSize) * alignmentSize
        fileOffset += int64(alignedShardSize) // Move to start of next shard
    }

    return messages, nil
}

func main() {
    messages, err := ReadLogFile("/var/log/app.log")
    if err != nil {
        fmt.Printf("Error reading log file: %v\n", err)
        return
    }

    for _, msg := range messages {
        fmt.Println(msg)
    }
}
```

## Limitations

### Current Limitations

1. **File Rotation**: Not supported. Log files grow indefinitely. Implement external rotation (e.g., `logrotate`) or application-level rotation before production use.

2. **Maximum Message Size**: Limited by shard capacity. Each message must fit within a single shard buffer. With default configuration (64MB total, 8 shards), each shard is ~8MB. Messages larger than `(BufferSize / NumShards) - 8 bytes` (accounting for header reservation) will be dropped. Calculate maximum message size as: `(BufferSize / NumShards) - 8 - 4` (subtract header reservation and length prefix).

3. **Platform Support**: Production Direct I/O only works on Linux. Non-Linux platforms use standard file I/O (suitable for testing only).

4. **Single File per Logger**: Each `Logger` instance writes to a single file. Use `LoggerManager` for multi-file logging with event separation.

5. **No Compression**: Log files are written uncompressed. Consider post-processing for compression.

6. **No Encryption**: Logs are written in plain text. Add encryption at the application level if needed.

### Memory Usage

The logger uses approximately:
- **Buffer Memory**: `BufferSize * 2` (double buffering: active + flushing sets)
- **Per-Shard Overhead**: ~100 bytes per shard
- **Total**: ~128MB for default configuration (64MB × 2)

Adjust `BufferSize` based on available memory and logging volume.

## API Reference

### Logger Methods

- `New(config Config) (*Logger, error)` - Create a new logger instance
- `Log(message string)` - Log a string message (convenience API)
- `LogBytes(data []byte)` - Log raw bytes (high-performance API)
- `Close() error` - Gracefully shutdown and flush all logs
- `GetStatsSnapshot() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64)` - Get current statistics
- `GetFlushMetrics() FlushMetrics` - Get detailed flush performance metrics
- `GetShardStats() []ShardStats` - Get per-shard statistics

### LoggerManager Methods

- `NewLoggerManager(config Config) (*LoggerManager, error)` - Create a new logger manager instance
- `LogWithEvent(eventName string, message string)` - Log a string message to an event-specific logger
- `LogBytesWithEvent(eventName string, data []byte)` - Log raw bytes to an event-specific logger
- `InitializeEventLogger(eventName string) error` - Explicitly initialize a logger for an event
- `CloseEventLogger(eventName string) error` - Close and remove a specific event logger
- `HasEventLogger(eventName string) bool` - Check if a logger exists for an event
- `ListEventLoggers() []string` - Get list of all active event logger names
- `Close() error` - Gracefully shutdown and flush all event loggers
- `GetStatsSnapshot() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64)` - Get aggregated statistics across all events
- `GetEventStats(eventName string) (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64, error)` - Get statistics for a specific event
- `GetAggregatedFlushMetrics() FlushMetrics` - Get aggregated flush metrics across all events

### Configuration

```go
type Config struct {
    LogFilePath   string        // Path to log file (required)
    BufferSize    int           // Total buffer size in bytes (default: 64MB)
    NumShards     int           // Number of shards (default: 8)
    FlushInterval time.Duration // Time-based flush trigger (default: 10s)
    FlushTimeout  time.Duration // Max wait for writes to complete (default: 10ms)
}
```

### Error Handling

- **`New()`**: Returns an error if:
  - Configuration is invalid (missing log path, invalid buffer size, etc.)
  - File cannot be opened (permissions, disk full, etc.)
  - Always check and handle this error

- **`Log()` / `LogBytes()`**: These methods do not return errors. They are designed for non-blocking operation:
  - Successful writes: Data is queued for flushing
  - Failed writes: Logs are dropped and counted in `DroppedLogs` statistic
  - Check `GetStatsSnapshot()` periodically to detect issues
  - If `DroppedLogs > 0`, investigate buffer size, shard count, or disk issues

- **`Close()`**: Returns an error if file close fails. Always check the return value:
  ```go
  if err := logger.Close(); err != nil {
      log.Printf("Error closing logger: %v", err)
  }
  ```

### Thread Safety

All methods are safe for concurrent use:
- Multiple goroutines can call `Log()` or `LogBytes()` simultaneously
- Statistics methods (`GetStatsSnapshot()`, `GetFlushMetrics()`, `GetShardStats()`) are safe to call concurrently
- `Close()` is idempotent and safe to call multiple times

### Recovery from Crashes

If the application crashes, log files may contain:
- **Complete shards**: Fully written shards with valid headers can be read normally
- **Incomplete shards**: Shards that were being written when crash occurred may have:
  - Valid header but incomplete data (use `validDataBytes` to read only complete entries)
  - Missing header (treat as end of file)
  
The log reading example above handles these cases by:
- Checking for complete headers before reading
- Using `validDataBytes` to limit reads to valid data
- Stopping at incomplete entries rather than reading garbage