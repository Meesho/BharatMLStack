package similar_candidate

import (
	"fmt"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	embeddingRepo "github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	protosd "google.golang.org/protobuf/proto"
)

// ============================================================
// Mock: inmemorycache.Database
// ============================================================

type mockInMemCache struct {
	mock.Mock
}

func (m *mockInMemCache) MGet(keys map[string]repositories.CacheStruct, metricTags []string) map[string][]byte {
	args := m.Called(keys, metricTags)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string][]byte)
}

func (m *mockInMemCache) MSet(responseData map[string]repositories.CandidateResponseStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, byteResponseMap map[string][]byte, metricTags []string) {
	m.Called(responseData, missingCacheKeys, ttl, byteResponseMap, metricTags)
}

func (m *mockInMemCache) MSetDotProduct(cacheKeys map[string]repositories.CacheStruct, foundcacheKeys map[string]repositories.CacheStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, metricTags []string) {
	m.Called(cacheKeys, foundcacheKeys, missingCacheKeys, ttl, metricTags)
}

// ============================================================
// Mock: distributedcache.Database
// ============================================================

type mockDistCache struct {
	mock.Mock
}

func (m *mockDistCache) MGet(keys map[string]repositories.CacheStruct, metricTags []string) (map[string][]byte, error) {
	args := m.Called(keys, metricTags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *mockDistCache) MSet(responseData map[string]repositories.CandidateResponseStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, byteResponseMap map[string][]byte, metricTags []string) {
	m.Called(responseData, missingCacheKeys, ttl, byteResponseMap, metricTags)
}

func (m *mockDistCache) MSetDotProduct(cacheKeys map[string]repositories.CacheStruct, foundcacheKeys map[string]repositories.CacheStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, metricTags []string) {
	m.Called(cacheKeys, foundcacheKeys, missingCacheKeys, ttl, metricTags)
}

// ============================================================
// Mock: embedding.Store
// ============================================================

type mockEmbeddingStore struct {
	mock.Mock
}

func (m *mockEmbeddingStore) BulkQuery(storeId string, bulkQuery *embeddingRepo.BulkQuery, queryType string) error {
	args := m.Called(storeId, bulkQuery, queryType)
	return args.Error(0)
}

func (m *mockEmbeddingStore) BulkQueryConsumer(storeId string, bulkQuery *embeddingRepo.BulkQuery) (map[string]map[string]interface{}, error) {
	args := m.Called(storeId, bulkQuery)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]map[string]interface{}), args.Error(1)
}

func (m *mockEmbeddingStore) Persist(storeId string, ttl int, payload embeddingRepo.Payload) error {
	args := m.Called(storeId, ttl, payload)
	return args.Error(0)
}

// helper to build a HandlerV1 with mocks
func newTestHandler() (*HandlerV1, *config.MockConfigManager, *mockInMemCache, *mockDistCache, *mockEmbeddingStore) {
	cm := new(config.MockConfigManager)
	inMem := new(mockInMemCache)
	dist := new(mockDistCache)
	emb := new(mockEmbeddingStore)
	h := &HandlerV1{
		configManager:    cm,
		inMemCache:       inMem,
		distributedCache: dist,
		embeddingStore:   emb,
	}
	return h, cm, inMem, dist, emb
}

// Test validation scenarios
func TestValidateSkyeRequest(t *testing.T) {
	t.Run("Scenario 1: Valid request with embeddings", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:     "test_entity",
			ModelName:  "test_model",
			Variant:    "test_variant",
			Limit:      5,
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
		}

		valid, msg := validateSkyeRequest(request)
		assert.True(t, valid)
		assert.Empty(t, msg)
	})

	t.Run("Scenario 2: Valid request with candidate IDs", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:       "test_entity",
			ModelName:    "test_model",
			Variant:      "test_variant",
			Limit:        5,
			CandidateIds: []string{"candidate1", "candidate2"},
		}

		valid, msg := validateSkyeRequest(request)
		assert.True(t, valid)
		assert.Empty(t, msg)
	})

	t.Run("Invalid request - missing entity", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:     "",
			ModelName:  "test_model",
			Variant:    "test_variant",
			Limit:      5,
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
		}

		valid, msg := validateSkyeRequest(request)
		assert.False(t, valid)
		assert.Equal(t, "Entity is required", msg)
	})

	t.Run("Invalid request - missing model name", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:     "test_entity",
			ModelName:  "",
			Variant:    "test_variant",
			Limit:      5,
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
		}

		valid, msg := validateSkyeRequest(request)
		assert.False(t, valid)
		assert.Equal(t, "modelName is required", msg)
	})

	t.Run("Invalid request - missing variant", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:     "test_entity",
			ModelName:  "test_model",
			Variant:    "",
			Limit:      5,
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
		}

		valid, msg := validateSkyeRequest(request)
		assert.False(t, valid)
		assert.Equal(t, "variant is required", msg)
	})

	t.Run("Invalid request - zero limit", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:     "test_entity",
			ModelName:  "test_model",
			Variant:    "test_variant",
			Limit:      0,
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
		}

		valid, msg := validateSkyeRequest(request)
		assert.False(t, valid)
		assert.Equal(t, "limit is required", msg)
	})

	t.Run("Invalid request - no embeddings or candidate IDs", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:    "test_entity",
			ModelName: "test_model",
			Variant:   "test_variant",
			Limit:     5,
		}

		valid, msg := validateSkyeRequest(request)
		assert.False(t, valid)
		assert.Equal(t, "candidateIds or embeddings is required", msg)
	})
}

