package feature

import (
	"context"
	"fmt"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/rs/zerolog/log"
)

var (
	v1   *Handler
	once sync.Once
)

type Handler struct {
	retrieve.FeatureServiceServer
	config          config.Manager
	retrieveHandler *RetrieveHandler
	persistHandler  *PersistHandler
}

func InitV1() retrieve.FeatureServiceServer {
	if v1 == nil {
		once.Do(func() {
			configs := config.Instance(1)
			v1 = &Handler{
				config:          configs,
				retrieveHandler: InitRetrieveHandler(configs),
				persistHandler:  InitPersistHandler(),
			}
		})
	}
	return v1
}

func (h *Handler) RetrieveFeatures(ctx context.Context, query *retrieve.Query) (*retrieve.Result, error) {
	startTime := time.Now()
	result, err := h.retrieveHandler.RetrieveFeatures(ctx, query)
	if err != nil {
		return nil, err
	}
	metric.Timing("orchestrator_retrieve_latency", time.Since(startTime), []string{"entity", query.GetEntityLabel()})
	return result, nil
}

/*
	RetrieveDecodedResult retrieves the decoded result for the given query.
	It first calls RetrieveFeatures to get the raw result and then decodes each feature value based on its data type.

To handle quantization, uncomment quantization support part 1 & part 2
*/
func (h *Handler) RetrieveDecodedResult(ctx context.Context, in *retrieve.Query) (*retrieve.DecodedResult, error) {
	result, err := h.RetrieveFeatures(ctx, in)

	if err != nil {
		return nil, err
	}

	// Quantization support Part 1
	featureLabelToDataTypeMap := make(map[string]types.DataType)
	for _, featureGroup := range in.FeatureGroups {
		fg, err := h.config.GetFeatureGroup(in.EntityLabel, featureGroup.Label)
		if err != nil {
			log.Error().Msgf("failed to fetch featureGroup for featureGroupLabel %s: %v", featureGroup.Label, err)
			return nil, err
		}
		dataType, _ := types.ParseDataType(fg.DataType.String())
		for _, featureLabel := range featureGroup.FeatureLabels {
			baseFeatureLabel, quantDataType, _ := ParseFeatureLabel(featureLabel, dataType)
			featureLabelToDataTypeMap[featureGroup.Label+":"+baseFeatureLabel] = quantDataType
		}
	}

	DecodedResult := retrieve.DecodedResult{
		FeatureSchemas: result.FeatureSchemas,
		Rows:           make([]*retrieve.DecodedRow, 0),
		KeysSchema:     result.KeysSchema,
	}
	for _, row := range result.Rows {
		decodeRow := retrieve.DecodedRow{
			Keys:    make([]string, 0),
			Columns: make([]string, 0),
		}
		decodeRow.Keys = append(decodeRow.Keys, row.Keys...)
		for _, featureSchema := range result.FeatureSchemas {
			featureGroupLabel := featureSchema.FeatureGroupLabel
			featureGroup, err := h.config.GetFeatureGroup(in.EntityLabel, featureGroupLabel)
			if err != nil {
				log.Error().Msgf("failed to fetch featureGroup for featureGroupLabel %s: %v", featureGroupLabel, err)
				return nil, err
			}
			for _, feature := range featureSchema.Features {
				dataType, _ := types.ParseDataType(featureGroup.DataType.String())
				//Quantization support Part 2: use the requested dataType
				if types.DataTypeUnknown != featureLabelToDataTypeMap[featureGroupLabel+":"+feature.Label] {
					dataType = featureLabelToDataTypeMap[featureGroupLabel+":"+feature.Label]
				}
				decodedValue, decodeErr := h.decodeFeatureValue(row.Columns[feature.ColumnIdx], in.EntityLabel, feature.Label, featureGroup, dataType)
				if decodeErr != nil {
					return nil, fmt.Errorf("failed to decode value at column %d for feature group %s: %w", feature.ColumnIdx, featureGroupLabel, decodeErr)
				}
				decodeRow.Columns = append(decodeRow.Columns, fmt.Sprintf("%v", decodedValue))
			}
		}
		DecodedResult.Rows = append(DecodedResult.Rows, &decodeRow)
	}
	return &DecodedResult, nil
}

