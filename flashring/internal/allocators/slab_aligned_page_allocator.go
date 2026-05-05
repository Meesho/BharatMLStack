package allocators

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/internal/pools"
	"github.com/rs/zerolog/log"
)

var (
	ErrSizeNotAligned = errors.New("size not aligned")
)

type SlabAlignedPageAllocatorConfig struct {
	SizeClasses []SizeClass
}

type SlabAlignedPageAllocator struct {
	config SlabAlignedPageAllocatorConfig
	pools  []*pools.LeakyPool[*fs.AlignedPage]
	sizes  []int
}

func NewSlabAlignedPageAllocator(config SlabAlignedPageAllocatorConfig) (*SlabAlignedPageAllocator, error) {
	sort.Slice(config.SizeClasses, func(i, j int) bool {
		return config.SizeClasses[i].Size < config.SizeClasses[j].Size
	})

	poolList := make([]*pools.LeakyPool[*fs.AlignedPage], len(config.SizeClasses))
	sizes := make([]int, len(config.SizeClasses))

	for i, sc := range config.SizeClasses {
		if sc.Size%fs.BLOCK_SIZE != 0 {
			return nil, ErrSizeNotAligned
		}
		sizes[i] = sc.Size
		size := sc.Size
		poolList[i] = pools.NewLeakyPool(pools.LeakyPoolConfig[*fs.AlignedPage]{
			Capacity:   sc.MinCount,
			Meta:       Meta{Size: sc.Size, Name: fmt.Sprintf("SlabAlignedPagePool-%dBytes", sc.Size)},
			CreateFunc: func() *fs.AlignedPage { return fs.NewAlignedPage(size) },
		})
		poolList[i].RegisterPreDrefHook(func(p *fs.AlignedPage) {
			fs.Unmap(p)
		})
		log.Debug().Msgf("SlabAlignedPageAllocator: size class - %d | min count - %d", sc.Size, sc.MinCount)
	}
	return &SlabAlignedPageAllocator{config: config, pools: poolList, sizes: sizes}, nil
}

func (a *SlabAlignedPageAllocator) Get(size int) *fs.AlignedPage {
	for i, s := range a.sizes {
		if size <= s {
			return a.pools[i].Get()
		}
	}
	return nil
}

func (a *SlabAlignedPageAllocator) Put(p *fs.AlignedPage) {
	for i, s := range a.sizes {
		if len(p.Buf) <= s {
			a.pools[i].Put(p)
			return
		}
	}
	log.Error().Msgf("SlabAlignedPageAllocator: Size class not found for size %d", len(p.Buf))
}
