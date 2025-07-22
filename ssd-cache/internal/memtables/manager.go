package memtables

import (
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocators"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/fs"
	"github.com/rs/zerolog/log"
)

type MemtableManager struct {
	file     *fs.RollingAppendFile
	capacity int32

	memtable1       *Memtable
	memtable2       *Memtable
	activeMemtable  *Memtable
	nextFileOffset  int64
	nextId          uint32
	flushInProgress bool
	flushChan       chan *Memtable
}

func NewMemtableManager(file *fs.RollingAppendFile, capacity int32) (*MemtableManager, error) {
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
	return &MemtableManager{
		file:            file,
		capacity:        capacity,
		memtable1:       memtable1,
		memtable2:       memtable2,
		activeMemtable:  memtable1,
		nextFileOffset:  2 * int64(capacity),
		nextId:          2,
		flushInProgress: false,
		flushChan:       make(chan *Memtable, 1),
	}, nil
}

func (mm *MemtableManager) GetMemtable() (*Memtable, uint32, uint64) {
	return mm.activeMemtable, mm.activeMemtable.Id, uint64(mm.activeMemtable.Id) * uint64(mm.capacity)
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

func (mm *MemtableManager) flushConsumer() {
	for {
		memtable, ok := <-mm.flushChan
		if !ok {
			break
		}
		n, fileOffset, err := memtable.Flush()
		if n != int(mm.capacity) {
			log.Error().Msgf("Flush size mismatch: memId:%d fileOffset:%d nextFileOffset:%d n:%d err:%v", memtable.Id, fileOffset, mm.nextFileOffset, n, err)
		}
		if err != nil {
			log.Error().Msgf("Failed to flush memtable: memId:%d fileOffset:%d nextFileOffset:%d n:%d err:%v", memtable.Id, fileOffset, mm.nextFileOffset, n, err)
		}
	}
}
func (mm *MemtableManager) Flush() error {
	if mm.flushInProgress {
		// Wait for previous flush to complete before starting new one
		<-mm.flushChan
	}

	mm.flushInProgress = true

	memtableToFlush := mm.activeMemtable

	// Swap to the other memtable
	if mm.activeMemtable == mm.memtable1 {
		mm.activeMemtable = mm.memtable2
	} else {
		mm.activeMemtable = mm.memtable1
	}

	// Async flush
	go func() {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		log.Error().Msgf("Recovered from panic: %v", r)
		// 		mm.flushInProgress = false
		// 		mm.flushChan <- struct{}{}
		// 	}
		// }()
		if !memtableToFlush.readyForFlush {
			panic("memtable not ready for flush")
		}
		n, fileOffset, err := memtableToFlush.Flush()
		if n != int(mm.capacity) {
			log.Error().Msgf("Flush size mismatch: memId:%d fileOffset:%d nextFileOffset:%d n:%d err:%v", memtableToFlush.Id, fileOffset, mm.nextFileOffset, n, err)
		}
		if err != nil {
			log.Error().Msgf("Failed to flush memtable: memId:%d fileOffset:%d nextFileOffset:%d n:%d err:%v", memtableToFlush.Id, fileOffset, mm.nextFileOffset, n, err)
		}

		// Update metadata (only flushed memtable touched here)
		memtableToFlush.Id = mm.nextId
		mm.nextId++
		mm.nextFileOffset += int64(n)

		// Mark flush as done and notify to unblock next flush
		mm.flushInProgress = false
		mm.flushChan <- struct{}{}
	}()

	return nil
}
