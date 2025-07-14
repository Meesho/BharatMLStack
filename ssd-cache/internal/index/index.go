package index

import (
	"encoding/binary"
	"unsafe"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/pool"
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

type Index struct {
	idx          map[string][]byte
	bufferPool   *pool.LeakyPool
	memtableSize int32
}

func NewIndex() *Index {
	if ByteOrder == nil {
		loadByteOrder()
	}
	return &Index{
		idx: make(map[string][]byte),
	}
}

func NewIndexV2(bufferPool *pool.LeakyPool, memtableSize int32) *Index {
	if ByteOrder == nil {
		loadByteOrder()
	}
	return &Index{
		idx:          make(map[string][]byte),
		bufferPool:   bufferPool,
		memtableSize: memtableSize,
	}
}

func (i *Index) Put(key string, offset int32, size int32, memtableId int64) {
	buf, _ := i.bufferPool.Get()
	b := (buf).([]byte)
	ByteOrder.PutInt32(b[0:4], offset)
	ByteOrder.PutInt32(b[4:8], size)
	ByteOrder.PutInt64(b[8:16], memtableId)
	i.idx[key] = b
}

func (i *Index) Get(key string) (int32, int32, int64, int64, bool) {
	value, ok := i.idx[key]
	if !ok {
		return 0, 0, 0, 0, false
	}
	offset := ByteOrder.Int32(value[0:4])
	size := ByteOrder.Int32(value[4:8])
	memtableId := ByteOrder.Int64(value[8:16])
	return offset, size, memtableId*int64(i.memtableSize) + int64(offset), memtableId, true
}

func (i *Index) Delete(key string) {
	if buf, ok := i.idx[key]; ok {
		i.bufferPool.Put(buf)
		delete(i.idx, key)
	}

}
