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

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
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

func newV1Etcd(config interface{}, configs *configs.AppConfigs) Etcd {
	if configs.Configs.ApplicationName == "" || configs.Configs.ETCD_SERVER == "" {
		logger.Panic(fmt.Sprintf("%s or %s is not set", configs.Configs.ApplicationName, configs.Configs.ETCD_SERVER), nil)
	}
	appName := configs.Configs.ApplicationName
	etcdServers := configs.Configs.ETCD_SERVER
	etcdBasePath := basePath + appName + "12" + configPath
	servers := strings.Split(etcdServers, ",")
	username := configs.Configs.ETCD_USERNAME
	password := configs.Configs.ETCD_PASSWORD

	conn, err := clientv3.New(clientv3.Config{Endpoints: servers, Username: username, Password: password, DialTimeout: connectionTimeout, DialKeepAliveTime: connectionTimeout, PermitWithoutStream: true})
	if err != nil {
		logger.Error("failed to create etcd client", err)
	}
	v1Etcd := &V1{
		conn:               conn,
		basePath:           etcdBasePath,
		config:             config,
		appName:            appName,
		WatchPathCallbacks: make(map[string][]interface{}),
	}
	err = v1Etcd.UpdateConfig(config)
	if configs.Configs.ETCD_WATCHER_ENABLED {
		v1Etcd.WatchPrefix(context.Background(), etcdBasePath)
	}
	if err != nil {
		logger.Panic("unable to create config from etcd", err)
	}
	return v1Etcd
}

func (v *V1) GetConfigInstance() interface{} {
	return v.config
}

func (v *V1) GetBasePath() string {
	return v.basePath
}

