//go:build !linux

package asynclogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// alignmentSize is the required alignment (512 bytes for compatibility)
const alignmentSize = 512

// openDirectIO opens a file without O_DIRECT (fallback for non-Linux systems)
// Note: This is for testing only. Production deployments should use Linux.
// Returns file, initial offset, and error
func openDirectIO(path string) (*os.File, int64, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, 0, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file without O_DIRECT on non-Linux systems
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}

	// Get initial file size if file exists
	var initialOffset int64
	if stat, err := os.Stat(path); err == nil {
		initialOffset = stat.Size()
	}

	return file, initialOffset, nil
}

// allocAlignedBuffer allocates a byte slice for non-Linux systems
// On non-Linux systems, alignment is not strictly required
func allocAlignedBuffer(size int) []byte {
	// Round up to alignment for consistency
	alignedSize := ((size + alignmentSize - 1) / alignmentSize) * alignmentSize
	return make([]byte, alignedSize)
}

// writeAligned writes data to file on non-Linux systems
func writeAligned(file *os.File, data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// On non-Linux systems, just write normally
	n, err := file.Write(data)
	if err != nil {
		return n, fmt.Errorf("write failed: %w", err)
	}

	return n, nil
}

// writevAlignedWithOffset writes multiple buffers to file at a specific offset
// On non-Linux systems, consolidates and writes as a single buffer
// Note: For non-Linux, we use *os.File directly instead of fd
func writevAlignedWithOffset(file *os.File, buffers [][]byte, offset int64) (int, error) {
	if len(buffers) == 0 {
		return 0, nil
	}

	// Calculate total size
	totalSize := 0
	for _, buf := range buffers {
		totalSize += len(buf)
	}

	if totalSize == 0 {
		return 0, nil
	}

	// Consolidate all buffers
	consolidatedBuf := make([]byte, 0, totalSize)
	for _, buf := range buffers {
		consolidatedBuf = append(consolidatedBuf, buf...)
	}

	// Seek to offset and write
	if _, err := file.Seek(offset, 0); err != nil {
		return 0, fmt.Errorf("failed to seek: %w", err)
	}

	n, err := file.Write(consolidatedBuf)
	if err != nil {
		return n, fmt.Errorf("batched write failed: %w", err)
	}

	return n, nil
}

// isAligned checks if a size is aligned to the required boundary
func isAligned(size int) bool {
	return size%alignmentSize == 0
}

// alignSize rounds up size to the nearest alignment boundary
func alignSize(size int) int {
	return ((size + alignmentSize - 1) / alignmentSize) * alignmentSize
}

// extractBasePath extracts directory and base filename from a full file path
// Returns directory, base filename without extension, and error
func extractBasePath(fullPath string) (dir, baseName string, err error) {
	dir = filepath.Dir(fullPath)
	if dir == "." || dir == "" {
		dir = "."
	}

	baseName = filepath.Base(fullPath)
	// Remove .log extension if present (TrimSuffix is safe even if suffix doesn't exist)
	baseName = strings.TrimSuffix(baseName, ".log")

	if baseName == "" {
		return "", "", fmt.Errorf("invalid file path: base name is empty after extraction")
	}

	return dir, baseName, nil
}

// FileWriter manages file handles, offset tracking, and rotation for non-Linux systems
// Note: Rotation is simplified on non-Linux (no O_DIRECT support)
type FileWriter struct {
	// Current file
	file          *os.File
	fd            int
	filePath      string
	fileOffset    atomic.Int64
	fileCreatedAt time.Time

	// Next file (for rotation)
	nextFile     *os.File
	nextFd       int
	nextFilePath string

	// Configuration
	baseDir          string
	baseFileName     string
	rotationInterval time.Duration

	// Mutex for rotation operations (only held during rotation)
	rotationMu sync.Mutex
}

// NewFileWriter creates a new FileWriter with the given configuration
func NewFileWriter(config Config) (*FileWriter, error) {
	// Extract base directory and filename
	baseDir, baseFileName, err := extractBasePath(config.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract base path: %w", err)
	}

	// Open initial file
	file, initialOffset, err := openDirectIO(config.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open initial file: %w", err)
	}

	fw := &FileWriter{
		file:             file,
		fd:               int(file.Fd()),
		filePath:         config.LogFilePath,
		fileCreatedAt:    time.Now(),
		baseDir:          baseDir,
		baseFileName:     baseFileName,
		rotationInterval: config.RotationInterval,
	}

	// Set initial offset (0 for new files, or existing file size)
	fw.fileOffset.Store(initialOffset)

	return fw, nil
}

