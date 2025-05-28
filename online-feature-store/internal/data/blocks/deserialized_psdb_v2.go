package blocks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

type DeserializedPSDB struct {
	// 64-bit aligned fields
	ExpiryAt       uint64
	Header         []byte
	CompressedData []byte
	OriginalData   []byte

	// 16-bit field
	FeatureSchemaVersion uint16

	// 8-bit fields
	LayoutVersion   uint8
	CompressionType compression.Type
	DataType        types.DataType

	// 1-bit fields (packed into bytes by compiler)
	NegativeCache bool
	Expired       bool
}

func DeserializePSDB(data []byte) (*DeserializedPSDB, error) {
	if len(data) == 0 {
		return &DeserializedPSDB{
			NegativeCache: true,
		}, nil
	}
	layoutVersion, err := extractLayoutVersionFromHeader(data)
	if err != nil {
		return nil, err
	}
	var ddb *DeserializedPSDB

	switch layoutVersion {
	case 1:
		ddb, err = deserializePSDBForLayout1(data)
	default:
		err = fmt.Errorf("unsupported layout version: %d", layoutVersion)
	}
	if err == nil {
		ddb.LayoutVersion = layoutVersion
	}
	return ddb, err
}

func DeserializePSDBWithoutDecompression(data []byte) (*DeserializedPSDB, error) {
	if len(data) == 0 {
		return &DeserializedPSDB{
			NegativeCache: true,
		}, nil
	}
	layoutVersion, err := extractLayoutVersionFromHeader(data)
	if err != nil {
		return nil, err
	}
	var ddb *DeserializedPSDB

	switch layoutVersion {
	case 1:
		ddb, err = deserializePSDBForLayout1WithoutDecompression(data)
	default:
		err = fmt.Errorf("unsupported layout version: %d", layoutVersion)
	}
	if err == nil {
		ddb.LayoutVersion = layoutVersion
	}
	return ddb, err
}

func extractLayoutVersionFromHeader(data []byte) (uint8, error) {
	if len(data) < 7 {
		return 0, fmt.Errorf("header is too short to contain layout version")
	}
	return (data[layoutVersionIdx] & 0xF0) >> 4, nil
}

func deserializePSDBForLayout1(data []byte) (*DeserializedPSDB, error) {
	if len(data) < PSDBLayout1LengthBytes {
		return nil, fmt.Errorf("data is too short to contain a valid PSDBV2 header")
	}
	featureSchemaVersion := system.ByteOrder.Uint16(data[0:2])
	expiryAt, err := system.DecodeExpiry(data[2:7])
	isExpired := system.IsExpired(data[2:7])
	if err != nil {
		return nil, err
	}
	compressionType := compression.Type((data[7] & 0x0E) >> 1)
	dtT := (data[7] & 0x01) << 4
	dtT = dtT | ((data[8] & 0xF0) >> 4)
	dataType := types.DataType(dtT)
	header := data[0:PSDBLayout1LengthBytes]
	var originalData []byte
	var compressedData []byte
	if compressionType == compression.TypeNone {
		originalData = data[PSDBLayout1LengthBytes:]
		compressedData = data[PSDBLayout1LengthBytes:]
	} else {
		dec, err := compression.GetDecoder(compressionType)
		if err != nil {
			return nil, err
		}
		compressedData = data[PSDBLayout1LengthBytes:]
		originalData, err = dec.Decode(compressedData)
	}
	return &DeserializedPSDB{
		FeatureSchemaVersion: featureSchemaVersion,
		LayoutVersion:        1,
		ExpiryAt:             expiryAt,
		CompressionType:      compressionType,
		OriginalData:         originalData,
		DataType:             dataType,
		Header:               header,
		CompressedData:       compressedData,
		NegativeCache:        false,
		Expired:              isExpired,
	}, nil
}

func deserializePSDBForLayout1WithoutDecompression(data []byte) (*DeserializedPSDB, error) {
	if len(data) < PSDBLayout1LengthBytes {
		return nil, fmt.Errorf("data is too short to contain a valid PSDBV2 header")
	}
	featureSchemaVersion := system.ByteOrder.Uint16(data[0:2])
	expiryAt, err := system.DecodeExpiry(data[2:7])
	isExpired := system.IsExpired(data[2:7])
	if err != nil {
		return nil, err
	}
	compressionType := compression.Type((data[7] & 0x0E) >> 1)
	dtT := (data[7] & 0x01) << 4
	dtT = dtT | ((data[8] & 0xF0) >> 4)
	dataType := types.DataType(dtT)
	header := data[0:PSDBLayout1LengthBytes]
	var originalData []byte
	var compressedData []byte
	originalData = data[PSDBLayout1LengthBytes:]
	compressedData = data[PSDBLayout1LengthBytes:]

	return &DeserializedPSDB{
		FeatureSchemaVersion: featureSchemaVersion,
		LayoutVersion:        1,
		ExpiryAt:             expiryAt,
		CompressionType:      compressionType,
		OriginalData:         originalData,
		DataType:             dataType,
		Header:               header,
		CompressedData:       compressedData,
		NegativeCache:        false,
		Expired:              isExpired,
	}, nil
}

