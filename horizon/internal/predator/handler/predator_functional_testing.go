package handler

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/serializer"
	"github.com/rs/zerolog/log"
)

// flattenInputTo3DByteSlice converts input data to 3D byte slice format [batch][feature][bytes]
// This matches the working adapter's data structure expectations
func (p *Predator) flattenInputTo3DByteSlice(data any, dataType string) ([][][]byte, error) {
	switch v := data.(type) {
	case [][]float32:
		batchSize := len(v)
		if batchSize == 0 {
			return [][][]byte{}, nil
		}
		featureCount := len(v[0])

		result := make([][][]byte, batchSize)
		for batchIdx := 0; batchIdx < batchSize; batchIdx++ {
			result[batchIdx] = make([][]byte, featureCount)
			for featureIdx := 0; featureIdx < featureCount; featureIdx++ {
				val := v[batchIdx][featureIdx]
				switch dataType {
				case "FP16":
					fp16Bytes, err := serializer.Float32ToFloat16Bytes(val)
					if err != nil {
						return nil, err
					}
					result[batchIdx][featureIdx] = fp16Bytes
				case "FP32":
					bytes := make([]byte, 4)
					binary.LittleEndian.PutUint32(bytes, math.Float32bits(val))
					result[batchIdx][featureIdx] = bytes
				default:
					return nil, fmt.Errorf("unsupported numeric type %s for float32 data", dataType)
				}
			}
		}
		return result, nil

	default:
		flattened, err := serializer.FlattenMatrixByType(data, dataType)
		if err != nil {
			return nil, err
		}

		switch dataType {
		case "FP16":
			if f32slice, ok := flattened.([]float32); ok {
				batchSize := 1
				featureCount := len(f32slice)
				result := make([][][]byte, batchSize)
				result[0] = make([][]byte, featureCount)
				for i, val := range f32slice {
					fp16Bytes, err := serializer.Float32ToFloat16Bytes(val)
					if err != nil {
						return nil, err
					}
					result[0][i] = fp16Bytes
				}
				return result, nil
			}
		case "BYTES":
			if byteSlice, ok := flattened.([][]byte); ok {
				result := make([][][]byte, 1)
				result[0] = byteSlice
				return result, nil
			}
		}

		return nil, fmt.Errorf("unsupported data format: %T for type %s", data, dataType)
	}
}

// getElementSize returns the byte size of a single element for the given data type
func getElementSize(dataType string) int {
	switch strings.ToUpper(dataType) {
	case "FP32", "TYPE_FP32":
		return 4
	case "FP64", "TYPE_FP64":
		return 8
	case "INT32", "TYPE_INT32":
		return 4
	case "INT64", "TYPE_INT64":
		return 8
	case "INT16", "TYPE_INT16":
		return 2
	case "INT8", "TYPE_INT8":
		return 1
	case "UINT32", "TYPE_UINT32":
		return 4
	case "UINT64", "TYPE_UINT64":
		return 8
	case "UINT16", "TYPE_UINT16":
		return 2
	case "UINT8", "TYPE_UINT8":
		return 1
	case "BOOL", "TYPE_BOOL":
		return 1
	case "FP16", "TYPE_FP16":
		return 2
	default:
		return 0
	}
}

// reshapeDataForBatch reshapes flattened data to preserve batch dimension
func reshapeDataForBatch(data interface{}, dims []int64) interface{} {
	if len(dims) == 0 {
		return data
	}

	batchSize := dims[0]
	featureDims := dims[1:]

	elementsPerBatch := int64(1)
	for _, dim := range featureDims {
		elementsPerBatch *= dim
	}

	var dataSlice []interface{}
	switch v := data.(type) {
	case []interface{}:
		dataSlice = v
	case []string:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []float32:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []float64:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []int32:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []int64:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	default:
		return data
	}

	var result [][]interface{}
	for i := int64(0); i < batchSize; i++ {
		start := i * elementsPerBatch
		end := start + elementsPerBatch
		if end <= int64(len(dataSlice)) {
			batch := dataSlice[start:end]
			result = append(result, batch)
		}
	}

	return result
}

// convertDimsToIntSlice converts input.Dims to []int, handling nested interfaces and various types
func convertDimsToIntSlice(dims interface{}) ([]int, error) {
	var result []int

	switch v := dims.(type) {
	case []int:
		result = make([]int, len(v))
		copy(result, v)
	case []int64:
		result = make([]int, len(v))
		for i, dim := range v {
			result[i] = int(dim)
		}
	case []interface{}:
		result = make([]int, len(v))
		for i, dim := range v {
			switch d := dim.(type) {
			case int:
				result[i] = d
			case int64:
				result[i] = int(d)
			case float64:
				result[i] = int(d)
			default:
				return nil, fmt.Errorf("unsupported dimension type in slice: %T", d)
			}
		}
	case int:
		result = []int{v}
	case int64:
		result = []int{int(v)}
	case float64:
		result = []int{int(v)}
	default:
		return nil, fmt.Errorf("unsupported dims type: %T", v)
	}

	for i, dim := range result {
		if dim == -1 {
			if i == 0 {
				result[i] = 10
			} else {
				result[i] = 128
			}
			log.Debug().Msgf("Replaced dynamic dimension -1 at position %d with %d", i, result[i])
		} else if dim < 0 {
			result[i] = 1
			log.Debug().Msgf("Replaced negative dimension %d at position %d with 1", dim, i)
		}
	}

	return result, nil
}