// rotateIfNeeded checks if rotation is needed and performs it if necessary
func (fw *FileWriter) rotateIfNeeded() error {
	// If rotation is disabled (interval is 0), skip
	if fw.rotationInterval <= 0 {
		return nil
	}

	// Check if rotation is needed
	if time.Since(fw.fileCreatedAt) < fw.rotationInterval {
		return nil
	}

	// Acquire rotation mutex to prevent concurrent rotations
	fw.rotationMu.Lock()
	defer fw.rotationMu.Unlock()

	// Double-check after acquiring lock (another goroutine might have rotated)
	if time.Since(fw.fileCreatedAt) < fw.rotationInterval {
		return nil
	}

	// If next file doesn't exist, create it
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

// createNextFile creates a new file for rotation
func (fw *FileWriter) createNextFile() error {
	// Generate timestamped filename: {baseFileName}_{YYYY-MM-DD_HH-MM-SS}.log
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	nextPath := filepath.Join(fw.baseDir, fmt.Sprintf("%s_%s.log", fw.baseFileName, timestamp))

	// Open new file
	file, initialOffset, err := openDirectIO(nextPath)
	if err != nil {
		return fmt.Errorf("failed to open next file: %w", err)
	}

	// Store next file details
	fw.nextFile = file
	fw.nextFd = int(file.Fd())
	fw.nextFilePath = nextPath

	// Next file should start at offset 0 (new file)
	if initialOffset != 0 {
		return fmt.Errorf("next file should be empty, but has size %d", initialOffset)
	}

	return nil
}

// swapFiles atomically swaps from current file to next file
func (fw *FileWriter) swapFiles() error {
	if fw.nextFile == nil || fw.nextFd == 0 || fw.nextFilePath == "" {
		return fmt.Errorf("next file is not set")
	}

	// Defensive check: ensure current file is valid (fast pointer check, no performance impact)
	if fw.file == nil {
		return fmt.Errorf("current file is nil")
	}

	// Sync current file to ensure all data is written
	if err := fw.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync current file: %w", err)
	}

	// Close current file
	if err := fw.file.Close(); err != nil {
		return fmt.Errorf("failed to close current file: %w", err)
	}

	// Swap next file to current
	fw.file = fw.nextFile
	fw.fd = fw.nextFd
	fw.filePath = fw.nextFilePath
	fw.fileOffset.Store(0) // Reset offset for new file
	fw.fileCreatedAt = time.Now()

	// Clear next file fields
	fw.nextFile = nil
	fw.nextFd = 0
	fw.nextFilePath = ""

	return nil
}

// WriteVectored writes multiple buffers to the file using vectored I/O
// Handles rotation automatically before writing
func (fw *FileWriter) WriteVectored(buffers [][]byte) (int, error) {
	// Fast path: skip if no data to write (defensive check, no performance impact)
	if len(buffers) == 0 {
		return 0, nil
	}

	// Check and perform rotation if needed
	if err := fw.rotateIfNeeded(); err != nil {
		return 0, fmt.Errorf("rotation failed: %w", err)
	}

	// Get current offset
	offset := fw.fileOffset.Load()

	// Write using vectored I/O at specific offset (non-Linux uses file directly)
	n, err := writevAlignedWithOffset(fw.file, buffers, offset)
	if err != nil {
		return n, err
	}

	// Update offset atomically after successful write
	fw.fileOffset.Add(int64(n))

	return n, nil
}

// Close syncs and closes the current file, and closes next file if it exists
func (fw *FileWriter) Close() error {
	var firstErr error

	// Sync and close current file
	if fw.file != nil {
		if err := fw.file.Sync(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to sync current file: %w", err)
		}
		if err := fw.file.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close current file: %w", err)
		}
		fw.file = nil // Clear reference after close
	}

	// Close next file if it exists
	if fw.nextFile != nil {
		if err := fw.nextFile.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close next file: %w", err)
		}
	}

	return firstErr
}
