//go:build linux
// +build linux

package fs

import (
	"os"
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
	buf := AlignedBlock(4096, 4096)

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
	buf2 := AlignedBlock(4096, 4096)
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
		buf3 := AlignedBlock(4096, 4096)
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

// AlignedBlock returns a 4096-byte-aligned buffer.
func AlignedBlock(size, alignment int) []byte {
	raw := make([]byte, size+alignment)
	addr := uintptr(unsafe.Pointer(&raw[0]))
	off := (alignment - int(addr%uintptr(alignment))) % alignment
	return raw[off : off+size]
}
