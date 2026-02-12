package zookeeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/go-zookeeper/zk"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Deprecated: V2 is in experimental phase. Please use the stable V1 implementation instead.
type V2 struct {
	conn                     *zk.Conn
	basePath                 string
	config                   interface{}
	appName                  string
	slackNotificationEnabled bool
	channelId                string
	isStartupFlow            bool
	handledPrefix            map[string]bool
	updateConfigLock         sync.Mutex
}

// Deprecated: newV2ZK is part of the experimental V2 implementation. Please use newV1ZK instead.
func newV2ZK(config interface{}) ZK {
	if !viper.IsSet(envAppName) || !viper.IsSet(envZookeeperServer) {
		log.Panic().Msgf("%s or %s is not set", envAppName, envZookeeperServer)
	}
	appName := viper.GetString(envAppName)
	zkServers := viper.GetString(envZookeeperServer)
	zkBasePath := configPath + appName
	servers := strings.Split(zkServers, ",")

	slackNotificationEnabled := false
	channelId := ""

	watcherEnabled := true
	if viper.IsSet(envWatcherEnabled) {
		watcherEnabled = viper.GetBool(envWatcherEnabled)
	}

	conn, _, err := zk.Connect(servers, timeout)
	if err != nil {
		return handleZookeeperConnectionErrorV2(err, zkBasePath, config, appName, slackNotificationEnabled, channelId, func(err error) {
			log.Fatal().Msgf("Failed to connect to Zookeeper: %v", err)
		})
	}
	v2Zk := &V2{
		conn:                     conn,
		basePath:                 zkBasePath,
		config:                   config,
		appName:                  appName,
		slackNotificationEnabled: slackNotificationEnabled,
		channelId:                channelId,
		isStartupFlow:            true,
	}

	err = v2Zk.updateConfig(config)
	if err != nil {
		return handleZookeeperConnectionErrorV2(err, zkBasePath, config, appName, slackNotificationEnabled, channelId, func(err error) {
			log.Fatal().Msgf("Failed to connect to Zookeeper: %v", err)
		})
	}

	if watcherEnabled {
		err := v2Zk.registerAndWatchNodes(zkBasePath)
		if err != nil {
			log.Error().Err(err).Msgf("unable to register watchers for all zk nodes")
		}
		v2Zk.isStartupFlow = false
	}
	return v2Zk
}

func (v *V2) GetConfigInstance() interface{} {
	return v.config
}

// Unimplemented method as not used in V2
func (v *V2) enablePeriodicConfigUpdate(duration time.Duration) {
	panic("implement me")
}

func (v *V2) GetChildren(zkAbsolutePath string, conn *zk.Conn, dataMap, metaMap *map[string]string) (*zk.Stat, error) {

	children, _, err := conn.Children(zkAbsolutePath)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting config from zk path %s ", zkAbsolutePath)
		return nil, err
	}

	// Iterate over the child nodes and fetch data for each one
	for _, child := range children {
		nodePath := strings.Join([]string{zkAbsolutePath, child}, "/")
		data, _, err := v.GetData(nodePath, conn)
		if err != nil {
			return nil, err
		} else if len(data) > 0 {
			key := strings.ToLower(strings.Replace(nodePath, "-", "", -1))
			(*metaMap)[key] = nodePath
			(*dataMap)[key] = string(data)
		} else {
			_, e := v.GetChildren(nodePath, conn, dataMap, metaMap)
			if e != nil {
				return nil, e
			}
		}
	}
	return nil, nil
}

func (v *V2) GetData(nodePath string, conn *zk.Conn) ([]byte, *zk.Stat, error) {
	data, stat, err := conn.Get(nodePath)
	if err != nil {
		return nil, nil, err
	}
	return data, stat, err
}

