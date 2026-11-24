//go:build linux
// +build linux

package iouring

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

func TestIoUringBasicRead(t *testing.T) {
	// 1. Create a temp file with known data
	f, err := os.CreateTemp("", "iouring_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 251) // non-zero pattern
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// 2. Open with O_DIRECT | O_RDONLY
	fd, err := syscall.Open(f.Name(), syscall.O_RDONLY|syscall.O_DIRECT, 0)
	if err != nil {
		t.Fatalf("open O_DIRECT: %v", err)
	}
	defer syscall.Close(fd)

	// 3. Create io_uring ring
	ring, err := NewIoUring(32, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	// 4. Allocate aligned buffer
	buf := alignedBlock(4096, 4096)

	// 5. Submit read via io_uring
	n, err := ring.SubmitRead(fd, buf, 0)
	if err != nil {
		t.Fatalf("SubmitRead: %v", err)
	}
	if n != 4096 {
		t.Fatalf("SubmitRead returned %d bytes, expected 4096", n)
	}

	// 6. Verify data
	for i := 0; i < 4096; i++ {
		if buf[i] != data[i] {
			t.Fatalf("data mismatch at byte %d: got %d, want %d", i, buf[i], data[i])
		}
	}
	t.Logf("io_uring read of 4096 bytes succeeded and data matches")

	// 7. Test a second read (to verify ring reuse works)
	buf2 := alignedBlock(4096, 4096)
	n2, err := ring.SubmitRead(fd, buf2, 0)
	if err != nil {
		t.Fatalf("SubmitRead #2: %v", err)
	}
	if n2 != 4096 {
		t.Fatalf("SubmitRead #2 returned %d bytes, expected 4096", n2)
	}
	for i := 0; i < 4096; i++ {
		if buf2[i] != data[i] {
			t.Fatalf("data mismatch #2 at byte %d: got %d, want %d", i, buf2[i], data[i])
		}
	}
	t.Logf("io_uring second read also succeeded")

	// 8. Test multiple sequential reads to exercise ring cycling
	for iter := 0; iter < 100; iter++ {
		buf3 := alignedBlock(4096, 4096)
		n3, err := ring.SubmitRead(fd, buf3, 0)
		if err != nil {
			t.Fatalf("SubmitRead iter %d: %v", iter, err)
		}
		if n3 != 4096 {
			t.Fatalf("SubmitRead iter %d returned %d bytes, expected 4096", iter, n3)
		}
	}
	t.Logf("100 sequential io_uring reads succeeded")
}

// alignedBlock returns a block-aligned buffer.
func alignedBlock(size, alignment int) []byte {
	raw := make([]byte, size+alignment)
	addr := uintptr(unsafe.Pointer(&raw[0]))
	off := (alignment - int(addr%uintptr(alignment))) % alignment
	return raw[off : off+size]
}

// --- Shared fixtures used across iouring_*_test.go files ----------------

// newTempReadOnlyFile creates a file pre-populated with a deterministic
// byte pattern and returns an O_DIRECT|O_RDONLY fd plus the original data.
// The fd is closed and the file removed via t.Cleanup.
func newTempReadOnlyFile(t *testing.T, size int) (int, []byte) {
	t.Helper()
	name := filepath.Join(t.TempDir(), "iouring_ro.dat")
	f, err := os.Create(name)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatalf("write: %v", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	_ = f.Close()

	fd, err := syscall.Open(name, syscall.O_RDONLY|syscall.O_DIRECT, 0)
	if err != nil {
		t.Fatalf("open O_DIRECT: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Close(fd)
		_ = os.Remove(name)
	})
	return fd, data
}

// newTempWritableFile opens a writable O_DIRECT file, pre-sized via
// ftruncate. Closed via t.Cleanup.
func newTempWritableFile(t *testing.T, size int64) int {
	t.Helper()
	name := filepath.Join(t.TempDir(), "iouring_rw.dat")
	fd, err := syscall.Open(name, syscall.O_RDWR|syscall.O_CREAT|syscall.O_DIRECT, 0644)
	if err != nil {
		t.Fatalf("open writable: %v", err)
	}
	if err := syscall.Ftruncate(fd, size); err != nil {
		_ = syscall.Close(fd)
		t.Fatalf("ftruncate: %v", err)
	}
	t.Cleanup(func() { _ = syscall.Close(fd) })
	return fd
}

// directIOAlignmentEnforced reports whether the filesystem backing the test's
// temp dir actually rejects misaligned O_DIRECT I/O. Some CI container
// filesystems (overlayfs / tmpfs) accept the O_DIRECT open flag but silently
// ignore the alignment requirement, which makes a "the kernel rejects a
// misaligned buffer" assertion impossible to exercise. Negative-path tests
// skip themselves when this returns false, matching the O_DIRECT-fallback
// skip pattern already used in internal/fs.
func directIOAlignmentEnforced(t *testing.T) bool {
	t.Helper()
	name := filepath.Join(t.TempDir(), "directio_probe.dat")
	fd, err := syscall.Open(name, syscall.O_RDWR|syscall.O_CREAT|syscall.O_DIRECT, 0644)
	if err != nil {
		// O_DIRECT unsupported entirely → alignment cannot be enforced.
		return false
	}
	defer func() { _ = syscall.Close(fd) }()
	if err := syscall.Ftruncate(fd, 8192); err != nil {
		return false
	}
	backing := alignedBlock(8192, 4096)
	misaligned := backing[1 : 1+4096]
	// On a filesystem honoring O_DIRECT this fails with EINVAL; if it succeeds
	// the fs is ignoring alignment, so the rejection path can't be exercised.
	_, werr := syscall.Pwrite(fd, misaligned, 0)
	return werr != nil
}

// captureStdout redirects fmt.Print* output to a pipe so tests can assert
// on diagnostic printouts emitted by SubmitRead's short-read branch. The
// returned function flushes the pipe and returns captured bytes; os.Stdout
// is always restored via t.Cleanup, even if the test fatals before calling
// it.
func captureStdout(t *testing.T) func() string {
	t.Helper()
	prev := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = prev
		_ = r.Close()
		_ = w.Close()
	})

	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	return func() string {
		_ = w.Close()
		out := <-done
		_ = r.Close()
		os.Stdout = prev
		return string(out)
	}
}

// --- IoUring (iouring.go) tests -----------------------------------------

func TestNewIoUring_InvalidEntries_Errors(t *testing.T) {
	// 0 entries → io_uring_setup returns EINVAL.
	_, err := NewIoUring(0, 0)
	if err == nil {
		t.Fatal("NewIoUring(0,0) returned nil err")
	}
}

func TestSubmitRead_EmptyBuffer_ReturnsZero(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	n, err := ring.SubmitRead(0, []byte{}, 0)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("n = %d, want 0", n)
	}
}

