package asyncloguploader

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/Meesho/go-core/metric"
	logger "github.com/rs/zerolog/log"
)

// Statistics holds operational statistics for the logger
type Statistics struct {
	TotalLogs    atomic.Int64 // Total log attempts (successful + dropped)
	DroppedLogs  atomic.Int64 // Logs dropped (buffer full, logger closed, etc.)
	BytesWritten atomic.Int64 // Total bytes successfully written to buffers
	Flushes      atomic.Int64 // Number of flush operations completed
	FlushErrors  atomic.Int64 // Number of flush operations that failed

	// Flush performance metrics
	TotalFlushDuration atomic.Int64 // Total time spent in flush operations (nanoseconds)
	MaxFlushDuration   atomic.Int64 // Maximum flush duration seen (nanoseconds)
	FlushQueueDepth    atomic.Int64 // Current depth of flush queue
	BlockedSwaps       atomic.Int64 // Number of swaps that blocked waiting for flush

	// Detailed I/O breakdown
	TotalWriteDuration atomic.Int64 // Time spent in WriteVectored() including rotation checks (nanoseconds)
	MaxWriteDuration   atomic.Int64 // Maximum write duration (nanoseconds)

	// Pwritev syscall timing (pure disk I/O, excludes rotation checks)
	TotalPwritevDuration atomic.Int64 // Time spent in Pwritev syscall only (nanoseconds)
	MaxPwritevDuration   atomic.Int64 // Maximum Pwritev duration (nanoseconds)
}

// Logger is an async logger using Sharded Double Buffer CAS with Direct I/O
// Each shard has its own double buffer and swaps individually
type Logger struct {
	// Collection of shards, each with its own double buffer
	shardCollection *ShardCollection

	// FileWriter for writing logs with Direct I/O and rotation support
	fileWriter FileWriter

	// Channel for flush requests (individual buffers sent on swap)
	flushChan chan *Buffer

	// Channel for shutdown signal
	done chan struct{}

	// Semaphore to prevent concurrent flushes
	semaphore chan struct{}

	// Configuration
	config Config

	// Statistics
	stats Statistics

	// Closed flag
	closed atomic.Bool
}

// NewLogger creates a new async logger
func NewLogger(config Config) (*Logger, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create file writer
	fileWriter, err := NewSizeFileWriter(config, config.UploadChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to create file writer: %w", err)
	}

	// Create flush channel first
	// Buffer size: numShards * 2 (one buffer per shard, but can have both buffers full)
	flushChan := make(chan *Buffer, config.NumShards*2)

	// Create shard collection (each shard has its own double buffer)
	// Pass flush channel so buffers can enqueue themselves on swap
	shardCollection, err := NewShardCollection(config.BufferSize, config.NumShards, flushChan)
	if err != nil {
		return nil, fmt.Errorf("failed to create shard collection: %w", err)
	}

	// Initialize logger
	l := &Logger{
		shardCollection: shardCollection,
		fileWriter:      fileWriter,
		flushChan:       flushChan,
		done:            make(chan struct{}),
		semaphore:       make(chan struct{}, 1),
		config:          config,
	}

	// Start background worker
	go l.flushWorker()
	// Note: tickerWorker removed - buffers are pushed directly to flushChan by trySwap()
	// Threshold-based flushing is handled by flushWorker

	return l, nil
}

