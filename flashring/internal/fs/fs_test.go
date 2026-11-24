//go:build linux
// +build linux

package fs

import (
	"path/filepath"
	"testing"
)

func TestIsAlignedBuffer_EmptyBuffer_ReturnsFalse(t *testing.T) {
	if isAlignedBuffer([]byte{}, 4096) {
		t.Fatal("isAlignedBuffer on empty buf = true, want false")
	}
}

func TestIsAlignedBuffer_AlignedBufferTrue(t *testing.T) {
	buf := createAlignedBuffer(4096, 4096)
	if !isAlignedBuffer(buf, 4096) {
		t.Fatal("isAlignedBuffer returned false for aligned buf")
	}
}

func TestIsAlignedBuffer_UnalignedSlice_ReturnsFalse(t *testing.T) {
	// Take a 1-byte offset into the aligned buffer to force misalignment.
	buf := createAlignedBuffer(4096, 4096)
	if isAlignedBuffer(buf[1:], 4096) {
		t.Fatal("isAlignedBuffer returned true for misaligned slice")
	}
}

func TestAlignRange_Examples(t *testing.T) {
	cases := []struct {
		offset, length, block            int64
		wantStart, wantSize              int64
	}{
		{0, 4096, 4096, 0, 4096},
		{100, 200, 4096, 0, 4096},      // both ends inside one block
		{4096, 4096, 4096, 4096, 4096}, // already aligned
		{4090, 100, 4096, 0, 8192},     // crosses one block boundary
		{0, 0, 4096, 0, 0},             // zero length
	}
	for _, tc := range cases {
		gotStart, gotSize := AlignRange(tc.offset, int(tc.length), tc.block)
		if gotStart != tc.wantStart || gotSize != tc.wantSize {
			t.Errorf("AlignRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
				tc.offset, tc.length, tc.block,
				gotStart, gotSize, tc.wantStart, tc.wantSize)
		}
	}
}

func TestOpenWithDirectIO_NonExistentDir_ReturnsError(t *testing.T) {
	_, _, err := openWithDirectIO("/nonexistent/dir/file.dat", O_WRONLY|O_CREAT)
	if err == nil {
		t.Fatal("openWithDirectIO on bad path returned nil err")
	}
}

func TestCreateAppendOnlyWriteFileDescriptor_BadPath_Errors(t *testing.T) {
	_, _, _, err := createAppendOnlyWriteFileDescriptor("/nonexistent/dir/file.dat")
	if err == nil {
		t.Fatal("createAppendOnlyWriteFileDescriptor returned nil err for bad path")
	}
}

func TestCreatePreAllocatedWriteFileDescriptor_BadPath_Errors(t *testing.T) {
	_, _, _, err := createPreAllocatedWriteFileDescriptor("/nonexistent/dir/file.dat", 4096)
	if err == nil {
		t.Fatal("createPreAllocatedWriteFileDescriptor returned nil err for bad path")
	}
}

func TestCreatePreAllocatedWriteFileDescriptor_FallocateNegativeSize_Errors(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "f.dat")
	_, _, _, err := createPreAllocatedWriteFileDescriptor(tmp, -1)
	if err == nil {
		t.Fatal("createPreAllocatedWriteFileDescriptor with -1 size returned nil err")
	}
}

func TestCreateReadFileDescriptor_BadPath_Errors(t *testing.T) {
	_, _, _, err := createReadFileDescriptor("/nonexistent/file.dat")
	if err == nil {
		t.Fatal("createReadFileDescriptor returned nil err for missing file")
	}
}

func TestCreateReadFileDescriptor_HappyPath(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "f.dat")
	_, file, _, err := createAppendOnlyWriteFileDescriptor(tmp)
	if err != nil {
		t.Fatalf("setup write fd: %v", err)
	}
	defer file.Close() // also closes fd

	rfd, rfile, _, err := createReadFileDescriptor(tmp)
	if err != nil {
		t.Fatalf("createReadFileDescriptor: %v", err)
	}
	defer rfile.Close() // also closes rfd

	if rfd <= 0 {
		t.Fatalf("read fd = %d, want > 0", rfd)
	}
	if rfile == nil {
		t.Fatal("read file is nil")
	}
}

func TestFdToFile_HappyPath_ReturnsOSFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "f.dat")
	fd, _, err := openWithDirectIO(tmp, O_WRONLY|O_CREAT)
	if err != nil {
		t.Fatalf("openWithDirectIO: %v", err)
	}
	f, err := fdToFile(fd, tmp)
	if err != nil {
		t.Fatalf("fdToFile: %v", err)
	}
	defer f.Close() // owns fd; a raw close would let its finalizer close a reused number
	if f == nil {
		t.Fatal("fdToFile returned nil file")
	}
}

func TestPreadv2_EmptyBuffer_ReturnsZero(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*4)
	n, err := preadv2(waf.ReadFd, []byte{}, 0, 0)
	if err != nil {
		t.Fatalf("preadv2 empty buf err = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("preadv2 empty buf n = %d, want 0", n)
	}
}

func TestPreadv2_HappyPath_ReadsBack(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)

	src := createAlignedBuffer(BLOCK_SIZE, BLOCK_SIZE)
	for i := range src {
		src[i] = 0xAB
	}
	if _, err := waf.Pwrite(src); err != nil {
		t.Fatalf("Pwrite: %v", err)
	}

	dst := createAlignedBuffer(BLOCK_SIZE, BLOCK_SIZE)
	n, err := preadv2(waf.ReadFd, dst, 0, 0)
	if err != nil {
		t.Fatalf("preadv2: %v", err)
	}
	if n != BLOCK_SIZE {
		t.Fatalf("n = %d, want %d", n, BLOCK_SIZE)
	}
	if dst[0] != 0xAB {
		t.Fatalf("dst[0] = %x, want 0xAB", dst[0])
	}
}
