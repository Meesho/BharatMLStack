package embedding

import (
	"testing"

	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/stretchr/testify/assert"
)

func TestGetCacheKeysForBulkEmbeddingRequest(t *testing.T) {
	req := &pb.SkyeBulkEmbeddingRequest{
		Entity:       "entity1",
		ModelName:    "model1",
		Variant:      "variant1",
		CandidateIds: []string{"id1", "id2"},
	}
	keys := GetCacheKeysForBulkEmbeddingRequest(req)
	assert.Len(t, keys, 2)
	for _, id := range req.CandidateIds {
		found := false
		for _, cs := range keys {
			if cs.CandidateId == id {
				found = true
				assert.NotEmpty(t, cs.Index)
				break
			}
		}
		assert.True(t, found, "expected candidate %s", id)
	}
}

func TestGetCacheKeysForCandidateDotProductScoreRequest(t *testing.T) {
	req := &pb.EmbeddingDotProductRequest{
		Entity:       "entity1",
		ModelName:    "model1",
		Variant:      "variant1",
		CandidateIds: []string{"id1"},
	}
	keys := GetCacheKeysForCandidateDotProductScoreRequest(req, 1)
	assert.Len(t, keys, 1)
	for k, cs := range keys {
		assert.NotEmpty(t, k)
		assert.Equal(t, "id1", cs.CandidateId)
		assert.Equal(t, []int{0}, cs.Index)
	}
}

func TestBuildBulkEmbeddingCacheKey(t *testing.T) {
	key := buildBulkEmbeddingCacheKey(Embedding, "e", "m", "id1")
	assert.Contains(t, key, Embedding)
	assert.Contains(t, key, "e")
	assert.Contains(t, key, "m")
	assert.Contains(t, key, "id1")
	assert.Contains(t, key, CacheVersion)
}

func TestBuildSimpleCacheKey(t *testing.T) {
	key := buildSimpleCacheKey(DotProduct, "e", "m", "1", "id1")
	assert.Contains(t, key, DotProduct)
	assert.Contains(t, key, "e")
	assert.Contains(t, key, "m")
	assert.Contains(t, key, "1")
	assert.Contains(t, key, "id1")
	assert.Contains(t, key, CacheVersion)
}
