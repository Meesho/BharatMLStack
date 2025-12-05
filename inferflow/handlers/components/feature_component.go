package components

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/models"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/typeconverter"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	extFS "github.com/Meesho/BharatMLStack/inferflow/handlers/external/featurestore"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
)

type FeatureComponent struct {
	ComponentName string
}

func (fComponent *FeatureComponent) GetComponentName() string {
	return fComponent.ComponentName
}

func (fComponent *FeatureComponent) Run(request interface{}) {
	componentRequest, ok := request.(ComponentRequest)
	if !ok {
		return
	}

	modelID := componentRequest.ModelId
	componentName := fComponent.GetComponentName()
	metricTags := []string{modelId, modelID, component, componentName}

	logErrorAndCount := func(msg string) {
		logger.Error(msg, nil)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Panic recovered for model-id %s and component %s: %v", modelID, componentName, r), nil)
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, "feature-store-component-panic"))
			return
		}
	}()

	startTime := time.Now()
	metrics.Count("inferflow.component.execution.total", 1, metricTags)

	iFeatureConfig, ok := componentRequest.ComponentConfig.FeatureComponentConfig.Get(componentName)
	if !ok {
		logErrorAndCount(fmt.Sprintf("Config not found for component %s", componentName))
		return
	}

	fConfig, ok := iFeatureConfig.(config.FeatureComponentConfig)
	if !ok {
		logErrorAndCount(fmt.Sprintf("Error casting config for component %s", componentName))
		return
	}

	if !validateFsConfig(&fConfig) {
		logErrorAndCount(fmt.Sprintf("Invalid component config for model-id %s and component %s", modelID, componentName))
		return
	}

	componentMatrix := *componentRequest.ComponentData
	featureComponentBuilder := &models.FeatureComponentBuilder{}
	fComponent.populateFeatureComponentBuilder(modelID, &componentRequest, fConfig, featureComponentBuilder, metricTags)

	// Iterate over all feature labels and populate corresponding columns in componentMatrix
	for labelIdx, label := range featureComponentBuilder.Labels {
		colName := fConfig.ColNamePrefix + label
		byteCol, hasByteCol := componentMatrix.ByteColumnIndexMap[colName]
		if !hasByteCol {
			continue
		}

		stringCol, hasStringCol := componentMatrix.StringColumnIndexMap[colName]
		entityStringValueMap := make(map[string]string)

		for rowIdx, entity := range featureComponentBuilder.RequestedEntityIds {
			// Get the feature value for this entity and label
			val := featureComponentBuilder.Result[featureComponentBuilder.UniqueEntityIndex[entity]][labelIdx]

			// Populate the byte data column
			componentMatrix.Rows[rowIdx].ByteData[byteCol.Index] = val

			// If string column exists and is empty, populate it
			if hasStringCol && componentMatrix.Rows[rowIdx].StringData[stringCol.Index] == "" {
				if strVal, isPresent := entityStringValueMap[entity]; isPresent {
					componentMatrix.Rows[rowIdx].StringData[stringCol.Index] = strVal
					continue
				}
				strVal, err := typeconverter.BytesToString(val, byteCol.DataType)
				if err != nil {
					log.Warn().
						Str("feature", byteCol.Name).
						AnErr("error", err).
						Msg("PopulateByteDataFromFeatureMatrix: error in BytesToString")
					strVal = ""
				}
				entityStringValueMap[entity] = strVal
				componentMatrix.Rows[rowIdx].StringData[stringCol.Index] = strVal
			}
		}
	}

	metrics.Count("inferflow.component.feature.count", int64(len(featureComponentBuilder.UniqueEntityIds))*int64(len(featureComponentBuilder.Labels)), metricTags)
	metrics.Timing("inferflow.component.execution.latency", time.Since(startTime), metricTags)
}

