package config

import (
	"github.com/stretchr/testify/mock"
)

// MockConfig is a Mock implementation of the ConfigManager interface.
type MockConfig struct {
	mock.Mock
}

// GetEntity ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetEntity(entityLabel string) (*Entity, error) {
	args := m.Called(entityLabel)
	return args.Get(0).(*Entity), args.Error(1)
}

// GetFeatureGroup ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetFeatureGroup(entityLabel, fgLabel string) (*FeatureGroup, error) {
	args := m.Called(entityLabel, fgLabel)
	return args.Get(0).(*FeatureGroup), args.Error(1)
}

// GetStore ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetStore(storeId int) (*Store, error) {
	args := m.Called(storeId)
	return args.Get(0).(*Store), args.Error(1)
}

// GetReaderSecurityConfig ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetReaderSecurityConfig(id string) (*Property, error) {
	args := m.Called(id)
	return args.Get(0).(*Property), args.Error(1)
}

// GetWriterSecurityConfig ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetWriterSecurityConfig(id string) (*Property, error) {
	args := m.Called(id)
	return args.Get(0).(*Property), args.Error(1)
}

// GetFeatureGroupColumn ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetFeatureGroupColumn(entityLabel, fgLabel, columnLabel string) (*Column, error) {
	args := m.Called(entityLabel, fgLabel, columnLabel)
	return args.Get(0).(*Column), args.Error(1)
}

// GetDistributedCacheConfForEntity ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetDistributedCacheConfForEntity(entityLabel string) (*Cache, error) {
	args := m.Called(entityLabel)
	return args.Get(0).(*Cache), args.Error(1)
}

// GetInMemoryCacheConfForEntity ConfigMocks the GetStoreConf method.
func (m *MockConfig) GetInMemoryCacheConfForEntity(entityLabel string) (*Cache, error) {
	args := m.Called(entityLabel)
	return args.Get(0).(*Cache), args.Error(1)
}
