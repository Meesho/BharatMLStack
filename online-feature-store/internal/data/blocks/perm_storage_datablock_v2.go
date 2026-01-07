package blocks

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

//Data Layout
//[0-15]bits [0th and 1st byte] - Feature Schema Version
//[16-55]bits [2nd to 6th byte] - Expiry At
//[56-59]bits [7th byte] - Layout Version
//[60-62]bits [7th byte] - Compression Type
//[63-67]bits [7th and 8th byte] - Data Type
//[68-71]bits [8th byte] - Bool Dtype Last Index
//Total 9 bytes Header Length

//Data Layout 2 Additional Bytes
// bitmapMeta (1 byte):
// bits 0–2 : bitmapLastBitIndex (1–8)
// bit 3    : bitmapPresent
// bits 4–7 : reserved (future)

const (
	PSDBLayout1LengthBytes = 9
	PSDBLayout2ExtraBytes  = 1
	maxStringLength        = 65535
	layoutVersionIdx       = 7
)

type PermStorageDataBlock struct {
	// 64-bit aligned fields
	expiryAt       uint64
	Data           interface{}
	bitmap         []byte // NEW, optional, nil by default
	buf            []byte
	originalData   []byte
	compressedData []byte
	stringLengths  []uint16
	vectorLengths  []uint16
	Builder        *PermStorageDataBlockBuilder

	// 32-bit fields
	noOfFeatures      int
	originalDataLen   int
	compressedDataLen int

	// 16-bit field
	featureSchemaVersion uint16

	// 8-bit fields
	layoutVersion    uint8
	compressionType  compression.Type
	dataType         types.DataType
	boolDtypeLastIdx uint8
	bitmapMeta       byte // NEW: layout-2 bitmap metadata
}

func (p *PermStorageDataBlock) Clear() {
	p.layoutVersion = 0
	p.featureSchemaVersion = 0
	p.expiryAt = 0
	p.noOfFeatures = 0
	p.compressionType = compression.TypeNone
	p.dataType = types.DataTypeUnknown
	p.boolDtypeLastIdx = 0
	p.originalDataLen = 0
	p.compressedDataLen = 0
	headerLen := PSDBLayout1LengthBytes
	if p.layoutVersion == 2 {
		headerLen = PSDBLayout1LengthBytes + PSDBLayout2ExtraBytes
	}
	if len(p.buf) > headerLen {
		p.buf = p.buf[:headerLen]
	}
	if len(p.originalData) > 0 {
		p.originalData = p.originalData[:0]
	}
	if len(p.compressedData) > 0 {
		p.compressedData = p.compressedData[:0]
	}
	p.Data = nil
	p.stringLengths = nil
	p.vectorLengths = nil
	p.bitmap = nil
	p.bitmapMeta = byte(0)
}

func (b *PermStorageDataBlockBuilder) SetBitmap(bitmap []byte) *PermStorageDataBlockBuilder {
	if len(bitmap) > 0 {
		b.psdb.bitmap = bitmap
	} else {
		b.psdb.bitmap = make([]byte, 0)
	}
	return b
}

func (b *PermStorageDataBlockBuilder) SetupBitmapMeta(numFeatures int) *PermStorageDataBlockBuilder {
	// Bitmap meta is only valid for layout-2
	if b.psdb.layoutVersion != 2 {
		return b
	}

	if len(b.psdb.bitmap) == 0 {
		b.psdb.bitmapMeta = 0 // bitmapPresent = 0
		return b
	}

	lastBits := numFeatures % 8
	if lastBits == 0 {
		lastBits = 8
	}

	meta := byte(0)
	meta |= 1 << 3                // bitmapPresent
	meta |= byte(lastBits & 0x07) // last bit count (1–8)
	b.psdb.bitmapMeta = meta
	return b
}

