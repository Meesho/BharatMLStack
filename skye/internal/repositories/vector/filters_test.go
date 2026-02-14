package vector

import (
	"testing"

	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
)

// ============================================================
// validateFilter
// ============================================================

func TestValidateFilter_IN_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_IN, Field: "f1", Values: []string{"a"}})
	assert.NoError(t, err)
}

func TestValidateFilter_IN_NoValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_IN, Field: "f1", Values: []string{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least one value")
}

func TestValidateFilter_NIN_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_NIN, Field: "f1", Values: []string{"a"}})
	assert.NoError(t, err)
}

func TestValidateFilter_NIN_NoValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_NIN, Field: "f1", Values: []string{}})
	assert.Error(t, err)
}

func TestValidateFilter_LTE_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_LTE, Field: "f1", Values: []string{"10"}})
	assert.NoError(t, err)
}

func TestValidateFilter_LTE_NoValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_LTE, Field: "f1", Values: []string{}})
	assert.Error(t, err)
}

func TestValidateFilter_GTE_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_GTE, Field: "f1", Values: []string{"5"}})
	assert.NoError(t, err)
}

func TestValidateFilter_LT_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_LT, Field: "f1", Values: []string{"10"}})
	assert.NoError(t, err)
}

func TestValidateFilter_GT_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_GT, Field: "f1", Values: []string{"5"}})
	assert.NoError(t, err)
}

func TestValidateFilter_BTW_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_BTW, Field: "f1", Values: []string{"1", "10"}})
	assert.NoError(t, err)
}

func TestValidateFilter_BTW_TooFewValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_BTW, Field: "f1", Values: []string{"1"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least two values")
}

func TestValidateFilter_BTWE_WithValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_BTWE, Field: "f1", Values: []string{"1", "10"}})
	assert.NoError(t, err)
}

func TestValidateFilter_BTWE_TooFewValues(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_BTWE, Field: "f1", Values: []string{}})
	assert.Error(t, err)
}

func TestValidateFilter_EX(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_EX, Field: "f1", Values: []string{}})
	assert.NoError(t, err)
}

