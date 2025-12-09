package asynclogger

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"
)

// Statistics holds operational statistics for the logger
type Statistics struct {
	TotalLogs    atomic.Int64 // Total log attempts (successful + dropped)
	DroppedLogs  atomic.Int64 // Logs dropped (buffer full, logger closed, etc.)
	BytesWritten atomic.Int64 // Total bytes successfully written to buffers
	Flushes      atomic.Int64 // Number of flush operations completed
	FlushErrors  atomic.Int64 // Number of flush operations that failed
	SetSwaps     atomic.Int64 // Number of buffer set swaps performed

	// Flush performance metrics (for 210s cliff investigation)
	TotalFlushDuration atomic.Int64 // Total time spent in flush operations (nanoseconds)
	MaxFlushDuration   atomic.Int64 // Maximum flush duration seen (nanoseconds)
	FlushQueueDepth    atomic.Int64 // Current depth of flush queue
	BlockedSwaps       atomic.Int64 // Number of swaps that blocked waiting for flush

	// Detailed I/O breakdown (for disk I/O investigation)
	TotalWriteDuration atomic.Int64 // Time spent in writeAligned() (nanoseconds)
	MaxWriteDuration   atomic.Int64 // Maximum write duration (nanoseconds)
}

// Logger is an async logger using Sharded Double Buffer CAS with Direct I/O
type Logger struct {
	// Two sets of sharded buffers for double buffering
	setA *BufferSet
	setB *BufferSet

	// Active set pointer (atomically swapped)
	activeSet atomic.Pointer[BufferSet]

	// FileWriter for writing logs with Direct I/O and rotation support
	fileWriter *FileWriter

	// Channel for flush requests
	flushChan chan *BufferSet

	// Ticker for periodic flushing
	ticker *time.Ticker

	// Channel for shutdown signal
	done chan struct{}

	// Semaphore to prevent concurrent flushes
	semaphore chan struct{}

	// Semaphore for swap coordination (allows multiple threads to wait for swap)
	// Capacity: 30 permits allows ~30 threads to coordinate simultaneously
	swapSemaphore chan struct{}

	// Configuration
	config Config

	// Statistics
	stats Statistics

	// Next set ID for tracking
	nextID atomic.Uint32

	// Swap in progress flag
	swapping atomic.Bool

	// Closed flag
	closed atomic.Bool
}

// New creates a new async logger
func New(config Config) (*Logger, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create FileWriter for Direct I/O with rotation support
	fileWriter, err := NewFileWriter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create file writer: %w", err)
	}

	// Create two buffer sets for double buffering
	setA := NewBufferSet(config.BufferSize, config.NumShards, 0)
	setB := NewBufferSet(config.BufferSize, config.NumShards, 1)

	// Initialize logger
	l := &Logger{
		setA:          setA,
		setB:          setB,
		fileWriter:    fileWriter,
		flushChan:     make(chan *BufferSet, 2), // Buffer for both sets
		ticker:        time.NewTicker(config.FlushInterval),
		done:          make(chan struct{}),
		semaphore:     make(chan struct{}, 1),
		swapSemaphore: make(chan struct{}, 30), // 30 permits for swap coordination
		config:        config,
	}

	l.activeSet.Store(setA)
	l.nextID.Store(2) // Start from 2 since setA=0, setB=1

	// Start background workers
	go l.flushWorker()
	go l.tickerWorker()

	return l, nil
}

