//go:build linux
// +build linux

package iouring

import (
	"fmt"
)

// IoUringWriter wraps a raw IoUring ring and exposes only the write API.
type IoUringWriter struct {
	ring       *IoUring
	maxBatchSz int
}

// NewIoUringWriter creates an IoUringWriter backed by a new io_uring ring.
func NewIoUringWriter(entries uint32, flags uint32) (*IoUringWriter, error) {
	ring, err := NewIoUring(entries, flags)
	if err != nil {
		return nil, fmt.Errorf("io_uring writer init: %w", err)
	}
	return &IoUringWriter{
		ring:       ring,
		maxBatchSz: int(ring.sqEntries),
	}, nil
}

// MaxBatchSize returns the maximum number of SQEs that can be submitted in
// a single SubmitWriteBatch call.
func (w *IoUringWriter) MaxBatchSize() int {
	return w.maxBatchSz
}

// SubmitWriteBatch submits N pwrite operations in a single io_uring_enter
// call and waits for all completions. Thread-safe.
func (w *IoUringWriter) SubmitWriteBatch(fd int, bufs [][]byte, offsets []uint64) ([]int, error) {
	return w.ring.SubmitWriteBatch(fd, bufs, offsets)
}

// Close releases the underlying io_uring ring.
func (w *IoUringWriter) Close() {
	w.ring.Close()
}