func TestSubmitRead_BadFd_ReturnsError(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	buf := alignedBlock(4096, 4096)
	_, err = ring.SubmitRead(-1, buf, 0)
	if err == nil {
		t.Fatal("SubmitRead with bad fd returned nil err")
	}
}

func TestSubmitRead_ShortRead_PrintsDiagnostic(t *testing.T) {
	// 4 KiB file; read 8 KiB at offset 0 → io_uring returns 4 KiB; the
	// `int(res) != len(buf)` diagnostic branch fires and prints.
	fd, _ := newTempReadOnlyFile(t, 4096)

	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	restore := captureStdout(t)
	buf := alignedBlock(8192, 4096)
	n, err := ring.SubmitRead(fd, buf, 0)
	out := restore()

	if err != nil {
		t.Fatalf("SubmitRead: %v", err)
	}
	if n != 4096 {
		t.Fatalf("n = %d, want 4096 (short read at EOF)", n)
	}
	if !bytes.Contains([]byte(out), []byte("[io_uring diag]")) {
		t.Fatalf("diagnostic not printed; stdout=%q", out)
	}
}

func TestSubmitRead_PastEOF_PrintsDiagnostic(t *testing.T) {
	// Reading at offset == file size should return 0 from io_uring; this
	// also exercises the short-read diagnostic.
	fd, _ := newTempReadOnlyFile(t, 4096)

	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	restore := captureStdout(t)
	buf := alignedBlock(4096, 4096)
	n, err := ring.SubmitRead(fd, buf, 4096)
	out := restore()

	if err != nil {
		t.Fatalf("SubmitRead: %v", err)
	}
	if n != 0 {
		t.Fatalf("n = %d, want 0 (read at EOF)", n)
	}
	if !bytes.Contains([]byte(out), []byte("[io_uring diag]")) {
		t.Fatalf("diagnostic not printed; stdout=%q", out)
	}
}

