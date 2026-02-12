package inferflow

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	kafkaLogger "github.com/Meesho/BharatMLStack/inferflow/handlers/external/prism"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/types"
	pb "github.com/Meesho/BharatMLStack/inferflow/server/grpc"

	parquet "github.com/parquet-go/parquet-go"

	"github.com/apache/arrow/go/v16/arrow"
	"github.com/apache/arrow/go/v16/arrow/array"
	"github.com/apache/arrow/go/v16/arrow/ipc"
	"github.com/apache/arrow/go/v16/arrow/memory"
)

// Format type constants for the metadata byte.
const (
	FormatTypeProto   = 0 // 00
	FormatTypeArrow   = 1 // 01
	FormatTypeParquet = 2 // 10
)

// parquetRow represents a single logged row in Parquet format.
type parquetRow struct {
	Features map[int][]byte
}

// --- Metadata packing ---

// packMetadataByte packs compression flag, version, and format type into a single byte.
//
//	Bits 0-1: compression (00=off, 01=on)
//	Bits 2-5: version (0-15)
//	Bits 6-7: format (00=proto, 01=arrow, 10=parquet)
func packMetadataByte(compressionEnabled bool, version int, formatType int) byte {
	var b byte
	if compressionEnabled {
		b |= 1 << 0
	}
	b |= byte(version) << 2
	b |= byte(formatType) << 6
	return b
}

// --- Common helpers ---

// extractParentEntityFromHeaders extracts parent entity value from gRPC metadata.
func extractParentEntityFromHeaders(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range []string{"parent_catalog_id", "search_query", "clp_id", "collection_id"} {
			if vals := md.Get(key); len(vals) > 0 && vals[0] != "" {
				return vals[0]
			}
		}
	}
	return ""
}

// buildMPLogBase creates the base InferflowLog structure with common fields.
func buildMPLogBase(ctx context.Context, userId, trackingId string, conf *config.Config, compRequest *components.ComponentRequest, formatType int) *pb.InferflowLog {
	svcConfig := config.GetModelConfigMap().ServiceConfig
	meta := packMetadataByte(svcConfig.CompressionEnabled, compRequest.ComponentConfig.CacheVersion, formatType)
	parent := extractParentEntityFromHeaders(ctx)

	mpLog := &pb.InferflowLog{
		UserId:     userId,
		TrackingId: trackingId,
		MpConfigId: compRequest.ModelId,
		Metadata:   []byte{meta},
	}
	if parent != "" {
		mpLog.ParentEntity = []string{parent}
	}
	return mpLog
}

// getBatchSize returns the batch size from config, defaulting to 500.
func getBatchSize(conf *config.Config) int {
	if conf == nil || conf.ResponseConfig.LogBatchSize <= 0 {
		return 500
	}
	return conf.ResponseConfig.LogBatchSize
}

// --- V2 Kafka transport ---

// logInferenceBatchInsights wraps InferflowLog in KafkaRequest and sends to Kafka via V2 writer.
func logInferenceBatchInsights(mpLog *pb.InferflowLog, trackingId, modelId string) {
	mpLogAny, err := anypb.New(mpLog)
	if err != nil {
		logger.Error("Error wrapping InferflowLog in Any", err)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", modelId, "error", "wrapping_mp_log"})
		return
	}

	now := time.Now()
	kafkaReq := &pb.KafkaRequest{
		Value: &pb.KafkaEventValue{
			EventName:  "inferflow_inference_logs",
			EventId:    trackingId,
			CreatedAt:  now.Unix(),
			Properties: mpLogAny,
			UserId:     mpLog.UserId,
		},
	}

	kafkaLogger.PublishInferenceInsightsLog(kafkaReq, modelId)
}

// logInferenceInsights splits InferflowLog entities into batches and sends each batch.
func logInferenceInsights(mpLog *pb.InferflowLog, conf *config.Config, compRequest *components.ComponentRequest, trackingId string) {
	batchSize := getBatchSize(conf)
	total := len(mpLog.Entities)

	if total == 0 {
		return
	}
	if total <= batchSize {
		logInferenceBatchInsights(mpLog, trackingId, compRequest.ModelId)
		return
	}

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batch := &pb.InferflowLog{
			UserId:       mpLog.UserId,
			TrackingId:   mpLog.TrackingId,
			MpConfigId:   mpLog.MpConfigId,
			Metadata:     mpLog.Metadata,
			ParentEntity: mpLog.ParentEntity,
			Entities:     mpLog.Entities[i:end],
			Features:     mpLog.Features[i:end],
		}
		logInferenceBatchInsights(batch, trackingId, compRequest.ModelId)
	}
}

