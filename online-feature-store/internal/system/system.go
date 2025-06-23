package system

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/float8"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/x448/float16"
)

var ByteOrder *CustomByteOrder

type CustomByteOrder struct {
	binary.ByteOrder
}

/**
 * Extensions for Uint8/16
 */
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

func (c *CustomByteOrder) Uint32Vector(b []byte) []uint32 {
	if len(b)%4 != 0 {
		panic("invalid byte slice length: must be a multiple of 4")
	}
	result := make([]uint32, len(b)/4)
	for i := 0; i < len(result); i++ {
		result[i] = c.Uint32(b[i*4 : i*4+4])
	}
	return result
}

func (c *CustomByteOrder) Uint8AsUint32Vector(b []byte) []uint32 {
	result := make([]uint32, len(b))
	for i, byteValue := range b {
		result[i] = c.Uint8AsUint32([]byte{byteValue})
	}
	return result
}

func (c *CustomByteOrder) Uint16AsUint32Vector(b []byte) []uint32 {
	if len(b)%2 != 0 {
		panic("invalid byte slice length: must be a multiple of 2")
	}
	result := make([]uint32, len(b)/2)
	for i := 0; i < len(result); i++ {
		result[i] = c.Uint16AsUint32(b[i*2 : i*2+2])
	}
	return result
}

func (c *CustomByteOrder) Uint64Vector(b []byte) []uint64 {
	if len(b)%8 != 0 {
		panic("invalid byte slice length: must be a multiple of 8")
	}
	result := make([]uint64, len(b)/8)
	for i := 0; i < len(result); i++ {
		result[i] = c.Uint64(b[i*8 : i*8+8])
	}
	return result
}

/**
 * Extensions for Int8/16/32/64
 */
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
	return int32(int8(b[0]))
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

/**
 * Extensions for Float8/16
 */
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

func (c *CustomByteOrder) Float32Vector(b []byte) []float32 {
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

func (c *CustomByteOrder) FP8E5M2Vector(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}

	result := make([]float32, len(b)) // Each byte represents one FP8E5M2
	for i, byteValue := range b {
		result[i] = c.Float8E5M2AsFP32([]byte{byteValue}) // Use a custom decoding function
	}

	return result
}

func (c *CustomByteOrder) FP8E4M3Vector(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}

	result := make([]float32, len(b))
	for i, byteValue := range b {
		result[i] = c.Float8E4M3AsFP32([]byte{byteValue}) // Use a custom decoding function
	}

	return result
}

func (c *CustomByteOrder) FP16Vector(b []byte) []float32 {
	if len(b)%2 != 0 {
		panic("invalid byte slice length: must be a multiple of 2")
	}

	n := len(b) / 2 // Number of FP16 elements
	result := make([]float32, n)

	for i := 0; i < n; i++ {
		offset := i * 2
		result[i] = c.Float16AsFP32(b[offset : offset+2])
	}

	return result
}

