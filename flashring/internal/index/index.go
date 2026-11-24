package index

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/xxh3"
)

var ErrGettingHeadEntry = errors.New("getting head entry failed")

type Status int

const (
	StatusOK Status = iota
	StatusNotFound
	StatusExpired
)

// Index is the in-memory hash table that maps keys to their (memId,
// offset) location in the shard's data file. Entries live in a fixed-size
// RingBuffer rb and are addressed indirectly through rm, a map from
// xxh3(key) to ring-buffer slot. Hash collisions are resolved by chaining
// slots together via prev/next pointers stored in each slot's
// HashNextPrev triple — the high 64 bits also hold xxhash(key) so Get
// can verify the chain entry actually belongs to the requested key
// (dual-hash collision detection).
//
// Frequency is tracked with a per-key MorrisLogCounter; lastAccess is
// stored as minutes-since-startAt to fit in 16 bits. Concurrency is
// controlled by mu, which is owned externally (the shard) so the
// caller can hold the write lock across Index + memtable + file
// operations.
type Index struct {
	mu       *sync.RWMutex
	rm       map[uint64]int
	rb       *RingBuffer
	mc       *maths.MorrisLogCounter
	startAt  time.Time // read with time.Since, so a wall-clock step cannot move it
	hashBits int

	// memBirth[memId % len] holds memId<<32 | the minute (since startAt)
	// that memtable id took its first write. Written by Put under the
	// shard lock, read lock-free by Get.
	memBirth []atomic.Uint64
}

// NewIndex builds an index. liveMemtables is how many memtable ids can hold
// readable entries at once: the file's slots plus the two in memory.
func NewIndex(hashBits int, rbInitial, rbMax, deleteAmortizedStep int, liveMemtables int, mu *sync.RWMutex) *Index {
	ix := &Index{
		mu:       mu,
		rm:       make(map[uint64]int),
		rb:       NewRingBuffer(rbInitial, rbMax),
		mc:       maths.New(),
		startAt:  time.Now(),
		hashBits: hashBits,
		memBirth: make([]atomic.Uint64, liveMemtables),
	}
	for s := range ix.memBirth {
		ix.memBirth[s].Store(math.MaxUint64) // no memtable id yet
	}
	return ix
}

// Expiry is stored in the 16-bit entry field as minutes since the entry's
// memtable took its first write, and that moment is kept per memtable id in
// memBirth. An entry's absolute expiry is birth + delta, so it never wraps
// however long the process runs. Two limits keep the delta inside 16 bits:
// a memtable stops taking writes once it is MaxMemtableAgeMinutes old (the
// shard seals it), and a TTL is at most MaxTTLMinutes.
const (
	MaxMemtableAgeMinutes = 4096 // about 2.8 days
	MaxTTLMinutes         = math.MaxUint16 - MaxMemtableAgeMinutes
)

// nowMinutes is the time since the index was created, in minutes.
func (i *Index) nowMinutes() int64 {
	return int64(time.Since(i.startAt) / time.Minute)
}

// memBirthOf returns the minute memId took its first write, or false when
// the slot has since been taken by a newer memtable id: memId's file slot
// has been overwritten by then, so its entries are dead.
func (i *Index) memBirthOf(memId uint32) (int64, bool) {
	v := i.memBirth[int(memId)%len(i.memBirth)].Load()
	if uint32(v>>32) != memId {
		return 0, false
	}
	return int64(uint32(v)), true
}

// MemtableTooOld reports whether memId has been taking writes for longer
// than MaxMemtableAgeMinutes. The shard then seals it so the next write
// starts a new memtable id with a fresh birth.
func (i *Index) MemtableTooOld(memId uint32) bool {
	birth, ok := i.memBirthOf(memId)
	return ok && i.nowMinutes()-birth > MaxMemtableAgeMinutes
}

func (i *Index) Put(key string, length, ttlInMinutes uint16, memId, offset uint32) {
	hhi, hlo := hash128(key)
	entry, hashNextPrev, idx, _ := i.rb.GetNextFreeSlot()
	lastAccess := i.generateLastAccess()
	freq := uint16(1)
	now := i.nowMinutes()
	birth, ok := i.memBirthOf(memId)
	if !ok {
		// First write into this memtable id.
		birth = now
		i.memBirth[int(memId)%len(i.memBirth)].Store(uint64(memId)<<32 | uint64(uint32(now)))
	}
	// Clamped, not wrapped: a too-long TTL then expires early, never late.
	delta := max(0, min(now-birth+int64(ttlInMinutes), math.MaxUint16))
	encode(key, length, uint16(delta), lastAccess, freq, memId, offset, entry)

	if headIdx, ok := i.rm[hlo]; !ok {
		encodeHashNextPrev(hhi, hlo, -1, -1, hashNextPrev)
		i.rm[hlo] = idx
	} else {
		_, headHashNextPrev, _ := i.rb.Get(int(headIdx))
		encodeUpdatePrev(int32(idx), headHashNextPrev)
		encodeHashNextPrev(hhi, hlo, -1, int32(headIdx), hashNextPrev)
		i.rm[hlo] = idx
	}
}

