package allocator

import (
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/pool"
	"github.com/rs/zerolog/log"
)

type SlabAlignedPageAllocatorConfig struct {
	PageSizeAlignement int
	Multipliers        []int
	MaxPages           []int
}

type SlabAlignedPageAllocator struct {
	config           SlabAlignedPageAllocatorConfig
	pools            []*pool.LeakyPool
	sizeClassToIndex map[int]int
}

func NewSlabAlignedPageAllocator(config SlabAlignedPageAllocatorConfig) *SlabAlignedPageAllocator {
	pools := make([]*pool.LeakyPool, len(config.Multipliers))
	sizeClassToIndex := make(map[int]int)
	for i, multiplier := range config.Multipliers {
		size := config.PageSizeAlignement * multiplier
		pools[i] = pool.NewLeakyPoolV2(config.MaxPages[i], size, func() interface{} {
			return NewAlignedPage(size)
		})
		pools[i].RegisterPreDrefHook(func(obj interface{}) {
			Unmap(obj.(*Page))
		})
		log.Debug().Msgf("SlabAlignedPageAllocator: Size class %d: %d", size, i)
		sizeClassToIndex[size] = i
	}
	return &SlabAlignedPageAllocator{config: config, pools: pools, sizeClassToIndex: sizeClassToIndex}
}

func (a *SlabAlignedPageAllocator) Get(size int) (*Page, bool) {
	for _, pool := range a.pools {
		if size <= pool.SizeClass {
			page, crossBound := pool.Get()
			if crossBound {
				log.Warn().Msgf("SlabAlignedPageAllocator: Crossed bound for size %d", size)
			}
			return page.(*Page), crossBound
		}
	}
	return nil, false
}

func (a *SlabAlignedPageAllocator) Put(p *Page) {
	for _, pool := range a.pools {
		if len(p.Buf) <= pool.SizeClass {
			pool.Put(p)
			return
		}
	}
	log.Error().Msgf("SlabAlignedPageAllocator: Size class not found for size %d", len(p.Buf))
}
