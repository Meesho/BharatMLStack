package utils

import (
	"encoding/binary"
	"fmt"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/types"
)

// FeatureEncoder encodes features into a compact binary format.
// The first byte indicates if any values were generated (1 = no generated, 0 = some generated).
type FeatureEncoder struct {
	buffer []byte
}

// NewFeatureEncoder creates a new FeatureEncoder with pre-allocated capacity.
func NewFeatureEncoder(estimatedSize int) *FeatureEncoder {
	encoder := &FeatureEncoder{
		buffer: make([]byte, 1, estimatedSize+1),
	}
	encoder.buffer[0] = 1 // Initialize flag to 1 (no generated values)
	return encoder
}

// MarkValueAsGenerated sets the flag indicating some values were auto-generated.
func (e *FeatureEncoder) MarkValueAsGenerated() {
	e.buffer[0] = 0
}

// AppendFeature appends a typed feature value to the buffer.
// For strings and vectors, includes a 2-byte size prefix.
func (e *FeatureEncoder) AppendFeature(dataType types.DataType, value []byte) (tag string, err error) {
	switch dataType {
	case types.DataTypeInt8:
		return e.appendScalar(value, 1)
	case types.DataTypeInt16:
		return e.appendScalar(value, 2)
	case types.DataTypeInt32:
		return e.appendScalar(value, 4)
	case types.DataTypeInt64:
		return e.appendScalar(value, 8)
	case types.DataTypeFP8E5M2, types.DataTypeFP8E4M3:
		return e.appendScalar(value, 1)
	case types.DataTypeFP16:
		return e.appendFP16Scalar(value)
	case types.DataTypeFP32:
		return e.appendScalar(value, 4)
	case types.DataTypeFP64:
		return e.appendScalar(value, 8)
	case types.DataTypeUint8:
		return e.appendScalar(value, 1)
	case types.DataTypeUint16:
		return e.appendScalar(value, 2)
	case types.DataTypeUint32:
		return e.appendScalar(value, 4)
	case types.DataTypeUint64:
		return e.appendScalar(value, 8)
	case types.DataTypeBool:
		return e.appendScalar(value, 1)
	case types.DataTypeString:
		return e.appendSizedValue(value)
	case types.DataTypeFP8E5M2Vector,
		types.DataTypeFP8E4M3Vector,
		types.DataTypeFP16Vector,
		types.DataTypeFP32Vector,
		types.DataTypeFP64Vector,
		types.DataTypeInt8Vector,
		types.DataTypeInt16Vector,
		types.DataTypeInt32Vector,
		types.DataTypeInt64Vector,
		types.DataTypeUint8Vector,
		types.DataTypeUint16Vector,
		types.DataTypeUint32Vector,
		types.DataTypeUint64Vector,
		types.DataTypeStringVector,
		types.DataTypeBoolVector:
		return e.appendSizedValue(value)
	default:
		return "invalid_data_type", fmt.Errorf("unsupported data type: %v", dataType)
	}
}

// AppendBytesFeature appends a raw bytes feature with size prefix.
func (e *FeatureEncoder) AppendBytesFeature(value []byte) (string, error) {
	return e.appendSizedValue(value)
}

func (e *FeatureEncoder) appendScalar(value []byte, expectedSize int) (string, error) {
	if len(value) != expectedSize {
		return "invalid_value_size", fmt.Errorf("invalid value size: expected %d, got %d", expectedSize, len(value))
	}
	e.buffer = append(e.buffer, value...)
	return "", nil
}

func (e *FeatureEncoder) appendFP16Scalar(value []byte) (string, error) {
	switch len(value) {
	case 2:
		e.buffer = append(e.buffer, value...)
	case 4:
		e.buffer = append(e.buffer, value[:2]...)
	default:
		return "invalid_value_size", fmt.Errorf("invalid FP16 value size: expected 2 or 4 bytes, got %d", len(value))
	}
	return "", nil
}

func (e *FeatureEncoder) appendSizedValue(value []byte) (string, error) {
	valueLen := len(value)
	if valueLen > 65535 {
		return "invalid_value_size", fmt.Errorf("value size %d exceeds maximum of 65535 bytes", valueLen)
	}

	sizeBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(sizeBytes, uint16(valueLen))
	e.buffer = append(e.buffer, sizeBytes...)
	e.buffer = append(e.buffer, value...)
	return "", nil
}

// Bytes returns the encoded byte buffer.
func (e *FeatureEncoder) Bytes() []byte {
	return e.buffer
}

// Reset clears the buffer for reuse.
func (e *FeatureEncoder) Reset() {
	e.buffer = e.buffer[:1]
	e.buffer[0] = 0
}