func (c *CustomByteOrder) Float64Vector(b []byte) []float64 {
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

func (c *CustomByteOrder) Int8Vector(b []byte) []int8 {
	result := make([]int8, len(b))
	for i, byteValue := range b {
		result[i] = c.Int8([]byte{byteValue})
	}
	return result
}

func (c *CustomByteOrder) Uint8Vector(b []byte) []uint8 {
	result := make([]uint8, len(b))
	for i, byteValue := range b {
		result[i] = c.Uint8([]byte{byteValue})
	}
	return result
}

func (c *CustomByteOrder) Int8AsInt32Vector(b []byte) []int32 {
	result := make([]int32, len(b))
	for i, byteValue := range b {
		result[i] = c.Int8AsInt32([]byte{byteValue})
	}
	return result
}

func (c *CustomByteOrder) Int16Vector(b []byte) []int16 {
	if len(b)%2 != 0 {
		panic("invalid byte slice length: must be a multiple of 2")
	}
	result := make([]int16, len(b)/2)
	for i := 0; i < len(result); i++ {
		result[i] = c.Int16(b[i*2 : i*2+2])
	}
	return result
}

func (c *CustomByteOrder) Int16AsInt32Vector(b []byte) []int32 {
	if len(b)%2 != 0 {
		panic("invalid byte slice length: must be a multiple of 2")
	}
	result := make([]int32, len(b)/2)
	for i := 0; i < len(result); i++ {
		result[i] = c.Int16AsInt32(b[i*2 : i*2+2])
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
	// Allocate a bool slice for the output
	result := make([]bool, vectorLength)

	// Iterate over each bit in the byte array
	for i := 0; i < vectorLength; i++ {
		// Determine which byte and bit within the byte to read
		byteIndex := i / 8
		bitIndex := i % 8

		// Extract the bit: shift the byte and mask it
		bit := (encodedValue[byteIndex] >> bitIndex) & 1

		// Convert the bit to a bool and store it
		result[i] = bit == 1
	}

	return result
}

func (c *CustomByteOrder) StringVector(encodedValue []byte, vectorLength int, stringLength int) []string {
	// Allocate a slice for the output strings
	result := make([]string, vectorLength)

	// Iterate over the vector length to decode each string
	for i := 0; i < vectorLength; i++ {
		// Calculate the start and end of the current string slice
		start := i * stringLength
		end := start + stringLength

		// Ensure we don't go out of bounds
		if end > len(encodedValue) {
			break
		}

		// Extract the string slice and decode it using the String method
		decodedString := c.String(encodedValue[start:end])
		result[i] = decodedString
	}

	return result
}

func (c *CustomByteOrder) Float64(b []byte) float64 {
	return math.Float64frombits(c.Uint64(b))
}

const (
	ByteLow5bitsMask  uint8  = 0x1F
	ByteLow3bitsMask  uint8  = 0x07
	ByteHigh3bitsMask uint8  = 0xE0
	MaxUint8          uint8  = 0xFF
	MinUint8          uint8  = 0x00
	MaxUint16         uint16 = 0xFFFF
	MinUint16         uint16 = 0x0000
	MaxUint32         uint32 = 0xFFFFFFFF
	MinUint32         uint32 = 0x00000000
	MaxUint64         uint64 = 0xFFFFFFFFFFFFFFFF
	MinUint64         uint64 = 0x0000000000000000
)

func Init() {
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

func GetToByteFP32AndLess(dType types.DataType) (func([]byte, float32), error) {
	switch dType {
	case types.DataTypeFP8E5M2, types.DataTypeFP8E5M2Vector:
		return ByteOrder.PutFloat8E5M2FromFP32, nil
	case types.DataTypeFP8E4M3, types.DataTypeFP8E4M3Vector:
		return ByteOrder.PutFloat8E4M3FromFP32, nil
	case types.DataTypeFP16, types.DataTypeFP16Vector:
		return ByteOrder.PutFloat16FromFP32, nil
	case types.DataTypeFP32, types.DataTypeFP32Vector:
		return ByteOrder.PutFloat32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetFromByteFP32AndLess(dType types.DataType) (func([]byte) float32, error) {
	switch dType {
	case types.DataTypeFP8E5M2, types.DataTypeFP8E5M2Vector:
		return ByteOrder.Float8E5M2AsFP32, nil
	case types.DataTypeFP8E4M3, types.DataTypeFP8E4M3Vector:
		return ByteOrder.Float8E4M3AsFP32, nil
	case types.DataTypeFP16, types.DataTypeFP16Vector:
		return ByteOrder.Float16AsFP32, nil
	case types.DataTypeFP32, types.DataTypeFP32Vector:
		return ByteOrder.Float32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetToByteInt32AndLess(dType types.DataType) (func([]byte, int32), error) {
	switch dType {
	case types.DataTypeInt8, types.DataTypeInt8Vector:
		return ByteOrder.PutInt8FromInt32, nil
	case types.DataTypeInt16, types.DataTypeInt16Vector:
		return ByteOrder.PutInt16FromInt32, nil
	case types.DataTypeInt32, types.DataTypeInt32Vector:
		return ByteOrder.PutInt32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetFromByteInt32AndLess(dType types.DataType) (func([]byte) int32, error) {
	switch dType {
	case types.DataTypeInt8, types.DataTypeInt8Vector:
		return ByteOrder.Int8AsInt32, nil
	case types.DataTypeInt16, types.DataTypeInt16Vector:
		return ByteOrder.Int16AsInt32, nil
	case types.DataTypeInt32, types.DataTypeInt32Vector:
		return ByteOrder.Int32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetToByteUint32AndLess(dType types.DataType) (func([]byte, uint32), error) {
	switch dType {
	case types.DataTypeUint8, types.DataTypeUint8Vector:
		return ByteOrder.PutUint8FromUint32, nil
	case types.DataTypeUint16, types.DataTypeUint16Vector:
		return ByteOrder.PutUint16FromUint32, nil
	case types.DataTypeUint32, types.DataTypeUint32Vector:
		return ByteOrder.PutUint32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func GetFromByteUint32AndLess(dType types.DataType) (func([]byte) uint32, error) {
	switch dType {
	case types.DataTypeUint8, types.DataTypeUint8Vector:
		return ByteOrder.Uint8AsUint32, nil
	case types.DataTypeUint16, types.DataTypeUint16Vector:
		return ByteOrder.Uint16AsUint32, nil
	case types.DataTypeUint32, types.DataTypeUint32Vector:
		return ByteOrder.Uint32, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dType)
	}
}

func BinPackUint32InUint64(high, low uint32) uint64 {
	return uint64(high)<<32 | uint64(low)
}

func UnpackUint64InUint32(highLow uint64) (uint32, uint32) {
	return uint32(highLow >> 32), uint32(highLow)
}

func BinPackUint16InUint32(high, low uint16) uint32 {
	return uint32(high)<<16 | uint32(low)
}

func UnpackUint32InUint16(highLow uint32) (uint16, uint16) {
	return uint16(highLow >> 16), uint16(highLow)
}

func BinPackUint8InUint16(high, low uint8) uint16 {
	return uint16(high)<<8 | uint16(low)
}

func UnpackUint16InUint8(highLow uint16) (uint8, uint8) {
	return uint8(highLow >> 8), uint8(highLow)
}

func ParseFeatureValue(featureLabels []string, features *persist.FeatureValues, dataType types.DataType, featureMeta map[string]config.FeatureMeta) (interface{}, error) {
	switch dataType {
	case types.DataTypeInt8, types.DataTypeInt16, types.DataTypeInt32:
		return GetInt32(featureLabels, features, featureMeta)
	case types.DataTypeUint8, types.DataTypeUint16, types.DataTypeUint32:
		return GetUInt32(featureLabels, features, featureMeta)
	case types.DataTypeInt64:
		return GetInt64(featureLabels, features, featureMeta)
	case types.DataTypeUint64:
		return GetUInt64(featureLabels, features, featureMeta)
	case types.DataTypeFP8E5M2, types.DataTypeFP8E4M3, types.DataTypeFP16, types.DataTypeFP32:
		return GetFP32(featureLabels, features, featureMeta)
	case types.DataTypeFP64:
		return GetFP64(featureLabels, features, featureMeta)
	case types.DataTypeBool:
		return GetUInt8(featureLabels, features, featureMeta)
	case types.DataTypeString:
		return GetString(featureLabels, features, featureMeta)
	case types.DataTypeInt8Vector, types.DataTypeInt16Vector, types.DataTypeInt32Vector:
		return GetInt32Vector(featureLabels, features, featureMeta)
	case types.DataTypeInt64Vector:
		return GetInt64Vector(featureLabels, features, featureMeta)
	case types.DataTypeUint8Vector, types.DataTypeUint16Vector, types.DataTypeUint32Vector:
		return GetUInt32Vector(featureLabels, features, featureMeta)
	case types.DataTypeUint64Vector:
		return GetUInt64Vector(featureLabels, features, featureMeta)
	case types.DataTypeFP8E5M2Vector, types.DataTypeFP8E4M3Vector, types.DataTypeFP16Vector, types.DataTypeFP32Vector:
		return GetFP32Vector(featureLabels, features, featureMeta)
	case types.DataTypeFP64Vector:
		return GetFP64Vector(featureLabels, features, featureMeta)
	case types.DataTypeBoolVector:
		return GetBoolVector(featureLabels, features, featureMeta)
	case types.DataTypeStringVector:
		return GetStringVector(featureLabels, features, featureMeta)
	default:
		return nil, fmt.Errorf("unknown Data type: %d", dataType)
	}
}

func GetInt32(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]int32, error) {
	if featureValues.GetValues().Int32Values == nil {
		return nil, fmt.Errorf("int32_values is nil")
	}
	if len(featureValues.GetValues().Int32Values) != len(featureLabels) {
		return nil, fmt.Errorf("int32_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Int32Values))
	}
	int32Array := make([]int32, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		int32Array[featureMeta[label].Sequence] = featureValues.GetValues().Int32Values[index]
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			int32Array[meta.Sequence] = ByteOrder.Int32(meta.DefaultValuesInBytes)
		}
	}
	return int32Array, nil
}

func GetUInt32(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]uint32, error) {
	if featureValues.GetValues().Uint32Values == nil {
		return nil, fmt.Errorf("uint32_values is nil")
	}
	if len(featureValues.GetValues().Uint32Values) != len(featureLabels) {
		return nil, fmt.Errorf("uint32_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Uint32Values))
	}
	uint32Array := make([]uint32, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		uint32Array[featureMeta[label].Sequence] = uint32(featureValues.GetValues().Uint32Values[index])
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			uint32Array[meta.Sequence] = ByteOrder.Uint32(meta.DefaultValuesInBytes)
		}
	}
	return uint32Array, nil
}

func GetInt64(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]int64, error) {
	if featureValues.GetValues().Int64Values == nil {
		return nil, fmt.Errorf("int64_values is nil")
	}
	if len(featureValues.GetValues().Int64Values) != len(featureLabels) {
		return nil, fmt.Errorf("int64_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Int64Values))
	}
	int64Array := make([]int64, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		int64Array[featureMeta[label].Sequence] = int64(featureValues.GetValues().Int64Values[index])
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			int64Array[meta.Sequence] = ByteOrder.Int64(meta.DefaultValuesInBytes)
		}
	}

	return int64Array, nil
}

func GetUInt64(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]uint64, error) {
	if featureValues.GetValues().Uint64Values == nil {
		return nil, fmt.Errorf("uint64_values is nil")
	}
	if len(featureValues.GetValues().Uint64Values) != len(featureLabels) {
		return nil, fmt.Errorf("uint64_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Uint64Values))
	}
	uint64Array := make([]uint64, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		uint64Array[featureMeta[label].Sequence] = featureValues.GetValues().Uint64Values[index]
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			uint64Array[meta.Sequence] = ByteOrder.Uint64(meta.DefaultValuesInBytes)
		}
	}
	return uint64Array, nil
}

func GetFP32(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]float32, error) {
	if featureValues.GetValues().Fp32Values == nil {
		return nil, fmt.Errorf("fp32_values is nil")
	}
	if len(featureValues.GetValues().Fp32Values) != len(featureLabels) {
		return nil, fmt.Errorf("fp32_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Fp32Values))
	}
	fp32Array := make([]float32, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		fp32Array[featureMeta[label].Sequence] = float32(featureValues.GetValues().Fp32Values[index])
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			fp32Array[meta.Sequence] = ByteOrder.Float32(meta.DefaultValuesInBytes)
		}
	}
	return fp32Array, nil
}

func GetFP64(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]float64, error) {
	if featureValues.GetValues().Fp64Values == nil {
		return nil, fmt.Errorf("fp64_values is nil")
	}
	if len(featureValues.GetValues().Fp64Values) != len(featureLabels) {
		return nil, fmt.Errorf("fp64_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Fp64Values))
	}
	fp64Array := make([]float64, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		fp64Array[featureMeta[label].Sequence] = featureValues.GetValues().Fp64Values[index]
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			fp64Array[meta.Sequence] = ByteOrder.Float64(meta.DefaultValuesInBytes)
		}
	}
	return fp64Array, nil
}

func GetUInt8(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]uint8, error) {
	if featureValues.GetValues().BoolValues == nil {
		return nil, fmt.Errorf("bool_values is nil")
	}
	if len(featureValues.GetValues().BoolValues) != len(featureLabels) {
		return nil, fmt.Errorf("bool_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().BoolValues))
	}
	uint8Array := make([]uint8, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		uint8Array[featureMeta[label].Sequence] = func(b bool) uint8 {
			if b {
				return 1
			}
			return 0
		}(featureValues.GetValues().BoolValues[index])
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			uint8Array[meta.Sequence] = ByteOrder.Uint8(meta.DefaultValuesInBytes)
		}
	}
	return uint8Array, nil
}

func GetString(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([]string, error) {
	if featureValues.GetValues().StringValues == nil {
		return nil, fmt.Errorf("string_values is nil")
	}
	if len(featureValues.GetValues().StringValues) != len(featureLabels) {
		return nil, fmt.Errorf("string_values length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().StringValues))
	}
	stringArray := make([]string, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		stringArray[featureMeta[label].Sequence] = featureValues.GetValues().StringValues[index]
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			stringArray[meta.Sequence] = ByteOrder.String(meta.DefaultValuesInBytes)
		}
	}
	return stringArray, nil
}

func GetInt32Vector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]int32, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	int32Vectors := make([][]int32, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		int32Vectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.Int32Values
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			int32Vectors[meta.Sequence] = ByteOrder.Int32Vector(meta.DefaultValuesInBytes)
		}
	}
	return int32Vectors, nil
}

func GetInt64Vector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]int64, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	int64Vectors := make([][]int64, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		int64Vectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.Int64Values
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			int64Vectors[meta.Sequence] = ByteOrder.Int64Vector(meta.DefaultValuesInBytes)
		}
	}
	return int64Vectors, nil
}

