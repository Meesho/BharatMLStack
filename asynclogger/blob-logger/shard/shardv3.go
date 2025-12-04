package shard

import (
	"sync/atomic"
	"unsafe"
	_ "unsafe" // for go:linkname
)

const cacheLineSize = 64

// Align Shard header so `offset` sits alone in its cache line.
// This reduces false sharing between producers.
type ShardV3 struct {
	_              [cacheLineSize]byte // padding before hot fields
	basePtr        unsafe.Pointer      // &data[0]
	baseUintptr    uintptr             // uintptr(basePtr)
	capacityBytes  uintptr
	offset         atomic.Uintptr      // atomic pointer-sized offset
	_              [cacheLineSize]byte // padding after hot fields
	alignedStorage []byte              // keeps backing array alive
	id             uint32
}

// NewShard allocates a 64-byte aligned region and stores precomputed pointers.
func NewShardV3(capacity int, id uint32) *ShardV3 {
	if capacity <= 0 {
		panic("invalid shard capacity")
	}

	raw := make([]byte, capacity+cacheLineSize)
	rawBase := uintptr(unsafe.Pointer(&raw[0]))

	// Align to next 64 bytes
	aligned := (rawBase + cacheLineSize - 1) &^ (cacheLineSize - 1)

	data := unsafe.Slice((*byte)(unsafe.Pointer(aligned)), capacity)

	s := &ShardV3{
		basePtr:        unsafe.Pointer(&data[0]),
		baseUintptr:    uintptr(unsafe.Pointer(&data[0])),
		capacityBytes:  uintptr(capacity),
		alignedStorage: raw,
		id:             id,
	}
	return s
}

//go:nosplit
func (s *ShardV3) Write(p []byte) (uintptr, bool) {
	size := uintptr(len(p))
	if size == 0 {
		return 0, false
	}

	// Atomically reserve space — relaxed atomic semantics.
	newOffset := s.offset.Add(size)

	// Bounds check - minimal branching
	if newOffset > s.capacityBytes {
		return 0, true
	}

	start := newOffset - size

	// Calculate destination quickly using uintptr math
	dst := unsafe.Pointer(s.baseUintptr + start)

	// Source pointer
	src := unsafe.Pointer(&p[0])

	// Hardware-accelerated memmove
	memmove(dst, src, size)

	return size, false
}

// Remaining returns how many bytes are left. Not exact under concurrency.
func (s *ShardV3) Remaining() uintptr {
	off := s.offset.Load()
	if off >= s.capacityBytes {
		return 0
	}
	return s.capacityBytes - off
}

func (s *ShardV3) Reset() {
	s.offset.Store(0)
}