// Test cache key generation functions
func TestCacheKeyGeneration(t *testing.T) {
	t.Run("GetCacheKeysForEmbeddings", func(t *testing.T) {
		request := SkyeStructRequest{
			Entity:     "test_entity",
			ModelName:  "test_model",
			Variant:    "test_variant",
			Embeddings: [][]float32{{0.1, 0.2, 0.3}},
			Filters:    make([][]*pb.Filter, 1),
			Attributes: []string{"attr1"},
		}

		cacheKeys := GetCacheKeysForEmbeddings(request)

		assert.Equal(t, 1, len(cacheKeys))
		for key, cacheStruct := range cacheKeys {
			assert.NotEmpty(t, key)
			assert.Equal(t, []int{0}, cacheStruct.Index)
			assert.Equal(t, []float32{0.1, 0.2, 0.3}, cacheStruct.Embedding)
			assert.Empty(t, cacheStruct.CandidateId)
		}
	})

	t.Run("GetCacheKeysForCandidateIds", func(t *testing.T) {
		request := SkyeStructRequest{
			Entity:       "test_entity",
			ModelName:    "test_model",
			Variant:      "test_variant",
			CandidateIds: []string{"candidate1", "candidate2"},
			Filters:      make([][]*pb.Filter, 2),
			Attributes:   []string{"attr1"},
		}

		cacheKeys := GetCacheKeysForCandidateIds(request)

		assert.Equal(t, 2, len(cacheKeys))

		foundCandidate1 := false
		foundCandidate2 := false

		for key, cacheStruct := range cacheKeys {
			assert.NotEmpty(t, key)
			if cacheStruct.CandidateId == "candidate1" {
				foundCandidate1 = true
				assert.Equal(t, []int{0}, cacheStruct.Index)
			} else if cacheStruct.CandidateId == "candidate2" {
				foundCandidate2 = true
				assert.Equal(t, []int{1}, cacheStruct.Index)
			}
			assert.Nil(t, cacheStruct.Embedding)
		}

		assert.True(t, foundCandidate1)
		assert.True(t, foundCandidate2)
	})

	t.Run("buildDetailedCacheKey", func(t *testing.T) {
		key := buildDetailedCacheKey("prefix", "entity", "model", "variant", "id", "filter", "attribute")
		expected := "prefix:entity:model:variant:id:filter:attribute:V2"
		assert.Equal(t, expected, key)
	})
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getTags", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:    "test_entity",
			ModelName: "test_model",
			Variant:   "test_variant",
		}

		tags := getTags(request, RequestTypeSimilarCandidate)
		expected := []string{"entity", "test_entity", "model_name", "test_model", "variant", "test_variant", "request_type", RequestTypeSimilarCandidate}
		assert.Equal(t, expected, tags)
	})

	t.Run("generateResponse", func(t *testing.T) {
		responseMap := map[string]repositories.CandidateResponseStruct{
			"key1": {Index: []int{0}, Response: &pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{{Id: "candidate1"}}}},
			"key2": {Index: []int{1}, Response: &pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{{Id: "candidate2"}}}},
		}

		responses := generateResponse(responseMap, 2)

		assert.Equal(t, 2, len(responses))
		// Check that responses are properly ordered by index
		assert.NotNil(t, responses[0])
		assert.NotNil(t, responses[1])
		assert.Equal(t, "candidate1", responses[0].Candidates[0].Id)
		assert.Equal(t, "candidate2", responses[1].Candidates[0].Id)
	})
}

