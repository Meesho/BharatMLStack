package allocator

import (
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/pool"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

type Page struct {
	Buf  []byte
	mmap []byte
}

func NewAlignedPage(pageSize int) *Page {
	b, err := unix.Mmap(-1, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		panic(err)
	}
	return &Page{
		Buf:  b,
		mmap: b,
	}
}

func Unmap(p *Page) error {
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

type AlignedPageAllocatorConfig struct {
	PageSizeAlignement int
	Multiplier         int
	MaxPages           int
}

type AlignedPageAllocator struct {
	config AlignedPageAllocatorConfig
	pool   *pool.LeakyPool
}

func NewAlignedPageAllocator(config AlignedPageAllocatorConfig) *AlignedPageAllocator {
	newFunc := func() interface{} {
		return NewAlignedPage(config.PageSizeAlignement * config.Multiplier)
	}
	pool := pool.NewLeakyPool(config.MaxPages, newFunc)
	pool.RegisterPreDrefHook(func(obj interface{}) {
		Unmap(obj.(*Page))
	})
	return &AlignedPageAllocator{config: config, pool: pool}
}

func (a *AlignedPageAllocator) Get() (*Page, bool) {
	page, crossBound := a.pool.Get()
	if crossBound {
		log.Warn().Msg("AlignedPageAllocator: Crossed bound")
	}
	return page.(*Page), true
}

func (a *AlignedPageAllocator) Put(p *Page) {
	a.pool.Put(p)
}
