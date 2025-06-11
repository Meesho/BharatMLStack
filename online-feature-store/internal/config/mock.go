package config

import (
	"github.com/stretchr/testify/mock"
)

// MockConfigManager implements Manager interface for testing
type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) GetEntity(entityLabel string) (*Entity, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Entity), args.Error(1)
}

func (m *MockConfigManager) GetAllEntities() []Entity {
	args := m.Called()
	return args.Get(0).([]Entity)
}

func (m *MockConfigManager) GetFeatureGroup(entityLabel, fgLabel string) (*FeatureGroup, error) {
	args := m.Called(entityLabel, fgLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeatureGroup), args.Error(1)
}

func (m *MockConfigManager) GetStore(storeId string) (*Store, error) {
	args := m.Called(storeId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Store), args.Error(1)
}

func (m *MockConfigManager) GetReaderSecurityConfig(id string) (*Property, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Property), args.Error(1)
}

func (m *MockConfigManager) GetWriterSecurityConfig(id string) (*Property, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Property), args.Error(1)
}

func (m *MockConfigManager) GetFeatureGroupColumn(entityLabel, fgLabel, columnLabel string) (*Column, error) {
	args := m.Called(entityLabel, fgLabel, columnLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Column), args.Error(1)
}

func (m *MockConfigManager) GetDistributedCacheConfForEntity(entityLabel string) (*Cache, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Cache), args.Error(1)
}

func (m *MockConfigManager) GetInMemoryCacheConfForEntity(entityLabel string) (*Cache, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Cache), args.Error(1)
}

func (m *MockConfigManager) GetP2PCacheConfForEntity(entityLabel string) (*Cache, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Cache), args.Error(1)
}

func (m *MockConfigManager) GetStores() (*map[string]Store, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*map[string]Store), args.Error(1)
}

func (m *MockConfigManager) GetInMemoryCacheableFeatureGroups(entityLabel string) ([]string, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigManager) GetDistributedCacheableFeatureGroups(entityLabel string) ([]string, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigManager) GetEntityKeyColumnMapping(entityLabel string) (map[string]string, error) {
	args := m.Called(entityLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockConfigManager) GetActiveFeatureSchema(entityLabel, fgLabel string) (*FeatureSchema, error) {
	args := m.Called(entityLabel, fgLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeatureSchema), args.Error(1)
}

func (m *MockConfigManager) GetColumnsForEntityAndFG(entityLabel string, fgId int) ([]string, error) {
	args := m.Called(entityLabel, fgId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigManager) GetPKMapAndPKColumnsForEntity(entityLabel string) (map[string]string, []string, error) {
	args := m.Called(entityLabel)
	return args.Get(0).(map[string]string), args.Get(1).([]string), args.Error(2)
}

func (m *MockConfigManager) GetDefaultValueByte(entityLabel string, fgId int, version int, featureLabel string) ([]byte, error) {
	args := m.Called(entityLabel, fgId, version, featureLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockConfigManager) GetSequenceNo(entityLabel string, fgId int, version int, featureLabel string) (int, error) {
	args := m.Called(entityLabel, fgId, version, featureLabel)
	return args.Int(0), args.Error(1)
}

func (m *MockConfigManager) GetStringLengths(entityLabel string, fgId int, version int) ([]uint16, error) {
	args := m.Called(entityLabel, fgId, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint16), args.Error(1)
}

func (m *MockConfigManager) GetVectorLengths(entityLabel string, fgId int, version int) ([]uint16, error) {
	args := m.Called(entityLabel, fgId, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint16), args.Error(1)
}

func (m *MockConfigManager) GetNumOfFeatures(entityLabel string, fgId int, version int) (int, error) {
	args := m.Called(entityLabel, fgId, version)
	return args.Int(0), args.Error(1)
}

func (m *MockConfigManager) GetActiveVersion(entityLabel string, fgId int) (int, error) {
	args := m.Called(entityLabel, fgId)
	return args.Int(0), args.Error(1)
}

func (m *MockConfigManager) GetNormalizedEntities() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) GetMaxColumnSize(storeId string) (int, error) {
	args := m.Called(storeId)
	return args.Int(0), args.Error(1)
}

func (m *MockConfigManager) GetAllRegisteredClients() (map[string]string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}
