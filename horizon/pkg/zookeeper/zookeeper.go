package zookeeper

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"

	"github.com/go-zookeeper/zk"
	"github.com/rs/zerolog/log"
)

var (
	conn              *zk.Conn
	WatcherEnabled    string
	ZKChannel         chan []byte
	zkServers         string
	zkWatcher         string
	nodePath          string
	initZKConfigsOnce sync.Once
)

func InitZKConnection(config configs.Configs) {
	initZKConfigsOnce.Do(func() {
		zkServers = config.ZookeeperServer
		zkWatcher = config.ZookeeperWatcher
		nodePath = config.ZookeeperBasePath
	})
	var zkConfig ZKConfig
	if len(zkWatcher) == 0 || len(zkServers) == 0 {
		log.Panic().Msg("Zookeeper server or watcher is not set")
	}
	zkConfig.Server = zkServers
	zkConfig.Watcher = zkWatcher

	servers := strings.Split(zkConfig.Server, ",")
	WatcherEnabled = zkConfig.Watcher
	log.Info().Msgf("Is Watcher Enabled: %s", WatcherEnabled)
	var err error
	conn, _, err = zk.Connect(servers, time.Second*5)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to connect to zk server ")
	}
	ZKChannel = make(chan []byte, 10)
}

func WatchZkNode() {
	// Create a new TreeCache instance
	treeCache := NewTreeCache(conn, nodePath)
	WatchAllNodes(nodePath, treeCache, treeCache.RootNode, treeCache.RootNode.Children)
	mapDataToStruct(treeCache)
}

func mapDataToStruct(treeCache *TreeCache) {
	jsonData := traverseNodes(treeCache.RootNode, "")
	// Convert the JSON data to a byte array
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("Error creating JSON:")
	}
	log.Info().Msgf("JSON : %s", string(jsonBytes))
	ZKChannel <- jsonBytes
}

func traverseNodes(node *Node, prefix string) interface{} {
	currentPath := fmt.Sprintf("%s", node.Path)

	// Check if the node has any children
	if len(node.Children) > 0 {
		// Create a map to hold the JSON data for the current node
		jsonData := make(map[string]interface{})

		// Traverse the children nodes recursively
		for key, child := range node.Children {
			// Create the nested JSON structure for the child node
			childJSON := traverseNodes(child, fmt.Sprintf("%s%s/", currentPath, key))

			// Assign the child JSON to the corresponding key in the current node's JSON data
			jsonData[key] = childJSON
		}

		return jsonData
	}

	// If the node has no children, assign the node data directly to the current path
	return node.Data
}

func Get(nodePath string) ([]byte, *zk.Stat, error) {

	data, stat, err := conn.Get(nodePath)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting config from zk path %s ", nodePath)
		return nil, nil, err
	}
	return data, stat, err
}

func GetChildern(zkPath string) (map[string]string, *zk.Stat, error) {

	children, _, err := conn.Children(zkPath)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting config from zk path %s ", zkPath)
		return nil, nil, err
	}

	stringMap := make(map[string]string)
	// Iterate over the child nodes and fetch data for each one
	for _, child := range children {
		nodePath := strings.Join([]string{zkPath, child}, "/")
		data, _, err := Get(nodePath)
		if err != nil {
			return nil, nil, err
		} else {
			stringMap[child] = string(data)
		}
	}
	return stringMap, nil, nil
}

func NewTreeCache(conn *zk.Conn, rootPath string) *TreeCache {
	return &TreeCache{
		conn: conn,
		RootNode: &Node{
			Path:     rootPath,
			Data:     nil,
			Children: make(map[string]*Node),
		},
	}
}