// LogBytes writes raw byte data to the logger (zero-allocation path)
func (l *Logger) LogBytes(data []byte) {

	// Count every log attempt (successful or dropped)
	l.stats.TotalLogs.Add(1)

	if l.closed.Load() {
		metric.Incr(MetricLogBytesDropped, []string{})
		l.stats.DroppedLogs.Add(1)
		return
	}

	// First attempt: Try to write (fast path)
	n, needsFlush, shardID := l.shardCollection.Write(data)

	if n > 0 {
		// Success! Shard is already enqueued to flush channel if needsFlush=true
		// Flush worker will accumulate and flush when threshold reached
		metric.Incr(MetricLogBytesSuccess, []string{})
		l.stats.BytesWritten.Add(int64(n))
		return
	}

	// Buffer full - use per-shard semaphore retry mechanism
	// Use non-blocking select with timeout to avoid blocking hot path
	shard := l.shardCollection.GetShard(shardID)
	if shard == nil {
		metric.Incr(MetricLogBytesDropped, []string{})
		l.stats.DroppedLogs.Add(1)
		return
	}

	// Increase timeout to 50ms to allow flush operations to complete
	// Under high load, flushes can take longer, and we want to avoid dropping logs
	timeout := time.NewTimer(50 * time.Millisecond)
	defer timeout.Stop()

	select {
	case shard.swapSemaphore <- struct{}{}: // Acquired permit for this shard
		defer func() { <-shard.swapSemaphore }() // Release when done

		// Re-check 1: Swap might have happened by another thread
		n, needsFlush = shard.Write(data)
		if n > 0 {
			// Success after re-check! Shard is already enqueued if needsFlush=true
			metric.Incr(MetricLogBytesSuccess, []string{})
			l.stats.BytesWritten.Add(int64(n))
			return
		}

		// Still full - trigger swap (only one thread will succeed per shard)
		if needsFlush {

			shard.trySwap()
			// After swap, readyForFlush is still true (inactive buffer needs flush)
			// But the new active buffer is empty and should accept writes
		}

		// Re-check 2: After swap, try writing again to the new active buffer
		// The Write() method now checks buffer space before readyForFlush,
		// so it will succeed if the new buffer has space
		n, _ = shard.Write(data)
		if n == 0 {
			// Still failed after swap - this means both buffers are truly full
			// (very rare, but possible under extreme load)
			metric.Incr(MetricLogBytesDropped, []string{})
			l.stats.DroppedLogs.Add(1)
		} else {
			// Success after swap! Shard is already enqueued if needsFlush=true
			metric.Incr(MetricLogBytesSuccess, []string{})
			l.stats.BytesWritten.Add(int64(n))
		}

	case <-timeout.C:
		// Timeout: Couldn't acquire semaphore quickly, drop log
		l.stats.DroppedLogs.Add(1)
	}
}

// Log writes a string message to the logger (convenience API)
func (l *Logger) Log(message string) {
	// Convert string to []byte without allocation using unsafe
	data := stringToBytes(message)
	l.LogBytes(data)
}

// stringToBytes converts a string to []byte without allocation
func stringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	// Use unsafe to access string's backing array directly
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// flushWorker processes flush requests
// Accumulates buffers in a list and flushes when threshold is reached
func (l *Logger) flushWorker() {
	flushList := make([]*Buffer, 0, l.shardCollection.NumShards()*2) // *2 for both buffers
	uniqueShards := make(map[uint32]bool)                            // Track unique shards for threshold

	for {
		select {
		case buffer := <-l.flushChan:
			// Deduplicate: Check if buffer already in list (by pointer)
			alreadyInList := false
			for _, b := range flushList {
				if b == buffer {
					alreadyInList = true
					break
				}
			}

			if !alreadyInList {
				flushList = append(flushList, buffer)
				uniqueShards[buffer.ShardID()] = true // Track unique shard
			}

			// Check if threshold reached (count unique shards)
			if len(uniqueShards) >= int(l.shardCollection.threshold) {
				l.flushBuffers(flushList)
				flushList = flushList[:0]            // Clear list
				uniqueShards = make(map[uint32]bool) // Reset shard tracking
			}

		case <-l.done:
			// Flush any remaining data in the list
			// Note: Close() will drain the channel to catch any buffers that arrived
			// after flushWorker exited (via drainFlushChannel())
			if len(flushList) > 0 {
				l.flushBuffers(flushList)
			}
			return
		}
	}
}