func (p *PermStorageDataBlock) Serialize() ([]byte, error) {
	switch p.layoutVersion {
	case 1:
		return p.serializeLayout1()
	case 2:
		return p.serializeLayout1()
	default:
		return nil, fmt.Errorf("unsupported layout version: %d", p.layoutVersion)
	}
}
func (p *PermStorageDataBlock) serializeLayout1() ([]byte, error) {
	err := setupHeadersV2(p)
	if err != nil {
		return nil, err
	}
	switch p.dataType {
	case types.DataTypeFP32, types.DataTypeFP16, types.DataTypeFP8E4M3, types.DataTypeFP8E5M2:
		return serializeFP32AndLessV2(p)
	case types.DataTypeInt32, types.DataTypeInt16, types.DataTypeInt8:
		return serializeInt32AndLessV2(p)
	case types.DataTypeUint32, types.DataTypeUint16, types.DataTypeUint8:
		return serializeUint32AndLessV2(p)
	case types.DataTypeFP32Vector, types.DataTypeFP16Vector, types.DataTypeFP8E4M3Vector, types.DataTypeFP8E5M2Vector:
		return serializeFP32VectorAndLessV2(p)
	case types.DataTypeInt32Vector, types.DataTypeInt16Vector, types.DataTypeInt8Vector:
		return serializeInt32VectorAndLessV2(p)
	case types.DataTypeUint32Vector, types.DataTypeUint16Vector, types.DataTypeUint8Vector:
		return serializeUint32VectorAndLessV2(p)
	case types.DataTypeFP64:
		return serializeFP64V2(p)
	case types.DataTypeInt64:
		return serializeInt64V2(p)
	case types.DataTypeUint64:
		return serializeUint64V2(p)
	case types.DataTypeFP64Vector:
		return serializeFP64VectorV2(p)
	case types.DataTypeInt64Vector:
		return serializeInt64VectorV2(p)
	case types.DataTypeUint64Vector:
		return serializeUint64VectorV2(p)
	case types.DataTypeString:
		return serializeStringV2(p)
	case types.DataTypeStringVector:
		return serializeStringVectorV2(p)
	case types.DataTypeBool:
		return serializeBoolV2(p)
	case types.DataTypeBoolVector:
		return serializeBoolVectorV2(p)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", p.dataType)
	}
}

func setupHeadersV2(p *PermStorageDataBlock) error {
	if p == nil {
		return errors.New("perm storage data block v2 is nil")
	}

	if len(p.buf) < PSDBLayout1LengthBytes {
		return fmt.Errorf("buffer too small: required=%d, actual=%d", PSDBLayout1LengthBytes, len(p.buf))
	}

	setupFeatureSchemaVersion(p)
	setupExpiryAt(p)
	setupLayoutVersion(p)
	setupDataType(p)
	return nil
}

func setupFeatureSchemaVersion(p *PermStorageDataBlock) {
	system.ByteOrder.PutUint16(p.buf[0:2], p.featureSchemaVersion)
}

func setupExpiryAt(p *PermStorageDataBlock) error {
	expiryAtBytes, err := system.EncodeExpiry(p.expiryAt)
	if err != nil {
		return err
	}
	copy(p.buf[2:7], expiryAtBytes)
	return nil
}

func setupLayoutVersion(p *PermStorageDataBlock) {
	// Clear the upper 4 bits (4-7) of byte 7, then set layout version in upper 4 bits
	p.buf[7] = (p.buf[7] & 0x0F) | ((p.layoutVersion & 0x0F) << 4)
}

func setupCompressionType(p *PermStorageDataBlock) {
	// Clear bits 1-3 of byte 7, then set compression type in bits 1-3
	p.buf[7] = (p.buf[7] & 0xF1) | ((uint8(p.compressionType) & 0x07) << 1)
}

func clearCompressionBits(header []byte) {
	header[7] = (header[7] & 0xF1)
}

