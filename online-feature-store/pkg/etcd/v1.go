package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type V1 struct {
	conn               *clientv3.Client
	basePath           string
	config             interface{}
	appName            string
	HandledPrefix      map[string]string
	WatchPathCallbacks map[string][]interface{}
	dataMap            map[string]string
	metaMap            map[string]string
	mu                 sync.Mutex
}

func newV1Etcd(config interface{}) Etcd {
	if !viper.IsSet(envAppName) || !viper.IsSet(envEtcdServer) {
		log.Panic().Msgf("%s or %s is not set", envAppName, envEtcdServer)
	}
	appName := viper.GetString(envAppName)
	etcdServers := viper.GetString(envEtcdServer)
	etcdBasePath := configPath + appName
	servers := strings.Split(etcdServers, ",")
	var username, password string
	if viper.IsSet(envEtcdUsername) && viper.IsSet(envEtcdPassword) {
		username = viper.GetString(envEtcdUsername)
		password = viper.GetString(envEtcdPassword)
	}

	conn, err := clientv3.New(clientv3.Config{Endpoints: servers, Username: username, Password: password, DialTimeout: timeout, DialKeepAliveTime: timeout, PermitWithoutStream: true})
	if err != nil {
		log.Error().Msgf("failed to create etcd client: %v", err)
	}
	v1Etcd := &V1{
		conn:               conn,
		basePath:           etcdBasePath,
		config:             config,
		appName:            appName,
		WatchPathCallbacks: make(map[string][]interface{}),
	}
	err = v1Etcd.updateConfig(config)
	var watcherEnabled bool
	if viper.IsSet(envWatcherEnabled) {
		watcherEnabled = viper.GetBool(envWatcherEnabled)
		if watcherEnabled {
			v1Etcd.WatchPrefix(context.Background(), etcdBasePath)
		}
	}
	if err != nil {
		log.Panic().Err(err).Msgf("unable to create config from etcd")
	}
	return v1Etcd
}

func (v *V1) GetConfigInstance() interface{} {
	return v.config
}

func (v *V1) updateConfig(config interface{}) error {
	v.HandledPrefix = make(map[string]string)
	dataMap := make(map[string]string)
	metaMap := make(map[string]string)
	err := GetChildren(v.basePath, v.conn, &dataMap, &metaMap)
	if err != nil {
		log.Error().Err(err).Msgf("unable to get children from etcd")
		return err
	}
	err = v.handleStruct(&dataMap, &metaMap, config, v.basePath)
	if err != nil {
		log.Error().Err(err).Msgf("unable to get children from etcd")
		return err
	}
	v.dataMap = dataMap
	v.metaMap = metaMap
	return nil
}

func (v *V1) WatchPrefix(ctx context.Context, prefix string) {
	watchChan := v.conn.Watch(ctx, prefix, clientv3.WithPrefix())
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Msgf("panic in watch prefix: %v", r)
					}
				}()
				for watchResp := range watchChan {
					for _, event := range watchResp.Events {
						v.HandledPrefix = make(map[string]string)
						log.Debug().Msgf("Key: %s | Type: %s | Value: %s", event.Kv.Key, event.Type.String(), event.Kv.Value)
						v.updateMaps(event, string(event.Kv.Key), string(event.Kv.Value))
						err := v.handleStruct(&v.dataMap, &v.metaMap, v.config, v.basePath)
						if err != nil {
							log.Error().Err(err).Msgf("unable to parse struct for etcd, not executing watch callbacks")
							continue
						}
						for key, functions := range v.WatchPathCallbacks {
							watchPath := prefix + key
							if strings.HasPrefix(string(event.Kv.Key), watchPath) {
								for _, value := range functions {
									err = value.(func() error)()
									if err != nil {
										log.Error().Err(err).Msgf("unable to execute the function for path %s", key)
									}
								}
							}
						}
					}
				}
			}()

			//Avoid frequent restarts on panics
			time.Sleep(5 * time.Second)
		}
	}()
}

