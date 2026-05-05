package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/types"
)

// StringToDataType converts a string representation of a data type to types.DataType.
// It adds "DataType" prefix if not already present.
func StringToDataType(dataTypeStr string) (types.DataType, error) {
	s := strings.TrimSpace(dataTypeStr)
	if !strings.HasPrefix(s, "DataType") {
		s = "DataType" + s
	}
	return types.ParseDataType(s)
}

// IsBytesDataType checks if the given data type string represents a BYTES type.
func IsBytesDataType(dataTypeStr string) bool {
	s := strings.TrimSpace(dataTypeStr)
	if !strings.HasPrefix(s, "DataType") {
		s = "DataType" + s
	}
	return strings.EqualFold(s, "DataTypeBYTES")
}

// ConvertStringToType converts a string value to its byte representation
// based on the specified data type.
func ConvertStringToType(value string, dataType types.DataType) ([]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("empty string provided for conversion")
	}

	switch dataType {
	case types.DataTypeUnknown:
		return nil, nil

	case types.DataTypeString:
		return []byte(value), nil

	case types.DataTypeInt8:
		val, err := strconv.ParseInt(value, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to parse INT8: %w", err)
		}
		return []byte{byte(int8(val))}, nil

	case types.DataTypeInt16:
		val, err := strconv.ParseInt(value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse INT16: %w", err)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(val))
		return buf, nil

	case types.DataTypeInt32:
		val, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse INT32: %w", err)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(val))
		return buf, nil

	case types.DataTypeInt64:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse INT64: %w", err)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(val))
		return buf, nil

	case types.DataTypeUint8:
		val, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UINT8: %w", err)
		}
		return []byte{byte(val)}, nil

	case types.DataTypeUint16:
		val, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UINT16: %w", err)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(val))
		return buf, nil

	case types.DataTypeUint32:
		val, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UINT32: %w", err)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(val))
		return buf, nil

	case types.DataTypeUint64:
		val, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UINT64: %w", err)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, val)
		return buf, nil

	case types.DataTypeFP16:
		val, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FP16: %w", err)
		}
		bits := math.Float32bits(float32(val))
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(bits>>16))
		return buf, nil

	case types.DataTypeFP32:
		val, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FP32: %w", err)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, math.Float32bits(float32(val)))
		return buf, nil

	case types.DataTypeFP64:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FP64: %w", err)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(val))
		return buf, nil

	case types.DataTypeBool:
		val, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BOOL: %w", err)
		}
		if val {
			return []byte{1}, nil
		}
		return []byte{0}, nil

	case types.DataTypeStringVector:
		var result []string
		err := json.Unmarshal([]byte(value), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse StringVector: %w", err)
		}
		return json.Marshal(result)

	case types.DataTypeFP32Vector:
		var result []float32
		err := json.Unmarshal([]byte(value), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FP32Vector: %w", err)
		}
		return json.Marshal(result)

	case types.DataTypeFP64Vector:
		var result []float64
		err := json.Unmarshal([]byte(value), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FP64Vector: %w", err)
		}
		return json.Marshal(result)

	case types.DataTypeInt32Vector:
		var result []int32
		err := json.Unmarshal([]byte(value), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Int32Vector: %w", err)
		}
		return json.Marshal(result)

	case types.DataTypeInt64Vector:
		var result []int64
		err := json.Unmarshal([]byte(value), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Int64Vector: %w", err)
		}
		return json.Marshal(result)

	default:
		return nil, fmt.Errorf("unsupported data type: %v", dataType)
	}
}
