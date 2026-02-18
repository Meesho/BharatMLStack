package config

import (
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/stretchr/testify/mock"
)

type MockConfigManager struct {
	mock.Mock
}

// Ensure MockConfigManager implements Manager interface
var _ Manager = (*MockConfigManager)(nil)

func (m *MockConfigManager) GetSkyeConfig() (*Skye, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Skye), args.Error(1)
}

func (m *MockConfigManager) GetEntities() (map[string]Models, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]Models), args.Error(1)
}

func (m *MockConfigManager) GetEntityConfig(entity string) (*Models, error) {
	args := m.Called(entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Models), args.Error(1)
}

func (m *MockConfigManager) GetModelConfig(entity, model string) (*Model, error) {
	args := m.Called(entity, model)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Model), args.Error(1)
}

func (m *MockConfigManager) GetVariantConfig(entity, model, variant string) (*Variant, error) {
	args := m.Called(entity, model, variant)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Variant), args.Error(1)
}

func (m *MockConfigManager) GetAllFiltersForActiveVariants(entity string) (map[string]map[string][]Criteria, error) {
	args := m.Called(entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]map[string][]Criteria), args.Error(1)
}

func (m *MockConfigManager) SetVariantOnboarded(entity, model, variant string, onboarded bool) error {
	args := m.Called(entity, model, variant, onboarded)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateVariantState(entity, model, variant string, variantState enums.VariantState) error {
	args := m.Called(entity, model, variant, variantState)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateVariantReadVersion(entity, model, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateVariantWriteVersion(entity, model, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockConfigManager) RegisterStore(confId int, db, embeddingTable, aggregatorTable string) error {
	args := m.Called(confId, db, embeddingTable, aggregatorTable)
	return args.Error(0)
}

func (m *MockConfigManager) RegisterFrequency(frequency string) error {
	args := m.Called(frequency)
	return args.Error(0)
}

func (m *MockConfigManager) RegisterEntity(entity, storeId string) error {
	args := m.Called(entity, storeId)
	return args.Error(0)
}

func (m *MockConfigManager) RegisterModel(entity, model string, embeddingStoreEnabled bool, embeddingStoreTtl int, modelConfig map[string]interface{}, modelType string, kafkaId int, trainingDataPath string, metadata Metadata, jobFrequency string, numberOfPartitions int, failureProducerKafkaId int, topicName string) error {
	args := m.Called(entity, model, embeddingStoreEnabled, embeddingStoreTtl, modelConfig, modelType, kafkaId, trainingDataPath, metadata, jobFrequency, numberOfPartitions, failureProducerKafkaId, topicName)
	return args.Error(0)
}

func (m *MockConfigManager) RegisterVariant(entity, model, variant string, vectorDbConfig VectorDbConfig, vectorDbType string, filter []Criteria, variantType enums.Type, distributedCacheEnabled bool, distributedCacheTtl int, inMemoryCacheEnabled bool, inMemoryCacheTtl int, rtPartition int, rateLimiters RateLimiter) error {
	args := m.Called(entity, model, variant, vectorDbConfig, vectorDbType, filter, variantType, distributedCacheEnabled, distributedCacheTtl, inMemoryCacheEnabled, inMemoryCacheTtl, rtPartition, rateLimiters)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateEmbeddingVersion(entity, model string, version int) error {
	args := m.Called(entity, model, version)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateVariantEmbeddingStoreReadVersion(entity, model, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateVariantEmbeddingStoreWriteVersion(entity, model, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockConfigManager) IsDefaultResponseEnabled(entity, model, variant string) bool {
	args := m.Called(entity, model, variant)
	return args.Bool(0)
}

func (m *MockConfigManager) UpdateVariantStates(entity, model string, variant map[string]string, state int) error {
	args := m.Called(entity, model, variant, state)
	return args.Error(0)
}

func (m *MockConfigManager) UpdatePartitionState(entity, model, partition string, eofState int) error {
	args := m.Called(entity, model, partition, eofState)
	return args.Error(0)
}

func (m *MockConfigManager) GetRateLimiters() map[int]RateLimiter {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]RateLimiter)
}

func (m *MockConfigManager) RegisterWatchPathCallbackWithEvent(path string, callback func(key, value, eventType string) error) error {
	args := m.Called(path, callback)
	return args.Error(0)
}

func (m *MockConfigManager) UpdateVectorDbConfig(entity string, model string, variant string, vectorDbConfig VectorDbConfig) error {
	args := m.Called(entity, model, variant, vectorDbConfig)
	return args.Error(0)
}

func (m *MockConfigManager) GetTestConfig(entity string, model string, variant string) TestConfig {
	args := m.Called(entity, model, variant)
	return args.Get(0).(TestConfig)
}

func (m *MockConfigManager) UpdateRateLimiter(entity string, model string, variant string, burstLimit int, rateLimit int) error {
	args := m.Called(entity, model, variant, burstLimit, rateLimit)
	return args.Error(0)
}