// Test request adaptation
func TestAdaptSkyeRequestToStruct(t *testing.T) {
	t.Run("Adapt request with embeddings", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:     "test_entity",
			ModelName:  "test_model",
			Variant:    "test_variant",
			Limit:      5,
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
			Attribute:  []string{"attr1"},
		}

		adapted := adaptSkyeRequestToStruct(request)

		assert.Equal(t, "test_entity", adapted.Entity)
		assert.Equal(t, "test_model", adapted.ModelName)
		assert.Equal(t, "test_variant", adapted.Variant)
		assert.Equal(t, 5, adapted.Limit)
		assert.Equal(t, 1, len(adapted.Embeddings))
		assert.Equal(t, []float32{0.1, 0.2}, adapted.Embeddings[0])
		assert.Equal(t, []string{"attr1"}, adapted.Attributes)
	})

	t.Run("Adapt request with candidate IDs", func(t *testing.T) {
		request := &pb.SkyeRequest{
			Entity:       "test_entity",
			ModelName:    "test_model",
			Variant:      "test_variant",
			Limit:        10,
			CandidateIds: []string{"candidate1", "candidate2"},
			Attribute:    []string{"attr1"},
		}

		adapted := adaptSkyeRequestToStruct(request)

		assert.Equal(t, "test_entity", adapted.Entity)
		assert.Equal(t, "test_model", adapted.ModelName)
		assert.Equal(t, "test_variant", adapted.Variant)
		assert.Equal(t, 10, adapted.Limit)
		assert.Equal(t, []string{"candidate1", "candidate2"}, adapted.CandidateIds)
		assert.Equal(t, []string{"attr1"}, adapted.Attributes)
		assert.Equal(t, 0, len(adapted.Embeddings))
	})
}

// Test modifyStagingSimilarCandidatesRequest
func TestModifyStagingSimilarCandidatesRequest(t *testing.T) {
	originalAppConfig := appConfig
	defer func() { appConfig = originalAppConfig }()

	t.Run("basic staging overrides - ModelName, Variant, Entity", func(t *testing.T) {
		appConfig = structs.Configs{
			StagingDefaultModelName: "staging_model",
			StagingDefaultVariant:   "staging_variant",
			StagingDefaultEntity:    "staging_entity",
		}
		request := &pb.SkyeRequest{
			Entity:    "orig_entity",
			ModelName: "orig_model",
			Variant:   "orig_variant",
		}

		modifyStagingSimilarCandidatesRequest(request)

		assert.Equal(t, "staging_entity", request.Entity)
		assert.Equal(t, "staging_model", request.ModelName)
		assert.Equal(t, "staging_variant", request.Variant)
	})

	t.Run("embedding truncation when longer than StagingDefaultEmbeddingLength", func(t *testing.T) {
		appConfig = structs.Configs{StagingDefaultEmbeddingLength: 3}
		request := &pb.SkyeRequest{
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5}}},
		}

		modifyStagingSimilarCandidatesRequest(request)

		assert.Equal(t, []float64{0.1, 0.2, 0.3}, request.Embeddings[0].Embedding)
	})

	t.Run("embedding padding when shorter than StagingDefaultEmbeddingLength", func(t *testing.T) {
		appConfig = structs.Configs{StagingDefaultEmbeddingLength: 5}
		request := &pb.SkyeRequest{
			Embeddings: []*pb.Embedding{{Embedding: []float64{0.1, 0.2}}},
		}

		modifyStagingSimilarCandidatesRequest(request)

		assert.Len(t, request.Embeddings[0].Embedding, 5)
		assert.Equal(t, []float64{0.1, 0.2, 0, 0, 0}, request.Embeddings[0].Embedding)
	})

	t.Run("no embeddings - should not panic", func(t *testing.T) {
		appConfig = structs.Configs{StagingDefaultEmbeddingLength: 5}
		request := &pb.SkyeRequest{
			Entity: "e", ModelName: "m", Variant: "v",
		}

		modifyStagingSimilarCandidatesRequest(request)

		assert.Nil(t, request.Embeddings)
	})
}

