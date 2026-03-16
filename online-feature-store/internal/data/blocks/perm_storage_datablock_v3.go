package blocks

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

// serializeLayout2 is the top-level serializer for Layout V2 PSDBs.
func (p *PermStorageDataBlock) serializeLayout2() ([]byte, error) {
	err := setupHeadersLayout2(p)
	if err != nil {
		return nil, err
	}
	switch p.dataType {
	case types.DataTypeFP32, types.DataTypeFP16, types.DataTypeFP8E4M3, types.DataTypeFP8E5M2:
		return serializeFP32AndLessLayout2(p)
	case types.DataTypeInt32, types.DataTypeInt16, types.DataTypeInt8:
		return serializeInt32AndLessLayout2(p)
	case types.DataTypeUint32, types.DataTypeUint16, types.DataTypeUint8:
		return serializeUint32AndLessLayout2(p)
	case types.DataTypeFP32Vector, types.DataTypeFP16Vector, types.DataTypeFP8E4M3Vector, types.DataTypeFP8E5M2Vector:
		return serializeFP32VectorAndLessLayout2(p)
	case types.DataTypeInt32Vector, types.DataTypeInt16Vector, types.DataTypeInt8Vector:
		return serializeInt32VectorAndLessLayout2(p)
	case types.DataTypeUint32Vector, types.DataTypeUint16Vector, types.DataTypeUint8Vector:
		return serializeUint32VectorAndLessLayout2(p)
	case types.DataTypeFP64:
		return serializeFP64Layout2(p)
	case types.DataTypeInt64:
		return serializeInt64Layout2(p)
	case types.DataTypeUint64:
		return serializeUint64Layout2(p)
	case types.DataTypeFP64Vector:
		return serializeFP64VectorLayout2(p)
	case types.DataTypeInt64Vector:
		return serializeInt64VectorLayout2(p)
	case types.DataTypeUint64Vector:
		return serializeUint64VectorLayout2(p)
	case types.DataTypeString:
		return serializeStringLayout2(p)
	case types.DataTypeStringVector:
		return serializeStringVectorLayout2(p)
	case types.DataTypeBool:
		return serializeBoolV2(p) // Bool scalar is identical for V1 and V2
	case types.DataTypeBoolVector:
		return serializeBoolVectorLayout2(p)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", p.dataType)
	}
}

// setupHeadersLayout2 writes the 10-byte Layout V2 header.
func setupHeadersLayout2(p *PermStorageDataBlock) error {
	if p == nil {
		return errors.New("perm storage data block v3 is nil")
	}
	if len(p.buf) < PSDBLayout1HeaderBytes {
		return fmt.Errorf("buffer too small: required=%d, actual=%d", PSDBLayout1HeaderBytes, len(p.buf))
	}
	setupFeatureSchemaVersion(p)
	setupExpiryAt(p)
	setupLayoutVersion(p)
	setupDataType(p)
	// Write 10th byte (index 9): bitmap present flag.
	// Only append if the buffer hasn't been extended yet; otherwise overwrite in place
	// to keep setupHeadersLayout2 idempotent across repeated Serialize() calls.
	bitmapByte := byte(0)
	if len(p.bitmap) > 0 {
		bitmapByte = bitmapPresentMask
	}
	if len(p.buf) <= PSDBLayout1HeaderBytes {
		p.buf = append(p.buf, bitmapByte)
	} else {
		p.buf[PSDBLayout1HeaderBytes] = bitmapByte
	}
	return nil
}

// prependBitmapToPayload prepends bitmap bytes to originalData for Layout V2.
func prependBitmapToPayload(p *PermStorageDataBlock) {
	if len(p.bitmap) > 0 {
		tmp := make([]byte, 0, len(p.bitmap)+len(p.originalData))
		tmp = append(tmp, p.bitmap...)
		tmp = append(tmp, p.originalData...)
		p.originalData = tmp
	}
}

// --- Layout V2 scalar numeric serializers ---