func WatchAllNodes(nodePath string, tc *TreeCache, node *Node, rootNodeChildren map[string]*Node) {
	// GetApplication the data and set a watch on the node
	data, _, watchCh, err := conn.GetW(nodePath)
	if err != nil {
		log.Error().Err(err).Msg("Error getting data from zk path")
		return
	}

	// Process the node data or trigger an action based on your requirements
	log.Info().Msgf("Node path: %s, Data: %s\n", nodePath, data)

	tc.TreeLock.Lock()
	nodePathSplit := strings.Split(nodePath, "/")
	node.Path = nodePathSplit[len(nodePathSplit)-1]
	node.Data = string(data)
	// tc.nodeMap[nodePath] = node
	tc.TreeLock.Unlock()

	// Retrieve the child nodes
	children, _, childCh, err := conn.ChildrenW(nodePath)
	if err != nil {
		log.Error().Err(err).Msg("Error getting children from zk path")
		return
	}

	if WatcherEnabled == "true" {
		// Start a goroutine to handle watch events for the current node
		go func() {
			for {
				select {
				case event := <-watchCh:
					if event.Type == zk.EventNodeDataChanged {
						// Node data changed, handle it based on your requirements
						log.Info().Msgf("Node data changed: %s\n", event.Path)

						// GetApplication the updated data and trigger an action
						data, _, err := conn.Get(event.Path)
						if err != nil {
							log.Error().Err(err).Msg("Error getting data from zk path")
							continue
						}

						log.Info().Msgf("Updated data for node %s: %s\n", event.Path, data)
						node.Data = string(data)
						mapDataToStruct(tc)
						// Recreate the watch
						_, _, watchCh, err = conn.GetW(event.Path)
						if err != nil {
							log.Error().Err(err).Msg("Error getting data from zk path")
							continue
						}
					}
					//  else if event.Type == zk.EventNodeDeleted {
					// 	// Node deleted, handle it based on your requirements
					// 	gozookeeperlogger.Info(fmt.Sprintf("Node deleted: %s\n", event.Path))
					// 	splitPath := strings.Split(event.Path, "/")
					// 	delete(node.Children, splitPath[len(splitPath)-1])
					// 	// Remove the watch for the deleted node
					// 	conn.ExistsW(event.Path)
					// 	// mapDataToStruct(tc)
					// }
					// TODO: Delete is not working as we have to maintain a parent maping as well to delete the node from parent list
					// This is not burning problem, Will fix this later

				case childEvent := <-childCh:
					if childEvent.Type == zk.EventNodeChildrenChanged {
						// Child nodes changed, handle it based on your requirements
						log.Info().Msgf("Child nodes changed: %s\n", childEvent.Path)

						// Retrieve the updated list of child nodes and the new watch channel
						children, _, _, err := conn.ChildrenW(childEvent.Path)
						if err != nil {
							log.Error().Err(err).Msg("Error getting children from zk path")
							continue
						}

						log.Info().Msgf("Updated child nodes for %s:\n", childEvent.Path)
						for _, child := range children {
							fmt.Println(child)
						}

						// Process the updated child nodes
						for _, child := range children {
							childPath := childEvent.Path + "/" + child
							childNode := &Node{
								Path:     child,
								Data:     string(data),
								Children: make(map[string]*Node),
							}
							// checking if the child Node is already present, Then we will skip the Node and Move forward
							_, ok := node.Children[child]
							if ok {
								continue
							}
							node.Children[child] = childNode
							// Recursively watch child nodes
							WatchAllNodes(childPath, tc, childNode, childNode.Children)
						}
						mapDataToStruct(tc)
						// Recreate the watch
						_, _, watchCh, err = conn.GetW(childEvent.Path)
						if err != nil {
							log.Error().Err(err).Msg("Error getting data from zk path")
							continue
						}
					}
				}

			}
		}()
	}
	// Process the child nodes
	for _, child := range children {
		childPath := nodePath + "/" + child

		childNode := &Node{
			Path:     child,
			Data:     string(data),
			Children: make(map[string]*Node),
		}
		node.Children[child] = childNode
		// Recursively watch child nodes
		WatchAllNodes(childPath, tc, childNode, childNode.Children)
	}
}
