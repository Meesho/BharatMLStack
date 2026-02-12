package inferflow

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	kafkaLogger "github.com/Meesho/BharatMLStack/inferflow/handlers/external/prism"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/typeconverter"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
)

// logInferflowResponse performs V1 logging by sending feature data in JSON format to Kafka.
func logInferflowResponse(ctx context.Context, userId, trackingId string, conf *config.Config, compRequest *components.ComponentRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic in logInferflowResponse: %v", r)
		}
	}()

	if compRequest == nil || compRequest.ComponentData == nil {
		logger.Error("Component request or component data is nil", nil)
		return
	}

	data := kafkaLogger.ItemsLoggingData{
		UserId:             userId,
		ModelProxyConfigId: compRequest.ModelId,
		IopId:              trackingId,
		ItemsMeta:          kafkaLogger.ItemsMeta{},
		Items:              make([]kafkaLogger.Item, 0),
	}

	featuresToLog := buildFeatureFilter(conf)

	// Structure to map schema index to original column info
	type columnMapping struct {
		IsStringColumn bool
		OriginalIndex  int
		DataType       string
	}

	shouldInclude := func(name string) bool {
		return featuresToLog == nil || featuresToLog[name]
	}

	totalCols := len(compRequest.ComponentData.ByteColumnIndexMap) + len(compRequest.ComponentData.StringColumnIndexMap)
	if featuresToLog != nil {
		totalCols = len(featuresToLog)
	}
	featureSchema := make([]string, 0, totalCols)
	mappings := make([]columnMapping, 0, totalCols)
	nameToIdx := make(map[string]int, totalCols)

	if len(conf.ResponseConfig.Features) == 0 {
		return
	}

	// First column is always the entity ID
	idColName := conf.ResponseConfig.Features[0]
	if col, ok := compRequest.ComponentData.StringColumnIndexMap[idColName]; ok {
		idx := len(featureSchema)
		featureSchema = append(featureSchema, idColName)
		nameToIdx[idColName] = idx
		mappings = append(mappings, columnMapping{IsStringColumn: true, OriginalIndex: col.Index})
	}

	// Collect byte columns
	for name, col := range compRequest.ComponentData.ByteColumnIndexMap {
		if name == idColName || !shouldInclude(name) {
			continue
		}
		if _, exists := nameToIdx[name]; exists {
			continue
		}
		idx := len(featureSchema)
		featureSchema = append(featureSchema, name)
		nameToIdx[name] = idx
		mappings = append(mappings, columnMapping{IsStringColumn: false, OriginalIndex: col.Index, DataType: col.DataType})
	}

	// Collect string columns
	for name, col := range compRequest.ComponentData.StringColumnIndexMap {
		if name == idColName || !shouldInclude(name) {
			continue
		}
		if idx, exists := nameToIdx[name]; exists {
			mappings[idx] = columnMapping{IsStringColumn: true, OriginalIndex: col.Index}
		} else {
			idx := len(featureSchema)
			featureSchema = append(featureSchema, name)
			nameToIdx[name] = idx
			mappings = append(mappings, columnMapping{IsStringColumn: true, OriginalIndex: col.Index})
		}
	}

	data.ItemsMeta.FeatureSchema = featureSchema

	// Add compute ID if available
	if computeId := getComputeIdForLogging(conf); computeId != "" {
		data.ItemsMeta.ComputeID = computeId
	}

	// Add headers from incoming gRPC context
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		hMap := make(map[string]string)
		for headerKey, field := range headersMap {
			if vals := md.Get(headerKey); len(vals) > 0 {
				hMap[field] = vals[0]
			}
		}
		data.ItemsMeta.Headers = hMap
	}

	// Build items
	schemaLen := len(data.ItemsMeta.FeatureSchema)
	for _, row := range compRequest.ComponentData.Rows {
		fullRow := make([]string, schemaLen)
		for si, cm := range mappings {
			if cm.IsStringColumn {
				fullRow[si] = row.StringData[cm.OriginalIndex]
			} else {
				decoded, err := typeconverter.BytesToString(row.ByteData[cm.OriginalIndex], cm.DataType)
				if err != nil {
					log.Warn().AnErr(fmt.Sprintf("Error converting bytes for feature %s", featureSchema[si]), err)
					fullRow[si] = ""
				} else {
					fullRow[si] = decoded
				}
			}
		}
		data.Items = append(data.Items, kafkaLogger.Item{Id: fullRow[0], Features: fullRow})
	}

	kafkaLogger.SendV1Log(&data, conf.ResponseConfig.LogBatchSize)
}

// buildFeatureFilter returns a set of feature names to log, or nil to log all.
func buildFeatureFilter(conf *config.Config) map[string]bool {
	if utils.IsNilOrEmpty(conf.ResponseConfig) ||
		!conf.ResponseConfig.LogFeatures ||
		utils.IsNilOrEmpty(conf.ResponseConfig.Features) {
		return nil
	}
	m := make(map[string]bool, len(conf.ResponseConfig.Features))
	for _, f := range conf.ResponseConfig.Features {
		m[f] = true
	}
	return m
}

// getComputeIdForLogging extracts the compute ID from Numerix component config.
func getComputeIdForLogging(c *config.Config) string {
	var computeId string
	for _, iComp := range c.ComponentConfig.NumerixComponentConfig.Values() {
		if comp, ok := iComp.(config.NumerixComponentConfig); ok {
			computeId = comp.ComputeId
		}
	}
	return computeId
}
