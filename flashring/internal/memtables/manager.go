package memtables

import (
	"github.com/Meesho/BharatMLStack/flashring/internal/allocators"
	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"github.com/rs/zerolog/log"
)

type MemtableManager struct {
	file     *fs.WrapAppendFile
	Capacity int32

	memtable1      *Memtable
	memtable2      *Memtable
	activeMemtable *Memtable
	nextFileOffset int64
	nextId         uint32
	semaphore      chan int
}

func NewMemtableManager(file *fs.WrapAppendFile, capacity int32) (*MemtableManager, error) {
	allocatorConfig := allocators.SlabAlignedPageAllocatorConfig{
		SizeClasses: []allocators.SizeClass{
			{Size: int(capacity), MinCount: 2},
		},
	}
	allocator, err := allocators.NewSlabAlignedPageAllocator(allocatorConfig)
	if err != nil {
		return nil, err
	}
	page1 := allocator.Get(int(capacity))
	page2 := allocator.Get(int(capacity))
	memtable1, err := NewMemtable(MemtableConfig{
		capacity: int(capacity),
		id:       0,
		page:     page1,
		file:     file,
	})
	if err != nil {
		return nil, err
	}
	memtable2, err := NewMemtable(MemtableConfig{
		capacity: int(capacity),
		id:       1,
		page:     page2,
		file:     file,
	})
	if err != nil {
		return nil, err
	}
	memtableManager := &MemtableManager{
		file:           file,
		Capacity:       capacity,
		memtable1:      memtable1,
		memtable2:      memtable2,
		activeMemtable: memtable1,
		nextFileOffset: 2 * int64(capacity),
		nextId:         2,
		semaphore:      make(chan int, 1),
	}
	return memtableManager, nil
}

func (mm *MemtableManager) GetMemtable() (*Memtable, uint32, uint64) {
	return mm.activeMemtable, mm.activeMemtable.Id, uint64(mm.activeMemtable.Id) * uint64(mm.Capacity)
}

func (mm *MemtableManager) GetMemtableById(id uint32) *Memtable {
	if mm.memtable1.Id == id {
		return mm.memtable1
	}
	if mm.memtable2.Id == id {
		return mm.memtable2
	}
	return nil
}

func (mm *MemtableManager) flushConsumer(memtable *Memtable) {
	n, fileOffset, err := memtable.Flush()
	if n != int(mm.Capacity) {
		log.Error().Msgf("Flush size mismatch: memId:%d fileOffset:%d nextFileOffset:%d n:%d err:%v", memtable.Id, fileOffset, mm.nextFileOffset, n, err)
	}
	if err != nil {
		log.Error().Msgf("Failed to flush memtable: memId:%d fileOffset:%d nextFileOffset:%d n:%d err:%v", memtable.Id, fileOffset, mm.nextFileOffset, n, err)
	}
	memtable.Id = mm.nextId
	mm.nextId++
	mm.nextFileOffset += int64(n)
	metrics.Incr(metrics.KEY_MEMTABLE_FLUSH_COUNT, append(metrics.GetShardTag(memtable.ShardIdx), metrics.GetMemtableTag(memtable.Id)...))
}
func (mm *MemtableManager) Flush() error {

	memtableToFlush := mm.activeMemtable
	mm.semaphore <- 1

	// Swap to the other memtable
	if mm.activeMemtable == mm.memtable1 {
		mm.activeMemtable = mm.memtable2
	} else {
		mm.activeMemtable = mm.memtable1
	}
	go func() {
		defer func() {
			<-mm.semaphore
			if r := recover(); r != nil {
				log.Error().Msgf("Recovered from panic in goroutine: %v", r)
			}
		}()
		mm.flushConsumer(memtableToFlush)
	}()

	return nil
}
