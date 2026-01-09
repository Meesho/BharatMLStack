//go:build linux

package asyncloguploader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// alignmentSize is the required alignment for O_DIRECT on Linux (ext4 filesystem)
	// Must be 4096 bytes (4KB), not 512 bytes!
	alignmentSize = 4096
)

// SizeFileWriter manages file handles, offset tracking, and size-based rotation for Direct I/O writes
type SizeFileWriter struct {
	// Current file
	file        *os.File
	fd          int
	filePath    string
	fileOffset  atomic.Int64
	maxFileSize int64 // Maximum file size before rotation

	// Next file (for rotation)
	nextFile     *os.File
	nextFd       int
	nextFilePath string

	// Configuration
	baseDir             string
	baseFileName        string
	preallocateFileSize int64 // Size to preallocate using fallocate

	// Mutex for rotation operations (only held during rotation)
	rotationMu sync.Mutex

	// Last Pwritev duration (for metrics tracking)
	lastPwritevDuration atomic.Int64 // Nanoseconds

	// Channel for completed files (for GCS upload)
	completedFileChan chan<- string
}

// NewSizeFileWriter creates a new SizeFileWriter with the given configuration
// completedFileChan is optional - if provided, completed files will be sent to this channel for upload
func NewSizeFileWriter(config Config, completedFileChan chan<- string) (*SizeFileWriter, error) {
	// Extract base directory and filename
	baseDir, baseFileName, err := extractBasePathSize(config.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract base path: %w", err)
	}

	// Generate timestamped filename for initial file (consistent naming)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	initialPath := filepath.Join(baseDir, fmt.Sprintf("%s_%s.log", baseFileName, timestamp))

	// Open initial file with preallocation (always starts at offset 0 for new files)
	file, err := openDirectIOSize(initialPath, config.PreallocateFileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to open initial file: %w", err)
	}

	fw := &SizeFileWriter{
		file:                file,
		fd:                  int(file.Fd()),
		filePath:            initialPath,
		maxFileSize:         config.MaxFileSize,
		baseDir:             baseDir,
		baseFileName:        baseFileName,
		preallocateFileSize: config.PreallocateFileSize,
		completedFileChan:   completedFileChan,
	}

	// New files always start at offset 0
	fw.fileOffset.Store(0)

	return fw, nil
}

// WriteVectored writes multiple buffers to the file using vectored I/O
// Handles rotation automatically before writing
func (fw *SizeFileWriter) WriteVectored(buffers [][]byte) (int, error) {
	// Fast path: skip if no data to write
	if len(buffers) == 0 {
		return 0, nil
	}

	// Check and perform rotation if needed
	if err := fw.rotateIfNeeded(); err != nil {
		return 0, fmt.Errorf("rotation failed: %w", err)
	}

	// Get current offset
	offset := fw.fileOffset.Load()

	// Write using vectored I/O at specific offset
	pwritevStart := time.Now()
	n, err := writevAlignedWithOffset(fw.fd, buffers, offset)
	pwritevDuration := time.Since(pwritevStart)

	// Store write duration for metrics
	fw.lastPwritevDuration.Store(pwritevDuration.Nanoseconds())

	if err != nil {
		return n, err
	}

	// Update offset atomically after successful write
	fw.fileOffset.Add(int64(n))

	return n, nil
}

// GetLastPwritevDuration returns the duration of the last Pwritev syscall
func (fw *SizeFileWriter) GetLastPwritevDuration() time.Duration {
	return time.Duration(fw.lastPwritevDuration.Load())
}