// --- Proto V2 logging ---

// logInferflowResponseBytes encodes features using proto format and sends to Kafka.
func logInferflowResponseBytes(ctx context.Context, userId, trackingId string, conf *config.Config, compRequest *components.ComponentRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic in logInferflowResponseBytes: %v", r)
		}
	}()

	featureSchema, err := config.GetFeatureSchema(compRequest.ModelId)
	if err != nil {
		log.Error().Msgf("Error getting feature schema for model %s: %v", compRequest.ModelId, err)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "schema_not_found"})
		return
	}

	mpLog := buildMPLogBase(ctx, userId, trackingId, conf, compRequest, FormatTypeProto)

	featureKeys := featureSchema.Keys()
	idColName := conf.ResponseConfig.Features[0]
	col, ok := compRequest.ComponentData.StringColumnIndexMap[idColName]
	if !ok {
		log.Error().Msgf("Entity column not found for config: %s", compRequest.ModelId)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "id_column_not_found"})
		return
	}

	encodingErrors := 0
	for _, row := range compRequest.ComponentData.Rows {
		valid := true
		encoder := utils.NewFeatureEncoder(len(featureKeys) * 10)

		for _, keyIface := range featureKeys {
			featureName := keyIface.(string)

			infoIface, ok := featureSchema.Get(featureName)
			if !ok {
				log.Error().Msgf("Feature info missing in schema: %s", featureName)
				valid = false
				break
			}

			info := infoIface.(config.SchemaComponents)
			featureType := info.FeatureType
			isBytesType := utils.IsBytesDataType(featureType)

			var dataType types.DataType
			if !isBytesType {
				dataType, err = utils.StringToDataType(featureType)
				if err != nil {
					valid = false
					break
				}
			}

			var featureValue []byte
			var found bool

			if byteCol, exists := compRequest.ComponentData.ByteColumnIndexMap[featureName]; exists {
				featureValue = row.ByteData[byteCol.Index]
				found = len(featureValue) > 0
			} else if strCol, exists := compRequest.ComponentData.StringColumnIndexMap[featureName]; exists {
				strVal := row.StringData[strCol.Index]
				if strVal != "" {
					featureValue, err = utils.ConvertStringToType(strVal, dataType)
					if err == nil {
						found = true
					}
				}
			}

			if !found {
				if dataType.IsVector() {
					featureValue = []byte{}
				} else {
					featureValue = utils.GetDefaultValueByType(featureType)
				}
				encoder.MarkValueAsGenerated()
			}

			if isBytesType {
				if _, err = encoder.AppendBytesFeature(featureValue); err != nil {
					valid = false
					break
				}
			} else if _, err = encoder.AppendFeature(dataType, featureValue); err != nil {
				valid = false
				break
			}
		}

		if valid {
			mpLog.Entities = append(mpLog.Entities, row.StringData[col.Index])
			mpLog.Features = append(mpLog.Features, &pb.PerEntityFeatures{EncodedFeatures: encoder.Bytes()})
		} else {
			encodingErrors++
		}
	}

	if encodingErrors > 0 {
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "encoding_error"})
	} else {
		logInferenceInsights(mpLog, conf, compRequest, trackingId)
	}
}

// --- Arrow V2 logging ---

