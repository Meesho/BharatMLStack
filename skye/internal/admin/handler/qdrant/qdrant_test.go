package qdrant

import (
	"fmt"
	"testing"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/workflow"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// helper to build a Qdrant with mocks
func newTestQdrant() (*Qdrant, *config.MockConfigManager, *workflow.MockStateMachine) {
	cm := new(config.MockConfigManager)
	sm := new(workflow.MockStateMachine)
	q := &Qdrant{configManager: cm, StateMachine: sm}
	return q, cm, sm
}

// --- validateVariantsCompleted ---

func TestValidateVariantsCompleted_AllCompleted(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{
			"v1": {VariantState: enums.COMPLETED},
			"v2": {VariantState: enums.COMPLETED},
		},
	}, nil)

	err := q.validateVariantsCompleted("e1", "m1")
	assert.NoError(t, err)
	cm.AssertExpectations(t)
}

func TestValidateVariantsCompleted_NotCompleted(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{
			"v1": {VariantState: enums.COMPLETED},
			"v2": {VariantState: enums.INDEXING_IN_PROGRESS},
		},
	}, nil)

	err := q.validateVariantsCompleted("e1", "m1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not completed")
}

// --- resetAllPartitions ---

func TestResetAllPartitions_Success(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		NumberOfPartitions: 3,
	}, nil)
	cm.On("UpdatePartitionState", "e1", "m1", "0", 0).Return(nil)
	cm.On("UpdatePartitionState", "e1", "m1", "1", 0).Return(nil)
	cm.On("UpdatePartitionState", "e1", "m1", "2", 0).Return(nil)

	err := q.resetAllPartitions("e1", "m1")
	assert.NoError(t, err)
	cm.AssertExpectations(t)
}

func TestResetAllPartitions_Error(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		NumberOfPartitions: 2,
	}, nil)
	cm.On("UpdatePartitionState", "e1", "m1", "0", 0).Return(fmt.Errorf("update failed"))

	err := q.resetAllPartitions("e1", "m1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// --- prepareEmbeddingStoreVersion ---

func TestPrepareEmbeddingStoreVersion_Delta(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		EmbeddingStoreVersion: 5,
		ModelType:             enums.DELTA,
	}, nil)

	version, err := q.prepareEmbeddingStoreVersion("e1", "m1")
	assert.NoError(t, err)
	assert.Equal(t, 5, version) // no increment for DELTA
}

func TestPrepareEmbeddingStoreVersion_Reset(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		EmbeddingStoreVersion: 5,
		ModelType:             enums.RESET,
	}, nil)
	cm.On("UpdateEmbeddingVersion", "e1", "m1", 6).Return(nil)

	version, err := q.prepareEmbeddingStoreVersion("e1", "m1")
	assert.NoError(t, err)
	assert.Equal(t, 6, version) // incremented for RESET
	cm.AssertExpectations(t)
}

func TestPrepareEmbeddingStoreVersion_ResetUpdateError(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		EmbeddingStoreVersion: 5,
		ModelType:             enums.RESET,
	}, nil)
	cm.On("UpdateEmbeddingVersion", "e1", "m1", 6).Return(fmt.Errorf("etcd error"))

	version, err := q.prepareEmbeddingStoreVersion("e1", "m1")
	assert.Error(t, err)
	assert.Equal(t, 0, version)
}

// --- buildProcessModelResponse ---

func TestBuildProcessModelResponse_WithVariants(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		TopicName:          "topic1",
		TrainingDataPath:   "/data/path",
		NumberOfPartitions: 4,
		ModelType:          enums.RESET,
	}, nil)

	variantsMap := map[string]int{"v1": 2, "v2": 3}
	resp, err := q.buildProcessModelResponse("e1", "m1", 5, variantsMap)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "topic1", resp.TopicName)
	assert.Equal(t, "/data/path", resp.TrainingDataPath)
	assert.Equal(t, 4, resp.NumberOfPartitions)
	assert.Equal(t, enums.RESET, resp.ModelType)
	assert.Equal(t, "m1", resp.Model)
	assert.Equal(t, 5, resp.EmbeddingStoreVersion)
	assert.Equal(t, variantsMap, resp.Variants)
}

func TestBuildProcessModelResponse_EmptyVariants(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{}, nil)

	resp, err := q.buildProcessModelResponse("e1", "m1", 5, map[string]int{})
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

// --- updateVariantAndJobState ---

func TestUpdateVariantAndJobState_Success(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.DATA_INGESTION_STARTED).Return(nil)

	err := q.updateVariantAndJobState("e1", "m1", "v1")
	assert.NoError(t, err)
	cm.AssertExpectations(t)
}

