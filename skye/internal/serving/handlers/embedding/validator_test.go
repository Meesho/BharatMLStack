package embedding

import (
	"testing"

	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/stretchr/testify/assert"
)

func TestIsValidateEmbeddingsRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &pb.SkyeBulkEmbeddingRequest{
			Entity:       "e1",
			ModelName:    "m1",
			Variant:      "v1",
			CandidateIds: []string{"c1"},
		}
		ok, msg := isValidateEmbeddingsRequest(req)
		assert.True(t, ok)
		assert.Empty(t, msg)
	})

	t.Run("missing entity", func(t *testing.T) {
		req := &pb.SkyeBulkEmbeddingRequest{ModelName: "m1", Variant: "v1", CandidateIds: []string{"c1"}}
		ok, msg := isValidateEmbeddingsRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "Entity is Required", msg)
	})

	t.Run("missing model name", func(t *testing.T) {
		req := &pb.SkyeBulkEmbeddingRequest{Entity: "e1", Variant: "v1", CandidateIds: []string{"c1"}}
		ok, msg := isValidateEmbeddingsRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "ModelName is required", msg)
	})

	t.Run("missing variant", func(t *testing.T) {
		req := &pb.SkyeBulkEmbeddingRequest{Entity: "e1", ModelName: "m1", CandidateIds: []string{"c1"}}
		ok, msg := isValidateEmbeddingsRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "Variant is required", msg)
	})

	t.Run("empty candidate IDs", func(t *testing.T) {
		req := &pb.SkyeBulkEmbeddingRequest{Entity: "e1", ModelName: "m1", Variant: "v1", CandidateIds: nil}
		ok, msg := isValidateEmbeddingsRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "candidateIds are required", msg)
	})
}

func TestIsValidDotProductRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &pb.EmbeddingDotProductRequest{
			Entity:       "e1",
			ModelName:    "m1",
			Variant:      "v1",
			Embedding:    []float64{0.1, 0.2},
			CandidateIds: []string{"c1"},
		}
		ok, msg := isValidDotProductRequest(req)
		assert.True(t, ok)
		assert.Empty(t, msg)
	})

	t.Run("missing entity", func(t *testing.T) {
		req := &pb.EmbeddingDotProductRequest{ModelName: "m1", Variant: "v1", Embedding: []float64{0.1}, CandidateIds: []string{"c1"}}
		ok, msg := isValidDotProductRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "Entity is required", msg)
	})

	t.Run("missing model name", func(t *testing.T) {
		req := &pb.EmbeddingDotProductRequest{Entity: "e1", Variant: "v1", Embedding: []float64{0.1}, CandidateIds: []string{"c1"}}
		ok, msg := isValidDotProductRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "ModelName is required", msg)
	})

	t.Run("missing variant", func(t *testing.T) {
		req := &pb.EmbeddingDotProductRequest{Entity: "e1", ModelName: "m1", Embedding: []float64{0.1}, CandidateIds: []string{"c1"}}
		ok, msg := isValidDotProductRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "Variant is required", msg)
	})

	t.Run("empty embedding", func(t *testing.T) {
		req := &pb.EmbeddingDotProductRequest{Entity: "e1", ModelName: "m1", Variant: "v1", CandidateIds: []string{"c1"}}
		ok, msg := isValidDotProductRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "embedding is required", msg)
	})

	t.Run("empty candidate IDs", func(t *testing.T) {
		req := &pb.EmbeddingDotProductRequest{Entity: "e1", ModelName: "m1", Variant: "v1", Embedding: []float64{0.1}}
		ok, msg := isValidDotProductRequest(req)
		assert.False(t, ok)
		assert.Equal(t, "at least one candidateId is required", msg)
	})
}
