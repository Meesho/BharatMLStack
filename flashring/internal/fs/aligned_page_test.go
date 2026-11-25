//go:build linux
// +build linux

package fs

import "testing"

func TestNewAlignedPage_PageSize_BufLengthMatches(t *testing.T) {
	p, err := NewAlignedPage(BLOCK_SIZE)
	if err != nil {
		t.Fatalf("NewAlignedPage: %v", err)
	}
	if len(p.Buf) != BLOCK_SIZE {
		t.Fatalf("len(Buf) = %d, want %d", len(p.Buf), BLOCK_SIZE)
	}
	if &p.Buf[0] != &p.mmap[0] {
		t.Fatal("Buf and mmap do not share backing memory")
	}

	// Anonymous mmap pages must be zeroed.
	for i := 0; i < BLOCK_SIZE; i++ {
		if p.Buf[i] != 0 {
			t.Fatalf("Buf[%d] = %d, want 0 (anonymous mmap)", i, p.Buf[i])
		}
	}
	if err := Unmap(p); err != nil {
		t.Fatalf("Unmap: %v", err)
	}
}

func TestNewAlignedPage_InvalidSize_ReturnsError(t *testing.T) {
	p, err := NewAlignedPage(-1)
	if err == nil {
		t.Fatal("NewAlignedPage(-1) returned no error")
	}
	if p != nil {
		t.Fatal("NewAlignedPage(-1) returned a non-nil page")
	}
}

func TestUnmap_NilMmap_Idempotent(t *testing.T) {
	p, err := NewAlignedPage(BLOCK_SIZE)
	if err != nil {
		t.Fatalf("NewAlignedPage: %v", err)
	}
	if err := Unmap(p); err != nil {
		t.Fatalf("first Unmap: %v", err)
	}
	// Buf and mmap both nil now → second Unmap is a no-op returning nil.
	if err := Unmap(p); err != nil {
		t.Fatalf("second Unmap (idempotent) returned err: %v", err)
	}
	if p.Buf != nil {
		t.Fatal("Buf not cleared after Unmap")
	}
}