// Test ProcessCacheResponse
func TestProcessCacheResponse(t *testing.T) {
	t.Run("cache hit scenario - valid protobuf bytes in cachedData", func(t *testing.T) {
		cachedBytes, _ := protosd.Marshal(&pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{{Id: "c1"}}})
		keys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, Embedding: []float32{0.1}},
		}
		cachedData := map[string][]byte{"key1": cachedBytes}
		resMap := make(map[string]repositories.CandidateResponseStruct)
		commonMetricTags := []string{}
		limit := 10

		missing := ProcessCacheResponse(keys, cachedData, resMap, limit, commonMetricTags, "in_memory", true)

		assert.NotEmpty(t, resMap["key1"])
		assert.Equal(t, "c1", resMap["key1"].Response.Candidates[0].Id)
		assert.NotContains(t, missing, "key1")
	})

	t.Run("cache miss scenario - key not in cachedData", func(t *testing.T) {
		keys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, Embedding: []float32{0.1}},
		}
		cachedData := map[string][]byte{}
		resMap := make(map[string]repositories.CandidateResponseStruct)
		commonMetricTags := []string{}
		limit := 10

		missing := ProcessCacheResponse(keys, cachedData, resMap, limit, commonMetricTags, "in_memory", true)

		assert.Empty(t, resMap)
		assert.Contains(t, missing, "key1")
		assert.Equal(t, []int{0}, missing["key1"].Index)
	})

	t.Run("partial hit - candidates < limit-1, partialHitDisabled=false", func(t *testing.T) {
		cachedBytes, _ := protosd.Marshal(&pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{{Id: "c1"}, {Id: "c2"}}})
		keys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, Embedding: []float32{0.1}},
		}
		cachedData := map[string][]byte{"key1": cachedBytes}
		resMap := make(map[string]repositories.CandidateResponseStruct)
		commonMetricTags := []string{}
		limit := 5

		missing := ProcessCacheResponse(keys, cachedData, resMap, limit, commonMetricTags, "in_memory", false)

		assert.Empty(t, resMap)
		assert.Contains(t, missing, "key1")
	})

	t.Run("truncation when candidates > limit", func(t *testing.T) {
		cachedBytes, _ := protosd.Marshal(&pb.CandidateResponse{
			Candidates: []*pb.CandidateResponse_Candidate{
				{Id: "c1"}, {Id: "c2"}, {Id: "c3"}, {Id: "c4"}, {Id: "c5"},
			},
		})
		keys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, Embedding: []float32{0.1}},
		}
		cachedData := map[string][]byte{"key1": cachedBytes}
		resMap := make(map[string]repositories.CandidateResponseStruct)
		commonMetricTags := []string{}
		limit := 3

		missing := ProcessCacheResponse(keys, cachedData, resMap, limit, commonMetricTags, "in_memory", true)

		assert.Len(t, resMap["key1"].Response.Candidates, 3)
		assert.Equal(t, []string{"c1", "c2", "c3"}, []string{resMap["key1"].Response.Candidates[0].Id, resMap["key1"].Response.Candidates[1].Id, resMap["key1"].Response.Candidates[2].Id})
		assert.NotContains(t, missing, "key1")
	})
}

// Test parseVectorDbResponse
func TestParseVectorDbResponse(t *testing.T) {
	t.Run("with isEmbeddingsRequest=true - no self-filtering", func(t *testing.T) {
		keys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, CandidateId: "self_id"},
		}
		batchResp := &vector.BatchQueryResponse{
			SimilarCandidatesList: map[string][]*vector.SimilarCandidate{
				"key1": {
					{Id: "self_id", Score: 1.0},
					{Id: "other_id", Score: 0.9},
				},
			},
		}
		responseMap := make(map[string]repositories.CandidateResponseStruct)

		parseVectorDbResponse(keys, batchResp, responseMap, true)

		assert.Len(t, responseMap["key1"].Response.Candidates, 2)
		assert.Equal(t, "self_id", responseMap["key1"].Response.Candidates[0].Id)
		assert.Equal(t, "other_id", responseMap["key1"].Response.Candidates[1].Id)
	})

	t.Run("with isEmbeddingsRequest=false - filters out candidate whose ID matches key's CandidateId", func(t *testing.T) {
		keys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, CandidateId: "self_id"},
		}
		batchResp := &vector.BatchQueryResponse{
			SimilarCandidatesList: map[string][]*vector.SimilarCandidate{
				"key1": {
					{Id: "self_id", Score: 1.0},
					{Id: "other_id", Score: 0.9},
				},
			},
		}
		responseMap := make(map[string]repositories.CandidateResponseStruct)

		parseVectorDbResponse(keys, batchResp, responseMap, false)

		assert.Len(t, responseMap["key1"].Response.Candidates, 1)
		assert.Equal(t, "other_id", responseMap["key1"].Response.Candidates[0].Id)
	})
}

