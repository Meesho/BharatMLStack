package indices

import (
	"encoding/binary"
	"time"
	"unsafe"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/maths"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"
)

var ByteOrder *CustomByteOrder

type CustomByteOrder struct {
	binary.ByteOrder
}

func loadByteOrder() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		ByteOrder = &CustomByteOrder{binary.LittleEndian}
	case [2]byte{0xAB, 0xCD}:
		ByteOrder = &CustomByteOrder{binary.BigEndian}
	default:
		panic("Could not determine endianness.")
	}
}

func (c *CustomByteOrder) PutInt64(b []byte, v int64) {
	c.PutUint64(b, uint64(v))
}

func (c *CustomByteOrder) Int64(b []byte) int64 {
	return int64(c.Uint64(b))
}

func (c *CustomByteOrder) PutInt32(b []byte, v int32) {
	c.PutUint32(b, uint32(v))
}

func (c *CustomByteOrder) Int32(b []byte) int32 {
	return int32(c.Uint32(b))
}

func (c *CustomByteOrder) PutUint32(b []byte, v uint32) {
	c.ByteOrder.PutUint32(b, v)
}

func (c *CustomByteOrder) Uint32(b []byte) uint32 {
	return c.ByteOrder.Uint32(b)
}

type KeyIndex struct {
	rm                  *RoundMap
	rb                  *RingBuffer
	mc                  *maths.MorrisLogCounter
	startAt             int64
	deleteAmortizedStep int
	deleteInProgress    bool
}

func NewKeyIndex(rounds int, rbInitial, rbMax, deleteAmortizedStep int) *KeyIndex {
	if ByteOrder == nil {
		loadByteOrder()
	}
	return &KeyIndex{
		rm:                  NewRoundMap(rounds),
		rb:                  NewRingBuffer(rbInitial, rbMax),
		mc:                  maths.New(10),
		startAt:             time.Now().Unix(),
		deleteAmortizedStep: deleteAmortizedStep,
		deleteInProgress:    false,
	}
}

func (ki *KeyIndex) Add(key string, length uint16, memId, offset, lastAccess, freq uint32, exptime uint64) {
	h64 := xxhash.Sum64String(key)
	h10 := Hash10(key)
	d1 := uint64(length)<<48 | uint64(lastAccess)<<24 | uint64(freq)&0xFFFFFF
	d2 := uint64(h10)<<54 | uint64(exptime)&_LO_54BIT_IN_64BIT
	entry, idx, _ := ki.rb.GetEntry()
	ByteOrder.PutUint64(entry[:8], h64)
	ByteOrder.PutUint32(entry[8:12], memId)
	ByteOrder.PutUint32(entry[12:16], offset)
	ByteOrder.PutUint64(entry[16:24], d1)
	ByteOrder.PutUint64(entry[24:32], d2)
	ki.rm.AddV2(key, uint32(idx), h64, h10)
	if ki.deleteInProgress {
		i := 0
		for i < ki.deleteAmortizedStep {
			deleted, next, ok := ki.delete()
			if !ok {
				ki.deleteInProgress = false
				log.Warn().Msg("no more entries to delete.")
				break
			}
			ki.rm.RemoveV2(extractHash(deleted))
			delMemId := extractMemId(deleted)
			nextMemId := extractMemId(next)
			if delMemId != nextMemId {
				ki.deleteInProgress = false
				break
			}
			i++
		}
	}
}

func (ki *KeyIndex) Put(key string, length uint16, memId, offset uint32, exptime uint64) {
	lastAccess := ki.GenerateLastAccess()
	freq := uint32(1)
	h64 := xxhash.Sum64String(key)
	h10 := Hash10(key)
	entry, idx, shouldDelete := ki.rb.GetEntry()
	encode(length, memId, offset, lastAccess, freq, exptime, h64, h10, entry)
	ki.rm.AddV2(key, uint32(idx), h64, h10)
	if ki.deleteInProgress || shouldDelete {
		i := 0
		for i < ki.deleteAmortizedStep {
			deleted, next, ok := ki.delete()
			if !ok {
				ki.deleteInProgress = false
				log.Warn().Msg("no more entries to delete.")
				break
			}
			ki.rm.RemoveV2(extractHash(deleted))
			delMemId := extractMemId(deleted)
			nextMemId := extractMemId(next)
			if delMemId != nextMemId {
				ki.deleteInProgress = false
				break
			}
			i++
		}
	}
}

func (ki *KeyIndex) PutV2(key string, length uint16, memId, offset uint32, exptime uint64) {
	lastAccess := ki.GenerateLastAccess()
	freq := uint32(1)
	h64 := xxhash.Sum64String(key)
	h10 := Hash10(key)
	entry, idx, _ := ki.rb.GetEntry()
	encode(length, memId, offset, lastAccess, freq, exptime, h64, h10, entry)
	ki.rm.AddV2(key, uint32(idx), h64, h10)
}

func (ki *KeyIndex) IncrLastAccessAndFreq(key string) {
	idx, found := ki.rm.Get(key)
	if !found {
		return
	}
	entry, ok := ki.rb.Get(int(idx))
	if !ok {
		return
	}
	d1 := ByteOrder.Uint64(entry[16:24])
	freq := uint32(d1) & 0xFFFFFF
	lastAccess := uint32(time.Now().Unix()-ki.startAt) / 60
	freq, _ = ki.mc.Inc(freq)
	d1 = uint64(lastAccess)<<24 | uint64(freq)&0xFFFFFF
	ByteOrder.PutUint64(entry[16:24], d1)
}

func (ki *KeyIndex) GenerateLastAccess() uint32 {
	return uint32(time.Now().Unix()-ki.startAt) / 60
}

