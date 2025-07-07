package memtable

import (
	"os"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocator"
)

type MemtableManager struct {
	file      *os.File
	capacity  int64
	allocator *allocator.AlignedPageAllocator

	// Two memtables for swapping
	memtable1 *Memtable
	memtable2 *Memtable

	// Which one is active for writing
	activeMemtable *Memtable

	// Next file offset for flushing
	nextFileOffset int64

	// ID management
	nextId int64

	// Flush state management
	flushInProgress bool
	flushChan       chan struct{}
}

func NewMemtableManager(file *os.File, capacity int64, allocator *allocator.AlignedPageAllocator) *MemtableManager {
	manager := &MemtableManager{
		file:            file,
		capacity:        capacity,
		allocator:       allocator,
		nextFileOffset:  0,
		nextId:          0,
		flushInProgress: false,
		flushChan:       make(chan struct{}, 1),
	}

	// Initialize two memtables with proper fileOffsets
	manager.memtable1 = NewMemtableV2(file, manager.nextFileOffset, capacity, allocator, manager.nextId)
	manager.nextFileOffset += capacity
	manager.nextId++

	manager.memtable2 = NewMemtableV2(file, manager.nextFileOffset, capacity, allocator, manager.nextId)
	manager.nextFileOffset += capacity
	manager.nextId++

	// Set first memtable as active
	manager.activeMemtable = manager.memtable1

	return manager
}

// GetMemtable returns the active memtable for writing and its ID
func (mm *MemtableManager) GetMemtable() (*Memtable, int64, int64) {
	return mm.activeMemtable, mm.activeMemtable.Id, mm.activeMemtable.fileOffset
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
		memtableToFlush.Flush()

		// Update metadata (only flushed memtable touched here)
		memtableToFlush.Id = mm.nextId
		mm.nextId++
		memtableToFlush.fileOffset = mm.nextFileOffset
		mm.nextFileOffset += mm.capacity

		// Mark flush as done and notify
		mm.flushInProgress = false
		mm.flushChan <- struct{}{}
	}()

	return nil
}

// GetMemtableById returns the memtable with the given ID, or nil if not found
func (mm *MemtableManager) GetMemtableById(id int64) *Memtable {
	if mm.memtable1.Id == id {
		return mm.memtable1
	}
	if mm.memtable2.Id == id {
		return mm.memtable2
	}

	// For old identifiers, return nil
	return nil
}
