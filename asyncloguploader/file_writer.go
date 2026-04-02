package asyncloguploader

import (
	"fmt"
	"os"
	"time"
)

// FileWriter defines the interface for file writing operations
type FileWriter interface {
	// WriteVectored writes multiple buffers to the file using vectored I/O
	// Returns the number of bytes written and any error
	WriteVectored(buffers [][]byte) (int, error)

	// GetLastPwritevDuration returns the duration of the last Pwritev syscall in nanoseconds
	GetLastPwritevDuration() time.Duration

	// Close closes the file writer and releases resources
	Close() error
}

// getHostname returns the system hostname, or empty string on error.
// In Kubernetes, the hostname equals the pod name.
func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

// generateFileName builds the log filename from base name, optional pod name, and timestamp.
// With podName:    {baseName}--{podName}_{timestamp}.log.tmp
// Without podName: {baseName}_{timestamp}.log.tmp
func generateFileName(baseName, podName, timestamp string) string {
	if podName != "" {
		return fmt.Sprintf("%s--%s_%s.log.tmp", baseName, podName, timestamp)
	}
	return fmt.Sprintf("%s_%s.log.tmp", baseName, timestamp)
}
