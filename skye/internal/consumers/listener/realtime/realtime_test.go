package realtime

import (
	"testing"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/handler/aggregator"
	"github.com/stretchr/testify/assert"
)

func TestShouldIndexDelta_AllMatch(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{
		"status": "active",
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.True(t, r.shouldIndexDelta(deltaData, criteria))
}

func TestShouldIndexDelta_Mismatch(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{
		"status": "inactive",
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.False(t, r.shouldIndexDelta(deltaData, criteria))
}

func TestShouldIndexDelta_NilColumn(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	// deltaData["status"] is nil so the condition is skipped -> shouldIndex stays true
	assert.True(t, r.shouldIndexDelta(deltaData, criteria))
}

func TestIsCriteriaPresent_Found(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{
		"status": "active",
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.True(t, r.isCriteriaPresent(deltaData, criteria))
}

func TestIsCriteriaPresent_NotFound(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{
		"category": "electronics",
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.False(t, r.isCriteriaPresent(deltaData, criteria))
}

func TestIsPayloadPresent_Found(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{
		"price": 100,
	}
	payload := map[string]config.Payload{
		"price": {FieldSchema: "integer"},
	}
	assert.True(t, r.isPayloadPresent(deltaData, payload))
}

func TestIsPayloadPresent_NotFound(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{
		"category": "electronics",
	}
	payload := map[string]config.Payload{
		"price": {FieldSchema: "integer"},
	}
	assert.False(t, r.isPayloadPresent(deltaData, payload))
}

func TestIsPayloadPresent_EmptyPayload(t *testing.T) {
	r := &RealtimeConsumer{}
	deltaData := map[string]interface{}{"price": 100}
	payload := map[string]config.Payload{}
	assert.False(t, r.isPayloadPresent(deltaData, payload))
}

func TestShouldIndexNonDelta_AllCriteriaMatch(t *testing.T) {
	r := &RealtimeConsumer{}
	resp := &aggregator.Response{
		DeltaData:    map[string]interface{}{"status": "active"},
		CompleteData: map[string]interface{}{"status": "active", "type": "premium"},
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.True(t, r.shouldIndexNonDelta(resp, criteria))
}

func TestShouldIndexNonDelta_DeltaMissingFallsBackToComplete(t *testing.T) {
	r := &RealtimeConsumer{}
	resp := &aggregator.Response{
		DeltaData:    map[string]interface{}{},
		CompleteData: map[string]interface{}{"status": "active"},
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.True(t, r.shouldIndexNonDelta(resp, criteria))
}

func TestShouldIndexNonDelta_CompleteMismatch(t *testing.T) {
	r := &RealtimeConsumer{}
	resp := &aggregator.Response{
		DeltaData:    map[string]interface{}{},
		CompleteData: map[string]interface{}{"status": "inactive"},
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.False(t, r.shouldIndexNonDelta(resp, criteria))
}

func TestShouldIndexNonDelta_CompleteNil(t *testing.T) {
	r := &RealtimeConsumer{}
	resp := &aggregator.Response{
		DeltaData:    map[string]interface{}{},
		CompleteData: map[string]interface{}{},
	}
	criteria := []config.Criteria{
		{ColumnName: "status", FilterValue: "active"},
	}
	assert.False(t, r.shouldIndexNonDelta(resp, criteria))
}

func TestCheckEmbeddingResponse_EmptyResponse(t *testing.T) {
	// Should not panic on empty response
	checkEmbeddingResponse(nil, "c1", "write", "e1", "m1", "v1", 1)
	checkEmbeddingResponse(map[string]map[string]any{}, "c1", "write", "e1", "m1", "v1", 1)
}

func TestCheckEmbeddingResponse_CandidateMissing(t *testing.T) {
	resp := map[string]map[string]any{
		"other": {"v1_to_be_indexed": true, "embedding": []float32{0.1}},
	}
	// Should not panic when candidate is missing
	checkEmbeddingResponse(resp, "c1", "write", "e1", "m1", "v1", 1)
}

func TestCheckEmbeddingResponse_CandidatePresent(t *testing.T) {
	resp := map[string]map[string]any{
		"c1": {"v1_to_be_indexed": true, "embedding": []float32{0.1, 0.2}},
	}
	// Should not panic when everything is valid
	checkEmbeddingResponse(resp, "c1", "write", "e1", "m1", "v1", 1)
}

func TestCheckEmbeddingResponse_NotToBeIndexed(t *testing.T) {
	resp := map[string]map[string]any{
		"c1": {"v1_to_be_indexed": false, "embedding": []float32{0.1}},
	}
	checkEmbeddingResponse(resp, "c1", "write", "e1", "m1", "v1", 1)
}

func TestCheckEmbeddingResponse_EmptyEmbedding(t *testing.T) {
	resp := map[string]map[string]any{
		"c1": {"v1_to_be_indexed": true, "embedding": []float32{}},
	}
	checkEmbeddingResponse(resp, "c1", "write", "e1", "m1", "v1", 1)
}
