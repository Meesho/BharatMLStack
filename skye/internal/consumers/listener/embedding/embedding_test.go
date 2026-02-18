package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdaptToPayloadValue_Keyword(t *testing.T) {
	val, err := adaptToPayloadValue("hello", "keyword")
	assert.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestAdaptToPayloadValue_KeywordArray(t *testing.T) {
	val, err := adaptToPayloadValue(`["a","b"]`, "keyword")
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, val)
}

func TestAdaptToPayloadValue_Integer(t *testing.T) {
	val, err := adaptToPayloadValue("42", "integer")
	assert.NoError(t, err)
	assert.Equal(t, 42, val)
}

func TestAdaptToPayloadValue_IntegerArray(t *testing.T) {
	val, err := adaptToPayloadValue("[1,2,3]", "integer")
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, val)
}

func TestAdaptToPayloadValue_Boolean(t *testing.T) {
	val, err := adaptToPayloadValue("true", "boolean")
	assert.NoError(t, err)
	assert.Equal(t, true, val)
}

func TestAdaptToPayloadValue_BooleanArray(t *testing.T) {
	val, err := adaptToPayloadValue("[true,false]", "boolean")
	assert.NoError(t, err)
	assert.Equal(t, []bool{true, false}, val)
}

func TestAdaptToPayloadValue_EmptyKeyword(t *testing.T) {
	val, err := adaptToPayloadValue("", "keyword")
	assert.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestAdaptToPayloadValue_EmptyInteger(t *testing.T) {
	val, err := adaptToPayloadValue("", "integer")
	assert.NoError(t, err)
	assert.Equal(t, 0, val)
}

func TestAdaptToPayloadValue_EmptyBoolean(t *testing.T) {
	val, err := adaptToPayloadValue("", "boolean")
	assert.NoError(t, err)
	assert.Equal(t, false, val)
}

func TestAdaptToPayloadValue_UnsupportedSchema(t *testing.T) {
	_, err := adaptToPayloadValue("value", "float")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported field_schema")
}

func TestAdaptToPayloadValue_EmptyUnsupportedSchema(t *testing.T) {
	_, err := adaptToPayloadValue("", "float")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported field_schema for empty value")
}

func TestAdaptToPayloadValue_InvalidInteger(t *testing.T) {
	_, err := adaptToPayloadValue("abc", "integer")
	assert.Error(t, err)
}

func TestAdaptToPayloadValue_InvalidBoolean(t *testing.T) {
	_, err := adaptToPayloadValue("notbool", "boolean")
	assert.Error(t, err)
}

func TestMapEventToEmbeddingStorePayload_WithSearchEmbedding(t *testing.T) {
	e := &EmbeddingConsumer{}
	event := Event{
		CandidateId:           "c1",
		Model:                 "m1",
		EmbeddingStoreVersion: 2,
		IndexSpace: IndexSpace{
			Embedding:        []float32{0.1, 0.2},
			VariantsIndexMap: map[string]bool{"v1": true},
		},
		SearchSpace: SearchSpace{
			Embedding: []float32{0.5, 0.6},
		},
	}
	payload, err := e.mapEventToEmbeddingStorePayload(event)
	assert.NoError(t, err)
	assert.Equal(t, "c1", payload.CandidateId)
	assert.Equal(t, "m1", payload.Model)
	assert.Equal(t, 2, payload.Version)
	assert.Equal(t, []float32{0.1, 0.2}, payload.Embedding)
	assert.Equal(t, []float32{0.5, 0.6}, payload.SearchEmbedding)
	assert.Equal(t, map[string]bool{"v1": true}, payload.VariantsIndexMap)
}

func TestMapEventToEmbeddingStorePayload_NoSearchEmbedding(t *testing.T) {
	e := &EmbeddingConsumer{}
	event := Event{
		CandidateId:           "c2",
		Model:                 "m2",
		EmbeddingStoreVersion: 1,
		IndexSpace: IndexSpace{
			Embedding:        []float32{0.3, 0.4},
			VariantsIndexMap: map[string]bool{"v1": false},
		},
	}
	payload, err := e.mapEventToEmbeddingStorePayload(event)
	assert.NoError(t, err)
	// When SearchSpace.Embedding is nil, falls back to IndexSpace.Embedding
	assert.Equal(t, []float32{0.3, 0.4}, payload.SearchEmbedding)
}