func NegativeCacheDeserializePSDB() *DeserializedPSDB {
	return &DeserializedPSDB{
		FeatureSchemaVersion: 0,
		LayoutVersion:        1,
		ExpiryAt:             0,
		CompressionType:      compression.TypeNone,
		DataType:             types.DataTypeUnknown,
		Header:               nil,
		CompressedData:       nil,
		OriginalData:         nil,
		NegativeCache:        true,
		Expired:              false,
	}
}

func (d *DeserializedPSDB) GetStringScalarFeature(pos int, noOfFeatures int) ([]byte, error) {
	if d.DataType != types.DataTypeString {
		return nil, fmt.Errorf("data type is not a string")
	}
	offset := 2 * noOfFeatures
	idx := 0
	var length uint16 = 0
	for pos >= 0 {
		offset += int(length)
		length = system.ByteOrder.Uint16(d.OriginalData[idx : idx+2])
		idx += 2
		pos--
	}
	if offset+int(length) > len(d.OriginalData) {
		return nil, fmt.Errorf("position out of bounds")
	}
	return d.OriginalData[offset : offset+int(length)], nil
}

// GetVectorStringFeature retrieves a specific vector's string data at position 'pos'
//
// Data Layout Example for 3 vectors with lengths [2,3,3]:
// Length Section: [v1-len1][v1-len2] [v2-len1][v2-len2][v2-len3] [v3-len1][v3-len2][v3-len3]
// Data Section:   [v1-str1][v1-str2] [v2-str1][v2-str2][v2-str3] [v3-str1][v3-str2][v3-str3]
//
// Algorithm:
// 1. Calculate offsets:
//   - offset: Counts total number of strings to find start of Data Section
//   - idx: Counts strings before target vector to find its length entries
//   - sum: Accumulates string lengths before target vector to find its data
//
// 2. For each vector before target position:
//   - Add its length to idx (to skip its length entries)
//   - Add up actual string lengths (from length entries) to sum
//
// 3. For target vector:
//   - Read each string length and corresponding data
//   - Combine length and data into result
//
// Example for pos=1 (second vector):
//
//	offset = 2+3+3 = 8 (total strings) * 2 (bytes per length) = start of Data Section
//	sum = len(v1-str1) + len(v1-str2) = offset to v2's string data
//  offset = 16 + sum (start of v2's string data)
//  idx = 2 (strings in v1) * 2 = position of v2's length entries

func (d *DeserializedPSDB) GetStringVectorFeature(pos int, noOfFeatures int, vectorLengths []uint16) ([]byte, error) {
	if d.DataType != types.DataTypeStringVector {
		return nil, fmt.Errorf("data type is not a string vector")
	}
	var offset int = 0
	var idx int = 0
	var sum int = 0
	k := 0
	for i := 0; i < len(vectorLengths); i++ {
		offset += int(vectorLengths[i])
		if i < pos {
			idx += int(vectorLengths[i])
			for j := 0; j < int(vectorLengths[i]); j++ {
				sum += int(system.ByteOrder.Uint16(d.OriginalData[k : k+2]))
				k += 2
			}
		}
	}
	offset *= 2
	offset += sum
	idx *= 2
	dim := vectorLengths[pos]
	data := make([]byte, 2*dim)
	j := 0
	for i := 0; i < int(dim); i++ {
		length := system.ByteOrder.Uint16(d.OriginalData[idx : idx+2])
		data[j] = d.OriginalData[idx]
		data[j+1] = d.OriginalData[idx+1]
		j += 2
		idx += 2
		data = append(data, d.OriginalData[offset:offset+int(length)]...)
		offset += int(length)
	}
	return data, nil
}
func (dd *DeserializedPSDB) GetNumericScalarFeature(pos int) ([]byte, error) {
	size := dd.DataType.Size()
	start := pos * size
	end := start + size
	if start >= len(dd.OriginalData) || end > len(dd.OriginalData) {
		return nil, fmt.Errorf("position out of bounds")
	}
	return dd.OriginalData[start:end], nil
}