func (fComponent *FeatureComponent) populateFeatureComponentBuilder(modelID string, componentRequest *ComponentRequest, oConfig config.FeatureComponentConfig, featureComponentBuilder *models.FeatureComponentBuilder, metricTags []string) {
	componentMatrix := componentRequest.ComponentData
	componentConfig := componentRequest.ComponentConfig
	cacheEnabled := componentRequest.ComponentConfig.CacheEnabled && oConfig.CompCacheEnabled
	errLoggingPercent := componentRequest.ComponentConfig.ErrorLoggingPercent

	initializeFeatureComponentBuilder(componentMatrix, oConfig, featureComponentBuilder, metricTags)
	if oConfig.CompositeId {
		componentMatrix.PopulateStringDataForCompositeKey(oConfig.ComponentId, featureComponentBuilder.KeyValues)
	}

	if len(oConfig.FSRequest.FeatureGroups) == 0 ||
		len(oConfig.FSRequest.FeatureGroups[0].Features) == 0 {
		return
	}

	if cacheEnabled {
		fComponent.populateFeatureValuesFromCache(modelID, componentConfig, featureComponentBuilder)
	}

	fComponent.populateFeatureValueFromFeatureStore(modelID, oConfig, featureComponentBuilder, errLoggingPercent)

	if cacheEnabled {
		go fComponent.putFeatureValuesIntoCache(modelID, featureComponentBuilder, componentConfig, &oConfig)
	}
}

func initializeFeatureComponentBuilder(componentMatrix *matrix.ComponentMatrix, config config.FeatureComponentConfig, featureComponentBuilder *models.FeatureComponentBuilder, metricTags []string) {
	// Extract key values based on FSKeys from the component matrix
	fsKeyValues := componentMatrix.GetColumnValuesWithKey(config.FSKeys)
	featureComponentBuilder.KeyValues = fsKeyValues.Values

	// Build the key schema from FSKeys
	featureComponentBuilder.KeySchema = extractKeySchema(config.FSKeys)

	// Count total number of features across all feature groups
	featureCount := countTotalFeatures(config.FSRequest.FeatureGroups)

	// Initialize labels and their index mapping
	initializeLabels(config.FSRequest, featureCount, featureComponentBuilder)

	// Build list of requested entities by concatenating key values for each row
	buildRequestedEntities(featureComponentBuilder, metricTags)

	// Initialize unique entities and related mappings and buffers
	initializeUniqueEntities(featureComponentBuilder, featureCount)
}

// Extracts schema strings from FSKeys
func extractKeySchema(fsKeys []config.FSKey) []string {
	keySchema := make([]string, len(fsKeys))
	for i, key := range fsKeys {
		keySchema[i] = key.Schema
	}
	return keySchema
}

// Counts total number of features across all feature groups
func countTotalFeatures(featureGroups []config.FeatureGroup) int {
	count := 0
	for _, group := range featureGroups {
		count += len(group.Features)
	}
	return count
}

// Initializes the Labels slice and LabelIndex map
func initializeLabels(fsRequest config.FeatureRequest, featureCount int, featureComponentBuilder *models.FeatureComponentBuilder) {
	featureComponentBuilder.Labels = make([]string, featureCount)
	featureComponentBuilder.LabelIndex = make(map[string]int, featureCount)

	entityLabelPrefix := fsRequest.Label + ":"
	var sb strings.Builder
	sb.Grow(64)

	idx := 0
	for _, group := range fsRequest.FeatureGroups {
		sb.Reset()
		sb.WriteString(entityLabelPrefix)
		sb.WriteString(group.Label)
		sb.WriteByte(':')
		prefix := sb.String()

		for _, feature := range group.Features {
			sb.Reset()
			sb.WriteString(prefix)
			sb.WriteString(feature)
			label := sb.String()

			featureComponentBuilder.Labels[idx] = label
			featureComponentBuilder.LabelIndex[label] = idx
			idx++
		}
	}
}

// Concatenates key values into strings to build RequestedEntities list
func buildRequestedEntities(featureComponentBuilder *models.FeatureComponentBuilder, metricTags []string) {
	numRows := len(featureComponentBuilder.KeyValues)
	featureComponentBuilder.RequestedEntityIds = make([]string, numRows)

	var sb strings.Builder
	for i, row := range featureComponentBuilder.KeyValues {
		sb.Reset()
		for j, val := range row {
			if j > 0 {
				sb.WriteByte('|')
			}
			sb.WriteString(val)
		}
		featureComponentBuilder.RequestedEntityIds[i] = sb.String()
	}
}

