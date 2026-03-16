package blocks

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

// DeserializedPSDBLayout2 is the Layout V2 deserialized permanent storage data block.
// It embeds the V1 struct and adds bitmap metadata for sparse feature encoding.
type DeserializedPSDBLayout2 struct {
	DeserializedPSDB
	BitmapMeta byte // V2-only: bit 0 = bitmap present
}

// PSDBBlock interface implementation — override methods that differ from V1

func (d *DeserializedPSDBLayout2) CopyBlock() PSDBBlock {
	if d == nil {
		return nil
	}
	cp := &DeserializedPSDBLayout2{
		BitmapMeta: d.BitmapMeta,
	}
	cp.FeatureSchemaVersion = d.FeatureSchemaVersion
	cp.LayoutVersion = d.LayoutVersion
	cp.ExpiryAt = d.ExpiryAt
	cp.CompressionType = d.CompressionType
	cp.DataType = d.DataType
	cp.NegativeCache = d.NegativeCache
	cp.Expired = d.Expired
	if d.Header != nil {
		cp.Header = make([]byte, len(d.Header))
		copy(cp.Header, d.Header)
	}
	if d.CompressedData != nil {
		cp.CompressedData = make([]byte, len(d.CompressedData))
		copy(cp.CompressedData, d.CompressedData)
	}
	if d.OriginalData != nil {
		cp.OriginalData = make([]byte, len(d.OriginalData))
		copy(cp.OriginalData, d.OriginalData)
	}
	return cp
}

// --- Deserialization ---

func deserializePSDBForLayout2(data []byte) (*DeserializedPSDBLayout2, error) {
	if len(data) < PSDBLayout2HeaderBytes {
		return nil, fmt.Errorf("data is too short to contain a valid layout-2 PSDB header")
	}
	featureSchemaVersion := system.ByteOrder.Uint16(data[0:2])
	expiryAt, err := system.DecodeExpiry(data[2:7])
	isExpired := system.IsExpired(data[2:7])
	if err != nil {
		return nil, err
	}
	compressionType := compression.Type((data[7] & 0x0E) >> 1)
	dtT := (data[7] & 0x01) << 4
	dtT |= ((data[8] & 0xF0) >> 4)
	dataType := types.DataType(dtT)

	bitmapMeta := data[PSDBLayout1HeaderBytes] & bitmapPresentMask
	header := data[:PSDBLayout2HeaderBytes]

	payload := data[PSDBLayout2HeaderBytes:]
	var originalData []byte
	var compressedData []byte

	if compressionType == compression.TypeNone {
		originalData = payload
		compressedData = payload
	} else {
		dec, err := compression.GetDecoder(compressionType)
		if err != nil {
			return nil, err
		}
		compressedData = payload
		originalData, err = dec.Decode(payload)
		if err != nil {
			return nil, err
		}
	}
	return &DeserializedPSDBLayout2{
		DeserializedPSDB: DeserializedPSDB{
			FeatureSchemaVersion: featureSchemaVersion,
			LayoutVersion:        2,
			ExpiryAt:             expiryAt,
			CompressionType:      compressionType,
			DataType:             dataType,
			Header:               header,
			CompressedData:       compressedData,
			OriginalData:         originalData,
			NegativeCache:        false,
			Expired:              isExpired,
		},
		BitmapMeta: bitmapMeta,
	}, nil
}

