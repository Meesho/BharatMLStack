package indicesv2

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
)

var ErrGettingHeadEntry = errors.New("getting head entry failed")

type Status int

const (
	StatusOK Status = iota
	StatusNotFound
	StatusExpired
)

type Index struct {
	rm       map[string]int
	rb       *RingBuffer
	mc       *maths.MorrisLogCounter
	startAt  int64
	hashBits int
}

func NewIndex(hashBits int, rbInitial, rbMax, deleteAmortizedStep int) *Index {
	if ByteOrder == nil {
		loadByteOrder()
	}
	rm := make(map[string]int)
	return &Index{
		rm:       rm,
		rb:       NewRingBuffer(rbInitial, rbMax),
		mc:       maths.New(12),
		startAt:  time.Now().Unix(),
		hashBits: hashBits,
	}
}

func (i *Index) Put(key string, length, ttlInMinutes uint16, memId, offset uint32) {
	if _, ok := i.rm[key]; ok {
		idx := i.rm[key]
		entry, _ := i.rb.Get(idx)
		length, delta, lastAccess, freq, _, _ := decode(entry)
		idx, _ = i.rb.PutInNextFreeSlot(func(entry *Entry) string {
			encode(key, length, delta, lastAccess, freq, memId, offset, entry)
			return key
		})
		i.rm[key] = idx
		return
	}
	lastAccess := i.generateLastAccess()
	freq := uint16(1)
	expiryAt := (time.Now().Unix() / 60) + int64(ttlInMinutes)
	delta := uint16(expiryAt - (i.startAt / 60))
	idx, _ := i.rb.PutInNextFreeSlot(func(entry *Entry) string {
		encode(key, length, delta, lastAccess, freq, memId, offset, entry)
		return key
	})
	i.rm[key] = idx
}

func (i *Index) Get(key string) (length, lastAccess, remainingTTL uint16, freq uint64, memId, offset uint32, status Status) {
	if idx, ok := i.rm[key]; ok {
		entry, _ := i.rb.Get(idx)
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
	return 0, 0, 0, 0, 0, 0, StatusNotFound
}

func (ix *Index) Delete(count int) (uint32, int) {
	for i := 0; i < count; i++ {
		deleted, deletedKey, next, _ := ix.rb.Delete()
		if deleted == nil {
			return 0, -1
		}
		delMemId, _ := decodeMemIdOffset(deleted)
		delete(ix.rm, deletedKey)
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
	entry, ok := ki.rb.Get(ki.rb.head)
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
