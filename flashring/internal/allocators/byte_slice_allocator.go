package allocators

import (
	"fmt"
	"sort"

	"github.com/Meesho/BharatMLStack/flashring/internal/pools"
	"github.com/rs/zerolog/log"
)

type ByteSliceAllocatorConfig struct {
	SizeClasses []SizeClass
}

type ByteSliceAllocator struct {
	config ByteSliceAllocatorConfig
	pools  []*pools.LeakyPool
}

func NewByteSliceAllocator(config ByteSliceAllocatorConfig) *ByteSliceAllocator {
	poolList := make([]*pools.LeakyPool, len(config.SizeClasses))
	sort.Slice(config.SizeClasses, func(i, j int) bool {
		return config.SizeClasses[i].Size < config.SizeClasses[j].Size
	})
	for i, sizeClass := range config.SizeClasses {
		poolConfig := pools.LeakyPoolConfig{
			Capacity:   sizeClass.MinCount,
			Meta:       Meta{Size: sizeClass.Size, Name: fmt.Sprintf("ByteSlicePool-%dBytes", sizeClass.Size)},
			CreateFunc: func() interface{} { return make([]byte, sizeClass.Size) },
		}
		poolList[i] = pools.NewLeakyPool(poolConfig)
		log.Debug().Msgf("ByteSliceAllocator: size class - %d | min count - %d", sizeClass.Size, sizeClass.MinCount)
	}
	return &ByteSliceAllocator{config: config, pools: poolList}
}

func (a *ByteSliceAllocator) Get(size int) []byte {
	for _, pool := range a.pools {
		if size <= pool.Meta.(Meta).Size {
			slice := pool.Get()
			return slice.([]byte)
		}
	}
	return nil
}

func (a *ByteSliceAllocator) Put(p []byte) {
	for _, pool := range a.pools {
		if len(p) <= pool.Meta.(Meta).Size {
			pool.Put(p)
			return
		}
	}
	log.Error().Msgf("ByteSliceAllocator: Size class not found for size %d", len(p))
}
