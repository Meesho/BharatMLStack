package vector

import (
	"testing"

	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
)

// ============================================================
// getCollectionName
// ============================================================

func TestGetCollectionName(t *testing.T) {
	assert.Equal(t, "v1_m1_1", getCollectionName("v1", "m1", "1"))
	assert.Equal(t, "variant_model_10", getCollectionName("variant", "model", "10"))
	assert.Equal(t, "__", getCollectionName("", "", ""))
}

// ============================================================
// convertToDistance
// ============================================================

func TestConvertToDistance(t *testing.T) {
	tests := []struct {
		input    string
		expected qdrant.Distance
	}{
		{"COSINE", qdrant.Distance_Cosine},
		{"EUCLIDEAN", qdrant.Distance_Euclid},
		{"DOT", qdrant.Distance_Dot},
		{"MANHATTAN", qdrant.Distance_Manhattan},
		{"UNKNOWN", qdrant.Distance_UnknownDistance},
		{"", qdrant.Distance_UnknownDistance},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, convertToDistance(tt.input))
		})
	}
}

// ============================================================
// GetFieldIndexType
// ============================================================

func TestGetFieldIndexType(t *testing.T) {
	tests := []struct {
		input    string
		expected qdrant.FieldType
	}{
		{"keyword", qdrant.FieldType_FieldTypeKeyword},
		{"KEYWORD", qdrant.FieldType_FieldTypeKeyword},
		{"Keyword", qdrant.FieldType_FieldTypeKeyword},
		{"boolean", qdrant.FieldType_FieldTypeBool},
		{"BOOLEAN", qdrant.FieldType_FieldTypeBool},
		{"integer", qdrant.FieldType_FieldTypeInteger},
		{"INTEGER", qdrant.FieldType_FieldTypeInteger},
		{"float", qdrant.FieldType_FieldTypeFloat},
		{"FLOAT", qdrant.FieldType_FieldTypeFloat},
		{"datetime", qdrant.FieldType_FieldTypeDatetime},
		{"DATETIME", qdrant.FieldType_FieldTypeDatetime},
		{"geo", qdrant.FieldType_FieldTypeGeo},
		{"GEO", qdrant.FieldType_FieldTypeGeo},
		{"text", qdrant.FieldType_FieldTypeText},
		{"TEXT", qdrant.FieldType_FieldTypeText},
		{"unknown", qdrant.FieldType_FieldTypeKeyword}, // default
		{"", qdrant.FieldType_FieldTypeKeyword},        // default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetFieldIndexType(tt.input))
		})
	}
}

// ============================================================
// adaptToStatus
// ============================================================

func TestAdaptToStatus(t *testing.T) {
	tests := []struct {
		input    qdrant.CollectionStatus
		expected string
	}{
		{qdrant.CollectionStatus_Green, "GREEN"},
		{qdrant.CollectionStatus_Yellow, "YELLOW"},
		{qdrant.CollectionStatus_Red, "RED"},
		{qdrant.CollectionStatus_Grey, "GREY"},
		{qdrant.CollectionStatus(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, adaptToStatus(tt.input))
		})
	}
}

// ============================================================
// getInt64FromStringValue
// ============================================================

func TestGetInt64FromStringValue(t *testing.T) {
	assert.Equal(t, uint64(0), getInt64FromStringValue("0"))
	assert.Equal(t, uint64(42), getInt64FromStringValue("42"))
	assert.Equal(t, uint64(999999), getInt64FromStringValue("999999"))
}

// ============================================================
// getInt32FromString
// ============================================================

func TestGetInt32FromString(t *testing.T) {
	params := map[string]string{"shard_number": "4", "replication": "3"}
	result := getInt32FromString(params, "shard_number")
	assert.Equal(t, uint32(4), *result)
	result = getInt32FromString(params, "replication")
	assert.Equal(t, uint32(3), *result)
}

// ============================================================
// getInt64FromString
// ============================================================

func TestGetInt64FromString(t *testing.T) {
	params := map[string]string{"segment_number": "8", "max_size": "204800"}
	result := getInt64FromString(params, "segment_number")
	assert.Equal(t, uint64(8), *result)
	result = getInt64FromString(params, "max_size")
	assert.Equal(t, uint64(204800), *result)
}

// ============================================================
// getBoolFromString
// ============================================================

func TestGetBoolFromString(t *testing.T) {
	params := map[string]string{"on_disk": "true", "disabled": "false"}
	result := getBoolFromString(params, "on_disk")
	assert.True(t, *result)
	result = getBoolFromString(params, "disabled")
	assert.False(t, *result)
}

// ============================================================
// adaptToPayloadValue — exhaustive type coverage
// ============================================================

func TestAdaptToPayloadValue_SingleString(t *testing.T) {
	v := adaptToPayloadValue("hello")
	assert.Equal(t, "hello", v.GetStringValue())
}

func TestAdaptToPayloadValue_CommaSeparatedString(t *testing.T) {
	v := adaptToPayloadValue("a,b,c")
	list := v.GetListValue()
	assert.NotNil(t, list)
	assert.Len(t, list.Values, 3)
	assert.Equal(t, "a", list.Values[0].GetStringValue())
	assert.Equal(t, "b", list.Values[1].GetStringValue())
	assert.Equal(t, "c", list.Values[2].GetStringValue())
}

func TestAdaptToPayloadValue_Int(t *testing.T) {
	v := adaptToPayloadValue(42)
	assert.Equal(t, int64(42), v.GetIntegerValue())
}

