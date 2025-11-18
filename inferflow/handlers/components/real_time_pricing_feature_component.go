package components

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/models"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/Meesho/go-core/datatypeconverter/typeconverter"
	"github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client"
	clientmodels "github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client/models"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/rs/zerolog/log"
)

const (
	cacheKeySeparator = "|"
	clientVersion     = 1
)

type RealTimePricingFeatureComponent struct {
	ComponentName string
}

func (rtpbComponent *RealTimePricingFeatureComponent) GetComponentName() string {
	return rtpbComponent.ComponentName
}

func (rtpbComponent *RealTimePricingFeatureComponent) Run(request interface{}) {
	componentRequest, ok := request.(ComponentRequest)
	if !ok {
		return
	}

	modelID := componentRequest.ModelId
	componentName := rtpbComponent.GetComponentName()
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

	iRtpConfig, ok := componentRequest.ComponentConfig.RealTimePricingFeatureComponentConfig.Get(componentName)
	if !ok {
		logErrorAndCount(fmt.Sprintf("Config not found for component %s", componentName))
		return
	}

	rtpConfig, ok := iRtpConfig.(config.RealTimePricingFeatureComponentConfig)
	if !ok {
		logErrorAndCount(fmt.Sprintf("Error casting config for component %s", componentName))
		return
	}

	if !rtpbComponent.ValidateConfig(&rtpConfig) {
		logErrorAndCount(fmt.Sprintf("Invalid component config for model-id %s and component %s", modelID, componentName))
		return
	}

	componentMatrix := *componentRequest.ComponentData
	rtpFeatureBuilder := &models.RealTimePricingFeatureBuilder{}

	rtpbComponent.populateRealTimePricingFeatureBuilder(modelID, &componentRequest, rtpConfig, rtpFeatureBuilder)

	// Iterate over all feature labels and populate corresponding columns in componentMatrix
	for labelIdx, label := range rtpFeatureBuilder.Labels {
		colName := rtpConfig.ColNamePrefix + label
		byteCol, hasByteCol := componentMatrix.ByteColumnIndexMap[colName]
		if !hasByteCol {
			continue
		}
		stringCol, hasStringCol := componentMatrix.StringColumnIndexMap[colName]
		entityStringValueMap := make(map[string]string)

		for rowIdx, entity := range rtpFeatureBuilder.RequestedEntityIds {
			// Get the feature value for this entity and label
			val := rtpFeatureBuilder.Result[rtpFeatureBuilder.UniqueEntityIndex[entity]][labelIdx]

			// Populate the byte data column
			componentMatrix.Rows[rowIdx].ByteData[byteCol.Index] = val

			// If string column exists and is empty, populate it
			if hasStringCol && componentMatrix.Rows[rowIdx].StringData[stringCol.Index] == "" {
				if cachedStrVal, isPresent := entityStringValueMap[entity]; isPresent {
					componentMatrix.Rows[rowIdx].StringData[stringCol.Index] = cachedStrVal
					continue
				}
				var strVal string
				var err error
				if len(val) == 0 {
					strVal = ""
				} else {
					strVal, err = typeconverter.BytesToString(val, byteCol.DataType)
					if err != nil {
						log.Warn().
							Str("feature", byteCol.Name).
							Str("dataType", byteCol.DataType).
							Int("dataLength", len(val)).
							AnErr("error", err).
							Msg("PopulateByteDataFromPricingFeatureMatrix: error in BytesToString")
						strVal = ""
					}
				}

				entityStringValueMap[entity] = strVal
				componentMatrix.Rows[rowIdx].StringData[stringCol.Index] = strVal
			}
		}
	}

	metrics.Count("inferflow.component.feature.count", int64(len(rtpFeatureBuilder.RequestedEntityIds))*int64(len(rtpFeatureBuilder.Labels)), metricTags)
	metrics.Timing("inferflow.component.execution.latency", time.Since(startTime), metricTags)
}

