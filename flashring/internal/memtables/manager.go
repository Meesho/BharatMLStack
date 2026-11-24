package memtables

import (
	"errors"
	"sync/atomic"

	"github.com/Meesho/BharatMLStack/flashring/internal/allocators"
	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"github.com/rs/zerolog/log"
)

// ErrPreviousFlushFailed is returned by ShardCache.Put when the async
// flushConsumer's most recent flush failed. The frozen memtable's page
// has already been recycled by the time flushConsumer errors, so its
// entries are lost. Surface once (compare-and-swap on the pending flag),
// then subsequent Puts proceed normally against the fresh memtable.
var ErrPreviousFlushFailed = errors.New("previous memtable flush failed — its entries were lost")

// MemtableManager owns the double-buffered memtable pair for a single
// shard. Writers serve out of activeMemtable until it fills; Flush then
// swaps in the other memtable instantly while a background goroutine
// pwrites the now-frozen one to disk. The semaphore caps in-flight
// flushes at one per shard, so a swap that finds the previous flush
// still running blocks until the disk catches up. memId monotonically
// increases across flushes and is encoded into every index entry, so
// stale readers that hold a memId can detect when their memtable has
// since been recycled (GetMemtableById returns nil).
type MemtableManager struct {
	file     *fs.WrapAppendFile
	Capacity int32

	memtable1      *Memtable
	memtable2      *Memtable
	activeMemtable atomic.Pointer[Memtable] // swapped by Flush, read lockless by Get
	nextFileOffset int64
	nextId         uint32
	semaphore      chan int

	// ShardIdx tags the flush counter. Set by the owning shard before use.
	ShardIdx uint32

	// flushErrPending is set to true by flushConsumer when Memtable.Flush
	// returns an error. Consumed (compare-and-swap to false) by the next
	// caller via ConsumeFlushErr — surfacing an ErrPreviousFlushFailed
	// exactly once per failed flush event.
	flushErrPending atomic.Bool
}

// ConsumeFlushErr returns true if the most recent async flush failed and
// hasn't yet been reported. Atomically clears the pending flag so the
// next caller sees a clean state. Called by WrapCache.Put at the top of
// each user-visible Put to convert async data-loss into a synchronous,
// caller-visible persist failure.
func (mm *MemtableManager) ConsumeFlushErr() bool {
	return mm.flushErrPending.CompareAndSwap(true, false)
}

// SetFlushErrForTest arms the pending-flush-error flag from tests without
// having to synthesise a real Memtable.Flush failure. Production callers
// must NOT use this — flushConsumer is the only legitimate setter. Kept
// here (rather than a _test.go file) so consumers in sibling packages
// (pkg/cache, internal/shard) can use it in their own tests.
func (mm *MemtableManager) SetFlushErrForTest() {
	mm.flushErrPending.Store(true)
}

// PeekFlushErrForTest returns the flag without clearing it. Same
// production-forbidden caveat as SetFlushErrForTest.
func (mm *MemtableManager) PeekFlushErrForTest() bool {
	return mm.flushErrPending.Load()
}

// NewMemtableManager creates a double-buffered memtable pair.
// flushStaggerOffset advances the active memtable's write position so that
// different shards fill (and therefore flush) at staggered times, avoiding
// synchronized flush storms that compete with reads for NVMe bandwidth.
func NewMemtableManager(file *fs.WrapAppendFile, capacity int32, flushStaggerOffset int) (*MemtableManager, error) {
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
	// Pre-advance the active memtable so this shard's first flush happens
	// earlier/later than its peers, spreading flush I/O over time.
	memtable1.currentOffset = flushStaggerOffset
	memtable1.flushStartOffset = flushStaggerOffset

	memtableManager := &MemtableManager{
		file:           file,
		Capacity:       capacity,
		memtable1:      memtable1,
		memtable2:      memtable2,
		nextFileOffset: 2 * int64(capacity),
		nextId:         2,
		semaphore:      make(chan int, 1),
	}
	memtableManager.activeMemtable.Store(memtable1)
	return memtableManager, nil
}

func (mm *MemtableManager) GetMemtable() (*Memtable, uint32, uint64) {
	mt := mm.activeMemtable.Load()
	id := atomic.LoadUint32(&mt.Id)
	return mt, id, uint64(id) * uint64(mm.Capacity)
}

func (mm *MemtableManager) GetMemtableById(id uint32) *Memtable {
	// Atomic loads pair with the atomic.StoreUint32 in flushConsumer —
	// the flush goroutine renames the recycled memtable's Id without any
	// lock, so a lockless read from the shard's Get path would otherwise
	// race per the Go memory model.
	if atomic.LoadUint32(&mm.memtable1.Id) == id {
		return mm.memtable1
	}
	if atomic.LoadUint32(&mm.memtable2.Id) == id {
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
		// Surface the failure to the next Put via ConsumeFlushErr. The
		// memtable's entries are gone: its file slot was consumed without
		// being fully written (see WrapAppendFile.PwriteBatch) and the
		// page is recycled below, so reads of them miss on CRC/key.
		mm.flushErrPending.Store(true)
		metrics.Incr(metrics.KEY_MEMTABLE_FLUSH_ERROR_COUNT, metrics.GetShardTag(mm.ShardIdx))
	}
	// Atomic store so the lockless read in GetMemtableById sees a
	// well-defined value under the Go memory model. nextId and
	// nextFileOffset are only mutated inside flushConsumer and read on
	// the next flush (serialized by the semaphore), so they don't need
	// atomics.
	atomic.StoreUint32(&memtable.Id, mm.nextId)
	mm.nextId++
	mm.nextFileOffset += int64(mm.Capacity) // a failed flush still consumes its slot
	metrics.Incr(metrics.KEY_MEMTABLE_FLUSH_COUNT, metrics.GetShardTag(mm.ShardIdx))
}

// WaitForFlush blocks until the in-flight background flush, if any, has
// finished.
func (mm *MemtableManager) WaitForFlush() {
	mm.semaphore <- 1
	<-mm.semaphore
}

// Flush swaps the active memtable with its buddy and pushes the frozen
// memtable through the file's io_uring writer in a goroutine. Returns
// nil immediately — the disk write is asynchronous. Subsequent Flush
// calls block on the per-shard semaphore until the previous flush
// completes, bounding the number of in-flight memtables at two
// (active + flushing).
func (mm *MemtableManager) Flush() error {

	memtableToFlush := mm.activeMemtable.Load()
	mm.semaphore <- 1

	// Swap to the other memtable
	if memtableToFlush == mm.memtable1 {
		mm.activeMemtable.Store(mm.memtable2)
	} else {
		mm.activeMemtable.Store(mm.memtable1)
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