func TestValidateFilter_UnsupportedOperator(t *testing.T) {
	err := validateFilter(&pb.Filter{Op: pb.Filter_Operator(999), Field: "f1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

// ============================================================
// convertFilterValuesBySchema
// ============================================================

func TestConvertFilterValuesBySchema_Integer(t *testing.T) {
	result, err := convertFilterValuesBySchema([]string{"1", "2", "3"}, "integer")
	assert.NoError(t, err)
	intValues := result.([]int64)
	assert.Equal(t, []int64{1, 2, 3}, intValues)
}

func TestConvertFilterValuesBySchema_Integer_Error(t *testing.T) {
	_, err := convertFilterValuesBySchema([]string{"abc"}, "integer")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse integer")
}

func TestConvertFilterValuesBySchema_Float(t *testing.T) {
	result, err := convertFilterValuesBySchema([]string{"1.5", "2.7"}, "float")
	assert.NoError(t, err)
	floatValues := result.([]float64)
	assert.InDelta(t, 1.5, floatValues[0], 0.001)
	assert.InDelta(t, 2.7, floatValues[1], 0.001)
}

func TestConvertFilterValuesBySchema_Float_Error(t *testing.T) {
	_, err := convertFilterValuesBySchema([]string{"notfloat"}, "float")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse float")
}

func TestConvertFilterValuesBySchema_Keyword(t *testing.T) {
	result, err := convertFilterValuesBySchema([]string{"a", "b"}, "keyword")
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestConvertFilterValuesBySchema_Default(t *testing.T) {
	result, err := convertFilterValuesBySchema([]string{"a"}, "unknown_type")
	assert.NoError(t, err)
	assert.Equal(t, []string{"a"}, result)
}

func TestConvertFilterValuesBySchema_CaseInsensitive(t *testing.T) {
	result, err := convertFilterValuesBySchema([]string{"10"}, "INTEGER")
	assert.NoError(t, err)
	intValues := result.([]int64)
	assert.Equal(t, []int64{10}, intValues)
}

// ============================================================
// buildFilterCondition
// ============================================================

func TestBuildFilterCondition_StringSlice(t *testing.T) {
	fc, err := buildFilterCondition(pb.Filter_IN, "category", []string{"a", "b"})
	assert.NoError(t, err)
	assert.NotNil(t, fc)
	assert.False(t, fc.IsNegated)
}

func TestBuildFilterCondition_Int64Slice(t *testing.T) {
	fc, err := buildFilterCondition(pb.Filter_IN, "price", []int64{10, 20})
	assert.NoError(t, err)
	assert.NotNil(t, fc)
	assert.False(t, fc.IsNegated)
}

func TestBuildFilterCondition_Float64Slice(t *testing.T) {
	fc, err := buildFilterCondition(pb.Filter_GTE, "score", []float64{0.5})
	assert.NoError(t, err)
	assert.NotNil(t, fc)
}

func TestBuildFilterCondition_UnsupportedType(t *testing.T) {
	_, err := buildFilterCondition(pb.Filter_IN, "f1", 42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected value type")
}

// ============================================================
// buildStringFilterCondition
// ============================================================

func TestBuildStringFilterCondition_IN(t *testing.T) {
	fc, err := buildStringFilterCondition(pb.Filter_IN, "tag", []string{"a", "b"})
	assert.NoError(t, err)
	assert.False(t, fc.IsNegated)
	match := fc.Condition.GetField().Match
	assert.Equal(t, []string{"a", "b"}, match.GetKeywords().Strings)
}

func TestBuildStringFilterCondition_NIN(t *testing.T) {
	fc, err := buildStringFilterCondition(pb.Filter_NIN, "tag", []string{"x"})
	assert.NoError(t, err)
	assert.True(t, fc.IsNegated)
}

func TestBuildStringFilterCondition_EX(t *testing.T) {
	fc, err := buildStringFilterCondition(pb.Filter_EX, "tag", []string{})
	assert.NoError(t, err)
	assert.True(t, fc.IsNegated)
	assert.NotNil(t, fc.Condition.GetIsNull())
}

func TestBuildStringFilterCondition_Unsupported(t *testing.T) {
	_, err := buildStringFilterCondition(pb.Filter_LTE, "tag", []string{"a"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

// ============================================================
// buildIntFilterCondition
// ============================================================

func TestBuildIntFilterCondition_IN(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_IN, "count", []int64{1, 2})
	assert.NoError(t, err)
	assert.False(t, fc.IsNegated)
	assert.Equal(t, []int64{1, 2}, fc.Condition.GetField().Match.GetIntegers().Integers)
}

func TestBuildIntFilterCondition_NIN(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_NIN, "count", []int64{5})
	assert.NoError(t, err)
	assert.True(t, fc.IsNegated)
}

func TestBuildIntFilterCondition_EX(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_EX, "count", []int64{})
	assert.NoError(t, err)
	assert.True(t, fc.IsNegated)
	assert.NotNil(t, fc.Condition.GetIsNull())
}

func TestBuildIntFilterCondition_LTE(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_LTE, "price", []int64{100})
	assert.NoError(t, err)
	assert.False(t, fc.IsNegated)
	assert.InDelta(t, float64(100), *fc.Condition.GetField().Range.Lte, 0.001)
}

func TestBuildIntFilterCondition_GTE(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_GTE, "price", []int64{10})
	assert.NoError(t, err)
	assert.InDelta(t, float64(10), *fc.Condition.GetField().Range.Gte, 0.001)
}

func TestBuildIntFilterCondition_LT(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_LT, "price", []int64{50})
	assert.NoError(t, err)
	assert.InDelta(t, float64(50), *fc.Condition.GetField().Range.Lt, 0.001)
}

func TestBuildIntFilterCondition_GT(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_GT, "price", []int64{5})
	assert.NoError(t, err)
	assert.InDelta(t, float64(5), *fc.Condition.GetField().Range.Gt, 0.001)
}

func TestBuildIntFilterCondition_BTW(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_BTW, "price", []int64{10, 100})
	assert.NoError(t, err)
	assert.InDelta(t, float64(10), *fc.Condition.GetField().Range.Gt, 0.001)
	assert.InDelta(t, float64(100), *fc.Condition.GetField().Range.Lt, 0.001)
}

