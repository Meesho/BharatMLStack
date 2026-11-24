//go:build linux
// +build linux

package fs

import (
	"golang.org/x/sys/unix"
)

const (
	PROT_READ   = unix.PROT_READ
	PROT_WRITE  = unix.PROT_WRITE
	MAP_PRIVATE = unix.MAP_PRIVATE
	MAP_ANON    = unix.MAP_ANON
)

// var mmapProf = pprof.NewProfile("mmap") // will show up in /debug/pprof/

// AlignedPage is a page-aligned read buffer backed by an anonymous mmap
// region. The mmap origin guarantees the buffer's start address is aligned
// to the system page size — required for O_DIRECT reads, which reject
// unaligned user buffers with EINVAL. Must be released with Unmap;
// otherwise the mapping persists for the process lifetime.
type AlignedPage struct {
	Buf  []byte
	mmap []byte
}

// NewAlignedPage allocates a pageSize-byte anonymous mmap region. Panics
// on mmap failure — the cache treats page allocation as an unrecoverable
// resource error.
func NewAlignedPage(pageSize int) *AlignedPage {
	b, err := unix.Mmap(-1, 0, pageSize, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON)
	if err != nil {
		panic(err)
	}
	// if pageSize > 0 {
	// 	mmapProf.Add(&b[0], pageSize) // attribute sz bytes to this callsite
	// }
	return &AlignedPage{
		Buf:  b,
		mmap: b,
	}
}

// Unmap releases the underlying mmap region. Idempotent — calling Unmap
// twice on the same page is a no-op. After Unmap the page's Buf is nil
// and the page must not be reused.
func Unmap(p *AlignedPage) error {
	// if len(p.mmap) > 0 {
	// 	mmapProf.Remove(&p.mmap[0]) // release from custom profile
	// }
	if p.mmap != nil {
		err := unix.Munmap(p.mmap)
		if err != nil {
			return err
		}
		p.mmap = nil
	}
	p.Buf = nil
	p.mmap = nil
	return nil
}
