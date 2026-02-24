package zookeeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/go-zookeeper/zk"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type V1 struct {
	conn                     *zk.Conn
	basePath                 string
	config                   interface{}
	appName                  string
	slackNotificationEnabled bool
	channelId                string
	handledPrefix            map[string]bool
}

func newV1ZK(config interface{}) ZK {
	if !viper.IsSet(envAppName) || !viper.IsSet(envZookeeperServer) {
		log.Panic().Msgf("%s or %s is not set", envAppName, envZookeeperServer)
	}
	appName := viper.GetString(envAppName)
	zkServers := viper.GetString(envZookeeperServer)
	zkBasePath := configPath + appName
	servers := strings.Split(zkServers, ",")

	slackNotificationEnabled := false
	channelId := ""
	if viper.IsSet(envSlackNotificationEnabled) {
		slackNotificationEnabled = viper.GetBool(envSlackNotificationEnabled)
		if slackNotificationEnabled {
			if !viper.IsSet(envSlackChannelId) {
				log.Panic().Msgf("Slack channel Id is not set, %v", envSlackChannelId)
			}
			channelId = viper.GetString(envSlackChannelId)
		}
	}

	var updateInterval time.Duration
	var periodicUpdateEnabled bool
	if viper.IsSet(envPeriodicUpdateEnabled) {
		periodicUpdateEnabled = viper.GetBool(envPeriodicUpdateEnabled)
		if periodicUpdateEnabled {
			if !viper.IsSet(envPeriodicUpdateInterval) {
				log.Panic().Msgf("%s is not set", envPeriodicUpdateInterval)
			}
			updateInterval = viper.GetDuration(envPeriodicUpdateInterval)
		}
	}
	conn, _, err := zk.Connect(servers, timeout)
	if err != nil {
		return handleZookeeperConnectionError(err, zkBasePath, config, appName, slackNotificationEnabled, channelId, func(err error) {
			log.Fatal().Msgf("Failed to connect to Zookeeper: %v", err)
		})
	}
	v1Zk := &V1{
		conn:                     conn,
		basePath:                 zkBasePath,
		config:                   config,
		appName:                  appName,
		slackNotificationEnabled: slackNotificationEnabled,
		channelId:                channelId,
	}
	err = v1Zk.updateConfig(config)
	if err != nil {
		return handleZookeeperConnectionError(err, zkBasePath, config, appName, slackNotificationEnabled, channelId, func(err error) {
			log.Fatal().Msgf("Failed to connect to Zookeeper: %v", err)
		})
	}
	if periodicUpdateEnabled {
		go v1Zk.enablePeriodicConfigUpdate(updateInterval)
	}
	return v1Zk
}

func (v *V1) GetConfigInstance() interface{} {
	return v.config
}

func (v *V1) updateConfig(config interface{}) error {
	v.handledPrefix = make(map[string]bool) // This reset is intentional here as we are re-initializing map on each update
	dataMap := make(map[string]string)
	metaMap := make(map[string]string)
	_, err := GetChildren(v.basePath, v.conn, &dataMap, &metaMap)
	if err != nil {
		metric.Count("zookeeper_config_update_error", 1, []string{"error_type:get_children"})
		log.Error().Err(err).Msgf("unable to get children from zk")
		return err
	}
	err = v.handleStruct(&dataMap, &metaMap, config, v.basePath, false)
	if err != nil {
		metric.Count("zookeeper_config_update_error", 1, []string{"error_type:handle_struct"})
		log.Error().Err(err).Msgf("unable to get children from zk")
		return err
	}
	return nil
}

func (v *V1) enablePeriodicConfigUpdate(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for range ticker.C {
		err := v.updateConfig(v.config)
		if err != nil {
			log.Error().Err(err).Msgf("unable to update config from zk")
		}
	}
}

func (v *V1) updateConfigForWatcherEvent(data, nodePath string, deleteNode bool) error {
	panic("implement me")
}

func (v *V1) registerAndWatchNodes(path string) error {
	panic("implement me")
}

