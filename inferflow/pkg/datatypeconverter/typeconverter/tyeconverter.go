package typeconverter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/byteorder"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/types"
	"github.com/h2so5/half"
)

// BytesToString converts byte data to string based on the specified data type
func BytesToString(data []byte, dataType string) (string, error) {
	if data == nil {
		return "", errors.New("data is nil")
	}

	dt := strings.ToLower(dataType)

	// Handle scalar types
	switch dt {
	case "string", "datatypestring", "bytes":
		return string(data), nil
	case "bool", "datatypebool":
		return boolBytesToString(data)
	case "int8", "datatypeint8":
		return int8BytesToString(data)
	case "int16", "datatypeint16":
		return int16BytesToString(data)
	case "int32", "datatypeint32":
		return int32BytesToString(data)
	case "int64", "datatypeint64":
		return int64BytesToString(data)
	case "uint8", "datatypeuint8":
		return uint8BytesToString(data)
	case "uint16", "datatypeuint16":
		return uint16BytesToString(data)
	case "uint32", "datatypeuint32":
		return uint32BytesToString(data)
	case "uint64", "datatypeuint64":
		return uint64BytesToString(data)
	case "fp32", "datatypefp32":
		return float32BytesToString(data)
	case "fp64", "datatypefp64":
		return float64BytesToString(data)
	case "fp16", "datatypefp16":
		return float16BytesToString(data)
	case "fp8e5m2", "datatypefp8e5m2":
		return float8E5M2BytesToString(data)
	case "fp8e4m3", "datatypefp8e4m3":
		return float8E4M3BytesToString(data)

	// Handle vector types
	case "boolvector", "datatypeboolvector":
		return boolVectorBytesToString(data)
	case "int8vector", "datatypeint8vector":
		return int8VectorBytesToString(data)
	case "int16vector", "datatypeint16vector":
		return int16VectorBytesToString(data)
	case "int32vector", "datatypeint32vector":
		return int32VectorBytesToString(data)
	case "int64vector", "datatypeint64vector":
		return int64VectorBytesToString(data)
	case "uint8vector", "datatypeuint8vector":
		return uint8VectorBytesToString(data)
	case "uint16vector", "datatypeuint16vector":
		return uint16VectorBytesToString(data)
	case "uint32vector", "datatypeuint32vector":
		return uint32VectorBytesToString(data)
	case "uint64vector", "datatypeuint64vector":
		return uint64VectorBytesToString(data)
	case "fp32vector", "datatypefp32vector":
		return float32VectorBytesToString(data)
	case "fp64vector", "datatypefp64vector":
		return float64VectorBytesToString(data)
	case "fp16vector", "datatypefp16vector":
		return float16VectorBytesToString(data)
	case "fp8e5m2vector", "datatypefp8e5m2vector":
		return float8E5M2VectorBytesToString(data)
	case "fp8e4m3vector", "datatypefp8e4m3vector":
		return float8E4M3VectorBytesToString(data)
	case "stringvector", "datatypestringvector":
		return string(data), nil
	}

	return "", errors.New("unsupported data type: " + dataType)
}

// Scalar converters - convert byte data to string representation
func boolBytesToString(data []byte) (string, error) {
	if len(data) != 1 {
		return "", errors.New("invalid byte length for bool")
	}
	return strconv.FormatBool(data[0] != 0), nil
}

func int8BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt8.Size() {
		return "", errors.New("invalid byte length for int8")
	}
	// return strconv.FormatInt(int64(byteorder.ByteOrder.Int8(data)), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Int8(data)), nil
}

func int16BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt16.Size() {
		return "", errors.New("invalid byte length for int16")
	}
	// return strconv.FormatInt(int64(byteorder.ByteOrder.Int16(data)), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Int16(data)), nil
}

func int32BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt32.Size() {
		return "", errors.New("invalid byte length for int32")
	}
	// return strconv.FormatInt(int64(byteorder.ByteOrder.Int32(data)), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Int32(data)), nil
}

func int64BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt64.Size() {
		return "", errors.New("invalid byte length for int64")
	}
	// return strconv.FormatInt(byteorder.ByteOrder.Int64(data), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Int64(data)), nil
}

func uint8BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint8.Size() {
		return "", errors.New("invalid byte length for uint8")
	}
	// return strconv.FormatUint(uint64(data[0]), 10), nil
	return fmt.Sprint(uint64(data[0])), nil
}

func uint16BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint16.Size() {
		return "", errors.New("invalid byte length for uint16")
	}
	// return strconv.FormatUint(uint64(byteorder.ByteOrder.Uint16(data)), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Uint16(data)), nil
}

func uint32BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint32.Size() {
		return "", errors.New("invalid byte length for uint32")
	}
	// return strconv.FormatUint(uint64(byteorder.ByteOrder.Uint32(data)), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Uint32(data)), nil
}

func uint64BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint64.Size() {
		return "", errors.New("invalid byte length for uint64")
	}
	// return strconv.FormatUint(byteorder.ByteOrder.Uint64(data), 10), nil
	return fmt.Sprint(byteorder.ByteOrder.Uint64(data)), nil
}

func float32BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP32.Size() {
		return "", errors.New("invalid byte length for float32")
	}
	return fmt.Sprint(byteorder.ByteOrder.Float32(data)), nil
	// return strconv.FormatFloat(float64(byteorder.ByteOrder.Float32(data)), 'f', -1, 32), nil
}

func float64BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP64.Size() {
		return "", errors.New("invalid byte length for float64")
	}
	return fmt.Sprint(byteorder.ByteOrder.Float64(data)), nil
	// return strconv.FormatFloat(byteorder.ByteOrder.Float64(data), 'f', -1, 64), nil
}

func float16BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP16.Size() {
		return "", errors.New("invalid byte length for float16")
	}
	fp32 := byteorder.ByteOrder.Float16AsFP32(data)
	return fmt.Sprint(fp32), nil
	// return strconv.FormatFloat(float64(fp32), 'f', -1, 32), nil
}

func float8E5M2BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP8E5M2.Size() {
		return "", errors.New("invalid byte length for float8 E5M2")
	}
	fp32 := byteorder.ByteOrder.Float8E5M2AsFP32(data)
	return fmt.Sprint(fp32), nil
	// return strconv.FormatFloat(float64(fp32), 'f', -1, 32), nil
}

func float8E4M3BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP8E4M3.Size() {
		return "", errors.New("invalid byte length for float8 E4M3")
	}
	fp32 := byteorder.ByteOrder.Float8E4M3AsFP32(data)
	return fmt.Sprint(fp32), nil
	// return strconv.FormatFloat(float64(fp32), 'f', -1, 32), nil
}

// Vector converters - convert byte data to comma-separated string representation
func vectorBytesToString[T any](data []byte, unitSize int, convertFunc func([]byte) T) (string, error) {
	if len(data)%unitSize != 0 {
		return "", errors.New("invalid byte length for vector data")
	}

	var values []string
	for i := 0; i < len(data); i += unitSize {
		val := convertFunc(data[i : i+unitSize])
		values = append(values, fmt.Sprint(val))
	}
	return strings.Join(values, ","), nil
}

func formatValue(val interface{}) string {
	switch v := val.(type) {
	case bool:
		return strconv.FormatBool(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func boolVectorBytesToString(data []byte) (string, error) {
	var values []string
	for _, b := range data {
		values = append(values, strconv.FormatBool(b != 0))
	}
	return strings.Join(values, ","), nil
}

func int8VectorBytesToString(data []byte) (string, error) {
	var values []string
	for _, b := range data {
		values = append(values, strconv.FormatInt(int64(int8(b)), 10))
	}
	return strings.Join(values, ","), nil
}

func int16VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeInt16.Size(), func(d []byte) int16 {
		return byteorder.ByteOrder.Int16(d)
	})
}

func int32VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeInt32.Size(), func(d []byte) int32 {
		return byteorder.ByteOrder.Int32(d)
	})
}