func (v *V1) updateMaps(event *clientv3.Event, nodePath, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	key := strings.ToLower(strings.Replace(nodePath, "-", "", -1))
	switch event.Type.String() {
	case "PUT":
		v.dataMap[key] = value
		v.metaMap[key] = nodePath
	case "DELETE":
		for k := range v.dataMap {
			if strings.HasSuffix(key, "/") {
				if strings.HasPrefix(k, key) {
					delete(v.dataMap, k)
					delete(v.metaMap, k)
				}
			} else {
				if k == key {
					delete(v.dataMap, k)
					delete(v.metaMap, k)
				}
			}
		}
	}
}

func GetChildren(etcdAbsolutePath string, conn *clientv3.Client, dataMap, metaMap *map[string]string) error {
	children, err := conn.Get(context.Background(), etcdAbsolutePath, clientv3.WithPrefix())

	if err != nil {
		log.Error().Msgf("Error getting config from etcd path %s ", etcdAbsolutePath)
	}
	if err != nil {
		log.Error().Err(err).Msgf("Error getting config from etcd path %s ", etcdAbsolutePath)
		return err
	}

	// Iterate over the child nodes and fetch data for each one
	for _, child := range children.Kvs {
		nodePath := string(child.Key)
		if nodePath == etcdAbsolutePath {
			continue
		}
		data := child.Value
		if len(data) > 0 {
			key := strings.ToLower(strings.Replace(nodePath, "-", "", -1))
			(*metaMap)[key] = nodePath
			(*dataMap)[key] = string(data)
		} else {
			e := GetChildren(nodePath, conn, dataMap, metaMap)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (v *V1) handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
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
		log.Debug().Msgf("handling field %s", field.Name)
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
			err := v.handleStruct(dataMap, metaMap, fieldVal.Addr().Interface(), nPrefix)
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
			err := v.handleMap(dataMap, metaMap, &mapInterfacePtr, nPrefix)
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
			case reflect.Uint | reflect.Uint64:
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
			case reflect.Uint64:
				floatVal, err := strconv.ParseUint(data, 10, 64)
				if err != nil {
					return err
				}
				fieldVal.SetUint(floatVal)
			case reflect.Slice:
				// Check if it's specifically a []byte (slice of uint8)
				if field.Type.Elem().Kind() == reflect.Uint8 {
					byteArray, _ := convertToByteArray(data)
					fieldVal.SetBytes(byteArray)
				}
			default:
				panic("unhandled default case")
			}
		}
	}
	return nil
}

func convertToByteArray(input string) ([]byte, error) {
	input = strings.Trim(input, "[]")
	parts := strings.Split(input, " ")
	result := make([]byte, len(parts))
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value at index %d: %v", i, err)
		}
		result[i] = byte(value)
	}
	return result, nil
}

