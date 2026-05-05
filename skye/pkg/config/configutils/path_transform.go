package configutils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func NestedMapToPathMap(input map[string]interface{}, currentPath string, result map[string]string) {
	for key, value := range input {
		slashPath := currentPath + "/" + key

		if nestedMap, ok := value.(map[string]interface{}); ok {
			NestedMapToPathMap(nestedMap, slashPath, result)
			continue
		}

		result[slashPath] = valueToString(value)
	}
}

// NormalizePathMap normalizes all keys in the dataMap by converting them to lowercase
// and removing dashes. It builds a mapping between normalized keys and their original form
// in the metaMap.
//
// Parameters:
// - dataMap: The map containing path keys to be normalized
// - metaMap: Map that will be populated with normalized → original key mappings
func NormalizePathMap(dataMap, metaMap map[string]string) {
	originalKeys := make([]string, 0, len(dataMap))
	for key := range dataMap {
		originalKeys = append(originalKeys, key)
	}

	for _, key := range originalKeys {
		normalizedKey := strings.ToLower(strings.Replace(key, "-", "", -1))
		value := dataMap[key]
		dataMap[normalizedKey] = value
		metaMap[normalizedKey] = key
		if key != normalizedKey {
			delete(dataMap, key)
		}
	}
}

// valueToString converts any value to its string representation
// Simple types are converted directly, complex types use JSON marshaling
func valueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case []byte:
		return string(v)
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal value: %v into json", v)
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	}
}