func (v *V2) updateConfig(config interface{}) error {
	v.handledPrefix = make(map[string]bool) // This reset is intentional here as we are re-initializing map on each update
	dataMap := make(map[string]string)
	metaMap := make(map[string]string)
	_, err := v.GetChildren(v.basePath, v.conn, &dataMap, &metaMap)
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

func (v *V2) updateConfigForWatcherEvent(data, nodePath string, deleteNode bool) error {
	v.updateConfigLock.Lock()               // Acquire lock
	defer v.updateConfigLock.Unlock()       // Ensure the lock is released
	v.handledPrefix = make(map[string]bool) // This reset is intentional here as we are re-initializing map on each update
	dataMap := make(map[string]string)
	metaMap := make(map[string]string)

	key := strings.ToLower(strings.Replace(nodePath, "-", "", -1))
	(metaMap)[key] = nodePath
	(dataMap)[key] = data

	err := v.handleStruct(&dataMap, &metaMap, v.config, v.basePath, deleteNode)
	if err != nil {
		metric.Count("zookeeper_config_update_error", 1, []string{"error_type:handle_struct"})
		log.Error().Err(err).Msgf("unable to update zk")
		return err
	}
	return nil
}

// registerAndWatchNodes recursively registers and watches a ZooKeeper node and its children for changes.
//
// It first retrieves the children of the specified path. If the path has no children, the method sets up
// data-watcher and child-watcher for that node only. If children exist, it sets up data-watcher and child-watcher for both the node and it's children.
//
// Parameters:
//   - path: The ZooKeeper path to register and watch.
//
// Returns:
//   - error: An error is returned if any of the watcher setup operations fail.
//
// The method ensures that both data and child changes are monitored for the given node,
// allowing real-time updates to configurations or states based on ZooKeeper events.
func (v *V2) registerAndWatchNodes(path string) error {
	// Check for children first
	children, _, err := v.conn.Children(path)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to retrieve children for path: %s", path)
		return err
	}
	if len(children) == 0 {
		err = v.watchNodeWithChildren(path)
		if err != nil {
			return err
		}
		return v.watchNodeWithData(path)
	} else {
		err = v.watchNodeWithData(path)
		if err != nil {
			return err
		}
		return v.watchNodeWithChildren(path)
	}
}

// Method to handle leaf nodes (no children)
func (v *V2) watchNodeWithData(path string) error {
	// If no children, it's a leaf node, watch for data changes
	data, _, dataWatcher, err := v.conn.GetW(path)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to watch data on path: %s", path)
		return err
	}
	// Watch for data changes in the leaf node
	go func() {
		defer func() {
			if r := recover(); r != nil {
				metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + path, "error_type:data_update_failure"})
				log.Error().Msgf("Error occured while updating the data for the path: %s and error: %v\n, stacktrace - %s", path, r, debug.Stack())
			}
		}()

		for {
			select {
			case event := <-dataWatcher:
				if event.Type == zk.EventNodeDataChanged {
					st := time.Now()
					metric.Incr(metric.TagZkRealtimeTotalUpdateEvent, []string{"path:" + event.Path, "event:" + event.Type.String()})
					log.Info().Msgf("Data changed for path: %s", path)
					// Re-register
					_, _, dataWatcher, err = v.conn.GetW(path)
					if err != nil {
						log.Error().Err(err).Msgf("Failed to re-register watch data on path: %s", path)
						metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:re-register_data_watcher_failure"})
						return
					}

					// Fetch the updated data after the change
					updatedData, _, err := v.conn.Get(path)
					if err != nil {
						log.Error().Err(err).Msgf("Failed to get updated data on path: %s", path)
						metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:get_data_failure"})
					} else {
						er := v.updateConfigForWatcherEvent(string(updatedData), path, false)
						if er != nil {
							log.Error().Err(err).Msgf("Failed to populate updated data of path: %s in zk config", path)
							metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:data_update_failure"})
						} else {
							metric.Incr(metric.TagZkRealtimeSuccessEvent, []string{"path:" + event.Path, "event:" + event.Type.String()})
						}
					}
					metric.Timing(metric.TagZkRealtimeEventUpdateLatency, time.Since(st), []string{"path:" + event.Path, "event:" + event.Type.String()})
				} else if event.Type == zk.EventNodeDeleted {
					st := time.Now()
					metric.Incr(metric.TagZkRealtimeTotalUpdateEvent, []string{"path:" + event.Path, "event:" + event.Type.String()})

					er := v.updateConfigForWatcherEvent(string(data), path, true)
					if er != nil {
						log.Error().Err(err).Msgf("Failed to delete data of path: %s in zk config", path)
						metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:node_deletion_failure"})
					} else {
						metric.Incr(metric.TagZkRealtimeSuccessEvent, []string{"path:" + event.Path, "event:" + event.Type.String()})
					}
					metric.Timing(metric.TagZkRealtimeEventUpdateLatency, time.Since(st), []string{"path:" + event.Path, "event:" + event.Type.String()})
				}
			}
		}
	}()

	if !v.isStartupFlow {
		children, _, err := v.conn.Children(path)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get children on path: %s", path)
		}

		if len(children) == 0 {
			er := v.updateConfigForWatcherEvent(string(data), path, false)
			if er != nil {
				log.Error().Err(err).Msgf("Failed to populate data of newly created path: %s in zk config", path)
			}
		}
	}
	return nil
}