func GetUInt32Vector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]uint32, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	uint32Vectors := make([][]uint32, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		uint32Vectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.Uint32Values
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			uint32Vectors[meta.Sequence] = ByteOrder.Uint32Vector(meta.DefaultValuesInBytes)
		}
	}
	return uint32Vectors, nil
}

func GetUInt64Vector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]uint64, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	uint64Vectors := make([][]uint64, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		uint64Vectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.Uint64Values
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			uint64Vectors[meta.Sequence] = ByteOrder.Uint64Vector(meta.DefaultValuesInBytes)
		}
	}
	return uint64Vectors, nil
}

func GetFP32Vector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]float32, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	fp32Vectors := make([][]float32, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		fp32FeatureValues := []float32{}
		for _, value := range featureValues.GetValues().Vector[index].Values.Fp32Values {
			fp32FeatureValues = append(fp32FeatureValues, float32(value))
		}
		fp32Vectors[featureMeta[label].Sequence] = fp32FeatureValues
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			fp32Vectors[meta.Sequence] = ByteOrder.FP16Vector(meta.DefaultValuesInBytes)
		}
	}
	return fp32Vectors, nil
}

func GetFP64Vector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]float64, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	fp64Vectors := make([][]float64, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		fp64Vectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.Fp64Values
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			fp64Vectors[meta.Sequence] = ByteOrder.Float64Vector(meta.DefaultValuesInBytes)
		}
	}
	return fp64Vectors, nil
}

