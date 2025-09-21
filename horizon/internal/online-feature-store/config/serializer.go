package config

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
)

func Serialize(value string, dataType enums.DataType) ([]byte, error) {
	switch dataType {
	case "DataTypeInt8", "DataTypeInt16", "DataTypeInt32":
		return serializeInt32AndLess(value, dataType)
	case "DataTypeUint8", "DataTypeUint16", "DataTypeUint32":
		return serializeUint32AndLess(value, dataType)
	case "DataTypeInt64":
		return serializeInt64(value, dataType)
	case "DataTypeUint64":
		return serializeUint64(value, dataType)
	case "DataTypeFP8E5M2", "DataTypeFP8E4M3", "DataTypeFP16", "DataTypeFP32":
		return serializeFP32AndLess(value, dataType)
	case "DataTypeFP64":
		return serializeFP64(value, dataType)
	case "DataTypeBool":
		return serializeBool(value)
	case "DataTypeString":
		return serializeString(value, dataType)
	case "DataTypeInt8Vector", "DataTypeInt16Vector", "DataTypeInt32Vector":
		return serializeInt32VectorAndLess(value, dataType)
	case "DataTypeInt64Vector":
		return serializeInt64Vector(value, dataType)
	case "DataTypeUint8Vector", "DataTypeUint16Vector", "DataTypeUint32Vector":
		return serializeUint32VectorAndLess(value, dataType)
	case "DataTypeUint64Vector":
		return serializeUint64Vector(value, dataType)
	case "DataTypeFP8E5M2Vector", "DataTypeFP8E4M3Vector", "DataTypeFP16Vector", "DataTypeFP32Vector":
		return serializeFP32VectorAndLess(value, dataType)
	case "DataTypeFP64Vector":
		return serializeFP64Vector(value, dataType)
	case "DataTypeBoolVector":
		return serializeBoolVector(value)
	case "DataTypeStringVector":
		return serializeStringVector(value)
	default:
		return nil, fmt.Errorf("unknown data type: %v", dataType)
	}
}

func serializeFP32AndLess(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var result float32
	val, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %s to float32: %w", value, err)
	}
	result = float32(val)
	data := make([]byte, unitSize)
	idx := 0
	putFloat, _ := GetToByteFP32AndLess(dataType)
	putFloat(data[idx:idx+unitSize], result)
	return data, nil
}

func serializeInt32AndLess(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var result int32
	val, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %v to int8: %w", val, err)
	}
	result = int32(val)
	idx := 0
	data := make([]byte, unitSize)
	putInt, _ := GetToByteInt32AndLess(dataType)
	putInt(data[idx:idx+unitSize], result)
	return data, nil
}

func serializeUint32AndLess(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var result uint32
	val, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %s to int8: %w", value, err)
	}
	result = uint32(val)
	idx := 0
	data := make([]byte, unitSize)
	putInt, _ := GetToByteUint32AndLess(dataType)
	putInt(data[idx:idx+unitSize], result)
	return data, nil
}

func serializeInt64(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %s to int8: %w", value, err)
	}
	idx := 0
	data := make([]byte, unitSize)
	ByteOrder.PutInt64(data[idx:idx+unitSize], result)
	return data, nil
}

func serializeUint64(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %s to int8: %w", value, err)
	}
	idx := 0
	data := make([]byte, unitSize)
	ByteOrder.PutUint64(data[idx:idx+unitSize], result)
	return data, nil
}

func serializeFP64(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %s to float64: %w", value, err)
	}
	data := make([]byte, unitSize)
	idx := 0
	ByteOrder.PutFloat64(data[idx:idx+unitSize], result)
	return data, nil
}

func serializeString(value string, dataType enums.DataType) ([]byte, error) {
	byteArray := []byte(value)
	return byteArray, nil
}