func GetChildren(zkAbsolutePath string, conn *zk.Conn, dataMap, metaMap *map[string]string) (*zk.Stat, error) {

	children, _, err := conn.Children(zkAbsolutePath)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting config from zk path %s ", zkAbsolutePath)
		return nil, err
	}

	// Iterate over the child nodes and fetch data for each one
	for _, child := range children {
		nodePath := strings.Join([]string{zkAbsolutePath, child}, "/")
		data, _, err := GetData(nodePath, conn)
		if err != nil {
			return nil, err
		} else if len(data) > 0 {
			key := strings.ToLower(strings.Replace(nodePath, "-", "", -1))
			(*metaMap)[key] = nodePath
			(*dataMap)[key] = string(data)
		} else {
			_, e := GetChildren(nodePath, conn, dataMap, metaMap)
			if e != nil {
				return nil, e
			}
		}
	}
	return nil, nil
}

func GetData(nodePath string, conn *zk.Conn) ([]byte, *zk.Stat, error) {
	data, stat, err := conn.Get(nodePath)
	if err != nil {
		return nil, nil, err
	}
	return data, stat, err
}

func (v *V1) handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error {
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
			err := v.handleStruct(dataMap, metaMap, fieldVal.Addr().Interface(), nPrefix, deleteNode)
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
			err := v.handleMap(dataMap, metaMap, &mapInterfacePtr, nPrefix, deleteNode)
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
			}
		}
	}
	return nil
}

func (v *V1) handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error {
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
		}
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
			delete(*dataMap, key)
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
			delete(*dataMap, key)
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
			}
			delete(*dataMap, key)
		} else if !v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Struct {
			mKey = strings.Split(mKey, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			log.Debug().Msgf("value type is struct and key is not terminal path parameter")
			newValue := reflect.New(valueType).Elem()
			if v.handledPrefix[prefix+"/"+mKey] {
				continue
			}
			v.handledPrefix[prefix+"/"+mKey] = true
			err := v.handleStruct(dataMap, metaMap, newValue.Addr().Interface(), prefix+"/"+mKey, deleteNode)
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
			if v.handledPrefix[prefix+"/"+mKey] {
				continue
			}
			v.handledPrefix[prefix+"/"+mKey] = true
			err := v.handleMap(dataMap, metaMap, &mapPtr, prefix+"/"+mKey, deleteNode)
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
	_, err := v.conn.Set(path, []byte(fmt.Sprintf("%v", value)), -1)
	if err != nil {
		log.Error().Msgf("Failed to set value at node %s: %v", path, err)
		return err
	}
	return nil
}

func (v *V1) SetValues(paths map[string]interface{}) error {
	for path, value := range paths {
		v.SetValue(path, value)
	}
	return nil
}

func (v *V1) CreateNode(path string, value interface{}) error {
	exists, err := v.IsNodeExist(path)
	if exists || err != nil {
		log.Error().Msgf("node already exist, not creating new node, returning")
		return errors.New("node already exist, not creating new node, returning")
	}
	createdPath, err := v.conn.Create(path, []byte(fmt.Sprintf("%v", value)), 0, zk.WorldACL(zk.PermAll))
	log.Debug().Msgf("Path Created: %s", createdPath)
	if err != nil {
		log.Error().Msgf("Failed to create node %s: %v", path, err)
		return err
	}
	return nil
}

func (v *V1) CreateNodes(paths map[string]interface{}) error {
	for path, value := range paths {
		err := v.CreateNode(path, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *V1) IsNodeExist(path string) (bool, error) {
	exist, _, err := v.conn.Exists(path)
	if err != nil {
		return false, err
	}
	return exist, nil
}

func handleZookeeperConnectionError(err error, zkBasePath string, config interface{}, appName string, slackNotificationEnabled bool, channelId string, onHardFailure func(err error)) *V1 {
	isOptional := viper.GetBool(configKeyDynamicSourceOptional)
	if isOptional {
		log.Warn().Err(err).Msgf("Unable to connect to Zookeeper. Falling back to YAML configuration.")
		return &V1{
			conn:                     nil,
			basePath:                 zkBasePath,
			config:                   config,
			appName:                  appName,
			slackNotificationEnabled: slackNotificationEnabled,
			channelId:                channelId,
		}
	} else {
		onHardFailure(err)
		return nil
	}
}
