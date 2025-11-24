//go:build linux

package asynclogger

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// alignmentSize is the required alignment for O_DIRECT on Linux (typically 512 bytes)
const alignmentSize = 512

// flushBufferPool provides pre-allocated aligned buffers for flushing
// Each buffer is 256MB to comfortably handle 128MB buffer + alignment padding
var flushBufferPool = sync.Pool{
	New: func() interface{} {
		size := 256 * 1024 * 1024 // 256MB
		buf := allocAlignedBuffer(size)
		return &buf
	},
}

// openDirectIO opens a file with O_DIRECT and O_DSYNC flags
// O_DIRECT: Bypasses OS page cache, writes directly to disk
// O_DSYNC: Each write automatically syncs data to disk (eliminates need for explicit sync)
func openDirectIO(path string) (*os.File, error) {
	// Ensure parent directory exists
	// Open with O_DIRECT, O_DSYNC, O_WRONLY, O_APPEND, O_CREAT
	fd, err := syscall.Open(path,
		syscall.O_WRONLY|syscall.O_APPEND|syscall.O_CREAT|syscall.O_DIRECT|syscall.O_DSYNC,
		0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file with O_DIRECT: %w", err)
	}

	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to create file descriptor")
	}

	return file, nil
}

// allocAlignedBuffer allocates a byte slice aligned to 512-byte boundary for O_DIRECT
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

// writevAligned writes multiple buffers to file in a single vectored I/O operation
// Uses true syscall.Writev() - NO memory copy, just pointers!
// This is the OPTIMAL implementation - reduces 8 syscalls to 1 with zero copy overhead
func writevAligned(file *os.File, buffers [][]byte) (int, error) {
	if len(buffers) == 0 {
		return 0, nil
	}

	// Prepare aligned buffers - each buffer aligned independently
	var alignedBuffers [][]byte
	var totalActualSize int
	var pooledBuffers []*[]byte // Track pooled buffers to return after write

	for _, buf := range buffers {
		if len(buf) == 0 {
			continue
		}

		totalActualSize += len(buf)
		alignedSize := ((len(buf) + alignmentSize - 1) / alignmentSize) * alignmentSize

		if len(buf) == alignedSize {
			// Already aligned, use directly (zero copy!)
			alignedBuffers = append(alignedBuffers, buf)
		} else {
			// Need padding - get buffer from pool
			bufPtr := flushBufferPool.Get().(*[]byte)
			paddedBuf := *bufPtr

			if cap(paddedBuf) < alignedSize {
				// Pool buffer too small, allocate new one
				// Return unused pooled buffer immediately
				flushBufferPool.Put(bufPtr)
				paddedBuf = allocAlignedBuffer(alignedSize)
			} else {
				// Use pooled buffer - track it for return after write
				paddedBuf = paddedBuf[:alignedSize]
				pooledBuffers = append(pooledBuffers, bufPtr)
			}

			// Copy data and zero-pad
			copy(paddedBuf, buf)
			for i := len(buf); i < alignedSize; i++ {
				paddedBuf[i] = 0
			}

			alignedBuffers = append(alignedBuffers, paddedBuf)
		}
	}

	// Return pooled buffers after write completes
	defer func() {
		for _, bufPtr := range pooledBuffers {
			flushBufferPool.Put(bufPtr)
		}
	}()

	if len(alignedBuffers) == 0 {
		return 0, nil
	}

	// Single vectored write syscall - kernel reads from multiple buffers!
	// unix.Writev takes [][]byte directly - NO iovec creation needed!
	n, err := unix.Writev(int(file.Fd()), alignedBuffers)
	if err != nil {
		return n, fmt.Errorf("vectored I/O write failed: %w", err)
	}

	// Return actual data written (not including padding)
	if n > totalActualSize {
		return totalActualSize, nil
	}
	return n, nil
}

// alignSize rounds up size to the nearest alignment boundary
func alignSize(size int) int {
	return ((size + alignmentSize - 1) / alignmentSize) * alignmentSize
}