func initializeUniqueEntities(featureComponentBuilder *models.FeatureComponentBuilder, featureCount int) {
	uniqueEntitySet := make(map[string]bool)
	for _, entity := range featureComponentBuilder.RequestedEntityIds {
		uniqueEntitySet[entity] = true
	}

	uniqueCount := len(uniqueEntitySet)
	featureComponentBuilder.InMemPresent = make([]bool, uniqueCount)
	featureComponentBuilder.UniqueEntityIds = make([]string, uniqueCount)
	featureComponentBuilder.UniqueEntityIndex = make(map[string]int, uniqueCount)
	featureComponentBuilder.Result = make([][][]byte, uniqueCount)
	featureComponentBuilder.InMemoryMissCount = uniqueCount
	featureComponentBuilder.EntitiesToKeyValuesMap = make(map[string][]string, uniqueCount)
	byteBuffer := make([][]byte, uniqueCount*featureCount)
	index := 0
	for i, entity := range featureComponentBuilder.RequestedEntityIds {
		if _, exists := featureComponentBuilder.UniqueEntityIndex[entity]; !exists {
			featureComponentBuilder.UniqueEntityIndex[entity] = index
			featureComponentBuilder.UniqueEntityIds[index] = entity
			featureComponentBuilder.EntitiesToKeyValuesMap[entity] = featureComponentBuilder.KeyValues[i]
			featureComponentBuilder.Result[index] = byteBuffer[index*featureCount : (index+1)*featureCount]
			featureComponentBuilder.InMemPresent[index] = false
			index++
		}
	}
}

func (fComponent *FeatureComponent) populateFeatureValuesFromCache(
	modelID string,
	componentConfig *config.ComponentConfig,
	featureComponentBuilder *models.FeatureComponentBuilder,
) {
	cache := inmemorycache.InMemoryCacheInstance
	componentName := fComponent.GetComponentName()

	metricTags := []string{modelId, modelID, component, componentName}
	metrics.Count("inferflow.component.inmemorycache.request.total", int64(len(featureComponentBuilder.UniqueEntityIds)), metricTags)

	cacheVersion := componentConfig.CacheVersion
	if cacheVersion == 0 {
		cacheVersion = defaultCacheVersion
	}

	var keyBuilder strings.Builder
	keyBuilder.Grow(len(modelID) + len(extFS.CacheKeySeparator)*2 + 10 + len(componentName))
	keyBuilder.WriteString(modelID)
	keyBuilder.WriteString(extFS.CacheKeySeparator)
	keyBuilder.WriteString(strconv.Itoa(cacheVersion))
	keyBuilder.WriteString(extFS.CacheKeySeparator)
	keyBuilder.WriteString(fComponent.GetComponentName())
	keyBuilder.WriteString(extFS.CacheKeySeparator)
	cacheKeyPrefix := keyBuilder.String()

	featureComponentBuilder.InMemoryMissCount = 0
	for idx, row := range featureComponentBuilder.UniqueEntityIds {
		cacheKey := cacheKeyPrefix + row
		data, err := cache.Get([]byte(cacheKey))
		if err != nil || len(data) == 0 {

			featureComponentBuilder.InMemoryMissCount++
			continue
		}
		featureComponentBuilder.Result[idx] = DecodeBytesArray(data)
		featureComponentBuilder.InMemPresent[idx] = true
	}

	if featureComponentBuilder.InMemoryMissCount > 0 {
		metrics.Count("inferflow.component.inmemorycache.request.miss.total", int64(featureComponentBuilder.InMemoryMissCount), metricTags)
	}
}