func (ki *KeyIndex) GetMeta(key string) (uint32, uint16, uint32, uint32, uint64, uint64, bool) {
	h64 := xxhash.Sum64String(key)
	h10 := Hash10(key)
	idx, found := ki.rm.GetV2(h64, h10)
	if !found {
		return 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	entry, ok := ki.rb.Get(int(idx))
	if !ok {
		return 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	length, memId, offset, lastAccessAt, freq, exptime, gotH64, gotH10 := extract(entry)
	if gotH64 != h64 || gotH10 != h10 {
		return 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	lastAccess := ki.GenerateLastAccess()
	freq, _ = ki.mc.Inc(freq)
	d1 := uint64(length&LENGTH_MASK)<<48 | uint64(lastAccess&LAST_ACCESS_MASK)<<24 | uint64(freq&FREQ_MASK)
	ByteOrder.PutUint64(entry[16:24], d1)
	return memId, length, offset, lastAccessAt, ki.mc.Value(freq), exptime, true
}
func (ki *KeyIndex) GetMetaV2(key string) (uint32, uint16, uint32, uint32, uint64, uint64, uint32, bool) {
	h64 := xxhash.Sum64String(key)
	h10 := Hash10(key)
	idx, found := ki.rm.GetV2(h64, h10)
	if !found {
		return 0, 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	entry, ok := ki.rb.Get(int(idx))
	if !ok {
		return 0, 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	length, memId, offset, lastAccessAt, freq, exptime, gotH64, gotH10 := extract(entry)
	if gotH64 != h64 || gotH10 != h10 {
		return 0, 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	lastAccess := ki.GenerateLastAccess()
	freq, _ = ki.mc.Inc(freq)
	d1 := uint64(length&LENGTH_MASK)<<48 | uint64(lastAccess&LAST_ACCESS_MASK)<<24 | uint64(freq&FREQ_MASK)
	ByteOrder.PutUint64(entry[16:24], d1)
	lastAccess = (uint32(time.Now().Unix()-ki.startAt) / 60) - lastAccessAt
	return memId, length, offset, lastAccess, ki.mc.Value(freq), exptime, uint32(idx), true
}

func (ki *KeyIndex) Get(key string) (uint16, uint32, uint32, uint32, uint32, uint64, bool) {
	h64 := xxhash.Sum64String(key)
	h10 := Hash10(key)
	idx, found := ki.rm.GetV2(h64, h10)
	if !found {
		return 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	entry, ok := ki.rb.Get(int(idx))
	if !ok {
		return 0, 0, 0, 0, 0, 0, false // TODO: return error
	}
	length, memId, offset, lastAccess, freq, exptime, _, _ := extract(entry)

	return length, memId, offset, lastAccess, freq, exptime, true
}
func (ki *KeyIndex) GetRB() *RingBuffer {
	return ki.rb
}

func (ki *KeyIndex) PeekMemIdAtHead() uint32 {
	return ByteOrder.Uint32(ki.rb.buf[ki.rb.head][8:12])
}

func extract(entry *Entry) (uint16, uint32, uint32, uint32, uint32, uint64, uint64, uint64) {
	h64 := ByteOrder.Uint64(entry[:8])
	memId := ByteOrder.Uint32(entry[8:12])
	offset := ByteOrder.Uint32(entry[12:16])
	d1 := ByteOrder.Uint64(entry[16:24])
	d2 := ByteOrder.Uint64(entry[24:32])

	length := uint64(d1>>48) & LENGTH_MASK
	lastAccess := uint64(d1>>24) & LAST_ACCESS_MASK
	freq := uint64(d1) & FREQ_MASK

	h10 := uint64(d2>>54) & H10_MASK
	exptime := uint64(d2) & EXPTIME_MASK
	return uint16(length), memId, offset, uint32(lastAccess), uint32(freq), exptime, h64, h10
}

func encode(length uint16, memId, offset, lastAccess, freq uint32, exptime uint64, h64, h10 uint64, entry *Entry) {
	d1 := uint64(length&LENGTH_MASK)<<48 | uint64(lastAccess&LAST_ACCESS_MASK)<<24 | uint64(freq&FREQ_MASK)
	d2 := uint64(h10&H10_MASK)<<54 | uint64(exptime&EXPTIME_MASK)
	ByteOrder.PutUint64(entry[:8], h64)
	ByteOrder.PutUint32(entry[8:12], memId)
	ByteOrder.PutUint32(entry[12:16], offset)
	ByteOrder.PutUint64(entry[16:24], d1)
	ByteOrder.PutUint64(entry[24:32], d2)
}

func extractHash(entry *Entry) (h64, h10 uint64) {
	h64 = ByteOrder.Uint64(entry[:8])
	d2 := ByteOrder.Uint64(entry[24:32])
	return h64, d2 >> 54
}

func extractMemId(entry *Entry) (memId uint32) {
	return ByteOrder.Uint32(entry[8:12])
}

func extractOffset(entry *Entry) (offset uint32) {
	return ByteOrder.Uint32(entry[12:16])
}

func (ki *KeyIndex) StartTrim() {
	ki.deleteInProgress = true
}

func (ki *KeyIndex) TrimStatus() bool {
	return ki.deleteInProgress
}

func (ki *KeyIndex) Delete(nKeys int) (uint32, int) {
	for i := 0; i < nKeys; i++ {
		deleted, next := ki.rb.Delete()
		if deleted == nil {
			return 0, -1
		}
		ki.rm.RemoveV2(extractHash(deleted))
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
func (ki *KeyIndex) delete() (*Entry, *Entry, bool) {
	deleted, next := ki.rb.Delete()
	return deleted, next, true
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