// flushBuffers writes all data from buffers to disk using batch flush
// Much simpler: each buffer knows how to get its data and reset itself
func (l *Logger) flushBuffers(buffers []*Buffer) {
	metric.Incr(MetricLogBytesFlush, []string{})
	// Track flush operation timing
	flushStart := time.Now()
	defer func() {
		metric.TimingWithStart(MetricLogBytesFlushDuration, flushStart, []string{})
	}()

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

	// Collect all buffer data for batched write (single Pwritev syscall)
	shardBuffers := make([][]byte, 0, len(buffers))
	buffersToReset := make([]*Buffer, 0, len(buffers))

	for _, buf := range buffers {
		// Skip if buffer has no data
		if !buf.HasData() {
			continue
		}

		// Get buffer data (waits for inflight writes)
		data, allWritesCompleted := buf.GetData(l.config.FlushTimeout)
		if data == nil {
			continue
		}

		shardOffset := buf.Offset()
		if shardOffset > headerOffset {
			capacity := buf.Capacity()
			validDataBytes := shardOffset - headerOffset
			if validDataBytes < 0 {
				validDataBytes = 0
			}

			if !allWritesCompleted {
				fmt.Printf("[WARNING] Shard %d: Not all writes completed before flush timeout, flushing partial data\n", buf.ShardID())
			}

			if len(data) >= int(headerOffset) {
				// Write header directly into the first 8 bytes
				binary.LittleEndian.PutUint32(data[0:4], uint32(capacity))
				binary.LittleEndian.PutUint32(data[4:8], uint32(validDataBytes))
				shardBuffers = append(shardBuffers, data)
				buffersToReset = append(buffersToReset, buf)
			}
		}
	}

	// Single batched write for all buffers - track timing
	if len(shardBuffers) > 0 {
		writeStart := time.Now()
		_, err := l.fileWriter.WriteVectored(shardBuffers)
		writeDuration := time.Since(writeStart)

		// Track write duration (includes rotation checks)
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

		// Track Pwritev syscall duration (pure disk I/O, excludes rotation checks)
		pwritevDuration := l.fileWriter.GetLastPwritevDuration()
		if pwritevDuration > 0 {
			pwritevDurationNs := pwritevDuration.Nanoseconds()
			l.stats.TotalPwritevDuration.Add(pwritevDurationNs)

			// Update max Pwritev duration atomically
			for {
				currentMax := l.stats.MaxPwritevDuration.Load()
				if pwritevDurationNs <= currentMax {
					break
				}
				if l.stats.MaxPwritevDuration.CompareAndSwap(currentMax, pwritevDurationNs) {
					break
				}
			}
		}

		if err != nil {
			l.stats.FlushErrors.Add(1)
			// Calculate total bytes for error message
			totalBytes := 0
			for _, buf := range shardBuffers {
				totalBytes += len(buf)
			}
			logger.Error().Err(err).Msgf(
				"flush error: buffers=%d bytes=%d duration=%s",
				len(shardBuffers),
				totalBytes,
				writeDuration,
			)
			fmt.Printf("[FLUSH_ERROR] Buffers=%d Bytes=%d Error=%v Duration=%v\n",
				len(shardBuffers), totalBytes, err, writeDuration)
			// Continue processing - reset buffers even on error to prevent deadlock
		} else {
			// Note: BytesWritten is already counted when data is written to buffers in LogBytes()
			// We don't count again here to avoid double-counting
			l.stats.Flushes.Add(1)
		}
	}

	// Reset all buffers that were flushed (each buffer resets itself)
	for _, buf := range buffersToReset {
		buf.Reset()
	}

	// Note: No need to reset ready shards count - flush worker tracks by buffer list size

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

// drainFlushChannel drains any remaining buffer flush requests from the channel
func (l *Logger) drainFlushChannel() {
	buffers := make([]*Buffer, 0, l.shardCollection.NumShards()*2)
	for {
		select {
		case buffer := <-l.flushChan:
			// Deduplicate by pointer
			alreadyInList := false
			for _, b := range buffers {
				if b == buffer {
					alreadyInList = true
					break
				}
			}
			if !alreadyInList {
				buffers = append(buffers, buffer)
			}
		default:
			// Channel drained
			if len(buffers) > 0 {
				l.flushBuffers(buffers)
			}
			return
		}
	}
}

// GetStats returns a snapshot of the current statistics
func (l *Logger) GetStats() Statistics {
	return Statistics{
		TotalLogs:            atomic.Int64{},
		DroppedLogs:          atomic.Int64{},
		BytesWritten:         atomic.Int64{},
		Flushes:              atomic.Int64{},
		FlushErrors:          atomic.Int64{},
		TotalFlushDuration:   atomic.Int64{},
		MaxFlushDuration:     atomic.Int64{},
		FlushQueueDepth:      atomic.Int64{},
		BlockedSwaps:         atomic.Int64{},
		TotalWriteDuration:   atomic.Int64{},
		MaxWriteDuration:     atomic.Int64{},
		TotalPwritevDuration: atomic.Int64{},
		MaxPwritevDuration:   atomic.Int64{},
	}
}

// GetStatsSnapshot returns a snapshot of current statistics values
func (l *Logger) GetStatsSnapshot() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64) {
	return l.stats.TotalLogs.Load(),
		l.stats.DroppedLogs.Load(),
		l.stats.BytesWritten.Load(),
		l.stats.Flushes.Load(),
		l.stats.FlushErrors.Load(),
		0 // setSwaps not applicable for per-shard swap
}