func int64VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeInt64.Size(), func(d []byte) int64 {
		return byteorder.ByteOrder.Int64(d)
	})
}

func uint8VectorBytesToString(data []byte) (string, error) {
	var values []string
	for _, b := range data {
		values = append(values, strconv.FormatUint(uint64(b), 10))
	}
	return strings.Join(values, ","), nil
}

func uint16VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeUint16.Size(), func(d []byte) uint16 {
		return byteorder.ByteOrder.Uint16(d)
	})
}

func uint32VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeUint32.Size(), func(d []byte) uint32 {
		return byteorder.ByteOrder.Uint32(d)
	})
}

func uint64VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeUint64.Size(), func(d []byte) uint64 {
		return byteorder.ByteOrder.Uint64(d)
	})
}

func float32VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeFP32.Size(), func(d []byte) float32 {
		return byteorder.ByteOrder.Float32(d)
	})
}

func float64VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeFP64.Size(), func(d []byte) float64 {
		return byteorder.ByteOrder.Float64(d)
	})
}

func float16VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeFP16.Size(), func(d []byte) float32 {
		return byteorder.ByteOrder.Float16AsFP32(d)
	})
}

func float8E5M2VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeFP8E5M2.Size(), func(d []byte) float32 {
		return byteorder.ByteOrder.Float8E5M2AsFP32(d)
	})
}

func float8E4M3VectorBytesToString(data []byte) (string, error) {
	return vectorBytesToString(data, types.DataTypeFP8E4M3.Size(), func(d []byte) float32 {
		return byteorder.ByteOrder.Float8E4M3AsFP32(d)
	})
}

// =======================
// STRING TO BYTES CONVERTERS
// =======================

// StringToBytes converts string data to byte representation based on data type
func StringToBytes(value string, dataType string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("value is empty")
	}

	dt := strings.ToLower(dataType)
	isVector := strings.Contains(dt, "vector")

	if !isVector {
		// Handle scalar types
		switch dt {
		case "string", "datatypestring", "bytes":
			return []byte(value), nil
		case "bool", "datatypebool":
			return stringToBoolBytes(value)
		case "int8", "datatypeint8":
			return stringToInt8Bytes(value)
		case "int16", "datatypeint16":
			return stringToInt16Bytes(value)
		case "int32", "datatypeint32":
			return stringToInt32Bytes(value)
		case "int64", "datatypeint64":
			return stringToInt64Bytes(value)
		case "uint8", "datatypeuint8":
			return stringToUint8Bytes(value)
		case "uint16", "datatypeuint16":
			return stringToUint16Bytes(value)
		case "uint32", "datatypeuint32":
			return stringToUint32Bytes(value)
		case "uint64", "datatypeuint64":
			return stringToUint64Bytes(value)
		case "fp32", "datatypefp32":
			return stringToFloat32Bytes(value)
		case "fp64", "datatypefp64":
			return stringToFloat64Bytes(value)
		case "fp16", "datatypefp16":
			return stringToFloat16Bytes(value)
		case "fp8e5m2", "datatypefp8e5m2":
			return stringToFloat8E5M2Bytes(value)
		case "fp8e4m3", "datatypefp8e4m3":
			return stringToFloat8E4M3Bytes(value)
		}
	} else {
		// Handle vector types
		switch dt {
		case "boolvector", "datatypeboolvector":
			return stringToBoolVectorBytes(value)
		case "int8vector", "datatypeint8vector":
			return stringToInt8VectorBytes(value)
		case "int16vector", "datatypeint16vector":
			return stringToInt16VectorBytes(value)
		case "int32vector", "datatypeint32vector":
			return stringToInt32VectorBytes(value)
		case "int64vector", "datatypeint64vector":
			return stringToInt64VectorBytes(value)
		case "uint8vector", "datatypeuint8vector":
			return stringToUint8VectorBytes(value)
		case "uint16vector", "datatypeuint16vector":
			return stringToUint16VectorBytes(value)
		case "uint32vector", "datatypeuint32vector":
			return stringToUint32VectorBytes(value)
		case "uint64vector", "datatypeuint64vector":
			return stringToUint64VectorBytes(value)
		case "fp32vector", "datatypefp32vector":
			return stringToFloat32VectorBytes(value)
		case "fp64vector", "datatypefp64vector":
			return stringToFloat64VectorBytes(value)
		case "fp16vector", "datatypefp16vector":
			return stringToFloat16VectorBytes(value)
		case "fp8e5m2vector", "datatypefp8e5m2vector":
			return stringToFloat8E5M2VectorBytes(value)
		case "fp8e4m3vector", "datatypefp8e4m3vector":
			return stringToFloat8E4M3VectorBytes(value)
		case "stringvector", "datatypestringvector":
			return stringToStringVectorBytes(value)
		}
	}

	return nil, errors.New("unsupported data type: " + dataType)
}

// Scalar string to bytes converters
func stringToBoolBytes(value string) ([]byte, error) {
	switch strings.ToLower(value) {
	case "true":
		return []byte{1}, nil
	case "false":
		return []byte{0}, nil
	default:
		return nil, errors.New("invalid bool value: " + value)
	}
}

func stringToInt8Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return nil, errors.New("failed to parse int8: " + value)
	}
	return []byte{byte(int8(i))}, nil
}

func stringToInt16Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return nil, errors.New("failed to parse int16: " + value)
	}
	buf := make([]byte, 2)
	byteorder.ByteOrder.PutUint16(buf, uint16(i))
	return buf, nil
}

func stringToInt32Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, errors.New("failed to parse int32: " + value)
	}
	buf := make([]byte, 4)
	byteorder.ByteOrder.PutUint32(buf, uint32(i))
	return buf, nil
}

func stringToInt64Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, errors.New("failed to parse int64: " + value)
	}
	buf := make([]byte, 8)
	byteorder.ByteOrder.PutUint64(buf, uint64(i))
	return buf, nil
}

func stringToUint8Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return nil, errors.New("failed to parse uint8: " + value)
	}
	return []byte{byte(i)}, nil
}

func stringToUint16Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return nil, errors.New("failed to parse uint16: " + value)
	}
	buf := make([]byte, 2)
	byteorder.ByteOrder.PutUint16(buf, uint16(i))
	return buf, nil
}

func stringToUint32Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return nil, errors.New("failed to parse uint32: " + value)
	}
	buf := make([]byte, 4)
	byteorder.ByteOrder.PutUint32(buf, uint32(i))
	return buf, nil
}

func stringToUint64Bytes(value string) ([]byte, error) {
	i, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, errors.New("failed to parse uint64: " + value)
	}
	buf := make([]byte, 8)
	byteorder.ByteOrder.PutUint64(buf, i)
	return buf, nil
}

func stringToFloat32Bytes(value string) ([]byte, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, errors.New("failed to parse float32: " + value)
	}
	buf := make([]byte, 4)
	byteorder.ByteOrder.PutFloat32(buf, float32(f))
	return buf, nil
}

func stringToFloat64Bytes(value string) ([]byte, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, errors.New("failed to parse float64: " + value)
	}
	buf := make([]byte, 8)
	byteorder.ByteOrder.PutFloat64(buf, f)
	return buf, nil
}

