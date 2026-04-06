package vector

import (
	"fmt"
	"testing"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
)

// ============================================================
// GetClientKey
// ============================================================

func TestGetClientKey(t *testing.T) {
	assert.Equal(t, "e:m:v", GetClientKey("e", "m", "v"))
	assert.Equal(t, "entity:model:variant", GetClientKey("entity", "model", "variant"))
	assert.Equal(t, "::", GetClientKey("", "", ""))
}

// ============================================================
// getMetricTags
// ============================================================

func TestGetMetricTags(t *testing.T) {
	tags := getMetricTags("e1", "m1", "v1", "1")
	expected := []string{"vector_db_type", "qdrant", "entity_name", "e1", "model_name", "m1", "variant_name", "v1", "variant_version", "1"}
	assert.Equal(t, expected, tags)
}

// ============================================================
// getQdrantClient
// ============================================================

func TestGetQdrantClient(t *testing.T) {
	client := &QdrantClient{ReadHost: "host1"}
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{
			"e1:m1:v1": client,
		},
	}
	assert.Equal(t, client, q.getQdrantClient("e1", "m1", "v1"))
	assert.Nil(t, q.getQdrantClient("e1", "m1", "missing"))
}

// ============================================================
// extractQdrantKey
// ============================================================

func TestExtractQdrantKey(t *testing.T) {
	q := &Qdrant{}

	tests := []struct {
		name                                           string
		key                                            string
		expectedEntity, expectedModel, expectedVariant string
	}{
		{
			name:            "vector-db-config key",
			key:             "/some/root/skye/entity1/models/model1/variants/variant1/vector-db-config",
			expectedEntity:  "entity1",
			expectedModel:   "model1",
			expectedVariant: "variant1",
		},
		{
			name:            "enabled key",
			key:             "/some/root/skye/entity1/models/model1/variants/variant1/enabled",
			expectedEntity:  "entity1",
			expectedModel:   "model1",
			expectedVariant: "variant1",
		},
		{
			name:            "backup-config key",
			key:             "/some/root/skye/entity1/models/model1/variants/variant1/backup-config",
			expectedEntity:  "entity1",
			expectedModel:   "model1",
			expectedVariant: "variant1",
		},
		{
			name:            "unknown key",
			key:             "/some/root/skye/entity1/models/model1/other",
			expectedEntity:  "",
			expectedModel:   "",
			expectedVariant: "",
		},
		{
			name:            "empty key",
			key:             "",
			expectedEntity:  "",
			expectedModel:   "",
			expectedVariant: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, m, v := q.extractQdrantKey(tt.key)
			assert.Equal(t, tt.expectedEntity, e)
			assert.Equal(t, tt.expectedModel, m)
			assert.Equal(t, tt.expectedVariant, v)
		})
	}
}

// ============================================================
// prepareUpsertPoints
// ============================================================

func TestPrepareUpsertPoints(t *testing.T) {
	q := &Qdrant{}

	data := []Data{
		{
			Id:      "123",
			Payload: map[string]interface{}{"color": "red", "size": 42},
			Vectors: []float32{0.1, 0.2, 0.3},
		},
		{
			Id:      "456",
			Payload: map[string]interface{}{"active": true},
			Vectors: []float32{0.4, 0.5},
		},
	}

	points, err := q.prepareUpsertPoints(data)
	assert.NoError(t, err)
	assert.Len(t, points, 2)

	// Verify first point
	assert.Equal(t, uint64(123), points[0].Id.GetNum())
	assert.NotNil(t, points[0].Payload["color"])
	assert.Equal(t, "red", points[0].Payload["color"].GetStringValue())
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, points[0].Vectors.GetVector().Data)

	// Verify second point
	assert.Equal(t, uint64(456), points[1].Id.GetNum())
	assert.True(t, points[1].Payload["active"].GetBoolValue())
}

func TestPrepareUpsertPoints_EmptyData(t *testing.T) {
	q := &Qdrant{}
	points, err := q.prepareUpsertPoints([]Data{})
	assert.NoError(t, err)
	assert.Nil(t, points)
}

func TestPrepareUpsertPoints_EmptyPayload(t *testing.T) {
	q := &Qdrant{}
	data := []Data{{Id: "1", Payload: map[string]interface{}{}, Vectors: []float32{1.0}}}
	points, err := q.prepareUpsertPoints(data)
	assert.NoError(t, err)
	assert.Len(t, points, 1)
	assert.Empty(t, points[0].Payload)
}