func EncodeBytesArray(data [][]byte) []byte {
	if len(data) == 0 || len(data[0]) == 0 {
		return nil
	}
	totalSize := 0
	for _, b := range data {
		totalSize += 4 + len(b)
	}

	result := make([]byte, totalSize)
	offset := 0

	for _, b := range data {
		length := int32(len(b))
		binary.LittleEndian.PutUint32(result[offset:offset+4], uint32(length))
		offset += 4
		copy(result[offset:offset+len(b)], b)
		offset += len(b)
	}

	return result
}

func DecodeBytesArray(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}
	count := 0
	for i := 0; i < len(data); {
		if i+4 > len(data) {
			break
		}
		length := int(binary.LittleEndian.Uint32(data[i : i+4]))
		i += 4
		if i+length > len(data) {
			break
		}
		count++
		i += length
	}

	result := make([][]byte, count)
	idx := 0

	for i := 0; i < len(data); {
		if i+4 > len(data) {
			break
		}
		length := int(binary.LittleEndian.Uint32(data[i : i+4]))
		i += 4
		if i+length > len(data) {
			break
		}
		result[idx] = data[i : i+length]
		idx++
		i += length
	}

	return result
}

func (fComponent *FeatureComponent) populateFeatureValueFromFeatureStore(modelID string, config config.FeatureComponentConfig, featureComponentBuilder *models.FeatureComponentBuilder, errLoggingPercent int) {
	metricTags := []string{modelId, modelID, component, fComponent.GetComponentName()}
	extFS.GetOnFsResponse(config, featureComponentBuilder, errLoggingPercent, metricTags)
}
func (fComponent *FeatureComponent) putFeatureValuesIntoCache(
	modelID string,
	featureComponentBuilder *models.FeatureComponentBuilder,
	componentConfig *config.ComponentConfig,
	oConfig *config.FeatureComponentConfig,
) {
	c := inmemorycache.InMemoryCacheInstance
	if c == nil {
		return
	}

	compCacheTtl := oConfig.CompCacheTtl
	if compCacheTtl == 0 {
		compCacheTtl = componentConfig.CacheTtl
	}
	if compCacheTtl == 0 {
		compCacheTtl = defaultCacheTtl
	}

	compCacheVersion := componentConfig.CacheVersion
	if compCacheVersion == 0 {
		compCacheVersion = defaultCacheVersion
	}

	var entityLabelKeyBuilder strings.Builder
	entityLabelKeyBuilder.Grow(len(modelID) + len(extFS.CacheKeySeparator)*2 + 10 + len(fComponent.GetComponentName()))
	entityLabelKeyBuilder.WriteString(modelID)
	entityLabelKeyBuilder.WriteString(extFS.CacheKeySeparator)
	entityLabelKeyBuilder.WriteString(strconv.Itoa(compCacheVersion))
	entityLabelKeyBuilder.WriteString(extFS.CacheKeySeparator)
	entityLabelKeyBuilder.WriteString(fComponent.GetComponentName())
	entityLabelKeyBuilder.WriteString(extFS.CacheKeySeparator)
	entityLabelKey := entityLabelKeyBuilder.String()

	for rowIdx, row := range featureComponentBuilder.UniqueEntityIds {
		cacheKey := entityLabelKey + row
		if featureComponentBuilder.InMemPresent[rowIdx] {
			continue
		}
		//Set
		serialized := EncodeBytesArray(featureComponentBuilder.Result[featureComponentBuilder.UniqueEntityIndex[row]])
		if serialized == nil {
			continue
		}
		err := c.SetEx([]byte(cacheKey), serialized, compCacheTtl)
		//c.SetWithTTL(cacheKey, featureMatrix.Values[featureMatrix.EntityIndex[row]], time.Duration(compCacheTtl)*time.Second)
		if err != nil {
			logger.Error(fmt.Sprintf("Error caching feature values for key: %s", cacheKey), err)
		}
	}
}

func validateFsConfig(fConfig *config.FeatureComponentConfig) bool {
	return !(fConfig == nil ||
		fConfig.ComponentId == "" ||
		fConfig.FSKeys == nil || len(fConfig.FSKeys) == 0 ||
		fConfig.FSFlattenRespKeys == nil || len(fConfig.FSFlattenRespKeys) == 0)
}