func stringToFloat16Bytes(value string) ([]byte, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, errors.New("failed to parse float16: " + value)
	}
	buf := make([]byte, 2)
	byteorder.ByteOrder.PutFloat16FromFP32(buf, float32(f))
	return buf, nil
}

func stringToFloat8E5M2Bytes(value string) ([]byte, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, errors.New("failed to parse fp8e5m2: " + value)
	}
	buf := make([]byte, 1)
	byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf, float32(f))
	return buf, nil
}

func stringToFloat8E4M3Bytes(value string) ([]byte, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, errors.New("failed to parse fp8e4m3: " + value)
	}
	buf := make([]byte, 1)
	byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf, float32(f))
	return buf, nil
}

// Vector string to bytes converters
func stringToBoolVectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, len(vals))
	for i, v := range vals {
		switch strings.ToLower(v) {
		case "true":
			buf[i] = 1
		case "false":
			buf[i] = 0
		default:
			return nil, errors.New("invalid bool string in vector: " + v)
		}
	}
	return buf, nil
}

func stringToInt8VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, len(vals))
	for i, v := range vals {
		n, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return nil, errors.New("failed to parse int8 vector element: " + v)
		}
		buf[i] = byte(int8(n))
	}
	return buf, nil
}

func stringToInt16VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*2)
	for _, v := range vals {
		n, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return nil, errors.New("failed to parse int16 vector element: " + v)
		}
		b := make([]byte, 2)
		byteorder.ByteOrder.PutUint16(b, uint16(n))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToInt32VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*4)
	for _, v := range vals {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, errors.New("failed to parse int32 vector element: " + v)
		}
		b := make([]byte, 4)
		byteorder.ByteOrder.PutUint32(b, uint32(n))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToInt64VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*8)
	for _, v := range vals {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.New("failed to parse int64 vector element: " + v)
		}
		b := make([]byte, 8)
		byteorder.ByteOrder.PutUint64(b, uint64(n))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToUint8VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, len(vals))
	for i, v := range vals {
		n, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return nil, errors.New("failed to parse uint8 vector element: " + v)
		}
		buf[i] = byte(n)
	}
	return buf, nil
}

func stringToUint16VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*2)
	for _, v := range vals {
		n, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return nil, errors.New("failed to parse uint16 vector element: " + v)
		}
		b := make([]byte, 2)
		byteorder.ByteOrder.PutUint16(b, uint16(n))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToUint32VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*4)
	for _, v := range vals {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, errors.New("failed to parse uint32 vector element: " + v)
		}
		b := make([]byte, 4)
		byteorder.ByteOrder.PutUint32(b, uint32(n))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToUint64VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*8)
	for _, v := range vals {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, errors.New("failed to parse uint64 vector element: " + v)
		}
		b := make([]byte, 8)
		byteorder.ByteOrder.PutUint64(b, n)
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToFloat32VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*4)
	for _, v := range vals {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, errors.New("failed to parse float32 vector element: " + v)
		}
		b := make([]byte, 4)
		byteorder.ByteOrder.PutFloat32(b, float32(f))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToFloat64VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*8)
	for _, v := range vals {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, errors.New("failed to parse float64 vector element: " + v)
		}
		b := make([]byte, 8)
		byteorder.ByteOrder.PutFloat64(b, f)
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToFloat16VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, 0, len(vals)*2)
	for _, v := range vals {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, errors.New("failed to parse float16 vector element: " + v)
		}
		b := make([]byte, 2)
		byteorder.ByteOrder.PutFloat16FromFP32(b, float32(f))
		buf = append(buf, b...)
	}
	return buf, nil
}

func stringToFloat8E5M2VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, len(vals))
	for i, v := range vals {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, errors.New("failed to parse fp8e5m2 vector element: " + v)
		}
		byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i:i+1], float32(f))
	}
	return buf, nil
}

func stringToFloat8E4M3VectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	buf := make([]byte, len(vals))
	for i, v := range vals {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, errors.New("failed to parse fp8e4m3 vector element: " + v)
		}
		byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i:i+1], float32(f))
	}
	return buf, nil
}

func stringToStringVectorBytes(value string) ([]byte, error) {
	vals := strings.Split(value, ":")
	// Join with comma as separator
	return []byte(strings.Join(vals, ",")), nil
}

// =======================
// BYTES TO BYTES CONVERTERS
// =======================

// ConvertBytesToBytes converts byte data from one type to another
func ConvertBytesToBytes(inputData []byte, inputType string, outputType string) ([]byte, error) {
	if inputData == nil {
		return nil, errors.New("input data is nil")
	}

	inputType = strings.ToLower(inputType)
	outputType = strings.ToLower(outputType)

	// If input and output types are the same, return input data as-is
	if inputType == outputType {
		return inputData, nil
	}

	// First convert input bytes to native Go type
	nativeValue, err := bytesToNativeType(inputData, inputType)
	if err != nil {
		return nil, errors.New("failed to convert input bytes: " + err.Error())
	}

	// Then convert native type to output bytes
	outputData, err := nativeTypeToBytes(nativeValue, outputType)
	if err != nil {
		return nil, errors.New("failed to convert to output bytes: " + err.Error())
	}

	return outputData, nil
}

func ConvertVectorBytesToBytesLessGcOverhead(inputData []byte, inputType string, outputType string) ([]byte, error) {
	if inputData == nil {
		return nil, errors.New("input data is nil")
	}

	inputType = strings.ToLower(inputType)
	outputType = strings.ToLower(outputType)

	if inputType == outputType {
		return inputData, nil
	}

	nativeSlice, err := bytesToNativeTypeList(inputData, inputType)
	if err != nil {
		return nil, errors.New("failed to convert input byte vector to native slice: " + err.Error())
	}

	outputData, err := nativeListToBytes(nativeSlice, outputType)
	if err != nil {
		return nil, errors.New("failed to convert native slice: " + err.Error())
	}

	return outputData, nil
}