func TestBuildIntFilterCondition_BTWE(t *testing.T) {
	fc, err := buildIntFilterCondition(pb.Filter_BTWE, "price", []int64{10, 100})
	assert.NoError(t, err)
	assert.InDelta(t, float64(10), *fc.Condition.GetField().Range.Gte, 0.001)
	assert.InDelta(t, float64(100), *fc.Condition.GetField().Range.Lte, 0.001)
}

func TestBuildIntFilterCondition_Unsupported(t *testing.T) {
	_, err := buildIntFilterCondition(pb.Filter_Operator(999), "f", []int64{1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

// ============================================================
// buildFloatFilterCondition
// ============================================================

func TestBuildFloatFilterCondition_EX(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_EX, "score", []float64{})
	assert.NoError(t, err)
	assert.True(t, fc.IsNegated)
}

func TestBuildFloatFilterCondition_LTE(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_LTE, "score", []float64{9.5})
	assert.NoError(t, err)
	assert.InDelta(t, 9.5, *fc.Condition.GetField().Range.Lte, 0.001)
}

func TestBuildFloatFilterCondition_GTE(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_GTE, "score", []float64{1.0})
	assert.NoError(t, err)
	assert.InDelta(t, 1.0, *fc.Condition.GetField().Range.Gte, 0.001)
}

func TestBuildFloatFilterCondition_LT(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_LT, "score", []float64{5.0})
	assert.NoError(t, err)
	assert.InDelta(t, 5.0, *fc.Condition.GetField().Range.Lt, 0.001)
}

func TestBuildFloatFilterCondition_GT(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_GT, "score", []float64{0.5})
	assert.NoError(t, err)
	assert.InDelta(t, 0.5, *fc.Condition.GetField().Range.Gt, 0.001)
}

func TestBuildFloatFilterCondition_BTW(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_BTW, "score", []float64{0.1, 0.9})
	assert.NoError(t, err)
	assert.InDelta(t, 0.1, *fc.Condition.GetField().Range.Gt, 0.001)
	assert.InDelta(t, 0.9, *fc.Condition.GetField().Range.Lt, 0.001)
}

func TestBuildFloatFilterCondition_BTWE(t *testing.T) {
	fc, err := buildFloatFilterCondition(pb.Filter_BTWE, "score", []float64{0.1, 0.9})
	assert.NoError(t, err)
	assert.InDelta(t, 0.1, *fc.Condition.GetField().Range.Gte, 0.001)
	assert.InDelta(t, 0.9, *fc.Condition.GetField().Range.Lte, 0.001)
}

