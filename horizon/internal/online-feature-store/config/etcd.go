package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	Delimitter           = "|"
	Layout1MetadataBytes = 9
)

type Etcd struct {
	instance etcd.Etcd
	appName  string
	env      string
}

func NewEtcdConfig() Manager {
	return &Etcd{
		instance: etcd.Instance()[viper.GetString("ONLINE_FEATURE_STORE_APP_NAME")],
		appName:  viper.GetString("ONLINE_FEATURE_STORE_APP_NAME"),
		env:      viper.GetString("APP_ENV"),
	}
}

func (e *Etcd) GetEtcdInstance() *FeatureRegistry {
	instance, ok := e.instance.GetConfigInstance().(*FeatureRegistry)
	if !ok {
		log.Panic().Msg("invalid etcd instance")
	}
	return instance
}

func (e *Etcd) RegisterStore(confId int, dbType string, table string, primaryKeys []string, tableTtl int) (map[string]interface{}, error) {
	stores, err := e.GetStores()
	if err != nil {
		return nil, err
	}
	storeId := len(stores) + 1
	storeMap := make(map[string]interface{})
	storeMap["db-type"] = dbType
	storeMap["conf-id"] = confId
	storeMap["table"] = table
	storeMap["max-column-size-in-bytes"] = 1024
	storeMap["max-row-size-in-bytes"] = 102400
	storeMap["primary-keys"] = primaryKeys
	storeMap["table-ttl"] = tableTtl
	for id, store := range stores {
		if store.Table == table {
			return nil, fmt.Errorf("table %s already present in store %v", table, id)
		}
	}
	storeJson, err := json.Marshal(storeMap)
	if err != nil {
		return nil, err
	}

	paths := map[string]interface{}{
		fmt.Sprintf("/config/%s/storage/stores/%v", e.appName, storeId): string(storeJson),
	}
	return paths, nil
}

func (e *Etcd) CreateStore(storeMap map[string]interface{}) error {
	return e.instance.CreateNodes(storeMap)
}

// GetStores retrieves all the available stores from the configuration.
// Returns a map of store names to their associated data or an error if stores are not found.
func (e *Etcd) GetStores() (map[string]Store, error) {
	featureRegistry := e.GetEtcdInstance()
	stores := featureRegistry.Storage.Stores
	if stores == nil {
		return nil, errors.New("stores not found in configuration")
	}
	return stores, nil
}

func (e *Etcd) RegisterEntity(entityLabel string, keyMap map[string]Key, distributedCache Cache, inMemoryCache Cache) error {
	paths := map[string]interface{}{
		fmt.Sprintf("/config/%s/entities/%s/label", e.appName, entityLabel):                               entityLabel,
		fmt.Sprintf("/config/%s/entities/%s/distributed-cache/enabled", e.appName, entityLabel):           distributedCache.Enabled,
		fmt.Sprintf("/config/%s/entities/%s/distributed-cache/ttl-in-seconds", e.appName, entityLabel):    distributedCache.TtlInSeconds,
		fmt.Sprintf("/config/%s/entities/%s/distributed-cache/jitter-percentage", e.appName, entityLabel): distributedCache.JitterPercentage,
		fmt.Sprintf("/config/%s/entities/%s/distributed-cache/conf-id", e.appName, entityLabel):           distributedCache.ConfId,
		fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/enabled", e.appName, entityLabel):             inMemoryCache.Enabled,
		fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/ttl-in-seconds", e.appName, entityLabel):      inMemoryCache.TtlInSeconds,
		fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/jitter-percentage", e.appName, entityLabel):   inMemoryCache.JitterPercentage,
		fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/conf-id", e.appName, entityLabel):             inMemoryCache.ConfId,
	}
	for key, value := range keyMap {
		paths[fmt.Sprintf("/config/%s/entities/%s/keys/%v/sequence", e.appName, entityLabel, key)] = value.Sequence
		paths[fmt.Sprintf("/config/%s/entities/%s/keys/%v/entity-label", e.appName, entityLabel, key)] = value.EntityLabel
		paths[fmt.Sprintf("/config/%s/entities/%s/keys/%v/column-label", e.appName, entityLabel, key)] = value.ColumnLabel
	}
	return e.instance.CreateNodes(paths)
}

