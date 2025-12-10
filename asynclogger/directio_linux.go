//go:build linux

package asynclogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// alignmentSize is the required alignment for O_DIRECT on Linux
// For ext4 filesystem, this must be 4096 bytes (4KB), not 512 bytes!
// O_DIRECT requires alignment to filesystem block size, not just sector size
const alignmentSize = 4096

// openDirectIO opens a file with O_DIRECT and O_DSYNC flags
// O_DIRECT: Bypasses OS page cache, writes directly to disk
// O_DSYNC: Each write automatically syncs data to disk (eliminates need for explicit sync)
// O_TRUNC: Truncates file to ensure it starts at offset 0 (4096-byte aligned) for O_DIRECT compliance
// Note: O_APPEND is removed to allow manual offset tracking for file rotation
func openDirectIO(path string) (*os.File, int64, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, 0, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open with O_DIRECT, O_DSYNC, O_WRONLY, O_CREAT, O_TRUNC
	// O_TRUNC ensures file starts at offset 0 (aligned) for O_DIRECT compliance
	// This avoids alignment issues when opening existing files
	fd, err := syscall.Open(path,
		syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC|syscall.O_DIRECT|syscall.O_DSYNC,
		0644)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file with O_DIRECT: %w", err)
	}

	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		syscall.Close(fd)
		return nil, 0, fmt.Errorf("failed to create file descriptor")
	}

	// File is always truncated, so offset is always 0 (aligned)
	return file, 0, nil
}

// allocAlignedBuffer allocates a byte slice aligned to filesystem block size (4096 bytes for ext4) for O_DIRECT
func allocAlignedBuffer(size int) []byte {
	// Round up to alignment
	alignedSize := ((size + alignmentSize - 1) / alignmentSize) * alignmentSize

	// Allocate extra space to ensure we can align
	buf := make([]byte, alignedSize+alignmentSize)

	// Get the address of the first byte
	addr := uintptr(unsafe.Pointer(&buf[0]))

	// Calculate offset needed for alignment
	offset := int(alignmentSize - (addr % alignmentSize))
	if offset == alignmentSize {
		offset = 0
	}

	// Return aligned slice
	return buf[offset : offset+alignedSize]
}

// writevAlignedWithOffset writes multiple buffers to file at a specific offset using vectored I/O
// Uses unix.Pwritev() - NO memory copy, just pointers! Maintains vectored I/O efficiency with offset control
// OPTIMIZATION: Buffers are already address and size-aligned (4096 bytes) from NewBuffer(),
// so no padding or buffer pooling is needed - true zero-copy!
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

	// Buffers are already aligned (address and size), so we can write directly!
	// Single vectored write syscall at specific offset - kernel reads from multiple buffers!
	// unix.Pwritev takes [][]byte directly with offset - NO iovec creation needed, NO copying!
	n, err := unix.Pwritev(fd, nonEmptyBuffers, offset)
	if err != nil {
		return n, fmt.Errorf("vectored I/O write failed: %w", err)
	}

	// Return actual bytes written by Pwritev
	// Buffers are already 4096-byte aligned, so offset stays aligned after write
	return n, nil
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
	// Remove .log extension if present
	if strings.HasSuffix(baseName, ".log") {
		baseName = strings.TrimSuffix(baseName, ".log")
	}

	if baseName == "" {
		return "", "", fmt.Errorf("invalid file path: base name is empty after extraction")
	}

	return dir, baseName, nil
}

// FileWriter manages file handles, offset tracking, and rotation for Direct I/O writes
// Encapsulates all file management logic, keeping logger.go unaware of rotation details
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
	if err := unix.Fsync(fw.fd); err != nil {
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

	// Write using vectored I/O at specific offset (Linux uses fd for Pwritev)
	n, err := writevAlignedWithOffset(fw.fd, buffers, offset)
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
		if err := unix.Fsync(fw.fd); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to sync current file: %w", err)
		}
		if err := fw.file.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close current file: %w", err)
		}
	}

	// Close next file if it exists
	if fw.nextFile != nil {
		if err := fw.nextFile.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close next file: %w", err)
		}
	}

	return firstErr
}
