package etcd

import (
	"github.com/stretchr/testify/mock"
)

// MockEtcd is a mock implementation of the Etcd interface for testing.
// It implements all exported methods. Unexported methods (updateConfig, handleStruct, handleMap)
// are implemented as no-ops since they are not used by SkyeManager.
type MockEtcd struct {
	mock.Mock
}

// Ensure MockEtcd implements Etcd interface
var _ Etcd = (*MockEtcd)(nil)

// GetConfigInstance returns the mock config instance.
func (m *MockEtcd) GetConfigInstance() interface{} {
	args := m.Called()
	return args.Get(0)
}

// SetValue mocks setting a value at the given path.
func (m *MockEtcd) SetValue(path string, value interface{}) error {
	args := m.Called(path, value)
	return args.Error(0)
}

// SetValues mocks setting multiple values.
func (m *MockEtcd) SetValues(paths map[string]interface{}) error {
	args := m.Called(paths)
	return args.Error(0)
}

// CreateNode mocks creating a node at the given path.
func (m *MockEtcd) CreateNode(path string, value interface{}) error {
	args := m.Called(path, value)
	return args.Error(0)
}

// CreateNodes mocks creating multiple nodes.
func (m *MockEtcd) CreateNodes(paths map[string]interface{}) error {
	args := m.Called(paths)
	return args.Error(0)
}

// IsNodeExist mocks checking if a node exists.
func (m *MockEtcd) IsNodeExist(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

// IsLeafNodeExist mocks checking if a leaf node exists.
func (m *MockEtcd) IsLeafNodeExist(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

// RegisterWatchPathCallbackWithEvent mocks registering a watch callback.
func (m *MockEtcd) RegisterWatchPathCallbackWithEvent(path string, callback func(key, value, eventType string) error) error {
	args := m.Called(path, callback)
	return args.Error(0)
}

// updateConfig is a no-op for testing - not used by SkyeManager.
func (m *MockEtcd) updateConfig(config interface{}) error {
	return nil
}

// handleStruct is a no-op for testing - not used by SkyeManager.
func (m *MockEtcd) handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
	return nil
}

// handleMap is a no-op for testing - not used by SkyeManager.
func (m *MockEtcd) handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error {
	return nil
}