// LogBytes writes raw byte data to the logger (zero-allocation path)
// This is the high-performance API that avoids allocations when the caller
// provides a reusable byte buffer. The data is copied into the internal buffer.
func (l *Logger) LogBytes(data []byte) {
	// Count every log attempt (successful or dropped)
	l.stats.TotalLogs.Add(1)

	if l.closed.Load() {
		l.stats.DroppedLogs.Add(1)
		return
	}

	// Get active set
	activeSet := l.activeSet.Load()
	if activeSet == nil {
		l.stats.DroppedLogs.Add(1)
		return
	}

	// First attempt: Try to write (fast path)
	n, needsFlush, _ := activeSet.Write(data)

	if n > 0 {
		// Success! Trigger swap if needed (existing behavior)
		if needsFlush {
			l.trySwap()
		}
		return
	}

	// Buffer full - use semaphore retry mechanism
	// Use non-blocking select with timeout to avoid blocking hot path
	timeout := time.NewTimer(10 * time.Millisecond)
	defer timeout.Stop()

	select {
	case l.swapSemaphore <- struct{}{}: // Acquired permit
		defer func() { <-l.swapSemaphore }() // Release when done

		// Re-check 1: Buffer might have been swapped by another thread
		activeSet = l.activeSet.Load()
		if activeSet == nil {
			l.stats.DroppedLogs.Add(1)
			return
		}

		n, needsFlush, _ = activeSet.Write(data)
		if n > 0 {
			// Success after re-check!
			if needsFlush {
				l.trySwap()
			}
			return
		}

		// Still full - trigger swap (only one thread will succeed)
		if needsFlush {
			l.trySwap()
		}

		// Re-check 2: After swap, try writing again
		activeSet = l.activeSet.Load()
		if activeSet == nil {
			l.stats.DroppedLogs.Add(1)
			return
		}

		n, _, _ = activeSet.Write(data)
		if n == 0 {
			// Still failed after swap - drop log
			l.stats.DroppedLogs.Add(1)
		}

	case <-timeout.C:
		// Timeout: Couldn't acquire semaphore quickly, drop log
		l.stats.DroppedLogs.Add(1)
	}
}

// Log writes a string message to the logger (convenience API)
// This method uses unsafe pointer conversion to avoid string-to-bytes allocation.
// For maximum performance in hot paths, use LogBytes() with a reused buffer.
func (l *Logger) Log(message string) {
	// Convert string to []byte without allocation using unsafe
	data := stringToBytes(message)
	l.LogBytes(data)
}

// stringToBytes converts a string to []byte without allocation
// Uses the string's backing array directly (read-only)
func stringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	// Use unsafe to access string's backing array directly
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// trySwap attempts to swap the active buffer set
func (l *Logger) trySwap() {
	// Check if already swapping
	if !l.swapping.CompareAndSwap(false, true) {
		return // Another goroutine is already swapping
	}
	defer l.swapping.Store(false)

	// Get current active set
	currentSet := l.activeSet.Load()
	if currentSet == nil {
		return
	}

	// Determine which set to swap to
	var nextSet *BufferSet
	if currentSet.ID() == l.setA.ID() {
		nextSet = l.setB
	} else {
		nextSet = l.setA
	}

	// Assign new ID to next set
	nextSet.SetID(l.nextID.Add(1))

	// Atomically swap the active set
	if !l.activeSet.CompareAndSwap(currentSet, nextSet) {
		// Swap failed, another goroutine beat us
		return
	}

	l.stats.SetSwaps.Add(1)

	// Send the old set for flushing (non-blocking)
	select {
	case l.flushChan <- currentSet:
		// Successfully queued for flush
	default:
		// Channel full, skip this flush (data will be flushed on next interval or shutdown)
	}
}

// flushWorker processes flush requests
func (l *Logger) flushWorker() {
	for {
		select {
		case set := <-l.flushChan:
			l.flushSet(set)
		case <-l.done:
			// Flush any remaining data in the channel
			l.drainFlushChannel()
			return
		}
	}
}

// tickerWorker triggers periodic flushes
func (l *Logger) tickerWorker() {
	for {
		select {
		case <-l.ticker.C:
			// Trigger a swap to flush accumulated data
			activeSet := l.activeSet.Load()
			if activeSet != nil && activeSet.HasData() {
				l.trySwap()
			}
		case <-l.done:
			return
		}
	}
}