func (dd *DeserializedPSDB) GetNumericVectorFeature(pos int, vectorLengths []uint16) ([]byte, error) {
	var start int = 0
	for i, vl := range vectorLengths {
		if i == pos {
			break
		}
		start += int(vl) * dd.DataType.Size()
	}
	end := start + int(vectorLengths[pos])*dd.DataType.Size()
	if start >= len(dd.OriginalData) || end > len(dd.OriginalData) {
		return nil, fmt.Errorf("position out of bounds")
	}
	return dd.OriginalData[start:end], nil
}

func (dd *DeserializedPSDB) GetBoolScalarFeature(pos int) ([]byte, error) {
	byteIndex := pos / 8
	bitOffset := 7 - (pos % 8)
	mask := byte(1 << bitOffset)
	if byteIndex >= len(dd.OriginalData) {
		return nil, fmt.Errorf("position out of bounds")
	}
	bb := (dd.OriginalData[byteIndex] & mask) >> bitOffset
	b := byte(bb)
	return []byte{b}, nil
}

func (dd *DeserializedPSDB) GetBoolVectorFeature(pos int, vectorLengths []uint16) ([]byte, error) {
	// Calculate the starting bit position by summing up previous vector lengths
	startBit := 0
	for i := 0; i < pos; i++ {
		startBit += int(vectorLengths[i])
	}

	vectorLen := int(vectorLengths[pos])
	result := make([]byte, vectorLen) // Allocate enough bytes to hold all bits

	// Read bits from the source and pack them into the result
	for i := 0; i < vectorLen; i++ {
		sourceBitPos := startBit + i
		sourceByteIndex := sourceBitPos / 8
		sourceBitOffset := 7 - (sourceBitPos % 8)
		sourceBitMask := byte(1 << sourceBitOffset)

		if sourceByteIndex >= len(dd.OriginalData) {
			return nil, fmt.Errorf("position out of bounds")
		}

		// Extract the bit from source
		bitValue := (dd.OriginalData[sourceByteIndex] & sourceBitMask) >> sourceBitOffset

		// Place the bit in the result
		result[i] = bitValue
	}

	return result, nil
}

func HelperVectorFeatureToTypeInt32(vector []byte) ([]int32, error) {
	unitSize := types.DataTypeInt32.Size()
	data := make([]int32, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		data[i/unitSize] = system.ByteOrder.Int32(vector[i : i+unitSize])
	}
	return data, nil
}

func HelperScalarFeatureToTypeInt32(feature []byte) (int32, error) {
	unitSize := types.DataTypeInt32.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Int32(feature), nil
}

func HelperVectorFeatureStringToConcatenatedString(vector []byte, vectorLength int) (string, error) {
	data := make([]string, vectorLength)
	dataOffset := int(2 * vectorLength)
	idx := 0
	for i := range vectorLength {
		length := int(system.ByteOrder.Uint16(vector[idx : idx+2]))
		idx += 2
		data[i] = string(vector[dataOffset : dataOffset+length])
		dataOffset += length
	}
	return strings.Join(data, ":"), nil
}

func HelperScalarFeatureToTypeString(feature []byte) (string, error) {
	return string(feature), nil
}

func HelperVectorFeatureBoolToConcatenatedString(vector []byte) (string, error) {
	values := make([]string, len(vector))
	for i := range vector {
		if vector[i] == 1 {
			values[i] = "1"
		} else {
			values[i] = "0"
		}
	}
	return strings.Join(values, ":"), nil
}

func HelperScalarFeatureToTypeBool(feature []byte) (bool, error) {
	if len(feature) != 1 {
		return false, fmt.Errorf("feature is not 1 byte")
	}
	return feature[0] == 1, nil
}