func deserializePSDBForLayout2WithoutDecompression(data []byte) (*DeserializedPSDBLayout2, error) {
	if len(data) < PSDBLayout2HeaderBytes {
		return nil, fmt.Errorf("data is too short to contain a valid layout-2 PSDB header")
	}
	featureSchemaVersion := system.ByteOrder.Uint16(data[0:2])
	expiryAt, err := system.DecodeExpiry(data[2:7])
	isExpired := system.IsExpired(data[2:7])
	if err != nil {
		return nil, err
	}
	compressionType := compression.Type((data[7] & 0x0E) >> 1)
	dtT := (data[7] & 0x01) << 4
	dtT |= ((data[8] & 0xF0) >> 4)
	dataType := types.DataType(dtT)

	bitmapMeta := data[PSDBLayout1HeaderBytes] & bitmapPresentMask
	header := data[:PSDBLayout2HeaderBytes]
	originalData := data[PSDBLayout2HeaderBytes:]
	compressedData := data[PSDBLayout2HeaderBytes:]

	return &DeserializedPSDBLayout2{
		DeserializedPSDB: DeserializedPSDB{
			FeatureSchemaVersion: featureSchemaVersion,
			LayoutVersion:        2,
			ExpiryAt:             expiryAt,
			CompressionType:      compressionType,
			DataType:             dataType,
			Header:               header,
			CompressedData:       compressedData,
			OriginalData:         originalData,
			NegativeCache:        false,
			Expired:              isExpired,
		},
		BitmapMeta: bitmapMeta,
	}, nil
}

// --- Layout V2 Feature Extraction Methods (bitmap-aware sparse access) ---

func (d *DeserializedPSDBLayout2) GetStringScalarFeature(pos int, noOfFeatures int, defaultValue []byte) ([]byte, error) {
	if d.DataType != types.DataTypeString {
		return nil, fmt.Errorf("data type is not a string")
	}
	data := d.OriginalData

	if (d.BitmapMeta & bitmapPresentMask) != 0 {
		bitmapSize := (noOfFeatures + 7) / 8
		if len(data) < bitmapSize {
			return nil, fmt.Errorf("corrupt bitmap payload")
		}
		bitmap := data[:bitmapSize]
		dense := data[bitmapSize:]
		byteIdx := pos / 8
		bitIdx := pos % 8
		if byteIdx >= len(bitmap) {
			return nil, fmt.Errorf("bitmap index out of bounds")
		}
		if (bitmap[byteIdx] & (1 << bitIdx)) == 0 {
			return defaultValue, nil
		}
		denseIdx := countSetBitsBefore(bitmap, pos, noOfFeatures)
		offset, length, err := skipStringsInDense(dense, denseIdx)
		if err != nil {
			return nil, err
		}
		if offset+int(length) > len(dense) {
			return nil, fmt.Errorf("string scalar dense offset out of bounds")
		}
		return dense[offset : offset+int(length)], nil
	}

	// Fallback to V1 dense path if bitmap not present
	return d.DeserializedPSDB.GetStringScalarFeature(pos, noOfFeatures, defaultValue)
}

func (d *DeserializedPSDBLayout2) GetStringVectorFeature(pos int, noOfFeatures int, vectorLengths []uint16, defaultValue []byte) ([]byte, error) {
	if d.DataType != types.DataTypeStringVector {
		return nil, fmt.Errorf("data type is not a string vector")
	}
	data := d.OriginalData
	numVectors := len(vectorLengths)

	if (d.BitmapMeta & bitmapPresentMask) != 0 {
		bitmapSize := (numVectors + 7) / 8
		if len(data) < bitmapSize {
			return nil, fmt.Errorf("corrupt bitmap payload")
		}
		bitmap := data[:bitmapSize]
		dense := data[bitmapSize:]
		byteIdx := pos / 8
		bitIdx := pos % 8
		if byteIdx >= len(bitmap) {
			return nil, fmt.Errorf("bitmap index out of bounds")
		}
		if (bitmap[byteIdx] & (1 << bitIdx)) == 0 {
			return defaultValue, nil
		}
		if pos >= len(vectorLengths) {
			return nil, fmt.Errorf("pos %d out of bounds for vectorLengths (len=%d)", pos, len(vectorLengths))
		}
		offset, err := skipStringVectorsInDense(dense, vectorLengths, bitmap, pos)
		if err != nil {
			return nil, err
		}
		dim := vectorLengths[pos]
		vecSize := 0
		o := offset
		for i := 0; i < int(dim); i++ {
			if o+2 > len(dense) {
				return nil, fmt.Errorf("string vector dense out of bounds")
			}
			length := system.ByteOrder.Uint16(dense[o : o+2])
			vecSize += 2 + int(length)
			o += 2 + int(length)
			if o > len(dense) {
				return nil, fmt.Errorf("string vector dense out of bounds")
			}
		}
		return dense[offset : offset+vecSize], nil
	}

	return d.DeserializedPSDB.GetStringVectorFeature(pos, noOfFeatures, vectorLengths, defaultValue)
}

