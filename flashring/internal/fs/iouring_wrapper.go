//go:build linux
// +build linux

package fs

import (
	"fmt"
)

// IOUringFile wraps an existing WrapAppendFile with an io_uring ring for async I/O.
// It does NOT own the WrapAppendFile -- the caller manages its lifecycle.
type IOUringFile struct {
	*WrapAppendFile          // embed existing file (shared, not owned)
	ring            *IoUring // our raw io_uring instance
	depth           uint32   // submission queue depth
}

// NewIOUringFile attaches an io_uring ring to an existing WrapAppendFile.
// The WrapAppendFile is shared (not duplicated) -- writes and reads use
// the same file descriptors, so offset tracking stays in sync.
// ringDepth controls the SQ/CQ size (64-256 is a good starting point).
// flags can be 0 for normal mode.
func NewIOUringFile(waf *WrapAppendFile, ringDepth uint32, flags uint32) (*IOUringFile, error) {
	ring, err := NewIoUring(ringDepth, flags)
	if err != nil {
		return nil, fmt.Errorf("io_uring init failed: %w", err)
	}

	return &IOUringFile{
		WrapAppendFile: waf,
		ring:           ring,
		depth:          ringDepth,
	}, nil
}

// Close releases only the io_uring ring. The underlying WrapAppendFile
// is NOT closed here since it is shared with the shard.
func (f *IOUringFile) Close() {
	f.ring.Close()
}
