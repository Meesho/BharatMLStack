package indices

import (
	"encoding/binary"
	"time"
	"unsafe"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/maths"
	"github.com/cespare/xxhash/v2"
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
	c.PutUint32(b, v)
}

func (c *CustomByteOrder) Uint32(b []byte) uint32 {
	return c.Uint32(b)
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
	d2 := uint64(h10)<<54 | uint64(exptime)&0xFFFFFFFFFFFF
	entry := Entry{}
	ByteOrder.PutUint64(entry[:8], h64)
	ByteOrder.PutUint32(entry[8:12], memId)
	ByteOrder.PutUint32(entry[12:16], offset)
	ByteOrder.PutUint64(entry[16:24], d1)
	ByteOrder.PutUint64(entry[24:32], d2)
	idx := ki.rb.Add(&entry)
	ki.rm.AddV2(key, uint32(idx), h64, h10)
	if ki.deleteInProgress {
		i := 0
		for i < ki.deleteAmortizedStep {
			shouldDeleteNext := ki.delete()
			if !shouldDeleteNext {
				ki.deleteInProgress = false
				break
			}
			i++
		}
	}
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
	length, memId, offset, lastAccess, freq, exptime := extract(entry)
	return length, memId, offset, lastAccess, freq, exptime, true
}

func extract(entry *Entry) (length uint16, memId, offset, lastAccess, freq uint32, exptime uint64) {
	d2 := ByteOrder.Uint64(entry[24:32])
	d1 := ByteOrder.Uint64(entry[16:24])
	freq = uint32(d1) & 0xFFFFFF
	lastAccess = uint32(d1>>24) & 0xFFFFFF
	length = uint16(d1>>48) & 0xFFFF
	memId = ByteOrder.Uint32(entry[8:12])
	offset = ByteOrder.Uint32(entry[12:16])
	exptime = uint64(d2) & 0xFFFFFFFFFFFF
	return length, memId, offset, lastAccess, freq, exptime
}

func (ki *KeyIndex) StartTrim() {
	ki.deleteInProgress = true
}

func (ki *KeyIndex) TrimStatus() bool {
	return ki.deleteInProgress
}

func (ki *KeyIndex) delete() bool {
	deleted, next, ok := ki.rb.Delete()
	if !ok {
		return false
	}
	_, delMemId, _, _, _, _ := extract(deleted)
	_, nextMemId, _, _, _, _ := extract(next)
	return delMemId == nextMemId
}
