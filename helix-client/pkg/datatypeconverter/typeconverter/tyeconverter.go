package typeconverter

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/datatypeconverter/byteorder"
	"github.com/Meesho/BharatMLStack/helix-client/pkg/datatypeconverter/types"
)

// BytesToString converts byte data to string based on the specified data type
func BytesToString(data []byte, dataType string) (string, error) {
	if data == nil {
		return "", errors.New("data is nil")
	}

	switch dataType {
	// Handle scalar types
	case "String", "DataTypeString", "BYTES":
		return string(data), nil
	case "Bool", "DataTypeBool":
		return boolBytesToString(data)
	case "Int8", "DataTypeInt8":
		return int8BytesToString(data)
	case "Int16", "DataTypeInt16":
		return int16BytesToString(data)
	case "Int32", "DataTypeInt32":
		return int32BytesToString(data)
	case "Int64", "DataTypeInt64":
		return int64BytesToString(data)
	case "Uint8", "DataTypeUint8":
		return uint8BytesToString(data)
	case "Uint16", "DataTypeUint16":
		return uint16BytesToString(data)
	case "Uint32", "DataTypeUint32":
		return uint32BytesToString(data)
	case "Uint64", "DataTypeUint64":
		return uint64BytesToString(data)
	case "FP32", "DataTypeFP32":
		return float32BytesToString(data)
	case "FP64", "DataTypeFP64":
		return float64BytesToString(data)
	case "FP16", "DataTypeFP16":
		return float16BytesToString(data)
	case "FP8E5M2", "DataTypeFP8E5M2":
		return float8E5M2BytesToString(data)
	case "FP8E4M3", "DataTypeFP8E4M3":
		return float8E4M3BytesToString(data)

	// Handle vector types
	case "BoolVector", "DataTypeBoolVector":
		return boolVectorBytesToString(data)
	case "Int8Vector", "DataTypeInt8Vector":
		return int8VectorBytesToString(data)
	case "Int16Vector", "DataTypeInt16Vector":
		return int16VectorBytesToString(data)
	case "Int32Vector", "DataTypeInt32Vector":
		return int32VectorBytesToString(data)
	case "Int64Vector", "DataTypeInt64Vector":
		return int64VectorBytesToString(data)
	case "Uint8Vector", "DataTypeUint8Vector":
		return uint8VectorBytesToString(data)
	case "Uint16Vector", "DataTypeUint16Vector":
		return uint16VectorBytesToString(data)
	case "Uint32Vector", "DataTypeUint32Vector":
		return uint32VectorBytesToString(data)
	case "Uint64Vector", "DataTypeUint64Vector":
		return uint64VectorBytesToString(data)
	case "FP32Vector", "DataTypeFP32Vector":
		return float32VectorBytesToString(data)
	case "FP64Vector", "DataTypeFP64Vector":
		return float64VectorBytesToString(data)
	case "FP16Vector", "DataTypeFP16Vector":
		return float16VectorBytesToString(data)
	case "FP8E5M2Vector", "DataTypeFP8E5M2Vector":
		return float8E5M2VectorBytesToString(data)
	case "FP8E4M3Vector", "DataTypeFP8E4M3Vector":
		return float8E4M3VectorBytesToString(data)
	case "StringVector", "DataTypeStringVector":
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
	return strconv.FormatInt(int64(byteorder.ByteOrder.Int8(data)), 10), nil
}

func int16BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt16.Size() {
		return "", errors.New("invalid byte length for int16")
	}
	return strconv.FormatInt(int64(byteorder.ByteOrder.Int16(data)), 10), nil
}

func int32BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt32.Size() {
		return "", errors.New("invalid byte length for int32")
	}
	return strconv.FormatInt(int64(byteorder.ByteOrder.Int32(data)), 10), nil
}

func int64BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeInt64.Size() {
		return "", errors.New("invalid byte length for int64")
	}
	return strconv.FormatInt(byteorder.ByteOrder.Int64(data), 10), nil
}

func uint8BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint8.Size() {
		return "", errors.New("invalid byte length for uint8")
	}
	return strconv.FormatUint(uint64(data[0]), 10), nil
}

func uint16BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint16.Size() {
		return "", errors.New("invalid byte length for uint16")
	}
	return strconv.FormatUint(uint64(byteorder.ByteOrder.Uint16(data)), 10), nil
}

