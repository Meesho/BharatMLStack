//go:build !linux

package asynclogger

import (
	"fmt"
	"os"
)

// alignmentSize is the required alignment (512 bytes for compatibility)
const alignmentSize = 512

// openDirectIO opens a file without O_DIRECT (fallback for non-Linux systems)
// Note: This is for testing only. Production deployments should use Linux.
func openDirectIO(path string) (*os.File, error) {
	// Open file without O_DIRECT on non-Linux systems
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// allocAlignedBuffer allocates a byte slice for non-Linux systems
// On non-Linux systems, alignment is not strictly required
func allocAlignedBuffer(size int) []byte {
	// Round up to alignment for consistency
	alignedSize := ((size + alignmentSize - 1) / alignmentSize) * alignmentSize
	return make([]byte, alignedSize)
}

// writevAligned writes multiple buffers to file in a single batched operation
// On non-Linux systems, consolidates and writes as a single buffer
func writevAligned(file *os.File, buffers [][]byte) (int, error) {
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

	// Single write
	n, err := file.Write(consolidatedBuf)
	if err != nil {
		return n, fmt.Errorf("batched write failed: %w", err)
	}

	return n, nil
}

// alignSize rounds up size to the nearest alignment boundary
func alignSize(size int) int {
	return ((size + alignmentSize - 1) / alignmentSize) * alignmentSize
}
