package asynclogger

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LoggerManager manages multiple Logger instances, one per event name
// Each event writes to its own log file (e.g., payment.log, login.log)
type LoggerManager struct {
	loggers sync.Map // eventName (string) -> *Logger
	baseDir string   // Base directory for log files
	config  Config   // Base config (shared settings)
}

// NewLoggerManager creates a new LoggerManager
// The base directory is extracted from config.LogFilePath
func NewLoggerManager(config Config) (*LoggerManager, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Extract base directory from LogFilePath
	baseDir := filepath.Dir(config.LogFilePath)
	if baseDir == "." || baseDir == "" {
		baseDir = "."
	}

	return &LoggerManager{
		baseDir: baseDir,
		config:  config,
	}, nil
}

// sanitizeEventName validates and sanitizes an event name for use as a filename
// Returns sanitized name or error if invalid
func sanitizeEventName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("event name cannot be empty")
	}

	// Remove invalid filesystem characters: / \ : * ? " < > |
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := name
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Replace spaces with underscores
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Limit length to 255 characters (typical filesystem limit)
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	// Ensure it's not empty after sanitization
	if sanitized == "" {
		return "", fmt.Errorf("event name becomes empty after sanitization")
	}

	return sanitized, nil
}

// getOrCreateLogger retrieves an existing logger or creates a new one for the event
// Thread-safe lazy initialization using sync.Map for lock-free reads
func (lm *LoggerManager) getOrCreateLogger(eventName string) (*Logger, error) {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return nil, err
	}

	// Fast path: check if logger exists (no lock needed with sync.Map)
	if logger, ok := lm.loggers.Load(sanitized); ok {
		return logger.(*Logger), nil
	}

	// Slow path: create new logger
	// Generate file path: {baseDir}/{eventName}.log
	eventLogPath := filepath.Join(lm.baseDir, sanitized+".log")

	// Create config for this event logger (same settings, different file path)
	eventConfig := lm.config
	eventConfig.LogFilePath = eventLogPath

	// Create new logger
	logger, err := New(eventConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger for event %s: %w", sanitized, err)
	}

	// Use LoadOrStore to ensure only one logger is created per event
	// If another goroutine created it first, close ours and return the existing one
	actual, loaded := lm.loggers.LoadOrStore(sanitized, logger)
	if loaded {
		// Another goroutine created it first, close ours to avoid resource leak
		logger.Close()
		return actual.(*Logger), nil
	}

	return logger, nil
}

// LogBytesWithEvent writes raw byte data to the event-specific logger (zero-allocation path)
func (lm *LoggerManager) LogBytesWithEvent(eventName string, data []byte) {
	logger, err := lm.getOrCreateLogger(eventName)
	if err != nil {
		// Drop log on error (could add error callback in future)
		return
	}
	logger.LogBytes(data)
}

// LogWithEvent writes a string message to the event-specific logger (convenience API)
func (lm *LoggerManager) LogWithEvent(eventName string, message string) {
	logger, err := lm.getOrCreateLogger(eventName)
	if err != nil {
		// Drop log on error (could add error callback in future)
		return
	}
	logger.Log(message)
}

// InitializeEventLogger creates a logger for the specified event if it doesn't exist
// Called via webhook when new event configuration is added
// Returns error if event name is invalid or logger creation fails
func (lm *LoggerManager) InitializeEventLogger(eventName string) error {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return fmt.Errorf("invalid event name: %w", err)
	}

	// Check if logger already exists (no lock needed with sync.Map)
	if _, exists := lm.loggers.Load(sanitized); exists {
		return nil // Already exists, no-op (idempotent)
	}

	// Create logger (same logic as getOrCreateLogger but explicit)
	_, err = lm.getOrCreateLogger(sanitized)
	return err
}

// CloseEventLogger closes and removes the logger for the specified event
// Called via webhook when event configuration is disabled
// Returns error if event logger doesn't exist or close fails
func (lm *LoggerManager) CloseEventLogger(eventName string) error {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return fmt.Errorf("invalid event name: %w", err)
	}

	// Load and delete atomically
	logger, exists := lm.loggers.LoadAndDelete(sanitized)
	if !exists {
		return fmt.Errorf("event logger not found: %s", sanitized)
	}

	// Close the logger (flushes pending logs and releases resources)
	return logger.(*Logger).Close()
}

