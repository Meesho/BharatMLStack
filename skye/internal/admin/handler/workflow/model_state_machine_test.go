package workflow

import (
	"fmt"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Override sleepFunc to no-op so tests don't block on time.Sleep calls.
	sleepFunc = func(d time.Duration) {}
}

func newTestMSM() (*ModelStateMachine, *config.MockConfigManager) {
	cm := new(config.MockConfigManager)
	return &ModelStateMachine{configManager: cm}, cm
}

// ============================================================
// ProcessStates
// ============================================================

func TestProcessStates_EmptyVariantState(t *testing.T) {
	msm, _ := newTestMSM()
	payload := &ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: "",
	}
	err := msm.ProcessStates(payload)
	assert.NoError(t, err)
}

func TestProcessStates_ProcessStateError(t *testing.T) {
	msm, cm := newTestMSM()
	payload := &ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.DATA_INGESTION_STARTED,
	}
	// GetVariantConfig called inside ProcessState
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return((*config.Variant)(nil), fmt.Errorf("config err"))

	err := msm.ProcessStates(payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config err")
}

// ============================================================
// ProcessState — dispatch to correct handler per VariantState
// ============================================================

func TestProcessState_GetVariantConfigError(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return((*config.Variant)(nil), fmt.Errorf("no config"))

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.DATA_INGESTION_STARTED,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
	assert.Equal(t, 0, counter)
}

func TestProcessState_DefaultState(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.VariantState("UNKNOWN_STATE"),
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.VariantState(""), state)
	assert.Equal(t, 0, counter)
}

// ============================================================
// handleDataIngestionStarted
// ============================================================

func TestHandleDataIngestionStarted_Success(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.DATA_INGESTION_COMPLETED).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.DATA_INGESTION_STARTED,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.DATA_INGESTION_COMPLETED, state)
	assert.Equal(t, 0, counter)
	cm.AssertExpectations(t)
}

func TestHandleDataIngestionStarted_UpdateError(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.DATA_INGESTION_COMPLETED).Return(fmt.Errorf("update err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.DATA_INGESTION_STARTED,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

// ============================================================
// handleDataIngestionCompleted — calls vector.GetRepository
// ============================================================

func TestHandleDataIngestionCompleted_UpdateThresholdError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"indexing_threshold": "20000"},
		},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 5, "20000").Return(fmt.Errorf("threshold err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.DATA_INGESTION_COMPLETED,
		Version:      5,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "threshold err")
	assert.Equal(t, enums.VariantState(""), state)
}

// ============================================================
// handleIndexingStarted — calls vector.GetRepository
// ============================================================

func TestHandleIndexingStarted_NotYetIndexed(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
	}, nil)
	// ratio 50/100 = 0.5 < 0.95 → not indexed
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount:         100,
		IndexedVectorsCount: 50,
	}, nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      0,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_STARTED, state)
	assert.Equal(t, 0, counter) // no increment
}

func TestHandleIndexingStarted_IndexedOnce(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
	}, nil)
	// ratio 98/100 = 0.98 > 0.95 → indexed, counter 0→1 (not 2 yet)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount:         100,
		IndexedVectorsCount: 98,
	}, nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      0,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_STARTED, state)
	assert.Equal(t, 1, counter) // incremented to 1
}

func TestHandleIndexingStarted_Counter2_NoPayloadIndex(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{}, // no after_collection_index_payload
		},
	}, nil)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount:         100,
		IndexedVectorsCount: 98,
	}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_IN_PROGRESS).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      1, // +1 = 2 → transitions
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_IN_PROGRESS, state)
	assert.Equal(t, 0, counter)
	cm.AssertExpectations(t)
}

func TestHandleIndexingStarted_Counter2_WithPayloadIndex_Success(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"after_collection_index_payload": "true"},
		},
	}, nil)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount:         100,
		IndexedVectorsCount: 98,
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "0").Return(nil)
	mockVectorDb.On("CreateFieldIndexes", "e1", "m1", "v1", 3).Return(nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_IN_PROGRESS).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      1,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_IN_PROGRESS, state)
	assert.Equal(t, 0, counter)
}