// logInferflowResponseArrow encodes features into Arrow IPC format and sends to Kafka.
func logInferflowResponseArrow(ctx context.Context, userId, trackingId string, conf *config.Config, compRequest *components.ComponentRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic in logInferflowResponseArrow: %v", r)
		}
	}()

	idColName := conf.ResponseConfig.Features[0]
	col, ok := compRequest.ComponentData.StringColumnIndexMap[idColName]
	if !ok {
		log.Error().Msgf("Entity column not found for config: %s", compRequest.ModelId)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "id_column_not_found"})
		return
	}

	rec, _, entityIDs, err := buildColumnarRecord(compRequest, col.Index)
	if err != nil {
		logger.Error("Error building Arrow record", err)
		return
	}
	if rec == nil {
		return
	}
	defer rec.Release()

	batchSize := getBatchSize(conf)
	totalRows := int(rec.NumRows())

	for i := 0; i < totalRows; i += batchSize {
		end := i + batchSize
		if end > totalRows {
			end = totalRows
		}

		batchMPLog := buildMPLogBase(ctx, userId, trackingId, conf, compRequest, FormatTypeArrow)
		batchRec := rec.NewSlice(int64(i), int64(end))
		defer batchRec.Release()

		var buf bytes.Buffer
		w := ipc.NewWriter(&buf, ipc.WithSchema(batchRec.Schema()))
		if err := w.Write(batchRec); err != nil {
			logger.Error("Error writing Arrow record", err)
			_ = w.Close()
			continue
		}
		if err := w.Close(); err != nil {
			logger.Error("Error closing Arrow IPC writer", err)
			continue
		}

		batchMPLog.Entities = entityIDs[i:end]
		batchMPLog.Features = []*pb.PerEntityFeatures{{EncodedFeatures: buf.Bytes()}}
		logInferenceBatchInsights(batchMPLog, trackingId, compRequest.ModelId)
	}
}

// buildColumnarRecord constructs an Arrow record with one binary column per feature.
func buildColumnarRecord(compRequest *components.ComponentRequest, entityColIndex int) (arrow.Record, time.Duration, []string, error) {
	start := time.Now()

	featureSchema, err := config.GetFeatureSchema(compRequest.ModelId)
	if err != nil {
		logger.Error("Error getting feature schema from config", err)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "schema_not_found"})
		return nil, 0, nil, err
	}

	rawKeys := featureSchema.Keys()
	if len(rawKeys) == 0 {
		return nil, time.Since(start), nil, nil
	}

	pool := memory.NewGoAllocator()
	fields := make([]arrow.Field, 0, len(rawKeys))
	for i := range rawKeys {
		fields = append(fields, arrow.Field{Name: fmt.Sprintf("%d", i), Type: arrow.BinaryTypes.Binary})
	}
	schema := arrow.NewSchema(fields, nil)

	builders := make([]*array.BinaryBuilder, len(rawKeys))
	for i := range builders {
		b := array.NewBinaryBuilder(pool, arrow.BinaryTypes.Binary)
		builders[i] = b
		defer b.Release()
	}

	entityIDs := make([]string, 0, len(compRequest.ComponentData.Rows))

	for _, row := range compRequest.ComponentData.Rows {
		if entityColIndex >= 0 && entityColIndex < len(row.StringData) {
			entityIDs = append(entityIDs, row.StringData[entityColIndex])
		} else {
			continue
		}

		for i, keyIface := range rawKeys {
			fname := keyIface.(string)
			var val []byte

			if byteCol, exists := compRequest.ComponentData.ByteColumnIndexMap[fname]; exists {
				if byteCol.Index >= 0 && byteCol.Index < len(row.ByteData) {
					val = row.ByteData[byteCol.Index]
				}
			} else if strCol, exists := compRequest.ComponentData.StringColumnIndexMap[fname]; exists {
				var dataType types.DataType
				if infoIface, ok := featureSchema.Get(fname); ok {
					if info, ok := infoIface.(config.SchemaComponents); ok {
						if dt, err := utils.StringToDataType(info.FeatureType); err == nil {
							dataType = dt
						}
					}
				}
				if strCol.Index >= 0 && strCol.Index < len(row.StringData) {
					strVal := row.StringData[strCol.Index]
					if strVal != "" && dataType != 0 {
						if converted, err := utils.ConvertStringToType(strVal, dataType); err == nil {
							val = converted
						}
					}
				}
			}

			if len(val) == 0 {
				builders[i].AppendNull()
			} else {
				builders[i].Append(val)
			}
		}
	}

	columns := make([]arrow.Array, 0, len(rawKeys))
	for _, b := range builders {
		columns = append(columns, b.NewArray())
	}

	var numRows int64
	if len(columns) > 0 {
		numRows = int64(columns[0].Len())
	}
	return array.NewRecord(schema, columns, numRows), time.Since(start), entityIDs, nil
}

