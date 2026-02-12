package zookeeper

import (
	"time"

	"github.com/stretchr/testify/mock"
)

type MockZK struct {
	mock.Mock
}

func (m *MockZK) GetConfigInstance() interface{} {
	ret := m.Called()
	return ret.Get(0)
}

func (m *MockZK) enablePeriodicConfigUpdate(duration time.Duration) {
	m.Called(duration)
}

func (m *MockZK) updateConfig(config interface{}) error {
	ret := m.Called(config)
	return ret.Error(0)
}

func (m *MockZK) handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error {
	ret := m.Called(dataMap, metaMap, output, prefix, deleteNode)
	return ret.Error(0)
}

func (m *MockZK) handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error {
	ret := m.Called(dataMap, metaMap, output, prefix, deleteNode)
	return ret.Error(0)
}

func (m *MockZK) updateConfigForWatcherEvent(data, nodePath string, deleteNode bool) error {
	ret := m.Called(data, nodePath, deleteNode)
	return ret.Error(0)
}

func (m *MockZK) registerAndWatchNodes(path string) error {
	ret := m.Called(path)
	return ret.Error(0)
}

func (m *MockZK) SetValues(paths map[string]interface{}) error {
	ret := m.Called(paths)
	return ret.Error(0)
}

func (m *MockZK) CreateNode(path string, value interface{}) error {
	ret := m.Called(path, value)
	return ret.Error(0)
}

func (m *MockZK) CreateNodes(paths map[string]interface{}) error {
	ret := m.Called(paths)
	return ret.Error(0)
}

func (m *MockZK) SetValue(path string, value interface{}) error {
	ret := m.Called(path, value)
	return ret.Error(0)
}

func (m *MockZK) IsNodeExist(path string) (bool, error) {
	ret := m.Called(path)
	return ret.Get(0).(bool), ret.Error(0)
}