// flushSet writes all data from a buffer set to disk
func (l *Logger) flushSet(set *BufferSet) {
	// Track flush operation timing
	flushStart := time.Now()

	// Increment queue depth (for monitoring)
	l.stats.FlushQueueDepth.Add(1)
	defer l.stats.FlushQueueDepth.Add(-1)

	// Acquire semaphore to prevent concurrent flushes
	semaphoreAcquireStart := time.Now()
	l.semaphore <- struct{}{}
	semaphoreWaitDuration := time.Since(semaphoreAcquireStart)
	if semaphoreWaitDuration > time.Millisecond {
		// Track if we blocked waiting for semaphore
		l.stats.BlockedSwaps.Add(1)
	}
	defer func() { <-l.semaphore }()

	// Collect all shard buffers for batched write (OPTIMIZATION: 8 syscalls → 1!)
	// Each shard buffer has 8-byte header reserved at the start: [4 bytes capacity][4 bytes valid data]
	// Headers are written directly into the buffer's reserved space, then buffer is used directly (zero-copy!)
	numShards := len(set.Shards())
	shardBuffers := make([][]byte, 0, numShards)

	for _, shard := range set.Shards() {
		// Quick check: skip shards with no data (offset <= 8 means no data written)
		if shard.Offset() <= 8 {
			continue
		}

		// Get buffer data - this waits for all writes to complete
		// After this returns, the offset is stable (no more writes can happen)
		data, _ := shard.GetData(l.config.FlushTimeout)

		// Read offset AFTER GetData() completes to ensure it reflects all completed writes
		// This is safe because GetData() is called with shard mutex held, preventing concurrent writes
		shardOffset := shard.Offset()
		if shardOffset <= 8 {
			// No data written (shouldn't happen if first check passed, but defensive)
			continue
		}

		// Note: If complete is false, timeout occurred and last write may be incomplete
		// This is acceptable - only the last incomplete write may be corrupted
		capacity := shard.Capacity()
		// validDataBytes is the actual data size (excluding the 8-byte header reservation)
		validDataBytes := shardOffset - 8

		// Defensive check: ensure validDataBytes is non-negative (should always be true)
		if validDataBytes < 0 {
			validDataBytes = 0
		}

		// Write header directly into the first 8 bytes of the buffer (in-place, zero-copy!)
		binary.LittleEndian.PutUint32(data[0:4], uint32(capacity))
		binary.LittleEndian.PutUint32(data[4:8], uint32(validDataBytes))

		// Use buffer directly - no copying needed! Header is already in place, data follows immediately
		shardBuffers = append(shardBuffers, data)
	}

	// Single batched write for all shards - track timing
	if len(shardBuffers) > 0 {
		writeStart := time.Now()
		n, err := l.fileWriter.WriteVectored(shardBuffers)
		writeDuration := time.Since(writeStart)

		// Track write duration
		writeDurationNs := writeDuration.Nanoseconds()
		l.stats.TotalWriteDuration.Add(writeDurationNs)

		// Update max write duration atomically
		for {
			currentMax := l.stats.MaxWriteDuration.Load()
			if writeDurationNs <= currentMax {
				break
			}
			if l.stats.MaxWriteDuration.CompareAndSwap(currentMax, writeDurationNs) {
				break
			}
		}

		if err != nil {
			l.stats.FlushErrors.Add(1)
			// Log flush error details for debugging
			// Note: Using fmt.Printf to avoid circular dependency on logger
			fmt.Printf("[FLUSH_ERROR] Logger=%s SetID=%d Shards=%d Bytes=%d Error=%v Duration=%v\n",
				l.config.LogFilePath, set.ID(), len(shardBuffers), func() int {
					total := 0
					for _, buf := range shardBuffers {
						total += len(buf)
					}
					return total
				}(), err, writeDuration)
		} else {
			l.stats.BytesWritten.Add(int64(n))
			l.stats.Flushes.Add(1)
		}
	}

	// Reset all shards after flush attempt
	for _, shard := range set.Shards() {
		shard.Reset()
	}

	// Note: With O_DSYNC flag, each write() automatically syncs data to disk
	// No explicit file.Sync() call needed - sync happens during WriteVectored()

	// Track flush duration
	flushDuration := time.Since(flushStart)
	flushDurationNs := flushDuration.Nanoseconds()
	l.stats.TotalFlushDuration.Add(flushDurationNs)

	// Update max flush duration atomically
	for {
		currentMax := l.stats.MaxFlushDuration.Load()
		if flushDurationNs <= currentMax {
			break
		}
		if l.stats.MaxFlushDuration.CompareAndSwap(currentMax, flushDurationNs) {
			break
		}
	}
}

// drainFlushChannel flushes all pending buffer sets in the channel
func (l *Logger) drainFlushChannel() {
	for {
		select {
		case set := <-l.flushChan:
			l.flushSet(set)
		default:
			return
		}
	}
}