// Test buildVectorBatchQueryFromEmbeddings
func TestBuildVectorBatchQueryFromEmbeddings(t *testing.T) {
	variantConfig := &config.Variant{
		VectorDbConfig:      config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion: 1,
	}

	t.Run("normal case with embeddings", func(t *testing.T) {
		request := SkyeStructRequest{
			Entity:     "e1",
			ModelName:  "m1",
			Variant:    "v1",
			Limit:      5,
			Embeddings: [][]float32{{0.1, 0.2, 0.3}},
			Attributes: []string{"attr1"},
		}
		cacheKeys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, Embedding: []float32{0.1, 0.2, 0.3}, Filters: nil},
		}
		responseMap := make(map[string]repositories.CandidateResponseStruct)

		result := buildVectorBatchQueryFromEmbeddings(request, variantConfig, cacheKeys, responseMap)

		assert.Equal(t, "e1", result.Entity)
		assert.Equal(t, "m1", result.Model)
		assert.Equal(t, "v1", result.Variant)
		assert.Len(t, result.RequestList, 1)
		assert.Equal(t, "key1", result.RequestList[0].CacheKey)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, result.RequestList[0].Embedding)
		assert.Empty(t, responseMap)
	})

	t.Run("empty embedding - should populate responseMap with empty candidates", func(t *testing.T) {
		request := SkyeStructRequest{
			Entity:     "e1",
			ModelName:  "m1",
			Variant:    "v1",
			Limit:      5,
			Attributes: []string{"attr1"},
		}
		cacheKeys := map[string]repositories.CacheStruct{
			"key1": {Index: []int{0}, Embedding: []float32{}, Filters: nil},
		}
		responseMap := make(map[string]repositories.CandidateResponseStruct)

		result := buildVectorBatchQueryFromEmbeddings(request, variantConfig, cacheKeys, responseMap)

		assert.Empty(t, result.RequestList)
		assert.NotEmpty(t, responseMap["key1"])
		assert.Empty(t, responseMap["key1"].Response.Candidates)
	})

	t.Run("empty cacheKeys", func(t *testing.T) {
		request := SkyeStructRequest{
			Entity:     "e1",
			ModelName:  "m1",
			Variant:    "v1",
			Limit:      5,
			Embeddings: [][]float32{{0.1, 0.2}},
			Attributes: []string{"attr1"},
		}
		cacheKeys := map[string]repositories.CacheStruct{}
		responseMap := make(map[string]repositories.CandidateResponseStruct)

		result := buildVectorBatchQueryFromEmbeddings(request, variantConfig, cacheKeys, responseMap)

		assert.Empty(t, result.RequestList)
		assert.Empty(t, responseMap)
	})
}

// ============================================================
// RetrieveSimilarCandidates — comprehensive tests
// ============================================================

func TestRetrieveSimilarCandidates_EmbeddingsRequest_NoCaching_Success(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2, 0.3}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{"attr1"},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:              enums.QDRANT,
		VectorDbConfig:            config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:       1,
		InMemoryCachingEnabled:    false,
		DistributedCachingEnabled: false,
	}
	tags := []string{"entity", "e1"}

	// Pre-compute cache keys so mock response maps match
	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{{Id: "100", Score: 0.9}}
	}

	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1) // expectedLength = 1 embedding
	assert.NotNil(t, resp.Responses[0])
	assert.Equal(t, "100", resp.Responses[0].Candidates[0].Id)
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_CandidateIds_NoCaching_Success(t *testing.T) {
	h, _, _, _, emb := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:       "e1",
		ModelName:    "m1",
		Variant:      "v1",
		Limit:        5,
		CandidateIds: []string{"cand1", "cand2"},
		Filters:      make([][]*pb.Filter, 2),
		Attributes:   []string{"attr1"},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:              enums.QDRANT,
		VectorDbConfig:            config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:       1,
		EmbeddingStoreReadVersion: 1,
		InMemoryCachingEnabled:    false,
		DistributedCachingEnabled: false,
	}
	tags := []string{"entity", "e1"}

	// BulkQuery on embedding store called for candidateIds requests
	emb.On("BulkQuery", "store1", mock.AnythingOfType("*embedding.BulkQuery"), "similar_candidate").Return(nil)

	// BatchQuery returns empty — embeddings not populated by mock store, so they'll be empty vectors
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: map[string][]*vector.SimilarCandidate{}}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 2) // expectedLength = 2 candidateIds
	emb.AssertCalled(t, "BulkQuery", "store1", mock.AnythingOfType("*embedding.BulkQuery"), "similar_candidate")
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_CandidateIds_EmbeddingStoreError(t *testing.T) {
	h, _, _, _, emb := newTestHandler()

	request := SkyeStructRequest{
		Entity:       "e1",
		ModelName:    "m1",
		Variant:      "v1",
		Limit:        5,
		CandidateIds: []string{"cand1"},
		Filters:      make([][]*pb.Filter, 1),
		Attributes:   []string{"attr1"},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:              enums.QDRANT,
		VectorDbConfig:            config.VectorDbConfig{Params: map[string]string{}},
		EmbeddingStoreReadVersion: 1,
	}
	tags := []string{"entity", "e1"}

	emb.On("BulkQuery", "store1", mock.AnythingOfType("*embedding.BulkQuery"), "similar_candidate").Return(fmt.Errorf("store err"))

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "error fetching embeddings from store")
}