// ============================================================
// prepareDeletePoints
// ============================================================

func TestPrepareDeletePoints(t *testing.T) {
	q := &Qdrant{}
	data := []Data{{Id: "100"}, {Id: "200"}, {Id: "300"}}
	points, err := q.prepareDeletePoints(data)
	assert.NoError(t, err)
	assert.Len(t, points, 3)
	assert.Equal(t, uint64(100), points[0].GetNum())
	assert.Equal(t, uint64(200), points[1].GetNum())
	assert.Equal(t, uint64(300), points[2].GetNum())
}

func TestPrepareDeletePoints_Empty(t *testing.T) {
	q := &Qdrant{}
	points, err := q.prepareDeletePoints([]Data{})
	assert.NoError(t, err)
	assert.Nil(t, points)
}

// ============================================================
// prepareUpsertPayload
// ============================================================

func TestPrepareUpsertPayload(t *testing.T) {
	q := &Qdrant{}
	d := Data{
		Id:      "999",
		Payload: map[string]interface{}{"tag": "premium", "count": 5},
	}
	payload, selector, err := q.prepareUpsertPayload(d)
	assert.NoError(t, err)
	assert.NotNil(t, payload)
	assert.NotNil(t, selector)
	assert.Equal(t, "premium", payload["tag"].GetStringValue())
	ids := selector.GetPoints().Ids
	assert.Len(t, ids, 1)
	assert.Equal(t, uint64(999), ids[0].GetNum())
}

func TestPrepareUpsertPayload_EmptyPayload(t *testing.T) {
	q := &Qdrant{}
	d := Data{Id: "1", Payload: map[string]interface{}{}}
	payload, selector, err := q.prepareUpsertPayload(d)
	assert.NoError(t, err)
	assert.Empty(t, payload)
	assert.NotNil(t, selector)
}

// ============================================================
// getFieldIndexParams
// ============================================================

func TestGetFieldIndexParams_Keyword(t *testing.T) {
	q := &Qdrant{}
	params := q.getFieldIndexParams(qdrant.FieldType_FieldTypeKeyword, config.Payload{})
	assert.NotNil(t, params)
	assert.NotNil(t, params.GetKeywordIndexParams())
}

func TestGetFieldIndexParams_Integer(t *testing.T) {
	q := &Qdrant{}
	params := q.getFieldIndexParams(qdrant.FieldType_FieldTypeInteger, config.Payload{
		LookupEnabled: true,
		IsPrincipal:   false,
	})
	assert.NotNil(t, params)
	intParams := params.GetIntegerIndexParams()
	assert.NotNil(t, intParams)
	assert.True(t, *intParams.Lookup)
	assert.False(t, *intParams.IsPrincipal)
}

func TestGetFieldIndexParams_Bool(t *testing.T) {
	q := &Qdrant{}
	params := q.getFieldIndexParams(qdrant.FieldType_FieldTypeBool, config.Payload{})
	assert.NotNil(t, params)
	assert.NotNil(t, params.GetBoolIndexParams())
}

func TestGetFieldIndexParams_Default(t *testing.T) {
	q := &Qdrant{}
	params := q.getFieldIndexParams(qdrant.FieldType_FieldTypeFloat, config.Payload{})
	assert.Nil(t, params)
}

// ============================================================
// mapCollectionInfoResponse
// ============================================================

func TestMapCollectionInfoResponse_Full(t *testing.T) {
	q := &Qdrant{}
	indexedCount := uint64(500)
	pointsCount := uint64(1000)
	payloadPoints := uint64(800)

	resp := &qdrant.GetCollectionInfoResponse{
		Result: &qdrant.CollectionInfo{
			Status:              qdrant.CollectionStatus_Green,
			IndexedVectorsCount: &indexedCount,
			PointsCount:         &pointsCount,
			PayloadSchema: map[string]*qdrant.PayloadSchemaInfo{
				"field1": {Points: &payloadPoints},
			},
		},
	}
	payloadSchema := map[string]config.Payload{
		"field1": {},
	}

	result := q.mapCollectionInfoResponse(resp, payloadSchema)
	assert.Equal(t, "GREEN", result.Status)
	assert.Equal(t, float64(500), result.IndexedVectorsCount)
	assert.Equal(t, float64(1000), result.PointsCount)
	assert.Len(t, result.PayloadPointsCount, 1)
	assert.Equal(t, float64(800), result.PayloadPointsCount[0])
}