func (rtpbComponent *RealTimePricingFeatureComponent) populateRealTimePricingFeatureBuilder(modelID string, componentRequest *ComponentRequest, rtpConfig config.RealTimePricingFeatureComponentConfig, rtpFeatureBuilder *models.RealTimePricingFeatureBuilder) {
	componentMatrix := componentRequest.ComponentData
	componentConfig := componentRequest.ComponentConfig
	cacheEnabled := componentConfig.CacheEnabled && rtpConfig.CompCacheEnabled
	errLoggingPercent := componentConfig.ErrorLoggingPercent
	headers := componentRequest.Headers

	initializeRealTimePricingFeatureBuilder(rtpbComponent, componentMatrix, rtpConfig, rtpFeatureBuilder)

	if rtpConfig.CompositeId {
		componentMatrix.PopulateStringDataForCompositeKey(rtpConfig.ComponentId, rtpFeatureBuilder.KeyValues)
	}

	if len(rtpConfig.FeatureRequest.FeatureGroups) == 0 ||
		len(rtpConfig.FeatureRequest.FeatureGroups[0].Features) == 0 {
		return
	}

	if cacheEnabled {
		rtpbComponent.populateFeatureValuesFromCache(modelID, componentConfig, rtpFeatureBuilder)
	}

	rtpbComponent.populateFeatureValueFromPricingService(modelID, rtpConfig, rtpFeatureBuilder, errLoggingPercent, headers)

	if cacheEnabled {
		go rtpbComponent.putFeatureValuesIntoCache(modelID, rtpFeatureBuilder, componentConfig, &rtpConfig)
	}
}

func initializeRealTimePricingFeatureBuilder(rtpbComponent *RealTimePricingFeatureComponent, componentMatrix *matrix.ComponentMatrix, rtpConfig config.RealTimePricingFeatureComponentConfig, rtpFeatureBuilder *models.RealTimePricingFeatureBuilder) {
	// Extract key values based on FSKeys from the component matrix (custom method to handle both string and byte columns)
	//rtpFeatureBuilder.KeyValues = rtpbComponent.extractKeyValuesFromMatrix(componentMatrix, rtpConfig.FSKeys)
	fsKeyValues := componentMatrix.GetColumnValuesWithKey(rtpConfig.FSKeys)
	rtpFeatureBuilder.KeyValues = fsKeyValues.Values
	// Build the key schema from FSKeys
	rtpFeatureBuilder.KeySchema = extractKeySchema(rtpConfig.FSKeys)

	// Count total number of features across all feature groups
	featureCount := countTotalPricingFeatures(rtpConfig.FeatureRequest.FeatureGroups)

	// Initialize labels and their index mapping
	initializePricingLabels(rtpConfig.FeatureRequest, featureCount, rtpFeatureBuilder)

	// Build list of requested entities by concatenating key values for each row
	buildRequestedPricingEntities(rtpFeatureBuilder)

	// Initialize unique entities and related mappings and buffers
	initializeUniquePricingEntities(rtpFeatureBuilder, featureCount)

	// Build EntityIdsMap directly from extracted key values
	rtpFeatureBuilder.EntityIdsMap = rtpbComponent.buildEntityIdsMapFromKeyValues(rtpFeatureBuilder, rtpConfig)

	// Legacy support - Extract configMap for backward compatibility
	configMap := linkedhashmap.New()
	for _, fsKey := range rtpConfig.FSKeys {
		configMap.Put(fsKey, fsKey.Column)
	}
	idMap := getColumnValuesWithKey(configMap, componentMatrix)
	rtpFeatureBuilder.IdMap = idMap
}

func (rtpbComponent *RealTimePricingFeatureComponent) ValidateConfig(rtpConfig *config.RealTimePricingFeatureComponentConfig) bool {

	if rtpbComponent.isNilOrEmpty(*rtpConfig) ||
		rtpbComponent.isNilOrEmpty(rtpConfig.ComponentId) ||
		rtpbComponent.isNilOrEmpty(rtpConfig.FSKeys) ||
		rtpbComponent.isNilOrEmpty(rtpConfig.FSFlattenRespKeys) {
		return false
	}
	return true
}

// isValidBinaryFeatureValues checks if binary feature values are valid (similar to isValidFeatureValues but for [][]byte)
func (rtpbComponent *RealTimePricingFeatureComponent) isValidBinaryFeatureValues(features [][]byte) bool {
	if rtpbComponent.isNilOrEmpty(features) {
		return false
	}
	// Check if all features are non-nil (basic validation for binary data)
	for _, feature := range features {
		if feature == nil {
			return false
		}
	}
	return true
}

// Helper functions
func (rtpbComponent *RealTimePricingFeatureComponent) isNilOrEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch v := v.(type) {
	case string:
		return v == ""
	case []string:
		return len(v) == 0
	case map[string]clientmodels.EntityId:
		return len(v) == 0
	case []clientmodels.EntityId:
		return len(v) == 0
	case [][]byte:
		return len(v) == 0
	case int:
		return v == 0
	case interface{}:
		return v == nil
	default:
		return false
	}
}

