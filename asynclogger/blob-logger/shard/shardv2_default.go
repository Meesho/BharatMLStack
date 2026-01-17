package shard

import (
	"sync/atomic"
	"unsafe"
	_ "unsafe" // required for go:linkname

	"golang.org/x/sys/unix"
)

// memmove is linked to the Go runtime's highly optimized memmove implementation.
// We mark it noescape so the compiler knows the pointers won't escape.
//
//go:noescape
//go:linkname memmove runtime.memmove
func memmove(dst, src unsafe.Pointer, n uintptr)

// Shard is a strictly append-only buffer with atomic offset.
// Writes are linear appends; when capacity is exceeded, Write reports "full".
type ShardV2 struct {
	data          []byte // 64-byte aligned backing buffer
	offset        int32  // current write offset (in bytes), updated atomically
	capacity      int32  // total capacity in bytes
	id            uint32
	inflight      atomic.Int64
	readyForFlush atomic.Uint32
}

// NewShard allocates a shard with a 64-byte–aligned data slice of the given capacity.
func NewShardV2(capacity int, id uint32) *ShardV2 {
	if capacity <= 0 {
		panic("NewShard: capacity must be > 0")
	}
	// Round capacity up to page size (4096)
	const page = 4096
	padded := (capacity + page - 1) &^ (page - 1)

	// Create an anonymous private mapping
	data, err := unix.Mmap(
		-1, 0,
		padded,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_PRIVATE|unix.MAP_ANONYMOUS,
	)
	if err != nil {
		panic(err)
	}

	// // Allocate a bit extra so that after alignment we still have `capacity` bytes.
	// raw := make([]byte, capacity+64)

	// base := uintptr(unsafe.Pointer(&raw[0]))
	// aligned := (base + 63) &^ 63 // round up to next multiple of 64

	// data := unsafe.Slice((*byte)(unsafe.Pointer(aligned)), capacity)

	return &ShardV2{
		data:          data,
		offset:        0,
		capacity:      int32(capacity),
		id:            id,
		inflight:      atomic.Int64{},
		readyForFlush: atomic.Uint32{},
	}
}

// Write appends p to the shard.
// Returns (bytesWritten, full):
//   - bytesWritten == len(p) and full == false on success
//   - bytesWritten == 0 and full == true if there is no space left
func (s *ShardV2) Write(p []byte) (int32, uint32, bool, bool) {
	size := int32(len(p))
	if size == 0 {
		return 0, s.id, false, false
	}

	// Atomically reserve space.
	newOffset := atomic.AddInt32(&s.offset, size)

	// If we exceeded capacity, signal "full" to the caller.
	if newOffset > s.capacity {

		if s.inflight.Load() == 0 {
			return 0, s.id, true, true
		}

		return 0, s.id, true, false
	}

	s.inflight.Add(1)

	start := newOffset - size

	// Bounds are guaranteed by the capacity check above.
	dst := unsafe.Add(unsafe.Pointer(&s.data[0]), uintptr(start))
	// Go 1.20+: unsafe.SliceData(p); fall back to &p[0] for broad compatibility.
	src := unsafe.Pointer(&p[0])

	memmove(dst, src, uintptr(size))

	s.inflight.Add(-1)

	if s.inflight.Load() == 0 {
		return size, s.id, false, true
	}

	return size, s.id, false, false
}

// Remaining returns how many bytes of capacity are left (approximate under concurrency).
func (s *ShardV2) Remaining() int32 {
	off := atomic.LoadInt32(&s.offset)
	rem := s.capacity - off
	if rem < 0 {
		return 0
	}
	return rem
}

// Reset zeroes the offset so the shard can be reused. Caller must ensure
// no concurrent writers when calling Reset.
func (s *ShardV2) Reset() {
	atomic.StoreInt32(&s.offset, 0)
	s.readyForFlush.Store(0)
	s.inflight.Store(0)
}
