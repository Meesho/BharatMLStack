package configutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

var handledPrefix = make(map[string]bool)

// MapToStruct maps config values from flattened path maps to a struct.
//
// Parameters:
//   - dataMap: Contains normalized configuration key-value pairs where keys are flattened paths
//     with hyphens removed. For example, "database/db-host" -> "localhost".
//   - metaMap: Maps normalized paths (no hyphens) to original paths (with hyphens).
//     For example, "database/dbhost" -> "database/db-host".
//   - prefix: The current path prefix in the configuration hierarchy being processed.
//   - When first called, prefix is usually empty string ("") for the root.
//   - During recursion, prefix builds up to represent nested paths(that can be used to match keys that
//     start with the given prefix and belong to this struct), like "database" or "database/dbhost".
func MapToStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
	log.Debug().Msgf("handling struct for prefix %s", prefix)
	prefix = strings.ToLower(strings.Replace(prefix, "-", "", -1))
	valPtr := reflect.ValueOf(output)
	//guardrail
	if valPtr.Kind() != reflect.Ptr || valPtr.IsNil() {
		return errors.New("output must be a non-nil pointer")
	}
	//Dereference the pointer
	val := valPtr.Elem()
	//guardrail
	if val.Kind() != reflect.Struct {
		return errors.New("output must be a pointer to struct")
	}
	//Get the type of the struct
	typ := val.Type()
	//Iterate over the fields of the struct
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		nPrefix := strings.ToLower(prefix + "/" + field.Name)
		data, ok := (*dataMap)[nPrefix]

		//If the field is a struct, and current path don't have data, recursively call the function
		//If the field is a struct, and current path have data, unmarshal the data into the struct
		//If the field is a map, and current path have data, unmarshal the data into the field
		//If the field is a map, and current path don't have data, recursively call the handle map
		//If the field is neither struct nor map, and current path have data, unmarshal the data into the field
		if !ok && field.Type.Kind() == reflect.Struct {
			log.Debug().Msgf("field %s is struct and no Data in current path", field.Name)
			err := MapToStruct(dataMap, metaMap, fieldVal.Addr().Interface(), nPrefix)
			if err != nil {
				return err
			}
			continue
		} else if ok && field.Type.Kind() == reflect.Struct {
			log.Debug().Msgf("field %s is struct and Data in current path", field.Name)
			err := json.Unmarshal([]byte(data), fieldVal.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", nPrefix, err)
			}
			continue
		} else if ok && field.Type.Kind() == reflect.Map {
			log.Debug().Msgf("field %s is map and Data in current path", field.Name)
			err := json.Unmarshal([]byte(data), fieldVal.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", nPrefix, err)
			}
			continue
		} else if !ok && field.Type.Kind() == reflect.Map {
			log.Debug().Msgf("field %s is map and no Data in current path", field.Name)
			newValue := reflect.MakeMap(field.Type)
			mapInterfacePtr := newValue.Interface()
			err := mapToMap(dataMap, metaMap, &mapInterfacePtr, nPrefix)
			if err != nil {
				return err
			}
			fieldVal.Set(newValue)
			continue
		} else if ok && field.Type.Kind() == reflect.Interface {
			log.Debug().Msgf("field %s is interface and Data in current path", field.Name)
			value := reflect.ValueOf(&data).Elem()
			fieldVal.Set(value)
		} else if ok {
			log.Debug().Msgf("field %s is neither struct nor map and Data in current path", field.Name)
			switch field.Type.Kind() {
			case reflect.String:
				fieldVal.SetString(data)
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(data)
				if err != nil {
					return err
				}
				fieldVal.SetBool(boolVal)
			case reflect.Float64:
				floatVal, err := strconv.ParseFloat(data, 64)
				if err != nil {
					return err
				}
				fieldVal.SetFloat(floatVal)
			case reflect.Float32:
				floatVal, err := strconv.ParseFloat(data, 32)
				if err != nil {
					return err
				}
				fieldVal.SetFloat(floatVal)
			case reflect.Int, reflect.Int64:
				floatVal, err := strconv.ParseInt(data, 10, 64)
				if err != nil {
					return err
				}
				fieldVal.SetInt(floatVal)
			case reflect.Int8:
				floatVal, err := strconv.ParseInt(data, 10, 8)
				if err != nil {
					return err
				}
				fieldVal.SetInt(floatVal)
			case reflect.Int16:
				floatVal, err := strconv.ParseInt(data, 10, 16)
				if err != nil {
					return err
				}
				fieldVal.SetInt(floatVal)
			case reflect.Int32:
				floatVal, err := strconv.ParseInt(data, 10, 32)
				if err != nil {
					return err
				}
				fieldVal.SetInt(floatVal)
			case reflect.Uint, reflect.Uint64:
				floatVal, err := strconv.ParseUint(data, 10, 64)
				if err != nil {
					return err
				}
				fieldVal.SetUint(floatVal)
			case reflect.Uint8:
				floatVal, err := strconv.ParseUint(data, 10, 8)
				if err != nil {
					return err
				}
				fieldVal.SetUint(floatVal)
			case reflect.Uint16:
				floatVal, err := strconv.ParseUint(data, 10, 16)
				if err != nil {
					return err
				}
				fieldVal.SetUint(floatVal)
			case reflect.Uint32:
				floatVal, err := strconv.ParseUint(data, 10, 32)
				if err != nil {
					return err
				}
				fieldVal.SetUint(floatVal)
			}
		}
	}
	return nil
}

func mapToMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
	log.Debug().Msgf("mapToMap called with prefix %s", prefix)
	prefix = strings.ToLower(strings.Replace(prefix, "-", "", -1))
	valPtr := reflect.ValueOf(output)
	//guardrail
	if valPtr.Kind() != reflect.Ptr || valPtr.IsNil() {
		return errors.New("output must be a non-nil pointer")
	}
	//Dereference the pointer
	valInterface := valPtr.Elem()
	//guardrail
	if valInterface.Kind() != reflect.Interface || valInterface.Elem().Kind() != reflect.Map {
		return errors.New("output must be a pointer to Interface of map")
	}

	//Dereference the interface
	val := valInterface.Elem()
	//Get the type of the map
	typ := val.Type()
	//Get value type of the map
	valueType := typ.Elem()

	//Iterate over the items in dataMap with the prefix
	for key, data := range *dataMap {
		//If key - prefix is terminal path parameter and value type is struct, implies json in data, unmarshal the data into the struct
		//If key - prefix is terminal path parameter and value type is map, implies json in data, unmarshal the data into the map
		//If key - prefix is terminal path parameter and value type is neither struct nor map, implies string in data, set the data into the map
		//If key - prefix is not terminal path parameter and value type is struct, implies no data, recursively call the MapToStruct function
		//If key - prefix is not terminal path parameter and value type is map, implies no data, recursively call the mapToMap function
		//If key - prefix is not terminal path parameter and value type is neither struct nor map, schema violation, return error
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// The config path relative to the current prefix
		// If prefix="database" and key="database/username",
		// then relativeConfigPath="username"
		relativeConfigPath := strings.TrimPrefix(key, prefix+"/")
		originalPath, ok := (*metaMap)[key]
		if !ok {
			return errors.New("original path missing for key " + key + " in metaMap")
		}
		originalPrefix := getOriginalPrefix(originalPath, prefix)
		mapKey := strings.TrimPrefix(originalPath, originalPrefix+"/")
		if isTerminalPathParameter(relativeConfigPath) && valueType.Kind() == reflect.Struct {
			log.Debug().Msgf("value type is struct and key is terminal path parameter")
			newValue := reflect.New(valueType).Elem()
			err := json.Unmarshal([]byte(data), newValue.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", key, err)
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
			delete(*dataMap, key)
			continue
		} else if isTerminalPathParameter(relativeConfigPath) && valueType.Kind() == reflect.Map {
			log.Debug().Msgf("value type is map and key is terminal path parameter")

			// Create a new pointer to a map of the expected type
			mapPtr := reflect.New(valueType).Interface()

			// Unmarshal into the correctly typed map
			err := json.Unmarshal([]byte(data), mapPtr)
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", key, err)
			}

			// Set the value from the dereferenced pointer
			val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(reflect.ValueOf(mapPtr).Elem().Interface()))
			delete(*dataMap, key)
			continue
		} else if isTerminalPathParameter(relativeConfigPath) {
			log.Debug().Msgf("value type is neither struct nor map and key is terminal path parameter")
			switch valueType.Kind() {
			case reflect.String:
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(data))
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(data)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(boolVal))
			case reflect.Float64:
				floatVal, err := strconv.ParseFloat(data, 64)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Float32:
				floatVal, err := strconv.ParseFloat(data, 32)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Int, reflect.Int64:
				floatVal, err := strconv.ParseInt(data, 10, 64)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Int8:
				floatVal, err := strconv.ParseInt(data, 10, 8)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Int16:
				floatVal, err := strconv.ParseInt(data, 10, 16)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Int32:
				floatVal, err := strconv.ParseInt(data, 10, 32)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Uint, reflect.Uint64:
				floatVal, err := strconv.ParseUint(data, 10, 64)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Uint8:
				floatVal, err := strconv.ParseUint(data, 10, 8)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Uint16:
				floatVal, err := strconv.ParseUint(data, 10, 16)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			case reflect.Uint32:
				floatVal, err := strconv.ParseUint(data, 10, 32)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			}
			delete(*dataMap, key)
		} else if !isTerminalPathParameter(relativeConfigPath) && valueType.Kind() == reflect.Struct {
			relativeConfigPath = strings.Split(relativeConfigPath, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			log.Debug().Msgf("value type is struct and key is not terminal path parameter")
			newValue := reflect.New(valueType).Elem()
			if handledPrefix[prefix+"/"+relativeConfigPath] {
				continue
			}
			handledPrefix[prefix+"/"+relativeConfigPath] = true
			err := MapToStruct(dataMap, metaMap, newValue.Addr().Interface(), prefix+"/"+relativeConfigPath)
			if err != nil {
				return err
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
		} else if !isTerminalPathParameter(relativeConfigPath) && valueType.Kind() == reflect.Map {
			relativeConfigPath = strings.Split(relativeConfigPath, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			log.Debug().Msgf("value type is map and key is not terminal path parameter")
			newValue := reflect.MakeMap(valueType)
			mapPtr := newValue.Interface()
			if handledPrefix[prefix+"/"+relativeConfigPath] {
				continue
			}
			handledPrefix[prefix+"/"+relativeConfigPath] = true
			err := mapToMap(dataMap, metaMap, &mapPtr, prefix+"/"+relativeConfigPath)
			if err != nil {
				return err
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
		} else {
			return errors.New("schema violation")
		}
	}
	return nil
}

func getOriginalPrefix(originalPath, prefix string) string {
	modifiedOriginalPath := strings.TrimPrefix(originalPath, "/")
	modifiedPrefix := strings.TrimPrefix(prefix, "/")
	originalSegments := strings.Split(modifiedOriginalPath, "/")
	prefixSegments := strings.Split(modifiedPrefix, "/")
	var commonSegments []string
	for i, segment := range originalSegments {
		modifiedOriginalSeg := strings.ReplaceAll(segment, "-", "")
		if i < len(prefixSegments) && strings.EqualFold(modifiedOriginalSeg, prefixSegments[i]) {
			commonSegments = append(commonSegments, segment)
		} else {
			break
		}
	}

	return "/" + strings.Join(commonSegments, "/")
}

func isTerminalPathParameter(key string) bool {
	return !strings.Contains(key, "/")
}