func bytesToNativeTypeList(data []byte, dataType string) (interface{}, error) {
	var elemSize int
	switch dataType {
	case "bool", "datatypebool", "int8", "datatypeint8", "uint8", "datatypeuint8",
		"fp8e5m2", "datatypefp8e5m2", "fp8e4m3", "datatypefp8e4m3":
		elemSize = 1
	case "int16", "datatypeint16", "uint16", "datatypeuint16", "fp16", "datatypefp16":
		elemSize = 2
	case "int32", "datatypeint32", "uint32", "datatypeuint32", "fp32", "datatypefp32":
		elemSize = 4
	case "int64", "datatypeint64", "uint64", "datatypeuint64", "fp64", "datatypefp64":
		elemSize = 8
	case "string", "datatypestring":
		return nil, errors.New("batch processing of strings not supported without length info")
	default:
		return nil, errors.New("unsupported input data type: " + dataType)
	}

	if len(data)%elemSize != 0 {
		return nil, fmt.Errorf("data length (%d) not multiple of element size (%d)", len(data), elemSize)
	}

	count := len(data) / elemSize

	switch dataType {
	case "bool", "datatypebool":
		result := make([]bool, count)
		for i := 0; i < count; i++ {
			result[i] = data[i] != 0
		}
		return result, nil

	case "int8", "datatypeint8":
		result := make([]int8, count)
		for i := 0; i < count; i++ {
			result[i] = int8(data[i])
		}
		return result, nil

	case "uint8", "datatypeuint8":
		result := make([]uint8, count)
		copy(result, data)
		return result, nil

	case "int16", "datatypeint16":
		result := make([]int16, count)
		for i := 0; i < count; i++ {
			start := i * 2
			val := byteorder.ByteOrder.Uint16(data[start : start+2])
			result[i] = int16(val)
		}
		return result, nil

	case "uint16", "datatypeuint16":
		result := make([]uint16, count)
		for i := 0; i < count; i++ {
			start := i * 2
			result[i] = byteorder.ByteOrder.Uint16(data[start : start+2])
		}
		return result, nil

	case "int32", "datatypeint32":
		result := make([]int32, count)
		for i := 0; i < count; i++ {
			start := i * 4
			val := byteorder.ByteOrder.Uint32(data[start : start+4])
			result[i] = int32(val)
		}
		return result, nil

	case "uint32", "datatypeuint32":
		result := make([]uint32, count)
		for i := 0; i < count; i++ {
			start := i * 4
			result[i] = byteorder.ByteOrder.Uint32(data[start : start+4])
		}
		return result, nil

	case "int64", "datatypeint64":
		result := make([]int64, count)
		for i := 0; i < count; i++ {
			start := i * 8
			val := byteorder.ByteOrder.Uint64(data[start : start+8])
			result[i] = int64(val)
		}
		return result, nil

	case "uint64", "datatypeuint64":
		result := make([]uint64, count)
		for i := 0; i < count; i++ {
			start := i * 8
			result[i] = byteorder.ByteOrder.Uint64(data[start : start+8])
		}
		return result, nil

	case "fp32", "datatypefp32":
		result := make([]float32, count)
		for i := 0; i < count; i++ {
			start := i * 4
			bits := byteorder.ByteOrder.Uint32(data[start : start+4])
			result[i] = math.Float32frombits(bits)
		}
		return result, nil

	case "fp64", "datatypefp64":
		result := make([]float64, count)
		for i := 0; i < count; i++ {
			start := i * 8
			bits := byteorder.ByteOrder.Uint64(data[start : start+8])
			result[i] = math.Float64frombits(bits)
		}
		return result, nil

	case "fp16", "datatypefp16":
		result := make([]float32, count)
		for i := 0; i < count; i++ {
			start := i * 2
			// Avoid slicing inside loop, pass single byte array by referencing data slice directly
			// If your Float16AsFP32 accepts []byte, just pass data[start:start+2]
			result[i] = byteorder.ByteOrder.Float16AsFP32(data[start : start+2])
		}
		return result, nil

	case "fp8e5m2", "datatypefp8e5m2":
		result := make([]float32, count)
		for i := 0; i < count; i++ {
			result[i] = byteorder.ByteOrder.Float8E5M2AsFP32(data[i : i+1])
		}
		return result, nil

	case "fp8e4m3", "datatypefp8e4m3":
		result := make([]float32, count)
		for i := 0; i < count; i++ {
			result[i] = byteorder.ByteOrder.Float8E4M3AsFP32(data[i : i+1])
		}
		return result, nil

	default:
		return nil, errors.New("unsupported input data type: " + dataType)
	}
}

func nativeSliceToBytes(value interface{}, outputType string) ([]byte, error) {
	var size int
	switch outputType {
	case "int8", "datatypeint8", "uint8", "datatypeuint8":
		if s, ok := value.([]int8); ok {
			size = len(s) * 1
		} else if s, ok := value.([]uint8); ok {
			return s, nil
		} else {
			return nil, errors.New("type assertion failed for " + outputType)
		}
	case "int16", "datatypeint16", "uint16", "datatypeuint16":
		if s, ok := value.([]int16); ok {
			size = len(s) * 2
		} else if s, ok := value.([]uint16); ok {
			size = len(s) * 2
		} else {
			return nil, errors.New("type assertion failed for " + outputType)
		}
	case "int32", "datatypeint32", "uint32", "datatypeuint32", "fp32", "datatypefp32":
		if s, ok := value.([]int32); ok {
			size = len(s) * 4
		} else if s, ok := value.([]uint32); ok {
			size = len(s) * 4
		} else if s, ok := value.([]float32); ok {
			size = len(s) * 4
		} else {
			return nil, errors.New("type assertion failed for" + outputType)
		}
	case "int64", "datatypeint64", "uint64", "datatypeuint64", "fp64", "datatypefp64":
		if s, ok := value.([]int64); ok {
			size = len(s) * 8
		} else if s, ok := value.([]uint64); ok {
			size = len(s) * 8
		} else if s, ok := value.([]float64); ok {
			size = len(s) * 8
		} else {
			return nil, errors.New("type assertion failed for" + outputType)
		}
	default:
		return nil, errors.New("unsupported output vector data type: " + outputType)
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(&value)), size), nil
}