// Close syncs and closes the current file, and closes next file if it exists
func (fw *SizeFileWriter) Close() error {
	var firstErr error

	// If nextFile exists, it means rotation was in progress
	// We need to complete the rotation: swap files, then close both
	if fw.nextFile != nil && fw.file != nil {
		// Complete the rotation by swapping files
		// This will send the current file to upload channel
		if err := fw.swapFiles(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to complete rotation during close: %w", err)
		}
		// After swap, nextFile becomes current file, and old current file is uploaded
	}

	// Now close the current file (which might be the swapped file or original current file)
	if fw.file != nil {
		// Check if file has data (offset > 0 means data was written)
		hasData := fw.fileOffset.Load() > 0

		// Store file path before closing (for upload)
		completedFilePath := fw.filePath

		// Get actual written size
		actualSize := fw.fileOffset.Load()

		// Sync file to ensure all data is written before closing
		if hasData && fw.fd > 0 {
			if err := unix.Fsync(fw.fd); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("failed to sync file: %w", err)
			}
		}

		// Truncate file to actual written size (removes preallocated space)
		// This is fast for sparse files (metadata-only operation)
		if hasData && actualSize > 0 && fw.fd > 0 {
			if err := unix.Ftruncate(fw.fd, actualSize); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("failed to truncate file to actual size: %w", err)
			}
		}

		// Close current file
		if err := fw.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}

		// Send completed file to upload channel (non-blocking) if it has data
		if hasData && fw.completedFileChan != nil {
			select {
			case fw.completedFileChan <- completedFilePath:
				// Successfully sent to channel
			default:
				// Channel full - log warning but don't block close
				fmt.Printf("[WARNING] Upload channel full, skipping upload for %s\n", completedFilePath)
			}
		}

		fw.file = nil
		fw.fd = 0
	}

	// Clean up nextFile if it still exists (shouldn't happen after swap, but be safe)
	if fw.nextFile != nil {
		if err := fw.nextFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		fw.nextFile = nil
		fw.nextFd = 0
		fw.nextFilePath = ""
	}

	return firstErr
}

// rotateIfNeeded checks if rotation is needed based on file size and performs it if necessary
func (fw *SizeFileWriter) rotateIfNeeded() error {
	// If rotation is disabled (maxFileSize is 0), skip
	if fw.maxFileSize <= 0 {
		return nil
	}

	// Acquire rotation mutex once (prevents concurrent rotations and ensures atomic checks)
	fw.rotationMu.Lock()
	defer fw.rotationMu.Unlock()

	// Get current offset (after acquiring lock to ensure consistency)
	currentOffset := fw.fileOffset.Load()

	// Check if we've actually exceeded the max file size (need to swap immediately)
	if currentOffset >= fw.maxFileSize {
		// Ensure next file exists
		if fw.nextFile == nil {
			if err := fw.createNextFile(); err != nil {
				return fmt.Errorf("failed to create next file: %w", err)
			}
		}

		// Swap to next file
		if err := fw.swapFiles(); err != nil {
			return fmt.Errorf("failed to swap files: %w", err)
		}
		return nil
	}

	// Check if we're approaching max file size (proactive rotation at 90%)
	if currentOffset >= int64(float64(fw.maxFileSize)*0.9) {
		// If next file doesn't exist, create it proactively
		if fw.nextFile == nil {
			if err := fw.createNextFile(); err != nil {
				// Don't fail the write if proactive creation fails - we'll try again next time
				// This prevents fallocate blocking from causing write failures
				return nil
			}
		}
	}

	return nil
}

// createNextFile creates a new file for rotation with preallocation
func (fw *SizeFileWriter) createNextFile() error {
	// Generate timestamped filename: {baseFileName}_{YYYY-MM-DD_HH-MM-SS}.log
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	nextPath := filepath.Join(fw.baseDir, fmt.Sprintf("%s_%s.log", fw.baseFileName, timestamp))

	// Try to open new file with preallocation
	file, err := openDirectIOSize(nextPath, fw.preallocateFileSize)
	if err != nil {
		// If preallocation fails, try creating file without preallocation as fallback
		file, err = openDirectIOSize(nextPath, 0)
		if err != nil {
			return fmt.Errorf("failed to open next file (with and without preallocation): %w", err)
		}
		// Log warning but continue (file will work, just without preallocation)
		fmt.Printf("[WARNING] Failed to preallocate %d bytes for %s, continuing without preallocation\n",
			fw.preallocateFileSize, nextPath)
	}

	// Store next file details
	fw.nextFile = file
	fw.nextFd = int(file.Fd())
	fw.nextFilePath = nextPath

	return nil
}

