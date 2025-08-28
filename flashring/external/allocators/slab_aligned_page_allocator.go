package allocators

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Meesho/BharatMLStack/flashring/external/fs"
	"github.com/Meesho/BharatMLStack/flashring/external/pools"
	"github.com/rs/zerolog/log"
)

var (
	ErrSizeNotAligned = errors.New("size not aligned")
)

type SlabAlignedPageAllocatorConfig struct {
	SizeClasses []SizeClass
}

type Meta struct {
	Size int
	Name string
}

type SlabAlignedPageAllocator struct {
	config SlabAlignedPageAllocatorConfig
	pools  []*pools.LeakyPool
}

func NewSlabAlignedPageAllocator(config SlabAlignedPageAllocatorConfig) (*SlabAlignedPageAllocator, error) {
	poolList := make([]*pools.LeakyPool, len(config.SizeClasses))
	sort.Slice(config.SizeClasses, func(i, j int) bool {
		return config.SizeClasses[i].Size < config.SizeClasses[j].Size
	})
	for i, sizeClass := range config.SizeClasses {
		if sizeClass.Size%fs.BLOCK_SIZE != 0 {
			return nil, ErrSizeNotAligned
		}
		poolConfig := pools.LeakyPoolConfig{
			Capacity:   sizeClass.MinCount,
			Meta:       Meta{Size: sizeClass.Size, Name: fmt.Sprintf("SlabAlignedPagePool-%dBytes", sizeClass.Size)},
			CreateFunc: func() interface{} { return fs.NewAlignedPage(sizeClass.Size) },
		}
		poolList[i] = pools.NewLeakyPool(poolConfig)
		poolList[i].RegisterPreDrefHook(func(obj interface{}) {
			fs.Unmap(obj.(*fs.AlignedPage))
		})
		log.Debug().Msgf("SlabAlignedPageAllocator: size class - %d | min count - %d", sizeClass.Size, sizeClass.MinCount)
	}
	return &SlabAlignedPageAllocator{config: config, pools: poolList}, nil
}

func (a *SlabAlignedPageAllocator) Get(size int) *fs.AlignedPage {
	for _, pool := range a.pools {
		if size <= pool.Meta.(Meta).Size {
			page := pool.Get()
			return page.(*fs.AlignedPage)
		}
	}
	return nil
}

func (a *SlabAlignedPageAllocator) Put(p *fs.AlignedPage) {
	for _, pool := range a.pools {
		if len(p.Buf) <= pool.Meta.(Meta).Size {
			pool.Put(p)
			return
		}
	}
	log.Error().Msgf("SlabAlignedPageAllocator: Size class not found for size %d", len(p.Buf))
}
