package embedding

import (
	"testing"

	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/stretchr/testify/assert"
)

func TestConvertFloat32ListToFloat64(t *testing.T) {
	out := convertFloat32ListToFloat64([]float32{1.0, 2.0, 3.0})
	assert.Equal(t, []float64{1.0, 2.0, 3.0}, out)
	out = convertFloat32ListToFloat64(nil)
	assert.Empty(t, out)
}

func TestParseEmbeddingResponse(t *testing.T) {
	responseMap := make(map[string]repositories.CandidateResponseStruct)
	cacheKeys := map[string]repositories.CacheStruct{
		"k1": {
			Index:       []int{0},
			Embedding:   []float32{0.1, 0.2},
			CandidateId: "c1",
		},
	}
	parseEmbeddingResponse(responseMap, cacheKeys)
	assert.Len(t, responseMap, 1)
	assert.Contains(t, responseMap, "k1")
	assert.Equal(t, []int{0}, responseMap["k1"].Index)
	assert.Len(t, responseMap["k1"].EmbeddingResponse.Embedding, 2)
	assert.InDelta(t, 0.1, responseMap["k1"].EmbeddingResponse.Embedding[0], 1e-6)
	assert.InDelta(t, 0.2, responseMap["k1"].EmbeddingResponse.Embedding[1], 1e-6)
	assert.Equal(t, "c1", responseMap["k1"].EmbeddingResponse.Id)
}