func serializeFP32AndLessLayout2(p *PermStorageDataBlock) ([]byte, error) {
	if p.Data == nil {
		return nil, fmt.Errorf("data is nil")
	}
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([]float32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp8, fp16, fp32 Data expected to come in fp32 container")
	}
	idx := 0
	putFloat, _ := system.GetToByteFP32AndLess(p.dataType)

	if len(p.bitmap) > 0 {
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
	prependBitmapToPayload(p)
	return encodeData(p, enc)
}

func serializeInt32AndLessLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([]int32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int8, int16, int32 Data expected to come in int32 container")
	}
	idx := 0
	putInt, _ := system.GetToByteInt32AndLess(p.dataType)

	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			putInt(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
		p.originalData = p.originalData[:idx]
	} else {
		for _, v := range values {
			putInt(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
	}
	prependBitmapToPayload(p)
	return encodeData(p, enc)
}

func serializeUint32AndLessLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([]uint32)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint8, uint16, uint32 Data expected to come in uint32 container")
	}
	idx := 0
	putUint, _ := system.GetToByteUint32AndLess(p.dataType)

	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			putUint(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
		p.originalData = p.originalData[:idx]
	} else {
		for _, v := range values {
			putUint(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
	}
	prependBitmapToPayload(p)
	return encodeData(p, enc)
}

func serializeFP64Layout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([]float64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp64 Data expected to come in fp64 container")
	}
	idx := 0
	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			system.ByteOrder.PutFloat64(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
		p.originalData = p.originalData[:idx]
	} else {
		for _, v := range values {
			system.ByteOrder.PutFloat64(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
	}
	prependBitmapToPayload(p)
	return encodeData(p, enc)
}

func serializeInt64Layout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([]int64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int64 Data expected to come in int64 container")
	}
	idx := 0
	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			system.ByteOrder.PutInt64(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
		p.originalData = p.originalData[:idx]
	} else {
		for _, v := range values {
			system.ByteOrder.PutInt64(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
	}
	prependBitmapToPayload(p)
	return encodeData(p, enc)
}

func serializeUint64Layout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([]uint64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint64 Data expected to come in uint64 container")
	}
	idx := 0
	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			system.ByteOrder.PutUint64(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
		p.originalData = p.originalData[:idx]
	} else {
		for _, v := range values {
			system.ByteOrder.PutUint64(p.originalData[idx:idx+unitSize], v)
			idx += unitSize
		}
	}
	prependBitmapToPayload(p)
	return encodeData(p, enc)
}

// --- Layout V2 string serializers ---

func serializeStringLayout2(p *PermStorageDataBlock) ([]byte, error) {
	values, ok := p.Data.([]string)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("string data expected to come in string container")
	}
	if len(values) != len(p.stringLengths) {
		return nil, fmt.Errorf("mismatch in number of strings (%d) and number of defined string lengths (%d)",
			len(values), len(p.stringLengths))
	}

	if len(p.bitmap) > 0 {
		dense := make([]byte, 0)
		for i, str := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			strLen := len(str)
			if strLen > maxStringLength || strLen > int(p.stringLengths[i]) {
				return nil, fmt.Errorf("string at index %d of length %d exceeds max length of %d or booked size %d", i, strLen, maxStringLength, p.stringLengths[i])
			}
			lenBuf := make([]byte, 2)
			system.ByteOrder.PutUint16(lenBuf, uint16(strLen))
			dense = append(dense, lenBuf...)
			dense = append(dense, []byte(str)...)
		}
		p.originalData = make([]byte, 0, len(p.bitmap)+len(dense))
		p.originalData = append(p.originalData, p.bitmap...)
		p.originalData = append(p.originalData, dense...)
	} else {
		// Dense path (no bitmap) — same as V1
		strLenOffsetIdx := 0
		strDataOffsetIdx := len(values) * 2
		for i, str := range values {
			strLen := len(str)
			if strLen > maxStringLength || strLen > int(p.stringLengths[i]) {
				return nil, fmt.Errorf("string at index %d of length %d exceeds max length of %d or booked size %d", i, strLen, maxStringLength, p.stringLengths[i])
			}
			system.ByteOrder.PutUint16(p.originalData[strLenOffsetIdx:], uint16(strLen))
			copy(p.originalData[strDataOffsetIdx:], []byte(str))
			strLenOffsetIdx += 2
			strDataOffsetIdx += strLen
		}
		p.originalData = p.originalData[:strDataOffsetIdx]
	}
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	return encodeData(p, enc)
}

func serializeStringVectorLayout2(p *PermStorageDataBlock) ([]byte, error) {
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

	if len(p.bitmap) > 0 {
		dense := make([]byte, 0)
		for i, vec := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
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
				lenBuf := make([]byte, 2)
				system.ByteOrder.PutUint16(lenBuf, uint16(strLen))
				dense = append(dense, lenBuf...)
				dense = append(dense, []byte(str)...)
			}
		}
		p.originalData = make([]byte, 0, len(p.bitmap)+len(dense))
		p.originalData = append(p.originalData, p.bitmap...)
		p.originalData = append(p.originalData, dense...)
		return encodeData(p, enc)
	}

	// Dense path (no bitmap) — same as V1
	totalStrings := 0
	for i := range values {
		totalStrings += int(p.vectorLengths[i])
	}
	strLenOffsetIdx := 0
	strDataOffsetIdx := totalStrings * 2
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
			system.ByteOrder.PutUint16(p.originalData[strLenOffsetIdx:], uint16(strLen))
			copy(p.originalData[strDataOffsetIdx:], []byte(str))
			strLenOffsetIdx += 2
			strDataOffsetIdx += strLen
		}
	}
	p.originalData = p.originalData[:strDataOffsetIdx]
	return encodeData(p, enc)
}