func serializeInt32VectorAndLess(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var flattened []int32
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, row := range splitRows {
		row = strings.TrimSpace(row) // Trim whitespace
		val, err := strconv.ParseUint(row, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s to uint32: %w", row, err)
		}
		flattened = append(flattened, int32(val))
	}

	data := make([]byte, len(flattened)*unitSize)
	idx := 0
	putUint, _ := GetToByteInt32AndLess(dataType) // Assuming this utility is defined
	for _, val := range flattened {
		putUint(data[idx:idx+unitSize], val)
		idx += unitSize
	}
	return data, nil
}

func serializeInt64Vector(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var flattened []int64
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		val, err := strconv.ParseInt(vv, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s to int64: %w", vv, err)
		}
		flattened = append(flattened, val)
	}
	data := make([]byte, len(flattened)*unitSize)
	idx := 0
	for _, val := range flattened {
		binary.LittleEndian.PutUint64(data[idx:idx+unitSize], uint64(val))
		idx += unitSize
	}
	return data, nil
}

func serializeUint32VectorAndLess(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var flattened []uint32
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		val, err := strconv.ParseUint(vv, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s to uint32: %w", vv, err)
		}
		flattened = append(flattened, uint32(val))
	}
	data := make([]byte, len(flattened)*unitSize)
	idx := 0
	putUint, _ := GetToByteUint32AndLess(dataType) // Assuming this utility is defined
	for _, val := range flattened {
		putUint(data[idx:idx+unitSize], val)
		idx += unitSize
	}
	return data, nil
}

func serializeUint64Vector(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var flattened []uint64
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		val, err := strconv.ParseUint(vv, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s to uint64: %w", vv, err)
		}
		flattened = append(flattened, val)
	}
	data := make([]byte, len(flattened)*unitSize)
	idx := 0
	for _, val := range flattened {
		binary.LittleEndian.PutUint64(data[idx:idx+unitSize], val)
		idx += unitSize
	}
	return data, nil
}

func serializeFP32VectorAndLess(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var flattened []float32
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		val, err := strconv.ParseFloat(vv, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value %s to float32: %w", vv, err)
		}
		flattened = append(flattened, float32(val))
	}
	data := make([]byte, len(flattened)*unitSize)
	idx := 0
	putFloat, _ := GetToByteFP32AndLess(dataType)
	for _, v := range flattened {
		putFloat(data[idx:idx+unitSize], v)
		idx += unitSize
	}
	return data, nil
}

func serializeFP64Vector(value string, dataType enums.DataType) ([]byte, error) {
	unitSize := dataType.Size()
	var flattened []float64
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		val, err := strconv.ParseFloat(vv, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value %s to float64: %w", vv, err)
		}
		flattened = append(flattened, val)
	}
	data := make([]byte, len(flattened)*unitSize)
	idx := 0
	for _, v := range flattened {
		ByteOrder.PutFloat64(data[idx:idx+unitSize], v)
		idx += unitSize
	}
	return data, nil
}

func serializeBool(value string) ([]byte, error) {
	data := make([]byte, 1) // 1 byte for every 8 bools
	idx, shift := 0, 7
	val, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bool value %s: %w", value, err)
	}
	if val {
		data[idx] |= 1 << shift
	}
	shift--
	if shift < 0 {
		idx++
	}
	return data, nil
}

func serializeBoolVector(value string) ([]byte, error) {
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")

	// Calculate total bits needed
	totalBits := len(splitRows)
	data := make([]byte, (totalBits+7)/8) // Allocate bytes for bits
	idx := 0
	shift := 7

	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		val, err := strconv.ParseBool(vv)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s to bool: %w", vv, err)
		}
		if val {
			data[idx] |= 1 << shift
		}
		shift--
		if shift < 0 {
			shift = 7
			idx++
		}
	}
	return data, nil
}

func serializeStringVector(value string) ([]byte, error) {
	data := make([]byte, binary.MaxVarintLen64)
	// Strip brackets and split by comma
	cleanValue := strings.Trim(value, "[]")
	splitRows := strings.Split(cleanValue, ",")
	for _, vv := range splitRows {
		vv = strings.TrimSpace(vv) // Trim whitespace
		length := uint64(len(vv))
		lenBytes := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBytes, length)
		data = append(data, lenBytes[:n]...)
		data = append(data, vv...)
	}
	return data, nil
}
