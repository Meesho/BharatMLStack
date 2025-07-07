package index

import (
	"encoding/binary"
	"unsafe"
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

type Index struct {
	idx map[string][]byte
}

func NewIndex() *Index {
	if ByteOrder == nil {
		loadByteOrder()
	}
	return &Index{
		idx: make(map[string][]byte),
	}
}

func (i *Index) Put(key string, offset, size int64, fileOffset int64, memtableId int64) {
	b := make([]byte, 16)
	ByteOrder.PutInt64(b[0:8], offset)
	ByteOrder.PutInt64(b[8:16], size)
	ByteOrder.PutInt64(b[16:24], fileOffset)
	ByteOrder.PutInt64(b[24:32], memtableId)
	i.idx[key] = b
}

func (i *Index) Get(key string) (int64, int64, int64, int64, bool) {
	value, ok := i.idx[key]
	if !ok {
		return 0, 0, 0, 0, false
	}
	return ByteOrder.Int64(value[0:8]), ByteOrder.Int64(value[8:16]), ByteOrder.Int64(value[16:24]), ByteOrder.Int64(value[24:32]), true
}

func (i *Index) Delete(key string) {
	delete(i.idx, key)
}
