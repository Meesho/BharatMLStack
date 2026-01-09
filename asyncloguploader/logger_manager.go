package asyncloguploader

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
	loggers       sync.Map      // eventName (string) -> *Logger
	baseDir       string        // Base directory for log files
	config        Config        // Base config (shared settings)
	uploadChannel chan<- string // Shared upload channel for all events
	uploader      *Uploader     // Optional: GCS uploader (created internally if GCSUploadConfig provided)
	ownUploader   bool          // True if uploader was created internally (needs cleanup)
}

// NewLoggerManager creates a new LoggerManager
// The base directory is extracted from config.LogFilePath
//
// If GCSUploadConfig is provided and UploadChannel is nil, LoggerManager will
// automatically create and manage an Uploader internally. The uploader will be
// started automatically and stopped when Close() is called.
//
// If UploadChannel is provided, it will be used directly (for custom uploaders
// or external upload management).
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

	lm := &LoggerManager{
		baseDir: baseDir,
		config:  config,
	}

	// If GCSUploadConfig is provided and no UploadChannel is set, create uploader internally
	if config.GCSUploadConfig != nil && config.UploadChannel == nil {
		uploader, err := NewUploader(*config.GCSUploadConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create uploader: %w", err)
		}

		// Get the upload channel from the uploader
		uploadChan := uploader.GetUploadChannel()

		// Start the uploader
		uploader.Start()

		lm.uploader = uploader
		lm.uploadChannel = uploadChan
		lm.ownUploader = true
	} else {
		// Use provided upload channel (or nil if not using uploads)
		lm.uploadChannel = config.UploadChannel
		lm.ownUploader = false
	}

	return lm, nil
}

// sanitizeEventName validates and sanitizes an event name for use as a filename
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
func (lm *LoggerManager) getOrCreateLogger(eventName string) (*Logger, error) {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return nil, err
	}

	// Fast path: check if logger exists
	if logger, ok := lm.loggers.Load(sanitized); ok {
		return logger.(*Logger), nil
	}

	// Slow path: create new logger
	// Generate file path: {baseDir}/{eventName}.log
	eventLogPath := filepath.Join(lm.baseDir, sanitized+".log")

	// Create config for this event logger (same settings, different file path)
	eventConfig := lm.config
	eventConfig.LogFilePath = eventLogPath
	eventConfig.UploadChannel = lm.uploadChannel // Share upload channel

	// Create new logger
	logger, err := NewLogger(eventConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger for event %s: %w", sanitized, err)
	}

	// Use LoadOrStore to ensure only one logger is created per event
	actual, loaded := lm.loggers.LoadOrStore(sanitized, logger)
	if loaded {
		// Another goroutine created it first, close ours to avoid resource leak
		logger.Close()
		return actual.(*Logger), nil
	}

	return logger, nil
}

// LogBytesWithEvent writes raw byte data to the event-specific logger
func (lm *LoggerManager) LogBytesWithEvent(eventName string, data []byte) {
	logger, err := lm.getOrCreateLogger(eventName)
	if err != nil {
		// Drop log on error
		return
	}
	logger.LogBytes(data)
}

// LogWithEvent writes a string message to the event-specific logger
func (lm *LoggerManager) LogWithEvent(eventName string, message string) {
	logger, err := lm.getOrCreateLogger(eventName)
	if err != nil {
		// Drop log on error
		return
	}
	logger.Log(message)
}

// InitializeEventLogger creates a logger for the specified event if it doesn't exist
func (lm *LoggerManager) InitializeEventLogger(eventName string) error {
	sanitized, err := sanitizeEventName(eventName)
	if err != nil {
		return fmt.Errorf("invalid event name: %w", err)
	}

	// Check if logger already exists
	if _, exists := lm.loggers.Load(sanitized); exists {
		return nil // Already exists, no-op
	}

	// Create logger
	_, err = lm.getOrCreateLogger(sanitized)
	return err
}

// CloseEventLogger closes and removes the logger for the specified event
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

	// Close the logger
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
// If LoggerManager created an uploader internally, it will also be stopped
func (lm *LoggerManager) Close() error {
	var firstErr error

	// Close all loggers first
	lm.loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		if err := logger.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		return true // continue iteration
	})

	// Stop uploader if we created it internally
	if lm.ownUploader && lm.uploader != nil {
		lm.uploader.Stop()
	}

	return firstErr
}

// GetAggregatedStats returns aggregated statistics across all loggers
func (lm *LoggerManager) GetAggregatedStats() (totalLogs, droppedLogs, bytesWritten, flushes, flushErrors, setSwaps int64) {
	lm.loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		t, d, b, f, fe, s := logger.GetStatsSnapshot()
		totalLogs += t
		droppedLogs += d
		bytesWritten += b
		flushes += f
		flushErrors += fe
		setSwaps += s
		return true
	})
	return
}

// GetAggregatedFlushMetrics returns aggregated flush metrics across all loggers
func (lm *LoggerManager) GetAggregatedFlushMetrics() FlushMetrics {
	var totalFlushDuration, maxFlushDuration int64
	var totalWriteDuration, maxWriteDuration int64
	var totalPwritevDuration, maxPwritevDuration int64
	var totalFlushes int64

	lm.loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		metrics := logger.GetFlushMetrics()
		flushes := logger.stats.Flushes.Load()

		if flushes > 0 {
			totalFlushDuration += metrics.AvgFlushDuration.Nanoseconds() * flushes
			if metrics.MaxFlushDuration.Nanoseconds() > maxFlushDuration {
				maxFlushDuration = metrics.MaxFlushDuration.Nanoseconds()
			}

			totalWriteDuration += metrics.AvgWriteDuration.Nanoseconds() * flushes
			if metrics.MaxWriteDuration.Nanoseconds() > maxWriteDuration {
				maxWriteDuration = metrics.MaxWriteDuration.Nanoseconds()
			}

			totalPwritevDuration += metrics.AvgPwritevDuration.Nanoseconds() * flushes
			if metrics.MaxPwritevDuration.Nanoseconds() > maxPwritevDuration {
				maxPwritevDuration = metrics.MaxPwritevDuration.Nanoseconds()
			}

			totalFlushes += flushes
		}
		return true
	})

	if totalFlushes == 0 {
		return FlushMetrics{}
	}

	avgFlushDuration := time.Duration(totalFlushDuration / totalFlushes)
	avgWriteDuration := time.Duration(totalWriteDuration / totalFlushes)
	avgPwritevDuration := time.Duration(totalPwritevDuration / totalFlushes)

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
		MaxFlushDuration:   time.Duration(maxFlushDuration),
		AvgWriteDuration:   avgWriteDuration,
		MaxWriteDuration:   time.Duration(maxWriteDuration),
		WritePercent:       writePercent,
		AvgPwritevDuration: avgPwritevDuration,
		MaxPwritevDuration: time.Duration(maxPwritevDuration),
		PwritevPercent:     pwritevPercent,
	}
}

// GetUploadStats returns upload statistics if an uploader is configured
// Returns nil if no uploader is configured
func (lm *LoggerManager) GetUploadStats() *Stats {
	if lm.uploader == nil {
		return nil
	}
	stats := lm.uploader.GetStats()
	return &stats
}