func TestRetrieveSimilarCandidates_VectorDBBatchQueryError(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:        enums.QDRANT,
		VectorDbConfig:      config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion: 1,
	}
	tags := []string{"entity", "e1"}

	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		(*vector.BatchQueryResponse)(nil), fmt.Errorf("vectordb err"))

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Unexpected Error fetching similar candidates from vectorDB")
}

func TestRetrieveSimilarCandidates_WithInMemoryCache_FullHit(t *testing.T) {
	h, _, inMem, _, _ := newTestHandler()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2, 0.3}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{"attr1"},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:            enums.QDRANT,
		VectorDbConfig:          config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:     1,
		InMemoryCachingEnabled:  true,
		InMemoryCacheTTLSeconds: 60,
		PartialHitDisabled:      true,
	}
	tags := []string{"entity", "e1"}

	// Pre-compute cache keys and build cached bytes
	cacheKeys := GetCacheKeysForEmbeddings(request)
	cachedResp := &pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{
		{Id: "cached1", Meta: &pb.CandidateResponse_Candidate_Meta{Score: 0.99}},
		{Id: "cached2", Meta: &pb.CandidateResponse_Candidate_Meta{Score: 0.88}},
	}}
	cachedBytes, _ := protosd.Marshal(cachedResp)
	cachedData := make(map[string][]byte)
	for k := range cacheKeys {
		cachedData[k] = cachedBytes
	}

	inMem.On("MGet", mock.Anything, tags).Return(cachedData)

	// After full cache hit, cacheKeys are deleted → len(cacheKeys) == 0 → no vector DB call
	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	assert.Equal(t, "cached1", resp.Responses[0].Candidates[0].Id)
	assert.Equal(t, "cached2", resp.Responses[0].Candidates[1].Id)
	inMem.AssertCalled(t, "MGet", mock.Anything, tags)
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_WithInMemoryCache_Miss_FallsToVectorDB(t *testing.T) {
	h, _, inMem, _, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.5, 0.6}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:            enums.QDRANT,
		VectorDbConfig:          config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:     1,
		InMemoryCachingEnabled:  true,
		InMemoryCacheTTLSeconds: 60,
	}
	tags := []string{"entity", "e1"}

	// Cache miss
	inMem.On("MGet", mock.Anything, tags).Return(map[string][]byte{})
	inMem.On("MSet", mock.Anything, mock.Anything, 60, mock.Anything, tags).Return()

	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{{Id: "vdb1", Score: 0.77}}
	}
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	assert.Equal(t, "vdb1", resp.Responses[0].Candidates[0].Id)
	time.Sleep(20 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_WithDistributedCache_Error(t *testing.T) {
	h, _, _, dist, _ := newTestHandler()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2, 0.3}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{"attr1"},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:              enums.QDRANT,
		VectorDbConfig:            config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:       1,
		DistributedCachingEnabled: true,
	}
	tags := []string{"entity", "e1"}

	dist.On("MGet", mock.Anything, tags).Return((map[string][]byte)(nil), fmt.Errorf("redis err"))

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "error fetching similar candidates from distributed cache")
}

