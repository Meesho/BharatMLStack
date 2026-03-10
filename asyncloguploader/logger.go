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

	// Event name for metric tag propagation
	eventName string

	// Closed flag
	closed atomic.Bool
}

// NewLogger creates a new async logger
// eventName is the sanitized event name for metric tag propagation
func NewLogger(config Config, eventName string) (*Logger, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	metricTags := MergeMetricTags(config.MetricTags, eventName)

	// Create file writer
	fileWriter, err := NewSizeFileWriter(config, metricTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create file writer: %w", err)
	}

	// Create flush channel first
	// Buffer size: numShards * 2 (one buffer per shard, but can have both buffers full)
	flushChan := make(chan *Buffer, config.NumShards*2)

	// Create shard collection (each shard has its own double buffer)
	// Pass flush channel and metric tags so buffers can enqueue themselves on swap
	shardCollection, err := NewShardCollection(config.BufferSize, config.NumShards, flushChan, metricTags)
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
		eventName:       eventName,
	}

	// Start background worker
	go l.flushWorker()
	// Note: tickerWorker removed - buffers are pushed directly to flushChan by trySwap()
	// Threshold-based flushing is handled by flushWorker

	return l, nil
}

// LogBytes writes raw byte data to the logger (zero-allocation path)
func (l *Logger) LogBytes(data []byte) {
	tags := MergeMetricTags(l.config.MetricTags, l.eventName)

	if l.closed.Load() {
		metric.Incr(MetricLogBytesDropped, tags)
		return
	}

	// First attempt: Try to write (fast path)
	n, needsFlush, shardID := l.shardCollection.Write(data)

	if n > 0 {
		// Success! Shard is already enqueued to flush channel if needsFlush=true
		metric.Incr(MetricLogBytesSuccess, tags)
		metric.Count(MetricLogBytesWritten, int64(n), tags)
		return
	}

	// Buffer full - use per-shard semaphore retry mechanism
	shard := l.shardCollection.GetShard(shardID)
	if shard == nil {
		metric.Incr(MetricLogBytesDropped, tags)
		return
	}

	// Increase timeout to 50ms to allow flush operations to complete
	timeout := time.NewTimer(50 * time.Millisecond)
	defer timeout.Stop()

	select {
	case shard.swapSemaphore <- struct{}{}:
		defer func() { <-shard.swapSemaphore }()

		n, needsFlush = shard.Write(data)
		if n > 0 {
			metric.Incr(MetricLogBytesSuccess, tags)
			metric.Count(MetricLogBytesWritten, int64(n), tags)
			return
		}

		if needsFlush {
			shard.trySwap()
		}

		n, _ = shard.Write(data)
		if n == 0 {
			metric.Incr(MetricLogBytesDropped, tags)
		} else {
			metric.Incr(MetricLogBytesSuccess, tags)
			metric.Count(MetricLogBytesWritten, int64(n), tags)
		}

	case <-timeout.C:
		metric.Incr(MetricLogBytesDropped, tags)
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
	tags := MergeMetricTags(l.config.MetricTags, l.eventName)
	metric.Incr(MetricLogBytesFlushAttempts, tags)
	flushStart := time.Now()
	defer func() {
		metric.TimingWithStart(MetricLogBytesFlushDuration, flushStart, tags)
	}()

	l.semaphore <- struct{}{}
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
				logger.Warn().Msgf("Shard %d: Not all writes completed before flush timeout, flushing partial data", buf.ShardID())
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

	if len(shardBuffers) > 0 {
		_, err := l.fileWriter.WriteVectored(shardBuffers)
		if err != nil {
			totalBytes := 0
			for _, buf := range shardBuffers {
				totalBytes += len(buf)
			}
			metric.Incr(MetricLogBytesFlushFailure, tags)
			logger.Error().Err(err).Msgf(
				"flush error: buffers=%d bytes=%d",
				len(shardBuffers),
				totalBytes,
			)
		} else {
			metric.Incr(MetricLogBytesFlushSuccess, tags)
		}
	}

	for _, buf := range buffersToReset {
		buf.Reset()
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