func (d *DeserializedPSDBLayout2) GetNumericScalarFeature(pos int, numFeatures int, defaultValue []byte) ([]byte, error) {
	size := d.DataType.Size()
	data := d.OriginalData

	if (d.BitmapMeta & bitmapPresentMask) != 0 {
		bitmapSize := (numFeatures + 7) / 8
		if len(data) < bitmapSize {
			return nil, fmt.Errorf("corrupt bitmap payload")
		}
		bitmap := data[:bitmapSize]
		dense := data[bitmapSize:]
		byteIdx := pos / 8
		bitIdx := pos % 8
		if byteIdx >= len(bitmap) {
			return nil, fmt.Errorf("bitmap index out of bounds")
		}
		if (bitmap[byteIdx] & (1 << bitIdx)) == 0 {
			return defaultValue, nil
		}
		denseIdx := countSetBitsBefore(bitmap, pos, numFeatures)
		start := denseIdx * size
		end := start + size
		if end > len(dense) {
			return nil, fmt.Errorf(
				"dense offset out of bounds (idx=%d start=%d len=%d)",
				denseIdx, start, len(dense),
			)
		}
		return dense[start:end], nil
	}

	return d.DeserializedPSDB.GetNumericScalarFeature(pos, numFeatures, defaultValue)
}

func (d *DeserializedPSDBLayout2) GetNumericVectorFeature(pos int, vectorLengths []uint16, defaultValue []byte) ([]byte, error) {
	data := d.OriginalData
	numVectors := len(vectorLengths)
	size := d.DataType.Size()

	if (d.BitmapMeta & bitmapPresentMask) != 0 {
		bitmapSize := (numVectors + 7) / 8
		if len(data) < bitmapSize {
			return nil, fmt.Errorf("corrupt bitmap payload")
		}
		bitmap := data[:bitmapSize]
		dense := data[bitmapSize:]
		byteIdx := pos / 8
		bitIdx := pos % 8
		if byteIdx >= len(bitmap) {
			return nil, fmt.Errorf("bitmap index out of bounds")
		}
		if (bitmap[byteIdx] & (1 << bitIdx)) == 0 {
			return defaultValue, nil
		}
		var start int
		for j := 0; j < pos; j++ {
			byteIdx := j / 8
			bitIdx := j % 8
			if byteIdx >= len(bitmap) {
				return nil, fmt.Errorf("bitmap index out of bounds")
			}
			if (bitmap[byteIdx] & (1 << bitIdx)) != 0 {
				start += int(vectorLengths[j]) * size
			}
		}
		end := start + int(vectorLengths[pos])*size
		if end > len(dense) {
			return nil, fmt.Errorf("numeric vector dense offset out of bounds")
		}
		return dense[start:end], nil
	}

	return d.DeserializedPSDB.GetNumericVectorFeature(pos, vectorLengths, defaultValue)
}

