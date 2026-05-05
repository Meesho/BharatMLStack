package vector

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/qdrant/go-client/qdrant"
	"github.com/rs/zerolog/log"
)

func buildMatchFilterCondition(field string, match *qdrant.Match, IsNegated bool) *FilterCondition {
	return &FilterCondition{
		Condition: &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   field,
					Match: match,
				},
			},
		},
		IsNegated: IsNegated,
	}
}

func buildRangeFilterCondition(field string, r *qdrant.Range) *FilterCondition {
	return &FilterCondition{
		Condition: &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   field,
					Range: r,
				},
			},
		},
		IsNegated: false,
	}
}

func buildIsNullFilterCondition(field string, IsNegated bool) *FilterCondition {
	return &FilterCondition{
		Condition: &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_IsNull{
				IsNull: &qdrant.IsNullCondition{
					Key: field,
				},
			},
		},
		IsNegated: IsNegated,
	}
}

func buildFilterCondition(operator pb.Filter_Operator, field string, value interface{}) (*FilterCondition, error) {
	log.Info().Msgf("Building filter condition for field %s with operator %v and value %v", field, operator, value)
	switch v := value.(type) {
	case []string:
		return buildStringFilterCondition(operator, field, v)
	case []int64:
		return buildIntFilterCondition(operator, field, v)
	case []float64:
		return buildFloatFilterCondition(operator, field, v)
	default:
		return nil, fmt.Errorf("unexpected value type in filter: %T for field %s", value, field)
	}
}

func buildStringFilterCondition(operator pb.Filter_Operator, field string, values []string) (*FilterCondition, error) {
	switch operator {
	case pb.Filter_IN:
		return buildMatchFilterCondition(
			field,
			&qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: values}}},
			false,
		), nil
	case pb.Filter_NIN:
		return buildMatchFilterCondition(
			field,
			&qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: values}}},
			true,
		), nil
	case pb.Filter_EX:
		return buildIsNullFilterCondition(field, true), nil
	default:
		return nil, fmt.Errorf("unsupported operator %v for string type on field %s", operator, field)
	}
}

func buildIntFilterCondition(operator pb.Filter_Operator, field string, values []int64) (*FilterCondition, error) {
	switch operator {
	case pb.Filter_IN:
		return buildMatchFilterCondition(
			field,
			&qdrant.Match{MatchValue: &qdrant.Match_Integers{Integers: &qdrant.RepeatedIntegers{Integers: values}}},
			false,
		), nil
	case pb.Filter_NIN:
		return buildMatchFilterCondition(
			field,
			&qdrant.Match{MatchValue: &qdrant.Match_Integers{Integers: &qdrant.RepeatedIntegers{Integers: values}}},
			true,
		), nil
	case pb.Filter_EX:
		return buildIsNullFilterCondition(field, true), nil
	case pb.Filter_LTE:
		lteValue := float64(values[0])
		return buildRangeFilterCondition(field, &qdrant.Range{Lte: &lteValue}), nil
	case pb.Filter_GTE:
		gteValue := float64(values[0])
		return buildRangeFilterCondition(field, &qdrant.Range{Gte: &gteValue}), nil
	case pb.Filter_LT:
		ltValue := float64(values[0])
		return buildRangeFilterCondition(field, &qdrant.Range{Lt: &ltValue}), nil
	case pb.Filter_GT:
		gtValue := float64(values[0])
		return buildRangeFilterCondition(field, &qdrant.Range{Gt: &gtValue}), nil
	case pb.Filter_BTW:
		gtValue := float64(values[0])
		ltValue := float64(values[1])
		return buildRangeFilterCondition(field, &qdrant.Range{Gt: &gtValue, Lt: &ltValue}), nil
	case pb.Filter_BTWE:
		gteValue := float64(values[0])
		lteValue := float64(values[1])
		return buildRangeFilterCondition(field, &qdrant.Range{Gte: &gteValue, Lte: &lteValue}), nil
	default:
		return nil, fmt.Errorf("unsupported operator %v for int type on field %s", operator, field)
	}
}

func buildFloatFilterCondition(operator pb.Filter_Operator, field string, values []float64) (*FilterCondition, error) {
	switch operator {
	case pb.Filter_EX:
		return buildIsNullFilterCondition(field, true), nil
	case pb.Filter_LTE:
		lteValue := values[0]
		return buildRangeFilterCondition(field, &qdrant.Range{Lte: &lteValue}), nil
	case pb.Filter_GTE:
		gteValue := values[0]
		return buildRangeFilterCondition(field, &qdrant.Range{Gte: &gteValue}), nil
	case pb.Filter_LT:
		ltValue := values[0]
		return buildRangeFilterCondition(field, &qdrant.Range{Lt: &ltValue}), nil
	case pb.Filter_GT:
		gtValue := values[0]
		return buildRangeFilterCondition(field, &qdrant.Range{Gt: &gtValue}), nil
	case pb.Filter_BTW:
		gtValue := values[0]
		ltValue := values[1]
		return buildRangeFilterCondition(field, &qdrant.Range{Gt: &gtValue, Lt: &ltValue}), nil
	case pb.Filter_BTWE:
		gteValue := values[0]
		lteValue := values[1]
		return buildRangeFilterCondition(field, &qdrant.Range{Gte: &gteValue, Lte: &lteValue}), nil
	default:
		return nil, fmt.Errorf("unsupported operator %v for float type on field %s", operator, field)
	}
}

func convertFilterValuesBySchema(values []string, fieldSchema string) (interface{}, error) {
	normalizedSchema := strings.ToLower(fieldSchema)

	switch normalizedSchema {
	case "integer":
		intValues := make([]int64, 0, len(values))
		for _, val := range values {
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse integer value '%s' for field schema %s: %w", val, fieldSchema, err)
			}
			intValues = append(intValues, intVal)
		}
		return intValues, nil

	case "float":
		floatValues := make([]float64, 0, len(values))
		for _, val := range values {
			floatVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse float value '%s' for field schema %s: %w", val, fieldSchema, err)
			}
			floatValues = append(floatValues, floatVal)
		}
		return floatValues, nil

	case "keyword":
		return values, nil

	default:
		return values, nil
	}
}