func (e *Etcd) EditEntity(entityLabel string, distributedCache Cache, inMemoryCache Cache) error {
	entityExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s", e.appName, entityLabel))
	if !entityExists {
		return fmt.Errorf("entity %s not found", entityLabel)
	}
	paths := map[string]interface{}{
		fmt.Sprintf("/config/%s/entities/%s/label", e.appName, entityLabel): entityLabel,
	}
	paths[fmt.Sprintf("/config/%s/entities/%s/distributed-cache/enabled", e.appName, entityLabel)] = distributedCache.Enabled
	paths[fmt.Sprintf("/config/%s/entities/%s/distributed-cache/ttl-in-seconds", e.appName, entityLabel)] = distributedCache.TtlInSeconds
	paths[fmt.Sprintf("/config/%s/entities/%s/distributed-cache/jitter-percentage", e.appName, entityLabel)] = distributedCache.JitterPercentage
	paths[fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/enabled", e.appName, entityLabel)] = inMemoryCache.Enabled
	paths[fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/ttl-in-seconds", e.appName, entityLabel)] = inMemoryCache.TtlInSeconds
	paths[fmt.Sprintf("/config/%s/entities/%s/in-memory-cache/jitter-percentage", e.appName, entityLabel)] = inMemoryCache.JitterPercentage
	return e.instance.SetValues(paths)
}

func (e *Etcd) RegisterFeatureGroup(entityLabel, fgLabel, JobId string, storeId, ttlInSeconds int, inMemoryCacheEnabled, distributedCacheEnabled bool, dataType enums.DataType, featureLabels, featureDefaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string, layoutVersion int) ([]string, Store, map[string]interface{}, error) {
	// check if entity,fg,store,job exists
	entityExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s", e.appName, entityLabel))
	if !entityExists {
		return nil, Store{}, nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	fgExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s", e.appName, entityLabel, fgLabel))
	if fgExists {
		return nil, Store{}, nil, fmt.Errorf("feature group %s is already present", fgLabel)
	}
	storeExists, _ := e.instance.IsLeafNodeExist(fmt.Sprintf("/config/%s/storage/stores/%v", e.appName, storeId))
	if !storeExists {
		return nil, Store{}, nil, fmt.Errorf("store id %v not found", storeId)
	}
	jobExists, _ := e.instance.IsLeafNodeExist(fmt.Sprintf("/config/%s/security/%s/%s", e.appName, "writer", JobId))
	if !jobExists {
		return nil, Store{}, nil, fmt.Errorf("job id %s not found", JobId)
	}
	if dataType.String() == "Unknown" {
		return nil, Store{}, nil, fmt.Errorf("invalid data type")
	}
	// Check for duplicates in the input feature labels
	seen := make(map[string]struct{}, len(featureLabels))
	for _, label := range featureLabels {
		if _, exists := seen[label]; exists {
			return nil, Store{}, nil, fmt.Errorf("duplicate feature label in input: '%s'", label)
		}
		seen[label] = struct{}{}
	}
	totalSize := len(featureLabels) * dataType.Size()
	if dataType == "DataTypeString" {
		var totalStringLength uint16
		for i := range stringLength {
			stringLengthUint16, _ := stringToUint16(stringLength[i])
			totalStringLength += stringLengthUint16
		}
		totalSize = int(totalStringLength) * dataType.Size()
	} else if dataType.IsVector() {
		var totalVectorLength uint16
		for i := range vectorLength {
			vectorLengthUint16, _ := stringToUint16(vectorLength[i])
			totalVectorLength += vectorLengthUint16
		}
		totalSize = int(totalVectorLength) * dataType.Size()
	}
	if dataType == "DataTypeStringVector" {
		var stringVectorSize uint16
		for i := range stringLength {
			stringLengthUint16, _ := stringToUint16(stringLength[i])
			vectorLengthUint16, _ := stringToUint16(vectorLength[i])
			stringVectorSize += stringLengthUint16 * vectorLengthUint16
		}
		totalSize = int(stringVectorSize) * dataType.Size()
	}
	metadataSize, err := getMetadataSizeForLayout(layoutVersion)
	if err != nil {
		return nil, Store{}, nil, err
	}
	totalSize = totalSize + metadataSize

	stores, err := e.GetStores()
	if err != nil {
		return nil, Store{}, nil, err
	}
	entityKeys, err := e.GetEntityKeys(entityLabel)
	if err != nil {
		return nil, Store{}, nil, err
	}
	entityKeyLabels := make([]string, 0, len(entityKeys))
	for _, key := range entityKeys {
		entityKeyLabels = append(entityKeyLabels, key.ColumnLabel)
	}
	store := stores[strconv.Itoa(storeId)]

	if len(entityKeyLabels) != len(store.PrimaryKeys) {
		return nil, Store{}, nil, fmt.Errorf("entity keys and store primary keys length mismatch")
	}

	for _, primaryKey := range store.PrimaryKeys {
		found := false
		for _, entityKeyLabel := range entityKeyLabels {
			if primaryKey == entityKeyLabel {
				found = true
				break
			}
		}
		if !found {
			return nil, Store{}, nil, fmt.Errorf("primary key %s not found in entity keys", primaryKey)
		}
	}

	maxColumnSize := stores[strconv.Itoa(storeId)].MaxColumnSizeInBytes
	numColumns := totalSize / maxColumnSize
	if totalSize%maxColumnSize != 0 {
		numColumns++
	}
	paths := make(map[string]interface{})
	maxColumn := e.getMaxColumnForEntity(entityLabel) + 1
	columnsToAdd := make([]string, 0)
	for i := 0; i < numColumns; i++ {
		columnLabel := fmt.Sprintf("seg_%d", maxColumn+i)
		columnSize := maxColumnSize
		if i == numColumns-1 {
			columnSize = totalSize - (maxColumnSize * (numColumns - 1))
		}
		paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/columns/%s/label", e.appName, entityLabel, fgLabel, columnLabel)] = columnLabel
		paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/columns/%s/current-size-in-bytes", e.appName, entityLabel, fgLabel, columnLabel)] = columnSize
		columnsToAdd = append(columnsToAdd, columnLabel)
	}
	fgId, err := e.GetFgIdForNewFeatureGroup(entityLabel)
	if err != nil {
		return nil, Store{}, nil, err
	}

	labels := strings.Join(featureLabels, ",")
	featureMetaMap := make(map[string]FeatureMeta)
	sourceMap := make(map[string]string)
	for i, featureLabel := range featureLabels {
		defaultValueInByte, _ := Serialize(featureDefaultValues[i], dataType)
		stringLengthUint16, err := stringToUint16(stringLength[i])
		if err != nil {
			return nil, Store{}, nil, err
		}
		vectorLengthUint16, err := stringToUint16(vectorLength[i])
		if err != nil {
			return nil, Store{}, nil, err
		}
		featureMeta := FeatureMeta{
			Sequence:             i,
			DefaultValuesInBytes: defaultValueInByte,
			StringLength:         stringLengthUint16,
			VectorLength:         vectorLengthUint16,
		}
		featureMetaMap[featureLabel] = featureMeta
		if storageProvider[i] != "" {
			sourceMap[entityLabel+Delimitter+fgLabel+Delimitter+featureLabel] =
				storageProvider[i] + Delimitter + sourceBasePath[i] + Delimitter + sourceDataPath[i] + Delimitter + featureDefaultValues[i]
		}
	}
	featureMetaJson, err := json.Marshal(featureMetaMap)
	if err != nil {
		return nil, Store{}, nil, err
	}
	featureDefaultValues, err = processFeatureDefaultValues(featureDefaultValues, dataType.String())
	if err != nil {
		return nil, Store{}, nil, err
	}
	defaultValues := strings.Join(featureDefaultValues, ",")
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/id", e.appName, entityLabel, fgLabel)] = fgId
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/store-id", e.appName, entityLabel, fgLabel)] = storeId
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/data-type", e.appName, entityLabel, fgLabel)] = dataType.String()
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/ttl-in-seconds", e.appName, entityLabel, fgLabel)] = ttlInSeconds
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/job-id", e.appName, entityLabel, fgLabel)] = JobId
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/in-memory-cache-enabled", e.appName, entityLabel, fgLabel)] = inMemoryCacheEnabled
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/distributed-cache-enabled", e.appName, entityLabel, fgLabel)] = distributedCacheEnabled
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/active-version", e.appName, entityLabel, fgLabel)] = 1
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/feature-meta", e.appName, entityLabel, fgLabel, 1)] = string(featureMetaJson)
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/labels", e.appName, entityLabel, fgLabel, 1)] = labels
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/default-values", e.appName, entityLabel, fgLabel, 1)] = defaultValues
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/layout-version", e.appName, entityLabel, fgLabel)] = layoutVersion
	for key, value := range sourceMap {
		paths[fmt.Sprintf("/config/%s/source/%s", e.appName, key)] = value
	}
	return columnsToAdd, stores[strconv.Itoa(storeId)], paths, nil
}

func (e *Etcd) CreateFeatureGroup(paths map[string]interface{}) error {
	return e.instance.CreateNodes(paths)
}

func (e *Etcd) EditFeatureGroup(entityLabel, fgLabel string, ttlInSeconds int, inMemoryCacheEnabled, distributedCacheEnabled bool, layoutVersion int) error {
	entityExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s", e.appName, entityLabel))
	if !entityExists {
		return fmt.Errorf("entity %s not found", entityLabel)
	}
	fgExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s", e.appName, entityLabel, fgLabel))
	if !fgExists {
		return fmt.Errorf("feature group %s not found", fgLabel)
	}
	paths := make(map[string]interface{})
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/ttl-in-seconds", e.appName, entityLabel, fgLabel)] = ttlInSeconds
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/in-memory-cache-enabled", e.appName, entityLabel, fgLabel)] = inMemoryCacheEnabled
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/distributed-cache-enabled", e.appName, entityLabel, fgLabel)] = distributedCacheEnabled
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/layout-version", e.appName, entityLabel, fgLabel)] = layoutVersion
	return e.instance.SetValues(paths)
}

func (e *Etcd) GetAllEntities() ([]string, error) {
	featureRegistry := e.GetEtcdInstance()
	entities := featureRegistry.Entities
	if entities == nil {
		return []string{}, errors.New("entities not found in configuration")
	}
	keys := make([]string, 0, len(entities))
	for key := range entities {
		keys = append(keys, key)
	}
	return keys, nil
}

func (e *Etcd) GetFgIdForNewFeatureGroup(entityLabel string) (int, error) {
	featureRegistry := e.GetEtcdInstance()
	entities := featureRegistry.Entities
	if entities == nil {
		return -1, errors.New("entities not found in configuration")
	}

	// Look up the entity by the given label
	entity, exists := entities[entityLabel]
	if !exists {
		return -1, fmt.Errorf("entity %s not found", entityLabel)
	}

	// Check if the entity has FeatureGroups
	featureGroups := entity.FeatureGroups
	if featureGroups == nil {
		return -1, fmt.Errorf("no feature groups found for entity %s", entityLabel)
	}

	// Collect all feature group keys for the specific entity
	fgIds := make([]int, 0)
	for _, fg := range featureGroups {
		fgIds = append(fgIds, fg.Id)
	}
	maxFgId := -1
	for _, fgId := range fgIds {
		if fgId > maxFgId {
			maxFgId = fgId
		}
	}
	maxFgId = maxFgId + 1
	return maxFgId, nil
}

func (e *Etcd) GetJobs(jobType string) ([]string, error) {
	featureRegistry := e.GetEtcdInstance()

	// Check if jobType is "writer"
	if jobType == "writer" {
		// If Security.Writer is a map, extract all keys
		writerKeys := make([]string, 0, len(featureRegistry.Security.Writer))
		for key := range featureRegistry.Security.Writer {
			writerKeys = append(writerKeys, key)
		}
		return writerKeys, nil
	} else if jobType == "reader" {
		// If Security.Reader is a map, extract all keys
		readerKeys := make([]string, 0, len(featureRegistry.Security.Reader))
		for key := range featureRegistry.Security.Reader {
			readerKeys = append(readerKeys, key)
		}
		return readerKeys, nil
	}
	return nil, fmt.Errorf("invalid job type: %s", jobType)
}

func (e *Etcd) GetJobsToken(jobType, jobId string) (string, error) {
	featureRegistry := e.GetEtcdInstance()

	// Check if jobType is "writer"
	if jobType == "writer" {
		for key, val := range featureRegistry.Security.Writer {
			if key == jobId {
				return val.Token, nil
			}
		}
	} else if jobType == "reader" {
		for key, val := range featureRegistry.Security.Reader {
			if key == jobId {
				return val.Token, nil
			}
		}
	}
	return "", fmt.Errorf("invalid job type: %s and job id: %s", jobType, jobId)
}

func (e *Etcd) AddFeatures(entityLabel, fgLabel string, labels, defaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string) ([]string, Store, map[string]interface{}, map[string]interface{}, error) {
	// check if entity,fg exists
	entityExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s", e.appName, entityLabel))
	if !entityExists {
		return nil, Store{}, nil, nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	fgExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s", e.appName, entityLabel, fgLabel))
	if !fgExists {
		return nil, Store{}, nil, nil, fmt.Errorf("feature group %s not found", entityLabel)
	}

	// Check for duplicates in the input feature labels
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		if _, exists := seen[label]; exists {
			return nil, Store{}, nil, nil, fmt.Errorf("duplicate feature label in input: '%s'", label)
		}
		seen[label] = struct{}{}
	}

	// if yes, process logic to add features
	entities := e.GetEtcdInstance().Entities
	featureGroup := entities[entityLabel].FeatureGroups[fgLabel]
	columns := featureGroup.Columns
	maxSegmentInFg, err := findLargestSegment(columns)
	maxSegmentInEntity := e.getMaxColumnForEntity(entityLabel) + 1
	columnToUpdate := "seg_" + strconv.Itoa(maxSegmentInFg)
	dataType := featureGroup.DataType
	if err != nil {
		log.Error().Msgf("Error finding largest segment: %s", err)
		return nil, Store{}, nil, nil, err
	}
	activeVersion := featureGroup.ActiveVersion
	existingLabels := featureGroup.Features[activeVersion].Labels
	featureMetaMap := featureGroup.Features[activeVersion].FeatureMeta
	existingDefaultValues := featureGroup.Features[activeVersion].DefaultValues
	newLabels := strings.Join(labels, ",")
	existingLabelsList := strings.Split(existingLabels, ",")
	existingLabelsSet := make(map[string]struct{}, len(existingLabelsList))
	for _, l := range existingLabelsList {
		existingLabelsSet[l] = struct{}{}
	}
	// Check if the new labels already exist in the existing labels
	for _, featureLabel := range labels {
		if _, exists := existingLabelsSet[featureLabel]; exists {
			return nil, Store{}, nil, nil, fmt.Errorf("feature label '%s' already exists", featureLabel)
		}
	}
	if existingLabels != "" {
		existingLabels = existingLabels + "," + newLabels
	} else {
		existingLabels = newLabels
	}
	processedDefaultValues, _ := processFeatureDefaultValues(defaultValues, featureGroup.DataType.String())
	newDefaultValues := strings.Join(processedDefaultValues, ",")
	if existingDefaultValues != "" {
		existingDefaultValues = existingDefaultValues + "," + newDefaultValues
	} else {
		existingDefaultValues = newDefaultValues
	}
	featureSequenceCurrentSize := len(featureMetaMap)
	for _, featureLabel := range labels {
		if _, exists := featureMetaMap[featureLabel]; exists {
			return nil, Store{}, nil, nil, fmt.Errorf("feature label %s already exists", featureLabel)
		}
	}
	sourceMap := make(map[string]string)
	for i, featureLabel := range labels {
		defaultValueInByte, err := Serialize(defaultValues[i], dataType)
		if err != nil {
			return nil, Store{}, nil, nil, err
		}
		stringLengthUint16, err := stringToUint16(stringLength[i])
		if err != nil {
			return nil, Store{}, nil, nil, err
		}
		vectorLengthUint16, err := stringToUint16(vectorLength[i])
		if err != nil {
			return nil, Store{}, nil, nil, err
		}
		featureMeta := FeatureMeta{
			Sequence:             featureSequenceCurrentSize + i,
			DefaultValuesInBytes: defaultValueInByte,
			StringLength:         stringLengthUint16,
			VectorLength:         vectorLengthUint16,
		}
		featureMetaMap[featureLabel] = featureMeta
		if storageProvider[i] != "" {
			sourceMap[entityLabel+Delimitter+fgLabel+Delimitter+featureLabel] =
				storageProvider[i] + Delimitter + sourceBasePath[i] + Delimitter + sourceDataPath[i] + Delimitter + defaultValues[i]
		}
	}
	featureMetaJson, err := json.Marshal(featureMetaMap)
	if err != nil {
		return nil, Store{}, nil, nil, err
	}
	totalSize := len(labels) * dataType.Size()
	if dataType == "DataTypeString" {
		var totalStringLength uint16
		for i := range stringLength {
			stringLengthUint16, _ := stringToUint16(stringLength[i])
			totalStringLength += stringLengthUint16
		}
		totalSize = int(totalStringLength) * dataType.Size()
	} else if dataType.IsVector() {
		var totalVectorLength uint16
		for i := range vectorLength {
			vectorLengthUint16, _ := stringToUint16(vectorLength[i])
			totalVectorLength += vectorLengthUint16
		}
		totalSize = int(totalVectorLength) * dataType.Size()
	}
	if dataType == "DataTypeStringVector" {
		var stringVectorSize uint16
		for i := range stringLength {
			stringLengthUint16, _ := stringToUint16(stringLength[i])
			vectorLengthUint16, _ := stringToUint16(vectorLength[i])
			stringVectorSize += stringLengthUint16 * vectorLengthUint16
		}
		totalSize = int(stringVectorSize) * dataType.Size()
	}
	newSize := featureGroup.Columns[columnToUpdate].CurrentSizeInBytes + totalSize
	storeId := featureGroup.StoreId
	stores, err := e.GetStores()
	if err != nil {
		return nil, Store{}, nil, nil, err
	}
	paths := make(map[string]interface{})
	activeVersionInt, _ := strconv.Atoi(activeVersion)
	pathsToUpdate := make(map[string]interface{})
	columnsToAdd := make([]string, 0)
	if newSize > stores[storeId].MaxColumnSizeInBytes {
		pathsToUpdate[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/columns/%s/current-size-in-bytes", e.appName, entityLabel, fgLabel, columnToUpdate)] = stores[storeId].MaxColumnSizeInBytes
		columnsLimit := maxSegmentInEntity + newSize/stores[storeId].MaxColumnSizeInBytes
		if newSize%stores[storeId].MaxColumnSizeInBytes == 0 {
			columnsLimit--
		}
		for i := maxSegmentInEntity + 1; i <= columnsLimit; i++ {
			columnLabel := fmt.Sprintf("seg_%d", i)
			columnSize := stores[storeId].MaxColumnSizeInBytes
			if i == columnsLimit {
				columnSize = newSize - (stores[storeId].MaxColumnSizeInBytes * (columnsLimit - maxSegmentInEntity))
				if columnSize == 0 {
					break
				}
			}
			paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/columns/%s/label", e.appName, entityLabel, fgLabel, columnLabel)] = columnLabel
			paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/columns/%s/current-size-in-bytes", e.appName, entityLabel, fgLabel, columnLabel)] = columnSize
			columnsToAdd = append(columnsToAdd, columnLabel)
		}
	}
	activeVersionInt = activeVersionInt + 1
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/feature-meta", e.appName, entityLabel, fgLabel, activeVersionInt)] = string(featureMetaJson)
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/labels", e.appName, entityLabel, fgLabel, activeVersionInt)] = existingLabels
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/default-values", e.appName, entityLabel, fgLabel, activeVersionInt)] = existingDefaultValues

	currentColumnKey := fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/columns/%s/current-size-in-bytes", e.appName, entityLabel, fgLabel, columnToUpdate)
	if _, exists := pathsToUpdate[currentColumnKey]; !exists { // for the cases where newSize <= MaxColumnSizeInBytes
		pathsToUpdate[currentColumnKey] = newSize
	}
	pathsToUpdate[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/active-version", e.appName, entityLabel, fgLabel)] = activeVersionInt
	for key, value := range sourceMap {
		paths[fmt.Sprintf("/config/%s/source/%s", e.appName, key)] = value
	}

	return columnsToAdd, stores[storeId], pathsToUpdate, paths, nil
}

func (e *Etcd) CreateAddFeaturesNodes(paths map[string]interface{}, pathsToUpdate map[string]interface{}) error {
	err := e.instance.CreateNodes(paths)
	if err != nil {
		log.Error().Msgf("Error Creating Nodes: %s", err)
	}
	err = e.instance.SetValues(pathsToUpdate)
	if err != nil {
		log.Error().Msgf("Error Setting Values: %s", err)
	}
	return nil
}

func (e *Etcd) EditFeatures(entityLabel, fgLabel string, featureLabels, defaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string) error {
	entityExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s", e.appName, entityLabel))
	if !entityExists {
		return fmt.Errorf("entity %s not found", entityLabel)
	}
	fgExists, _ := e.instance.IsNodeExist(fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s", e.appName, entityLabel, fgLabel))
	if !fgExists {
		return fmt.Errorf("feature group %s not found", fgLabel)
	}
	sourceMap := make(map[string]string)
	// Re-Add all offline source mapping for the featureLabels which are editable
	for i, featureLabel := range featureLabels {
		sourceMap[entityLabel+Delimitter+fgLabel+Delimitter+featureLabel] =
			storageProvider[i] + Delimitter + sourceBasePath[i] + Delimitter + sourceDataPath[i] + Delimitter + defaultValues[i]
	}
	fg, err := e.GetFeatureGroup(entityLabel, fgLabel)
	if err != nil {
		return err
	}
	currentActiveVersion := fg.ActiveVersion
	featureMetaMap := fg.Features[currentActiveVersion].FeatureMeta
	existingLabels := fg.Features[currentActiveVersion].Labels
	existingDefaultValues := fg.Features[currentActiveVersion].DefaultValues
	// Convert existingLabels to []string
	existingLabelsSlice := strings.Split(existingLabels, ",")
	// Convert existingDefaultValues to []string
	existingDefaultValuesSlice := strings.Split(existingDefaultValues, ",")
	if fg.DataType.IsVector() {
		existingDefaultValuesSlice = strings.Split(existingDefaultValues, "],")
	}
	for i, featureLabel := range featureLabels {
		// Update featureMetaMap with new feature metadata
		stringLengthUint16, err := stringToUint16(stringLength[i])
		if err != nil {
			return err
		}
		vectorLengthUint16, err := stringToUint16(vectorLength[i])
		if err != nil {
			return err
		}
		defaultValueInByte, _ := Serialize(defaultValues[i], fg.DataType)
		currentSequence := featureMetaMap[featureLabel].Sequence
		featureMetaMap[featureLabel] = FeatureMeta{
			Sequence:             currentSequence,
			StringLength:         stringLengthUint16,
			VectorLength:         vectorLengthUint16,
			DefaultValuesInBytes: defaultValueInByte,
		}
		// Find the index of the featureLabel in existingLabelsSlice
		index := -1
		for j, label := range existingLabelsSlice {
			if label == featureLabel {
				index = j
				break
			}
		}
		// If the featureLabel is found, update the corresponding default value
		if index != -1 {
			existingDefaultValuesSlice[index] = defaultValues[i]
		}
	}
	featureMetaJson, err := json.Marshal(featureMetaMap)
	if err != nil {
		return err
	}
	activeVersionInt, _ := strconv.Atoi(currentActiveVersion)
	// Convert existingDefaultValuesSlice back to a comma-separated string
	updatedDefaultValues := strings.Join(existingDefaultValuesSlice, ",")
	paths := make(map[string]interface{})

	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/feature-meta", e.appName, entityLabel, fgLabel, activeVersionInt)] = string(featureMetaJson)
	paths[fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%v/default-values", e.appName, entityLabel, fgLabel, activeVersionInt)] = updatedDefaultValues
	for key, value := range sourceMap {
		paths[fmt.Sprintf("/config/%s/source/%s", e.appName, key)] = value
	}
	e.instance.SetValues(paths)
	return nil
}

func (e *Etcd) RegisterJob(jobType, jobId, token string) error {
	job, err := e.GetJob(jobType)
	if err != nil {
		log.Error().Msgf("Error while getting job")
		return err
	}
	if _, exists := job[jobId]; exists {
		return fmt.Errorf("job %s already exists", jobId)
	}
	jobMap := make(map[string]string)
	jobMap["token"] = token
	jobJson, err := json.Marshal(jobMap)
	if err != nil {
		log.Error().Msgf("Error while marshalling property")
		return err
	}
	paths := map[string]interface{}{
		fmt.Sprintf("/config/%s/security/%s/%s", e.appName, jobType, jobId): string(jobJson),
	}
	return e.instance.CreateNodes(paths)
}

// GetEntities retrieves all entities from the configuration.
// Returns a map of entity names to their models or an error if entities are not found.
func (e *Etcd) GetEntities() (map[string]Entity, error) {
	featureRegistry := e.GetEtcdInstance()
	entities := featureRegistry.Entities
	if entities == nil {
		return nil, errors.New("entities not found in configuration")
	}
	return entities, nil
}

func (e *Etcd) GetEntityKeys(entityLabel string) (map[string]Key, error) {
	entities, err := e.GetEntities()
	if err != nil {
		return nil, err
	}
	return entities[entityLabel].Keys, nil
}

func (e *Etcd) GetFeatureGroups(entityLabel string) (map[string]FeatureGroup, error) {
	entities, err := e.GetEntities()
	if err != nil {
		return nil, err
	}
	entity, exists := entities[entityLabel]
	if !exists {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	return entity.FeatureGroups, nil
}

func (e *Etcd) GetFeatureGroup(entityLabel, fgLabel string) (*FeatureGroup, error) {
	entities, err := e.GetEntities()
	if err != nil {
		return nil, err
	}
	entity, exists := entities[entityLabel]
	if !exists {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	fg, exists := entity.FeatureGroups[fgLabel]
	if !exists {
		return nil, fmt.Errorf("feature group %s not found", fgLabel)
	}
	return &fg, nil
}

func (e *Etcd) GetSource() (map[string]string, error) {
	featureRegistry := e.GetEtcdInstance()
	return featureRegistry.Source, nil
}

func (e *Etcd) GetJob(jobType string) (map[string]Property, error) {
	if jobType != "writer" && jobType != "reader" {
		return nil, fmt.Errorf("invalid Job Type")
	}
	featureRegistry := e.GetEtcdInstance()
	security := featureRegistry.Security
	if jobType == "writer" {
		return security.Writer, nil
	}
	return security.Reader, nil
}

func findLargestSegment(columns map[string]Column) (int, error) {
	var largestKey string
	maxSegment := -1

	for key, column := range columns {
		// Ensure the value starts with "seg_"
		if !strings.HasPrefix(column.Label, "seg_") {
			return -1, fmt.Errorf("invalid format for value: %s", column.Label)
		}

		// Extract the numeric part after "seg_"
		numStr := column.Label[4:]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return -1, fmt.Errorf("error parsing numeric part of value %s: %v", column.Label, err)
		}

		// Compare and update the largest segment found
		if num > maxSegment {
			maxSegment = num
			largestKey = key
		}
	}

	if largestKey == "" {
		return -1, fmt.Errorf("no valid segments found")
	}

	return maxSegment, nil
}

func (e *Etcd) getMaxColumnForEntity(entityLabel string) int {
	entities, err := e.GetEntities()
	maxColumnUsed := -1
	if err != nil {
		log.Error().Msgf("Error getting entities: %s", err)
		return maxColumnUsed
	}
	entityConf, exists := entities[entityLabel]
	if !exists || entityConf.FeatureGroups == nil {
		return maxColumnUsed
	}
	for _, featureGroupConf := range entityConf.FeatureGroups {
		for _, featureColumnConf := range featureGroupConf.Columns {
			if strings.HasPrefix(featureColumnConf.Label, "seg_") {
				segment := featureColumnConf.Label[4:]
				if value, err := strconv.Atoi(segment); err == nil {
					if value > maxColumnUsed {
						maxColumnUsed = value
					}
				}
			}
		}
	}
	return maxColumnUsed
}

func processFeatureDefaultValues(featureDefaultValues []string, dataType string) ([]string, error) {
	// Check if dataType contains "vector"
	containsVector := strings.Contains(strings.ToLower(dataType), "vector")

	var formattedValues []string

	for _, value := range featureDefaultValues {
		// Convert to vector format if "vector" is in dataType
		if containsVector {
			if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
				value = "[" + value + "]"
			}
		}

		formattedValues = append(formattedValues, value)
	}

	return formattedValues, nil
}

func stringToUint16(s string) (uint16, error) {
	// Parse the string as an unsigned integer with base 10 and 16-bit size
	val, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}

	// Convert the uint64 result to uint16
	return uint16(val), nil
}

func (e *Etcd) GetAllFeatureGroupByEntityLabel(entityLabel string) ([]string, error) {
	featureRegistry := e.GetEtcdInstance()
	entities := featureRegistry.Entities
	if entities == nil {
		return nil, errors.New("entities not found in configuration")
	}

	// Look up the entity by the given label
	entity, exists := entities[entityLabel]
	if !exists {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}

	// Check if the entity has FeatureGroups
	featureGroups := entity.FeatureGroups
	if featureGroups == nil {
		return nil, fmt.Errorf("no feature groups found for entity %s", entityLabel)
	}

	// Collect all feature group keys for the specific entity
	featureGroupKeys := make([]string, 0, len(featureGroups))
	for key := range featureGroups {
		featureGroupKeys = append(featureGroupKeys, key)
	}

	return featureGroupKeys, nil
}

func getMetadataSizeForLayout(version int) (int, error) {

	switch version {
	case 1:
		return Layout1MetadataBytes, nil
	default:
		return 0, fmt.Errorf("unsupported layout version: %d", version)

	}

}
