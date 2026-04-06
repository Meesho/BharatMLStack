package utils

import (
	"encoding/binary"
	"math"
	"time"
)

var ByteOrder *CustomByteOrder

func init() {
	ByteOrder = &CustomByteOrder{ByteOrder: binary.LittleEndian}
}

type CustomByteOrder struct {
	binary.ByteOrder
}

func (c *CustomByteOrder) PutInt32(b []byte, v int32) {
	c.PutUint32(b, uint32(v))
}

func (c *CustomByteOrder) PutInt64(b []byte, v int64) {
	c.PutUint64(b, uint64(v))
}

func (c *CustomByteOrder) Int32(b []byte) int32 {
	return int32(c.Uint32(b))
}

func (c *CustomByteOrder) Int64(b []byte) int64 {
	return int64(c.Uint64(b))
}

func (c *CustomByteOrder) String(b []byte) string {
	var decodedString string
	// Iterate over each byte
	for _, b := range b {
		// Ignore padding (zeroes)
		if b != 0 {
			// Convert the byte to a character and append to the decoded string
			decodedString += string(b)
		}
	}
	return decodedString
}

func (c *CustomByteOrder) Bool(b []byte) bool {
	if len(b) < 1 {
		return false
	}
	return b[0] != 0
}

func (c *CustomByteOrder) PutFloat32(b []byte, v float32) {
	c.PutUint32(b, math.Float32bits(v))
}

func (c *CustomByteOrder) PutFloat64(b []byte, v float64) {
	c.PutUint64(b, math.Float64bits(v))
}

func (c *CustomByteOrder) Float32(b []byte) float32 {
	return math.Float32frombits(c.Uint32(b))
}

func (c *CustomByteOrder) Float64(b []byte) float64 {
	return math.Float64frombits(c.Uint64(b))
}

func (c *CustomByteOrder) FP32Vector(b []byte) []float32 {
	if len(b)%4 != 0 {
		panic("invalid byte slice length: must be a multiple of 4")
	}
	n := len(b) / 4
	result := make([]float32, n)
	for i := 0; i < n; i++ {
		offset := i * 4
		result[i] = math.Float32frombits(c.Uint32(b[offset : offset+4]))
	}
	return result
}

func (c *CustomByteOrder) FP64Vector(b []byte) []float64 {
	if len(b)%8 != 0 {
		panic("invalid byte slice length: must be a multiple of 8")
	}
	n := len(b) / 8
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		offset := i * 8
		result[i] = math.Float64frombits(c.Uint64(b[offset : offset+8]))
	}
	return result
}

func (c *CustomByteOrder) Int32Vector(b []byte) []int32 {
	if len(b)%4 != 0 {
		panic("invalid byte slice length: must be a multiple of 4")
	}
	result := make([]int32, len(b)/4)
	for i := 0; i < len(result); i++ {
		result[i] = c.Int32(b[i*4 : i*4+4])
	}
	return result
}

func (c *CustomByteOrder) Int64Vector(b []byte) []int64 {
	if len(b)%8 != 0 {
		panic("invalid byte slice length: must be a multiple of 8")
	}
	result := make([]int64, len(b)/8)
	for i := 0; i < len(result); i++ {
		result[i] = c.Int64(b[i*8 : i*8+8])
	}
	return result
}

func (c *CustomByteOrder) BoolVector(encodedValue []byte, vectorLength int) []bool {
	requiredBytes := (vectorLength + 7) / 8
	if len(encodedValue) < requiredBytes {
		return make([]bool, vectorLength)
	}
	result := make([]bool, vectorLength)
	for i := 0; i < vectorLength; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bit := (encodedValue[byteIndex] >> bitIndex) & 1
		result[i] = bit == 1
	}
	return result
}

func (c *CustomByteOrder) StringVector(encodedValue []byte, vectorLength int, stringLength int) []string {
	result := make([]string, vectorLength)
	for i := 0; i < vectorLength; i++ {
		start := i * stringLength
		end := start + stringLength
		if end > len(encodedValue) {
			break
		}
		decodedString := c.String(encodedValue[start:end])
		result[i] = decodedString
	}
	return result
}

func WeekFromTimestampMs(timestampMs int64) int {
	_, week := time.UnixMilli(timestampMs).ISOWeek()
	return week
}

func TimestampDiffInWeeks(timestampMs1 int64, timestampMs2 int64) int {
	diff := timestampMs1 - timestampMs2
	if diff < 0 {
		diff = -diff
	}
	return int(diff / (7 * 24 * 60 * 60 * 1000))
}
