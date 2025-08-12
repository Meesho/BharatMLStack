package indices

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/maths"
)

var (
	ErrGettingHeadEntry = errors.New("getting head entry failed")
)

type KeyIndex struct {
	rm      *RoundMap
	rb      *RingBuffer
	mc      *maths.MorrisLogCounter
	startAt int64
}

func NewKeyIndex(rounds int, rbInitial, rbMax, deleteAmortizedStep int) *KeyIndex {
	if ByteOrder == nil {
		loadByteOrder()
	}
	return &KeyIndex{
		rm:      NewRoundMap(rounds),
		rb:      NewRingBuffer(rbInitial, rbMax),
		mc:      maths.New(10),
		startAt: time.Now().Unix(),
	}
}

func (ki *KeyIndex) Put(key string, length uint16, memId, offset uint32, exptime uint64) {
	lastAccess := ki.GenerateLastAccess()
	freq := uint32(1)
	h64 := Hash64(key)
	h34 := Hash34(key)
	entry, idx, _ := ki.rb.GetEntry()
	round, next24bits, slicePos := ki.rm.Add(key, uint32(idx), h64, h34)
	encode(length, memId, offset, lastAccess, freq, exptime, round, next24bits, slicePos, entry)
}

func (ki *KeyIndex) GenerateLastAccess() uint32 {
	return uint32(time.Now().Unix()-ki.startAt) / 60
}

func (ki *KeyIndex) Get(key string) (uint32, uint16, uint32, uint32, uint64, uint64, uint32, bool) {
	h64 := Hash64(key)
	h34 := Hash34(key)
	idx, slicePos, found := ki.rm.Get(h64, h34)
	if !found {
		return 0, 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	entry, ok := ki.rb.Get(int(idx))
	if !ok {
		return 0, 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	length, memId, offset, lastAccessAt, freq, exptime, _, _, gotSlicePos := extract(entry)
	if gotSlicePos != slicePos {
		return 0, 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	lastAccess := ki.GenerateLastAccess()
	freq, _ = ki.mc.Inc(freq)
	encodeD2(length, lastAccess, freq, entry)
	lastAccess = ki.GenerateLastAccess() - lastAccessAt
	return memId, length, offset, lastAccess, ki.mc.Value(freq), exptime, uint32(idx), true
}

func (ki *KeyIndex) Delete(nKeys int) (uint32, int) {
	for i := 0; i < nKeys; i++ {
		deleted, next := ki.rb.Delete()
		if deleted == nil {
			return 0, -1
		}
		round, route, slicePos := extractD1(deleted)
		ki.rm.RemoveV2(round, route, slicePos)
		delMemId := extractMemId(deleted)
		nextMemId := extractMemId(next)
		if nextMemId == delMemId+1 {
			return nextMemId, i + 1
		} else if nextMemId == delMemId && i == nKeys-1 {
			return delMemId, i + 1
		} else if nextMemId == delMemId {
			continue
		} else {
			return 0, -1
		}
	}
	return 0, -1
}

func (ki *KeyIndex) GetRB() *RingBuffer {
	return ki.rb
}

func (ki *KeyIndex) PeekMemIdAtHead() (uint32, error) {
	entry, ok := ki.rb.Get(ki.rb.head)
	if !ok {
		return 0, ErrGettingHeadEntry
	}
	memId, _ := extractD3(entry)
	return memId, nil
}

// Debug methods to expose ring buffer state
func (ki *KeyIndex) GetRingBufferNextIndex() int {
	return ki.rb.nextIndex
}

func (ki *KeyIndex) GetRingBufferSize() int {
	return ki.rb.size
}

func (ki *KeyIndex) GetRingBufferCapacity() int {
	return ki.rb.capacity
}

func (ki *KeyIndex) GetRingBufferActiveEntries() int {
	return ki.rb.ActiveEntries()
}