func createEntityBulkQuery(config config.RealTimePricingFeatureComponentConfig, entityIds []clientmodels.EntityId) *clientmodels.EntityBulkQuery {
	var featureGroups []clientmodels.FeatureGroupQuery
	for _, fg := range config.FeatureRequest.FeatureGroups {
		featureGroups = append(featureGroups, clientmodels.FeatureGroupQuery{
			Label:          fg.Label,
			FeatureQueries: fg.Features,
		})
	}
	return &clientmodels.EntityBulkQuery{
		Label:         config.FeatureRequest.Label,
		EntityIds:     entityIds,
		FeatureGroups: featureGroups,
	}
}

// Counts total number of features across all feature groups
func countTotalPricingFeatures(featureGroups []config.FeatureGroup) int {
	count := 0
	for _, group := range featureGroups {
		count += len(group.Features)
	}
	return count
}

// Initializes the Labels slice and LabelIndex map for pricing features
func initializePricingLabels(featureRequest config.FeatureRequest, featureCount int, rtpFeatureBuilder *models.RealTimePricingFeatureBuilder) {
	rtpFeatureBuilder.Labels = make([]string, featureCount)
	rtpFeatureBuilder.LabelIndex = make(map[string]int, featureCount)

	entityLabelPrefix := featureRequest.Label + ":"
	var sb strings.Builder
	sb.Grow(64)

	idx := 0
	for _, group := range featureRequest.FeatureGroups {
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

			rtpFeatureBuilder.Labels[idx] = label
			rtpFeatureBuilder.LabelIndex[label] = idx
			idx++
		}
	}
}

// Concatenates key values into strings to build RequestedEntities list for pricing
func buildRequestedPricingEntities(rtpFeatureBuilder *models.RealTimePricingFeatureBuilder) {
	numRows := len(rtpFeatureBuilder.KeyValues)
	rtpFeatureBuilder.RequestedEntityIds = make([]string, numRows)

	var sb strings.Builder
	for i, row := range rtpFeatureBuilder.KeyValues {
		sb.Reset()
		var nonEmptyValues []string
		for _, val := range row {
			if val != "" {
				nonEmptyValues = append(nonEmptyValues, val)
			}
		}
		for j, val := range nonEmptyValues {
			if j > 0 {
				sb.WriteByte('|')
			}
			sb.WriteString(val)
		}
		rtpFeatureBuilder.RequestedEntityIds[i] = sb.String()
	}
}

func initializeUniquePricingEntities(rtpFeatureBuilder *models.RealTimePricingFeatureBuilder, featureCount int) {
	uniqueEntitySet := make(map[string]bool)
	for _, entity := range rtpFeatureBuilder.RequestedEntityIds {
		uniqueEntitySet[entity] = true
	}

	uniqueCount := len(uniqueEntitySet)
	rtpFeatureBuilder.InMemPresent = make([]bool, uniqueCount)
	rtpFeatureBuilder.UniqueEntityIds = make([]string, uniqueCount)
	rtpFeatureBuilder.UniqueEntityIndex = make(map[string]int, uniqueCount)
	rtpFeatureBuilder.Result = make([][][]byte, uniqueCount)
	rtpFeatureBuilder.InMemoryMissCount = uniqueCount
	rtpFeatureBuilder.EntitiesToKeyValuesMap = make(map[string][]string, uniqueCount)
	byteBuffer := make([][]byte, uniqueCount*featureCount)
	index := 0
	for i, entity := range rtpFeatureBuilder.RequestedEntityIds {
		if _, exists := rtpFeatureBuilder.UniqueEntityIndex[entity]; !exists {
			rtpFeatureBuilder.UniqueEntityIndex[entity] = index
			rtpFeatureBuilder.UniqueEntityIds[index] = entity
			rtpFeatureBuilder.EntitiesToKeyValuesMap[entity] = rtpFeatureBuilder.KeyValues[i]
			rtpFeatureBuilder.Result[index] = byteBuffer[index*featureCount : (index+1)*featureCount]
			rtpFeatureBuilder.InMemPresent[index] = false
			index++
		}
	}
}

// Helper function to extract column values with key (similar pattern to feature_component.go)
func getColumnValuesWithKey(configMap *linkedhashmap.Map, componentData *matrix.ComponentMatrix) []map[interface{}][]string {
	// This is a simplified implementation - you may need to adjust based on your actual data structure
	fsKeys := make([]config.FSKey, 0)
	for _, key := range configMap.Keys() {
		if fsKey, ok := key.(struct {
			Schema string
			Column string
		}); ok {
			fsKeys = append(fsKeys, config.FSKey{
				Schema: fsKey.Schema,
				Column: fsKey.Column,
			})
		}
	}

	keyValues := componentData.GetColumnValuesWithKey(fsKeys)

	// Convert to expected format
	result := make([]map[interface{}][]string, 1)
	result[0] = make(map[interface{}][]string)

	for i, fsKey := range fsKeys {
		result[0][fsKey] = keyValues.Values[i]
	}

	return result
}

