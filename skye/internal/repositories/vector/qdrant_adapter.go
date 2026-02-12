package vector

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/qdrant/go-client/qdrant"
	"github.com/rs/zerolog/log"
)

func getInt32FromString(params map[string]string, name string) *uint32 {
	intVar, err := strconv.Atoi(params[name])
	if err != nil {
		log.Fatal().Msgf("Cannot Convert string to int: %v", err)
	}
	uInt := uint32(intVar)
	return &uInt
}

func getInt64FromString(params map[string]string, name string) *uint64 {
	intVar, err := strconv.Atoi(params[name])
	if err != nil {
		log.Fatal().Msgf("Cannot Convert string to int: %v", err)
	}
	uInt := uint64(intVar)
	return &uInt
}

func getInt64FromStringValue(data string) uint64 {
	intVar, err := strconv.Atoi(data)
	if err != nil {
		log.Fatal().Msgf("Cannot Convert string to int: %v", err)
	}
	uInt := uint64(intVar)
	return uInt
}

func getBoolFromString(params map[string]string, name string) *bool {
	boolVar, err := strconv.ParseBool(params[name])
	if err != nil {
		log.Fatal().Msgf("Cannot Convert string to int: %v", err)
	}
	return &boolVar
}

func convertToDistance(s string) qdrant.Distance {
	switch s {
	case "COSINE":
		return qdrant.Distance_Cosine
	case "EUCLIDEAN":
		return qdrant.Distance_Euclid
	case "DOT":
		return qdrant.Distance_Dot
	case "MANHATTAN":
		return qdrant.Distance_Manhattan
	default:
		return qdrant.Distance_UnknownDistance
	}
}

func adaptToPayloadValue(value interface{}) *qdrant.Value {
	switch v := value.(type) {
	case string:
		vList := strings.Split(v, ",")
		if len(vList) == 1 {
			return &qdrant.Value{
				Kind: &qdrant.Value_StringValue{
					StringValue: v,
				},
			}
		} else {
			listValues := &qdrant.ListValue{}
			for _, item := range vList {
				listValues.Values = append(listValues.Values, &qdrant.Value{
					Kind: &qdrant.Value_StringValue{
						StringValue: item,
					},
				})
			}
			return &qdrant.Value{
				Kind: &qdrant.Value_ListValue{
					ListValue: listValues,
				},
			}
		}
	case int:
		return &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{
				IntegerValue: int64(v),
			},
		}

	case int64:
		return &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{
				IntegerValue: v,
			},
		}
	case float32:
	case float64:
		return &qdrant.Value{
			Kind: &qdrant.Value_DoubleValue{
				DoubleValue: v,
			},
		}
	case bool:
		return &qdrant.Value{
			Kind: &qdrant.Value_BoolValue{
				BoolValue: v,
			},
		}
	case []int:
		listValues := &qdrant.ListValue{}
		for _, item := range v {
			listValues.Values = append(listValues.Values, &qdrant.Value{
				Kind: &qdrant.Value_IntegerValue{
					IntegerValue: int64(item),
				},
			})
		}
		return &qdrant.Value{
			Kind: &qdrant.Value_ListValue{
				ListValue: listValues,
			},
		}

	case []string:
		listValues := &qdrant.ListValue{}
		for _, item := range v {
			listValues.Values = append(listValues.Values, &qdrant.Value{
				Kind: &qdrant.Value_StringValue{
					StringValue: item,
				},
			})
		}
		return &qdrant.Value{
			Kind: &qdrant.Value_ListValue{
				ListValue: listValues,
			},
		}
	case []bool:
		listValues := &qdrant.ListValue{}
		for _, item := range v {
			listValues.Values = append(listValues.Values, &qdrant.Value{
				Kind: &qdrant.Value_BoolValue{
					BoolValue: item,
				},
			})
		}
		return &qdrant.Value{
			Kind: &qdrant.Value_ListValue{
				ListValue: listValues,
			},
		}
	default:
		return &qdrant.Value{
			Kind: &qdrant.Value_NullValue{
				NullValue: 0,
			},
		}
	}
	return nil
}

func GetFieldIndexType(fieldIndexType string) qdrant.FieldType {
	fieldIndexType = strings.ToUpper(fieldIndexType)
	switch fieldIndexType {
	case "KEYWORD":
		return qdrant.FieldType_FieldTypeKeyword
	case "BOOLEAN":
		return qdrant.FieldType_FieldTypeBool
	case "INTEGER":
		return qdrant.FieldType_FieldTypeInteger
	case "FLOAT":
		return qdrant.FieldType_FieldTypeFloat
	case "DATETIME":
		return qdrant.FieldType_FieldTypeDatetime
	case "GEO":
		return qdrant.FieldType_FieldTypeGeo
	case "TEXT":
		return qdrant.FieldType_FieldTypeText
	default:
		return qdrant.FieldType_FieldTypeKeyword
	}
}