func TestUpdateVariantAndJobState_Error(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.DATA_INGESTION_STARTED).Return(fmt.Errorf("state update error"))

	err := q.updateVariantAndJobState("e1", "m1", "v1")
	assert.Error(t, err)
}

// --- ProcessModelsWithFrequency ---

func TestProcessModelsWithFrequency_MatchingModels(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetEntityConfig", "e1").Return(&config.Models{
		StoreId: "store1",
		Models: map[string]config.Model{
			"m1": {JobFrequency: "daily"},
			"m2": {JobFrequency: "weekly"},
			"m3": {JobFrequency: "daily"},
		},
	}, nil)

	resp, err := q.ProcessModelsWithFrequency(&ProcessModelsWithFrequencyRequest{
		Entity:    "e1",
		Frequency: "daily",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.ProcessModelResponse, 2)
	models := make(map[string]bool)
	for _, r := range resp.ProcessModelResponse {
		models[r.Model] = true
	}
	assert.True(t, models["m1"])
	assert.True(t, models["m3"])
}

func TestProcessModelsWithFrequency_NoMatch(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetEntityConfig", "e1").Return(&config.Models{
		Models: map[string]config.Model{
			"m1": {JobFrequency: "daily"},
		},
	}, nil)

	resp, err := q.ProcessModelsWithFrequency(&ProcessModelsWithFrequencyRequest{
		Entity:    "e1",
		Frequency: "weekly",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, resp.ProcessModelResponse)
}

// --- ProcessModel (validation failure paths) ---

func TestProcessModel_VariantsNotCompleted(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{
			"v1": {VariantState: enums.INDEXING_IN_PROGRESS},
		},
	}, nil)

	resp, err := q.ProcessModel(&ProcessModelRequest{Entity: "e1", Model: "m1", VectorDbType: enums.QDRANT})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not completed")
}

func TestProcessModel_ResetPartitionsError(t *testing.T) {
	q, cm, _ := newTestQdrant()
	// validateVariantsCompleted passes
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants:           map[string]config.Variant{"v1": {VariantState: enums.COMPLETED}},
		NumberOfPartitions: 1,
	}, nil)
	// resetAllPartitions fails
	cm.On("UpdatePartitionState", "e1", "m1", "0", 0).Return(fmt.Errorf("partition fail"))

	resp, err := q.ProcessModel(&ProcessModelRequest{Entity: "e1", Model: "m1", VectorDbType: enums.QDRANT})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProcessModel_NoEnabledVariantsForDbType(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{
			"v1": {VariantState: enums.COMPLETED, Enabled: true, VectorDbType: enums.NGT}, // different DB type
		},
		NumberOfPartitions:    1,
		EmbeddingStoreVersion: 1,
		ModelType:             enums.DELTA,
	}, nil)
	cm.On("UpdatePartitionState", "e1", "m1", "0", 0).Return(nil)

	resp, err := q.ProcessModel(&ProcessModelRequest{Entity: "e1", Model: "m1", VectorDbType: enums.QDRANT})
	// No matching variants → nil response
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

// --- ProcessMultiVariant (validation failure paths) ---

func TestProcessMultiVariant_VariantNotPresent(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{},
	}, nil)

	resp, err := q.ProcessMultiVariant(&ProcessMultiVariantRequest{
		Entity: "e1", Model: "m1", Variants: []string{"v_missing"},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not present")
}

func TestProcessMultiVariant_VariantNotCompleted(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{
			"v1": {Enabled: true, VariantState: enums.INDEXING_IN_PROGRESS},
		},
	}, nil)
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		Enabled:      true,
		VariantState: enums.INDEXING_IN_PROGRESS,
	}, nil)

	resp, err := q.ProcessMultiVariant(&ProcessMultiVariantRequest{
		Entity: "e1", Model: "m1", Variants: []string{"v1"},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not completed")
}

func TestProcessMultiVariant_DisabledVariantSkipped(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		Variants: map[string]config.Variant{
			"v1": {Enabled: false, VariantState: enums.COMPLETED},
		},
		NumberOfPartitions:    1,
		EmbeddingStoreVersion: 2,
		ModelType:             enums.DELTA,
	}, nil)
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		Enabled: false,
	}, nil)
	cm.On("UpdatePartitionState", "e1", "m1", "0", 0).Return(nil)

	resp, err := q.ProcessMultiVariant(&ProcessMultiVariantRequest{
		Entity: "e1", Model: "m1", Variants: []string{"v1"},
	})
	assert.NoError(t, err)
	assert.Nil(t, resp) // no variants processed → nil
}