func GetBoolVector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]bool, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	boolVectors := make([][]bool, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		boolVectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.BoolValues
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			boolVectors[meta.Sequence] = ByteOrder.BoolVector(meta.DefaultValuesInBytes, int(meta.VectorLength))
		}
	}
	return boolVectors, nil
}

func GetStringVector(featureLabels []string, featureValues *persist.FeatureValues, featureMeta map[string]config.FeatureMeta) ([][]string, error) {
	if featureValues.GetValues().Vector == nil {
		return nil, fmt.Errorf("vector is nil")
	}
	if len(featureValues.GetValues().Vector) != len(featureLabels) {
		return nil, fmt.Errorf("vector length mismatch with feature labels, expected %d, received %d", len(featureLabels), len(featureValues.GetValues().Vector))
	}
	stringVectors := make([][]string, len(featureMeta))
	labelExists := make(map[string]bool, len(featureLabels))
	for index, label := range featureLabels {
		labelExists[label] = true
		stringVectors[featureMeta[label].Sequence] = featureValues.GetValues().Vector[index].Values.StringValues
	}

	for label, meta := range featureMeta {
		if !labelExists[label] {
			stringVectors[meta.Sequence] = ByteOrder.StringVector(meta.DefaultValuesInBytes, int(meta.VectorLength), int(meta.StringLength))
		}
	}
	return stringVectors, nil
}
