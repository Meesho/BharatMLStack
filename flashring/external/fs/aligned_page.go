//go:build linux
// +build linux

package fs

import "golang.org/x/sys/unix"

const (
	PROT_READ   = unix.PROT_READ
	PROT_WRITE  = unix.PROT_WRITE
	MAP_PRIVATE = unix.MAP_PRIVATE
	MAP_ANON    = unix.MAP_ANON
)

type AlignedPage struct {
	Buf  []byte
	mmap []byte
}

func NewAlignedPage(pageSize int) *AlignedPage {
	b, err := unix.Mmap(-1, 0, pageSize, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON)
	if err != nil {
		panic(err)
	}
	return &AlignedPage{
		Buf:  b,
		mmap: b,
	}
}

func Unmap(p *AlignedPage) error {
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