// --- Parquet V2 logging ---

// logInferflowResponseParquet encodes features into Parquet format and sends to Kafka.
func logInferflowResponseParquet(ctx context.Context, userId, trackingId string, conf *config.Config, compRequest *components.ComponentRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic in logInferflowResponseParquet: %v", r)
		}
	}()

	idColName := conf.ResponseConfig.Features[0]
	col, ok := compRequest.ComponentData.StringColumnIndexMap[idColName]
	if !ok {
		log.Error().Msgf("Entity column not found for config: %s", compRequest.ModelId)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "id_column_not_found"})
		return
	}

	rows, _, _, entityIDs, err := buildParquetRows(compRequest, col.Index)
	if err != nil {
		logger.Error("Error building Parquet rows", err)
		return
	}

	batchSize := getBatchSize(conf)
	totalRows := len(rows)

	for i := 0; i < totalRows; i += batchSize {
		end := i + batchSize
		if end > totalRows {
			end = totalRows
		}

		batchMPLog := buildMPLogBase(ctx, userId, trackingId, conf, compRequest, FormatTypeParquet)

		var buf bytes.Buffer
		pw := parquet.NewGenericWriter[parquetRow](&buf)
		if _, err := pw.Write(rows[i:end]); err != nil {
			logger.Error("Error writing Parquet rows", err)
			_ = pw.Close()
			continue
		}
		if err := pw.Close(); err != nil {
			logger.Error("Error closing Parquet writer", err)
			continue
		}

		batchMPLog.Entities = entityIDs[i:end]
		batchMPLog.Features = []*pb.PerEntityFeatures{{EncodedFeatures: buf.Bytes()}}
		logInferenceBatchInsights(batchMPLog, trackingId, compRequest.ModelId)
	}
}

// buildParquetRows constructs rows for parquet-go.
func buildParquetRows(compRequest *components.ComponentRequest, entityColIndex int) ([]parquetRow, int, time.Duration, []string, error) {
	start := time.Now()

	featureSchema, err := config.GetFeatureSchema(compRequest.ModelId)
	if err != nil {
		logger.Error("Error getting feature schema from config", err)
		metrics.Count("inferflow.logging.error", 1, []string{"model-id", compRequest.ModelId, "error", "schema_not_found"})
		return nil, 0, 0, nil, err
	}

	rawKeys := featureSchema.Keys()
	if len(rawKeys) == 0 {
		return nil, 0, 0, nil, nil
	}

	rows := make([]parquetRow, 0, len(compRequest.ComponentData.Rows))
	entityIDs := make([]string, 0, len(compRequest.ComponentData.Rows))

	for _, perEntityRow := range compRequest.ComponentData.Rows {
		if entityColIndex < 0 || entityColIndex >= len(perEntityRow.StringData) {
			continue
		}
		entityIDs = append(entityIDs, perEntityRow.StringData[entityColIndex])

		row := parquetRow{Features: make(map[int][]byte, len(rawKeys))}

		for i, keyIface := range rawKeys {
			fname := keyIface.(string)
			var val []byte

			if byteCol, exists := compRequest.ComponentData.ByteColumnIndexMap[fname]; exists {
				if byteCol.Index >= 0 && byteCol.Index < len(perEntityRow.ByteData) {
					val = perEntityRow.ByteData[byteCol.Index]
				}
			} else if strCol, exists := compRequest.ComponentData.StringColumnIndexMap[fname]; exists {
				if strCol.Index >= 0 && strCol.Index < len(perEntityRow.StringData) {
					strVal := perEntityRow.StringData[strCol.Index]
					if strVal != "" {
						if infoIface, ok := featureSchema.Get(fname); ok {
							if info, ok := infoIface.(config.SchemaComponents); ok {
								if dt, err := utils.StringToDataType(info.FeatureType); err == nil && dt != 0 {
									if converted, err := utils.ConvertStringToType(strVal, dt); err == nil {
										val = converted
									}
								}
							}
						}
					}
				}
			}

			if len(val) == 0 {
				row.Features[i] = nil
			} else {
				row.Features[i] = val
			}
		}

		rows = append(rows, row)
	}

	return rows, len(rawKeys), time.Since(start), entityIDs, nil
}