func convertNativeSlice(inputSlice interface{}, outputType string) (interface{}, error) {
	switch v := inputSlice.(type) {
	case []int8:
		switch outputType {
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "int32", "datatypeint32":
			output := make([]int32, len(v))
			for i, val := range v {
				output[i] = int32(val)
			}
			return output, nil
		case "int64", "datatypeint64":
			output := make([]int64, len(v))
			for i, val := range v {
				output[i] = int64(val)
			}
			return output, nil
		case "uint8", "datatypeuint8":
			output := make([]uint8, len(v))
			for i, val := range v {
				output[i] = uint8(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion type not supported")
		}
	case []int16:
		switch outputType {
		case "int8", "datatypeint8":
			output := make([]int8, len(v))
			for i, val := range v {
				output[i] = int8(val)
			}
			return output, nil
		case "int32", "datatypeint32":
			output := make([]int32, len(v))
			for i, val := range v {
				output[i] = int32(val)
			}
			return output, nil
		case "int64", "datatypeint64":
			output := make([]int64, len(v))
			for i, val := range v {
				output[i] = int64(val)
			}
			return output, nil
		case "uint16", "datatypeuint16":
			output := make([]uint16, len(v))
			for i, val := range v {
				output[i] = uint16(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []int32:
		switch outputType {
		case "int8", "datatypeint8":
			output := make([]int8, len(v))
			for i, val := range v {
				output[i] = int8(val)
			}
			return output, nil
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "int64", "datatypeint64":
			output := make([]int64, len(v))
			for i, val := range v {
				output[i] = int64(val)
			}
			return output, nil
		case "uint32", "datatypeuint32":
			output := make([]uint32, len(v))
			for i, val := range v {
				output[i] = uint32(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []int64:
		switch outputType {
		case "int8", "datatypeint8":
			output := make([]int8, len(v))
			for i, val := range v {
				output[i] = int8(val)
			}
			return output, nil
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "int32", "datatypeint32":
			output := make([]int32, len(v))
			for i, val := range v {
				output[i] = int32(val)
			}
			return output, nil
		case "uint64", "datatypeuint64":
			output := make([]uint64, len(v))
			for i, val := range v {
				output[i] = uint64(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []uint8:
		switch outputType {
		case "int8", "datatypeint8":
			output := make([]int8, len(v))
			for i, val := range v {
				output[i] = int8(val)
			}
			return output, nil
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "uint16", "datatypeuint16":
			output := make([]uint16, len(v))
			for i, val := range v {
				output[i] = uint16(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []uint16:
		switch outputType {
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "uint8", "datatypeuint8":
			output := make([]uint8, len(v))
			for i, val := range v {
				output[i] = uint8(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []uint32:
		switch outputType {
		case "int32", "datatypeint32":
			output := make([]int32, len(v))
			for i, val := range v {
				output[i] = int32(val)
			}
			return output, nil
		case "uint16", "datatypeuint16":
			output := make([]uint16, len(v))
			for i, val := range v {
				output[i] = uint16(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []uint64:
		switch outputType {
		case "int64", "datatypeint64":
			output := make([]int64, len(v))
			for i, val := range v {
				output[i] = int64(val)
			}
			return output, nil
		case "uint32", "datatypeuint32":
			output := make([]uint32, len(v))
			for i, val := range v {
				output[i] = uint32(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []float32:
		switch outputType {
		case "int8", "datatypeint8":
			output := make([]int8, len(v))
			for i, val := range v {
				output[i] = int8(val)
			}
			return output, nil
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "int32", "datatypeint32":
			output := make([]int32, len(v))
			for i, val := range v {
				output[i] = int32(val)
			}
			return output, nil
		case "int64", "datatypeint64":
			output := make([]int64, len(v))
			for i, val := range v {
				output[i] = int64(val)
			}
			return output, nil
		case "uint8", "datatypeuint8":
			output := make([]uint8, len(v))
			for i, val := range v {
				output[i] = uint8(val)
			}
			return output, nil
		case "fp64", "datatypefp64":
			output := make([]float64, len(v))
			for i, val := range v {
				output[i] = float64(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	case []float64:
		switch outputType {
		case "int8", "datatypeint8":
			output := make([]int8, len(v))
			for i, val := range v {
				output[i] = int8(val)
			}
			return output, nil
		case "int16", "datatypeint16":
			output := make([]int16, len(v))
			for i, val := range v {
				output[i] = int16(val)
			}
			return output, nil
		case "int32", "datatypeint32":
			output := make([]int32, len(v))
			for i, val := range v {
				output[i] = int32(val)
			}
			return output, nil
		case "int64", "datatypeint64":
			output := make([]int64, len(v))
			for i, val := range v {
				output[i] = int64(val)
			}
			return output, nil
		case "uint8", "datatypeuint8":
			output := make([]uint8, len(v))
			for i, val := range v {
				output[i] = uint8(val)
			}
			return output, nil
		case "fp32", "datatypefp32":
			output := make([]float32, len(v))
			for i, val := range v {
				output[i] = float32(val)
			}
			return output, nil
		default:
			return nil, errors.New("conversion not supported")
		}
	default:
		return nil, errors.New("conversion not supported")
	}
}

func bytesToNativeSlice(data []byte, dataType string) (interface{}, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data slice")
	}
	dataLen := len(data)
	// Use pointer to first element of the slice's underlying array
	dataPtr := unsafe.Pointer(&data[0])

	switch dataType {
	case "int8", "datatypeint8":
		// int8 is 1 byte, so any length is valid
		return unsafe.Slice((*int8)(dataPtr), dataLen), nil
	case "int16", "datatypeint16":
		if dataLen%2 != 0 {
			return nil, errors.New("invalid size for int16 slice")
		}
		return unsafe.Slice((*int16)(dataPtr), dataLen/2), nil
	case "int32", "datatypeint32":
		if dataLen%4 != 0 {
			return nil, errors.New("invalid size for int32 slice")
		}
		return unsafe.Slice((*int32)(dataPtr), dataLen/4), nil
	case "int64", "datatypeint64":
		if dataLen%8 != 0 {
			return nil, errors.New("invalid size for int64 slice")
		}
		return unsafe.Slice((*int64)(dataPtr), dataLen/8), nil
	case "uint8", "datatypeuint8":
		return data, nil // A []byte is already a []uint8.
	case "uint16", "datatypeuint16":
		if dataLen%2 != 0 {
			return nil, errors.New("invalid size for uint16 slice")
		}
		return unsafe.Slice((*uint16)(dataPtr), dataLen/2), nil
	case "uint32", "datatypeuint32":
		if dataLen%4 != 0 {
			return nil, errors.New("invalid size for uint32 slice")
		}
		return unsafe.Slice((*uint32)(dataPtr), dataLen/4), nil
	case "uint64", "datatypeuint64":
		if dataLen%8 != 0 {
			return nil, errors.New("invalid size for uint64 slice")
		}
		return unsafe.Slice((*uint64)(dataPtr), dataLen/8), nil
	case "fp32", "datatypefp32":
		if dataLen%4 != 0 {
			return nil, errors.New("invalid size for fp32 slice")
		}
		return unsafe.Slice((*float32)(dataPtr), dataLen/4), nil
	case "fp64", "datatypefp64":
		if dataLen%8 != 0 {
			return nil, errors.New("invalid size for fp64 slice")
		}
		return unsafe.Slice((*float64)(dataPtr), dataLen/8), nil
	default:
		return nil, errors.New("unsupported input vector data type: " + dataType)
	}
}

// bytesToNativeType converts byte data to native Go type
func bytesToNativeType(data []byte, dataType string) (interface{}, error) {
	switch dataType {
	case "string", "datatypestring":
		// If the string represents a number, parse it
		str := string(data)
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f, nil
		}
		return str, nil
	case "bool", "datatypebool":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for bool")
		}
		return data[0] != 0, nil
	case "int8", "datatypeint8":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for int8")
		}
		return byteorder.ByteOrder.Int8(data), nil
	case "int16", "datatypeint16":
		if len(data) < 2 {
			return nil, errors.New("insufficient bytes for int16")
		}
		return byteorder.ByteOrder.Int16(data), nil
	case "int32", "datatypeint32":
		if len(data) < 4 {
			return nil, errors.New("insufficient bytes for int32")
		}
		return byteorder.ByteOrder.Int32(data), nil
	case "int64", "datatypeint64":
		if len(data) < 8 {
			return nil, errors.New("insufficient bytes for int64")
		}
		return byteorder.ByteOrder.Int64(data), nil
	case "uint8", "datatypeuint8":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for uint8")
		}
		return uint8(data[0]), nil
	case "uint16", "datatypeuint16":
		if len(data) < 2 {
			return nil, errors.New("insufficient bytes for uint16")
		}
		return byteorder.ByteOrder.Uint16(data), nil
	case "uint32", "datatypeuint32":
		if len(data) < 4 {
			return nil, errors.New("insufficient bytes for uint32")
		}
		return byteorder.ByteOrder.Uint32(data), nil
	case "uint64", "datatypeuint64":
		if len(data) < 8 {
			return nil, errors.New("insufficient bytes for uint64")
		}
		return byteorder.ByteOrder.Uint64(data), nil
	case "fp32", "datatypefp32":
		if len(data) < 4 {
			return nil, errors.New("insufficient bytes for fp32")
		}
		return byteorder.ByteOrder.Float32(data), nil
	case "fp64", "datatypefp64":
		if len(data) < 8 {
			return nil, errors.New("insufficient bytes for fp64")
		}
		return byteorder.ByteOrder.Float64(data), nil
	case "fp16", "datatypefp16":
		if len(data) < 2 {
			return nil, errors.New("insufficient bytes for fp16")
		}
		return byteorder.ByteOrder.Float16AsFP32(data), nil
	case "fp8e5m2", "datatypefp8e5m2":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for fp8e5m2")
		}
		return byteorder.ByteOrder.Float8E5M2AsFP32(data), nil
	case "fp8e4m3", "datatypefp8e4m3":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for fp8e4m3")
		}
		return byteorder.ByteOrder.Float8E4M3AsFP32(data), nil
	default:
		return nil, errors.New("unsupported input data type: " + dataType)
	}
}

// nativeTypeToBytes converts native Go type to byte data
func nativeTypeToBytes(value interface{}, outputType string) ([]byte, error) {
	switch outputType {
	case "bool", "datatypebool":
		var b bool
		switch v := value.(type) {
		case bool:
			b = v
		case int8:
			b = v != 0
		case int16:
			b = v != 0
		case int32:
			b = v != 0
		case int64:
			b = v != 0
		case uint8:
			b = v != 0
		case uint16:
			b = v != 0
		case uint32:
			b = v != 0
		case uint64:
			b = v != 0
		case float32:
			b = v != 0
		case float64:
			b = v != 0
		default:
			return nil, errors.New("cannot convert to bool from unknown type")
		}
		if b {
			return []byte{1}, nil
		}
		return []byte{0}, nil

	case "int8", "datatypeint8":
		var i8 int8
		switch v := value.(type) {
		case bool:
			if v {
				i8 = 1
			} else {
				i8 = 0
			}
		case int8:
			i8 = v
		case int16:
			i8 = int8(v)
		case int32:
			i8 = int8(v)
		case int64:
			i8 = int8(v)
		case uint8:
			i8 = int8(v)
		case uint16:
			i8 = int8(v)
		case uint32:
			i8 = int8(v)
		case uint64:
			i8 = int8(v)
		case float32:
			i8 = int8(v)
		case float64:
			i8 = int8(v)
		default:
			return nil, errors.New("cannot convert to int8 from unknown type")
		}
		buf := make([]byte, 1)
		byteorder.ByteOrder.PutInt8FromInt32(buf, int32(i8))
		return buf, nil

	case "int16", "datatypeint16":
		var i16 int16
		switch v := value.(type) {
		case bool:
			if v {
				i16 = 1
			} else {
				i16 = 0
			}
		case int8:
			i16 = int16(v)
		case int16:
			i16 = v
		case int32:
			i16 = int16(v)
		case int64:
			i16 = int16(v)
		case uint8:
			i16 = int16(v)
		case uint16:
			i16 = int16(v)
		case uint32:
			i16 = int16(v)
		case uint64:
			i16 = int16(v)
		case float32:
			i16 = int16(v)
		case float64:
			i16 = int16(v)
		default:
			return nil, errors.New("cannot convert to int16 from unknown type")
		}
		buf := make([]byte, 2)
		byteorder.ByteOrder.PutInt16FromInt32(buf, int32(i16))
		return buf, nil

	case "int32", "datatypeint32":
		var i32 int32
		switch v := value.(type) {
		case bool:
			if v {
				i32 = 1
			} else {
				i32 = 0
			}
		case int8:
			i32 = int32(v)
		case int16:
			i32 = int32(v)
		case int32:
			i32 = v
		case int64:
			i32 = int32(v)
		case uint8:
			i32 = int32(v)
		case uint16:
			i32 = int32(v)
		case uint32:
			i32 = int32(v)
		case uint64:
			i32 = int32(v)
		case float32:
			i32 = int32(v)
		case float64:
			i32 = int32(v)
		default:
			return nil, errors.New("cannot convert to int32 from unknown type")
		}
		buf := make([]byte, 4)
		byteorder.ByteOrder.PutInt32(buf, i32)
		return buf, nil

	case "int64", "datatypeint64":
		var i64 int64
		switch v := value.(type) {
		case bool:
			if v {
				i64 = 1
			} else {
				i64 = 0
			}
		case int8:
			i64 = int64(v)
		case int16:
			i64 = int64(v)
		case int32:
			i64 = int64(v)
		case int64:
			i64 = v
		case uint8:
			i64 = int64(v)
		case uint16:
			i64 = int64(v)
		case uint32:
			i64 = int64(v)
		case uint64:
			i64 = int64(v)
		case float32:
			i64 = int64(v)
		case float64:
			i64 = int64(v)
		default:
			return nil, errors.New("cannot convert to int64 from unknown type")
		}
		buf := make([]byte, 8)
		byteorder.ByteOrder.PutInt64(buf, i64)
		return buf, nil

	case "uint8", "datatypeuint8":
		var ui8 uint8
		switch v := value.(type) {
		case bool:
			if v {
				ui8 = 1
			} else {
				ui8 = 0
			}
		case int8:
			ui8 = uint8(v)
		case int16:
			ui8 = uint8(v)
		case int32:
			ui8 = uint8(v)
		case int64:
			ui8 = uint8(v)
		case uint8:
			ui8 = v
		case uint16:
			ui8 = uint8(v)
		case uint32:
			ui8 = uint8(v)
		case uint64:
			ui8 = uint8(v)
		case float32:
			ui8 = uint8(v)
		case float64:
			ui8 = uint8(v)
		default:
			return nil, errors.New("cannot convert to uint8 from unknown type")
		}
		return []byte{ui8}, nil

	case "uint16", "datatypeuint16":
		var ui16 uint16
		switch v := value.(type) {
		case bool:
			if v {
				ui16 = 1
			} else {
				ui16 = 0
			}
		case int8:
			ui16 = uint16(v)
		case int16:
			ui16 = uint16(v)
		case int32:
			ui16 = uint16(v)
		case int64:
			ui16 = uint16(v)
		case uint8:
			ui16 = uint16(v)
		case uint16:
			ui16 = v
		case uint32:
			ui16 = uint16(v)
		case uint64:
			ui16 = uint16(v)
		case float32:
			ui16 = uint16(v)
		case float64:
			ui16 = uint16(v)
		default:
			return nil, errors.New("cannot convert to uint16 from unknown type")
		}
		buf := make([]byte, 2)
		byteorder.ByteOrder.PutUint16(buf, ui16)
		return buf, nil

	case "uint32", "datatypeuint32":
		var ui32 uint32
		switch v := value.(type) {
		case bool:
			if v {
				ui32 = 1
			} else {
				ui32 = 0
			}
		case int8:
			ui32 = uint32(v)
		case int16:
			ui32 = uint32(v)
		case int32:
			ui32 = uint32(v)
		case int64:
			ui32 = uint32(v)
		case uint8:
			ui32 = uint32(v)
		case uint16:
			ui32 = uint32(v)
		case uint32:
			ui32 = v
		case uint64:
			ui32 = uint32(v)
		case float32:
			ui32 = uint32(v)
		case float64:
			ui32 = uint32(v)
		default:
			return nil, errors.New("cannot convert to uint32 from unknown type")
		}
		buf := make([]byte, 4)
		byteorder.ByteOrder.PutUint32(buf, ui32)
		return buf, nil

	case "uint64", "datatypeuint64":
		var ui64 uint64
		switch v := value.(type) {
		case bool:
			if v {
				ui64 = 1
			} else {
				ui64 = 0
			}
		case int8:
			ui64 = uint64(v)
		case int16:
			ui64 = uint64(v)
		case int32:
			ui64 = uint64(v)
		case int64:
			ui64 = uint64(v)
		case uint8:
			ui64 = uint64(v)
		case uint16:
			ui64 = uint64(v)
		case uint32:
			ui64 = uint64(v)
		case uint64:
			ui64 = v
		case float32:
			ui64 = uint64(v)
		case float64:
			ui64 = uint64(v)
		default:
			return nil, errors.New("cannot convert to uint64 from unknown type")
		}
		buf := make([]byte, 8)
		byteorder.ByteOrder.PutUint64(buf, ui64)
		return buf, nil

	case "fp32", "datatypefp32":
		var f32 float32
		switch v := value.(type) {
		case bool:
			if v {
				f32 = 1.0
			} else {
				f32 = 0.0
			}
		case int8:
			f32 = float32(v)
		case int16:
			f32 = float32(v)
		case int32:
			f32 = float32(v)
		case int64:
			f32 = float32(v)
		case uint8:
			f32 = float32(v)
		case uint16:
			f32 = float32(v)
		case uint32:
			f32 = float32(v)
		case uint64:
			f32 = float32(v)
		case float32:
			f32 = v
		case float64:
			f32 = float32(v)
		default:
			return nil, errors.New("cannot convert to fp32 from unknown type")
		}
		buf := make([]byte, 4)
		byteorder.ByteOrder.PutFloat32(buf, f32)
		return buf, nil

	case "fp64", "datatypefp64":
		var f64 float64
		switch v := value.(type) {
		case bool:
			if v {
				f64 = 1.0
			} else {
				f64 = 0.0
			}
		case int8:
			f64 = float64(v)
		case int16:
			f64 = float64(v)
		case int32:
			f64 = float64(v)
		case int64:
			f64 = float64(v)
		case uint8:
			f64 = float64(v)
		case uint16:
			f64 = float64(v)
		case uint32:
			f64 = float64(v)
		case uint64:
			f64 = float64(v)
		case float32:
			f64 = float64(v)
		case float64:
			f64 = v
		default:
			return nil, errors.New("cannot convert to fp64 from unknown type")
		}
		buf := make([]byte, 8)
		byteorder.ByteOrder.PutFloat64(buf, f64)
		return buf, nil

	case "fp16", "datatypefp16":
		var f32 float32
		switch v := value.(type) {
		case bool:
			if v {
				f32 = 1.0
			} else {
				f32 = 0.0
			}
		case int8:
			f32 = float32(v)
		case int16:
			f32 = float32(v)
		case int32:
			f32 = float32(v)
		case int64:
			f32 = float32(v)
		case uint8:
			f32 = float32(v)
		case uint16:
			f32 = float32(v)
		case uint32:
			f32 = float32(v)
		case uint64:
			f32 = float32(v)
		case float32:
			f32 = float32(v)
		case float64:
			f32 = float32(v)
		default:
			return nil, errors.New("cannot convert to fp16 from unknown type")
		}
		buf := FP32toFP16(f32)
		return buf, nil

	case "fp8e5m2", "datatypefp8e5m2":
		var f32 float32
		switch v := value.(type) {
		case bool:
			if v {
				f32 = 1.0
			} else {
				f32 = 0.0
			}
		case int8:
			f32 = float32(v)
		case int16:
			f32 = float32(v)
		case int32:
			f32 = float32(v)
		case int64:
			f32 = float32(v)
		case uint8:
			f32 = float32(v)
		case uint16:
			f32 = float32(v)
		case uint32:
			f32 = float32(v)
		case uint64:
			f32 = float32(v)
		case float32:
			f32 = v
		case float64:
			f32 = float32(v)
		default:
			return nil, errors.New("cannot convert to fp8e5m2 from unknown type")
		}
		buf := make([]byte, 1)
		byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf, f32)
		return buf, nil

	case "fp8e4m3", "datatypefp8e4m3":
		var f32 float32
		switch v := value.(type) {
		case bool:
			if v {
				f32 = 1.0
			} else {
				f32 = 0.0
			}
		case int8:
			f32 = float32(v)
		case int16:
			f32 = float32(v)
		case int32:
			f32 = float32(v)
		case int64:
			f32 = float32(v)
		case uint8:
			f32 = float32(v)
		case uint16:
			f32 = float32(v)
		case uint32:
			f32 = float32(v)
		case uint64:
			f32 = float32(v)
		case float32:
			f32 = v
		case float64:
			f32 = float32(v)
		default:
			return nil, errors.New("cannot convert to fp8e4m3 from unknown type")
		}
		buf := make([]byte, 1)
		byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf, f32)
		return buf, nil

	default:
		return nil, errors.New("unsupported output data type: " + outputType)
	}
}

func nativeListToBytes(values interface{}, outputType string) ([]byte, error) {
	var buf []byte

	switch outputType {
	case "bool", "datatypebool":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []int8:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []int16:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []int32:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []int64:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []uint8:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []uint16:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []uint32:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []uint64:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []float32:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0.0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []float64:
			buf = make([]byte, len(v))
			for i, elem := range v {
				if elem != 0.0 {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		default:
			return nil, errors.New("cannot convert to bool from unknown slice type")
		}
		return buf, nil

	case "int8", "datatypeint8":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				if elem {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []int8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []int16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []int32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []int64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []uint8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []uint16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []uint32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []uint64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []float32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		case []float64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = byte(elem)
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "int16", "datatypeint16":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				val := int16(0)
				if elem {
					val = 1
				}
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(val))
			}
		case []int8:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt16FromInt32(buf[i*2:i*2+2], int32(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "int32", "datatypeint32":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				val := int32(0)
				if elem {
					val = 1
				}
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], val)
			}
		case []int8:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt32(buf[i*4:i*4+4], int32(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "int64", "datatypeint64":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				val := int64(0)
				if elem {
					val = 1
				}
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], val)
			}
		case []int8:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutInt64(buf[i*8:i*8+8], int64(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "uint8", "datatypeuint8":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				if elem {
					buf[i] = 1
				} else {
					buf[i] = 0
				}
			}
		case []int8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []int16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []int32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []int64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []uint8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []uint16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []uint32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []uint64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []float32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		case []float64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				buf[i] = uint8(elem)
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "uint16", "datatypeuint16":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				val := uint16(0)
				if elem {
					val = 1
				}
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], val)
			}
		case []int8:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint16(buf[i*2:i*2+2], uint16(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "uint32", "datatypeuint32":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				val := uint32(0)
				if elem {
					val = 1
				}
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], val)
			}
		case []int8:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint32(buf[i*4:i*4+4], uint32(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "uint64", "datatypeuint64":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				val := uint64(0)
				if elem {
					val = 1
				}
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], val)
			}
		case []int8:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutUint64(buf[i*8:i*8+8], uint64(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "fp32", "datatypefp32":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				val := float32(0.0)
				if elem {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], val)
			}
		case []int8:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*4)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat32(buf[i*4:i*4+4], float32(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "fp64", "datatypefp64":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				val := float64(0.0)
				if elem {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], val)
			}
		case []int8:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*8)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat64(buf[i*8:i*8+8], float64(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "fp16", "datatypefp16":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				val := float32(0.0)
				if elem {
					val = 1.0
				}
				FP32toFP16Buffer(val, buf, i*2)
			}
		case []int8:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []int16:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []int32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []int64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []uint8:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []uint16:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []uint32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []uint64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []float32:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		case []float64:
			buf = make([]byte, len(v)*2)
			for i, elem := range v {
				FP32toFP16Buffer(float32(elem), buf, i*2)
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "fp8e5m2", "datatypefp8e5m2":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				val := float32(0.0)
				if elem {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], val)
			}
		case []int8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	case "fp8e4m3", "datatypefp8e4m3":
		switch v := values.(type) {
		case []bool:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				val := float32(0.0)
				if elem {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], val)
			}
		case []int8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []int16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []int32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []int64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint8:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint16:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []uint64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []float32:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		case []float64:
			buf = make([]byte, len(v)*1)
			for i, elem := range v {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(buf[i*1:i*1+1], float32(elem))
			}
		default:
			return nil, fmt.Errorf("cannot convert to %s from unknown slice type", outputType)
		}
		return buf, nil

	default:
		return nil, errors.New("unsupported output data type: " + outputType)
	}
}

func getElemSize(dataType string) (int, error) {
	switch strings.ToLower(dataType) {
	case "bool", "datatypebool", "int8", "datatypeint8", "uint8", "datatypeuint8",
		"fp8e5m2", "datatypefp8e5m2", "fp8e4m3", "datatypefp8e4m3":
		return 1, nil
	case "int16", "datatypeint16", "uint16", "datatypeuint16", "fp16", "datatypefp16":
		return 2, nil
	case "int32", "datatypeint32", "uint32", "datatypeuint32", "fp32", "datatypefp32":
		return 4, nil
	case "int64", "datatypeint64", "uint64", "datatypeuint64", "fp64", "datatypefp64":
		return 8, nil
	case "string", "datatypestring":
		return 0, errors.New("batch processing of strings not supported without length info")
	default:
		return 0, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

func ConvertVectorBytesToBytesOptimized(inputData []byte, inputType, outputType string) ([]byte, error) {
	if inputData == nil {
		return nil, errors.New("input data is nil")
	}

	inputType = strings.ToLower(inputType)
	outputType = strings.ToLower(outputType)

	if inputType == outputType {
		return inputData, nil
	}

	return directConvert(inputData, inputType, outputType)
}

func directConvert(inputData []byte, inputType, outputType string) ([]byte, error) {
	var inputElemSize int
	var err error

	if inputElemSize, err = getElemSize(inputType); err != nil {
		return nil, err
	}
	if len(inputData)%inputElemSize != 0 {
		return nil, fmt.Errorf("input data length (%d) not a multiple of input element size (%d)", len(inputData), inputElemSize)
	}

	count := len(inputData) / inputElemSize
	var outputData []byte

	switch inputType {
	case "bool", "datatypebool":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				if inputData[i] != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				if inputData[i] != 0 {
					outputData[i] = 1
				}
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				if inputData[i] != 0 {
					outputData[i] = 1
				}
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int16(0)
				if inputData[i] != 0 {
					val = 1
				}
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := uint16(0)
				if inputData[i] != 0 {
					val = 1
				}
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], val)
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int32(0)
				if inputData[i] != 0 {
					val = 1
				}
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := uint32(0)
				if inputData[i] != 0 {
					val = 1
				}
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], val)
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int64(0)
				if inputData[i] != 0 {
					val = 1
				}
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := uint64(0)
				if inputData[i] != 0 {
					val = 1
				}
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], val)
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := float32(0.0)
				if inputData[i] != 0 {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], val)
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := float64(0.0)
				if inputData[i] != 0 {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], val)
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := float32(0.0)
				if inputData[i] != 0 {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], val)
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := float32(0.0)
				if inputData[i] != 0 {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i*1:i*1+1], val)
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := float32(0.0)
				if inputData[i] != 0 {
					val = 1.0
				}
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i*1:i*1+1], val)
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "int8", "datatypeint8":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i, v := range inputData {
				if int8(v) != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i, v := range inputData {
				outputData[i] = v
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i, v := range inputData {
				outputData[i] = uint8(v)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i, v := range inputData {
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(int8(v)))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i, v := range inputData {
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(int8(v)))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i, v := range inputData {
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(int8(v)))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i, v := range inputData {
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(int8(v)))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i, v := range inputData {
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(int8(v)))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i, v := range inputData {
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(int8(v)))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(int8(v)))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(int8(v)))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(int8(v)))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(int8(v)))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(int8(v)))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "uint8", "datatypeuint8":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i, v := range inputData {
				if v != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i, v := range inputData {
				outputData[i] = v
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i, v := range inputData {
				outputData[i] = v
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i, v := range inputData {
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(v))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i, v := range inputData {
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(v))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i, v := range inputData {
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(v))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i, v := range inputData {
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(v))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i, v := range inputData {
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(v))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i, v := range inputData {
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(v))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(v))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(v))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(v))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(v))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i, v := range inputData {
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(v))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "int16", "datatypeint16":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				if val != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], val)
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(int16(val)))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int16(byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2]))
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "uint16", "datatypeuint16":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				if val != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(int16(val)))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], val)
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint16(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "int32", "datatypeint32":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				if val != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], val)
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], val)
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], val)
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int32(byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4]))
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "uint32", "datatypeuint32":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				if val != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], val)
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "int64", "datatypeint64":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				if val != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], val)
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := int64(byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8]))
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "uint64", "datatypeuint64":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				if val != 0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], val)
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "fp32", "datatypefp32":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				if val != 0.0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], bits)
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], val)
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], val)
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint32(inputData[i*4 : i*4+4])
				val := math.Float32frombits(bits)
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], val)
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "fp64", "datatypefp64":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				if val != 0.0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], float32(val))
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], bits)
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], float32(val))
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], float32(val))
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				bits := byteorder.ByteOrder.Uint64(inputData[i*8 : i*8+8])
				val := math.Float64frombits(bits)
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], float32(val))
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "fp16", "datatypefp16":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				if val != 0.0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], val)
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				copy(outputData[i*2:i*2+2], inputData[i*2:i*2+2])
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], val)
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float16AsFP32(inputData[i*2 : i*2+2])
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], val)
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "fp8e5m2", "datatypefp8e5m2":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				if val != 0.0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], val)
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], val)
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				outputData[i] = inputData[i]
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E5M2AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(outputData[i:i+1], val)
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	case "fp8e4m3", "datatypefp8e4m3":
		switch outputType {
		case "bool", "datatypebool":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				if val != 0.0 {
					outputData[i] = 1
				}
			}
		case "int8", "datatypeint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				outputData[i] = byte(val)
			}
		case "uint8", "datatypeuint8":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				outputData[i] = uint8(val)
			}
		case "int16", "datatypeint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutInt16FromInt32(outputData[i*2:i*2+2], int32(val))
			}
		case "uint16", "datatypeuint16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutUint16(outputData[i*2:i*2+2], uint16(val))
			}
		case "int32", "datatypeint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutInt32(outputData[i*4:i*4+4], int32(val))
			}
		case "uint32", "datatypeuint32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutUint32(outputData[i*4:i*4+4], uint32(val))
			}
		case "int64", "datatypeint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutInt64(outputData[i*8:i*8+8], int64(val))
			}
		case "uint64", "datatypeuint64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutUint64(outputData[i*8:i*8+8], uint64(val))
			}
		case "fp32", "datatypefp32":
			outputData = make([]byte, count*4)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat32(outputData[i*4:i*4+4], val)
			}
		case "fp64", "datatypefp64":
			outputData = make([]byte, count*8)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat64(outputData[i*8:i*8+8], float64(val))
			}
		case "fp16", "datatypefp16":
			outputData = make([]byte, count*2)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat16FromFP32(outputData[i*2:i*2+2], val)
			}
		case "fp8e5m2", "datatypefp8e5m2":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				val := byteorder.ByteOrder.Float8E4M3AsFP32(inputData[i : i+1])
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(outputData[i:i+1], val)
			}
		case "fp8e4m3", "datatypefp8e4m3":
			outputData = make([]byte, count)
			for i := 0; i < count; i++ {
				outputData[i] = inputData[i]
			}
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", inputType, outputType)
		}
	default:
		return nil, fmt.Errorf("unsupported input data type: %s", inputType)
	}

	return outputData, nil
}

func FP32toFP16(f32 float32) []byte {
	buf := make([]byte, 2)
	if f32 > 65000 {
		f32 = 65000
	}
	if f32 < -65000 {
		f32 = -65000
	}

	f16 := half.NewFloat16(f32)
	bits := uint16(f16)
	binary.LittleEndian.PutUint16(buf, bits)
	return buf
}

func FP32toFP16Buffer(f32 float32, buffer []byte, offset int) {
	f16 := FP32toFP16(f32)
	for _, byte := range f16 {
		buffer[offset] = byte
		offset++
	}
}