func TestHandleIndexingStarted_Counter2_CreateFieldIndexesError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"after_collection_index_payload": "true"},
		},
	}, nil)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount: 100, IndexedVectorsCount: 98,
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "0").Return(nil)
	mockVectorDb.On("CreateFieldIndexes", "e1", "m1", "v1", 3).Return(fmt.Errorf("field index err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field index err")
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleIndexingStarted_Counter2_UpdateStateError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType:   enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{Params: map[string]string{}},
	}, nil)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount: 100, IndexedVectorsCount: 98,
	}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_IN_PROGRESS).Return(fmt.Errorf("state err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      1,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleIndexingStarted_NilCollectionInfo(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
	}, nil)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return((*vector.CollectionInfoResponse)(nil), nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_STARTED,
		Version:      3,
		Counter:      0,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_STARTED, state)
	assert.Equal(t, 0, counter)
}

// ============================================================
// handleIndexingInProgress — calls vector.GetRepository
// ============================================================

func TestHandleIndexingInProgress_ThresholdError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"default_indexing_threshold": "30000"},
		},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "30000").Return(fmt.Errorf("threshold err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_IN_PROGRESS,
		Version:      3,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleIndexingInProgress_DeltaModel(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"default_indexing_threshold": "30000"},
		},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "30000").Return(nil)
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{ModelType: enums.DELTA}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_IN_PROGRESS,
		Version:      3,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_COMPLETED, state)
	assert.Equal(t, 0, counter)
}

func TestHandleIndexingInProgress_ResetModel(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"default_indexing_threshold": "30000"},
		},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "30000").Return(nil)
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{ModelType: enums.RESET}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED_WITH_RESET).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_IN_PROGRESS,
		Version:      3,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_COMPLETED_WITH_RESET, state)
	assert.Equal(t, 0, counter)
}

func TestHandleIndexingInProgress_UpdateStateError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType:   enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{Params: map[string]string{"default_indexing_threshold": "30000"}},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "30000").Return(nil)
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{ModelType: enums.DELTA}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(fmt.Errorf("state err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_IN_PROGRESS,
		Version:      3,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleIndexingInProgress_WithPayloadIndex_NotReady(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{
				"default_indexing_threshold":     "30000",
				"after_collection_index_payload": "true",
			},
		},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "30000").Return(nil)
	// payload points ratio < 0.90 → not ready, counter stays 0
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount:        100,
		PayloadPointsCount: []float64{50}, // 50/100 = 0.5 < 0.90
	}, nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_IN_PROGRESS,
		Version:      3,
		Counter:      0,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_IN_PROGRESS, state)
	assert.Equal(t, 0, counter)
}

func TestHandleIndexingInProgress_WithPayloadIndex_Ready_Counter5(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{
				"default_indexing_threshold":     "30000",
				"after_collection_index_payload": "true",
			},
		},
	}, nil)
	mockVectorDb.On("UpdateIndexingThreshold", "e1", "m1", "v1", 3, "30000").Return(nil)
	mockVectorDb.On("GetCollectionInfo", "e1", "m1", "v1", 3).Return(&vector.CollectionInfoResponse{
		PointsCount:        100,
		PayloadPointsCount: []float64{96}, // 96/100 > 0.95, counter 4→5
	}, nil)
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{ModelType: enums.DELTA}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_IN_PROGRESS,
		Version:      3,
		Counter:      4, // +1 = 5 → transitions
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_COMPLETED, state)
	assert.Equal(t, 0, counter)
}

// ============================================================
// handleIndexingCompletedWithReset
// ============================================================

func TestHandleIndexingCompletedWithReset_Success(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantReadVersion", "e1", "m1", "v1", 5).Return(nil)
	cm.On("UpdateVariantEmbeddingStoreReadVersion", "e1", "m1", "v1", 10).Return(nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.MODEL_VERSION_UPDATED).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState:          enums.INDEXING_COMPLETED_WITH_RESET,
		Version:               5,
		EmbeddingStoreVersion: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.MODEL_VERSION_UPDATED, state)
	assert.Equal(t, 0, counter)
	cm.AssertExpectations(t)
}

