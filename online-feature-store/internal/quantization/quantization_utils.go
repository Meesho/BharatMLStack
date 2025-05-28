package quantization

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

// Byte sizes for floating-point formats
const (
	FP32NumBytes = 4
	FP64NumBytes = 8
	FP16NumBytes = 2
	FP8NumBytes  = 1
)

// IsQuantizationCompatible checks if the source data type can be quantized to the target type
func IsQuantizationCompatible(sourceType, targetType types.DataType) bool {
	// String and Bool types cannot be quantized
	if sourceType == types.DataTypeString || sourceType == types.DataTypeStringVector ||
		sourceType == types.DataTypeBool || sourceType == types.DataTypeBoolVector {
		return false
	}

	// Cannot convert between scalar and vector types
	if sourceType.IsVector() != targetType.IsVector() {
		return false
	}

	// Only floating point types can be quantized
	validSourceTypes := map[types.DataType]bool{
		types.DataTypeFP64:          true,
		types.DataTypeFP32:          true,
		types.DataTypeFP16:          true,
		types.DataTypeFP8E4M3:       true,
		types.DataTypeFP8E5M2:       true,
		types.DataTypeFP64Vector:    true,
		types.DataTypeFP32Vector:    true,
		types.DataTypeFP16Vector:    true,
		types.DataTypeFP8E4M3Vector: true,
		types.DataTypeFP8E5M2Vector: true,
	}

	validTargetTypes := map[types.DataType]bool{
		types.DataTypeFP32:          true,
		types.DataTypeFP16:          true,
		types.DataTypeFP8E4M3:       true,
		types.DataTypeFP8E5M2:       true,
		types.DataTypeFP32Vector:    true,
		types.DataTypeFP16Vector:    true,
		types.DataTypeFP8E4M3Vector: true,
		types.DataTypeFP8E5M2Vector: true,
	}

	if !validSourceTypes[sourceType] || !validTargetTypes[targetType] {
		return false
	}

	// Check for forward/backward compatibility
	// We only allow quantization to lower precision
	sourceRank := getDataTypePrecisionRank(sourceType)
	targetRank := getDataTypePrecisionRank(targetType)

	return sourceRank > targetRank
}

// getDataTypePrecisionRank returns a rank for data types based on their precision
// Higher rank means higher precision
func getDataTypePrecisionRank(dataType types.DataType) int {
	baseType := dataType
	if dataType.IsVector() {
		switch dataType {
		case types.DataTypeFP64Vector:
			baseType = types.DataTypeFP64
		case types.DataTypeFP32Vector:
			baseType = types.DataTypeFP32
		case types.DataTypeFP16Vector:
			baseType = types.DataTypeFP16
		case types.DataTypeFP8E4M3Vector:
			baseType = types.DataTypeFP8E4M3
		case types.DataTypeFP8E5M2Vector:
			baseType = types.DataTypeFP8E5M2
		}
	}

	switch baseType {
	case types.DataTypeFP64:
		return 4
	case types.DataTypeFP32:
		return 3
	case types.DataTypeFP16:
		return 2
	case types.DataTypeFP8E4M3:
		return 1
	case types.DataTypeFP8E5M2:
		return 1
	default:
		return 0
	}
}

// Generic function to quantize a vector of one type to another
func quantizeVector(
	bytes []byte,
	sourceNumBytes int,
	targetNumBytes int,
	converter func([]byte) []byte,
) []byte {
	numValues := len(bytes) / sourceNumBytes
	result := make([]byte, numValues*targetNumBytes)
	for i := 0; i < numValues; i++ {
		start := i * sourceNumBytes
		end := start + sourceNumBytes

		quantized := converter(bytes[start:end])
		copy(result[i*targetNumBytes:(i+1)*targetNumBytes], quantized)
	}
	return result
}

// quantization functions for single values
func quantizeBytesFromFP64ToFP32(bytes []byte) []byte {
	valFP32 := float32(system.ByteOrder.Float64(bytes))
	result := make([]byte, FP32NumBytes)
	system.ByteOrder.PutFloat32(result, valFP32)
	return result
}

func quantizeBytesFromFP64ToFP16(bytes []byte) []byte {
	valFP32 := float32(system.ByteOrder.Float64(bytes))
	result := make([]byte, FP16NumBytes)
	system.ByteOrder.PutFloat16FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP32ToFP16(bytes []byte) []byte {
	valFP32 := system.ByteOrder.Float32(bytes)
	result := make([]byte, FP16NumBytes)
	system.ByteOrder.PutFloat16FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP64ToFP8E5M2(bytes []byte) []byte {
	valFP32 := float32(system.ByteOrder.Float64(bytes))
	result := make([]byte, FP8NumBytes)
	system.ByteOrder.PutFloat8E5M2FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP32ToFP8E5M2(bytes []byte) []byte {
	valFP32 := system.ByteOrder.Float32(bytes)
	result := make([]byte, FP8NumBytes)
	system.ByteOrder.PutFloat8E5M2FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP16ToFP8E5M2(bytes []byte) []byte {
	valFP32 := system.ByteOrder.Float16AsFP32(bytes)
	result := make([]byte, FP8NumBytes)
	system.ByteOrder.PutFloat8E5M2FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP64ToFP8E4M3(bytes []byte) []byte {
	valFP32 := float32(system.ByteOrder.Float64(bytes))
	result := make([]byte, FP8NumBytes)
	system.ByteOrder.PutFloat8E4M3FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP32ToFP8E4M3(bytes []byte) []byte {
	valFP32 := system.ByteOrder.Float32(bytes)
	result := make([]byte, FP8NumBytes)
	system.ByteOrder.PutFloat8E4M3FromFP32(result, valFP32)
	return result
}

func quantizeBytesFromFP16ToFP8E4M3(bytes []byte) []byte {
	valFP32 := system.ByteOrder.Float16AsFP32(bytes)
	result := make([]byte, FP8NumBytes)
	system.ByteOrder.PutFloat8E4M3FromFP32(result, valFP32)
	return result
}

// Vector quantization wrappers
func quantizeBytesFromFP64VectorToFP32Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP64NumBytes, FP32NumBytes, quantizeBytesFromFP64ToFP32)
}

func quantizeBytesFromFP64VectorToFP16Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP64NumBytes, FP16NumBytes, quantizeBytesFromFP64ToFP16)
}

func quantizeBytesFromFP32VectorToFP16Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP32NumBytes, FP16NumBytes, quantizeBytesFromFP32ToFP16)
}