func uint32BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint32.Size() {
		return "", errors.New("invalid byte length for uint32")
	}
	return strconv.FormatUint(uint64(byteorder.ByteOrder.Uint32(data)), 10), nil
}

func uint64BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeUint64.Size() {
		return "", errors.New("invalid byte length for uint64")
	}
	return strconv.FormatUint(byteorder.ByteOrder.Uint64(data), 10), nil
}

func float32BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP32.Size() {
		return "", errors.New("invalid byte length for float32")
	}
	return strconv.FormatFloat(float64(byteorder.ByteOrder.Float32(data)), 'f', -1, 32), nil
}

func float64BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP64.Size() {
		return "", errors.New("invalid byte length for float64")
	}
	return strconv.FormatFloat(byteorder.ByteOrder.Float64(data), 'f', -1, 64), nil
}

func float16BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP16.Size() {
		return "", errors.New("invalid byte length for float16")
	}
	fp32 := byteorder.ByteOrder.Float16AsFP32(data)
	return strconv.FormatFloat(float64(fp32), 'f', -1, 32), nil
}

func float8E5M2BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP8E5M2.Size() {
		return "", errors.New("invalid byte length for float8 E5M2")
	}
	fp32 := byteorder.ByteOrder.Float8E5M2AsFP32(data)
	return strconv.FormatFloat(float64(fp32), 'f', -1, 32), nil
}

func float8E4M3BytesToString(data []byte) (string, error) {
	if len(data) != types.DataTypeFP8E4M3.Size() {
		return "", errors.New("invalid byte length for float8 E4M3")
	}
	fp32 := byteorder.ByteOrder.Float8E4M3AsFP32(data)
	return strconv.FormatFloat(float64(fp32), 'f', -1, 32), nil
}