// GetFlushMetrics returns flush performance metrics
func (l *Logger) GetFlushMetrics() FlushMetrics {
	flushes := l.stats.Flushes.Load()
	if flushes == 0 {
		return FlushMetrics{}
	}

	avgFlushDuration := time.Duration(l.stats.TotalFlushDuration.Load() / flushes)
	maxFlushDuration := time.Duration(l.stats.MaxFlushDuration.Load())
	avgWriteDuration := time.Duration(l.stats.TotalWriteDuration.Load() / flushes)
	maxWriteDuration := time.Duration(l.stats.MaxWriteDuration.Load())
	avgPwritevDuration := time.Duration(l.stats.TotalPwritevDuration.Load() / flushes)
	maxPwritevDuration := time.Duration(l.stats.MaxPwritevDuration.Load())

	writePercent := 0.0
	if avgFlushDuration > 0 {
		writePercent = float64(avgWriteDuration) / float64(avgFlushDuration) * 100.0
	}

	pwritevPercent := 0.0
	if avgFlushDuration > 0 {
		pwritevPercent = float64(avgPwritevDuration) / float64(avgFlushDuration) * 100.0
	}

	return FlushMetrics{
		AvgFlushDuration:   avgFlushDuration,
		MaxFlushDuration:   maxFlushDuration,
		AvgWriteDuration:   avgWriteDuration,
		MaxWriteDuration:   maxWriteDuration,
		WritePercent:       writePercent,
		AvgPwritevDuration: avgPwritevDuration,
		MaxPwritevDuration: maxPwritevDuration,
		PwritevPercent:     pwritevPercent,
	}
}

// FlushMetrics holds flush performance metrics
type FlushMetrics struct {
	AvgFlushDuration   time.Duration
	MaxFlushDuration   time.Duration
	AvgWriteDuration   time.Duration
	MaxWriteDuration   time.Duration
	WritePercent       float64
	AvgPwritevDuration time.Duration
	MaxPwritevDuration time.Duration
	PwritevPercent     float64
}

// StatsSnapshot is a snapshot of statistics values (safe to copy)
type StatsSnapshot struct {
	TotalLogs            int64
	DroppedLogs          int64
	BytesWritten         int64
	Flushes              int64
	FlushErrors          int64
	TotalFlushDuration   int64
	MaxFlushDuration     int64
	FlushQueueDepth      int64
	BlockedSwaps         int64
	TotalWriteDuration   int64
	MaxWriteDuration     int64
	TotalPwritevDuration int64
	MaxPwritevDuration   int64
}

// Close gracefully shuts down the logger
func (l *Logger) Close() error {
	if !l.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Signal shutdown (this will cause flushWorker to exit)
	close(l.done)

	// Drain any buffers that arrived in the channel after flushWorker exited
	// flushWorker processes its flushList and exits, but buffers might still be in the channel
	l.drainFlushChannel()

	// Wait for any ongoing flush to complete by acquiring and releasing the semaphore
	// This ensures no flush is in progress before we swap buffers
	// We acquire and immediately release to ensure the flush worker has finished
	// Use a timeout to prevent deadlock if flush worker is stuck
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case l.semaphore <- struct{}{}:
		// Successfully acquired semaphore - flush worker has finished
		<-l.semaphore
	case <-timeout.C:
		// Timeout: flush worker might be stuck, but we'll proceed anyway
		// This prevents deadlock during shutdown
		logger.Warn().Msg("timeout waiting for flush semaphore during Close(), proceeding anyway")
		fmt.Printf("[WARNING] Timeout waiting for flush semaphore during Close(), proceeding anyway\n")
	}

	// Now it's safe to prepare buffers for final flush
	// Get all buffers with data, not just ready ones (threshold doesn't matter during close)
	allShards := l.shardCollection.Shards()
	buffersWithData := make([]*Buffer, 0, len(allShards)*2)
	for _, shard := range allShards {
		// Check if shard has data in active buffer
		if shard.Offset() > headerOffset {
			// Data is in active buffer - need to swap first so buffer can be flushed
			// It's safe to swap now because:
			// 1. We've drained the flush channel (no pending flushes)
			// 2. We've confirmed no flush is in progress (semaphore was available)
			// 3. The inactive buffer (if any) was already flushed or is empty
			shard.trySwap() // Swap so active buffer becomes inactive (flushable)
		}

		// Collect all buffers with data
		buffersWithData = append(buffersWithData, shard.GetBuffersWithData()...)
	}

	// Flush remaining data (flushBuffers will acquire semaphore itself)
	if len(buffersWithData) > 0 {
		l.flushBuffers(buffersWithData)
	}

	// Close shard collection
	l.shardCollection.Close()

	// Close file writer
	return l.fileWriter.Close()
}