func TestAdaptToPayloadValue_Int64(t *testing.T) {
	v := adaptToPayloadValue(int64(999))
	assert.Equal(t, int64(999), v.GetIntegerValue())
}

func TestAdaptToPayloadValue_Float64(t *testing.T) {
	v := adaptToPayloadValue(float64(3.14))
	assert.InDelta(t, 3.14, v.GetDoubleValue(), 0.001)
}

func TestAdaptToPayloadValue_Bool(t *testing.T) {
	v := adaptToPayloadValue(true)
	assert.True(t, v.GetBoolValue())
	v = adaptToPayloadValue(false)
	assert.False(t, v.GetBoolValue())
}

func TestAdaptToPayloadValue_IntSlice(t *testing.T) {
	v := adaptToPayloadValue([]int{1, 2, 3})
	list := v.GetListValue()
	assert.NotNil(t, list)
	assert.Len(t, list.Values, 3)
	assert.Equal(t, int64(1), list.Values[0].GetIntegerValue())
	assert.Equal(t, int64(2), list.Values[1].GetIntegerValue())
	assert.Equal(t, int64(3), list.Values[2].GetIntegerValue())
}

func TestAdaptToPayloadValue_StringSlice(t *testing.T) {
	v := adaptToPayloadValue([]string{"x", "y"})
	list := v.GetListValue()
	assert.NotNil(t, list)
	assert.Len(t, list.Values, 2)
	assert.Equal(t, "x", list.Values[0].GetStringValue())
	assert.Equal(t, "y", list.Values[1].GetStringValue())
}

func TestAdaptToPayloadValue_BoolSlice(t *testing.T) {
	v := adaptToPayloadValue([]bool{true, false})
	list := v.GetListValue()
	assert.NotNil(t, list)
	assert.Len(t, list.Values, 2)
	assert.True(t, list.Values[0].GetBoolValue())
	assert.False(t, list.Values[1].GetBoolValue())
}

func TestAdaptToPayloadValue_Default_Nil(t *testing.T) {
	v := adaptToPayloadValue(struct{}{})
	assert.NotNil(t, v)
	assert.NotNil(t, v.GetNullValue())
}

func TestAdaptToPayloadValue_Float32_ReturnsNil(t *testing.T) {
	// float32 case falls through without return in source code
	v := adaptToPayloadValue(float32(1.5))
	assert.Nil(t, v)
}

// ============================================================
// parseBatchResponse
// ============================================================

func TestParseBatchResponse_SingleResult(t *testing.T) {
	batchResult := []*qdrant.BatchResult{
		{
			Result: []*qdrant.ScoredPoint{
				{
					Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: 10}},
					Score:   0.95,
					Payload: map[string]*qdrant.Value{"name": {Kind: &qdrant.Value_StringValue{StringValue: "item1"}}},
				},
				{
					Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: 20}},
					Score:   0.85,
					Payload: map[string]*qdrant.Value{"name": {Kind: &qdrant.Value_StringValue{StringValue: "item2"}}},
				},
			},
		},
	}
	requestList := []*QueryDetails{{CacheKey: "key1"}}
	bulkRequest := &BatchQueryRequest{Entity: "e1", Model: "m1", Variant: "v1"}

	result := parseBatchResponse(batchResult, requestList, bulkRequest)
	assert.NotNil(t, result)
	assert.Len(t, result.SimilarCandidatesList["key1"], 2)
	assert.Equal(t, "10", result.SimilarCandidatesList["key1"][0].Id)
	assert.Equal(t, float32(0.95), result.SimilarCandidatesList["key1"][0].Score)
	assert.Equal(t, "item1", result.SimilarCandidatesList["key1"][0].Payload["name"])
	assert.Equal(t, "20", result.SimilarCandidatesList["key1"][1].Id)
}

func TestParseBatchResponse_EmptyResult(t *testing.T) {
	batchResult := []*qdrant.BatchResult{
		{Result: []*qdrant.ScoredPoint{}},
	}
	requestList := []*QueryDetails{{CacheKey: "key1"}}
	bulkRequest := &BatchQueryRequest{Entity: "e1", Model: "m1", Variant: "v1"}

	result := parseBatchResponse(batchResult, requestList, bulkRequest)
	assert.NotNil(t, result)
	assert.Len(t, result.SimilarCandidatesList["key1"], 0)
}

func TestParseBatchResponse_MultipleQueries(t *testing.T) {
	batchResult := []*qdrant.BatchResult{
		{
			Result: []*qdrant.ScoredPoint{
				{
					Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: 1}},
					Score:   0.9,
					Payload: map[string]*qdrant.Value{},
				},
			},
		},
		{
			Result: []*qdrant.ScoredPoint{
				{
					Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: 2}},
					Score:   0.8,
					Payload: map[string]*qdrant.Value{},
				},
			},
		},
	}
	requestList := []*QueryDetails{{CacheKey: "q1"}, {CacheKey: "q2"}}
	bulkRequest := &BatchQueryRequest{Entity: "e1", Model: "m1", Variant: "v1"}

	result := parseBatchResponse(batchResult, requestList, bulkRequest)
	assert.Len(t, result.SimilarCandidatesList["q1"], 1)
	assert.Len(t, result.SimilarCandidatesList["q2"], 1)
	assert.Equal(t, "1", result.SimilarCandidatesList["q1"][0].Id)
	assert.Equal(t, "2", result.SimilarCandidatesList["q2"][0].Id)
}