// Close gracefully shuts down the logger, flushing all pending data
func (l *Logger) Close() error {
	// Check if already closed
	if !l.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Stop the ticker
	l.ticker.Stop()

	// Signal workers to stop
	close(l.done)

	// Give workers time to finish
	time.Sleep(100 * time.Millisecond)

	// Flush the currently active set
	activeSet := l.activeSet.Load()
	if activeSet != nil && activeSet.HasData() {
		l.flushSet(activeSet)
	}

	// Flush the inactive set if it has data
	var inactiveSet *BufferSet
	if activeSet.ID() == l.setA.ID() {
		inactiveSet = l.setB
	} else {
		inactiveSet = l.setA
	}
	if inactiveSet != nil && inactiveSet.HasData() {
		l.flushSet(inactiveSet)
	}

	// Close the file writer (handles rotation cleanup)
	if err := l.fileWriter.Close(); err != nil {
		return fmt.Errorf("failed to close file writer: %w", err)
	}

	return nil
}

// GetStatsSnapshot returns current statistics values
func (l *Logger) GetStatsSnapshot() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64) {
	return l.stats.TotalLogs.Load(),
		l.stats.DroppedLogs.Load(),
		l.stats.BytesWritten.Load(),
		l.stats.Flushes.Load(),
		l.stats.FlushErrors.Load(),
		l.stats.SetSwaps.Load()
}

// FlushMetrics holds flush performance metrics for investigation
type FlushMetrics struct {
	TotalFlushDuration time.Duration // Total time spent in flush operations
	AvgFlushDuration   time.Duration // Average flush duration
	MaxFlushDuration   time.Duration // Maximum flush duration seen
	FlushQueueDepth    int64         // Current depth of flush queue
	BlockedSwaps       int64         // Number of swaps that blocked
	TotalFlushes       int64         // Total number of flushes

	// I/O breakdown (for disk I/O investigation)
	AvgWriteDuration time.Duration // Average time for writeAligned()
	MaxWriteDuration time.Duration // Maximum write duration
	WritePercent     float64       // % of flush time spent in write
}

// GetFlushMetrics returns flush performance metrics
func (l *Logger) GetFlushMetrics() FlushMetrics {
	totalDuration := l.stats.TotalFlushDuration.Load()
	totalWrite := l.stats.TotalWriteDuration.Load()
	flushes := l.stats.Flushes.Load()

	avgFlushDuration := time.Duration(0)
	avgWriteDuration := time.Duration(0)
	writePercent := 0.0

	if flushes > 0 {
		avgFlushDuration = time.Duration(totalDuration / flushes)
		avgWriteDuration = time.Duration(totalWrite / flushes)
	}

	if totalDuration > 0 {
		writePercent = float64(totalWrite) / float64(totalDuration) * 100.0
	}

	return FlushMetrics{
		TotalFlushDuration: time.Duration(totalDuration),
		AvgFlushDuration:   avgFlushDuration,
		MaxFlushDuration:   time.Duration(l.stats.MaxFlushDuration.Load()),
		FlushQueueDepth:    l.stats.FlushQueueDepth.Load(),
		BlockedSwaps:       l.stats.BlockedSwaps.Load(),
		TotalFlushes:       flushes,
		AvgWriteDuration:   avgWriteDuration,
		MaxWriteDuration:   time.Duration(l.stats.MaxWriteDuration.Load()),
		WritePercent:       writePercent,
	}
}

// ShardStats holds statistics for a single shard
type ShardStats struct {
	ShardID        int
	WriteCount     int64
	BytesUsed      int32
	Capacity       int32
	UtilizationPct float64
}

// GetShardStats returns per-shard statistics from the currently active set
func (l *Logger) GetShardStats() []ShardStats {
	activeSet := l.activeSet.Load()
	if activeSet == nil {
		return nil
	}

	shards := activeSet.Shards()
	stats := make([]ShardStats, len(shards))

	for i, shard := range shards {
		// Offset includes the 8-byte header reservation, so subtract it for actual data size
		bytesUsed := shard.Offset() - 8
		capacity := shard.Capacity()
		utilizationPct := 0.0
		if capacity > 0 {
			// Utilization is based on usable capacity (excluding header reservation)
			utilizationPct = float64(bytesUsed) / float64(capacity-8) * 100.0
		}

		stats[i] = ShardStats{
			ShardID:        i,
			WriteCount:     shard.buffer.WriteCount(),
			BytesUsed:      bytesUsed,
			Capacity:       capacity,
			UtilizationPct: utilizationPct,
		}
	}

	return stats
}
