package config

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"

	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
	"github.com/Meesho/BharatMLStack/horizon/pkg/float8"
	"github.com/x448/float16"
)

var ByteOrder *CustomByteOrder

type CustomByteOrder struct {
	binary.ByteOrder
}

func (c *CustomByteOrder) PutUint8FromUint32(b []byte, v uint32) {
	_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
	b[0] = uint8(v)
}

func (c *CustomByteOrder) PutUint16FromUint32(b []byte, v uint32) {
	c.ByteOrder.PutUint16(b, uint16(v))
}

func (c *CustomByteOrder) Uint8(b []byte) uint8 {
	_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
	return b[0]
}

func (c *CustomByteOrder) Uint8AsUint32(b []byte) uint32 {
	_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
	return uint32(b[0])
}

func (c *CustomByteOrder) Uint16AsUint32(b []byte) uint32 {
	return uint32(c.ByteOrder.Uint16(b))
}

func (c *CustomByteOrder) PutInt8FromInt32(b []byte, v int32) {
	_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
	b[0] = uint8(v)
}

func (c *CustomByteOrder) PutInt16FromInt32(b []byte, v int32) {
	c.ByteOrder.PutUint16(b, uint16(v))
}

func (c *CustomByteOrder) PutInt32(b []byte, v int32) {
	c.PutUint32(b, uint32(v))
}

func (c *CustomByteOrder) PutInt64(b []byte, v int64) {
	c.PutUint64(b, uint64(v))
}

func (c *CustomByteOrder) Int8(b []byte) int8 {
	_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
	return int8(b[0])
}

func (c *CustomByteOrder) Int8AsInt32(b []byte) int32 {
	_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
	return int32(b[0])
}

func (c *CustomByteOrder) Int16(b []byte) int16 {
	return int16(c.ByteOrder.Uint16(b))
}

func (c *CustomByteOrder) Int16AsInt32(b []byte) int32 {
	return int32(int16(c.ByteOrder.Uint16(b)))
}

func (c *CustomByteOrder) Int32(b []byte) int32 {
	return int32(c.Uint32(b))
}

func (c *CustomByteOrder) Int64(b []byte) int64 {
	return int64(c.Uint64(b))
}

func (c *CustomByteOrder) PutFloat8E5M2FromFP32(b []byte, v float32) {
	b[0] = uint8(float8.FP8E5M2FromFP32Value(v))
}

func (c *CustomByteOrder) PutFloat8E4M3FromFP32(b []byte, v float32) {
	b[0] = uint8(float8.FP8E4M3FromFP32Value(v))
}

func (c *CustomByteOrder) PutFloat16FromFP32(b []byte, v float32) {
	fp16 := float16.Fromfloat32(v)
	c.ByteOrder.PutUint16(b, fp16.Bits())
}

func (c *CustomByteOrder) PutFloat32(b []byte, v float32) {
	c.PutUint32(b, math.Float32bits(v))
}

func (c *CustomByteOrder) PutFloat64(b []byte, v float64) {
	c.PutUint64(b, math.Float64bits(v))
}

func (c *CustomByteOrder) Float8E5M2AsFP32(b []byte) float32 {
	return float8.FP8E5M2ToFP32Value(float8.Float8e5m2(b[0]))
}

func (c *CustomByteOrder) Float8E4M3AsFP32(b []byte) float32 {
	return float8.FP8E4M3ToFP32Value(float8.Float8e4m3(b[0]))
}

func (c *CustomByteOrder) Float16AsFP32(b []byte) float32 {
	return float16.Frombits(c.ByteOrder.Uint16(b)).Float32()
}

func (c *CustomByteOrder) Float32(b []byte) float32 {
	return math.Float32frombits(c.Uint32(b))
}

func (c *CustomByteOrder) Float64(b []byte) float64 {
	return math.Float64frombits(c.Uint64(b))
}

func init() {
	loadByteOrder()
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

func GetToByteFP32AndLess(dType enums.DataType) (func([]byte, float32), error) {
	switch dType {
	case "DataTypeFP8E5M2", "DataTypeFP8E5M2Vector":
		return ByteOrder.PutFloat8E5M2FromFP32, nil
	case "DataTypeFP8E4M3", "DataTypeFP8E4M3Vector":
		return ByteOrder.PutFloat8E4M3FromFP32, nil
	case "DataTypeFP16", "DataTypeFP16Vector":
		return ByteOrder.PutFloat16FromFP32, nil
	case "DataTypeFP32", "DataTypeFP32Vector":
		return ByteOrder.PutFloat32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetToByteInt32AndLess(dType enums.DataType) (func([]byte, int32), error) {
	switch dType {
	case "DataTypeInt8", "DataTypeInt8Vector":
		return ByteOrder.PutInt8FromInt32, nil
	case "DataTypeInt16", "DataTypeInt16Vector":
		return ByteOrder.PutInt16FromInt32, nil
	case "DataTypeInt32", "DataTypeInt32Vector":
		return ByteOrder.PutInt32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetToByteUint32AndLess(dType enums.DataType) (func([]byte, uint32), error) {
	switch dType {
	case "DataTypeUint8", "DataTypeUint8Vector":
		return ByteOrder.PutUint8FromUint32, nil
	case "DataTypeUint16", "DataTypeUint16Vector":
		return ByteOrder.PutUint16FromUint32, nil
	case "DataTypeUint32", "DataTypeUint32Vector":
		return ByteOrder.PutUint32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}