func TestMapCollectionInfoResponse_NilCounts(t *testing.T) {
	q := &Qdrant{}
	resp := &qdrant.GetCollectionInfoResponse{
		Result: &qdrant.CollectionInfo{
			Status:              qdrant.CollectionStatus_Yellow,
			IndexedVectorsCount: nil,
			PointsCount:         nil,
			PayloadSchema:       nil,
		},
	}

	result := q.mapCollectionInfoResponse(resp, nil)
	assert.Equal(t, "YELLOW", result.Status)
	assert.Equal(t, float64(0), result.IndexedVectorsCount)
	assert.Equal(t, float64(0), result.PointsCount)
	assert.Nil(t, result.PayloadPointsCount)
}

func TestMapCollectionInfoResponse_RedStatus(t *testing.T) {
	q := &Qdrant{}
	resp := &qdrant.GetCollectionInfoResponse{
		Result: &qdrant.CollectionInfo{Status: qdrant.CollectionStatus_Red},
	}
	result := q.mapCollectionInfoResponse(resp, nil)
	assert.Equal(t, "RED", result.Status)
}

func TestMapCollectionInfoResponse_PayloadSchemaKeyNotFound(t *testing.T) {
	q := &Qdrant{}
	payloadPoints := uint64(100)
	resp := &qdrant.GetCollectionInfoResponse{
		Result: &qdrant.CollectionInfo{
			Status: qdrant.CollectionStatus_Green,
			PayloadSchema: map[string]*qdrant.PayloadSchemaInfo{
				"field1": {Points: &payloadPoints},
			},
		},
	}
	// Request field2 which doesn't exist in response schema
	payloadSchema := map[string]config.Payload{
		"field2": {},
	}
	result := q.mapCollectionInfoResponse(resp, payloadSchema)
	assert.Nil(t, result.PayloadPointsCount)
}

// ============================================================
// BulkUpsert — key parsing errors
// ============================================================

func TestBulkUpsert_InvalidKeyFormat(t *testing.T) {
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: new(config.MockConfigManager),
		AppConfig:     &structs.AppConfig{},
	}
	req := UpsertRequest{
		Data: map[string][]Data{
			"invalid-key": {{Id: "1", Vectors: []float32{1.0}}},
		},
	}
	err := q.BulkUpsert(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key format")
}

func TestBulkUpsert_EmptyData(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkUpsert(UpsertRequest{Data: map[string][]Data{}})
	assert.NoError(t, err)
}