// --- Layout V2 vector numeric serializers ---

func serializeFP32VectorAndLessLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
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

	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				putFloat(p.originalData[idx:idx+unitSize], vv)
				idx += unitSize
			}
		}
		p.originalData = p.originalData[:idx]
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}

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

func serializeInt32VectorAndLessLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
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

	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				putInt(p.originalData[idx:idx+unitSize], vv)
				idx += unitSize
			}
		}
		p.originalData = p.originalData[:idx]
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}

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

func serializeUint32VectorAndLessLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
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

	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				putUint(p.originalData[idx:idx+unitSize], vv)
				idx += unitSize
			}
		}
		p.originalData = p.originalData[:idx]
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}

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

func serializeFP64VectorLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([][]float64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("fp64 vector Data expected to come in fp64 vector container")
	}
	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}
	idx := 0
	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				system.ByteOrder.PutFloat64(p.originalData[idx:idx+unitSize], vv)
				idx += unitSize
			}
		}
		p.originalData = p.originalData[:idx]
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}
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

func serializeInt64VectorLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([][]int64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("int64 vector Data expected to come in int64 vector container")
	}
	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}
	idx := 0
	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				system.ByteOrder.PutInt64(p.originalData[idx:idx+unitSize], vv)
				idx += unitSize
			}
		}
		p.originalData = p.originalData[:idx]
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}
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

func serializeUint64VectorLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
	unitSize := p.dataType.Size()
	values, ok := p.Data.([][]uint64)
	if !ok || values == nil || len(values) == 0 {
		return nil, fmt.Errorf("uint64 vector Data expected to come in uint64 vector container")
	}
	if len(values) != len(p.vectorLengths) {
		return nil, fmt.Errorf("mismatch in number of vectors (%d) and number of defined vector lengths (%d)",
			len(values), len(p.vectorLengths))
	}
	idx := 0
	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				system.ByteOrder.PutUint64(p.originalData[idx:idx+unitSize], vv)
				idx += unitSize
			}
		}
		p.originalData = p.originalData[:idx]
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}
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

// --- Layout V2 bool vector serializer ---

func serializeBoolVectorLayout2(p *PermStorageDataBlock) ([]byte, error) {
	enc, err := compression.GetEncoder(p.compressionType)
	if err != nil {
		return nil, err
	}
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

	if len(p.bitmap) > 0 {
		for i, v := range values {
			if (p.bitmap[i/8] & (1 << (i % 8))) == 0 {
				continue
			}
			if len(v) != int(p.vectorLengths[i]) {
				return nil, fmt.Errorf("mismatch in vector length at index %d", i)
			}
			for _, vv := range v {
				if vv > 1 {
					return nil, fmt.Errorf("invalid bool value: %d; expected 0 or 1", vv)
				}
				p.originalData[idx] |= vv << shift
				shift--
				if shift < 0 {
					shift = 7
					idx++
				}
			}
		}
		usedLen := idx
		if shift != 7 {
			usedLen++
		}
		p.originalData = p.originalData[:usedLen]
		x := byte((shift + 1) % 8)
		setupBoolDtypeLastIdx(p, x)
		prependBitmapToPayload(p)
		return encodeData(p, enc)
	}

	for i, v := range values {
		if len(v) != int(p.vectorLengths[i]) {
			return nil, fmt.Errorf("mismatch in vector length at index %d", i)
		}
		for _, vv := range v {
			if vv > 1 {
				return nil, fmt.Errorf("invalid bool value: %d; expected 0 or 1", vv)
			}
			p.originalData[idx] |= vv << shift
			shift--
			if shift < 0 {
				shift = 7
				idx++
			}
		}
	}
	x := byte((shift + 1) % 8)
	if x&0x07 != x {
		return nil, fmt.Errorf("issue with shift operation in bool v")
	}
	setupBoolDtypeLastIdx(p, x)
	return encodeData(p, enc)
}