func TestRetrieveSimilarCandidates_WithDistributedCache_Miss_ThenVectorDB(t *testing.T) {
	h, _, _, dist, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:               enums.QDRANT,
		VectorDbConfig:             config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:        1,
		DistributedCachingEnabled:  true,
		DistributedCacheTTLSeconds: 300,
	}
	tags := []string{"entity", "e1"}

	dist.On("MGet", mock.Anything, tags).Return(map[string][]byte{}, nil)
	dist.On("MSet", mock.Anything, mock.Anything, 300, mock.Anything, tags).Return()

	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{{Id: "vdb_dist", Score: 0.8}}
	}
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	dist.AssertCalled(t, "MGet", mock.Anything, tags)
	mockVectorDb.AssertCalled(t, "BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags)
	time.Sleep(20 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_BothCaches_Miss_ThenVectorDB_WithResults(t *testing.T) {
	h, _, inMem, dist, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{"name"},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:               enums.QDRANT,
		VectorDbConfig:             config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:        1,
		InMemoryCachingEnabled:     true,
		InMemoryCacheTTLSeconds:    30,
		DistributedCachingEnabled:  true,
		DistributedCacheTTLSeconds: 300,
	}
	tags := []string{"entity", "e1"}

	// Both caches miss
	inMem.On("MGet", mock.Anything, tags).Return(map[string][]byte{})
	dist.On("MGet", mock.Anything, tags).Return(map[string][]byte{}, nil)
	inMem.On("MSet", mock.Anything, mock.Anything, 30, mock.Anything, tags).Return()
	dist.On("MSet", mock.Anything, mock.Anything, 300, mock.Anything, tags).Return()

	// Pre-compute cache keys and build matching response
	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{
			{Id: "100", Score: 0.95, Payload: map[string]string{"name": "item100"}},
			{Id: "200", Score: 0.85, Payload: map[string]string{"name": "item200"}},
		}
	}
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	assert.Len(t, resp.Responses[0].Candidates, 2)
	assert.Equal(t, "100", resp.Responses[0].Candidates[0].Id)
	assert.Equal(t, "200", resp.Responses[0].Candidates[1].Id)
	assert.InDelta(t, 0.95, resp.Responses[0].Candidates[0].Meta.Score, 0.01)
	time.Sleep(20 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_DistributedCache_FullHit(t *testing.T) {
	h, _, _, dist, _ := newTestHandler()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.9, 0.8}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:               enums.QDRANT,
		VectorDbConfig:             config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:        1,
		DistributedCachingEnabled:  true,
		DistributedCacheTTLSeconds: 300,
		PartialHitDisabled:         true,
	}
	tags := []string{"entity", "e1"}

	cacheKeys := GetCacheKeysForEmbeddings(request)
	cachedResp := &pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{
		{Id: "dc1", Meta: &pb.CandidateResponse_Candidate_Meta{Score: 0.75}},
	}}
	cachedBytes, _ := protosd.Marshal(cachedResp)
	cachedData := make(map[string][]byte)
	for k := range cacheKeys {
		cachedData[k] = cachedBytes
	}

	dist.On("MGet", mock.Anything, tags).Return(cachedData, nil)

	// Full hit → no vector DB call
	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	assert.Equal(t, "dc1", resp.Responses[0].Candidates[0].Id)
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_CandidateIds_VectorDBReturnsWithSelfFilter(t *testing.T) {
	h, _, _, _, emb := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:       "e1",
		ModelName:    "m1",
		Variant:      "v1",
		Limit:        5,
		CandidateIds: []string{"cand1"},
		Filters:      make([][]*pb.Filter, 1),
		Attributes:   []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:              enums.QDRANT,
		VectorDbConfig:            config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:       1,
		EmbeddingStoreReadVersion: 1,
	}
	tags := []string{"entity", "e1"}

	emb.On("BulkQuery", "store1", mock.AnythingOfType("*embedding.BulkQuery"), "similar_candidate").Return(nil)

	// Pre-compute cache keys so we can build a response with self-candidate present
	cacheKeys := GetCacheKeysForCandidateIds(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k, v := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{
			{Id: v.CandidateId, Score: 1.0}, // self — should be filtered out
			{Id: "cand2", Score: 0.9},
		}
	}
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	// Self-candidate "cand1" filtered out (isEmbeddingsRequest=false)
	assert.Len(t, resp.Responses[0].Candidates, 1)
	assert.Equal(t, "cand2", resp.Responses[0].Candidates[0].Id)
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_MultipleEmbeddings_Success(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      3,
		Embeddings: [][]float32{{0.1, 0.2}, {0.3, 0.4}, {0.5, 0.6}},
		Filters:    make([][]*pb.Filter, 3),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:        enums.QDRANT,
		VectorDbConfig:      config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion: 1,
	}
	tags := []string{"entity", "e1"}

	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{{Id: "r1", Score: 0.9}}
	}
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 3) // 3 embeddings
	for _, r := range resp.Responses {
		assert.NotNil(t, r)
		assert.Equal(t, "r1", r.Candidates[0].Id)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_EmptyEmbedding_SkipsVectorQuery(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{}}, // empty embedding
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:        enums.QDRANT,
		VectorDbConfig:      config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion: 1,
	}
	tags := []string{"entity", "e1"}

	// BatchQuery still called but with empty RequestList → empty response is fine
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: map[string][]*vector.SimilarCandidate{}}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	assert.Empty(t, resp.Responses[0].Candidates)
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_InMemCacheHit_DistMiss_NoVectorDB(t *testing.T) {
	// When in-memory cache hits, the key is deleted from cacheKeys.
	// Distributed cache then sees no keys → no distributed call needed.
	// VectorDB should not be called either.
	h, _, inMem, _, _ := newTestHandler()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.7, 0.8}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:            enums.QDRANT,
		VectorDbConfig:          config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:     1,
		InMemoryCachingEnabled:  true,
		InMemoryCacheTTLSeconds: 60,
		PartialHitDisabled:      true,
		// DistributedCachingEnabled is false
	}
	tags := []string{"entity", "e1"}

	cacheKeys := GetCacheKeysForEmbeddings(request)
	cachedResp := &pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{
		{Id: "mem1"},
	}}
	cachedBytes, _ := protosd.Marshal(cachedResp)
	cachedData := make(map[string][]byte)
	for k := range cacheKeys {
		cachedData[k] = cachedBytes
	}
	inMem.On("MGet", mock.Anything, tags).Return(cachedData)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)
	assert.NotNil(t, resp.Responses[0])
	assert.Equal(t, "mem1", resp.Responses[0].Candidates[0].Id)
	time.Sleep(10 * time.Millisecond)
}