func setupDataType(p *PermStorageDataBlock) {
	// For byte 7: Clear bit 0, then set the highest bit of dataType in bit 0
	p.buf[7] = (p.buf[7] & 0xFE) | ((uint8(p.dataType) & 0x10) >> 4)

	// For byte 8: Clear upper 4 bits (4-7), then set the lower 4 bits of dataType
	p.buf[8] = (p.buf[8] & 0x0F) | ((uint8(p.dataType) & 0x0F) << 4)
}

func setupBoolDtypeLastIdx(p *PermStorageDataBlock, boolDtypeLastIdx uint8) {
	// For byte 8: Clear lower 4 bits, then set the lower 4 bits of boolDtypeLastIdx
	p.buf[8] = (p.buf[8] & 0xF0) | (boolDtypeLastIdx & 0x0F)
}

func encodeData(p *PermStorageDataBlock, enc compression.Encoder) ([]byte, error) {
	p.originalDataLen = len(p.originalData)
	enc.EncodeV2(p.originalData, &p.compressedData)
	// If compression is not effective, use original data
	if len(p.compressedData) >= p.originalDataLen {
		copy(p.compressedData, p.originalData)
		p.compressedDataLen = p.originalDataLen
		p.compressedData = p.compressedData[:p.compressedDataLen]
		p.compressionType = compression.TypeNone
	} else {
		p.compressedDataLen = len(p.compressedData)
		p.compressionType = enc.EncoderType()
	}
	setupCompressionType(p)
	p.buf = append(p.buf, p.compressedData...)
	return p.buf, nil
}

func serializeFP32AndLessV2(p *PermStorageDataBlock) ([]byte, error) {
	if p.Data == nil {
		return nil, fmt.Errorf("data is nil")
	}
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values []float32
	values, ok := p.Data.([]float32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp8, fp16, fp32 Data expected to come in fp32 container")
	}
	idx := 0
	putFloat, _ := system.GetToByteFP32AndLess(p.dataType)

	if p.layoutVersion == 2 && len(p.bitmap) > 0 {

		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			putFloat(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}

		p.originalData = p.originalData[:idx]
	} else {
		for _, v := range values {
			putFloat(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
	}

	// ─────────────────────────────
	// Step 2: layout-2 payload handling
	// ─────────────────────────────
	if p.layoutVersion == 2 {
		// prepend bitmap to payload if present
		if len(p.bitmap) > 0 {
			p.bitmapMeta = p.bitmapMeta | 1<<3 // bitmapPresent = 1
			tmp := make([]byte, 0, len(p.bitmap)+len(p.originalData))
			tmp = append(tmp, p.bitmap...)
			tmp = append(tmp, p.originalData...)
			p.originalData = tmp
		}

		// append bitmapMeta to header
		if len(p.buf) != PSDBLayout1LengthBytes {
			return nil, fmt.Errorf("invalid base header length for layout-2")
		}
		p.buf = append(p.buf, p.bitmapMeta)
	}

	return encodeData(p, enc)
}

func serializeInt32AndLessV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values []int32
	values, ok := p.Data.([]int32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int8, int16, int32 Data expected to come in int32 container")
	}
	idx := 0
	putInt, _ := system.GetToByteInt32AndLess(p.dataType)
	for _, v := range values {
		putInt(p.originalData[idx:idx+unitSize], v)
		idx += unitSize
	}
	return encodeData(p, enc)
}

func serializeUint32AndLessV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values []uint32
	values, ok := p.Data.([]uint32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint8, uint16, uint32 Data expected to come in uint32 container")
	}
	idx := 0
	putUint, _ := system.GetToByteUint32AndLess(p.dataType)
	for _, v := range values {
		putUint(p.originalData[idx:idx+unitSize], v)
		idx += unitSize
	}
	return encodeData(p, enc)
}

func serializeFP64V2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values []float64
	values, ok := p.Data.([]float64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp64 Data expected to come in fp64 container")
	}
	idx := 0
	for _, v := range values {
		system.ByteOrder.PutFloat64(p.originalData[idx:idx+unitSize], v)
		idx += unitSize
	}
	return encodeData(p, enc)
}