func quantizeBytesFromFP32VectorToFP8E5M2Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP32NumBytes, FP8NumBytes, quantizeBytesFromFP32ToFP8E5M2)
}

func quantizeBytesFromFP64VectorToFP8E5M2Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP64NumBytes, FP8NumBytes, quantizeBytesFromFP64ToFP8E5M2)
}

func quantizeBytesFromFP16VectorToFP8E5M2Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP16NumBytes, FP8NumBytes, quantizeBytesFromFP16ToFP8E5M2)
}

func quantizeBytesFromFP32VectorToFP8E4M3Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP32NumBytes, FP8NumBytes, quantizeBytesFromFP32ToFP8E4M3)
}

func quantizeBytesFromFP64VectorToFP8E4M3Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP64NumBytes, FP8NumBytes, quantizeBytesFromFP64ToFP8E4M3)
}

func quantizeBytesFromFP16VectorToFP8E4M3Vector(bytes []byte) []byte {
	return quantizeVector(bytes, FP16NumBytes, FP8NumBytes, quantizeBytesFromFP16ToFP8E4M3)
}

// GetQuantizationFunction Function to get a quantization function based on data types
func GetQuantizationFunction(sourceDataType types.DataType, requiredDataType types.DataType) (func([]byte) []byte, error) {
	switch {

	// single value quantizations
	case sourceDataType == types.DataTypeFP64 && requiredDataType == types.DataTypeFP32:
		return quantizeBytesFromFP64ToFP32, nil
	case sourceDataType == types.DataTypeFP64 && requiredDataType == types.DataTypeFP16:
		return quantizeBytesFromFP64ToFP16, nil
	case sourceDataType == types.DataTypeFP32 && requiredDataType == types.DataTypeFP16:
		return quantizeBytesFromFP32ToFP16, nil
	case sourceDataType == types.DataTypeFP64 && requiredDataType == types.DataTypeFP8E4M3:
		return quantizeBytesFromFP64ToFP8E4M3, nil
	case sourceDataType == types.DataTypeFP32 && requiredDataType == types.DataTypeFP8E4M3:
		return quantizeBytesFromFP32ToFP8E4M3, nil
	case sourceDataType == types.DataTypeFP16 && requiredDataType == types.DataTypeFP8E4M3:
		return quantizeBytesFromFP16ToFP8E4M3, nil
	case sourceDataType == types.DataTypeFP32 && requiredDataType == types.DataTypeFP8E5M2:
		return quantizeBytesFromFP32ToFP8E5M2, nil
	case sourceDataType == types.DataTypeFP16 && requiredDataType == types.DataTypeFP8E5M2:
		return quantizeBytesFromFP16ToFP8E5M2, nil
	case sourceDataType == types.DataTypeFP64 && requiredDataType == types.DataTypeFP8E5M2:
		return quantizeBytesFromFP64ToFP8E5M2, nil

	// vector quantizations
	case sourceDataType == types.DataTypeFP64Vector && requiredDataType == types.DataTypeFP32Vector:
		return quantizeBytesFromFP64VectorToFP32Vector, nil
	case sourceDataType == types.DataTypeFP64Vector && requiredDataType == types.DataTypeFP16Vector:
		return quantizeBytesFromFP64VectorToFP16Vector, nil
	case sourceDataType == types.DataTypeFP32Vector && requiredDataType == types.DataTypeFP16Vector:
		return quantizeBytesFromFP32VectorToFP16Vector, nil
	case sourceDataType == types.DataTypeFP32Vector && requiredDataType == types.DataTypeFP8E4M3Vector:
		return quantizeBytesFromFP32VectorToFP8E4M3Vector, nil
	case sourceDataType == types.DataTypeFP64Vector && requiredDataType == types.DataTypeFP8E4M3Vector:
		return quantizeBytesFromFP64VectorToFP8E4M3Vector, nil
	case sourceDataType == types.DataTypeFP16Vector && requiredDataType == types.DataTypeFP8E4M3Vector:
		return quantizeBytesFromFP16VectorToFP8E4M3Vector, nil
	case sourceDataType == types.DataTypeFP32Vector && requiredDataType == types.DataTypeFP8E5M2Vector:
		return quantizeBytesFromFP32VectorToFP8E5M2Vector, nil
	case sourceDataType == types.DataTypeFP64Vector && requiredDataType == types.DataTypeFP8E5M2Vector:
		return quantizeBytesFromFP64VectorToFP8E5M2Vector, nil
	case sourceDataType == types.DataTypeFP16Vector && requiredDataType == types.DataTypeFP8E5M2Vector:
		return quantizeBytesFromFP16VectorToFP8E5M2Vector, nil
	default:
		return nil, fmt.Errorf("unsupported quantization from %s to %s", sourceDataType, requiredDataType)
	}
}
