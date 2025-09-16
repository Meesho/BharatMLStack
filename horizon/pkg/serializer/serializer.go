package serializer

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

func Float32ToBytesLE(value string) ([]byte, error) {
	f64, err := strconv.ParseFloat(value, 32)
	f32 := float32(f64)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	err = binary.Write(buf, binary.LittleEndian, f32)

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func BytesToFloat64LE(b []byte) (float64, error) {
	if len(b) != 8 {
		return 0, fmt.Errorf("invalid byte array length: expected 8 bytes, got %d", len(b))
	}
	var result float64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &result)
	return result, err
}

func Float64ToBytesLE(value string) ([]byte, error) {
	f64, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	err = binary.Write(buf, binary.LittleEndian, f64)

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func BytesToFloat32LE(b []byte) (float32, error) {
	if len(b) != 4 {
		return 0, fmt.Errorf("invalid byte array length: expected 4 bytes, got %d", len(b))
	}
	var result float32
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &result)
	return result, err
}

func BytesToInt32LE(b []byte) (int32, error) {
	if len(b) != 4 {
		return 0, fmt.Errorf("invalid byte array length: expected 4 bytes, got %d", len(b))
	}
	var result int32
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &result)
	return result, err
}

func BytesToInt64LE(b []byte) (int64, error) {
	if len(b) != 8 {
		return 0, fmt.Errorf("invalid byte array length: expected 8 bytes, got %d", len(b))
	}
	var result int64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &result)
	return result, err
}

func BytesToBoolLE(b []byte) (bool, error) {
	if len(b) != 1 {
		return false, fmt.Errorf("invalid byte array length: expected 1 byte, got %d", len(b))
	}
	var result bool
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &result)
	return result, err
}

func BytesToStringLE(b []byte) (string, error) {
	return string(b), nil
}

func ConvertToRawBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	switch v := data.(type) {
	case []float32:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	case []float64:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	case []int32:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	case []int64:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	case []uint32:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	case []uint64:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	case []bool:
		for _, val := range v {
			var b byte
			if val {
				b = 1
			} else {
				b = 0
			}
			if err := binary.Write(&buf, binary.LittleEndian, b); err != nil {
				return nil, err
			}
		}
	case []string:
		for _, val := range v {
			length := uint32(len(val))
			if err := binary.Write(&buf, binary.LittleEndian, length); err != nil {
				return nil, err
			}
			if _, err := buf.Write([]byte(val)); err != nil {
				return nil, err
			}
		}
	case [][]byte:
		for _, val := range v {
			if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.New("unsupported data type")
	}

	return buf.Bytes(), nil
}