func (v *V1) UpdateConfig(configuration interface{}) error {
	v.HandledPrefix = make(map[string]string)
	dataMap := make(map[string]string)
	metaMap := make(map[string]string)
	err := GetChildren(v.basePath, v.conn, &dataMap, &metaMap)
	if err != nil {
		logger.Error("unable to get children from etcd", err)
		return err
	}
	err = v.HandleStruct(&dataMap, &metaMap, configuration, v.basePath)
	if err != nil {
		logger.Error("unable to get children from etcd", err)
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
						logger.Error("panic in watch prefix", fmt.Errorf("%v", r))
					}
				}()
				for watchResp := range watchChan {
					for _, event := range watchResp.Events {
						v.HandledPrefix = make(map[string]string)
						v.UpdateMaps(event, string(event.Kv.Key), string(event.Kv.Value))
						err := v.HandleStruct(&v.dataMap, &v.metaMap, v.config, v.basePath)
						if err != nil {
							logger.Error("unable to parse struct for etcd, not executing watch callbacks", err)
							continue
						}
						for key, functions := range v.WatchPathCallbacks {
							watchPath := prefix + key
							if strings.HasPrefix(string(event.Kv.Key), watchPath) {
								for _, value := range functions {
									err = value.(func() error)()
									if err != nil {
										logger.Error(fmt.Sprintf("unable to execute the function for path %s", key), err)
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

func (v *V1) UpdateMaps(event *clientv3.Event, nodePath, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	key := strings.ToLower(strings.Replace(nodePath, "-", "", -1))
	switch event.Type.String() {
	case "PUT":
		v.dataMap[key] = value
		v.metaMap[key] = nodePath
	case "DELETE":
		for k := range v.dataMap {
			if strings.HasPrefix(k, key) {
				delete(v.dataMap, k)
				delete(v.metaMap, k)
			}
		}
	}
}

func GetChildren(etcdAbsolutePath string, conn *clientv3.Client, dataMap, metaMap *map[string]string) error {
	children, err := conn.Get(context.Background(), etcdAbsolutePath, clientv3.WithPrefix())

	if err != nil {
		logger.Error(fmt.Sprintf("Error getting config from etcd path %s ", etcdAbsolutePath), err)
	}
	if err != nil {
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

func (v *V1) HandleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
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
			err := v.HandleStruct(dataMap, metaMap, fieldVal.Addr().Interface(), nPrefix)
			if err != nil {
				return err
			}
			continue
		} else if ok && field.Type.Kind() == reflect.Struct {
			err := json.Unmarshal([]byte(data), fieldVal.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", nPrefix, err)
			}
			continue
		} else if ok && field.Type.Kind() == reflect.Map {
			err := json.Unmarshal([]byte(data), fieldVal.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", nPrefix, err)
			}
			continue
		} else if !ok && field.Type.Kind() == reflect.Map {
			newValue := reflect.MakeMap(field.Type)
			mapInterfacePtr := newValue.Interface()
			err := v.HandleMap(dataMap, metaMap, &mapInterfacePtr, nPrefix)
			if err != nil {
				return err
			}
			fieldVal.Set(newValue)
			continue
		} else if ok && field.Type.Kind() == reflect.Interface {
			value := reflect.ValueOf(&data).Elem()
			fieldVal.Set(value)
		} else if ok {
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

func (v *V1) HandleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
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
		} //catalogger/1/4524

		mKey := strings.TrimPrefix(key, prefix+"/")
		originalPath, ok := (*metaMap)[key]
		if !ok {
			return errors.New("original path missing for key " + key + " in metaMap")
		}
		originalPrefix := v.GetOriginalPrefix(originalPath, prefix)
		mapKey := strings.TrimPrefix(originalPath, originalPrefix+"/")
		if v.IsTerminalPathParameter(mKey) && valueType.Kind() == reflect.Struct {
			newValue := reflect.New(valueType).Elem()
			err := json.Unmarshal([]byte(data), newValue.Addr().Interface())
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", key, err)
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
			//delete(*dataMap, key)
			continue
		} else if v.IsTerminalPathParameter(mKey) && valueType.Kind() == reflect.Map {
			newValue := reflect.MakeMap(valueType)
			mapPtr := newValue.Interface()
			err := json.Unmarshal([]byte(data), &mapPtr)
			if err != nil {
				return fmt.Errorf("failed to unmarshal JSON for key %s: %v", key, err)
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
			//delete(*dataMap, key)
			continue
		} else if v.IsTerminalPathParameter(mKey) {
			logger.Info("value type is neither struct nor map and key is terminal path parameter")
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
		} else if !v.IsTerminalPathParameter(mKey) && valueType.Kind() == reflect.Struct { //prefix /config/catalogger
			mKey = strings.Split(mKey, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			logger.Info("value type is struct and key is not terminal path parameter")
			newValue := reflect.New(valueType).Elem()
			if v.HandledPrefix[prefix+"/"+mKey] == prefix+"/"+mKey {
				continue
			}
			v.HandledPrefix[prefix+"/"+mKey] = prefix + "/" + mKey
			err := v.HandleStruct(dataMap, metaMap, newValue.Addr().Interface(), prefix+"/"+mKey)
			if err != nil {
				return err
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), newValue)
		} else if !v.IsTerminalPathParameter(mKey) && valueType.Kind() == reflect.Map {
			mKey = strings.Split(mKey, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			logger.Info("value type is map and key is not terminal path parameter")
			newValue := reflect.MakeMap(valueType)
			mapPtr := newValue.Interface()
			err := v.HandleMap(dataMap, metaMap, &mapPtr, prefix+"/"+mKey)
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

func (v *V1) IsTerminalPathParameter(key string) bool {
	return !strings.Contains(key, "/")
}

func (v *V1) GetOriginalPrefix(originalPath, prefix string) string {
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
		logger.Error(fmt.Sprintf("Failed to set value at node %s: %v", path, err), err)
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
		logger.Error("Node already exist, not creating new node, returning", err)
		return fmt.Errorf("node already exist for %s: %w", path, err)
	}
	response, err := v.conn.Put(context.Background(), path, fmt.Sprintf("%v", value))
	logger.Info(fmt.Sprintf("Path Created: %v", response))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create node %s: %v", path, err), err)
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