func TestBuildFloatFilterCondition_Unsupported(t *testing.T) {
	_, err := buildFloatFilterCondition(pb.Filter_IN, "score", []float64{1.0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

// ============================================================
// buildMatchFilterCondition
// ============================================================

func TestBuildMatchFilterCondition(t *testing.T) {
	match := &qdrant.Match{MatchValue: &qdrant.Match_Keywords{
		Keywords: &qdrant.RepeatedStrings{Strings: []string{"a"}},
	}}
	fc := buildMatchFilterCondition("tag", match, false)
	assert.NotNil(t, fc)
	assert.False(t, fc.IsNegated)
	assert.Equal(t, "tag", fc.Condition.GetField().Key)
}

func TestBuildMatchFilterCondition_Negated(t *testing.T) {
	match := &qdrant.Match{}
	fc := buildMatchFilterCondition("tag", match, true)
	assert.True(t, fc.IsNegated)
}

// ============================================================
// buildRangeFilterCondition
// ============================================================

func TestBuildRangeFilterCondition(t *testing.T) {
	v := float64(42)
	fc := buildRangeFilterCondition("price", &qdrant.Range{Gte: &v})
	assert.NotNil(t, fc)
	assert.False(t, fc.IsNegated)
	assert.Equal(t, "price", fc.Condition.GetField().Key)
	assert.InDelta(t, 42.0, *fc.Condition.GetField().Range.Gte, 0.001)
}

// ============================================================
// buildIsNullFilterCondition
// ============================================================

func TestBuildIsNullFilterCondition(t *testing.T) {
	fc := buildIsNullFilterCondition("field1", true)
	assert.True(t, fc.IsNegated)
	assert.Equal(t, "field1", fc.Condition.GetIsNull().Key)
}

func TestBuildIsNullFilterCondition_NotNegated(t *testing.T) {
	fc := buildIsNullFilterCondition("field1", false)
	assert.False(t, fc.IsNegated)
}

// ============================================================
// parseFiltersToQdrantFilters — integration
// ============================================================

func TestParseFiltersToQdrantFilters_NoFilters(t *testing.T) {
	request := &QueryDetails{MetadataFilters: []*pb.Filter{}}
	filter, err := parseFiltersToQdrantFilters(request, map[string]config.Payload{})
	assert.NoError(t, err)
	assert.Nil(t, filter.Must)
	assert.Nil(t, filter.MustNot)
}

func TestParseFiltersToQdrantFilters_MustAndMustNot(t *testing.T) {
	payloadSchema := map[string]config.Payload{
		"category": {FieldSchema: "keyword"},
		"brand":    {FieldSchema: "keyword"},
	}
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_IN, Field: "category", Values: []string{"shoes"}},
			{Op: pb.Filter_NIN, Field: "brand", Values: []string{"nike"}},
		},
	}
	filter, err := parseFiltersToQdrantFilters(request, payloadSchema)
	assert.NoError(t, err)
	assert.Len(t, filter.Must, 1)
	assert.Len(t, filter.MustNot, 1)
}

func TestParseFiltersToQdrantFilters_FieldNotInSchema(t *testing.T) {
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_IN, Field: "missing_field", Values: []string{"x"}},
		},
	}
	_, err := parseFiltersToQdrantFilters(request, map[string]config.Payload{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in payload schema")
}

func TestParseFiltersToQdrantFilters_ValidationError(t *testing.T) {
	payloadSchema := map[string]config.Payload{
		"f1": {FieldSchema: "keyword"},
	}
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_IN, Field: "f1", Values: []string{}}, // IN with no values
		},
	}
	_, err := parseFiltersToQdrantFilters(request, payloadSchema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate filter")
}

func TestParseFiltersToQdrantFilters_ConversionError(t *testing.T) {
	payloadSchema := map[string]config.Payload{
		"price": {FieldSchema: "integer"},
	}
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_IN, Field: "price", Values: []string{"not_a_number"}},
		},
	}
	_, err := parseFiltersToQdrantFilters(request, payloadSchema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert filter values")
}

func TestParseFiltersToQdrantFilters_IntRange(t *testing.T) {
	payloadSchema := map[string]config.Payload{
		"price": {FieldSchema: "integer"},
	}
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_GTE, Field: "price", Values: []string{"100"}},
		},
	}
	filter, err := parseFiltersToQdrantFilters(request, payloadSchema)
	assert.NoError(t, err)
	assert.Len(t, filter.Must, 1)
}

func TestParseFiltersToQdrantFilters_FloatRange(t *testing.T) {
	payloadSchema := map[string]config.Payload{
		"score": {FieldSchema: "float"},
	}
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_BTW, Field: "score", Values: []string{"0.1", "0.9"}},
		},
	}
	filter, err := parseFiltersToQdrantFilters(request, payloadSchema)
	assert.NoError(t, err)
	assert.Len(t, filter.Must, 1)
}

func TestParseFiltersToQdrantFilters_EXFilter(t *testing.T) {
	payloadSchema := map[string]config.Payload{
		"optional": {FieldSchema: "keyword"},
	}
	request := &QueryDetails{
		MetadataFilters: []*pb.Filter{
			{Op: pb.Filter_EX, Field: "optional", Values: []string{}},
		},
	}
	filter, err := parseFiltersToQdrantFilters(request, payloadSchema)
	assert.NoError(t, err)
	// EX on string → IsNull negated → goes to MustNot
	assert.Len(t, filter.MustNot, 1)
}