// swapFiles atomically swaps from current file to next file
func (fw *SizeFileWriter) swapFiles() error {
	if fw.nextFile == nil || fw.nextFd == 0 || fw.nextFilePath == "" {
		return fmt.Errorf("next file is not set")
	}

	// Defensive check: ensure current file is valid
	if fw.file == nil {
		return fmt.Errorf("current file is nil")
	}

	// Sync current file to ensure all data is written
	if err := unix.Fsync(fw.fd); err != nil {
		return fmt.Errorf("failed to sync current file: %w", err)
	}

	// Get actual written size
	actualSize := fw.fileOffset.Load()

	// Truncate file to actual written size (removes preallocated space)
	// This is fast for sparse files (metadata-only operation)
	if actualSize > 0 {
		if err := unix.Ftruncate(fw.fd, actualSize); err != nil {
			return fmt.Errorf("failed to truncate file to actual size: %w", err)
		}
	}

	// Store current file path before closing (for upload)
	completedFilePath := fw.filePath

	// Close current file
	if err := fw.file.Close(); err != nil {
		return fmt.Errorf("failed to close current file: %w", err)
	}

	// Send completed file to upload channel (non-blocking)
	if fw.completedFileChan != nil {
		select {
		case fw.completedFileChan <- completedFilePath:
			// Successfully sent to channel
		default:
			// Channel full - log warning but don't block rotation
			fmt.Printf("[WARNING] Upload channel full, skipping upload for %s\n", completedFilePath)
		}
	}

	// Swap next file to current
	fw.file = fw.nextFile
	fw.fd = fw.nextFd
	fw.filePath = fw.nextFilePath
	fw.fileOffset.Store(0) // Reset offset for new file

	// Clear next file fields
	fw.nextFile = nil
	fw.nextFd = 0
	fw.nextFilePath = ""

	return nil
}

// openDirectIOSize opens a file with O_DIRECT and O_DSYNC flags, preallocating with fallocate
// Returns the file and error. New files always start at offset 0.
func openDirectIOSize(path string, preallocateSize int64) (*os.File, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Align preallocate size to filesystem block size
	alignedSize := alignUp(preallocateSize, alignmentSize)

	// Open with O_DIRECT, O_DSYNC, O_WRONLY, O_CREAT, O_TRUNC using unix package
	fd, err := unix.Open(path,
		unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC|unix.O_DIRECT|unix.O_DSYNC,
		0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file with O_DIRECT: %w", err)
	}

	// Preallocate file using fallocate
	if preallocateSize > 0 {
		if err := unix.Fallocate(fd, 0, 0, alignedSize); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("failed to preallocate file with fallocate: %w", err)
		}
	}

	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		unix.Close(fd)
		return nil, fmt.Errorf("failed to create file descriptor")
	}

	// File is always truncated and preallocated, so offset is always 0
	return file, nil
}

// writevAlignedWithOffset writes multiple buffers to file at a specific offset using vectored I/O
func writevAlignedWithOffset(fd int, buffers [][]byte, offset int64) (int, error) {
	if len(buffers) == 0 {
		return 0, nil
	}

	// Filter out empty buffers
	nonEmptyBuffers := make([][]byte, 0, len(buffers))
	for _, buf := range buffers {
		if len(buf) > 0 {
			nonEmptyBuffers = append(nonEmptyBuffers, buf)
		}
	}

	if len(nonEmptyBuffers) == 0 {
		return 0, nil
	}

	// Use unix.Pwritev for vectored I/O
	n, err := unix.Pwritev(fd, nonEmptyBuffers, offset)
	if err != nil {
		return n, fmt.Errorf("vectored I/O write failed: %w", err)
	}

	return n, nil
}

// alignUp rounds n up to the next multiple of align (power of 2)
func alignUp(n, align int64) int64 {
	return (n + align - 1) &^ (align - 1)
}

// extractBasePathSize extracts directory and base filename from a full file path
func extractBasePathSize(fullPath string) (dir, baseName string, err error) {
	dir = filepath.Dir(fullPath)
	if dir == "." || dir == "" {
		dir = "."
	}

	baseName = filepath.Base(fullPath)
	// Remove .log extension if present
	if strings.HasSuffix(baseName, ".log") {
		baseName = strings.TrimSuffix(baseName, ".log")
	}

	if baseName == "" {
		return "", "", fmt.Errorf("invalid file path: base name is empty after extraction")
	}

	return dir, baseName, nil
}
