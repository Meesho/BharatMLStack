package asyncloguploader

import (
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