func ConvertBytesToSlice[T any](data [][]byte) ([]T, error) {
	var result []T
	for _, b := range data {
		// Check if it's binary data that needs binary deserialization
		var item T
		if len(b) > 0 {
			switch any(item).(type) {
			case float32:
				if len(b) != 4 {
					return nil, fmt.Errorf("invalid byte length for float32: %d", len(b))
				}
				val, err := BytesToFloat32LE(b)
				if err != nil {
					return nil, err
				}
				item = any(val).(T)
			case float64:
				if len(b) != 8 {
					return nil, fmt.Errorf("invalid byte length for float64: %d", len(b))
				}
				val, err := BytesToFloat64LE(b)
				if err != nil {
					return nil, err
				}
				item = any(val).(T)
			case int32:
				if len(b) != 4 {
					return nil, fmt.Errorf("invalid byte length for int32: %d", len(b))
				}
				val, err := BytesToInt32LE(b)
				if err != nil {
					return nil, err
				}
				item = any(val).(T)
			case int64:
				if len(b) != 8 {
					return nil, fmt.Errorf("invalid byte length for int64: %d", len(b))
				}
				val, err := BytesToInt64LE(b)
				if err != nil {
					return nil, err
				}
				item = any(val).(T)
			case bool:
				if len(b) != 1 {
					return nil, fmt.Errorf("invalid byte length for bool: %d", len(b))
				}
				val, err := BytesToBoolLE(b)
				if err != nil {
					return nil, err
				}
				item = any(val).(T)
			case string:
				val, err := BytesToStringLE(b)
				if err != nil {
					return nil, err
				}
				item = any(val).(T)
			default:
				// Try JSON unmarshaling as fallback
				err := json.Unmarshal(b, &item)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal: %w", err)
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func Float16ToFloat32(bytes []byte) (float32, error) {
	if len(bytes) != 2 {
		return 0, fmt.Errorf("invalid byte array length: expected 2 bytes for float16, got %d", len(bytes))
	}

	// Convert 2 bytes to uint16
	bits := uint16(bytes[0]) | uint16(bytes[1])<<8

	// Extract components
	sign := bits >> 15
	exponent := (bits >> 10) & 0x1F
	mantissa := bits & 0x3FF

	// Convert to float32 components
	var f32bits uint32

	if exponent == 0 {
		// Zero or subnormal
		if mantissa == 0 {
			// Zero
			f32bits = uint32(sign) << 31
		} else {
			// Subnormal
			// Convert to float32 subnormal
			mantissa32 := uint32(mantissa) << 13
			f32bits = (uint32(sign) << 31) | mantissa32
		}
	} else if exponent == 0x1F {
		// Infinity or NaN
		if mantissa == 0 {
			// Infinity
			f32bits = (uint32(sign) << 31) | 0x7F800000
		} else {
			// NaN
			f32bits = (uint32(sign) << 31) | 0x7F800000 | (uint32(mantissa) << 13)
		}
	} else {
		// Normal number
		// Adjust exponent bias (float16 has bias 15, float32 has bias 127)
		exponent32 := int32(exponent) - 15 + 127

		// Handle possible overflow/underflow in exponent conversion
		if exponent32 <= 0 {
			// Underflow to 0
			f32bits = uint32(sign) << 31
		} else if exponent32 >= 255 {
			// Overflow to infinity
			f32bits = (uint32(sign) << 31) | 0x7F800000
		} else {
			// Normal conversion
			f32bits = (uint32(sign) << 31) | (uint32(exponent32) << 23) | (uint32(mantissa) << 13)
		}
	}

	// Convert bits to float32
	result := math.Float32frombits(f32bits)
	return result, nil
}

func Float32ToFloat16Bytes(value float32) ([]byte, error) {
	// Get float32 bits
	f32bits := math.Float32bits(value)

	// Extract components
	sign := f32bits >> 31
	exponent := (f32bits >> 23) & 0xFF
	mantissa := f32bits & 0x7FFFFF

	// Convert to float16 components
	var f16bits uint16

	if exponent == 0 {
		// Zero or subnormal
		if mantissa == 0 {
			// Zero
			f16bits = uint16(sign) << 15
		} else {
			// Subnormal float32 - will be zero in float16
			f16bits = uint16(sign) << 15
		}
	} else if exponent == 0xFF {
		// Infinity or NaN
		if mantissa == 0 {
			// Infinity
			f16bits = (uint16(sign) << 15) | 0x7C00
		} else {
			// NaN
			f16bits = (uint16(sign) << 15) | 0x7C00 | 0x3FF
		}
	} else {
		// Normal number
		// Adjust exponent bias (float32 has bias 127, float16 has bias 15)
		exponent16 := int32(exponent) - 127 + 15

		// Handle overflow/underflow
		if exponent16 <= 0 {
			// Underflow to 0
			f16bits = uint16(sign) << 15
		} else if exponent16 >= 31 {
			// Overflow to infinity
			f16bits = (uint16(sign) << 15) | 0x7C00
		} else {
			// Get 10 most significant bits of mantissa
			mantissa16 := (mantissa >> 13) & 0x3FF

			// Normal conversion
			f16bits = (uint16(sign) << 15) | (uint16(exponent16) << 10) | uint16(mantissa16)
		}
	}

	// Convert to bytes
	bytes := make([]byte, 2)
	bytes[0] = byte(f16bits)
	bytes[1] = byte(f16bits >> 8)

	return bytes, nil
}

func convertFloat16BytesToFloat32Slice(data [][]byte) ([]float32, error) {
	result := make([]float32, 0, len(data))
	for _, bytes := range data {
		if len(bytes) != 2 {
			return nil, fmt.Errorf("invalid byte length for float16: %d", len(bytes))
		}
		f32, err := Float16ToFloat32(bytes)
		if err != nil {
			return nil, err
		}
		result = append(result, f32)
	}
	return result, nil
}

func ConvertFromRawBytes(data [][]byte, dataType string) (any, error) {
	switch dataType {
	case "TYPE_FP32", "FP32":
		convertedData, err := ConvertBytesToSlice[float32](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_FP64", "FP64":
		convertedData, err := ConvertBytesToSlice[float64](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_FP16", "FP16":
		// Use dedicated function for float16 conversion
		convertedData, err := convertFloat16BytesToFloat32Slice(data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_INT8", "INT8":
		convertedData, err := ConvertBytesToSlice[int8](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_INT16", "INT16":
		convertedData, err := ConvertBytesToSlice[int16](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_INT32", "INT32":
		convertedData, err := ConvertBytesToSlice[int32](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_INT64", "INT64":
		convertedData, err := ConvertBytesToSlice[int64](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_UINT8", "UINT8":
		convertedData, err := ConvertBytesToSlice[uint8](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_UINT16", "UINT16":
		convertedData, err := ConvertBytesToSlice[uint16](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_UINT32", "UINT32":
		convertedData, err := ConvertBytesToSlice[uint32](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_UINT64", "UINT64":
		convertedData, err := ConvertBytesToSlice[uint64](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_BOOL", "BOOL":
		convertedData, err := ConvertBytesToSlice[bool](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_STRING", "STRING":
		convertedData, err := ConvertBytesToSlice[string](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	case "TYPE_BYTES", "BYTES":
		convertedData, err := ConvertBytesToSlice[[]byte](data)
		if err != nil {
			return nil, err
		}
		return convertedData, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

func FlattenMatrix[T interface{}](matrix interface{}) ([]T, error) {
	if matrix == nil {
		return []T{}, nil
	}

	val := reflect.ValueOf(matrix)

	// Handle non-slice input
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		// Try to convert single value directly to T
		singleValue := val.Interface()
		if typed, ok := singleValue.(T); ok {
			return []T{typed}, nil
		}
		targetType := reflect.TypeOf((*T)(nil)).Elem()
		if reflect.TypeOf(singleValue).ConvertibleTo(targetType) {
			converted := reflect.ValueOf(singleValue).Convert(targetType).Interface().(T)
			return []T{converted}, nil
		}
		return nil, fmt.Errorf("cannot convert single value %T to %s", singleValue, targetType)
	}

	// For simple 1D slices of type T
	if val.Type().Elem() == reflect.TypeOf((*T)(nil)).Elem() {
		result := make([]T, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = val.Index(i).Interface().(T)
		}
		return result, nil
	}

	var result []T

	// Recursive flattening function
	var flattenRec func(v reflect.Value) error

	flattenRec = func(v reflect.Value) error {
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			// Leaf node - try to convert to T
			leafValue := v.Interface()
			if typed, ok := leafValue.(T); ok {
				result = append(result, typed)
				return nil
			}

			targetType := reflect.TypeOf((*T)(nil)).Elem()
			if reflect.TypeOf(leafValue).ConvertibleTo(targetType) {
				converted := reflect.ValueOf(leafValue).Convert(targetType).Interface().(T)
				result = append(result, converted)
				return nil
			}

			return fmt.Errorf("cannot convert leaf value %T to %s", leafValue, targetType)
		}

		// Handle nested slices/arrays
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)

			// If we have an interface{} containing another slice/array, we need to unwrap it
			if elem.Kind() == reflect.Interface {
				elem = elem.Elem()
			}

			// If it's a slice/array, recursively process it
			if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array {
				if err := flattenRec(elem); err != nil {
					return err
				}
			} else {
				// It's a leaf value, try to convert
				leafValue := elem.Interface()
				if typed, ok := leafValue.(T); ok {
					result = append(result, typed)
				} else {
					targetType := reflect.TypeOf((*T)(nil)).Elem()
					if reflect.TypeOf(leafValue).ConvertibleTo(targetType) {
						converted := reflect.ValueOf(leafValue).Convert(targetType).Interface().(T)
						result = append(result, converted)
					} else {
						return fmt.Errorf("cannot convert array element %T to %s", leafValue, targetType)
					}
				}
			}
		}
		return nil
	}

	if err := flattenRec(val); err != nil {
		return nil, err
	}

	return result, nil
}

// FlattenMatrixByType flattens a nested matrix into a 1D slice of the appropriate type based on dataType
func FlattenMatrixByType(matrix interface{}, dataType string) (interface{}, error) {
	switch dataType {
	case "TYPE_FP16", "FP16":
		return FlattenMatrix[float32](matrix) // Using float32 as Go doesn't have native float16
	case "TYPE_FP32", "FP32":
		return FlattenMatrix[float32](matrix)
	case "TYPE_FP64", "FP64":
		return FlattenMatrix[float64](matrix)
	case "TYPE_INT8", "INT8":
		return FlattenMatrix[int8](matrix)
	case "TYPE_INT16", "INT16":
		return FlattenMatrix[int16](matrix)
	case "TYPE_INT32", "INT32":
		return FlattenMatrix[int32](matrix)
	case "TYPE_INT64", "INT64":
		return FlattenMatrix[int64](matrix)
	case "TYPE_UINT8", "UINT8":
		return FlattenMatrix[uint8](matrix)
	case "TYPE_UINT16", "UINT16":
		return FlattenMatrix[uint16](matrix)
	case "TYPE_UINT32", "UINT32":
		return FlattenMatrix[uint32](matrix)
	case "TYPE_UINT64", "UINT64":
		return FlattenMatrix[uint64](matrix)
	case "TYPE_BOOL", "BOOL":
		return FlattenMatrix[bool](matrix)
	case "TYPE_STRING", "STRING":
		return FlattenMatrix[string](matrix)
	case "TYPE_BYTES", "BYTES":
		return FlattenMatrix[[]byte](matrix)
	default:
		return nil, fmt.Errorf("unsupported data type for flattening: %s", dataType)
	}
}