func TestRetrieveSimilarCandidates_CacheMSet_CalledInBackground(t *testing.T) {
	// Verify that MSet is called asynchronously for both caches when there are missing keys.
	h, _, inMem, dist, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.4, 0.5}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:               enums.QDRANT,
		VectorDbConfig:             config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:        1,
		InMemoryCachingEnabled:     true,
		InMemoryCacheTTLSeconds:    30,
		DistributedCachingEnabled:  true,
		DistributedCacheTTLSeconds: 300,
	}
	tags := []string{"entity", "e1"}

	inMem.On("MGet", mock.Anything, tags).Return(map[string][]byte{})
	dist.On("MGet", mock.Anything, tags).Return(map[string][]byte{}, nil)
	inMem.On("MSet", mock.Anything, mock.Anything, 30, mock.Anything, tags).Return()
	dist.On("MSet", mock.Anything, mock.Anything, 300, mock.Anything, tags).Return()

	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{{Id: "x1", Score: 0.5}}
	}
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Wait for background goroutine to complete MSet calls
	time.Sleep(50 * time.Millisecond)
	dist.AssertCalled(t, "MSet", mock.Anything, mock.Anything, 300, mock.Anything, tags)
	inMem.AssertCalled(t, "MSet", mock.Anything, mock.Anything, 30, mock.Anything, tags)
}

func TestRetrieveSimilarCandidates_TestQueryFired_WhenPercentage100(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:     "e1",
		ModelName:  "m1",
		Variant:    "v1",
		Limit:      5,
		Embeddings: [][]float32{{0.1, 0.2}},
		Filters:    make([][]*pb.Filter, 1),
		Attributes: []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:        enums.QDRANT,
		VectorDbConfig:      config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion: 1,
		TestConfig: config.TestConfig{
			Percentage:   100, // always fire test query
			Entity:       "test_entity",
			Model:        "test_model",
			Variant:      "test_variant",
			Version:      2,
			VectorDbType: enums.QDRANT,
		},
	}
	tags := []string{"entity", "e1"}

	cacheKeys := GetCacheKeysForEmbeddings(request)
	similarList := make(map[string][]*vector.SimilarCandidate)
	for k := range cacheKeys {
		similarList[k] = []*vector.SimilarCandidate{{Id: "p1", Score: 0.6}}
	}

	// Primary BatchQuery
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: similarList}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 1)

	// Wait for the async test query goroutine
	time.Sleep(50 * time.Millisecond)
	// BatchQuery should be called at least twice — primary + test query
	mockVectorDb.AssertNumberOfCalls(t, "BatchQuery", 2)
}

func TestRetrieveSimilarCandidates_MultipleCandidateIds_PartialEmbeddingHit(t *testing.T) {
	// Two candidate IDs, one has embedding (hit from store), the other doesn't
	h, _, _, _, emb := newTestHandler()
	mockVectorDb := new(vector.MockDatabase)
	vector.SetTestInstance(mockVectorDb)
	defer vector.ResetTestInstance()

	request := SkyeStructRequest{
		Entity:       "e1",
		ModelName:    "m1",
		Variant:      "v1",
		Limit:        5,
		CandidateIds: []string{"c1", "c2"},
		Filters:      make([][]*pb.Filter, 2),
		Attributes:   []string{},
	}
	entityConfig := &config.Models{StoreId: "store1"}
	variantConfig := &config.Variant{
		VectorDbType:              enums.QDRANT,
		VectorDbConfig:            config.VectorDbConfig{Params: map[string]string{}},
		VectorDbReadVersion:       1,
		EmbeddingStoreReadVersion: 2,
	}
	tags := []string{"entity", "e1"}

	emb.On("BulkQuery", "store1", mock.AnythingOfType("*embedding.BulkQuery"), "similar_candidate").Return(nil)

	// Empty response (embedding store populates cacheKeys in-place, but our mock doesn't)
	mockVectorDb.On("BatchQuery", mock.AnythingOfType("*vector.BatchQueryRequest"), tags).Return(
		&vector.BatchQueryResponse{SimilarCandidatesList: map[string][]*vector.SimilarCandidate{}}, nil)

	resp, err := h.RetrieveSimilarCandidates(request, entityConfig, variantConfig, tags)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Responses, 2)
	time.Sleep(10 * time.Millisecond)
}