func (h *Handler) decodeFeatureValue(encodedValue []byte, entityLabel, featureLabel string, featureGroup *config.FeatureGroup, dataType types.DataType) (interface{}, error) {
	switch dataType.String() {
	case "DataTypeFP64":
		return blocks.HelperScalarFeatureToTypeFloat64(encodedValue)
	case "DataTypeFP32":
		return blocks.HelperScalarFeatureToTypeFloat32(encodedValue)
	case "DataTypeFP16":
		return blocks.HelperScalarFeatureToTypeFloat16(encodedValue)
	case "DataTypeFP8E5M2":
		return blocks.HelperScalarFeatureToTypeFloat8E5M2(encodedValue)
	case "DataTypeFP8E4M3":
		return blocks.HelperScalarFeatureToTypeFloat8E4M3(encodedValue)
	case "DataTypeUint64":
		return blocks.HelperScalarFeatureToTypeUint64(encodedValue)
	case "DataTypeUint32":
		return blocks.HelperScalarFeatureToTypeUint32(encodedValue)
	case "DataTypeUint16":
		return blocks.HelperScalarFeatureToTypeUint16(encodedValue)
	case "DataTypeUint8":
		return blocks.HelperScalarFeatureToTypeUint8(encodedValue)
	case "DataTypeInt8":
		return blocks.HelperScalarFeatureToTypeInt8(encodedValue)
	case "DataTypeInt16":
		return blocks.HelperScalarFeatureToTypeInt16(encodedValue)
	case "DataTypeInt32":
		return blocks.HelperScalarFeatureToTypeInt32(encodedValue)
	case "DataTypeInt64":
		return blocks.HelperScalarFeatureToTypeInt64(encodedValue)
	case "DataTypeString":
		return blocks.HelperScalarFeatureToTypeString(encodedValue)
	case "DataTypeBool":
		return blocks.HelperScalarFeatureToTypeBool(encodedValue)
	case "DataTypeFP8E5M2Vector":
		return blocks.HelperVectorFeatureFp8E5M2ToConcatenatedString(encodedValue)
	case "DataTypeFP8E4M3Vector":
		return blocks.HelperVectorFeatureFp8E4M3ToConcatenatedString(encodedValue)
	case "DataTypeFP16Vector":
		return blocks.HelperVectorFeatureFp16ToConcatenatedString(encodedValue)
	case "DataTypeFP32Vector":
		return blocks.HelperVectorFeatureFp32ToConcatenatedString(encodedValue)
	case "DataTypeFP64Vector":
		return blocks.HelperVectorFeatureFp64ToConcatenatedString(encodedValue)
	case "DataTypeBoolVector":
		return blocks.HelperVectorFeatureBoolToConcatenatedString(encodedValue)
	case "DataTypeStringVector":
		activeVersion, err := strconv.Atoi(featureGroup.ActiveVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to convert active version to int: %w", err)
		}
		stringLengths, err := h.config.GetStringLengths(entityLabel, featureGroup.Id, activeVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to get string lengths for feature group %v: %s", featureGroup.Id, err)
		}
		sequence, err := h.config.GetSequenceNo(entityLabel, featureGroup.Id, activeVersion, featureLabel)
		if err != nil || sequence == -1 {
			return nil, fmt.Errorf("failed to get sequence number for feature %s in feature group %v: %s", featureLabel, featureGroup.Id, err)
		}
		decodedValue, err := blocks.HelperVectorFeatureStringToConcatenatedString(encodedValue, int(stringLengths[sequence]))
		if err != nil {
			return nil, fmt.Errorf("failed to decode string vector: %w", err)
		}
		return decodedValue, nil
	case "DataTypeInt8Vector":
		return blocks.HelperVectorFeatureInt8ToConcatenatedString(encodedValue)
	case "DataTypeInt16Vector":
		return blocks.HelperVectorFeatureInt16ToConcatenatedString(encodedValue)
	case "DataTypeInt32Vector":
		return blocks.HelperVectorFeatureInt32ToConcatenatedString(encodedValue)
	case "DataTypeInt64Vector":
		return blocks.HelperVectorFeatureInt64ToConcatenatedString(encodedValue)
	case "DataTypeUint8Vector":
		return blocks.HelperVectorFeatureUint8ToConcatenatedString(encodedValue)
	case "DataTypeUint16Vector":
		return blocks.HelperVectorFeatureUint16ToConcatenatedString(encodedValue)
	case "DataTypeUint32Vector":
		return blocks.HelperVectorFeatureUint32ToConcatenatedString(encodedValue)
	case "DataTypeUint64Vector":
		return blocks.HelperVectorFeatureUint64ToConcatenatedString(encodedValue)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", featureGroup.DataType.String())
	}
}
