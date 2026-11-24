//go:build linux
// +build linux

package iouring

import (
	"testing"
)

func TestNewIoUringWriter_PassthroughToBatch(t *testing.T) {
	w, err := NewIoUringWriter(8, 0)
	if err != nil {
		t.Fatalf("NewIoUringWriter: %v", err)
	}
	defer w.Close()

	if w.MaxBatchSize() != 8 {
		t.Fatalf("MaxBatchSize = %d, want 8", w.MaxBatchSize())
	}
}

func TestIoUringWriter_SubmitWriteBatch_WritesAllChunks(t *testing.T) {
	fd := newTempWritableFile(t, 4096*8)

	w, err := NewIoUringWriter(8, 0)
	if err != nil {
		t.Fatalf("NewIoUringWriter: %v", err)
	}
	defer w.Close()

	bufs := make([][]byte, 4)
	offs := make([]uint64, 4)
	for i := range bufs {
		b := alignedBlock(4096, 4096)
		for j := range b {
			b[j] = byte(i + 1)
		}
		bufs[i] = b
		offs[i] = uint64(i * 4096)
	}

	results, err := w.SubmitWriteBatch(fd, bufs, offs)
	if err != nil {
		t.Fatalf("SubmitWriteBatch: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("results len = %d, want 4", len(results))
	}
	for i, n := range results {
		if n != 4096 {
			t.Fatalf("results[%d] = %d, want 4096", i, n)
		}
	}
}