// Method to handle nodes with children
func (v *V2) watchNodeWithChildren(path string) error {
	// Check for children first
	children, _, childWatcher, err := v.conn.ChildrenW(path)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to register children watcher for path: %s", path)
		return err
	}

	// If node has children, watch children changes
	go func() {
		defer func() {
			if r := recover(); r != nil {
				metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + path, "error_type:children_update_failure"})
				log.Error().Msgf("Error occured while updating the children for the path: %s and error: %v\n, stacktrace - %s", path, r, debug.Stack())
			}
		}()

		for {
			select {
			case event := <-childWatcher:
				if event.Type == zk.EventNodeChildrenChanged {
					st := time.Now()
					metric.Incr(metric.TagZkRealtimeTotalUpdateEvent, []string{"path:" + event.Path, "event:" + event.Type.String()})

					// Check for children first
					_, _, childWatcher, err = v.conn.ChildrenW(path)
					if err != nil {
						log.Error().Err(err).Msgf("Failed to re-register watcher for children on path: %s", path)
						metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:re-register_child_watcher_failure"})
						return
					}

					// Fetch the updated children after the change
					newChildren, _, err := v.conn.Children(path)
					if err != nil {
						log.Error().Err(err).Msgf("Failed to get updated children on path: %s", path)
						metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:get_children_failure"})
					} else {
						log.Info().Msgf("Updated children on node %s: %+v", path, newChildren)
						// Detect if new nodes are created or deleted by comparing with previous children list
						added := v.findNewlyCreatedChildren(children, newChildren)
						// Handle newly added children (EventNodeCreated)
						for _, newChild := range added {
							childPath := path + "/" + newChild
							log.Info().Msgf("New child node added: %s", childPath)

							// Register watcher for new child node
							err := v.registerAndWatchNodes(childPath)
							if err != nil {
								metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"path:" + event.Path, "event:" + event.Type.String(), "error_type:children_update_failure"})
								return
							}
						}
						// Update children reference for the next event check
						children = newChildren
						metric.Incr(metric.TagZkRealtimeSuccessEvent, []string{"path:" + event.Path, "event:" + event.Type.String()})
					}
					metric.Timing(metric.TagZkRealtimeEventUpdateLatency, time.Since(st), []string{"path:" + event.Path, "event:" + event.Type.String()})
				}
			}
		}
	}()

	// Recursively watch each child
	for _, child := range children {
		childPath := path + "/" + child
		err := v.registerAndWatchNodes(childPath)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to register watcher for childPath: %s", childPath)
			return err
		}
	}
	return nil
}

