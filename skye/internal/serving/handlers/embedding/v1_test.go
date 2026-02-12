package embedding

import (
	"testing"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/stretchr/testify/assert"
)

func TestGetTags(t *testing.T) {
	tags := getTags("e1", "m1", "v1", RequestTypeEmbeddingRetrieval)
	assert.Equal(t, []string{"entity", "e1", "model_name", "m1", "variant", "v1", "request_type", RequestTypeEmbeddingRetrieval}, tags)
}

func TestCalculateDotProduct(t *testing.T) {
	t.Run("same length", func(t *testing.T) {
		// [1,2] . [3,4] = 3+8 = 11
		score := CalculateDotProduct([]float64{1, 2}, []float32{3, 4})
		assert.InDelta(t, 11.0, score, 1e-5)
	})

	t.Run("mismatched length returns 0", func(t *testing.T) {
		score := CalculateDotProduct([]float64{1, 2}, []float32{3})
		assert.Equal(t, 0.0, score)
	})

	t.Run("empty", func(t *testing.T) {
		score := CalculateDotProduct([]float64{}, []float32{})
		assert.Equal(t, 0.0, score)
	})
}

func TestGenerateResponseEmbedding(t *testing.T) {
	responseMap := map[string]repositories.CandidateResponseStruct{
		"k1": {Index: []int{0}, EmbeddingResponse: &pb.CandidateEmbedding{Id: "c1"}},
		"k2": {Index: []int{1}, EmbeddingResponse: &pb.CandidateEmbedding{Id: "c2"}},
	}
	out := generateResponseEmbedding(responseMap, 2)
	assert.Len(t, out, 2)
	assert.NotNil(t, out[0])
	assert.NotNil(t, out[1])
	assert.Equal(t, "c1", out[0].Id)
	assert.Equal(t, "c2", out[1].Id)
}

func TestGenerateResponseDotProduct(t *testing.T) {
	responseMap := map[string]repositories.CandidateResponseStruct{
		"k1": {Index: []int{0}, DotProductResponse: &pb.CandidateEmbeddingScore{CandidateId: "c1", Score: 0.5}},
		"k2": {Index: []int{1}, DotProductResponse: &pb.CandidateEmbeddingScore{CandidateId: "c2", Score: 0.6}},
	}
	out := generateResponseDotProduct(responseMap, 2)
	assert.Len(t, out, 2)
	assert.NotNil(t, out[0])
	assert.NotNil(t, out[1])
	assert.Equal(t, "c1", out[0].CandidateId)
	assert.Equal(t, "c2", out[1].CandidateId)
}

// Test modifyStagingRetrievalRequest
func TestModifyStagingRetrievalRequest(t *testing.T) {
	originalAppConfig := appConfig
	defer func() { appConfig = originalAppConfig }()

	t.Run("basic overrides", func(t *testing.T) {
		appConfig = structs.Configs{
			StagingDefaultModelName: "staging_model",
			StagingDefaultVariant:   "staging_variant",
			StagingDefaultEntity:    "staging_entity",
		}
		request := &pb.SkyeBulkEmbeddingRequest{
			Entity:       "orig_entity",
			ModelName:    "orig_model",
			Variant:      "orig_variant",
			CandidateIds: []string{"c1", "c2"},
		}

		modifyStagingRetrievalRequest(request)

		assert.Equal(t, "staging_entity", request.Entity)
		assert.Equal(t, "staging_model", request.ModelName)
		assert.Equal(t, "staging_variant", request.Variant)
		assert.Equal(t, []string{"c1", "c2"}, request.CandidateIds)
	})
}

// Test modifyStagingDotProductRequest
func TestModifyStagingDotProductRequest(t *testing.T) {
	originalAppConfig := appConfig
	defer func() { appConfig = originalAppConfig }()

	t.Run("basic overrides", func(t *testing.T) {
		appConfig = structs.Configs{
			StagingDefaultModelName: "staging_model",
			StagingDefaultVariant:   "staging_variant",
			StagingDefaultEntity:    "staging_entity",
		}
		request := &pb.EmbeddingDotProductRequest{
			Entity:       "orig_entity",
			ModelName:    "orig_model",
			Variant:      "orig_variant",
			Embedding:    []float64{0.1, 0.2, 0.3},
			CandidateIds: []string{"c1"},
		}

		modifyStagingDotProductRequest(request)

		assert.Equal(t, "staging_entity", request.Entity)
		assert.Equal(t, "staging_model", request.ModelName)
		assert.Equal(t, "staging_variant", request.Variant)
	})

	t.Run("embedding truncation", func(t *testing.T) {
		appConfig = structs.Configs{
			StagingDefaultModelName:       "staging_model",
			StagingDefaultVariant:         "staging_variant",
			StagingDefaultEntity:          "staging_entity",
			StagingDefaultEmbeddingLength: 3,
		}
		request := &pb.EmbeddingDotProductRequest{
			Entity:    "e",
			ModelName: "m",
			Variant:   "v",
			Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		}

		modifyStagingDotProductRequest(request)

		assert.Equal(t, []float64{0.1, 0.2, 0.3}, request.Embedding)
	})

	t.Run("embedding padding", func(t *testing.T) {
		appConfig = structs.Configs{
			StagingDefaultModelName:       "staging_model",
			StagingDefaultVariant:         "staging_variant",
			StagingDefaultEntity:          "staging_entity",
			StagingDefaultEmbeddingLength: 5,
		}
		request := &pb.EmbeddingDotProductRequest{
			Entity:    "e",
			ModelName: "m",
			Variant:   "v",
			Embedding: []float64{0.1, 0.2},
		}

		modifyStagingDotProductRequest(request)

		assert.Len(t, request.Embedding, 5)
		assert.Equal(t, []float64{0.1, 0.2, 0, 0, 0}, request.Embedding)
	})

	t.Run("no embedding", func(t *testing.T) {
		appConfig = structs.Configs{
			StagingDefaultModelName:       "staging_model",
			StagingDefaultVariant:         "staging_variant",
			StagingDefaultEntity:          "staging_entity",
			StagingDefaultEmbeddingLength: 5,
		}
		request := &pb.EmbeddingDotProductRequest{
			Entity:       "e",
			ModelName:    "m",
			Variant:      "v",
			Embedding:    []float64{},
			CandidateIds: []string{"c1"},
		}

		modifyStagingDotProductRequest(request)

		assert.Equal(t, "staging_entity", request.Entity)
		assert.Equal(t, "staging_model", request.ModelName)
		assert.Equal(t, "staging_variant", request.Variant)
		assert.Empty(t, request.Embedding)
	})
}