// --- PromoteVariant (validation failure paths) ---

func TestPromoteVariant_NotEnabled(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{Enabled: false}, nil)

	resp, err := q.PromoteVariant(&PromoteVariantRequest{Entity: "e1", Model: "m1", Variant: "v1"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestPromoteVariant_NotCompleted(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		Enabled:      true,
		VariantState: enums.INDEXING_IN_PROGRESS,
	}, nil)

	resp, err := q.PromoteVariant(&PromoteVariantRequest{Entity: "e1", Model: "m1", Variant: "v1"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not completed")
}

func TestPromoteVariant_NotExperiment(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		Enabled:      true,
		VariantState: enums.COMPLETED,
		Type:         enums.SCALE_UP,
	}, nil)

	resp, err := q.PromoteVariant(&PromoteVariantRequest{Entity: "e1", Model: "m1", Variant: "v1"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not in experiment state")
}

// --- ProcessMultiVariantForceReset (validation failure paths) ---

func TestProcessMultiVariantForceReset_ResetModelType(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		ModelType: enums.RESET,
	}, nil)

	resp, err := q.ProcessMultiVariantForceReset(&ProcessMultiVariantRequest{
		Entity: "e1", Model: "m1", Variants: []string{"v1"},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Force reset is not supported for reset model type")
}

func TestProcessMultiVariantForceReset_VariantNotPresent(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		ModelType: enums.DELTA,
		Variants:  map[string]config.Variant{},
	}, nil)

	resp, err := q.ProcessMultiVariantForceReset(&ProcessMultiVariantRequest{
		Entity: "e1", Model: "m1", Variants: []string{"v_missing"},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not present")
}

func TestProcessMultiVariantForceReset_VariantNotCompleted(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		ModelType: enums.DELTA,
		Variants: map[string]config.Variant{
			"v1": {Enabled: true, VariantState: enums.INDEXING_STARTED},
		},
	}, nil)
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		Enabled:      true,
		VariantState: enums.INDEXING_STARTED,
	}, nil)

	resp, err := q.ProcessMultiVariantForceReset(&ProcessMultiVariantRequest{
		Entity: "e1", Model: "m1", Variants: []string{"v1"},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not completed")
}

// --- revertConfig ---

func TestRevertConfig_NilOriginal(t *testing.T) {
	q, _, _ := newTestQdrant()
	// Should not panic
	q.revertConfig(nil, "e1", "m1", "v1")
}

func TestRevertConfig_NotOnboarded(t *testing.T) {
	q, cm, _ := newTestQdrant()
	original := &config.Variant{
		EmbeddingStoreWriteVersion: 3,
		VectorDbWriteVersion:       2,
		VariantState:               enums.COMPLETED,
		RateLimiter:                config.RateLimiter{BurstLimit: 10, RateLimit: 5},
		Onboarded:                  false, // won't call vector.GetRepository
	}
	cm.On("UpdateVariantEmbeddingStoreWriteVersion", "e1", "m1", "v1", 3).Return(nil)
	cm.On("UpdateVariantWriteVersion", "e1", "m1", "v1", 2).Return(nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.COMPLETED).Return(nil)
	cm.On("UpdateRateLimiter", "e1", "m1", "v1", 10, 5).Return(nil)

	q.revertConfig(original, "e1", "m1", "v1")
	cm.AssertExpectations(t)
}

func TestRevertConfig_WithErrors(t *testing.T) {
	q, cm, _ := newTestQdrant()
	original := &config.Variant{
		EmbeddingStoreWriteVersion: 1,
		VectorDbWriteVersion:       1,
		VariantState:               enums.COMPLETED,
		RateLimiter:                config.RateLimiter{},
		Onboarded:                  false,
	}
	cm.On("UpdateVariantEmbeddingStoreWriteVersion", "e1", "m1", "v1", 1).Return(fmt.Errorf("err1"))
	cm.On("UpdateVariantWriteVersion", "e1", "m1", "v1", 1).Return(fmt.Errorf("err2"))
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.COMPLETED).Return(fmt.Errorf("err3"))
	cm.On("UpdateRateLimiter", "e1", "m1", "v1", 0, 0).Return(fmt.Errorf("err4"))

	// Should not panic even with all errors
	q.revertConfig(original, "e1", "m1", "v1")
	cm.AssertExpectations(t)
}