func (d *DeserializedPSDBLayout2) GetBoolVectorFeature(pos int, vectorLengths []uint16, defaultValue []byte) ([]byte, error) {
	numVectors := len(vectorLengths)
	if pos < 0 || pos >= numVectors {
		return nil, fmt.Errorf("pos %d out of bounds for vectorLengths (len=%d)", pos, numVectors)
	}
	vectorLen := int(vectorLengths[pos])
	data := d.OriginalData

	if (d.BitmapMeta & bitmapPresentMask) != 0 {
		bitmapSize := (numVectors + 7) / 8
		if len(data) < bitmapSize {
			return nil, fmt.Errorf("corrupt bitmap payload")
		}
		bitmap := data[:bitmapSize]
		dense := data[bitmapSize:]
		byteIdx := pos / 8
		bitIdx := pos % 8
		if byteIdx >= len(bitmap) {
			return nil, fmt.Errorf("bitmap index out of bounds")
		}
		// Bitmap uses LSB-first ordering: vector i is present if bit (i%8) of byte (i/8) is set.
		if (bitmap[byteIdx] & (1 << bitIdx)) == 0 {
			return defaultValue, nil
		}
		var startByte int
		for j := 0; j < pos; j++ {
			byteIdx := j / 8
			bitIdx := j % 8
			if byteIdx >= len(bitmap) {
				return nil, fmt.Errorf("bitmap index out of bounds")
			}
			if (bitmap[byteIdx] & (1 << bitIdx)) != 0 {
				startByte += (int(vectorLengths[j]) + 7) / 8
			}
		}
		startBit := startByte * 8
		result := make([]byte, vectorLen)
		for i := 0; i < vectorLen; i++ {
			sourceBitPos := startBit + i
			sourceByteIndex := sourceBitPos / 8
			// Dense bool payload uses MSB-first ordering within each byte:
			// bit 0 of the logical stream is stored in the MSB (bit 7) of byte 0.
			sourceBitOffset := 7 - (sourceBitPos % 8)
			sourceBitMask := byte(1 << sourceBitOffset)
			if sourceByteIndex >= len(dense) {
				return nil, fmt.Errorf("bool vector dense out of bounds")
			}
			bitValue := (dense[sourceByteIndex] & sourceBitMask) >> sourceBitOffset
			result[i] = bitValue
		}
		return result, nil
	}

	return d.DeserializedPSDB.GetBoolVectorFeature(pos, vectorLengths, defaultValue)
}

// GetBoolScalarFeature — no bitmap handling for bool scalars, same as V1
// Inherited from DeserializedPSDB via embedding.

// --- V2 helper functions ---

func countSetBitsBefore(bitmap []byte, pos int, numFeatures int) int {
	count := 0
	for i := 0; i < pos; i++ {
		if i >= numFeatures {
			break
		}
		byteIdx := i / 8
		bitIdx := i % 8
		if byteIdx >= len(bitmap) {
			break // Stop counting if bitmap is exhausted
		}
		if (bitmap[byteIdx] & (1 << bitIdx)) != 0 {
			count++
		}
	}
	return count
}

func skipStringsInDense(dense []byte, skipCount int) (offset int, length uint16, err error) {
	for i := 0; i < skipCount; i++ {
		if offset+2 > len(dense) {
			return 0, 0, fmt.Errorf("dense string section out of bounds")
		}
		length = system.ByteOrder.Uint16(dense[offset : offset+2])
		offset += 2 + int(length)
		if offset > len(dense) {
			return 0, 0, fmt.Errorf("dense string section out of bounds")
		}
	}
	if offset+2 > len(dense) {
		return 0, 0, fmt.Errorf("dense string section out of bounds")
	}
	length = system.ByteOrder.Uint16(dense[offset : offset+2])
	return offset + 2, length, nil
}

func skipStringVectorsInDense(dense []byte, vectorLengths []uint16, bitmap []byte, pos int) (int, error) {
	offset := 0
	for j := 0; j < pos; j++ {
		byteIdx := j / 8
		bitIdx := j % 8
		if byteIdx >= len(bitmap) {
			return 0, fmt.Errorf("bitmap index out of bounds")
		}
		if (bitmap[byteIdx] & (1 << bitIdx)) == 0 {
			continue
		}
		for k := 0; k < int(vectorLengths[j]); k++ {
			if offset+2 > len(dense) {
				return 0, fmt.Errorf("string vector dense out of bounds")
			}
			length := system.ByteOrder.Uint16(dense[offset : offset+2])
			offset += 2 + int(length)
			if offset > len(dense) {
				return 0, fmt.Errorf("string vector dense out of bounds")
			}
		}
	}
	return offset, nil
}