func TestBulkUpsert_TooFewParts(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkUpsert(UpsertRequest{Data: map[string][]Data{"a|b": {}}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key format")
}

func TestBulkUpsert_TooManyParts(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkUpsert(UpsertRequest{Data: map[string][]Data{"a|b|c|d|e": {}}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key format")
}

// ============================================================
// BulkDelete — key parsing errors
// ============================================================

func TestBulkDelete_InvalidKeyFormat(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkDelete(DeleteRequest{Data: map[string][]Data{"bad": {}}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key format")
}

func TestBulkDelete_EmptyData(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkDelete(DeleteRequest{Data: map[string][]Data{}})
	assert.NoError(t, err)
}

// ============================================================
// BulkUpsertPayload — key parsing errors
// ============================================================

func TestBulkUpsertPayload_InvalidKeyFormat(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkUpsertPayload(UpsertPayloadRequest{Data: map[string][]Data{"bad|key": {}}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key format")
}

func TestBulkUpsertPayload_EmptyData(t *testing.T) {
	q := &Qdrant{}
	err := q.BulkUpsertPayload(UpsertPayloadRequest{Data: map[string][]Data{}})
	assert.NoError(t, err)
}

// ============================================================
// createQdrantClient
// ============================================================

func TestCreateQdrantClient_InvalidPort(t *testing.T) {
	vectorConfig := config.VectorDbConfig{Port: "not_a_number"}
	client, err := createQdrantClient(vectorConfig, "localhost")
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestCreateQdrantClient_ValidPort(t *testing.T) {
	vectorConfig := config.VectorDbConfig{Port: "6334"}
	client, err := createQdrantClient(vectorConfig, "localhost")
	// This will succeed in creating a client (lazy connection)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

// ============================================================
// CreateCollection — needs gRPC client but test error paths
// ============================================================

func TestCreateCollection_NoClient(t *testing.T) {
	cm := new(config.MockConfigManager)
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: cm,
	}
	// getQdrantClient returns nil → will panic when accessing nil client
	assert.Panics(t, func() {
		q.CreateCollection("e1", "m1", "v1", 1)
	})
}

func TestCreateCollection_GetModelConfigError(t *testing.T) {
	cm := new(config.MockConfigManager)
	client := &QdrantClient{WriteDeadline: 1000}
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{"e1:m1:v1": client},
		configManager: cm,
	}
	cm.On("GetModelConfig", "e1", "m1").Return((*config.Model)(nil), fmt.Errorf("model err"))

	err := q.CreateCollection("e1", "m1", "v1", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model err")
}

func TestCreateCollection_GetVariantConfigError(t *testing.T) {
	cm := new(config.MockConfigManager)
	client := &QdrantClient{WriteDeadline: 1000}
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{"e1:m1:v1": client},
		configManager: cm,
	}
	cm.On("GetModelConfig", "e1", "m1").Return(&config.Model{
		ModelConfig: config.ModelConfig{VectorDimension: 128, DistanceFunction: "COSINE"},
	}, nil)
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return((*config.Variant)(nil), fmt.Errorf("variant err"))

	err := q.CreateCollection("e1", "m1", "v1", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variant err")
}

// ============================================================
// DeleteCollection — error paths
// ============================================================

func TestDeleteCollection_NoClient(t *testing.T) {
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: new(config.MockConfigManager),
	}
	assert.Panics(t, func() {
		q.DeleteCollection("e1", "m1", "v1", 1)
	})
}

// ============================================================
// UpdateIndexingThreshold — error paths
// ============================================================

func TestUpdateIndexingThreshold_NoClient(t *testing.T) {
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: new(config.MockConfigManager),
	}
	assert.Panics(t, func() {
		q.UpdateIndexingThreshold("e1", "m1", "v1", 1, "20000")
	})
}

// ============================================================
// CreateFieldIndexes — error paths
// ============================================================

func TestCreateFieldIndexes_GetVariantConfigError(t *testing.T) {
	cm := new(config.MockConfigManager)
	client := &QdrantClient{WriteDeadline: 1000}
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{"e1:m1:v1": client},
		configManager: cm,
	}
	cm.On("GetVariantConfig", "e1", "m1", "v1").Return((*config.Variant)(nil), fmt.Errorf("variant err"))

	err := q.CreateFieldIndexes("e1", "m1", "v1", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variant err")
}

// ============================================================
// RefreshClients
// ============================================================

func TestRefreshClients_DeleteEvent(t *testing.T) {
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: new(config.MockConfigManager),
	}
	err := q.RefreshClients("key", "value", "DELETE")
	assert.NoError(t, err)
}

func TestRefreshClients_EmptyKey(t *testing.T) {
	cm := new(config.MockConfigManager)
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: cm,
	}
	// extractQdrantKey with key that has no matching parts → returns "", "", ""
	err := q.RefreshClients("/some/random/path", "value", "PUT")
	assert.NoError(t, err)
}

func TestRefreshClients_NonQdrantVariant(t *testing.T) {
	cm := new(config.MockConfigManager)
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: cm,
	}
	cm.On("GetVariantConfig", "entity1", "model1", "variant1").Return(&config.Variant{
		VectorDbType: enums.NGT, // not QDRANT
	}, nil)

	err := q.RefreshClients("/some/root/skye/entity1/models/model1/variants/variant1/vector-db-config", "value", "PUT")
	assert.NoError(t, err)
}

func TestRefreshClients_GetVariantConfigError(t *testing.T) {
	cm := new(config.MockConfigManager)
	q := &Qdrant{
		QdrantClients: map[string]*QdrantClient{},
		configManager: cm,
	}
	cm.On("GetVariantConfig", "entity1", "model1", "variant1").Return((*config.Variant)(nil), fmt.Errorf("config err"))

	err := q.RefreshClients("/some/root/skye/entity1/models/model1/variants/variant1/vector-db-config", "value", "PUT")
	assert.NoError(t, err) // returns nil on error, just logs
}
