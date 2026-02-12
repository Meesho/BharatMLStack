package utils

import (
	"strings"
)

var defaultBytes = map[string][]byte{
	"bool":          {0},
	"datatypebool":  {0},
	"int8":          {0},
	"datatypeint8":  {0},
	"uint8":         {0},
	"datatypeuint8": {0},

	"int16":          {0, 0},
	"datatypeint16":  {0, 0},
	"uint16":         {0, 0},
	"datatypeuint16": {0, 0},
	"fp16":           {0, 0},
	"datatypefp16":   {0, 0},

	"int32":          {0, 0, 0, 0},
	"datatypeint32":  {0, 0, 0, 0},
	"uint32":         {0, 0, 0, 0},
	"datatypeuint32": {0, 0, 0, 0},
	"fp32":           {0, 0, 0, 0},
	"datatypefp32":   {0, 0, 0, 0},

	"int64":          {0, 0, 0, 0, 0, 0, 0, 0},
	"datatypeint64":  {0, 0, 0, 0, 0, 0, 0, 0},
	"uint64":         {0, 0, 0, 0, 0, 0, 0, 0},
	"datatypeuint64": {0, 0, 0, 0, 0, 0, 0, 0},
	"fp64":           {0, 0, 0, 0, 0, 0, 0, 0},
	"datatypefp64":   {0, 0, 0, 0, 0, 0, 0, 0},

	"fp8e5m2":         {0},
	"datatypefp8e5m2": {0},
	"fp8e4m3":         {0},
	"datatypefp8e4m3": {0},

	"string":       []byte("0.0"),
	"stringvector": []byte("0.0"),
	"bytes":        []byte("0.0"),
}

func GetDefaultValuesInBytes(dataType string) []byte {
	dt := strings.ToLower(dataType)

	if strings.Contains(dt, "vector") {
		dt = strings.Replace(dt, "vector", "", 1)
		dt = strings.Replace(dt, "datatype", "", 1)
	}

	if val, ok := defaultBytes[dt]; ok {
		return val
	}
	return []byte("0.0")
}

// getHardcodedZeroBytes returns zero-filled bytes for numeric types from defaultBytes,
// or 4 zero bytes (fp32 default) for unknown or non-numeric types (e.g. string).
func getHardcodedZeroBytes(dataType string) []byte {
	dt := strings.ToLower(dataType)
	dt = strings.Replace(dt, "vector", "", 1)
	dt = strings.Replace(dt, "datatype", "", 1)
	dt = strings.TrimSpace(dt)
	if val, ok := defaultBytes[dt]; ok {
		for _, b := range val {
			if b != 0 {
				return make([]byte, 4) // non-numeric (e.g. string/bytes)
			}
		}
		return val
	}
	return make([]byte, 4)
}

// GetDefaultValueByType returns default byte value for scalar types.
// Vectors are handled separately with empty bytes (2-byte size = 0).
func GetDefaultValueByType(dataType string) []byte {
	dt := strings.ToLower(dataType)
	dt = strings.TrimPrefix(dt, "datatype")
	dt = strings.TrimPrefix(dt, "_")
	dt = strings.TrimSpace(dt)

	if val, ok := defaultBytes[dt]; ok {
		return val
	}
	return []byte("0.0")
}

func GetDefaultValuesInBytesForVector(dataType, featureStoreDataType string, shape []int) []byte {
	dt := strings.ToLower(dataType)

	if strings.Contains(strings.ToLower(featureStoreDataType), "vector") {
		numElements := 1
		for _, dim := range shape {
			if dim > 0 {
				numElements *= dim
			}
		}
		val, ok := defaultBytes[dt]
		if !ok {
			val = []byte("0.0")
		}
		result := make([]byte, 0, len(val)*numElements)
		for i := 0; i < numElements; i++ {
			result = append(result, val...)
		}
		return result
	}

	if val, ok := defaultBytes[dt]; ok {
		return val
	}
	return []byte("0.0")
}
