package config

import (
	"encoding/json"
	"testing"

	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/pkg/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetVariantPath(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	// Test indirectly through SetVariantOnboarded which uses getVariantPath
	mockEtcd.On("SetValue", "/config/test-app/entity/e1/models/m1/variants/v1/onboarded", true).Return(nil)

	err := sm.SetVariantOnboarded("e1", "m1", "v1", true)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestGetModelPath(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	// Test indirectly through UpdateEmbeddingVersion which uses getModelPath
	mockEtcd.On("SetValue", "/config/test-app/entity/e1/models/m1/embedding-store-version", 5).Return(nil)

	err := sm.UpdateEmbeddingVersion("e1", "m1", 5)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestGetSkyeConfig_Success(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	expectedSkye := &Skye{
		Entity: map[string]Models{
			"entity1": {StoreId: "store1", Models: map[string]Model{}},
		},
	}
	mockEtcd.On("GetConfigInstance").Return(expectedSkye)

	config, err := sm.GetSkyeConfig()
	assert.NoError(t, err)
	assert.Equal(t, expectedSkye, config)
	mockEtcd.AssertExpectations(t)
}

func TestGetSkyeConfig_CastFailure(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	// Return non-Skye type
	mockEtcd.On("GetConfigInstance").Return("not a Skye")

	config, err := sm.GetSkyeConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to cast etcd config instance to Skye type")
	mockEtcd.AssertExpectations(t)
}

func TestGetSkyeConfig_NilResult(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	// Return nil *Skye - cast succeeds but value is nil
	mockEtcd.On("GetConfigInstance").Return((*Skye)(nil))

	config, err := sm.GetSkyeConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "etcdConf not found")
	mockEtcd.AssertExpectations(t)
}

func TestGetEntities_Success(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	expectedEntities := map[string]Models{
		"entity1": {StoreId: "store1", Models: map[string]Model{}},
	}
	skye := &Skye{Entity: expectedEntities}
	mockEtcd.On("GetConfigInstance").Return(skye)

	entities, err := sm.GetEntities()
	assert.NoError(t, err)
	assert.Equal(t, expectedEntities, entities)
	mockEtcd.AssertExpectations(t)
}

func TestGetEntities_NilEntities(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{Entity: nil}
	mockEtcd.On("GetConfigInstance").Return(skye)

	entities, err := sm.GetEntities()
	assert.Error(t, err)
	assert.Nil(t, entities)
	assert.Contains(t, err.Error(), "entities not found")
	mockEtcd.AssertExpectations(t)
}

func TestGetEntities_CastFailure(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	mockEtcd.On("GetConfigInstance").Return("not Skye")

	entities, err := sm.GetEntities()
	assert.Error(t, err)
	assert.Nil(t, entities)
	mockEtcd.AssertExpectations(t)
}

func TestGetEntityConfig_Found(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	entityModels := Models{StoreId: "store1", Models: map[string]Model{"m1": {}}}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetEntityConfig("entity1")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "store1", config.StoreId)
	mockEtcd.AssertExpectations(t)
}

func TestGetEntityConfig_NotFound(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{Entity: map[string]Models{"entity1": {}}}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetEntityConfig("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "not found")
	mockEtcd.AssertExpectations(t)
}

func TestGetEntityConfig_SkyeConfigError(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	mockEtcd.On("GetConfigInstance").Return("not Skye")

	config, err := sm.GetEntityConfig("entity1")
	assert.Error(t, err)
	assert.Nil(t, config)
	mockEtcd.AssertExpectations(t)
}

func TestGetModelConfig_Found(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	model := Model{TrainingDataPath: "/path/to/data"}
	entityModels := Models{
		Models: map[string]Model{"m1": model},
	}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetModelConfig("entity1", "m1")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "/path/to/data", config.TrainingDataPath)
	mockEtcd.AssertExpectations(t)
}

func TestGetModelConfig_NotFound(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{
		Entity: map[string]Models{"entity1": {Models: map[string]Model{}}},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetModelConfig("entity1", "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "not found")
	mockEtcd.AssertExpectations(t)
}

func TestGetModelConfig_EntityError(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{Entity: map[string]Models{}}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetModelConfig("nonexistent", "m1")
	assert.Error(t, err)
	assert.Nil(t, config)
	mockEtcd.AssertExpectations(t)
}

func TestGetVariantConfig_Found(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	variant := Variant{Enabled: true, Onboarded: true}
	model := Model{Variants: map[string]Variant{"v1": variant}}
	entityModels := Models{Models: map[string]Model{"m1": model}}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetVariantConfig("entity1", "m1", "v1")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.True(t, config.Onboarded)
	mockEtcd.AssertExpectations(t)
}

func TestGetVariantConfig_NotFound(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	model := Model{Variants: map[string]Variant{}}
	entityModels := Models{Models: map[string]Model{"m1": model}}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetVariantConfig("entity1", "m1", "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "not found")
	mockEtcd.AssertExpectations(t)
}

func TestGetVariantConfig_ModelError(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{
		Entity: map[string]Models{"entity1": {Models: map[string]Model{}}},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	config, err := sm.GetVariantConfig("entity1", "nonexistent", "v1")
	assert.Error(t, err)
	assert.Nil(t, config)
	mockEtcd.AssertExpectations(t)
}

func TestGetAllFiltersForActiveVariants_WithActiveVariants(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	criteria := []Criteria{{ColumnName: "col1", FilterValue: "val1"}}
	variant := Variant{
		Enabled:   true,
		Onboarded: true,
		Filter:    map[string][]Criteria{"criteria": criteria},
	}
	model := Model{Variants: map[string]Variant{"v1": variant}}
	entityModels := Models{Models: map[string]Model{"m1": model}}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	filters, err := sm.GetAllFiltersForActiveVariants("entity1")
	assert.NoError(t, err)
	assert.NotNil(t, filters)
	assert.Contains(t, filters, "m1")
	assert.Contains(t, filters["m1"], "v1")
	assert.Equal(t, criteria, filters["m1"]["v1"])
	mockEtcd.AssertExpectations(t)
}

func TestGetAllFiltersForActiveVariants_NoActiveVariants(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	// Variant with Enabled but not Onboarded
	variant := Variant{Enabled: true, Onboarded: false}
	model := Model{Variants: map[string]Variant{"v1": variant}}
	entityModels := Models{Models: map[string]Model{"m1": model}}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	filters, err := sm.GetAllFiltersForActiveVariants("entity1")
	assert.NoError(t, err)
	assert.NotNil(t, filters)
	assert.Contains(t, filters, "m1")
	assert.Empty(t, filters["m1"])
	mockEtcd.AssertExpectations(t)
}

func TestSetVariantOnboarded(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/variants/v1/onboarded"
	mockEtcd.On("SetValue", path, true).Return(nil)

	err := sm.SetVariantOnboarded("e1", "m1", "v1", true)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVariantState(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/variants/v1/variant-state"
	state := enums.COMPLETED
	mockEtcd.On("SetValue", path, state).Return(nil)

	err := sm.UpdateVariantState("e1", "m1", "v1", state)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVariantReadVersion(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/variants/v1/vector-db-read-version"
	mockEtcd.On("SetValue", path, 3).Return(nil)

	err := sm.UpdateVariantReadVersion("e1", "m1", "v1", 3)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVariantWriteVersion(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/variants/v1/vector-db-write-version"
	mockEtcd.On("SetValue", path, 4).Return(nil)

	err := sm.UpdateVariantWriteVersion("e1", "m1", "v1", 4)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateEmbeddingVersion(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/embedding-store-version"
	mockEtcd.On("SetValue", path, 2).Return(nil)

	err := sm.UpdateEmbeddingVersion("e1", "m1", 2)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVariantEmbeddingStoreReadVersion(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/variants/v1/embedding-store-read-version"
	mockEtcd.On("SetValue", path, 5).Return(nil)

	err := sm.UpdateVariantEmbeddingStoreReadVersion("e1", "m1", "v1", 5)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVariantEmbeddingStoreWriteVersion(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/variants/v1/embedding-store-write-version"
	mockEtcd.On("SetValue", path, 6).Return(nil)

	err := sm.UpdateVariantEmbeddingStoreWriteVersion("e1", "m1", "v1", 6)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestRegisterStore(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{
		Storage: Storage{
			Stores: map[string]Data{
				"1": {ConfId: 1, Db: "db1", EmbeddingTable: "emb1", AggregatorTable: "agg1"},
			},
		},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	// storeId will be len(stores)+1 = 2
	path := "/config/test-app/storage/stores/2"
	data := Data{
		ConfId:          10,
		EmbeddingTable:  "emb_table",
		AggregatorTable: "agg_table",
		Db:              "test_db",
	}
	jsonData, _ := json.Marshal(data)
	mockEtcd.On("CreateNode", path, string(jsonData)).Return(nil)

	err := sm.RegisterStore(10, "test_db", "emb_table", "agg_table")
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestRegisterFrequency(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{
		Storage: Storage{Frequencies: "daily"},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	path := "/config/test-app/storage/frequencies"
	mockEtcd.On("SetValue", path, "daily,hourly").Return(nil)

	err := sm.RegisterFrequency("hourly")
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestRegisterFrequency_AlreadyRegistered(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	skye := &Skye{
		Storage: Storage{Frequencies: "daily,hourly"},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	err := sm.RegisterFrequency("hourly")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	mockEtcd.AssertNotCalled(t, "SetValue")
}

func TestRegisterEntity(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	paths := map[string]interface{}{
		"/config/test-app/entity/myentity/store-id": "store123",
	}
	mockEtcd.On("CreateNodes", paths).Return(nil)

	err := sm.RegisterEntity("myentity", "store123")
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVariantStates(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	variants := map[string]string{"v1": "", "v2": ""}
	basePath := "/config/test-app/entity/e1/models/m1/variant-state"

	mockEtcd.On("SetValue", basePath+"/v1", 1).Return(nil)
	mockEtcd.On("SetValue", basePath+"/v2", 1).Return(nil)

	err := sm.UpdateVariantStates("e1", "m1", variants, 1)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateVectorDbConfig(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	vectorDbConfig := VectorDbConfig{
		ReadHost:  "read.example.com",
		WriteHost: "write.example.com",
		Port:      "6333",
	}
	path := "/config/test-app/entity/e1/models/m1/variants/v1/vector-db-config"
	jsonData, _ := json.Marshal(vectorDbConfig)

	mockEtcd.On("SetValue", path, string(jsonData)).Return(nil)

	err := sm.UpdateVectorDbConfig("e1", "m1", "v1", vectorDbConfig)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdatePartitionState(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/partition-states/0"
	mockEtcd.On("SetValue", path, 2).Return(nil)

	err := sm.UpdatePartitionState("e1", "m1", "0", 2)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdatePartitionsForEof(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/entity/e1/models/m1/partition-eof-state/0"
	mockEtcd.On("SetValue", path, 1).Return(nil)

	err := sm.UpdatePartitionsForEof("e1", "m1", "0", 1)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestGetRateLimiters(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	rl := RateLimiter{BurstLimit: 100, RateLimit: 10}
	variant := Variant{RTPartition: 1, RateLimiter: rl}
	model := Model{Variants: map[string]Variant{"v1": variant}}
	entityModels := Models{Models: map[string]Model{"m1": model}}
	skye := &Skye{
		Entity: map[string]Models{"entity1": entityModels},
	}
	mockEtcd.On("GetConfigInstance").Return(skye)

	limiters := sm.GetRateLimiters()
	assert.NotNil(t, limiters)
	assert.Contains(t, limiters, 1)
	assert.Equal(t, rl, limiters[1])
	mockEtcd.AssertExpectations(t)
}

func TestRegisterWatchPathCallbackWithEvent(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	path := "/config/test-app/some/path"
	callback := func(key, value, eventType string) error { return nil }

	mockEtcd.On("RegisterWatchPathCallbackWithEvent", path, mock.AnythingOfType("func(string, string, string) error")).Return(nil)

	err := sm.RegisterWatchPathCallbackWithEvent(path, callback)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateRateLimiter(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	basePath := "/config/test-app/entity/e1/models/m1/variants/v1"
	mockEtcd.On("SetValue", basePath+"/rate-limiter/burst-limit", 50).Return(nil)
	mockEtcd.On("SetValue", basePath+"/rate-limiter/rate-limit", 5).Return(nil)

	err := sm.UpdateRateLimiter("e1", "m1", "v1", 50, 5)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}

func TestUpdateRateLimiter_SetValueError(t *testing.T) {
	mockEtcd := &etcd.MockEtcd{}
	sm := NewSkyeManager(mockEtcd, "test-app")

	basePath := "/config/test-app/entity/e1/models/m1/variants/v1"
	mockEtcd.On("SetValue", basePath+"/rate-limiter/burst-limit", 50).Return(assert.AnError)

	err := sm.UpdateRateLimiter("e1", "m1", "v1", 50, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "burst limit")
	mockEtcd.AssertExpectations(t)
}
