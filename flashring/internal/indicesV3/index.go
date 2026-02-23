package indicesv2

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
	if ByteOrder == nil {
		loadByteOrder()
	}
	// rm := make(map[uint64]int)
	return &Index{
		mu:       mu,
		rm:       make(map[uint64]int),
		rb:       NewRingBuffer(rbInitial, rbMax),
		mc:       maths.New(12),
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
		return
	} else {
		_, headHashNextPrev, _ := i.rb.Get(int(headIdx))
		encodeUpdatePrev(int32(idx), headHashNextPrev)
		encodeHashNextPrev(hhi, hlo, -1, int32(headIdx), hashNextPrev)
		i.rm[hlo] = idx
		return
	}

}

func (i *Index) Get(key string) (length, lastAccess, remainingTTL uint16, freq uint64, memId, offset uint32, status Status) {
	hhi, hlo := hash128(key)

	i.mu.RLock()
	idx, ok := i.rm[hlo]
	i.mu.RUnlock()

	if ok {
		for {
			entry, hashNextPrev, _ := i.rb.Get(int(idx))
			if isHashMatch(hhi, hlo, hashNextPrev) {
				length, deltaExptime, lastAccess, freq, memId, offset := decode(entry)
				exptime := int(deltaExptime) + int(i.startAt/60)
				currentTime := int(time.Now().Unix() / 60)
				remainingTTL := exptime - currentTime
				if remainingTTL <= 0 {
					return 0, 0, 0, 0, 0, 0, StatusExpired
				}
				lastAccess = i.generateLastAccess()
				freq = i.incrFreq(freq)
				encodeLastAccessNFreq(lastAccess, freq, entry)
				return length, lastAccess, uint16(remainingTTL), i.mc.Value(uint32(freq)), memId, offset, StatusOK
			}
			if hasNext(hashNextPrev) {
				idx = int(decodeNext(hashNextPrev))
			} else {
				return 0, 0, 0, 0, 0, 0, StatusNotFound
			}
		}

	}
	return 0, 0, 0, 0, 0, 0, StatusNotFound
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
		delMemId, _ := decodeMemIdOffset(deleted)
		deletedHlo := decodeHashLo(deletedHashNextPrev)
		mapIdx, ok := ix.rm[deletedHlo]
		if ok && mapIdx == deletedIdx {
			delete(ix.rm, deletedHlo)
		} else if ok && hasPrev(deletedHashNextPrev) {
			prevIdx := decodePrev(deletedHashNextPrev)
			_, hashNextPrev, _ := ix.rb.Get(int(prevIdx))
			encodeUpdateNext(-1, hashNextPrev)
		} else {
			//log.Warn().Msgf("broken link. Entry in RB but cannot be linked to map. deletedIdx: %d", deletedIdx)
		}

		nextMemId, _ := decodeMemIdOffset(next)
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

func (ki *Index) GetRB() *RingBuffer {
	return ki.rb
}

func (ki *Index) PeekMemIdAtHead() (uint32, error) {
	entry, _, ok := ki.rb.Get(ki.rb.head)
	if !ok {
		return 0, ErrGettingHeadEntry
	}
	memId, _ := decodeMemIdOffset(entry)
	return memId, nil
}

func (i *Index) generateLastAccess() uint16 {
	return uint16((time.Now().Unix() - i.startAt) / 60)
}

func (i *Index) incrFreq(freq uint16) uint16 {
	newFreq, _ := i.mc.Inc(uint32(freq))
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
