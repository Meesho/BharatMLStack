package system

import (
	"encoding/binary"
	"unsafe"
)

var (
	ByteOrder binary.ByteOrder
)

func init() {
	loadByteOrder()
}

func loadByteOrder() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		ByteOrder = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		ByteOrder = binary.BigEndian
	default:
		panic("Could not determine endianness.")
	}
}