// Vector converters - convert byte data to comma-separated string representation
func vectorBytesToString[T any](data []byte, unitSize int, convertFunc func([]byte) T) (string, error) {
	if len(data)%unitSize != 0 {
		return "", errors.New("invalid byte length for vector data")
	}

	var values []string
	for i := 0; i < len(data); i += unitSize {
		val := convertFunc(data[i : i+unitSize])
		values = append(values, formatValue(val))
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

	switch dataType {
	// Scalar types
	case "String", "DataTypeString", "Bytes":
		return []byte(value), nil
	case "Bool", "DataTypeBool":
		return stringToBoolBytes(value)
	case "Int8", "DataTypeInt8":
		return stringToInt8Bytes(value)
	case "Int16", "DataTypeInt16":
		return stringToInt16Bytes(value)
	case "Int32", "DataTypeInt32":
		return stringToInt32Bytes(value)
	case "Int64", "DataTypeInt64":
		return stringToInt64Bytes(value)
	case "Uint8", "DataTypeUint8":
		return stringToUint8Bytes(value)
	case "Uint16", "DataTypeUint16":
		return stringToUint16Bytes(value)
	case "Uint32", "DataTypeUint32":
		return stringToUint32Bytes(value)
	case "Uint64", "DataTypeUint64":
		return stringToUint64Bytes(value)
	case "FP32", "DataTypeFP32":
		return stringToFloat32Bytes(value)
	case "FP64", "DataTypeFP64":
		return stringToFloat64Bytes(value)
	case "FP16", "DataTypeFP16":
		return stringToFloat16Bytes(value)
	case "FP8E5M2", "DataTypeFP8E5M2":
		return stringToFloat8E5M2Bytes(value)
	case "FP8E4M3", "DataTypeFP8E4M3":
		return stringToFloat8E4M3Bytes(value)
	// Vector types
	case "BoolVector", "DataTypeBoolVector":
		return stringToBoolVectorBytes(value)
	case "Int8Vector", "DataTypeInt8Vector":
		return stringToInt8VectorBytes(value)
	case "Int16Vector", "DataTypeInt16Vector":
		return stringToInt16VectorBytes(value)
	case "Int32Vector", "DataTypeInt32Vector":
		return stringToInt32VectorBytes(value)
	case "Int64Vector", "DataTypeInt64Vector":
		return stringToInt64VectorBytes(value)
	case "Uint8Vector", "DataTypeUint8Vector":
		return stringToUint8VectorBytes(value)
	case "Uint16Vector", "DataTypeUint16Vector":
		return stringToUint16VectorBytes(value)
	case "Uint32Vector", "DataTypeUint32Vector":
		return stringToUint32VectorBytes(value)
	case "Uint64Vector", "DataTypeUint64Vector":
		return stringToUint64VectorBytes(value)
	case "FP32Vector", "DataTypeFP32Vector":
		return stringToFloat32VectorBytes(value)
	case "FP64Vector", "DataTypeFP64Vector":
		return stringToFloat64VectorBytes(value)
	case "FP16Vector", "DataTypeFP16Vector":
		return stringToFloat16VectorBytes(value)
	case "FP8E5M2Vector", "DataTypeFP8E5M2Vector":
		return stringToFloat8E5M2VectorBytes(value)
	case "FP8E4M3Vector", "DataTypeFP8E4M3Vector":
		return stringToFloat8E4M3VectorBytes(value)
	case "StringVector", "DataTypeStringVector":
		return stringToStringVectorBytes(value)
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

// bytesToNativeType converts byte data to native Go type
func bytesToNativeType(data []byte, dataType string) (interface{}, error) {
	switch dataType {
	case "String", "DataTypeString":
		// If the string represents a number, parse it
		str := string(data)
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f, nil
		}
		return str, nil
	case "Bool", "DataTypeBool":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for bool")
		}
		return data[0] != 0, nil
	case "Int8", "DataTypeInt8":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for int8")
		}
		return byteorder.ByteOrder.Int8(data), nil
	case "Int16", "DataTypeInt16":
		if len(data) < 2 {
			return nil, errors.New("insufficient bytes for int16")
		}
		return byteorder.ByteOrder.Int16(data), nil
	case "Int32", "DataTypeInt32":
		if len(data) < 4 {
			return nil, errors.New("insufficient bytes for int32")
		}
		return byteorder.ByteOrder.Int32(data), nil
	case "Int64", "DataTypeInt64":
		if len(data) < 8 {
			return nil, errors.New("insufficient bytes for int64")
		}
		return byteorder.ByteOrder.Int64(data), nil
	case "Uint8", "DataTypeUint8":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for uint8")
		}
		return uint8(data[0]), nil
	case "Uint16", "DataTypeUint16":
		if len(data) < 2 {
			return nil, errors.New("insufficient bytes for uint16")
		}
		return byteorder.ByteOrder.Uint16(data), nil
	case "Uint32", "DataTypeUint32":
		if len(data) < 4 {
			return nil, errors.New("insufficient bytes for uint32")
		}
		return byteorder.ByteOrder.Uint32(data), nil
	case "Uint64", "DataTypeUint64":
		if len(data) < 8 {
			return nil, errors.New("insufficient bytes for uint64")
		}
		return byteorder.ByteOrder.Uint64(data), nil
	case "FP32", "DataTypeFP32":
		if len(data) < 4 {
			return nil, errors.New("insufficient bytes for fp32")
		}
		return byteorder.ByteOrder.Float32(data), nil
	case "FP64", "DataTypeFP64":
		if len(data) < 8 {
			return nil, errors.New("insufficient bytes for fp64")
		}
		return byteorder.ByteOrder.Float64(data), nil
	case "FP16", "DataTypeFP16":
		if len(data) < 2 {
			return nil, errors.New("insufficient bytes for fp16")
		}
		return byteorder.ByteOrder.Float16AsFP32(data), nil
	case "FP8E5M2", "DataTypeFP8E5M2":
		if len(data) < 1 {
			return nil, errors.New("insufficient bytes for fp8e5m2")
		}
		return byteorder.ByteOrder.Float8E5M2AsFP32(data), nil
	case "FP8E4M3", "DataTypeFP8E4M3":
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
	case "Bool", "DataTypeBool":
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

	case "Int8", "DataTypeInt8":
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

	case "Int16", "DataTypeInt16":
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

	case "Int32", "DataTypeInt32":
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

	case "Int64", "DataTypeInt64":
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

	case "Uint8", "DataTypeUint8":
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

	case "Uint16", "DataTypeUint16":
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

	case "Uint32", "DataTypeUint32":
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

	case "Uint64", "DataTypeUint64":
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

	case "FP32", "DataTypeFP32":
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

	case "FP64", "DataTypeFP64":
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

	case "FP16", "DataTypeFP16":
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
			return nil, errors.New("cannot convert to fp16 from unknown type")
		}
		buf := make([]byte, 2)
		byteorder.ByteOrder.PutFloat16FromFP32(buf, f32)
		return buf, nil

	case "FP8E5M2", "DataTypeFP8E5M2":
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

	case "FP8E4M3", "DataTypeFP8E4M3":
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
