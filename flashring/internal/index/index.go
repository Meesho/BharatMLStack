package index

import (
	"errors"
	"sync"
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

type Index struct {
	mu       *sync.RWMutex
	rm       map[uint64]int
	rb       *RingBuffer
	mc       *maths.MorrisLogCounter
	startAt  int64
	hashBits int
}

func NewIndex(hashBits int, rbInitial, rbMax, deleteAmortizedStep int, mu *sync.RWMutex) *Index {
	return &Index{
		mu:       mu,
		rm:       make(map[uint64]int),
		rb:       NewRingBuffer(rbInitial, rbMax),
		mc:       maths.New(),
		startAt:  time.Now().Unix(),
		hashBits: hashBits,
	}
}

func (i *Index) Put(key string, length, ttlInMinutes uint16, memId, offset uint32) {
	hhi, hlo := hash128(key)
	entry, hashNextPrev, idx, _ := i.rb.GetNextFreeSlot()
	lastAccess := i.generateLastAccess()
	freq := uint16(1)
	expiryAt := (time.Now().Unix() / 60) + int64(ttlInMinutes)
	delta := uint16(expiryAt - (i.startAt / 60))
	encode(key, length, delta, lastAccess, freq, memId, offset, entry)

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

	for {
		entry, hashNextPrev, _ := i.rb.Get(int(idx))
		if isHashMatch(hhi, hlo, hashNextPrev) {
			length, deltaExptime, oldLastAccess, freq, memId, offset := decode(entry)
			exptime := int(deltaExptime) + int(i.startAt/60)
			currentTime := int(time.Now().Unix() / 60)
			remainingTTL := exptime - currentTime
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
}

func (ix *Index) Delete(count int) (uint32, int) {
	if count == 0 {
		return 0, 0
	}
	for i := 0; i < count; i++ {
		deleted, deletedHashNextPrev, deletedIdx, next := ix.rb.Delete()
		if deleted == nil {
			return 0, -1
		}
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
	return uint16((time.Now().Unix() - i.startAt) / 60)
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