func TestRevertConfig_Onboarded_Success(t *testing.T) {
	q, cm, _ := newTestQdrant()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	original := &config.Variant{
		EmbeddingStoreWriteVersion: 3,
		VectorDbWriteVersion:       2,
		VariantState:               enums.COMPLETED,
		RateLimiter:                config.RateLimiter{BurstLimit: 10, RateLimit: 5},
		Onboarded:                  true,
	}
	cm.On("UpdateVariantEmbeddingStoreWriteVersion", "e1", "m1", "v1", 3).Return(nil)
	cm.On("UpdateVariantWriteVersion", "e1", "m1", "v1", 2).Return(nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.COMPLETED).Return(nil)
	cm.On("UpdateRateLimiter", "e1", "m1", "v1", 10, 5).Return(nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 2, "100").Return(nil)

	q.revertConfig(original, "e1", "m1", "v1")
	cm.AssertExpectations(t)
	mockVectorDb.AssertExpectations(t)
}

func TestRevertConfig_Onboarded_ThresholdError(t *testing.T) {
	q, cm, _ := newTestQdrant()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	original := &config.Variant{
		EmbeddingStoreWriteVersion: 1,
		VectorDbWriteVersion:       1,
		VariantState:               enums.COMPLETED,
		RateLimiter:                config.RateLimiter{},
		Onboarded:                  true,
	}
	cm.On("UpdateVariantEmbeddingStoreWriteVersion", "e1", "m1", "v1", 1).Return(nil)
	cm.On("UpdateVariantWriteVersion", "e1", "m1", "v1", 1).Return(nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.COMPLETED).Return(nil)
	cm.On("UpdateRateLimiter", "e1", "m1", "v1", 0, 0).Return(nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 1, "100").Return(fmt.Errorf("threshold error"))

	// Should not panic
	q.revertConfig(original, "e1", "m1", "v1")
	cm.AssertExpectations(t)
	mockVectorDb.AssertExpectations(t)
}

// --- TriggerIndexing (partial coverage - partition state update + partition check logic) ---

func TestTriggerIndexing_UpdatePartitionStateError(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("UpdatePartitionState", "e1", "m1", "0", 1).Return(fmt.Errorf("partition error"))

	err := q.TriggerIndexing(&TriggerIndexingRequest{
		Entity:    "e1",
		Model:     "m1",
		Partition: "0",
	})
	assert.Error(t, err)
}

func TestTriggerIndexing_InvalidPartition(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("UpdatePartitionState", "e1", "m1", "abc", 1).Return(nil)

	err := q.TriggerIndexing(&TriggerIndexingRequest{
		Entity:    "e1",
		Model:     "m1",
		Partition: "abc",
	})
	assert.Error(t, err) // strconv.Atoi fails on "abc"
}

// --- processVariant (config error path) ---

func TestProcessVariant_GetVariantConfigError(t *testing.T) {
	q, cm, _ := newTestQdrant()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return((*config.Variant)(nil), fmt.Errorf("config not found"))

	variantsMap := make(map[string]int)
	err := q.processVariant("v1", "daily", "e1", "m1", "", 1, enums.DELTA, variantsMap, false, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not found")
}

func TestProcessVariant_HostMismatch(t *testing.T) {
	q, cm, _ := newTestQdrant()
	// First call: originalConfig for defer
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbConfig: config.VectorDbConfig{
			ReadHost:  "host-read",
			WriteHost: "host-write", // mismatch
			Params:    map[string]string{},
		},
		VectorDbReadVersion:        1,
		EmbeddingStoreWriteVersion: 1,
		VectorDbWriteVersion:       1,
		VariantState:               enums.COMPLETED,
		Onboarded:                  false,
	}, nil)
	cm.On("UpdateVariantEmbeddingStoreWriteVersion", "e1", "m1", "v1", mock.Anything).Return(nil)

	variantsMap := make(map[string]int)
	err := q.processVariant("v1", "", "e1", "m1", "", 1, enums.DELTA, variantsMap, false, false)
	// Host mismatch → returns nil (logged as error but not returned as error)
	assert.NoError(t, err)
	assert.Empty(t, variantsMap) // variant not added because no updateVariantAndJobState was called
}

// ============================================================
// PublishCollectionMetrics
// ============================================================

func TestPublishCollectionMetrics_Disabled(t *testing.T) {
	q, _, _ := newTestQdrant()
	// appConfig.CollectionMetricEnabled defaults to false
	origConfig := appConfig
	defer func() { appConfig = origConfig }()
	appConfig.CollectionMetricEnabled = false

	err := q.PublishCollectionMetrics()
	assert.NoError(t, err)
}