func (v *V2) handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error {
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

		if deleteNode && ok {
			fieldVal.Set(reflect.Zero(field.Type))
			continue
		}
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
				metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"deployable_name:" + v.appName, "error_type:json_unmarshal_struct_failure", "key:" + nPrefix})
			}
			continue
		} else if ok && field.Type.Kind() == reflect.Map {
			log.Debug().Msgf("field %s is map and Data in current path", field.Name)
			err := json.Unmarshal([]byte(data), fieldVal.Addr().Interface())
			if err != nil {
				metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"deployable_name:" + v.appName, "error_type:json_unmarshal_map_failure", "key:" + nPrefix})
			}
			continue
		} else if !ok && field.Type.Kind() == reflect.Map {
			log.Debug().Msgf("field %s is map and no Data in current path", field.Name)

			// Check if the map is nil, initialize if necessary
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.MakeMap(field.Type))
			}
			mapInterfacePtr := fieldVal.Interface()
			err := v.handleMap(dataMap, metaMap, &mapInterfacePtr, nPrefix, deleteNode)
			if err != nil {
				return err
			}
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

func (v *V2) handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error {
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

		if deleteNode && v.isTerminalPathParameter(mKey) {
			log.Debug().Msgf("Deleting node with key %s", mapKey)
			val.SetMapIndex(reflect.ValueOf(mapKey), reflect.Value{}) // This deletes the key from the map
			delete(*dataMap, key)                                     // Remove the corresponding key from dataMap as well
			continue
		}

		if v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Struct {
			log.Debug().Msgf("value type is struct and key is terminal path parameter")
			existingVal := val.MapIndex(reflect.ValueOf(mapKey))
			var structVal reflect.Value
			if (existingVal.IsValid()) && (existingVal.Kind() == reflect.Struct) {
				// Create an addressable copy of the struct
				structCopy := reflect.New(valueType).Elem()
				structCopy.Set(existingVal)
				structVal = structCopy
			} else {
				newValue := reflect.New(valueType).Elem()
				structVal = newValue
			}
			err := json.Unmarshal([]byte(data), structVal.Addr().Interface())
			if err != nil {
				metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"deployable_name:" + v.appName, "error_type:json_unmarshal_struct_failure", "key:" + key})
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), structVal)
			delete(*dataMap, key)
			continue
		} else if v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Map {
			log.Debug().Msgf("value type is map and key is terminal path parameter")
			existingVal := val.MapIndex(reflect.ValueOf(mapKey))
			var mapVal reflect.Value
			if existingVal.IsValid() && !existingVal.IsNil() {
				// Create an addressable copy of the map
				mapVal = reflect.MakeMap(valueType)
				for _, key := range existingVal.MapKeys() {
					mapVal.SetMapIndex(key, existingVal.MapIndex(key))
				}
			} else {
				newValue := reflect.MakeMap(valueType)
				mapVal = newValue
			}
			err := json.Unmarshal([]byte(data), &mapVal)
			if err != nil {
				metric.Incr(metric.TagZkRealtimeFailureEvent, []string{"deployable_name:" + v.appName, "error_type:json_unmarshal_map_failure", "key:" + key})
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), mapVal)
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

			existingVal := val.MapIndex(reflect.ValueOf(mapKey))
			var structVal reflect.Value
			if (existingVal.IsValid()) && (existingVal.Kind() == reflect.Struct) {
				// Create an addressable copy of the struct
				structCopy := reflect.New(valueType).Elem()
				structCopy.Set(existingVal)
				structVal = structCopy
			} else {
				newValue := reflect.New(valueType).Elem()
				structVal = newValue
			}

			if v.handledPrefix[prefix+"/"+mKey] {
				continue
			}
			v.handledPrefix[prefix+"/"+mKey] = true
			err := v.handleStruct(dataMap, metaMap, structVal.Addr().Interface(), prefix+"/"+mKey, deleteNode)
			if err != nil {
				return err
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), structVal)
		} else if !v.isTerminalPathParameter(mKey) && valueType.Kind() == reflect.Map {
			mKey = strings.Split(mKey, "/")[0]
			mapKey = strings.Split(mapKey, "/")[0]
			log.Debug().Msgf("value type is map and key is not terminal path parameter")

			existingVal := val.MapIndex(reflect.ValueOf(mapKey))
			var mapVal reflect.Value
			// If mapKey does not exist or is nil, initialize a new map; otherwise, use the existing one
			if !existingVal.IsValid() || existingVal.IsNil() {
				newValue := reflect.MakeMap(valueType)
				mapVal = newValue
			} else {
				// Create an addressable copy of the map
				mapVal = reflect.MakeMap(valueType)
				for _, key := range existingVal.MapKeys() {
					mapVal.SetMapIndex(key, existingVal.MapIndex(key))
				}
			}
			if v.handledPrefix[prefix+"/"+mKey] {
				continue
			}
			v.handledPrefix[prefix+"/"+mKey] = true

			mapInterfacePtr := mapVal.Interface()
			err := v.handleMap(dataMap, metaMap, &mapInterfacePtr, prefix+"/"+mKey, deleteNode)
			if err != nil {
				return err
			}
			val.SetMapIndex(reflect.ValueOf(mapKey), mapVal)
		} else {
			return errors.New("schema violation")
		}
	}
	return nil
}

