package asyncloguploader

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Meesho/go-core/metric"
)

// LoggerManager manages multiple Logger instances, one per event name
// Each event writes to its own log file (e.g., payment.log, login.log)
type LoggerManager struct {
	loggers     sync.Map     // eventName (string) -> *Logger
	baseDir     string       // Base directory for log files
	config      Config       // Base config (shared settings)
	uploader    *Uploader    // Optional: GCS uploader (created internally if GCSUploadConfig provided)
	ownUploader bool         // True if uploader was created internally (needs cleanup)
	ssdManager  *SSDManager  // Optional: SSD lifecycle manager (nil when SSDConfig not set)
}

// NewLoggerManager creates a new LoggerManager
// The base directory is extracted from config.LogFilePath
//
// If GCSUploadConfig is provided, LoggerManager will automatically create and manage
// an Uploader internally. The uploader scans baseDir for .log files every ScanInterval
// and uploads them to GCS. It is started automatically and stopped when Close() is called.
func NewLoggerManager(config Config) (*LoggerManager, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		metric.Incr(MetricLoggerInitializationFailed, getBaseTags(config.MetricTags))
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	tags := getBaseTags(config.MetricTags)

	var baseDir string
	var ssdManager *SSDManager

	// SSD lifecycle: claim an SSD and use it as the base directory
	if config.SSDConfig != nil {
		mgr, err := NewSSDManager(*config.SSDConfig, tags)
		if err != nil {
			metric.Incr(MetricLoggerInitializationFailed, tags)
			return nil, fmt.Errorf("ssd claim failed: %w", err)
		}
		ssdManager = mgr
		baseDir = mgr.SSDPath

		// Snapshot orphan .log.tmp files BEFORE any writers are created,
		// then rename them to .log in the background so the Uploader picks them up.
		orphanFiles := listOrphanTmpFiles(baseDir)
		if len(orphanFiles) > 0 {
			go recoverOrphanTmpFiles(baseDir, orphanFiles, tags)
		}
	} else {
		// No SSD config — extract base directory from LogFilePath
		cleanedPath := filepath.Clean(config.LogFilePath)
		hasFileExtension := filepath.Ext(cleanedPath) != ""
		if hasFileExtension {
			baseDir = filepath.Dir(cleanedPath)
		} else {
			baseDir = cleanedPath
		}
		if baseDir == "." || baseDir == "" {
			baseDir = "."
		}
	}

	lm := &LoggerManager{
		baseDir:    baseDir,
		config:     config,
		ssdManager: ssdManager,
	}

	// If GCSUploadConfig is provided, create uploader that scans baseDir for .log files
	if config.GCSUploadConfig != nil {
		uploader, err := NewUploader(*config.GCSUploadConfig, baseDir, tags)
		if err != nil {
			// Clean up SSD claim if uploader creation fails
			if ssdManager != nil {
				ssdManager.CancelRenewal()
				ssdManager.Release()
			}
			return nil, fmt.Errorf("failed to create uploader: %w", err)
		}

		uploader.Start()
		lm.uploader = uploader
		lm.ownUploader = true
	}

	metric.Incr(MetricLoggerInitialized, tags)
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

	// Create new logger (pass sanitized event name for tag propagation)
	logger, err := NewLogger(eventConfig, sanitized)
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
	metric.Incr(MetricLogBytes, MergeMetricTags(lm.config.MetricTags, eventName))
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

// Close gracefully shuts down all loggers, flushing all pending data.
// If LoggerManager created an uploader internally, it will also be stopped.
// SSD claim is released as the absolute last action.
func (lm *LoggerManager) Close() error {
	var firstErr error

	// 1. Cancel SSD renewal — stop touching claim file
	if lm.ssdManager != nil {
		lm.ssdManager.CancelRenewal()
	}

	// 2. Close all loggers (drains buffers, rotates .tmp → .log)
	lm.loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		if err := logger.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		return true // continue iteration
	})

	// 3. Stop uploader (does final scan + upload before exiting)
	if lm.ownUploader && lm.uploader != nil {
		lm.uploader.Stop()
	}

	// 4. Release SSD claim — absolute last action
	if lm.ssdManager != nil {
		if err := lm.ssdManager.Release(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
