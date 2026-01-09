# Testing Guide for Async Log Uploader

## Overview

Comprehensive unit tests and functional tests for the `asyncloguploader` module, including file integrity verification.

## Test Files

### Unit Tests

1. **`shard_test.go`** - Tests for Shard struct
   - Shard creation and initialization
   - Write operations (with length prefix)
   - Buffer swap coordination
   - Write completion tracking
   - Concurrent writes
   - Data retrieval and reset

2. **`shard_collection_test.go`** - Tests for ShardCollection
   - Collection creation with correct shard count
   - 25% threshold calculation
   - Round-robin shard selection
   - Threshold reached detection
   - Ready shards collection
   - Reset operations

3. **`logger_test.go`** - Tests for Logger
   - Logger creation and configuration
   - LogBytes and Log operations
   - Concurrent writes
   - Swap coordination with semaphore
   - Flush operations (threshold and interval)
   - Graceful shutdown
   - Statistics tracking

4. **`file_writer_test.go`** - Tests for FileWriter
   - Vectored write operations
   - File rotation (size-based)
   - File integrity verification
   - Header format validation
   - Duration tracking

### Functional/Integration Tests

5. **`integration_test.go`** - End-to-end tests
   - File integrity end-to-end verification
   - Concurrent writes with file integrity
   - File rotation integrity
   - Data correctness verification
   - Multiple events file integrity
   - File format verification (shard headers, data layout)

## Running Tests

### Prerequisites

Install test dependencies:
```bash
go mod tidy
```

### Run All Tests

```bash
go test ./asyncloguploader/...
```

### Run with Coverage

```bash
go test -cover ./asyncloguploader/...
```

### Run Specific Test

```bash
go test -v ./asyncloguploader -run TestShard_Write
```

### Run with Verbose Output

```bash
go test -v ./asyncloguploader/...
```

## Test Coverage Areas

### ✅ Core Functionality
- Shard operations (write, swap, reset)
- ShardCollection operations (write, threshold, ready shards)
- Logger operations (LogBytes, Log, flush)
- FileWriter operations (write, rotation)

### ✅ Concurrency
- Concurrent writes to shards
- Swap coordination with semaphore
- Multiple goroutines writing simultaneously

### ✅ File Integrity
- Data correctness (written data matches read data)
- File format verification (shard headers, data layout)
- Data order preservation
- Header format correctness

### ✅ Edge Cases
- Empty data handling
- Buffer full scenarios
- Logger closed scenarios
- Invalid configurations
- Double close handling

### ✅ Integration
- End-to-end logging flow
- File rotation with integrity
- Multiple events (LoggerManager)
- Concurrent writes with file verification

## File Integrity Verification

The `verifyFileFormat` function in `integration_test.go` performs comprehensive file format verification:

1. **Shard Header Verification**
   - Reads 8-byte headers (capacity + validDataBytes)
   - Verifies header values are reasonable
   - Ensures validDataBytes <= capacity

2. **Data Layout Verification**
   - Verifies data starts immediately after header (no padding)
   - Checks data sections contain actual data (not all zeros)
   - Validates data order

3. **End-to-End Verification**
   - Writes known data
   - Reads file and verifies exact match
   - Verifies all messages are present
   - Checks file structure integrity

## Test Scenarios

### Basic Operations
- ✅ Create shard with double buffer
- ✅ Write data to shard
- ✅ Swap buffers
- ✅ Get data from inactive buffer
- ✅ Reset buffer after flush

### Threshold and Flush
- ✅ 25% threshold calculation (2 out of 8 shards)
- ✅ Threshold reached detection
- ✅ Batch flush of ready shards
- ✅ Periodic flush on interval

### Swap Coordination
- ✅ Semaphore-based swap coordination
- ✅ Retry after swap
- ✅ CAS protection for swap
- ✅ Multiple writers coordination

### File Operations
- ✅ Write vectored buffers
- ✅ Size-based rotation
- ✅ File integrity after rotation
- ✅ Header format correctness

### Concurrency
- ✅ Concurrent writes to shards
- ✅ Concurrent writes to logger
- ✅ Swap coordination under load
- ✅ File integrity under concurrent writes

## Expected Test Results

All tests should pass with:
- ✅ No data loss
- ✅ Correct file format
- ✅ Proper header structure
- ✅ Data integrity maintained
- ✅ Correct statistics tracking

## Troubleshooting

### Missing Dependencies
If you see errors about missing `go.sum` entries:
```bash
go mod tidy
```

### Test Failures
1. Check file permissions (tests use temp directories)
2. Verify sufficient disk space
3. Check for concurrent test execution issues
4. Review test output for specific failure details

### Coverage Issues
Run with coverage to identify untested code paths:
```bash
go test -coverprofile=coverage.out ./asyncloguploader/...
go tool cover -html=coverage.out
```

## Test Data Patterns

Tests use various data patterns:
- Small messages (few bytes)
- Large messages (MBs)
- Special characters
- Binary data
- Sequential patterns (for order verification)

## Performance Considerations

Tests are designed to:
- Run quickly (< 1 minute for all tests)
- Use temporary directories (cleaned up automatically)
- Avoid blocking operations where possible
- Use appropriate timeouts for async operations