func (i *Index) Get(key string) (length, lastAccess, remainingTTL uint16, freq uint64, memId, offset uint32, status Status) {
	hhi, hlo := hash128(key)

	i.mu.RLock()
	idx, ok := i.rm[hlo]
	i.mu.RUnlock()

	if !ok {
		return 0, 0, 0, 0, 0, 0, StatusNotFound
	}

	// Bounded: a chain is at most the ring's capacity long, so a longer
	// walk means the links are corrupt and must not spin a request forever.
	for steps := 0; steps <= i.rb.capacity; steps++ {
		entry, hashNextPrev, _ := i.rb.Get(int(idx))
		if isHashMatch(hhi, hlo, hashNextPrev) {
			length, deltaExptime, oldLastAccess, freq, memId, offset := decode(entry)
			birth, ok := i.memBirthOf(memId)
			if !ok {
				return 0, 0, 0, 0, 0, 0, StatusNotFound
			}
			remainingTTL := birth + int64(deltaExptime) - i.nowMinutes()
			if remainingTTL <= 0 {
				return 0, 0, 0, 0, 0, 0, StatusExpired
			}
			newLastAccess := i.generateLastAccess()
			recency := newLastAccess - oldLastAccess // minutes since previous access
			freq = i.incrFreq(freq, hlo)
			encodeLastAccessNFreq(newLastAccess, freq, entry)
			return length, recency, uint16(remainingTTL), i.mc.Value(uint16(freq)), memId, offset, StatusOK
		}
		if hasNext(hashNextPrev) {
			idx = int(decodeNext(hashNextPrev))
		} else {
			return 0, 0, 0, 0, 0, 0, StatusNotFound
		}
	}
	return 0, 0, 0, 0, 0, 0, StatusNotFound
}

// Delete reclaims up to count slots from the ring buffer's head, walking
// forward until the head's memId crosses the boundary of the memtable
// being recycled. Returns (newHeadMemId, slotsDeleted) on a successful
// boundary crossing, (currentMemId, slotsDeleted) when the count is
// fully drained while still inside the same memtable's region, or
// (0, -1) on a corruption-class error (e.g. ring drained but no boundary
// found). DeleteManager calls this in amortized chunks so a single Put
// never pays the full cost of recycling a memtable's worth of entries.
func (ix *Index) Delete(count int) (uint32, int) {
	if count == 0 {
		return 0, 0
	}
	for i := 0; i < count; i++ {
		deleted, deletedHashNextPrev, deletedIdx, next := ix.rb.Delete()
		// RingBuffer.Delete returns pointers that are always non-nil (Go
		// escape analysis heap-allocates the local copies), so no nil
		// guard is needed here.
		delMemId, _ := DecodeMemIdOffset(deleted)
		deletedHlo := decodeHashLo(deletedHashNextPrev)
		mapIdx, ok := ix.rm[deletedHlo]
		if ok && mapIdx == deletedIdx {
			delete(ix.rm, deletedHlo)
		} else if ok && hasPrev(deletedHashNextPrev) {
			prevIdx := decodePrev(deletedHashNextPrev)
			_, hashNextPrev, _ := ix.rb.Get(int(prevIdx))
			encodeUpdateNext(-1, hashNextPrev)
		}

		nextMemId, _ := DecodeMemIdOffset(next)
		if nextMemId == delMemId+1 {
			return nextMemId, i + 1
		} else if nextMemId == delMemId && i == count-1 {
			return delMemId, i + 1
		} else if nextMemId == delMemId {
			continue
		} else {
			return 0, -1
		}
	}
	return 0, -1
}

// DeleteKey removes the key from the index map only. Debug use only.
func (ix *Index) DeleteKey(key string) bool {
	_, hlo := hash128(key)
	if _, ok := ix.rm[hlo]; !ok {
		return false
	}
	delete(ix.rm, hlo)
	return true
}

func (ki *Index) GetRB() *RingBuffer {
	return ki.rb
}

func (ki *Index) PeekMemIdAtHead() (uint32, error) {
	entry, _, ok := ki.rb.Get(ki.rb.head)
	if !ok {
		return 0, ErrGettingHeadEntry
	}
	memId, _ := DecodeMemIdOffset(entry)
	return memId, nil
}

func (i *Index) generateLastAccess() uint16 {
	return uint16(i.nowMinutes())
}

func (i *Index) incrFreq(freq uint16, hlo uint64) uint16 {
	newFreq, _ := i.mc.Inc(uint16(freq), hlo)
	return uint16(newFreq)
}

func hash128(key string) (uint64, uint64) {
	return xxhash.Sum64String(key), xxh3.HashString(key)
}

func isHashMatch(hhi, hlo uint64, entry *HashNextPrev) bool {
	return entry[0] == hhi && entry[1] == hlo
}

func hasNext(entry *HashNextPrev) bool {
	return int32(entry[2]&NEXT_MASK) != -1
}

func hasPrev(entry *HashNextPrev) bool {
	return int32((entry[2]>>32)&PREV_MASK) != -1
}
