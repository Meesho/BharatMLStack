//go:build linux
// +build linux

package fs

import (
	"runtime/pprof"

	"golang.org/x/sys/unix"
)

const (
	PROT_READ   = unix.PROT_READ
	PROT_WRITE  = unix.PROT_WRITE
	MAP_PRIVATE = unix.MAP_PRIVATE
	MAP_ANON    = unix.MAP_ANON
)

var mmapProf = pprof.NewProfile("mmap") // will show up in /debug/pprof/

type AlignedPage struct {
	Buf  []byte
	mmap []byte
}

func NewAlignedPage(pageSize int) *AlignedPage {
	b, err := unix.Mmap(-1, 0, pageSize, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON)
	if err != nil {
		panic(err)
	}
	if pageSize > 0 {
		mmapProf.Add(&b[0], pageSize) // attribute sz bytes to this callsite
	}
	return &AlignedPage{
		Buf:  b,
		mmap: b,
	}
}

func Unmap(p *AlignedPage) error {
	if len(p.mmap) > 0 {
		mmapProf.Remove(&p.mmap[0]) // release from custom profile
	}
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