// HasEventLogger checks if a logger exists for the specified event
func (lm *LoggerManager) HasEventLogger(eventName string) bool {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return false
	}

	_, exists := lm.loggers.Load(sanitized)
	return exists
}

// ListEventLoggers returns a list of all active event logger names
func (lm *LoggerManager) ListEventLoggers() []string {
	events := make([]string, 0)
	lm.loggers.Range(func(key, value interface{}) bool {
		events = append(events, key.(string))
		return true // continue iteration
	})
	return events
}

// Close gracefully shuts down all loggers, flushing all pending data
func (lm *LoggerManager) Close() error {
	var firstErr error
	lm.loggers.Range(func(key, value interface{}) bool {
		eventName := key.(string)
		logger := value.(*Logger)
		if err := logger.Close(); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("error closing logger for event %s: %w", eventName, err)
			}
		}
		// Delete from map as we iterate
		lm.loggers.Delete(key)
		return true // continue iteration
	})

	return firstErr
}

// GetStatsSnapshot returns aggregated statistics from all event loggers
func (lm *LoggerManager) GetStatsSnapshot() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64) {
	lm.loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		tl, dl, bw, f, fe, ss := logger.GetStatsSnapshot()
		totalLogs += tl
		droppedLogs += dl
		bytesWritten += bw
		flushes += f
		flushErrors += fe
		setSwaps += ss
		return true // continue iteration
	})

	return totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps
}

// GetAggregatedFlushMetrics returns aggregated flush metrics from all event loggers
func (lm *LoggerManager) GetAggregatedFlushMetrics() FlushMetrics {
	var totalFlushDuration int64
	var totalWriteDuration int64
	var maxFlushDuration int64
	var maxWriteDuration int64
	var totalFlushes int64
	var totalBlockedSwaps int64

	lm.loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		metrics := logger.GetFlushMetrics()
		totalFlushDuration += metrics.TotalFlushDuration.Nanoseconds()
		totalWriteDuration += metrics.AvgWriteDuration.Nanoseconds() * metrics.TotalFlushes
		if metrics.MaxFlushDuration.Nanoseconds() > maxFlushDuration {
			maxFlushDuration = metrics.MaxFlushDuration.Nanoseconds()
		}
		if metrics.MaxWriteDuration.Nanoseconds() > maxWriteDuration {
			maxWriteDuration = metrics.MaxWriteDuration.Nanoseconds()
		}
		totalFlushes += metrics.TotalFlushes
		totalBlockedSwaps += metrics.BlockedSwaps
		return true // continue iteration
	})

	avgFlushDuration := time.Duration(0)
	avgWriteDuration := time.Duration(0)
	writePercent := 0.0

	if totalFlushes > 0 {
		avgFlushDuration = time.Duration(totalFlushDuration / totalFlushes)
		avgWriteDuration = time.Duration(totalWriteDuration / totalFlushes)
	}

	if totalFlushDuration > 0 {
		writePercent = float64(totalWriteDuration) / float64(totalFlushDuration) * 100.0
	}

	return FlushMetrics{
		TotalFlushDuration: time.Duration(totalFlushDuration),
		AvgFlushDuration:   avgFlushDuration,
		MaxFlushDuration:   time.Duration(maxFlushDuration),
		FlushQueueDepth:    0, // Not aggregated
		BlockedSwaps:       totalBlockedSwaps,
		TotalFlushes:       totalFlushes,
		AvgWriteDuration:   avgWriteDuration,
		MaxWriteDuration:   time.Duration(maxWriteDuration),
		WritePercent:       writePercent,
	}
}

// GetEventStats returns statistics for a specific event logger
func (lm *LoggerManager) GetEventStats(eventName string) (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64, err error) {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("invalid event name: %w", err)
	}

	logger, exists := lm.loggers.Load(sanitized)
	if !exists {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("event logger not found: %s", sanitized)
	}

	totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps = logger.(*Logger).GetStatsSnapshot()
	return totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps, nil
}