// Build EntityIdsMap directly from extracted key values (more reliable than legacy approach)
func (rtpbComponent *RealTimePricingFeatureComponent) buildEntityIdsMapFromKeyValues(
	rtpFeatureBuilder *models.RealTimePricingFeatureBuilder,
	rtpConfig config.RealTimePricingFeatureComponentConfig,
) map[string]clientmodels.EntityId {
	entityIdsMap := make(map[string]clientmodels.EntityId)

	// Build EntityId for each row of key values
rowLoop:
	for _, keyValueRow := range rtpFeatureBuilder.KeyValues {
		if len(keyValueRow) != len(rtpConfig.FSKeys) {
			continue
		}

		entityId := clientmodels.EntityId{}
		var entityKeyParts []string

		// Build all entity keys for this entity
		for keyIdx, fsKey := range rtpConfig.FSKeys {
			keyValue := keyValueRow[keyIdx]
			if keyValue != "" {
				entityKey := clientmodels.EntityKey{
					Type:  fsKey.Schema,
					Value: keyValue,
				}
				entityId.EntityKeys = append(entityId.EntityKeys, entityKey)
				entityKeyParts = append(entityKeyParts, keyValue)
			} else {
				continue rowLoop
			}
		}

		// Create the composite key string (same as built by buildRequestedPricingEntities)
		compositeKey := strings.Join(entityKeyParts, cacheKeySeparator)
		entityIdsMap[compositeKey] = entityId
	}

	return entityIdsMap
}

func (rtpbComponent *RealTimePricingFeatureComponent) populateFeatureValueFromPricingService(
	modelID string,
	rtpConfig config.RealTimePricingFeatureComponentConfig,
	rtpFeatureBuilder *models.RealTimePricingFeatureBuilder,
	errLoggingPercent int,
	headers map[string]string,
) {
	// Get entity IDs that are not cached
	entityIds := make([]clientmodels.EntityId, 0)

	for idx, entityId := range rtpFeatureBuilder.UniqueEntityIds {
		if !rtpFeatureBuilder.InMemPresent[idx] {
			// Convert entity string back to EntityId for API call
			if fsEntityId, exists := rtpFeatureBuilder.EntityIdsMap[entityId]; exists {
				entityIds = append(entityIds, fsEntityId)
			}
		}
	}

	if len(entityIds) > 0 {
		entityQueries := createEntityBulkQuery(rtpConfig, entityIds)

		clientInstance := client.Instance(clientVersion)
		if clientInstance == nil {
			logger.Error(fmt.Sprintf("Client instance is nil for model-id %s and component %s", modelID, rtpbComponent.GetComponentName()), nil)
			return
		}

		resp, err := clientInstance.GetBytestreamPricingFeatures(context.Background(), entityQueries, headers)
		if err != nil {
			logger.Error(fmt.Sprintf("Error getting pricing features for model-id %s and component %s", modelID, rtpbComponent.GetComponentName()), err)
			return
		}

		if resp != nil {
			// Convert the bytestream response to the binary format expected by the builder
			rtpbComponent.processBytestreamResponse(resp.ByteEntityPayloads, rtpFeatureBuilder, entityIds)
		}
	}
}

func (rtpbComponent *RealTimePricingFeatureComponent) processBytestreamResponse(
	entityPayloads []*clientmodels.ByteEntityPayloads,
	rtpFeatureBuilder *models.RealTimePricingFeatureBuilder, entityIds []clientmodels.EntityId,
) {
	if len(entityPayloads) == 0 {
		return
	}

	// Build list of indices that were sent to the pricing service (not cached)
	var uncachedIndices []int
	for idx := range rtpFeatureBuilder.UniqueEntityIds {
		if !rtpFeatureBuilder.InMemPresent[idx] {
			uncachedIndices = append(uncachedIndices, idx)
		}
	}
	responseIndex := 0

	for _, response := range entityPayloads {
		for _, data := range response.Data {
			if len(data.Features) == 0 {
				responseIndex++
				continue
			}

			if responseIndex >= len(uncachedIndices) {
				return
			}

			resultIndex := uncachedIndices[responseIndex]
			featureStartIdx := response.KeySize
			for i, featureValue := range data.Features[featureStartIdx:] {
				if i < len(rtpFeatureBuilder.Result[resultIndex]) {
					rtpFeatureBuilder.Result[resultIndex][i] = featureValue
				}
			}

			responseIndex++
		}
	}
}

