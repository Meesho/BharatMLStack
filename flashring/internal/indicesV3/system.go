package indicesv2

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