func TestSubmitWrite_EmptyBuffer_ReturnsZero(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	n, err := ring.SubmitWrite(0, []byte{}, 0)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("n = %d, want 0", n)
	}
}

func TestSubmitWrite_HappyPath_WritesAndPersists(t *testing.T) {
	fd := newTempWritableFile(t, 4096)

	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	buf := alignedBlock(4096, 4096)
	for i := range buf {
		buf[i] = byte(i % 251)
	}
	n, err := ring.SubmitWrite(fd, buf, 0)
	if err != nil {
		t.Fatalf("SubmitWrite: %v", err)
	}
	if n != 4096 {
		t.Fatalf("n = %d, want 4096", n)
	}

	// Verify via syscall.Pread that data is on disk.
	read := alignedBlock(4096, 4096)
	rn, err := syscall.Pread(fd, read, 0)
	if err != nil {
		t.Fatalf("Pread: %v", err)
	}
	if rn != 4096 {
		t.Fatalf("Pread returned %d, want 4096", rn)
	}
	if !bytes.Equal(read, buf) {
		t.Fatal("disk content differs from written content")
	}
}

func TestSubmitWrite_BadFd_ReturnsError(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	buf := alignedBlock(4096, 4096)
	_, err = ring.SubmitWrite(-1, buf, 0)
	if err == nil {
		t.Fatal("SubmitWrite with bad fd returned nil err")
	}
}

func TestSubmitWrite_NonAlignedBuf_KernelReturnsError(t *testing.T) {
	// O_DIRECT requires aligned buffers; passing a misaligned one makes the
	// kernel return EINVAL through the CQE → `res < 0` branch in SubmitWrite.
	fd := newTempWritableFile(t, 4096)
	if !directIOAlignmentEnforced(t) {
		t.Skip("filesystem does not enforce O_DIRECT alignment (CI fallback); misaligned-buffer rejection path is untestable here")
	}

	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	// Misalign: slice a 1-byte offset into an aligned backing array.
	backing := alignedBlock(8192, 4096)
	buf := backing[1 : 1+4096]

	_, err = ring.SubmitWrite(fd, buf, 0)
	if err == nil {
		t.Fatal("SubmitWrite with misaligned buf returned nil err")
	}
}

func TestSubmitWriteBatch_EmptyInputs(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	res, err := ring.SubmitWriteBatch(0, nil, nil)
	if err != nil || res != nil {
		t.Fatalf("SubmitWriteBatch empty = (%v, %v), want (nil, nil)", res, err)
	}
}

func TestSubmitWriteBatch_HappyPath_DirectRing(t *testing.T) {
	fd := newTempWritableFile(t, 4096*4)

	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	bufs := [][]byte{
		alignedBlock(4096, 4096),
		alignedBlock(4096, 4096),
	}
	for _, b := range bufs {
		for i := range b {
			b[i] = 0x55
		}
	}
	offs := []uint64{0, 4096}
	results, err := ring.SubmitWriteBatch(fd, bufs, offs)
	if err != nil {
		t.Fatalf("SubmitWriteBatch: %v", err)
	}
	for i, n := range results {
		if n != 4096 {
			t.Fatalf("results[%d] = %d, want 4096", i, n)
		}
	}
}