func (rtpbComponent *RealTimePricingFeatureComponent) populateFeatureValuesFromCache(
	modelID string,
	componentConfig *config.ComponentConfig,
	rtpFeatureBuilder *models.RealTimePricingFeatureBuilder,
) {
	cache := inmemorycache.InMemoryCacheInstance
	componentName := rtpbComponent.GetComponentName()

	metricTags := []string{modelId, modelID, component, componentName}
	metrics.Count("inferflow.component.inmemorycache.request.total", int64(len(rtpFeatureBuilder.UniqueEntityIds)), metricTags)

	cacheVersion := componentConfig.CacheVersion
	if cacheVersion == 0 {
		cacheVersion = defaultCacheVersion
	}

	var keyBuilder strings.Builder
	keyBuilder.Grow(len(modelID) + len(cacheKeySeparator)*2 + 10 + len(componentName))
	keyBuilder.WriteString(modelID)
	keyBuilder.WriteString(cacheKeySeparator)
	keyBuilder.WriteString(strconv.Itoa(cacheVersion))
	keyBuilder.WriteString(cacheKeySeparator)
	keyBuilder.WriteString(rtpbComponent.GetComponentName())
	keyBuilder.WriteString(cacheKeySeparator)
	cacheKeyPrefix := keyBuilder.String()

	rtpFeatureBuilder.InMemoryMissCount = 0
	for idx, row := range rtpFeatureBuilder.UniqueEntityIds {
		cacheKey := cacheKeyPrefix + row
		data, err := cache.Get([]byte(cacheKey))
		if err != nil || len(data) == 0 {
			rtpFeatureBuilder.InMemoryMissCount++
			continue
		}
		rtpFeatureBuilder.Result[idx] = rtpbComponent.DecodeBytesArray(data)
		rtpFeatureBuilder.InMemPresent[idx] = true
	}

	if rtpFeatureBuilder.InMemoryMissCount > 0 {
		metrics.Count("inferflow.component.inmemorycache.request.miss.total", int64(rtpFeatureBuilder.InMemoryMissCount), metricTags)
	}
}

func (rtpbComponent *RealTimePricingFeatureComponent) putFeatureValuesIntoCache(
	modelID string,
	rtpFeatureBuilder *models.RealTimePricingFeatureBuilder,
	componentConfig *config.ComponentConfig,
	rtpConfig *config.RealTimePricingFeatureComponentConfig,
) {
	cache := inmemorycache.InMemoryCacheInstance
	if cache == nil {
		return
	}

	compCacheTtl := rtpConfig.CompCacheTtl
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
	entityLabelKeyBuilder.Grow(len(modelID) + len(cacheKeySeparator)*2 + 10 + len(rtpbComponent.GetComponentName()))
	entityLabelKeyBuilder.WriteString(modelID)
	entityLabelKeyBuilder.WriteString(cacheKeySeparator)
	entityLabelKeyBuilder.WriteString(strconv.Itoa(compCacheVersion))
	entityLabelKeyBuilder.WriteString(cacheKeySeparator)
	entityLabelKeyBuilder.WriteString(rtpbComponent.GetComponentName())
	entityLabelKeyBuilder.WriteString(cacheKeySeparator)
	entityLabelKey := entityLabelKeyBuilder.String()

	for rowIdx, row := range rtpFeatureBuilder.UniqueEntityIds {
		cacheKey := entityLabelKey + row
		if rtpFeatureBuilder.InMemPresent[rowIdx] {
			continue
		}
		//Set
		serialized := rtpbComponent.EncodeBytesArray(rtpFeatureBuilder.Result[rtpFeatureBuilder.UniqueEntityIndex[row]])
		if serialized == nil {
			continue
		}
		err := cache.SetEx([]byte(cacheKey), serialized, compCacheTtl)
		if err != nil {
			logger.Error(fmt.Sprintf("Error caching pricing feature values for key: %s", cacheKey), err)
		}
	}
}

func (rtpbComponent *RealTimePricingFeatureComponent) EncodeBytesArray(data [][]byte) []byte {
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

func (rtpbComponent *RealTimePricingFeatureComponent) DecodeBytesArray(data []byte) [][]byte {
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
