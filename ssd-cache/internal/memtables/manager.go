package memtables

import (
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocators"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/fs"
	"github.com/rs/zerolog/log"
)

type MemtableManager struct {
	file     *fs.WrapAppendFile
	capacity int32

	memtable1         *Memtable
	memtable2         *Memtable
	activeMemtable    *Memtable
	nextFileOffset    int64
	nextId            uint32
	flushInProgress   bool
	flushChan         chan *Memtable
	flushCompleteChan chan struct{}
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
		file:              file,
		capacity:          capacity,
		memtable1:         memtable1,
		memtable2:         memtable2,
		activeMemtable:    memtable1,
		nextFileOffset:    2 * int64(capacity),
		nextId:            2,
		flushChan:         make(chan *Memtable, 1),
		flushCompleteChan: make(chan struct{}, 1),
		flushInProgress:   false,
	}
	go memtableManager.flushConsumer()
	return memtableManager, nil
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
		mm.flushInProgress = true
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
		memtable.Id = mm.nextId
		mm.nextId++
		mm.nextFileOffset += int64(n)
		mm.flushInProgress = false
		mm.flushCompleteChan <- struct{}{}
	}
}
func (mm *MemtableManager) Flush() error {

	if mm.flushInProgress {
		// Wait for previous flush to complete before starting new one
		<-mm.flushCompleteChan
	}

	mm.flushInProgress = true

	memtableToFlush := mm.activeMemtable
	mm.flushChan <- memtableToFlush

	// Swap to the other memtable
	if mm.activeMemtable == mm.memtable1 {
		mm.activeMemtable = mm.memtable2
	} else {
		mm.activeMemtable = mm.memtable1
	}

	return nil
}