func (v *V1) handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
	log.Debug().Msgf("handleMap called with prefix %s", prefix)
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
		//If key - prefix is not terminal path parameter and value type is struct, implies no data, recursively call the handleStruct function
		//If key - prefix is not terminal path parameter and value type is map, implies no data, recursively call the handleMap function
		//If key - prefix is not terminal path parameter and value type is neither struct nor map, schema violation, return error
		if !strings.HasPrefix(key, prefix) {
			continue
		} //catalog/1/4524

		mKey := strings.TrimPrefix(key, prefix+"/")
		originalPath, ok := (*metaMap)[key]
		if !ok {
			return errors.New("original path missing for key " + key + " in metaMap")
		}
		originalPrefix := v.getOriginalPrefix(originalPath, prefix)
		mapKey := strings.TrimPrefix(originalPath, originalPrefix+"/")
		if v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Struct {
			log.Debug().Msgf("value type is struct and key is terminal path parameter")
			newValue := reflect.New(valueType).Elem()
			err := json.Unmarshal([]byte(data), newValue.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", key, err)
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
			//delete(*dataMap, key)
			continue
		} else if v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Map {
			log.Debug().Msgf("value type is map and key is terminal path parameter")
			newValue := reflect.MakeMap(valueType)
			mapPtr := newValue.Interface()
			err := json.Unmarshal([]byte(data), &mapPtr)
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", key, err)
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
			//delete(*dataMap, key)
			continue
		} else if v.isTerminalPathParameter(mKey) {
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
			case reflect.Uint | reflect.Uint64:
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
			case reflect.Uint64:
				floatVal, err := strconv.ParseUint(data, 10, 64)
				if err != nil {
					return err
				}
				val.SetMapIndex(reflect.ValueOf(mapKey), reflect.ValueOf(floatVal))
			default:
				panic("unhandled default case")
			}
			//delete(*dataMap, key)
		} else if !v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Struct { //prefix /config/catalog
			mKey = strings.Split(mKey, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			log.Debug().Msgf("value type is struct and key is not terminal path parameter")
			newValue := reflect.New(valueType).Elem()
			if v.HandledPrefix[prefix+"/"+mKey] == prefix+"/"+mKey {
				continue
			}
			v.HandledPrefix[prefix+"/"+mKey] = prefix + "/" + mKey
			err := v.handleStruct(dataMap, metaMap, newValue.Addr().Interface(), prefix+"/"+mKey)
			if err != nil {
				return err
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
		} else if !v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Map {
			mKey = strings.Split(mKey, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			log.Debug().Msgf("value type is map and key is not terminal path parameter")
			newValue := reflect.MakeMap(valueType)
			mapPtr := newValue.Interface()
			err := v.handleMap(dataMap, metaMap, &mapPtr, prefix+"/"+mKey)
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

func (v *V1) isTerminalPathParameter(key string) bool {
	return !strings.Contains(key, "/")
}

func (v *V1) getOriginalPrefix(originalPath, prefix string) string {
	// Split the paths into segments
	modifiedOriginalPath := strings.TrimPrefix(originalPath, "/")
	modifiedPrefix := strings.TrimPrefix(prefix, "/")
	originalSegments := strings.Split(modifiedOriginalPath, "/")
	prefixSegments := strings.Split(modifiedPrefix, "/")
	// Find the common path
	var commonSegments []string
	for i, segment := range originalSegments {
		modifiedOriginalSeg := strings.ReplaceAll(segment, "-", "")
		if i < len(prefixSegments) && strings.EqualFold(modifiedOriginalSeg, prefixSegments[i]) {
			commonSegments = append(commonSegments, segment)
		} else {
			break
		}
	}

	// Join the common segments to form the path
	return "/" + strings.Join(commonSegments, "/")
}

func (v *V1) SetValue(path string, value interface{}) error {
	_, err := v.conn.Put(context.Background(), path, fmt.Sprintf("%v", value))
	if err != nil {
		log.Error().Msgf("Failed to set value at node %s: %v", path, err)
		return err
	}
	return nil
}

// SetValues sets the values at the given paths
func (v *V1) SetValues(paths map[string]interface{}) error {
	for path, value := range paths {
		err := v.SetValue(path, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateNode creates a node at the given path with the given value
func (v *V1) CreateNode(path string, value interface{}) error {
	exists, err := v.IsNodeExist(path)
	if exists || err != nil {
		log.Error().Msgf("Node already exist, not creating new node, returning")
		return fmt.Errorf("node already exist for %s: %w", path, err)
	}
	response, err := v.conn.Put(context.Background(), path, fmt.Sprintf("%v", value))
	log.Debug().Msgf("Path Created: %v", response)
	if err != nil {
		log.Error().Msgf("Failed to create node %s: %v", path, err)
		return err
	}
	return nil
}

// CreateNodes creates nodes at the given paths with the given values
func (v *V1) CreateNodes(paths map[string]interface{}) error {
	for path, value := range paths {
		err := v.CreateNode(path, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsNodeExist checks if a node exists at the given path
func (v *V1) IsNodeExist(path string) (bool, error) {
	response, err := v.conn.Get(context.Background(), path, clientv3.WithPrefix())
	if err != nil {
		return false, err
	}
	for _, kv := range response.Kvs {
		if strings.Contains(string(kv.Key), path+"/") {
			return true, nil
		}
	}
	return false, nil
}

// RegisterWatchPathCallback registers a callback function to be called when a change is detected in the given path
func (v *V1) RegisterWatchPathCallback(path string, callback func() error) error {
	_, ok := v.WatchPathCallbacks[path]
	if ok {
		v.WatchPathCallbacks[path] = append(v.WatchPathCallbacks[path], callback)
	} else {
		v.WatchPathCallbacks[path] = []interface{}{callback}
	}
	return nil
}
