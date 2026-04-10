//go:build linux
// +build linux

package iouring

// IoUringWriter wraps a BatchIoUringWriter with decoupled submit/complete
// goroutines. The ring mutex is held only during SQE prep + io_uring_enter,
// not during CQE drain, allowing concurrent flush batches from different
// shards to interleave submission.
type IoUringWriter struct {
	batch *BatchIoUringWriter
}

// NewIoUringWriter creates an IoUringWriter backed by a decoupled batch writer.
func NewIoUringWriter(entries uint32, flags uint32) (*IoUringWriter, error) {
	b, err := NewBatchIoUringWriter(BatchIoUringConfig{
		RingDepth:   entries,
		MaxBatch:    int(entries),
		MaxInflight: int(entries),
		QueueSize:   1024,
	})
	if err != nil {
		return nil, err
	}
	return &IoUringWriter{batch: b}, nil
}

// MaxBatchSize returns the maximum number of SQEs that can be submitted in
// a single SubmitWriteBatch call.
func (w *IoUringWriter) MaxBatchSize() int {
	return w.batch.MaxBatchSize()
}

// SubmitWriteBatch submits N pwrite operations and waits for all completions.
// Thread-safe. The ring mutex is NOT held during CQE drain.
func (w *IoUringWriter) SubmitWriteBatch(fd int, bufs [][]byte, offsets []uint64) ([]int, error) {
	return w.batch.SubmitWriteBatch(fd, bufs, offsets)
}

// Close releases the underlying io_uring ring and stops background goroutines.
func (w *IoUringWriter) Close() {
	w.batch.Close()
}