func (v *V2) isTerminalPathParameter(key string) bool {
	return !strings.Contains(key, "/")
}

func (v *V2) getOriginalPrefix(originalPath, prefix string) string {
	// Split the paths into segments
	modifiedOriginalPath := strings.TrimPrefix(originalPath, "/")
	modifiedPrefix := strings.TrimPrefix(prefix, "/")
	originalSegments := strings.Split(modifiedOriginalPath, "/")
	prefixSegments := strings.Split(modifiedPrefix, "/")
	// Find the common path
	var commonSegments []string
	for i, segment := range originalSegments {
		modifiedOriginalSeg := strings.ReplaceAll(segment, "-", "")
		if i < len(prefixSegments) && strings.ToLower(modifiedOriginalSeg) == strings.ToLower(prefixSegments[i]) {
			commonSegments = append(commonSegments, segment)
		} else {
			break
		}
	}

	// Join the common segments to form the path
	return "/" + strings.Join(commonSegments, "/")
}

// Utility function to find added and deleted children
func (v *V2) findNewlyCreatedChildren(oldChildren, newChildren []string) (createdNodes []string) {
	oldSet := make(map[string]struct{})
	for _, child := range oldChildren {
		oldSet[child] = struct{}{}
	}
	newCreatedNodes := make([]string, 0)
	for _, child := range newChildren {
		// If the new child wasn't in the old set, it's newly added
		if _, exists := oldSet[child]; !exists {
			newCreatedNodes = append(newCreatedNodes, child)
		}
	}
	return newCreatedNodes
}

func (v *V2) SetValue(path string, value interface{}) error {
	_, err := v.conn.Set(path, []byte(fmt.Sprintf("%v", value)), -1)
	if err != nil {
		log.Error().Msgf("Failed to set value at node %s: %v", path, err)
		return err
	}
	return nil
}

func (v *V2) SetValues(paths map[string]interface{}) error {
	for path, value := range paths {
		_ = v.SetValue(path, value)
	}
	return nil
}

func (v *V2) CreateNode(path string, value interface{}) error {
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

func (v *V2) CreateNodes(paths map[string]interface{}) error {
	for path, value := range paths {
		err := v.CreateNode(path, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *V2) IsNodeExist(path string) (bool, error) {
	exist, _, err := v.conn.Exists(path)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// Deprecated: handleZookeeperConnectionErrorV2 is part of the experimental V2 implementation. Please use handleZookeeperConnectionError instead.
func handleZookeeperConnectionErrorV2(err error, zkBasePath string, config interface{}, appName string, slackNotificationEnabled bool, channelId string, onHardFailure func(err error)) *V2 {
	isOptional := viper.GetBool(configKeyDynamicSourceOptional)
	if isOptional {
		log.Warn().Err(err).Msgf("Unable to connect to Zookeeper. Falling back to YAML configuration.")
		return &V2{
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