func TestSubmitWriteBatch_MultipleSlots_AllSucceed(t *testing.T) {
	fd := newTempWritableFile(t, 4096*8)

	ring, err := NewIoUring(16, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	const n = 6
	bufs := make([][]byte, n)
	offs := make([]uint64, n)
	for i := 0; i < n; i++ {
		b := alignedBlock(4096, 4096)
		for j := range b {
			b[j] = byte(i + 1)
		}
		bufs[i] = b
		offs[i] = uint64(i * 4096)
	}

	results, err := ring.SubmitWriteBatch(fd, bufs, offs)
	if err != nil {
		t.Fatalf("SubmitWriteBatch: %v", err)
	}
	if len(results) != n {
		t.Fatalf("len(results) = %d, want %d", len(results), n)
	}
	for i, got := range results {
		if got != 4096 {
			t.Fatalf("results[%d] = %d, want 4096", i, got)
		}
	}
}

// TestSubmitWriteBatch_SQFull triggers the `sqe == nil` branch in
// SubmitWriteBatch by asking for more slots than the ring exposes.
func TestSubmitWriteBatch_SQFull_ReturnsError(t *testing.T) {
	fd := newTempWritableFile(t, 4096*64)

	ring, err := NewIoUring(2, 0) // tiny ring
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	const n = 32
	bufs := make([][]byte, n)
	offs := make([]uint64, n)
	for i := 0; i < n; i++ {
		bufs[i] = alignedBlock(4096, 4096)
		offs[i] = uint64(i * 4096)
	}

	_, err = ring.SubmitWriteBatch(fd, bufs, offs)
	if err == nil {
		t.Fatal("SubmitWriteBatch returned nil err for over-capacity batch")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("SQ full")) {
		t.Fatalf("err = %v, want 'SQ full' substring", err)
	}
}

func TestSubmitWriteBatch_BadFd_ReturnsError(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	bufs := [][]byte{alignedBlock(4096, 4096)}
	offs := []uint64{0}
	_, err = ring.SubmitWriteBatch(-1, bufs, offs)
	if err == nil {
		t.Fatal("SubmitWriteBatch with bad fd returned nil err")
	}
}

func TestSubmitWriteBatch_NonAlignedBuf_KernelReturnsError(t *testing.T) {
	fd := newTempWritableFile(t, 8192)
	if !directIOAlignmentEnforced(t) {
		t.Skip("filesystem does not enforce O_DIRECT alignment (CI fallback); misaligned-buffer rejection path is untestable here")
	}

	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	backing := alignedBlock(8192, 4096)
	misaligned := backing[1 : 1+4096]

	_, err = ring.SubmitWriteBatch(fd, [][]byte{misaligned}, []uint64{0})
	if err == nil {
		t.Fatal("SubmitWriteBatch with misaligned buf returned nil err")
	}
}

func TestPrepRead_PrepWrite_EmptyBuffer_OpcodeNop(t *testing.T) {
	ring, err := NewIoUring(8, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	sqe := ring.getSqe()
	if sqe == nil {
		t.Fatal("getSqe returned nil")
	}
	prepRead(sqe, 0, nil, 0)
	if sqe.Opcode != iouringOpNop {
		t.Fatalf("prepRead empty: opcode = %d, want NOP", sqe.Opcode)
	}

	sqe = ring.getSqe()
	prepWrite(sqe, 0, nil, 0)
	if sqe.Opcode != iouringOpNop {
		t.Fatalf("prepWrite empty: opcode = %d, want NOP", sqe.Opcode)
	}
}

func TestGetSqe_FullRing_ReturnsNil(t *testing.T) {
	ring, err := NewIoUring(4, 0)
	if err != nil {
		t.Fatalf("NewIoUring: %v", err)
	}
	defer ring.Close()

	// Consume all available SQEs (ring depth=4 → sqEntries might be rounded
	// up to next pow2 = 4). Loop until getSqe returns nil.
	for i := 0; i < 100; i++ {
		if ring.getSqe() == nil {
			return // exercised the nil-return branch
		}
	}
	t.Fatal("getSqe never returned nil after 100 calls")
}

// TestIoUring_Close_Idempotent locks down the fix in PR #2377: Close is now
// idempotent on the fd (checks fd>=0 and marks -1 after closing) to support
// BatchIoUringReader/Writer's early-fd-close shutdown fallback. Double-close
// would otherwise error/panic depending on kernel state.
func TestIoUring_Close_Idempotent(t *testing.T) {
	r, err := NewIoUring(8, 0)
	if err != nil {
		t.Skipf("io_uring init unavailable: %v", err)
	}
	r.Close()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("second Close panicked: %v", rec)
		}
	}()
	r.Close()
	if r.fd != -1 {
		t.Fatalf("after Close, fd = %d, want -1", r.fd)
	}
}