func serializeInt64V2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values []int64
	values, ok := p.Data.([]int64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int64 Data expected to come in int64 container")
	}
	idx := 0
	for _, v := range values {
		system.ByteOrder.PutInt64(p.originalData[idx:idx+unitSize], v)
		idx += unitSize
	}
	return encodeData(p, enc)
}

func serializeUint64V2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values []uint64
	values, ok := p.Data.([]uint64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint64 Data expected to come in uint64 container")
	}
	idx := 0
	for _, v := range values {
		system.ByteOrder.PutUint64(p.originalData[idx:idx+unitSize], v)
		idx += unitSize
	}
	return encodeData(p, enc)
}

// serializeStringV2 serializes string data into a byte slice, using pascal string encoding.
// Pascal string format: https://wiki.freepascal.org/String_Types#ShortString_.28String.5B1..255.5D.29
// Each string is stored as a 2-byte length prefix followed by the string data:
// [len1][len2]...[lenN][str1][str2]...[strN]
// where each len is a uint16 (max 65535) and stored in system byte order.
func serializeStringV2(p *PermStorageDataBlock) ([]byte, error) {
	values, ok := p.Data.([]string)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("string data expected to come in string container")
	}

	if len(values) != len(p.stringLengths) {
		return nil, fmt.Errorf("mismatch in number of strings (%d) and number of defined string lengths (%d)",
			len(values), len(p.stringLengths))
	}

	strLenOffsetIdx := 0
	strDataOffsetIdx := len(values) * 2 // Start of string data after all length offsets

	for i, str := range values {
		strLen := len(str)
		if strLen > maxStringLength || strLen > int(p.stringLengths[i]) {
			return nil, fmt.Errorf("string at index %d of length %d exceeds max length of %d or booked size %d", i, strLen, maxStringLength, p.stringLengths[i])
		}
		// Write offset
		system.ByteOrder.PutUint16(p.originalData[strLenOffsetIdx:], uint16(strLen))

		// Write string
		copy(p.originalData[strDataOffsetIdx:], []byte(str))

		// Update indices
		strLenOffsetIdx += 2
		strDataOffsetIdx += strLen
	}
	p.originalData = p.originalData[:strDataOffsetIdx]

	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}

	return encodeData(p, enc)
}

func serializeBoolV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	var values []uint8
	values, ok := p.Data.([]uint8)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("bool Data expected to come in uin8 container")
	}
	idx := 0
	shift := 7
	for _, v := range values {
		p.originalData[idx] |= v << shift
		shift--
		if shift < 0 {
			shift = 7
			idx++
		}
	}
	// Handling terminal bit position for the last bit of Data
	x := byte((shift + 1) % 8)
	if x&0x07 != x {
		return nil, fmt.Errorf("issue with shift operation in bool v")
	}
	setupBoolDtypeLastIdx(p, x)
	return encodeData(p, enc)
}

func serializeFP32VectorAndLessV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values [][]float32
	values, ok := p.Data.([][]float32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp32 vector Data expected to come in fp32 vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	idx := 0
	putFloat, _ := system.GetToByteFP32AndLess(p.dataType)
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			putFloat(p.originalData[idx:idx+unitSize], vv)
			idx += unitSize
		}
	}
	return encodeData(p, enc)
}

func serializeInt32VectorAndLessV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values [][]int32
	values, ok := p.Data.([][]int32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int32 vector Data expected to come in int32 vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	idx := 0
	putInt, _ := system.GetToByteInt32AndLess(p.dataType)
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			putInt(p.originalData[idx:idx+unitSize], vv)
			idx += unitSize
		}
	}
	return encodeData(p, enc)
}

func serializeUint32VectorAndLessV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values [][]uint32
	values, ok := p.Data.([][]uint32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint32 vector Data expected to come in uint32 vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	idx := 0
	putUint, _ := system.GetToByteUint32AndLess(p.dataType)
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			putUint(p.originalData[idx:idx+unitSize], vv)
			idx += unitSize
		}
	}
	return encodeData(p, enc)
}

