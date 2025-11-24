package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIntToBytes(t *testing.T) {
	i1 := int(2)
	bI1, _ := IntToBytes(i1)
	i2 := int8(8)
	bI2, _ := IntToBytes(i2)
	i3 := int16(392)
	bI3, _ := IntToBytes(i3)
	i4 := int32(109_353)
	bI4, _ := IntToBytes(i4)
	i5 := int64(2)
	bI5, _ := IntToBytes(i5)

	b, err := IntToBytes(i1)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI1, b, "byte representation didn't match for int")

	b, err = IntToBytes(i2)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI2, b, "byte representation didn't match for int8")

	b, err = IntToBytes(i3)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI3, b, "byte representation didn't match for int16")

	b, err = IntToBytes(i4)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI4, b, "byte representation didn't match for int32")

	b, err = IntToBytes(i5)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI5, b, "byte representation didn't match for int64")

}

func TestIntToBytes_Unsigned(t *testing.T) {
	i1 := uint(2)
	bI1, _ := IntToBytes(i1)
	i2 := uint8(8)
	bI2, _ := IntToBytes(i2)
	i3 := uint16(392)
	bI3, _ := IntToBytes(i3)
	i4 := uint32(109_353)
	bI4, _ := IntToBytes(i4)
	i5 := uint64(2)
	bI5, _ := IntToBytes(i5)

	b, err := IntToBytes(i1)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI1, b, "byte representation didn't match for uint")

	b, err = IntToBytes(i2)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI2, b, "byte representation didn't match for uint8")

	b, err = IntToBytes(i3)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI3, b, "byte representation didn't match for uint16")

	b, err = IntToBytes(i4)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI4, b, "byte representation didn't match for uint32")

	b, err = IntToBytes(i5)
	assert.Nilf(t, err, "No error expected")
	assert.Equalf(t, bI5, b, "byte representation didn't match for uint64")

}

func TestBytesToInt(t *testing.T) {
	b, err := IntToBytes(234_599)
	assert.Nilf(t, err, "error must be nil")
	var i int
	err2 := BytesToInt(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, 234_599, i, "byte must translate to inbound integer")
}

func TestBytesToInt8(t *testing.T) {
	b, err := IntToBytes(int8(25))
	assert.Nilf(t, err, "error must be nil")
	var i int8
	err2 := BytesToInt8(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, int8(25), i, "byte must translate to inbound integer")
}

func TestBytesToInt16(t *testing.T) {
	b, err := IntToBytes(int16(392))
	assert.Nilf(t, err, "error must be nil")
	var i int16
	err2 := BytesToInt16(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, int16(392), i, "byte must translate to inbound integer")
}

func TestBytesToInt32(t *testing.T) {
	b, err := IntToBytes(int32(109_353))
	assert.Nilf(t, err, "error must be nil")
	var i int32
	err2 := BytesToInt32(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, int32(109_353), i, "byte must translate to inbound integer")
}

func TestBytesToInt64(t *testing.T) {
	b, err := IntToBytes(int64(234_599))
	assert.Nilf(t, err, "error must be nil")
	var i int64
	err2 := BytesToInt64(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, int64(234_599), i, "byte must translate to inbound integer")
}

func TestBytesToUint8(t *testing.T) {
	b, err := IntToBytes(uint8(25))
	assert.Nilf(t, err, "error must be nil")
	var i uint8
	err2 := BytesToUint8(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, uint8(25), i, "byte must translate to inbound integer")
}

func TestBytesToUint16(t *testing.T) {
	b, err := IntToBytes(uint16(392))
	assert.Nilf(t, err, "error must be nil")
	var i uint16
	err2 := BytesToUint16(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, uint16(392), i, "byte must translate to inbound integer")
}

func TestBytesToUint32(t *testing.T) {
	b, err := IntToBytes(uint32(109_353))
	assert.Nilf(t, err, "error must be nil")
	var i uint32
	err2 := BytesToUint32(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, uint32(109_353), i, "byte must translate to inbound integer")
}

func TestBytesToUint64(t *testing.T) {
	b, err := IntToBytes(uint64(234_599))
	assert.Nilf(t, err, "error must be nil")
	var i uint64
	err2 := BytesToUint64(b, &i)
	assert.Nilf(t, err2, "error must be nil")
	assert.Equalf(t, uint64(234_599), i, "byte must translate to inbound integer")
}