func TestHandleIndexingCompletedWithReset_ReadVersionError(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantReadVersion", "e1", "m1", "v1", 5).Return(fmt.Errorf("read ver err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_COMPLETED_WITH_RESET,
		Version:      5,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleIndexingCompletedWithReset_EmbeddingStoreVersionError(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantReadVersion", "e1", "m1", "v1", 5).Return(nil)
	cm.On("UpdateVariantEmbeddingStoreReadVersion", "e1", "m1", "v1", 10).Return(fmt.Errorf("emb ver err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState:          enums.INDEXING_COMPLETED_WITH_RESET,
		Version:               5,
		EmbeddingStoreVersion: 10,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleIndexingCompletedWithReset_UpdateStateError(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantReadVersion", "e1", "m1", "v1", 5).Return(nil)
	cm.On("UpdateVariantEmbeddingStoreReadVersion", "e1", "m1", "v1", 10).Return(nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.MODEL_VERSION_UPDATED).Return(fmt.Errorf("state err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState:          enums.INDEXING_COMPLETED_WITH_RESET,
		Version:               5,
		EmbeddingStoreVersion: 10,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

// ============================================================
// handleModelVersionUpdated — calls vector.GetRepository
// ============================================================

func TestHandleModelVersionUpdated_Success(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType:   enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{Params: map[string]string{}},
	}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(nil)
	mockVectorDb.On("DeleteCollection", "e1", "m1", "v1", 4).Return(nil) // version-1

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.MODEL_VERSION_UPDATED,
		Version:      5,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_COMPLETED, state)
	assert.Equal(t, 0, counter)
}

func TestHandleModelVersionUpdated_UpdateStateError(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{VectorDbType: enums.QDRANT}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(fmt.Errorf("state err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.MODEL_VERSION_UPDATED,
		Version:      5,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleModelVersionUpdated_DeleteCollectionError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{VectorDbType: enums.QDRANT}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(nil)
	mockVectorDb.On("DeleteCollection", "e1", "m1", "v1", 4).Return(fmt.Errorf("delete err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.MODEL_VERSION_UPDATED,
		Version:      5,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete err")
	assert.Equal(t, enums.VariantState(""), state)
}

func TestHandleModelVersionUpdated_WithRateLimiterUpdate(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"after_collection_index_payload": "true"},
		},
	}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(nil)
	mockVectorDb.On("DeleteCollection", "e1", "m1", "v1", 4).Return(nil)
	cm.On("UpdateRateLimiter", "e1", "m1", "v1", 0, 0).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.MODEL_VERSION_UPDATED,
		Version:      5,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.INDEXING_COMPLETED, state)
	assert.Equal(t, 0, counter)
	cm.AssertExpectations(t)
}

func TestHandleModelVersionUpdated_RateLimiterError(t *testing.T) {
	msm, cm := newTestMSM()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{
		VectorDbType: enums.QDRANT,
		VectorDbConfig: config.VectorDbConfig{
			Params: map[string]string{"after_collection_index_payload": "true"},
		},
	}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.INDEXING_COMPLETED).Return(nil)
	mockVectorDb.On("DeleteCollection", "e1", "m1", "v1", 4).Return(nil)
	cm.On("UpdateRateLimiter", "e1", "m1", "v1", 0, 0).Return(fmt.Errorf("limiter err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.MODEL_VERSION_UPDATED,
		Version:      5,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limiter err")
	assert.Equal(t, enums.VariantState(""), state)
}

// ============================================================
// handleIndexingCompleted
// ============================================================

func TestHandleIndexingCompleted_Success(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.COMPLETED).Return(nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_COMPLETED,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.COMPLETED, state)
	assert.Equal(t, 0, counter)
}

func TestHandleIndexingCompleted_Error(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)
	cm.On("UpdateVariantState", "e1", "m1", "v1", enums.COMPLETED).Return(fmt.Errorf("update err"))

	state, _, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.INDEXING_COMPLETED,
	})
	assert.Error(t, err)
	assert.Equal(t, enums.VariantState(""), state)
}

// ============================================================
// handleCompleted
// ============================================================

func TestHandleCompleted(t *testing.T) {
	msm, cm := newTestMSM()
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return(&config.Variant{}, nil)

	state, counter, err := msm.ProcessState(&ModelStateExecutorPayload{
		Entity: "e1", Model: "m1", Variant: "v1",
		VariantState: enums.COMPLETED,
	})
	assert.NoError(t, err)
	assert.Equal(t, enums.VariantState(""), state)
	assert.Equal(t, 0, counter)
}