func serializeFP64VectorV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values [][]float64
	values, ok := p.Data.([][]float64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp64 vector Data expected to come in fp64 vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	idx := 0
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			system.ByteOrder.PutFloat64(p.originalData[idx:idx+unitSize], vv)
			idx += unitSize
		}
	}
	return encodeData(p, enc)
}

func serializeInt64VectorV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values [][]int64
	values, ok := p.Data.([][]int64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int64 vector Data expected to come in int64 vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}
	idx := 0
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			system.ByteOrder.PutInt64(p.originalData[idx:idx+unitSize], vv)
			idx += unitSize
		}
	}
	return encodeData(p, enc)
}

func serializeUint64VectorV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	var values [][]uint64
	values, ok := p.Data.([][]uint64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint64 vector Data expected to come in uint64 vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	idx := 0
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			system.ByteOrder.PutUint64(p.originalData[idx:idx+unitSize], vv)
			idx += unitSize
		}
	}
	return encodeData(p, enc)
}

// serializeStringVectorV2 serializes string vector data into a byte slice, using pascal string encoding.
// Each string in each vector is stored as a 2-byte length prefix followed by the string data.
// stringLengths[i] defines the maximum length for each string in vector[i].
// Format: [len1][len2]...[lenN][str1][str2]...[strN]
// Example: For stringLengths [5,6] and vectorLengths [2,3]:
// - First vector (length 2) can have strings up to length 5
// - Second vector (length 3) can have strings up to length 6
// [len1][len2][len3][len4][len5][V1str1][V1str2][V2str1][V2str2][V2str3]
func serializeStringVectorV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}

	values, ok := p.Data.([][]string)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("string vector data expected to come in string vector container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	if len(values) != len(p.stringLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined string lengths (%d)",
			len(values), len(p.stringLengths))
	}

	// Calculate total number of strings
	totalStrings := 0
	for i := range values {
		totalStrings += int(p.vectorLengths[i])
	}

	strLenOffsetIdx := 0
	strDataOffsetIdx := totalStrings * 2 // Start of string data after all length prefixes

	for i, vec := range values {
		if len(vec) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d: expected %d, got %d",
				i, p.vectorLengths[i], len(vec))
		}
		for _, str := range vec {
			strLen := len(str)
			if strLen > maxStringLength || strLen > int(p.stringLengths[i]) {
				return nil, fmt.Errorf("string in vector %d of length %d exceeds max length of %d or booked size %d",
					i, strLen, maxStringLength, p.stringLengths[i])
			}

			// Write string length
			system.ByteOrder.PutUint16(p.originalData[strLenOffsetIdx:], uint16(strLen))

			// Write string data
			copy(p.originalData[strDataOffsetIdx:], []byte(str))

			// Update indices
			strLenOffsetIdx += 2
			strDataOffsetIdx += strLen
		}
	}

	// Trim the buffer to actual size
	p.originalData = p.originalData[:strDataOffsetIdx]
	return encodeData(p, enc)
}

func serializeBoolVectorV2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	// Casting p.Data to expected Data type []uint8 (assuming each bool is represented by 1 bit)
	var values [][]uint8
	values, ok := p.Data.([][]uint8)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("bool v Data expected to come in [][]uint8 container")
	}

	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}

	idx := 0
	shift := 7
	// Iterate over each v in the 2D slice
	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			if vv > 1 {
				return nil, fmt.Errorf("invalid bool value: %d; expected 0 or 1", vv)
			}
			// Set each bit in the current byte in p.originalData[idx]
			p.originalData[idx] |= vv << shift
			shift--
			if shift < 0 {
				shift = 7
				idx++
			}
		}
	}
	// Handling terminal bit position for the last bit of Data
	x := byte((shift + 1) % 8)
	if x&0x07 != x {
		return nil, fmt.Errorf("issue with shift operation in bool v")
	}
	setupBoolDtypeLastIdx(p, x)
	return encodeData(p, enc)
}
