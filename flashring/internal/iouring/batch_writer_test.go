//go:build linux
// +build linux

package iouring

import (
	"sync"
	"testing"
)

func TestNewBatchIoUringWriter_DefaultsApplied(t *testing.T) {
	w, err := NewBatchIoUringWriter(BatchIoUringConfig{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer w.Close()
	if w.maxBatch == 0 {
		t.Fatal("maxBatch = 0, defaults not applied")
	}
}

func TestNewBatchIoUringWriter_QueueSizeDefault(t *testing.T) {
	w, err := NewBatchIoUringWriter(BatchIoUringConfig{
		RingDepth:   8,
		MaxBatch:    4,
		MaxInflight: 4,
		QueueSize:   0, // exercise default branch
	})
	if err != nil {
		t.Fatalf("NewBatchIoUringWriter: %v", err)
	}
	defer w.Close()

	if cap(w.reqCh) == 0 {
		t.Fatal("reqCh capacity = 0, QueueSize default not applied")
	}
}

func TestNewBatchIoUringWriter_MaxBatchClampedToInflight(t *testing.T) {
	w, err := NewBatchIoUringWriter(BatchIoUringConfig{
		RingDepth:   8,
		MaxBatch:    1024, // > MaxInflight → must be clamped
		MaxInflight: 4,
	})
	if err != nil {
		t.Fatalf("NewBatchIoUringWriter: %v", err)
	}
	defer w.Close()

	if w.maxBatch > 4 {
		t.Fatalf("maxBatch = %d, want <= MaxInflight=4 (clamped)", w.maxBatch)
	}
}

func TestBatchIoUringWriter_EmptyBatch_NoOp(t *testing.T) {
	w, err := NewBatchIoUringWriter(BatchIoUringConfig{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer w.Close()

	res, err := w.SubmitWriteBatch(0, nil, nil)
	if err != nil || res != nil {
		t.Fatalf("SubmitWriteBatch empty = (%v, %v), want (nil, nil)", res, err)
	}
}

func TestBatchIoUringWriter_BadFd_ReturnsErrFromKernel(t *testing.T) {
	w, err := NewBatchIoUringWriter(BatchIoUringConfig{RingDepth: 8})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer w.Close()

	bufs := [][]byte{alignedBlock(4096, 4096)}
	offs := []uint64{0}
	// fd = -1 is invalid → kernel returns EBADF → completeLoop surfaces it.
	_, err = w.SubmitWriteBatch(-1, bufs, offs)
	if err == nil {
		t.Fatal("SubmitWriteBatch with bad fd returned nil err")
	}
}

func TestBatchIoUringWriter_ConcurrentWrites_AllComplete(t *testing.T) {
	fd := newTempWritableFile(t, 4096*32)

	w, err := NewBatchIoUringWriter(BatchIoUringConfig{
		RingDepth:   16,
		MaxBatch:    8,
		MaxInflight: 8,
		QueueSize:   64,
	})
	if err != nil {
		t.Fatalf("NewBatchIoUringWriter: %v", err)
	}
	defer w.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b := alignedBlock(4096, 4096)
			for j := range b {
				b[j] = byte(idx + 1)
			}
			_, err := w.SubmitWriteBatch(fd, [][]byte{b}, []uint64{uint64(idx * 4096)})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("concurrent write error: %v", e)
	}
}