func adaptToStatus(status qdrant.CollectionStatus) string {
	switch status {
	case qdrant.CollectionStatus_Green:
		return "GREEN"
	case qdrant.CollectionStatus_Yellow:
		return "YELLOW"
	case qdrant.CollectionStatus_Red:
		return "RED"
	case qdrant.CollectionStatus_Grey:
		return "GREY"
	default:
		return "UNKNOWN"
	}
}

func getCollectionName(variant string, model string, version string) string {
	return variant + "_" + model + "_" + version
}

func parseFiltersToQdrantFilters(request *QueryDetails, payloadSchema map[string]config.Payload) (*qdrant.Filter, error) {
	mustConditions := make([]*qdrant.Condition, 0)
	mustNotConditions := make([]*qdrant.Condition, 0)

	for _, filter := range request.MetadataFilters {
		payloadInfo, exists := payloadSchema[filter.Field]
		if !exists {
			return nil, fmt.Errorf("field %s not found in payload schema", filter.Field)
		}
		fieldSchema := payloadInfo.FieldSchema

		err := validateFilter(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to validate filter for field %s with operator %v: %w", filter.Field, filter.Op, err)
		}

		convertedValue, err := convertFilterValuesBySchema(filter.Values, fieldSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter values for field %s: %w", filter.Field, err)
		}
		filterCondition, err := buildFilterCondition(filter.Op, filter.Field, convertedValue)
		if err != nil {
			return nil, fmt.Errorf("failed to build condition for field %s with operator %v: %w", filter.Field, filter.Op, err)
		}

		if filterCondition.IsNegated {
			mustNotConditions = append(mustNotConditions, filterCondition.Condition)
		} else {
			mustConditions = append(mustConditions, filterCondition.Condition)
		}
	}

	result := &qdrant.Filter{}
	if len(mustConditions) > 0 {
		result.Must = mustConditions
	}
	if len(mustNotConditions) > 0 {
		result.MustNot = mustNotConditions
	}

	return result, nil
}

func validateFilter(filter *pb.Filter) error {
	switch filter.Op {
	case pb.Filter_IN, pb.Filter_NIN, pb.Filter_LTE, pb.Filter_GTE, pb.Filter_LT, pb.Filter_GT:
		if len(filter.Values) == 0 {
			return fmt.Errorf("operator %v requires at least one value for field %s", filter.Op, filter.Field)
		}
	case pb.Filter_BTW, pb.Filter_BTWE:
		if len(filter.Values) < 2 {
			return fmt.Errorf("operator %v requires at least two values for field %s", filter.Op, filter.Field)
		}
	case pb.Filter_EX:
		// EX doesn't require values
	default:
		return fmt.Errorf("unsupported operator %v for field %s", filter.Op, filter.Field)
	}
	return nil
}

func parseBatchResponse(batchResult []*qdrant.BatchResult, requestList []*QueryDetails, bulkRequest *BatchQueryRequest) *BatchQueryResponse {
	var similarCandidatesList = make(map[string][]*SimilarCandidate, len(batchResult))

	for i := range batchResult {
		var similarCandidates = make([]*SimilarCandidate, len(batchResult[i].Result))
		totalScore := 0.0
		totalResults := 0
		for j := range batchResult[i].Result {
			payloadKeyValue := batchResult[i].Result[j].Payload
			keyValueString := make(map[string]string)
			for key, value := range payloadKeyValue {
				keyValueString[key] = value.GetStringValue()
			}
			similarCandidates[j] = &SimilarCandidate{
				Id:      strconv.FormatUint(batchResult[i].Result[j].Id.GetNum(), 10),
				Score:   batchResult[i].Result[j].Score,
				Payload: keyValueString,
			}
			totalScore += float64(batchResult[i].Result[j].Score)
			totalResults++
		}
		similarCandidatesList[requestList[i].CacheKey] = similarCandidates

		if totalResults > 0 {
			maxScore := batchResult[i].Result[0].Score
			minScore := batchResult[i].Result[len(batchResult[i].Result)-1].Score
			avgScore := totalScore / float64(totalResults)

			tags := []string{
				"vector_db_type", "qdrant",
				"entity", bulkRequest.Entity,
				"model_name", bulkRequest.Model,
				"variant", bulkRequest.Variant,
			}

			metric.Gauge("qdrant_query_similarity_mean_score", avgScore, tags)
			metric.Gauge("qdrant_query_similarity_max_score", float64(maxScore), tags)
			metric.Gauge("qdrant_query_similarity_min_score", float64(minScore), tags)
		}
	}

	return &BatchQueryResponse{SimilarCandidatesList: similarCandidatesList}
}