func HelperVectorFeatureFp32ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeFP32.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		floatVal := system.ByteOrder.Float32(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatFloat(float64(floatVal), 'f', -1, 32)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureFp16ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeFP16.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		floatVal := system.ByteOrder.Float16AsFP32(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatFloat(float64(floatVal), 'f', -1, 32)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureFp8E5M2ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeFP8E5M2.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		floatVal := system.ByteOrder.Float8E5M2AsFP32(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatFloat(float64(floatVal), 'f', -1, 32)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureFp8E4M3ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeFP8E4M3.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		floatVal := system.ByteOrder.Float8E4M3AsFP32(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatFloat(float64(floatVal), 'f', -1, 32)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureFp64ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeFP64.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		floatVal := system.ByteOrder.Float64(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatFloat(floatVal, 'f', -1, 64)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureInt8ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeInt8.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		intVal := int8(vector[i])
		values[i/unitSize] = strconv.FormatInt(int64(intVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureInt16ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeInt16.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		intVal := system.ByteOrder.Int16(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatInt(int64(intVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureInt32ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeInt32.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		intVal := system.ByteOrder.Int32(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatInt(int64(intVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureInt64ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeInt64.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		intVal := system.ByteOrder.Int64(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatInt(intVal, 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureUint8ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeUint8.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		uintVal := uint8(vector[i])
		values[i/unitSize] = strconv.FormatUint(uint64(uintVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureUint16ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeUint16.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		uintVal := system.ByteOrder.Uint16(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatUint(uint64(uintVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureUint32ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeUint32.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		uintVal := system.ByteOrder.Uint32(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatUint(uint64(uintVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperVectorFeatureUint64ToConcatenatedString(vector []byte) (string, error) {
	unitSize := types.DataTypeUint64.Size()
	values := make([]string, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		uintVal := system.ByteOrder.Uint64(vector[i : i+unitSize])
		values[i/unitSize] = strconv.FormatUint(uint64(uintVal), 10)
	}
	return strings.Join(values, ":"), nil
}

func HelperScalarFeatureToTypeFloat32(feature []byte) (float32, error) {
	unitSize := types.DataTypeFP32.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Float32(feature), nil
}

func HelperScalarFeatureToTypeFloat16(feature []byte) (float32, error) {
	unitSize := types.DataTypeFP16.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Float16AsFP32(feature), nil
}

func HelperScalarFeatureToTypeFloat8E5M2(feature []byte) (float32, error) {
	unitSize := types.DataTypeFP8E5M2.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Float8E5M2AsFP32(feature), nil
}

func HelperScalarFeatureToTypeFloat8E4M3(feature []byte) (float32, error) {
	unitSize := types.DataTypeFP8E4M3.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Float8E4M3AsFP32(feature), nil
}

func HelperScalarFeatureToTypeFloat64(feature []byte) (float64, error) {
	unitSize := types.DataTypeFP64.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Float64(feature), nil
}

func HelperScalarFeatureToTypeInt8(feature []byte) (int8, error) {
	unitSize := types.DataTypeInt8.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Int8(feature), nil
}

func HelperScalarFeatureToTypeInt16(feature []byte) (int16, error) {
	unitSize := types.DataTypeInt16.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Int16(feature), nil
}

func HelperScalarFeatureToTypeInt64(feature []byte) (int64, error) {
	unitSize := types.DataTypeInt64.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Int64(feature), nil
}

func HelperScalarFeatureToTypeUint8(feature []byte) (uint8, error) {
	unitSize := types.DataTypeUint8.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return feature[0], nil
}

func HelperScalarFeatureToTypeUint16(feature []byte) (uint16, error) {
	unitSize := types.DataTypeUint16.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Uint16(feature), nil
}

func HelperScalarFeatureToTypeUint32(feature []byte) (uint32, error) {
	unitSize := types.DataTypeUint32.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Uint32(feature), nil
}

func HelperScalarFeatureToTypeUint64(feature []byte) (uint64, error) {
	unitSize := types.DataTypeUint64.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Uint64(feature), nil
}

func HelperScalarFeatureToTypeFP8E4M3(feature []byte) (float32, error) {
	unitSize := types.DataTypeFP8E4M3.Size()
	if len(feature) != unitSize {
		return 0, fmt.Errorf("feature is not %v bytes", unitSize)
	}
	return system.ByteOrder.Float8E4M3AsFP32(feature), nil
}

func HelperVectorFeatureToTypeFP8E4M3(vector []byte) ([]float32, error) {
	unitSize := types.DataTypeFP8E4M3.Size()
	data := make([]float32, len(vector)/unitSize)
	for i := 0; i < len(vector); i += unitSize {
		data[i/unitSize] = system.ByteOrder.Float8E4M3AsFP32(vector[i : i+unitSize])
	}
	return data, nil
}

func (d *DeserializedPSDB) Copy() *DeserializedPSDB {
	if d == nil {
		return nil
	}

	copy := &DeserializedPSDB{
		FeatureSchemaVersion: d.FeatureSchemaVersion,
		LayoutVersion:        d.LayoutVersion,
		ExpiryAt:             d.ExpiryAt,
		CompressionType:      d.CompressionType,
		DataType:             d.DataType,
		NegativeCache:        d.NegativeCache,
		Expired:              d.Expired,
	}

	// Deep copy byte slices
	if d.Header != nil {
		copy.Header = make([]byte, len(d.Header))
		copy.Header = append(copy.Header[:0], d.Header...)
	}

	if d.CompressedData != nil {
		copy.CompressedData = make([]byte, len(d.CompressedData))
		copy.CompressedData = append(copy.CompressedData[:0], d.CompressedData...)
	}

	if d.OriginalData != nil {
		copy.OriginalData = make([]byte, len(d.OriginalData))
		copy.OriginalData = append(copy.OriginalData[:0], d.OriginalData...)
	}

	return copy
}
