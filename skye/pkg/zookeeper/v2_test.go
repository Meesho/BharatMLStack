package zookeeper

//
//import (
//	"context"
//	"fmt"
//	"github.com/stretchr/testify/require"
//	"github.com/testcontainers/testcontainers-go"
//	"github.com/testcontainers/testcontainers-go/wait"
//	"strings"
//	"sync"
//	"testing"
//	"time"
//
//	"github.com/go-zookeeper/zk"
//	"github.com/stretchr/testify/assert"
//)
//
//// setupV2 initializes a Zookeeper container, connects to it, and returns a V2 instance along with cleanup functions.
//func setupV2(t *testing.T) (*V2, func()) {
//	t.Helper()
//
//	// Set up the Zookeeper container
//	ctx := context.Background()
//	zookeeperContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
//		ContainerRequest: testcontainers.ContainerRequest{
//			Image:        "zookeeper:3.6", // Use the Zookeeper Docker image
//			ExposedPorts: []string{"2181/tcp"},
//			WaitingFor:   wait.ForListeningPort("2181/tcp").WithStartupTimeout(5 * time.Minute),
//		},
//		Started: true,
//	})
//	assert.NoError(t, err)
//
//	// Get the container's exposed port
//	host, err := zookeeperContainer.Host(ctx)
//	assert.NoError(t, err)
//	port, err := zookeeperContainer.MappedPort(ctx, "2181/tcp")
//	assert.NoError(t, err)
//
//	// Connect to the existing Zookeeper instance
//	zkConn, _, err := zk.Connect([]string{host + ":" + port.Port()}, time.Minute*5)
//	assert.NoError(t, err)
//
//	// Create the V2 instance
//	v2 := &V2{
//		conn: zkConn,
//	}
//
//	// Cleanup function
//	cleanup := func() {
//		zkConn.Close()
//		assert.NoError(t, zookeeperContainer.Terminate(ctx))
//	}
//
//	return v2, cleanup
//}
//
//func TestIsNodeExist_NodeExists(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Create a generic node for testing
//	zkPath := "/test-node"
//	_, err := v2.conn.Create(zkPath, []byte("node-data"), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Test the IsNodeExist method
//	exists, err := v2.IsNodeExist(zkPath)
//	assert.NoError(t, err)
//	assert.True(t, exists)
//
//	// Clean up
//	err = v2.conn.Delete(zkPath, -1)
//	assert.NoError(t, err)
//}
//
//func TestIsNodeExist_NodeDoesNotExist(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Test the IsNodeExist method for a non-existent node
//	zkPath := "/non-existent-node"
//	exists, err := v2.IsNodeExist(zkPath)
//	assert.NoError(t, err)
//	assert.False(t, exists)
//}
//
//func TestIsNodeExist_InvalidPath(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Test the IsNodeExist method with an invalid path
//	invalidPath := ""
//	_, err := v2.IsNodeExist(invalidPath)
//	assert.Error(t, err)
//}
//
//func TestIsNodeExist_ConnectionLost(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Close the ZooKeeper connection to simulate connection loss
//	v2.conn.Close()
//
//	// Test the IsNodeExist method after connection is lost
//	zkPath := "/test-node"
//	_, err := v2.IsNodeExist(zkPath)
//	assert.Error(t, err)
//}
//
//func TestCreateNodes_AllNodesCreatedSuccessfully(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define nodes to create
//	nodes := map[string]interface{}{
//		"/node1": "value1",
//		"/node2": "value2",
//		"/node3": "value3",
//	}
//
//	// Call CreateNodes
//	err := v2.CreateNodes(nodes)
//	assert.NoError(t, err)
//
//	// Verify all nodes were created
//	for path, expectedValue := range nodes {
//		data, _, err := v2.conn.Get(path)
//		assert.NoError(t, err)
//		assert.Equal(t, expectedValue, string(data))
//
//		// Clean up
//		err = v2.conn.Delete(path, -1)
//		assert.NoError(t, err)
//	}
//}
//
//func TestCreateNodes_PartialFailure(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define nodes, including one with an invalid path
//	nodes := map[string]interface{}{
//		"/valid-node": "valid-value",
//		"":            "invalid-value",
//	}
//
//	// Call CreateNodes
//	err := v2.CreateNodes(nodes)
//	assert.Error(t, err)
//}
//
//func TestCreateNodes_NoNodesProvided(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Call CreateNodes with an empty map
//	nodes := map[string]interface{}{}
//	err := v2.CreateNodes(nodes)
//	assert.NoError(t, err)
//}
//
//func TestCreateNodes_ConnectionLost(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define nodes to create
//	nodes := map[string]interface{}{
//		"/node1": "value1",
//		"/node2": "value2",
//	}
//
//	// Simulate connection loss
//	v2.conn.Close()
//
//	// Call CreateNodes
//	err := v2.CreateNodes(nodes)
//	assert.Error(t, err)
//}
//
//func TestCreateNode_NodeCreatedSuccessfully(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define the node path and value
//	nodePath := "/test-node"
//	nodeValue := "test-value"
//
//	// Call CreateNode
//	err := v2.CreateNode(nodePath, nodeValue)
//	assert.NoError(t, err)
//
//	// Verify the node was created
//	data, _, err := v2.conn.Get(nodePath)
//	assert.NoError(t, err)
//	assert.Equal(t, nodeValue, string(data))
//
//	// Clean up
//	err = v2.conn.Delete(nodePath, -1)
//	assert.NoError(t, err)
//}
//
//func TestCreateNode_NodeAlreadyExists(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define the node path and value
//	nodePath := "/test-node"
//	nodeValue := "test-value"
//
//	// Create the node first
//	_, err := v2.conn.Create(nodePath, []byte(nodeValue), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Attempt to create the node again
//	err = v2.CreateNode(nodePath, "new-value")
//	assert.Error(t, err)
//	assert.EqualError(t, err, "node already exist, not creating new node, returning")
//
//	// Verify the original data remains unchanged
//	data, _, err := v2.conn.Get(nodePath)
//	assert.NoError(t, err)
//	assert.Equal(t, nodeValue, string(data))
//
//	// Clean up
//	err = v2.conn.Delete(nodePath, -1)
//	assert.NoError(t, err)
//}
//
//func TestCreateNode_InvalidPath(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define an invalid node path
//	invalidPath := ""
//	nodeValue := "test-value"
//
//	// Call CreateNode
//	err := v2.CreateNode(invalidPath, nodeValue)
//	assert.Error(t, err)
//}
//
//func TestCreateNode_ConnectionLost(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define the node path and value
//	nodePath := "/test-node"
//	nodeValue := "test-value"
//
//	// Simulate connection loss
//	v2.conn.Close()
//
//	// Call CreateNode
//	err := v2.CreateNode(nodePath, nodeValue)
//	assert.Error(t, err)
//}
//
//func TestSetValues_Success(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define test paths and values
//	paths := map[string]interface{}{
//		"/node1": "value1",
//		"/node2": "value2",
//	}
//
//	// Pre-create the nodes to set values on
//	for path, _ := range paths {
//		_, err := v2.conn.Create(path, []byte("initial-value"), 0, zk.WorldACL(zk.PermAll))
//		assert.NoError(t, err)
//	}
//
//	// Call SetValues
//	err := v2.SetValues(paths)
//	assert.NoError(t, err)
//
//	// Verify the values are updated
//	for path, expectedValue := range paths {
//		data, _, err := v2.conn.Get(path)
//		assert.NoError(t, err)
//		assert.Equal(t, expectedValue, string(data))
//	}
//
//	// Clean up
//	for path, _ := range paths {
//		err := v2.conn.Delete(path, -1)
//		assert.NoError(t, err)
//	}
//}
//
//func TestFindNewlyCreatedChildren_NewNodes(t *testing.T) {
//	v2 := &V2{}
//
//	oldChildren := []string{"child1", "child2", "child3"}
//	newChildren := []string{"child1", "child2", "child3", "child4", "child5"}
//
//	expectedNewChildren := []string{"child4", "child5"}
//	newNodes := v2.findNewlyCreatedChildren(oldChildren, newChildren)
//
//	assert.ElementsMatch(t, expectedNewChildren, newNodes, "Expected newly created children to match")
//}
//
//func TestFindNewlyCreatedChildren_NoNewNodes(t *testing.T) {
//	v2 := &V2{}
//
//	oldChildren := []string{"child1", "child2", "child3"}
//	newChildren := []string{"child1", "child2", "child3"}
//
//	expectedNewChildren := []string{}
//	newNodes := v2.findNewlyCreatedChildren(oldChildren, newChildren)
//
//	assert.ElementsMatch(t, expectedNewChildren, newNodes, "Expected no newly created children")
//}
//
//func TestFindNewlyCreatedChildren_EmptyOldChildren(t *testing.T) {
//	v2 := &V2{}
//
//	oldChildren := []string{}
//	newChildren := []string{"child1", "child2"}
//
//	expectedNewChildren := []string{"child1", "child2"}
//	newNodes := v2.findNewlyCreatedChildren(oldChildren, newChildren)
//
//	assert.ElementsMatch(t, expectedNewChildren, newNodes, "Expected all new children when old children are empty")
//}
//
//func TestFindNewlyCreatedChildren_EmptyNewChildren(t *testing.T) {
//	v2 := &V2{}
//
//	oldChildren := []string{"child1", "child2"}
//	newChildren := []string{}
//
//	expectedNewChildren := []string{}
//	newNodes := v2.findNewlyCreatedChildren(oldChildren, newChildren)
//
//	assert.ElementsMatch(t, expectedNewChildren, newNodes, "Expected no new children when new children are empty")
//}
//
//func TestFindNewlyCreatedChildren_BothEmpty(t *testing.T) {
//	v2 := &V2{}
//
//	oldChildren := []string{}
//	newChildren := []string{}
//
//	expectedNewChildren := []string{}
//	newNodes := v2.findNewlyCreatedChildren(oldChildren, newChildren)
//
//	assert.ElementsMatch(t, expectedNewChildren, newNodes, "Expected no new children when both old and new children are empty")
//}
//
//func TestFindNewlyCreatedChildren_PartialOverlap(t *testing.T) {
//	v2 := &V2{}
//
//	oldChildren := []string{"child1", "child3"}
//	newChildren := []string{"child1", "child2", "child3", "child4"}
//
//	expectedNewChildren := []string{"child2", "child4"}
//	newNodes := v2.findNewlyCreatedChildren(oldChildren, newChildren)
//
//	assert.ElementsMatch(t, expectedNewChildren, newNodes, "Expected only children not in oldChildren to be newly created")
//}
//
//func TestGetOriginalPrefix_NoMatchingPrefix(t *testing.T) {
//	v2 := &V2{}
//
//	originalPath := "/config/pdp-iop-web/node1"
//	prefix := "/other-path"
//	expected := "/"
//
//	result := v2.getOriginalPrefix(originalPath, prefix)
//	assert.Equal(t, expected, result, "Expected no matching prefix")
//}
//
//func TestGetOriginalPrefix_HandlesHyphens(t *testing.T) {
//	v2 := &V2{}
//
//	originalPath := "/config/pdp-iop-web/node1"
//	prefix := "/config/pdpiopweb"
//	expected := "/config/pdp-iop-web"
//
//	result := v2.getOriginalPrefix(originalPath, prefix)
//	assert.Equal(t, expected, result, "Expected matching with hyphen handling")
//}
//
//func TestGetOriginalPrefix_EmptyOriginalPath(t *testing.T) {
//	v2 := &V2{}
//
//	originalPath := ""
//	prefix := "/config/pdp-iop-web"
//	expected := "/"
//
//	result := v2.getOriginalPrefix(originalPath, prefix)
//	assert.Equal(t, expected, result, "Expected '/' for empty original path")
//}
//
//func TestGetOriginalPrefix_EmptyPrefix(t *testing.T) {
//	v2 := &V2{}
//
//	originalPath := "/config/pdp-iop-web/node1"
//	prefix := ""
//	expected := "/"
//
//	result := v2.getOriginalPrefix(originalPath, prefix)
//	assert.Equal(t, expected, result, "Expected '/' for empty prefix")
//}
//
//func TestGetOriginalPrefix_BothEmpty(t *testing.T) {
//	v2 := &V2{}
//
//	originalPath := ""
//	prefix := ""
//	expected := "/"
//
//	result := v2.getOriginalPrefix(originalPath, prefix)
//	assert.Equal(t, expected, result, "Expected '/' for both empty inputs")
//}
//
//func TestIsTerminalPathParameter_NoSlash(t *testing.T) {
//	v2 := &V2{}
//
//	key := "node1"
//	result := v2.isTerminalPathParameter(key)
//	assert.True(t, result, "Expected true for a key with no slash")
//}
//
//func TestIsTerminalPathParameter_WithSingleSlash(t *testing.T) {
//	v2 := &V2{}
//
//	key := "config/node1"
//	result := v2.isTerminalPathParameter(key)
//	assert.False(t, result, "Expected false for a key with a single slash")
//}
//
//func TestIsTerminalPathParameter_WithMultipleSlashes(t *testing.T) {
//	v2 := &V2{}
//
//	key := "config/pdp-iop-web/node1"
//	result := v2.isTerminalPathParameter(key)
//	assert.False(t, result, "Expected false for a key with multiple slashes")
//}
//
//func TestIsTerminalPathParameter_EmptyKey(t *testing.T) {
//	v2 := &V2{}
//
//	key := ""
//	result := v2.isTerminalPathParameter(key)
//	assert.True(t, result, "Expected true for an empty key (no slash)")
//}
//
//func TestIsTerminalPathParameter_SlashOnly(t *testing.T) {
//	v2 := &V2{}
//
//	key := "/"
//	result := v2.isTerminalPathParameter(key)
//	assert.False(t, result, "Expected false for a key that is just a slash")
//}
//
//func TestIsTerminalPathParameter_TrailingSlash(t *testing.T) {
//	v2 := &V2{}
//
//	key := "node1/"
//	result := v2.isTerminalPathParameter(key)
//	assert.False(t, result, "Expected false for a key with a trailing slash")
//}
//
//func TestIsTerminalPathParameter_LeadingSlash(t *testing.T) {
//	v2 := &V2{}
//
//	key := "/node1"
//	result := v2.isTerminalPathParameter(key)
//	assert.False(t, result, "Expected false for a key with a leading slash")
//}
//
//func TestIsTerminalPathParameter_SpecialCharacters(t *testing.T) {
//	v2 := &V2{}
//
//	key := "node@1"
//	result := v2.isTerminalPathParameter(key)
//	assert.True(t, result, "Expected true for a key with special characters but no slash")
//}
//
//func TestIsTerminalPathParameter_NumericKey(t *testing.T) {
//	v2 := &V2{}
//
//	key := "12345"
//	result := v2.isTerminalPathParameter(key)
//	assert.True(t, result, "Expected true for a purely numeric key with no slash")
//}
//
//func Test_GetData(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Create a sample node with data
//	testNodePath := "/testNode"
//	testNodeData := []byte("sample data")
//
//	// Ensure the node exists and has data
//	_, err := v2.conn.Create(testNodePath, testNodeData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	t.Run("Retrieve existing node data", func(t *testing.T) {
//		data, stat, err := v2.GetData(testNodePath, v2.conn)
//		assert.NoError(t, err)
//		assert.NotNil(t, stat)
//		assert.Equal(t, testNodeData, data, "Data should match the expected value")
//	})
//
//	t.Run("Handle non-existent node", func(t *testing.T) {
//		nonExistentPath := "/nonExistentNode"
//		data, stat, err := v2.GetData(nonExistentPath, v2.conn)
//		assert.Error(t, err, "Expected an error for non-existent node")
//		assert.Nil(t, stat, "Stat should be nil for non-existent node")
//		assert.Nil(t, data, "Data should be nil for non-existent node")
//	})
//
//	t.Run("Handle invalid connection", func(t *testing.T) {
//		// Define the node path
//		testNodePath := "/testNode"
//
//		// Simulate connection loss by closing the ZooKeeper connection
//		v2.conn.Close()
//
//		// Attempt to get data using the closed connection
//		data, stat, err := v2.GetData(testNodePath, v2.conn)
//
//		// Validate the error and assert nil values for data and stat
//		assert.Error(t, err, "Expected an error due to connection loss")
//		assert.Nil(t, stat, "Stat should be nil due to connection loss")
//		assert.Nil(t, data, "Data should be nil due to connection loss")
//	})
//}
//
//func TestGetChildren(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	t.Run("Handle valid children nodes", func(t *testing.T) {
//		// Create a root node with children
//		rootNodePath := "/test-root"
//		childNode1Path := rootNodePath + "/child1"
//		childNode2Path := rootNodePath + "/child2"
//
//		err := v2.CreateNode(rootNodePath, "")
//		assert.NoError(t, err)
//
//		err = v2.CreateNode(childNode1Path, "child1-data")
//		assert.NoError(t, err)
//
//		err = v2.CreateNode(childNode2Path, "child2-data")
//		assert.NoError(t, err)
//
//		// Prepare dataMap and metaMap
//		dataMap := make(map[string]string)
//		metaMap := make(map[string]string)
//
//		// Call GetChildren
//		stat, err := v2.GetChildren(rootNodePath, v2.conn, &dataMap, &metaMap)
//		assert.NoError(t, err)
//		assert.Nil(t, stat, "Stat should be nil as the method returns nil for it")
//
//		// Validate dataMap and metaMap
//		expectedData := map[string]string{
//			"/test-root/child1": "child1-data",
//			"/test-root/child2": "child2-data",
//		}
//		for k, v := range expectedData {
//			assert.Equal(t, v, dataMap[strings.ToLower(strings.Replace(k, "-", "", -1))])
//			assert.Equal(t, k, metaMap[strings.ToLower(strings.Replace(k, "-", "", -1))])
//		}
//	})
//
//	t.Run("Handle empty children nodes", func(t *testing.T) {
//		// Create a root node with no children
//		rootNodePath := "/test-empty-root"
//		err := v2.CreateNode(rootNodePath, "")
//		assert.NoError(t, err)
//
//		// Prepare dataMap and metaMap
//		dataMap := make(map[string]string)
//		metaMap := make(map[string]string)
//
//		// Call GetChildren
//		stat, err := v2.GetChildren(rootNodePath, v2.conn, &dataMap, &metaMap)
//		assert.NoError(t, err)
//		assert.Nil(t, stat, "Stat should be nil as the method returns nil for it")
//
//		// Validate dataMap and metaMap are empty
//		assert.Empty(t, dataMap, "dataMap should be empty for nodes with no data")
//		assert.Empty(t, metaMap, "metaMap should be empty for nodes with no data")
//	})
//
//	t.Run("Handle invalid node path", func(t *testing.T) {
//		// Define an invalid node path
//		invalidNodePath := ""
//
//		// Prepare dataMap and metaMap
//		dataMap := make(map[string]string)
//		metaMap := make(map[string]string)
//
//		// Call GetChildren
//		stat, err := v2.GetChildren(invalidNodePath, v2.conn, &dataMap, &metaMap)
//		assert.Error(t, err, "Expected an error for invalid node path")
//		assert.Nil(t, stat, "Stat should be nil for invalid node path")
//		assert.Empty(t, dataMap, "dataMap should be empty for invalid node path")
//		assert.Empty(t, metaMap, "metaMap should be empty for invalid node path")
//	})
//
//	t.Run("Handle connection loss", func(t *testing.T) {
//		// Define a valid node path
//		nodePath := "/test-connection-lost"
//
//		// Create the node
//		err := v2.CreateNode(nodePath, "test-data")
//		assert.NoError(t, err)
//
//		// Simulate connection loss
//		v2.conn.Close()
//
//		// Prepare dataMap and metaMap
//		dataMap := make(map[string]string)
//		metaMap := make(map[string]string)
//
//		// Call GetChildren
//		stat, err := v2.GetChildren(nodePath, v2.conn, &dataMap, &metaMap)
//		assert.Error(t, err, "Expected an error due to connection loss")
//		assert.Nil(t, stat, "Stat should be nil due to connection loss")
//		assert.Empty(t, dataMap, "dataMap should be empty due to connection loss")
//		assert.Empty(t, metaMap, "metaMap should be empty due to connection loss")
//	})
//}
//
//func TestV2_handleStruct_EmptyMaps(t *testing.T) {
//	type ZKConfigs struct {
//		FY map[string]interface{}
//	}
//
//	dataMap := map[string]string{}
//	metaMap := map[string]string{}
//
//	config := &ZKConfigs{
//		FY: map[string]interface{}{},
//	}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.Empty(t, config.FY)
//}
//
//func TestV2_handleStruct_NonStructOutput(t *testing.T) {
//	dataMap := map[string]string{
//		"/config/testservice/key": `{"enabled": true}`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/key": `/config/test-service/key`,
//	}
//
//	var config map[string]interface{}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, &config, "/config/test-service", false)
//	assert.NotNil(t, err)
//	assert.Contains(t, err.Error(), "output must be a pointer to struct")
//}
//
//func TestV2_handleStruct_RecursiveStructHandling(t *testing.T) {
//	type NestedConfig struct {
//		Enabled bool
//	}
//	type ParentConfig struct {
//		Nested NestedConfig
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/nested": `{"enabled": true}`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/nested": `/config/test-service/nested`,
//	}
//
//	config := &ParentConfig{}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.True(t, config.Nested.Enabled)
//}
//
//func TestV2_handleStruct_RecursiveMapHandling(t *testing.T) {
//	type Config struct {
//		MapField map[string]string
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/mapfield": `{"key1": "value1", "key2": "value2"}`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/mapfield": `/config/test-service/mapfield`,
//	}
//
//	config := &Config{}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.Equal(t, "value1", config.MapField["key1"])
//	assert.Equal(t, "value2", config.MapField["key2"])
//}
//
//func TestV2_handleStruct_DeleteNode(t *testing.T) {
//	type Config struct {
//		Enabled bool
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/enabled": `true`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/enabled": `/config/test-service/enabled`,
//	}
//
//	config := &Config{
//		Enabled: true,
//	}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", true)
//	assert.Nil(t, err)
//	assert.False(t, config.Enabled) // Should reset to zero value
//}
//
//func TestV2_handleStruct_InvalidJSON(t *testing.T) {
//	type Config struct {
//		Enabled bool
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/enabled": `invalid_json`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/enabled": `/config/test-service/enabled`,
//	}
//
//	config := &Config{}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
//	assert.NotNil(t, err)
//}
//
//func TestV2_handleStruct_MissingFields(t *testing.T) {
//	type Config struct {
//		Enabled bool
//		Name    string
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/enabled": `true`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/enabled": `/config/test-service/enabled`,
//	}
//
//	config := &Config{}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.True(t, config.Enabled)
//	assert.Empty(t, config.Name) // Should remain zero value
//}
//
//func TestV2_handleStruct_FieldTypeMismatch(t *testing.T) {
//	type Config struct {
//		Enabled bool
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/enabled": `"not_a_boolean"`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/enabled": `/config/test-service/enabled`,
//	}
//
//	config := &Config{}
//	zk := &V2{}
//
//	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
//	assert.NotNil(t, err)
//}
//
//func TestV2_handleMap_EmptyMaps(t *testing.T) {
//	dataMap := map[string]string{}
//	metaMap := map[string]string{}
//
//	var output interface{} = map[string]interface{}{} // Ensure output is initialized properly
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.Empty(t, output)
//}
//
//func TestV2_handleMap_TerminalPathString(t *testing.T) {
//	dataMap := map[string]string{
//		"/config/testservice/key": "value",
//	}
//	metaMap := map[string]string{
//		"/config/testservice/key": "/config/test-service/key",
//	}
//
//	var output interface{} = (map[string]string{"key": "value"})
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.Equal(t, map[string]string{"key": "value"}, output)
//}
//
//func TestV2_handleMap_TerminalPathStruct(t *testing.T) {
//	type Config struct {
//		Field string `json:"field"`
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/key": `{"field":"value"}`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/key": "/config/test-service/key",
//	}
//
//	var output interface{} = map[string]Config{
//		"key": {Field: "value1"},
//	}
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", false)
//	assert.Nil(t, err)
//	assert.Equal(t, map[string]Config{"key": {Field: "value"}}, output)
//}
//
//func TestV2_handleMap_DeleteTerminalNode(t *testing.T) {
//	dataMap := map[string]string{
//		"/config/testservice/key": "value",
//	}
//	metaMap := map[string]string{
//		"/config/testservice/key": "/config/test-service/key",
//	}
//
//	var output interface{} = map[string]string{"key": "value"}
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", true)
//	assert.Nil(t, err)
//	assert.Empty(t, output)
//	assert.Empty(t, dataMap)
//}
//
//func TestV2_handleMap_NestedMap(t *testing.T) {
//	dataMap := map[string]string{
//		"/config/testservice/parent/child": `{"key":"value"}`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/parent/child": "/config/test-service/parent/child",
//	}
//
//	var output interface{} = map[string]map[string]string{
//		"child": {
//			"key": "value",
//		},
//	}
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service/parent", false)
//	assert.Nil(t, err)
//	assert.Equal(t, map[string]map[string]string{
//		"child": {
//			"key": "value",
//		},
//	}, output)
//}
//
//func TestV2_handleMap_NestedStruct(t *testing.T) {
//	type ChildStruct struct {
//		Key string `json:"key"`
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/parent/child": `{"key":"value"}`,
//	}
//	metaMap := map[string]string{
//		"/config/testservice/parent/child": "/config/test-service/parent/child",
//	}
//
//	var output interface{} = map[string]ChildStruct{
//		"child": {
//			Key: "value",
//		},
//	}
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service/parent", false)
//	assert.Nil(t, err)
//	assert.Equal(t, map[string]ChildStruct{
//		"child": {
//			Key: "value",
//		},
//	}, output)
//}
//
//func TestV2_handleMap_InvalidOutputType(t *testing.T) {
//	dataMap := map[string]string{
//		"/config/test-service/key": "value",
//	}
//	metaMap := map[string]string{
//		"/config/test-service/key": "/config/test-service/key",
//	}
//
//	output := "invalid output type"
//	zk := &V2{}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", false)
//	assert.NotNil(t, err)
//	assert.Contains(t, err.Error(), "output must be a pointer to Interface of map")
//}
//
//func TestV2_handleMap_NonTerminalPathStruct(t *testing.T) {
//	type Config struct {
//		Field string `json:"field"`
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/key/field": "value", // Nested path
//	}
//	metaMap := map[string]string{
//		"/config/testservice/key/field": "/config/test-service/key/field",
//	}
//
//	var output interface{} = map[string]Config{
//		"key": {
//			Field: "value1",
//		},
//	} // Non-terminal struct
//
//	zk := &V2{
//		handledPrefix: make(map[string]bool),
//	}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", false)
//	assert.Nil(t, err)
//	expected := map[string]Config{
//		"key": {
//			Field: "value",
//		},
//	}
//	assert.Equal(t, expected, output)
//}
//
//func TestV2_handleMap_NonTerminalPathNestedStruct(t *testing.T) {
//	type NestedConfig struct {
//		FielderOn string `json:"fielderon"`
//	}
//
//	type Config struct {
//		Field NestedConfig `json:"field"`
//	}
//
//	dataMap := map[string]string{
//		"/config/testservice/key/field/fielderon": "value1", // Deeply nested path
//	}
//	metaMap := map[string]string{
//		"/config/testservice/key/field/fielderon": "/config/test-service/key/field/fielderon",
//	}
//
//	var output interface{} = map[string]Config{
//		"key": {
//			Field: NestedConfig{
//				FielderOn: "value",
//			},
//		},
//	} // Non-terminal struct with nested struct
//
//	zk := &V2{
//		handledPrefix: make(map[string]bool),
//	}
//
//	err := zk.handleMap(&dataMap, &metaMap, &output, "/config/test-service", false)
//	assert.Nil(t, err)
//
//	expected := map[string]Config{
//		"key": {
//			Field: NestedConfig{
//				FielderOn: "value1",
//			},
//		},
//	}
//	assert.Equal(t, expected, output)
//}
//
//func TestV2_GetData(t *testing.T) {
//	// Setup Zookeeper and V2 instance
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define test node path and data
//	nodePath := "/test-node"
//	expectedData := "test-data"
//
//	// Create the test node in Zookeeper
//	_, err := v2.conn.Create(nodePath, []byte(expectedData), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Use GetData to retrieve the data
//	data, stat, err := v2.GetData(nodePath, v2.conn)
//	assert.NoError(t, err)
//	assert.NotNil(t, stat)
//	assert.Equal(t, expectedData, string(data))
//}
//
//func TestV2_GetData_NodeDoesNotExist(t *testing.T) {
//	// Setup Zookeeper and V2 instance
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define a non-existent node path
//	nodePath := "/non-existent-node"
//
//	// Attempt to fetch data from the non-existent node
//	data, stat, err := v2.GetData(nodePath, v2.conn)
//
//	// Validate the results
//	assert.Error(t, err)
//	assert.Nil(t, data)
//	assert.Nil(t, stat)
//}
//
//func TestV2_GetData_NodeWithoutData(t *testing.T) {
//	// Setup Zookeeper and V2 instance
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define a node path
//	nodePath := "/empty-node"
//
//	// Create a node without data
//	_, err := v2.conn.Create(nodePath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Fetch data from the node
//	data, stat, err := v2.GetData(nodePath, v2.conn)
//
//	// Validate the results
//	assert.NoError(t, err)
//	assert.NotNil(t, stat)
//	assert.Empty(t, data)
//}
//
//func TestV2_GetData_InvalidConnection(t *testing.T) {
//	// Setup a mock V2 instance with nil connection
//	v2 := &V2{conn: nil}
//
//	// Define a node path
//	nodePath := ""
//
//	// Attempt to fetch data with an invalid connection
//	data, stat, err := v2.GetData(nodePath, v2.conn)
//
//	// Validate the results
//	assert.Error(t, err)
//	assert.Nil(t, data)
//	assert.Nil(t, stat)
//}
//
//func TestV2_GetData_MultipleNodes(t *testing.T) {
//	// Setup Zookeeper and V2 instance
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define multiple nodes and data
//	nodes := map[string]string{
//		"/node1": "data1",
//		"/node2": "data2",
//		"/node3": "data3",
//	}
//
//	// Create nodes with respective data
//	for path, data := range nodes {
//		_, err := v2.conn.Create(path, []byte(data), 0, zk.WorldACL(zk.PermAll))
//		assert.NoError(t, err)
//	}
//
//	// Fetch and validate data for each node
//	for path, expectedData := range nodes {
//		data, stat, err := v2.GetData(path, v2.conn)
//		assert.NoError(t, err)
//		assert.NotNil(t, stat)
//		assert.Equal(t, expectedData, string(data))
//	}
//}
//
//func TestV2_GetChildren_NodeWithChildrenAndData(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	conn := v2.conn
//
//	// Setup Zookeeper nodes
//	parentPath := "/parent"
//	child1Path := "/parent/child1"
//	child2Path := "/parent/child2"
//
//	_, err := conn.Create(parentPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = conn.Create(child1Path, []byte(`{"key":"value1"}`), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = conn.Create(child2Path, []byte(`{"key":"value2"}`), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Initialize dataMap and metaMap
//	dataMap := make(map[string]string)
//	metaMap := make(map[string]string)
//
//	// Invoke GetChildren
//	_, err = v2.GetChildren(parentPath, conn, &dataMap, &metaMap)
//
//	// Assert results
//	assert.NoError(t, err)
//	assert.Equal(t, map[string]string{
//		"/parent/child1": `{"key":"value1"}`,
//		"/parent/child2": `{"key":"value2"}`,
//	}, dataMap)
//
//	assert.Equal(t, map[string]string{
//		"/parent/child1": child1Path,
//		"/parent/child2": child2Path,
//	}, metaMap)
//}
//
//func TestV2_GetChildren_NodeWithoutData(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	conn := v2.conn
//
//	// Setup Zookeeper nodes
//	parentPath := "/parent"
//	childPath := "/parent/child"
//
//	_, err := conn.Create(parentPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = conn.Create(childPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Initialize dataMap and metaMap
//	dataMap := make(map[string]string)
//	metaMap := make(map[string]string)
//
//	// Invoke GetChildren
//	_, err = v2.GetChildren(parentPath, conn, &dataMap, &metaMap)
//
//	// Assert results
//	assert.NoError(t, err)
//	assert.Empty(t, dataMap)
//	assert.Empty(t, metaMap)
//}
//
//func TestV2_GetChildren_NodeDoesNotExist(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	conn := v2.conn
//
//	// Define a non-existent node path
//	nonExistentPath := "/nonexistent"
//
//	// Initialize dataMap and metaMap
//	dataMap := make(map[string]string)
//	metaMap := make(map[string]string)
//
//	// Invoke GetChildren
//	_, err := v2.GetChildren(nonExistentPath, conn, &dataMap, &metaMap)
//
//	// Assert results
//	assert.Error(t, err)
//	assert.Empty(t, dataMap)
//	assert.Empty(t, metaMap)
//}
//
//func TestV2_GetChildren_DeepNestedChildren(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	conn := v2.conn
//
//	// Setup Zookeeper nodes
//	rootPath := "/root"
//	childPath := "/root/child"
//	grandchildPath := "/root/child/grandchild"
//
//	_, err := conn.Create(rootPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = conn.Create(childPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = conn.Create(grandchildPath, []byte(`{"key":"value"}`), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Initialize dataMap and metaMap
//	dataMap := make(map[string]string)
//	metaMap := make(map[string]string)
//
//	// Invoke GetChildren
//	_, err = v2.GetChildren(rootPath, conn, &dataMap, &metaMap)
//
//	// Assert results
//	assert.NoError(t, err)
//	assert.Equal(t, map[string]string{
//		"/root/child/grandchild": `{"key":"value"}`,
//	}, dataMap)
//
//	assert.Equal(t, map[string]string{
//		"/root/child/grandchild": grandchildPath,
//	}, metaMap)
//}
//
//func TestV2_GetChildren_InvalidConnection(t *testing.T) {
//	v2 := &V2{conn: nil} // Invalid connection
//
//	// Define a node path
//	nodePath := ""
//
//	// Initialize dataMap and metaMap
//	dataMap := make(map[string]string)
//	metaMap := make(map[string]string)
//
//	// Invoke GetChildren
//	_, err := v2.GetChildren(nodePath, v2.conn, &dataMap, &metaMap)
//
//	// Assert results
//	assert.Error(t, err)
//	assert.Empty(t, dataMap)
//	assert.Empty(t, metaMap)
//}
//
//func TestV2_UpdateConfig_Success(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Setup Zookeeper nodes
//	parentPath := "/base"
//	childPath := "/base/child"
//
//	_, err := v2.conn.Create(parentPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath, []byte(`{"key":"value"}`), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Prepare a configuration struct to be updated
//	type Config struct {
//		Child map[string]string `json:"child"`
//	}
//	config := &Config{}
//
//	// Set basePath for the test
//	v2.basePath = parentPath
//
//	// Call updateConfig
//	err = v2.updateConfig(config)
//
//	// Assertions
//	assert.NoError(t, err)
//	assert.Equal(t, map[string]string{"key": "value"}, config.Child)
//}
//
//func TestV2_UpdateConfig_GetChildrenError(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Set a non-existent basePath to trigger an error in GetChildren
//	v2.basePath = "/nonexistent"
//
//	// Prepare a configuration struct to be updated
//	type Config struct {
//		Child map[string]string `json:"child"`
//	}
//	config := &Config{}
//
//	// Call updateConfig
//	err := v2.updateConfig(config)
//
//	// Assertions
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "zk: node does not exist")
//}
//
//func TestV2_UpdateConfig_Data(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Setup Zookeeper nodes
//	parentPath := "/base"
//	childPath := "/base/child"
//
//	_, err := v2.conn.Create(parentPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath, []byte("invalid_json"), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Prepare a configuration struct to be updated
//	type Config struct {
//		Child map[string]string `json:"child"`
//	}
//	config := &Config{}
//
//	// Set basePath for the test
//	v2.basePath = parentPath
//
//	// Call updateConfig
//	err = v2.updateConfig(config)
//
//	// Assertions
//	assert.NoError(t, err)
//}
//
//func TestV2_UpdateConfig_EmptyConfig(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Setup an empty Zookeeper configuration
//	parentPath := "/base"
//	_, err := v2.conn.Create(parentPath, nil, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Prepare a configuration struct to be updated
//	type Config struct {
//		Child map[string]string `json:"child"`
//	}
//	config := &Config{}
//
//	// Set basePath for the test
//	v2.basePath = parentPath
//
//	// Call updateConfig
//	err = v2.updateConfig(config)
//
//	// Assertions
//	assert.NoError(t, err)
//}
//
//func TestV2_UpdateConfigForWatcherEvent_Success(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//	v2.config = &Config{}
//
//	// Mock data and path
//	data := `{"field":"value"}`
//	nodePath := "/test-node/child"
//
//	// Set basePath for the test
//	v2.basePath = "/test-node"
//
//	// Call updateConfigForWatcherEvent
//	err := v2.updateConfigForWatcherEvent(data, nodePath, false)
//
//	// Assertions
//	assert.NoError(t, err)
//}
//
//func TestV2_UpdateConfigForWatcherEvent_NodeDeletion(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//	v2.config = &Config{
//		Field: "existingValue",
//	}
//
//	// Mock node path
//	nodePath := "/base/child"
//
//	// Set basePath for the test
//	v2.basePath = "/base"
//
//	// Call updateConfigForWatcherEvent with deleteNode = true
//	err := v2.updateConfigForWatcherEvent("", nodePath, true)
//
//	// Assertions
//	assert.NoError(t, err)
//}
//
//func TestV2_UpdateConfigForWatcherEvent_EmptyData(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//	v2.config = &Config{
//		Field: "existingValue",
//	}
//
//	// Mock node path
//	nodePath := "/base/child"
//
//	// Set basePath for the test
//	v2.basePath = "/base"
//
//	// Call updateConfigForWatcherEvent with empty data
//	err := v2.updateConfigForWatcherEvent("", nodePath, false)
//
//	// Assertions
//	assert.NoError(t, err)
//	assert.Equal(t, "existingValue", v2.config.(*Config).Field) // Should not overwrite with empty data
//}
//
//func TestV2_UpdateConfigForWatcherEvent_MultipleUpdates(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field1 string `json:"field1"`
//		Field2 string `json:"field2"`
//	}
//	v2.config = &Config{}
//
//	// Mock data and paths
//	data1 := `{"field1":"value1"}`
//	nodePath1 := "/base/node1"
//	data2 := `{"field2":"value2"}`
//	nodePath2 := "/base/node2"
//
//	// Set basePath for the test
//	v2.basePath = "/base"
//
//	// Call updateConfigForWatcherEvent for node1
//	err := v2.updateConfigForWatcherEvent(data1, nodePath1, false)
//	assert.NoError(t, err)
//
//	// Call updateConfigForWatcherEvent for node2
//	err = v2.updateConfigForWatcherEvent(data2, nodePath2, false)
//	assert.NoError(t, err)
//}
//
//func TestV2_RegisterAndWatchNodes_NonExistentNode(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Call registerAndWatchNodes on a non-existent node
//	nonExistentPath := "/non-existent-node"
//
//	err := v2.registerAndWatchNodes(nonExistentPath)
//	assert.Error(t, err)
//}
//
//// Test error on GetW when watching a node
//func TestWatchNodeWithData_ErrorOnGetW(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Watch a non-existent node
//	zkPath := "/test-node"
//	err := v2.watchNodeWithData(zkPath)
//	assert.Error(t, err)
//}
//
//// Test data change event
//func TestWatchNodeWithData_DataChanged(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	v2.isStartupFlow = true
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//	v2.config = &Config{}
//
//	// Mock data and path
//	data := `{"field":"value"}`
//	updatedData := `{"field":"value1"}`
//
//	// Create a node with initial data
//	zkPath := "/test-node"
//	v2.basePath = zkPath
//
//	initialData := []byte(data)
//	_, err := v2.conn.Create(zkPath, initialData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Start watching the node with data
//	err = v2.registerAndWatchNodes(zkPath)
//	assert.NoError(t, err)
//
//	// Simulate data change (Update the node data)
//	updatedDataByte := []byte(updatedData)
//	_, err = v2.conn.Set(zkPath, updatedDataByte, -1)
//	assert.NoError(t, err)
//
//	//Wait for the watcher to catch the change (ensure sufficient time for the watcher to react)
//	time.Sleep(2 * time.Second)
//
//	testData, _, er := v2.GetData(zkPath, v2.conn)
//	assert.NoError(t, er)
//	assert.Equal(t, []byte(updatedData), testData)
//
//	// Clean up
//	err = v2.conn.Delete(zkPath, -1)
//	assert.NoError(t, err)
//}
//
//// Test node deletion event
//func TestWatchNodeWithData_NodeDeleted(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	v2.isStartupFlow = true
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//	v2.config = &Config{}
//
//	// Mock data and path
//	data := `{"field":"value"}`
//
//	// Create a node with initial data
//	zkPath := "/test-node"
//	v2.basePath = zkPath
//
//	initialData := []byte(data)
//	_, err := v2.conn.Create(zkPath, initialData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Start watching the node with data
//	err = v2.registerAndWatchNodes(zkPath)
//	assert.NoError(t, err)
//
//	// Simulate node deletion
//	err = v2.conn.Delete(zkPath, -1)
//	require.NoError(t, err)
//
//	//Wait for the watcher to catch the change (ensure sufficient time for the watcher to react)
//	time.Sleep(2 * time.Second)
//
//	_, _, er := v2.GetData(zkPath, v2.conn)
//	assert.Error(t, er)
//}
//
//// Test multiple node deletion in hierarchy
//func TestWatchNodeWithData_MultipleNodesDeleted(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	v2.isStartupFlow = true
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//
//	type Child1 struct {
//		Child2 Config
//	}
//
//	type TestParent struct {
//		Child1 Child1
//	}
//
//	v2.config = &TestParent{}
//
//	// Create a hierarchical structure of nodes
//	parentPath := "/test-parent"
//	childPath1 := parentPath + "/child1"
//	childPath2 := childPath1 + "/child2"
//
//	initialData := []byte(`{"field":"value"}`)
//
//	// Create parent node
//	_, err := v2.conn.Create(parentPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Create child nodes
//	_, err = v2.conn.Create(childPath1, []byte(""), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath2, initialData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	v2.isStartupFlow = false
//	// Start watching the nodes
//	err = v2.registerAndWatchNodes(parentPath)
//	assert.NoError(t, err)
//
//	// Simulate simultaneous deletion of nodes
//	var wg sync.WaitGroup
//	wg.Add(2)
//
//	go func() {
//		defer wg.Done()
//		err := v2.conn.Delete(childPath1, -1)
//		assert.NoError(t, err)
//	}()
//
//	go func() {
//		defer wg.Done()
//		err := v2.conn.Delete(childPath2, -1)
//		assert.NoError(t, err)
//	}()
//
//	// Wait for both deletions to complete
//	wg.Wait()
//
//	// Wait for the watcher to catch the changes (ensure sufficient time for the watcher to react)
//	time.Sleep(2 * time.Second)
//
//	_, _, err = v2.conn.Get(childPath1)
//	assert.Error(t, err)
//
//	_, _, err = v2.conn.Get(childPath2)
//	assert.Error(t, err)
//}
//
//func TestWatchNodeWithChildren_ChildAdded(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	v2.isStartupFlow = true
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//
//	type Child1 struct {
//		Child2 Config
//	}
//
//	type TestParent struct {
//		Child1 Child1
//	}
//
//	v2.config = &TestParent{}
//
//	// Create a hierarchical structure of nodes
//	parentPath := "/test-parent"
//	childPath1 := parentPath + "/child1"
//
//	initialData := []byte(`{"field":"value"}`)
//
//	_, err := v2.conn.Create(parentPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Start watching the parent node
//	err = v2.registerAndWatchNodes(parentPath)
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath1, initialData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Wait for the watcher to detect the child addition
//	time.Sleep(2 * time.Second)
//
//	// Verify the watcher detected the new child
//	children, _, err := v2.conn.Children(parentPath)
//	assert.NoError(t, err)
//	assert.Contains(t, children, "child1")
//}
//
//func TestWatchNodeWithChildren_ChildRemoved(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	v2.isStartupFlow = true
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//
//	type Child1 struct {
//		Child2 Config
//	}
//
//	type TestParent struct {
//		Child1 Child1
//	}
//
//	v2.config = &TestParent{}
//
//	// Create a hierarchical structure of nodes
//	parentPath := "/test-parent"
//	childPath1 := parentPath + "/child1"
//
//	initialData := []byte(`{"field":"value"}`)
//
//	_, err := v2.conn.Create(parentPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath1, initialData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	v2.isStartupFlow = false
//
//	// Start watching the parent node
//	err = v2.registerAndWatchNodes(parentPath)
//	assert.NoError(t, err)
//
//	// Remove the child node
//	err = v2.conn.Delete(childPath1, -1)
//	assert.NoError(t, err)
//
//	// Wait for the watcher to detect the child removal
//	time.Sleep(2 * time.Second)
//
//	// Verify the watcher detected the child removal
//	children, _, err := v2.conn.Children(parentPath)
//	assert.NoError(t, err)
//	assert.NotContains(t, children, "child1")
//}
//
//func TestWatchNodeWithChildren_HierarchicalNodesCreated(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	v2.isStartupFlow = true
//	defer cleanup()
//
//	// Mock configuration
//	type Config struct {
//		Field string `json:"field"`
//	}
//
//	type Child1 struct {
//		Child2 Config
//	}
//
//	type TestParent struct {
//		Child1 Child1
//	}
//
//	v2.config = &TestParent{}
//
//	// Create a hierarchical structure of nodes
//	parentPath := "/test-parent"
//	childPath1 := parentPath + "/child1"
//	childPath2 := childPath1 + "/child2"
//
//	initialData := []byte(`{"field":"value"}`)
//
//	_, err := v2.conn.Create(parentPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	v2.isStartupFlow = false
//
//	// Start watching the nodes
//	err = v2.registerAndWatchNodes(parentPath)
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath1, []byte(""), 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Start watching the nodes
//	err = v2.registerAndWatchNodes(childPath1)
//	assert.NoError(t, err)
//
//	_, err = v2.conn.Create(childPath2, initialData, 0, zk.WorldACL(zk.PermAll))
//	assert.NoError(t, err)
//
//	// Start watching the nodes
//	err = v2.registerAndWatchNodes(childPath2)
//	assert.NoError(t, err)
//
//	// Wait for the watcher to detect all created nodes
//	time.Sleep(3 * time.Second)
//
//	// Verify the watcher detects the entire hierarchical structure
//	childrenLevel1, _, err := v2.conn.Children(parentPath)
//	assert.NoError(t, err)
//	assert.Contains(t, childrenLevel1, "child1")
//
//	childrenLevel2, _, err := v2.conn.Children(childPath1)
//	assert.NoError(t, err)
//	assert.Contains(t, childrenLevel2, "child2")
//
//	// Verify the data for the deepest node
//	data, _, err := v2.conn.Get(childPath2)
//	assert.NoError(t, err)
//	assert.Equal(t, initialData, data)
//}
//
//func TestGetConfigInstance(t *testing.T) {
//	v2, cleanup := setupV2(t)
//	defer cleanup()
//
//	// Define a mock configuration
//	type MockConfig struct {
//		Field1 string
//		Field2 int
//	}
//	mockConfig := &MockConfig{
//		Field1: "test-value",
//		Field2: 42,
//	}
//
//	// Assign mockConfig to v2.config
//	v2.config = mockConfig
//
//	// Call GetConfigInstance and assert the returned value
//	configInstance := v2.GetConfigInstance()
//	assert.NotNil(t, configInstance, "GetConfigInstance should not return nil")
//
//	// Type assertion to verify the returned type
//	retrievedConfig, ok := configInstance.(*MockConfig)
//	assert.True(t, ok, "Returned config should be of type *MockConfig")
//
//	// Validate the fields of the retrieved configuration
//	assert.Equal(t, "test-value", retrievedConfig.Field1)
//	assert.Equal(t, 42, retrievedConfig.Field2)
//}
//
//func BenchmarkSprintf(b *testing.B) {
//	path, child := "home", "user"
//	for i := 0; i < b.N; i++ {
//		_ = fmt.Sprintf("%s/%s", path, child)
//	}
//}
//
//func BenchmarkConcat(b *testing.B) {
//	path, child := "home", "user"
//	for i := 0; i < b.N; i++ {
//		_ = path + "/" + child
//	}
//}
