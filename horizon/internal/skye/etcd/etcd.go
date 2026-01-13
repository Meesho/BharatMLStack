package etcd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/skye"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/rs/zerolog/log"
)

type Etcd struct {
	instance etcd.Etcd
	appName  string
	env      string
}

func NewEtcdInstance() *Etcd {
	return &Etcd{
		instance: etcd.Instance()[skye.SkyeAppName],
		appName:  skye.SkyeAppName,
		env:      skye.AppEnv,
	}
}

// Store Configuration Structures
type StoreConfig struct {
	ConfID          int    `json:"conf_id"`
	DB              string `json:"db"`
	EmbeddingsTable string `json:"embeddings_table"`
	AggregatorTable string `json:"aggregator_table"`
}

// Entity Configuration Structures
type EntityConfig struct {
	Entity  string `json:"entity"`
	StoreID string `json:"store_id"`
}

// Model Configuration Structures
type ModelConfig struct {
	Entity                string             `json:"entity"`
	Model                 string             `json:"model"`
	EmbeddingStoreEnabled bool               `json:"embedding_store_enabled"`
	EmbeddingStoreVersion int                `json:"embedding_store_version"`
	EmbeddingStoreTTL     int                `json:"embedding_store_ttl"`
	ModelConfigDetails    ModelConfigDetails `json:"model_config"`
	ModelType             string             `json:"model_type"`
	MQID                  int                `json:"mq_id"`
	JobFrequency          string             `json:"job_frequency"`
	TrainingDataPath      string             `json:"training_data_path"`
	NumberOfPartitions    int                `json:"number_of_partitions"`
	TopicName             string             `json:"topic_name"`
	Metadata              string             `json:"metadata"`
	FailureProducerMqId   int                `json:"failure_producer_mq_id"`
	PartitionStatus       map[string]int     `json:"partition_status"`
}

type ModelConfigDetails struct {
	DistanceFunction string `json:"distance_function"`
	VectorDimension  int    `json:"vector_dimension"`
}

// Variant Configuration Structures
type VariantConfig struct {
	Entity                              string               `json:"entity"`
	Model                               string               `json:"model"`
	Variant                             string               `json:"variant"`
	VariantState                        string               `json:"variant_state"`
	VectorDBType                        string               `json:"vector_db_type"`
	Type                                string               `json:"type"`
	CachingConfiguration                CachingConfiguration `json:"caching_configuration"`
	FilterConfiguration                 FilterConfiguration  `json:"filter_configuration"`
	VectorDBConfig                      VectorDBConfig       `json:"vector_db_config"`
	EmbeddingStoreReadVersion           int                  `json:"embedding_store_read_version"`
	EmbeddingStoreWriteVersion          int                  `json:"embedding_store_write_version"`
	VectorDBReadVersion                 int                  `json:"vector_db_read_version"`
	VectorDBWriteVersion                int                  `json:"vector_db_write_version"`
	Enabled                             bool                 `json:"enabled"`
	RateLimiter                         RateLimiter          `json:"rate_limiter"`
	RTPartition                         int                  `json:"rt_partition"`
	Onboarded                           bool                 `json:"onboarded"`
	PartialHitEnabled                   bool                 `json:"partial_hit_enabled"`
	RTDeltaProcessing                   bool                 `json:"rt_delta_processing"`
	BackupConfig                        BackupConfig         `json:"backup_config"`
	DefaultResponsePercentage           int                  `json:"default_response_percentage"`
	PartialHitDisabled                  bool                 `json:"partial_hit_disabled"`
	TestConfig                          TestConfig           `json:"test_config"`
	EmbeddingRetrievalInMemoryConfig    Config               `json:"embedding_retrieval_in_memory_config"`
	EmbeddingRetrievalDistributedConfig Config               `json:"embedding_retrieval_distributed_config"`
	DotProductInMemoryConfig            Config               `json:"dot_product_in_memory_config"`
	DotProductDistributedConfig         Config               `json:"dot_product_distributed_config"`
}

type CachingConfiguration struct {
	InMemoryCachingEnabled     bool `json:"in_memory_caching_enabled"`
	InMemoryCacheTTLSeconds    int  `json:"in_memory_cache_ttl_seconds"`
	DistributedCachingEnabled  bool `json:"distributed_caching_enabled"`
	DistributedCacheTTLSeconds int  `json:"distributed_cache_ttl_seconds"`
}

type FilterConfiguration struct {
	Criteria []FilterCriteria `json:"criteria"`
}

type FilterCriteria struct {
	ColumnName   string `json:"column_name"`
	FilterValue  string `json:"filter_value"`
	DefaultValue string `json:"default_value"`
}

// VectorDBConfig can be any JSON structure - different vector DBs have different requirements
type VectorDBConfig map[string]interface{}

type RateLimiter struct {
	RateLimit  int `json:"rate_limit"`
	BurstLimit int `json:"burst_limit"`
}

type BackupConfig struct {
	Enabled          bool   `json:"enabled"`
	RoutePercentage  int    `json:"route_percentage"`
	DualWriteEnabled bool   `json:"dual_write_enabled"`
	Host             string `json:"host"`
	Version          int    `json:"version"`
}

type TestConfig struct {
	VectorDbType string `json:"vector_db_type"`
	Percentage   int    `json:"percentage"`
	Entity       string `json:"entity"`
	Model        string `json:"model"`
	Variant      string `json:"variant"`
	Version      int    `json:"version"`
}

type Config struct {
	Enabled bool `json:"enabled"`
	TTL     int  `json:"ttl"`
}

// HTTP2Config and VectorDBParams are no longer needed since VectorDBConfig is now flexible

// Filter Configuration Structures
type FilterConfig struct {
	Entity       string `json:"entity"` // Entity-level filters
	ColumnName   string `json:"column_name"`
	FilterValue  string `json:"filter_value"`
	DefaultValue string `json:"default_value"`
}

// Job Frequency Configuration Structures
type JobFrequencyConfig struct {
	JobFrequency string `json:"job_frequency"`
	Description  string `json:"description"`
}

// Qdrant Cluster Configuration Structures
type QdrantClusterConfig struct {
	NodeConf      NodeConfiguration `json:"node_conf"`
	QdrantVersion string            `json:"qdrant_version"`
	DNSSubdomain  string            `json:"dns_subdomain"`
	Project       string            `json:"project"`
}

type NodeConfiguration struct {
	Count        int    `json:"count"`
	InstanceType string `json:"instance_type"`
	Storage      string `json:"storage"`
}

// Variant Promotion Configuration
type VariantPromotionConfig struct {
	Entity       string `json:"entity"`
	Model        string `json:"model"`
	VectorDBType string `json:"vector_db_type"`
	Variant      string `json:"variant"`
	Host         string `json:"host"`
}

// Variant Onboarding Configuration
type VariantOnboardingConfig struct {
	Entity       string `json:"entity"`
	Model        string `json:"model"`
	Variant      string `json:"variant"`
	VectorDBType string `json:"vector_db_type"`
}

// ====================  STORE ETCD OPERATIONS ====================

func (e *Etcd) CreateStoreConfig(storeId string, storeConfig StoreConfig) error {
	// Log the input parameters
	log.Info().Msgf("CreateStoreConfig called with storeId: %s", storeId)
	log.Info().Msgf("Store config data: %+v", storeConfig)

	// Check if ETCD instance is available
	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}
	log.Info().Msg("ETCD instance is available")

	// Log environment info
	log.Info().Msgf("ETCD client appName: %s, env: %s", e.appName, e.env)
	configJson, err := json.Marshal(storeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal store config: %w", err)
	}

	path := fmt.Sprintf("/config/%s/storage/stores/%s", e.appName, storeId)
	log.Info().Msgf("ETCD path to create: %s", path)

	// Attempt to create the node
	log.Info().Msg("Calling ETCD CreateNode...")
	err = e.instance.CreateNode(path, string(configJson))

	if err != nil {
		log.Error().Err(err).Msgf("Failed to create ETCD node at path: %s", path)
		return fmt.Errorf("failed to create ETCD node at %s: %w", path, err)
	}

	log.Info().Msgf("Successfully created ETCD store config at path: %s", path)
	return nil
}

func (e *Etcd) UpdateStoreConfig(storeId string, storeConfig StoreConfig) error {
	configJson, err := json.Marshal(storeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal store config: %w", err)
	}

	path := fmt.Sprintf("/config/%s/storage/stores/%s", e.appName, storeId)
	return e.instance.SetValue(path, string(configJson))
}

func (e *Etcd) DeleteStoreConfig(storeId string) error {
	path := fmt.Sprintf("/config/%s/storage/stores/%s", e.appName, storeId)
	return e.instance.DeleteNode(path)
}

func (e *Etcd) StoreExists(storeId string) (bool, error) {
	path := fmt.Sprintf("/config/%s/storage/stores/%s", e.appName, storeId)
	return e.instance.IsLeafNodeExist(path)
}

// GetStores retrieves all the available stores from the configuration.
// Returns a map of store names to their associated data or an error if stores are not found.
func (e *Etcd) GetStores() (map[string]StoreConfig, error) {
	skyeConfigRegistry := e.GetEtcdInstance()
	fmt.Println("skyeConfigRegistry", skyeConfigRegistry)

	stores := skyeConfigRegistry.Storage.Stores
	if stores == nil {
		return nil, fmt.Errorf("stores not found in configuration")
	}
	return stores, nil
}

// GetEtcdInstance returns the etcd configuration instance
func (e *Etcd) GetEtcdInstance() *SkyeConfigRegistry {
	instance, ok := e.instance.GetConfigInstance().(*SkyeConfigRegistry)
	if !ok {
		log.Panic().Msg("invalid etcd instance - expected SkyeConfigRegistry")
		return nil
	}
	return instance
}

// GetJobFrequencies retrieves all the available job frequencies from the configuration.
// Returns a map of frequency names to their associated data or an error if frequencies are not found.
func (e *Etcd) GetJobFrequencies() (map[string]JobFrequencyConfig, error) {
	skyeConfigRegistry := e.GetEtcdInstance()
	frequencies := skyeConfigRegistry.JobFrequencyConfigs
	if frequencies == nil {
		return nil, fmt.Errorf("job frequencies not found in configuration")
	}
	return frequencies, nil
}

// GetJobFrequenciesAsString retrieves job frequencies as a comma-separated string for backward compatibility
func (e *Etcd) GetJobFrequenciesAsString() (string, error) {
	frequencies, err := e.GetJobFrequencies()
	if err != nil {
		return "", err
	}

	if len(frequencies) == 0 {
		return "", nil
	}

	var frequencyIds []string
	for frequencyId := range frequencies {
		frequencyIds = append(frequencyIds, frequencyId)
	}

	return strings.Join(frequencyIds, ","), nil
}

func (e *Etcd) getDataMapAndMetaMapFromWatcher() (map[string]string, map[string]string, error) {
	dataMap, metaMap, err := e.instance.GetWatcherDataMaps()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get watcher data maps: %w", err)
	}

	return dataMap, metaMap, nil
}

// GetEntities retrieves all entities from hierarchical ETCD paths using watcher's dataMap
// Reads from /config/{appName}/entity/{entityName}/store-id
func (e *Etcd) GetEntities() (map[string]EntityConfig, error) {
	entityPathPrefix := fmt.Sprintf("/config/%s/entity/", e.appName)
	log.Info().Msgf("Fetching entities from watcher's dataMap at path: %s", entityPathPrefix)

	dataMap, metaMap, err := e.getDataMapAndMetaMapFromWatcher()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get dataMap/metaMap from watcher, falling back to registry")
		// Fallback to registry (may be empty)
		skyeConfigRegistry := e.GetEtcdInstance()
		entities := skyeConfigRegistry.EntityConfigs
		if len(entities) == 0 {
			log.Info().Msg("No entities found in registry. Returning empty map.")
			return make(map[string]EntityConfig), nil
		}
		return entities, nil
	}

	log.Info().Msgf("Successfully accessed watcher's dataMap (size: %d keys), metaMap (size: %d keys)", len(dataMap), len(metaMap))
	entities := make(map[string]EntityConfig)

	// Iterate through dataMap to find all entity store-id files
	for normalizedKey, value := range dataMap {
		// Get original path from metaMap
		originalPath, exists := metaMap[normalizedKey]
		if !exists {
			log.Debug().Msgf("metaMap missing entry for normalizedKey: %s", normalizedKey)
			continue
		}

		// Look for paths matching: /config/{appName}/entity/{entityName}/store-id
		if strings.HasPrefix(originalPath, entityPathPrefix) && strings.HasSuffix(originalPath, "/store-id") {
			// Extract entity name from the path: /config/{appName}/entity/{entityName}/store-id
			pathWithoutPrefix := strings.TrimPrefix(originalPath, entityPathPrefix)
			pathWithoutSuffix := strings.TrimSuffix(pathWithoutPrefix, "/store-id")
			entityName := pathWithoutSuffix

			if entityName != "" {
				storeID := strings.TrimSpace(value)
				entityId := fmt.Sprintf("%s_%s", entityName, storeID)
				entities[entityId] = EntityConfig{
					Entity:  entityName,
					StoreID: storeID,
				}
				log.Info().Msgf("✅ Found entity from watcher: path=%s, entity=%s, storeID=%s", originalPath, entityName, storeID)
			}
		}
	}

	if len(entities) == 0 {
		log.Info().Msgf("No entities found in watcher's dataMap at path: %s (checked %d keys total). Returning empty map.", entityPathPrefix, len(dataMap))
		return make(map[string]EntityConfig), nil
	}

	log.Info().Msgf("✅ Retrieved %d entities from watcher's dataMap", len(entities))
	return entities, nil
}

// GetModels retrieves all models from hierarchical ETCD paths
// Reads from /config/{appName}/entity/{entityName}/models/{modelName}/...
func (e *Etcd) GetModels() (map[string]ModelConfig, error) {
	dataMap, metaMap, err := e.getDataMapAndMetaMapFromWatcher()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get dataMap/metaMap from watcher, falling back to registry")
		skyeConfigRegistry := e.GetEtcdInstance()
		models := skyeConfigRegistry.ModelConfigs
		if len(models) == 0 {
			log.Info().Msg("No models found in registry. Returning empty map.")
			return make(map[string]ModelConfig), nil
		}
		return models, nil
	}

	modelsMap := make(map[string]ModelConfig) // Key: {entity}_{model}

	// Group all model-related paths by entity and model
	modelFields := make(map[string]map[string]string) // Key: {entity}_{model}, Value: fieldName -> value

	for normalizedKey, value := range dataMap {
		originalPath, exists := metaMap[normalizedKey]
		if !exists {
			continue
		}

		if strings.Contains(originalPath, "/models/") && !strings.Contains(originalPath, "/variants/") {
			// Extract entity and model from path
			// Format: /config/{appName}/entity/{entity}/models/{model}/{field}
			parts := strings.Split(originalPath, "/")
			entityIdx := -1
			modelsIdx := -1
			for i, part := range parts {
				if part == "entity" && i+1 < len(parts) {
					entityIdx = i + 1
				}
				if part == "models" && i+1 < len(parts) {
					modelsIdx = i + 1
				}
			}

			if entityIdx > 0 && modelsIdx > 0 && modelsIdx < len(parts)-1 {
				entityName := parts[entityIdx]
				modelName := parts[modelsIdx]
				fieldName := parts[len(parts)-1] // Last part is the field name

				modelKey := fmt.Sprintf("%s_%s", entityName, modelName)
				if modelFields[modelKey] == nil {
					modelFields[modelKey] = make(map[string]string)
				}
				modelFields[modelKey][fieldName] = value
			}
		}
	}

	// Construct ModelConfig from collected fields
	entityModelMap := make(map[string][]string) // modelKey -> [entityName, modelName]
	for normalizedKey := range dataMap {
		originalPath, exists := metaMap[normalizedKey]
		if !exists {
			continue
		}
		if strings.Contains(originalPath, "/models/") && !strings.Contains(originalPath, "/variants/") {
			parts := strings.Split(originalPath, "/")
			entityIdx := -1
			modelsIdx := -1
			for i, part := range parts {
				if part == "entity" && i+1 < len(parts) {
					entityIdx = i + 1
				}
				if part == "models" && i+1 < len(parts) {
					modelsIdx = i + 1
				}
			}
			if entityIdx > 0 && modelsIdx > 0 {
				entityName := parts[entityIdx]
				modelName := parts[modelsIdx]
				modelKey := fmt.Sprintf("%s_%s", entityName, modelName)
				if _, exists := entityModelMap[modelKey]; !exists {
					entityModelMap[modelKey] = []string{entityName, modelName}
				}
			}
		}
	}

	for modelKey, fields := range modelFields {
		entityModel, exists := entityModelMap[modelKey]
		if !exists {
			// Fallback: try to split (may not work correctly for names with underscores)
			parts := strings.Split(modelKey, "_")
			if len(parts) < 2 {
				continue
			}
			entityModel = []string{parts[0], strings.Join(parts[1:], "_")}
		}
		entityName := entityModel[0]
		modelName := entityModel[1]

		modelConfig := ModelConfig{
			Entity: entityName,
			Model:  modelName,
		}

		// Parse fields
		if val, ok := fields["embedding-store-enabled"]; ok {
			modelConfig.EmbeddingStoreEnabled = strings.ToLower(val) == "true"
		}
		if val, ok := fields["embedding-store-version"]; ok {
			if v, err := strconv.Atoi(val); err == nil {
				modelConfig.EmbeddingStoreVersion = v
			}
		}
		if val, ok := fields["embedding-store-ttl"]; ok {
			if v, err := strconv.Atoi(val); err == nil {
				modelConfig.EmbeddingStoreTTL = v
			}
		}
		if val, ok := fields["model-type"]; ok {
			modelConfig.ModelType = val
		}
		if val, ok := fields["mq-id"]; ok {
			if v, err := strconv.Atoi(val); err == nil {
				modelConfig.MQID = v
			}
		}
		if val, ok := fields["job-frequency"]; ok {
			modelConfig.JobFrequency = val
		}
		if val, ok := fields["training-data-path"]; ok {
			modelConfig.TrainingDataPath = val
		}
		if val, ok := fields["number-of-partitions"]; ok {
			if v, err := strconv.Atoi(val); err == nil {
				modelConfig.NumberOfPartitions = v
			}
		}
		if val, ok := fields["topic-name"]; ok {
			modelConfig.TopicName = val
		}
		if val, ok := fields["metadata"]; ok {
			modelConfig.Metadata = val
		}
		if val, ok := fields["failure-producer-mq-id"]; ok {
			if v, err := strconv.Atoi(val); err == nil {
				modelConfig.FailureProducerMqId = v
			}
		}
		if val, ok := fields["model-config"]; ok {
			// Parse JSON model config details
			var modelConfigDetails ModelConfigDetails
			if err := json.Unmarshal([]byte(val), &modelConfigDetails); err == nil {
				modelConfig.ModelConfigDetails = modelConfigDetails
			}
		}

		modelsMap[modelKey] = modelConfig
		log.Debug().Msgf("Constructed model from hierarchical path: %s -> entity=%s, model=%s", modelKey, entityName, modelName)
	}

	if len(modelsMap) == 0 {
		log.Info().Msg("No models found in ETCD hierarchical structure. Returning empty map.")
		return make(map[string]ModelConfig), nil
	}

	log.Info().Msgf("Retrieved %d models from hierarchical ETCD paths", len(modelsMap))
	return modelsMap, nil
}

// GetVariants retrieves all variants from hierarchical ETCD paths
// Reads from /config/{appName}/entity/{entityName}/models/{modelName}/variants/{variantName}/...
func (e *Etcd) GetVariants() (map[string]VariantConfig, error) {
	dataMap, metaMap, err := e.getDataMapAndMetaMapFromWatcher()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get dataMap/metaMap from watcher, falling back to registry")
		skyeConfigRegistry := e.GetEtcdInstance()
		variants := skyeConfigRegistry.VariantConfigs
		if len(variants) == 0 {
			log.Info().Msg("No variants found in registry. Returning empty map.")
			return make(map[string]VariantConfig), nil
		}
		return variants, nil
	}

	variantsMap := make(map[string]VariantConfig)       // Key: {entity}_{model}_{variant}
	variantFields := make(map[string]map[string]string) // Key: {entity}_{model}_{variant}, Value: fieldName -> value

	for normalizedKey, value := range dataMap {
		originalPath, exists := metaMap[normalizedKey]
		if !exists {
			continue
		}

		if strings.Contains(originalPath, "/variants/") {
			// Extract entity, model, and variant from path
			// Format: /config/{appName}/entity/{entity}/models/{model}/variants/{variant}/{field}
			parts := strings.Split(originalPath, "/")
			entityIdx := -1
			modelsIdx := -1
			variantsIdx := -1
			for i, part := range parts {
				if part == "entity" && i+1 < len(parts) {
					entityIdx = i + 1
				}
				if part == "models" && i+1 < len(parts) {
					modelsIdx = i + 1
				}
				if part == "variants" && i+1 < len(parts) {
					variantsIdx = i + 1
				}
			}

			if entityIdx > 0 && modelsIdx > 0 && variantsIdx > 0 && variantsIdx < len(parts)-1 {
				entityName := parts[entityIdx]
				modelName := parts[modelsIdx]
				variantName := parts[variantsIdx]
				fieldName := parts[len(parts)-1]

				variantKey := fmt.Sprintf("%s_%s_%s", entityName, modelName, variantName)
				if variantFields[variantKey] == nil {
					variantFields[variantKey] = make(map[string]string)
				}
				variantFields[variantKey][fieldName] = value
			}
		}
	}

	// Construct VariantConfig from collected fields
	entityModelVariantMap := make(map[string][]string) // variantKey -> [entityName, modelName, variantName]
	for normalizedKey := range dataMap {
		originalPath, exists := metaMap[normalizedKey]
		if !exists {
			continue
		}
		if strings.Contains(originalPath, "/variants/") {
			parts := strings.Split(originalPath, "/")
			entityIdx := -1
			modelsIdx := -1
			variantsIdx := -1
			for i, part := range parts {
				if part == "entity" && i+1 < len(parts) {
					entityIdx = i + 1
				}
				if part == "models" && i+1 < len(parts) {
					modelsIdx = i + 1
				}
				if part == "variants" && i+1 < len(parts) {
					variantsIdx = i + 1
				}
			}
			if entityIdx > 0 && modelsIdx > 0 && variantsIdx > 0 {
				entityName := parts[entityIdx]
				modelName := parts[modelsIdx]
				variantName := parts[variantsIdx]
				variantKey := fmt.Sprintf("%s_%s_%s", entityName, modelName, variantName)
				if _, exists := entityModelVariantMap[variantKey]; !exists {
					entityModelVariantMap[variantKey] = []string{entityName, modelName, variantName}
				}
			}
		}
	}

	for variantKey, fields := range variantFields {
		entityModelVariant, exists := entityModelVariantMap[variantKey]
		if !exists {
			// Fallback: try to split (may not work correctly for names with underscores)
			parts := strings.Split(variantKey, "_")
			if len(parts) < 3 {
				continue
			}
			entityModelVariant = []string{parts[0], parts[1], strings.Join(parts[2:], "_")}
		}
		entityName := entityModelVariant[0]
		modelName := entityModelVariant[1]
		variantName := entityModelVariant[2]

		variantConfig := VariantConfig{
			Entity:  entityName,
			Model:   modelName,
			Variant: variantName,
		}

		// Parse key fields
		if val, ok := fields["vector-db-type"]; ok {
			variantConfig.VectorDBType = val
		}
		if val, ok := fields["type"]; ok {
			variantConfig.Type = val
		}
		if val, ok := fields["enabled"]; ok {
			variantConfig.Enabled = strings.ToLower(val) == "true"
		}
		if val, ok := fields["onboarded"]; ok {
			variantConfig.Onboarded = strings.ToLower(val) == "true"
		}
		if val, ok := fields["variant-state"]; ok {
			variantConfig.VariantState = val
		}
		if val, ok := fields["in-memory-caching-enabled"]; ok {
			variantConfig.CachingConfiguration.InMemoryCachingEnabled = strings.ToLower(val) == "true"
		}
		if val, ok := fields["distributed-caching-enabled"]; ok {
			variantConfig.CachingConfiguration.DistributedCachingEnabled = strings.ToLower(val) == "true"
		}
		if val, ok := fields["rt-partition"]; ok {
			if v, err := strconv.Atoi(val); err == nil {
				variantConfig.RTPartition = v
			}
		}
		if val, ok := fields["vector-db-config"]; ok {
			// Parse JSON vector DB config
			var vectorDbConfig VectorDBConfig
			if err := json.Unmarshal([]byte(val), &vectorDbConfig); err == nil {
				variantConfig.VectorDBConfig = vectorDbConfig
			}
		}
		if val, ok := fields["filter"]; ok {
			// Parse JSON filter configuration
			var filterConfig FilterConfiguration
			if err := json.Unmarshal([]byte(val), &filterConfig); err == nil {
				variantConfig.FilterConfiguration = filterConfig
			}
		}

		variantsMap[variantKey] = variantConfig
		log.Debug().Msgf("Constructed variant from hierarchical path: %s -> entity=%s, model=%s, variant=%s", variantKey, entityName, modelName, variantName)
	}

	if len(variantsMap) == 0 {
		log.Info().Msg("No variants found in ETCD hierarchical structure. Returning empty map.")
		return make(map[string]VariantConfig), nil
	}

	log.Info().Msgf("Retrieved %d variants from hierarchical ETCD paths", len(variantsMap))
	return variantsMap, nil
}

// GetFilters retrieves all filters from hierarchical ETCD paths
// Reads from /config/{appName}/entity/{entityName}/filters/{columnName}
func (e *Etcd) GetFilters() (map[string]FilterConfig, error) {
	dataMap, metaMap, err := e.getDataMapAndMetaMapFromWatcher()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get dataMap/metaMap from watcher, falling back to registry")
		skyeConfigRegistry := e.GetEtcdInstance()
		filters := skyeConfigRegistry.FilterConfigs
		if len(filters) == 0 {
			log.Info().Msg("No filters found in registry. Returning empty map.")
			return make(map[string]FilterConfig), nil
		}
		return filters, nil
	}

	filtersMap := make(map[string]FilterConfig) // Key: {columnName}_{filterValue} or just {columnName}

	for normalizedKey, value := range dataMap {
		originalPath, exists := metaMap[normalizedKey]
		if !exists {
			continue
		}

		if strings.Contains(originalPath, "/filters/") && !strings.Contains(originalPath, "/models/") {
			// Format: /config/{appName}/entity/{entity}/filters/{columnName}
			parts := strings.Split(originalPath, "/")
			entityIdx := -1
			filtersIdx := -1
			for i, part := range parts {
				if part == "entity" && i+1 < len(parts) {
					entityIdx = i + 1
				}
				if part == "filters" && i+1 < len(parts) {
					filtersIdx = i + 1
				}
			}

			if entityIdx > 0 && filtersIdx > 0 && filtersIdx < len(parts) {
				entityName := parts[entityIdx]
				columnName := parts[filtersIdx]

				var storedFilterConfig FilterConfig
				err := json.Unmarshal([]byte(value), &storedFilterConfig)
				if err != nil {
					log.Warn().Err(err).Msgf("Failed to parse filter config JSON at path: %s, value: %s. Treating as plain string.", originalPath, value)
					// Fallback: treat value as plain filter value (for backward compatibility)
					filterValue := strings.TrimSpace(value)
					filterId := fmt.Sprintf("%s_%s", columnName, filterValue)
					filtersMap[filterId] = FilterConfig{
						Entity:       entityName,
						ColumnName:   columnName,
						FilterValue:  filterValue,
						DefaultValue: "",
					}
				} else {
					filterValue := storedFilterConfig.FilterValue
					defaultValue := storedFilterConfig.DefaultValue
					filterId := fmt.Sprintf("%s_%s", columnName, filterValue)

					filtersMap[filterId] = FilterConfig{
						Entity:       entityName, // Use entity from path (source of truth)
						ColumnName:   columnName, // Use columnName from path (source of truth)
						FilterValue:  filterValue,
						DefaultValue: defaultValue,
					}
					log.Debug().Msgf("Found filter from hierarchical path: %s -> entity=%s, column=%s, value=%s, default=%s", originalPath, entityName, columnName, filterValue, defaultValue)
				}
			}
		}
	}

	if len(filtersMap) == 0 {
		log.Info().Msg("No filters found in ETCD hierarchical structure. Returning empty map.")
		return make(map[string]FilterConfig), nil
	}

	log.Info().Msgf("Retrieved %d filters from hierarchical ETCD paths", len(filtersMap))
	return filtersMap, nil
}

func (e *Etcd) EntityExists(entityName string) (bool, error) {
	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return false, fmt.Errorf("ETCD instance not initialized")
	}

	// Check if entity exists by looking for the store-id file
	path := fmt.Sprintf("/config/%s/entity/%s/store-id", e.appName, entityName)
	log.Info().Msgf("Checking if entity exists at path: %s", path)

	// Use IsLeafNodeExist method to check if the entity's store_id file exists (it's a leaf node)
	exists, err := e.instance.IsLeafNodeExist(path)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if entity %s exists in ETCD", entityName)
		return false, fmt.Errorf("error checking entity existence: %w", err)
	}

	if exists {
		log.Info().Msgf("✅ Entity %s exists in ETCD", entityName)
	} else {
		log.Info().Msgf("❌ Entity %s does not exist in ETCD", entityName)
	}

	return exists, nil
}

func (e *Etcd) ModelExists(entityName string, modelName string) (bool, error) {
	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return false, fmt.Errorf("ETCD instance not initialized")
	}

	// Check if model exists by checking for a specific field node (hierarchical structure)
	// Using model-type as it's a required field that should always exist
	path := fmt.Sprintf("/config/%s/entity/%s/models/%s/model-type", e.appName, entityName, modelName)
	log.Info().Msgf("Checking if model exists by checking model_type node at path: %s", path)

	// Use IsLeafNodeExist method to check if the model's model_type node exists (it's a leaf node)
	exists, err := e.instance.IsLeafNodeExist(path)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if model %s exists in entity %s", modelName, entityName)
		return false, fmt.Errorf("error checking model existence: %w", err)
	}

	if exists {
		log.Info().Msgf("✅ Model %s exists in entity %s (hierarchical structure confirmed)", modelName, entityName)
	} else {
		log.Info().Msgf("❌ Model %s does not exist in entity %s", modelName, entityName)
	}

	return exists, nil
}

func (e *Etcd) VariantExists(entityName string, modelName string, variantName string) (bool, error) {
	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return false, fmt.Errorf("ETCD instance not initialized")
	}

	// Check if variant exists by checking for a specific field node (hierarchical structure)
	// Using vector-db-type as it's a required field that should always exist
	path := fmt.Sprintf("/config/%s/entity/%s/models/%s/variants/%s/vector-db-type", e.appName, entityName, modelName, variantName)
	log.Info().Msgf("Checking if variant exists by checking vector_db_type node at path: %s", path)

	// Use IsLeafNodeExist method to check if the variant's vector_db_type node exists (it's a leaf node)
	exists, err := e.instance.IsLeafNodeExist(path)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if variant %s exists in model %s, entity %s", variantName, modelName, entityName)
		return false, fmt.Errorf("error checking variant existence: %w", err)
	}

	if exists {
		log.Info().Msgf("✅ Variant %s exists in model %s, entity %s (hierarchical structure confirmed)", variantName, modelName, entityName)
	} else {
		log.Info().Msgf("❌ Variant %s does not exist in model %s, entity %s", variantName, modelName, entityName)
	}

	return exists, nil
}

func (e *Etcd) FilterExistsByColumnName(entityName string, columnName string) (bool, error) {
	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return false, fmt.Errorf("ETCD instance not initialized")
	}

	filterPath := fmt.Sprintf("/config/%s/entity/%s/filters/%s", e.appName, entityName, columnName)
	log.Info().Msgf("Checking if filter with column_name '%s' exists in entity '%s' at path: %s", columnName, entityName, filterPath)

	// Use IsLeafNodeExist since filters are stored as leaf nodes with JSON values
	exists, err := e.instance.IsLeafNodeExist(filterPath)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if filter exists at path: %s", filterPath)
		return false, fmt.Errorf("error checking filter existence at path %s: %w", filterPath, err)
	}

	if exists {
		log.Info().Msgf("✅ Filter with column_name '%s' found in entity '%s'", columnName, entityName)
	} else {
		log.Info().Msgf("❌ Filter with column_name '%s' does not exist in entity '%s'", columnName, entityName)
	}

	return exists, nil
}

// ==================== ENTITY ETCD OPERATIONS ====================

func (e *Etcd) CreateEntityConfig(entityId string, entityConfig EntityConfig) error {
	log.Info().Msgf("CreateEntityConfig called for entity: %s with store_id: %s", entityConfig.Entity, entityConfig.StoreID)

	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}

	storeIdPath := fmt.Sprintf("/config/%s/entity/%s/store-id", e.appName, entityConfig.Entity)
	log.Info().Msgf("Creating store-id file at path: %s with content: %s", storeIdPath, entityConfig.StoreID)

	err := e.instance.CreateNode(storeIdPath, entityConfig.StoreID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create store_id file at path: %s", storeIdPath)
		return fmt.Errorf("failed to create entity store-id file at %s: %w", storeIdPath, err)
	}

	modelsPath := fmt.Sprintf("/config/%s/entity/%s/models", e.appName, entityConfig.Entity)
	log.Info().Msgf("Creating models folder at path: %s", modelsPath)

	modelsFolderPath := fmt.Sprintf("%s/.folder", modelsPath)
	err = e.instance.CreateNode(modelsFolderPath, "")
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create models folder at path: %s", modelsPath)
		return fmt.Errorf("failed to create models folder at %s: %w", modelsPath, err)
	}

	log.Info().Msgf("Successfully created entity structure for: %s", entityConfig.Entity)
	return nil
}

func (e *Etcd) UpdateEntityConfig(entityId string, entityConfig EntityConfig) error {
	// Update the store-id file content
	storeIdPath := fmt.Sprintf("/config/%s/entity/%s/store-id", e.appName, entityConfig.Entity)
	log.Info().Msgf("Updating store-id file at path: %s with content: %s", storeIdPath, entityConfig.StoreID)
	return e.instance.SetValue(storeIdPath, entityConfig.StoreID)
}

func (e *Etcd) DeleteEntityConfig(entityId string) error {
	// Extract entity name from entityId (assuming format like "catalog_2")
	parts := strings.Split(entityId, "_")
	if len(parts) == 0 {
		return fmt.Errorf("invalid entityId format: %s", entityId)
	}
	entityName := parts[0]

	// Delete the entire entity folder
	entityPath := fmt.Sprintf("/config/%s/entity/%s", e.appName, entityName)
	log.Info().Msgf("Deleting entity folder at path: %s", entityPath)
	return e.instance.DeleteNode(entityPath)
}

// ==================== MODEL ETCD OPERATIONS ====================

func (e *Etcd) CreateModelConfig(modelId string, modelConfig ModelConfig) error {
	log.Info().Msgf("CreateModelConfig called for model: %s in entity: %s", modelConfig.Model, modelConfig.Entity)

	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}

	// Base path for the model: /config/{appName}/entity/{entityName}/models/{modelName}/
	basePath := fmt.Sprintf("/config/%s/entity/%s/models/%s", e.appName, modelConfig.Entity, modelConfig.Model)
	log.Info().Msgf("Creating hierarchical model config at base path: %s", basePath)

	// Create individual nodes for each field (matching reference structure)
	nodesToCreate := map[string]interface{}{
		fmt.Sprintf("%s/job-frequency", basePath):           modelConfig.JobFrequency,
		fmt.Sprintf("%s/embedding-store-enabled", basePath): modelConfig.EmbeddingStoreEnabled,
		fmt.Sprintf("%s/embedding-store-version", basePath): modelConfig.EmbeddingStoreVersion,
		fmt.Sprintf("%s/embedding-store-ttl", basePath):     modelConfig.EmbeddingStoreTTL,
		fmt.Sprintf("%s/model-type", basePath):              modelConfig.ModelType,
		fmt.Sprintf("%s/mq-id", basePath):                   modelConfig.MQID,
		fmt.Sprintf("%s/topic-name", basePath):              modelConfig.TopicName,
		fmt.Sprintf("%s/training-data-path", basePath):      modelConfig.TrainingDataPath,
		fmt.Sprintf("%s/number-of-partitions", basePath):    modelConfig.NumberOfPartitions,
		fmt.Sprintf("%s/failure-producer-mq-id", basePath):  modelConfig.FailureProducerMqId,
	}

	// Create partition-states nodes for each partition
	for i := 0; i < modelConfig.NumberOfPartitions; i++ {
		partitionKey := fmt.Sprintf("%d", i)
		partitionValue := 0 // Default partition state
		if modelConfig.PartitionStatus != nil {
			if val, exists := modelConfig.PartitionStatus[partitionKey]; exists {
				partitionValue = val
			}
		}
		nodesToCreate[fmt.Sprintf("%s/partition-states/%s", basePath, partitionKey)] = partitionValue
	}

	// For model-config, marshal it to JSON as it's a complex object
	modelConfigJson, err := json.Marshal(modelConfig.ModelConfigDetails)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal model config details to JSON")
		return fmt.Errorf("failed to marshal model config details: %w", err)
	}
	nodesToCreate[fmt.Sprintf("%s/model-config", basePath)] = string(modelConfigJson)

	// Add metadata as JSON string (can be any additional metadata)
	if modelConfig.Metadata != "" {
		nodesToCreate[fmt.Sprintf("%s/metadata", basePath)] = modelConfig.Metadata
	} else {
		// Default empty metadata JSON object
		nodesToCreate[fmt.Sprintf("%s/metadata", basePath)] = "{}"
	}

	// Create all nodes using CreateNodes (batch operation)
	log.Info().Msgf("Creating %d individual nodes for model config", len(nodesToCreate))
	err = e.instance.CreateNodes(nodesToCreate)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create model config nodes at base path: %s", basePath)
		return fmt.Errorf("failed to create model config nodes: %w", err)
	}

	log.Info().Msgf("Successfully created hierarchical model config for: %s with %d nodes", modelConfig.Model, len(nodesToCreate))
	return nil
}

func (e *Etcd) UpdateModelConfig(modelId string, modelConfig ModelConfig) error {
	// Base path for the model: /config/{appName}/entity/{entityName}/models/{modelName}/
	basePath := fmt.Sprintf("/config/%s/entity/%s/models/%s", e.appName, modelConfig.Entity, modelConfig.Model)
	log.Info().Msgf("Updating hierarchical model config at base path: %s", basePath)

	// Update individual nodes for each field (matching reference structure)
	nodesToUpdate := map[string]interface{}{
		fmt.Sprintf("%s/job-frequency", basePath):           modelConfig.JobFrequency,
		fmt.Sprintf("%s/embedding-store-enabled", basePath): modelConfig.EmbeddingStoreEnabled,
		fmt.Sprintf("%s/embedding-store-version", basePath): modelConfig.EmbeddingStoreVersion,
		fmt.Sprintf("%s/embedding-store-ttl", basePath):     modelConfig.EmbeddingStoreTTL,
		fmt.Sprintf("%s/model-type", basePath):              modelConfig.ModelType,
		fmt.Sprintf("%s/mq-id", basePath):                   modelConfig.MQID,
		fmt.Sprintf("%s/topic-name", basePath):              modelConfig.TopicName,
		fmt.Sprintf("%s/training-data-path", basePath):      modelConfig.TrainingDataPath,
		fmt.Sprintf("%s/number-of-partitions", basePath):    modelConfig.NumberOfPartitions,
		fmt.Sprintf("%s/failure-producer-mq-id", basePath):  modelConfig.FailureProducerMqId,
	}

	// Update partition-states nodes for each partition
	for i := 0; i < modelConfig.NumberOfPartitions; i++ {
		partitionKey := fmt.Sprintf("%d", i)
		partitionValue := 0 // Default partition state
		if modelConfig.PartitionStatus != nil {
			if val, exists := modelConfig.PartitionStatus[partitionKey]; exists {
				partitionValue = val
			}
		}
		nodesToUpdate[fmt.Sprintf("%s/partition-states/%s", basePath, partitionKey)] = partitionValue
	}

	// For model-config, marshal it to JSON as it's a complex object
	modelConfigJson, err := json.Marshal(modelConfig.ModelConfigDetails)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal model config details to JSON")
		return fmt.Errorf("failed to marshal model config details: %w", err)
	}
	nodesToUpdate[fmt.Sprintf("%s/model-config", basePath)] = string(modelConfigJson)

	// Add metadata as JSON string (can be any additional metadata)
	if modelConfig.Metadata != "" {
		nodesToUpdate[fmt.Sprintf("%s/metadata", basePath)] = modelConfig.Metadata
	} else {
		// Default empty metadata JSON object
		nodesToUpdate[fmt.Sprintf("%s/metadata", basePath)] = "{}"
	}

	// Update all nodes using SetValues (batch operation)
	log.Info().Msgf("Updating %d individual nodes for model config", len(nodesToUpdate))
	err = e.instance.SetValues(nodesToUpdate)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to update model config nodes at base path: %s", basePath)
		return fmt.Errorf("failed to update model config nodes: %w", err)
	}

	log.Info().Msgf("Successfully updated hierarchical model config for: %s with %d nodes", modelConfig.Model, len(nodesToUpdate))
	return nil
}

// ==================== VARIANT ETCD OPERATIONS ====================

func (e *Etcd) CreateVariantConfig(variantId string, variantConfig VariantConfig) error {
	log.Info().Msgf("CreateVariantConfig called for variant: %s in model: %s, entity: %s", variantConfig.Variant, variantConfig.Model, variantConfig.Entity)

	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}

	// Base path for the variant: /config/{appName}/entity/{entityName}/models/{modelName}/variants/{variantName}/
	basePath := fmt.Sprintf("/config/%s/entity/%s/models/%s/variants/%s", e.appName, variantConfig.Entity, variantConfig.Model, variantConfig.Variant)
	log.Info().Msgf("Creating hierarchical variant config at base path: %s", basePath)

	// Create individual nodes for all fields matching the reference structure
	nodesToCreate := map[string]interface{}{
		// Core variant properties
		fmt.Sprintf("%s/enabled", basePath):             variantConfig.Enabled,
		fmt.Sprintf("%s/vector-db-type", basePath):      variantConfig.VectorDBType,
		fmt.Sprintf("%s/type", basePath):                variantConfig.Type,
		fmt.Sprintf("%s/onboarded", basePath):           variantConfig.Onboarded,
		fmt.Sprintf("%s/partial-hit-enabled", basePath): variantConfig.PartialHitEnabled,
		fmt.Sprintf("%s/rt-delta-processing", basePath): variantConfig.RTDeltaProcessing,
		fmt.Sprintf("%s/variant-state", basePath):       variantConfig.VariantState,

		// Version management
		fmt.Sprintf("%s/embedding-store-read-version", basePath):  variantConfig.EmbeddingStoreReadVersion,
		fmt.Sprintf("%s/embedding-store-write-version", basePath): variantConfig.EmbeddingStoreWriteVersion,
		fmt.Sprintf("%s/vector-db-read-version", basePath):        variantConfig.VectorDBReadVersion,
		fmt.Sprintf("%s/vector-db-write-version", basePath):       variantConfig.VectorDBWriteVersion,

		// Caching configuration (individual nodes)
		fmt.Sprintf("%s/in-memory-caching-enabled", basePath):     variantConfig.CachingConfiguration.InMemoryCachingEnabled,
		fmt.Sprintf("%s/in-memory-cache-TTL-seconds", basePath):   variantConfig.CachingConfiguration.InMemoryCacheTTLSeconds,
		fmt.Sprintf("%s/distributed-caching-enabled", basePath):   variantConfig.CachingConfiguration.DistributedCachingEnabled,
		fmt.Sprintf("%s/distributed-cache-TTL-seconds", basePath): variantConfig.CachingConfiguration.DistributedCacheTTLSeconds,

		// Rate limiter configuration
		fmt.Sprintf("%s/rate-limiters/rate-limit", basePath):  variantConfig.RateLimiter.RateLimit,
		fmt.Sprintf("%s/rate-limiters/burst-limit", basePath): variantConfig.RateLimiter.BurstLimit,

		// RT partition
		fmt.Sprintf("%s/rt-partition", basePath): variantConfig.RTPartition,
	}

	// Marshal complex objects to JSON
	filterConfigJson, err := json.Marshal(variantConfig.FilterConfiguration)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal filter configuration to JSON")
		return fmt.Errorf("failed to marshal filter configuration: %w", err)
	}
	nodesToCreate[fmt.Sprintf("%s/filter", basePath)] = string(filterConfigJson)

	vectorDbConfigJson, err := json.Marshal(variantConfig.VectorDBConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal vector DB config to JSON")
		return fmt.Errorf("failed to marshal vector DB config: %w", err)
	}
	nodesToCreate[fmt.Sprintf("%s/vector-db-config", basePath)] = string(vectorDbConfigJson)

	// Create all nodes using CreateNodes (batch operation)
	log.Info().Msgf("Creating %d individual nodes for variant config matching reference structure", len(nodesToCreate))
	err = e.instance.CreateNodes(nodesToCreate)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create variant config nodes at base path: %s", basePath)
		return fmt.Errorf("failed to create variant config nodes: %w", err)
	}

	log.Info().Msgf("Successfully created hierarchical variant config for: %s with %d nodes matching reference structure", variantConfig.Variant, len(nodesToCreate))
	return nil
}

func (e *Etcd) UpdateVariantConfig(variantId string, variantConfig VariantConfig) error {
	// Base path for the variant: /config/{appName}/entity/{entityName}/models/{modelName}/variants/{variantName}/
	basePath := fmt.Sprintf("/config/%s/entity/%s/models/%s/variants/%s", e.appName, variantConfig.Entity, variantConfig.Model, variantConfig.Variant)
	log.Info().Msgf("Updating hierarchical variant config at base path: %s", basePath)

	// Update individual nodes for all fields matching the reference structure
	nodesToUpdate := map[string]interface{}{
		// Core variant properties
		fmt.Sprintf("%s/enabled", basePath):             variantConfig.Enabled,
		fmt.Sprintf("%s/vector-db-type", basePath):      variantConfig.VectorDBType,
		fmt.Sprintf("%s/type", basePath):                variantConfig.Type,
		fmt.Sprintf("%s/onboarded", basePath):           variantConfig.Onboarded,
		fmt.Sprintf("%s/partial-hit-enabled", basePath): variantConfig.PartialHitEnabled,
		fmt.Sprintf("%s/rt-delta-processing", basePath): variantConfig.RTDeltaProcessing,
		fmt.Sprintf("%s/variant-state", basePath):       variantConfig.VariantState,

		// Version management
		fmt.Sprintf("%s/embedding-store-read-version", basePath):  variantConfig.EmbeddingStoreReadVersion,
		fmt.Sprintf("%s/embedding-store-write-version", basePath): variantConfig.EmbeddingStoreWriteVersion,
		fmt.Sprintf("%s/vector-db-read-version", basePath):        variantConfig.VectorDBReadVersion,
		fmt.Sprintf("%s/vector-db-write-version", basePath):       variantConfig.VectorDBWriteVersion,

		// Caching configuration (individual nodes)
		fmt.Sprintf("%s/in-memory-caching-enabled", basePath):     variantConfig.CachingConfiguration.InMemoryCachingEnabled,
		fmt.Sprintf("%s/in-memory-cache-TTL-seconds", basePath):   variantConfig.CachingConfiguration.InMemoryCacheTTLSeconds,
		fmt.Sprintf("%s/distributed-caching-enabled", basePath):   variantConfig.CachingConfiguration.DistributedCachingEnabled,
		fmt.Sprintf("%s/distributed-cache-TTL-seconds", basePath): variantConfig.CachingConfiguration.DistributedCacheTTLSeconds,

		// Rate limiter configuration
		fmt.Sprintf("%s/rate-limiters/rate-limit", basePath):  variantConfig.RateLimiter.RateLimit,
		fmt.Sprintf("%s/rate-limiters/burst-limit", basePath): variantConfig.RateLimiter.BurstLimit,

		// RT partition
		fmt.Sprintf("%s/rt-partition", basePath): variantConfig.RTPartition,
	}

	// Marshal complex objects to JSON
	filterConfigJson, err := json.Marshal(variantConfig.FilterConfiguration)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal filter configuration to JSON")
		return fmt.Errorf("failed to marshal filter configuration: %w", err)
	}
	nodesToUpdate[fmt.Sprintf("%s/filter", basePath)] = string(filterConfigJson)

	vectorDbConfigJson, err := json.Marshal(variantConfig.VectorDBConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal vector DB config to JSON")
		return fmt.Errorf("failed to marshal vector DB config: %w", err)
	}
	nodesToUpdate[fmt.Sprintf("%s/vector-db-config", basePath)] = string(vectorDbConfigJson)

	// Update all nodes using SetValues (batch operation)
	log.Info().Msgf("Updating %d individual nodes for variant config matching reference structure", len(nodesToUpdate))
	err = e.instance.SetValues(nodesToUpdate)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to update variant config nodes at base path: %s", basePath)
		return fmt.Errorf("failed to update variant config nodes: %w", err)
	}

	log.Info().Msgf("Successfully updated hierarchical variant config for: %s with %d nodes matching reference structure", variantConfig.Variant, len(nodesToUpdate))
	return nil
}

// ==================== FILTER ETCD OPERATIONS ====================

func (e *Etcd) CreateFilterConfig(filterId string, filterConfig FilterConfig) error {
	log.Info().Msgf("CreateFilterConfig called for column_name: %s in entity: %s", filterConfig.ColumnName, filterConfig.Entity)

	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}

	configJson, err := json.Marshal(filterConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal filter config: %w", err)
	}

	// Base path for filter: /config/{appName}/entity/{entityName}/filters/{columnName}
	path := fmt.Sprintf("/config/%s/entity/%s/filters/%s", e.appName, filterConfig.Entity, filterConfig.ColumnName)
	log.Info().Msgf("Creating filter config at path: %s", path)

	err = e.instance.CreateNode(path, string(configJson))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create filter config at path: %s", path)
		return fmt.Errorf("failed to create filter config at %s: %w", path, err)
	}

	log.Info().Msgf("Successfully created filter config for column_name: %s in entity: %s", filterConfig.ColumnName, filterConfig.Entity)
	return nil
}

func (e *Etcd) UpdateFilterConfig(filterId string, filterConfig FilterConfig) error {
	log.Info().Msgf("UpdateFilterConfig called for column_name: %s in entity: %s", filterConfig.ColumnName, filterConfig.Entity)

	configJson, err := json.Marshal(filterConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal filter config: %w", err)
	}

	// Update filter under: /config/{appName}/entity/{entityName}/filters/{columnName}
	path := fmt.Sprintf("/config/%s/entity/%s/filters/%s", e.appName, filterConfig.Entity, filterConfig.ColumnName)
	log.Info().Msgf("Updating filter config at path: %s", path)

	err = e.instance.SetValue(path, string(configJson))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to update filter config at path: %s", path)
		return fmt.Errorf("failed to update filter config at %s: %w", path, err)
	}

	log.Info().Msgf("Successfully updated filter config for column_name: %s in entity: %s", filterConfig.ColumnName, filterConfig.Entity)
	return nil
}

// ==================== JOB FREQUENCY ETCD OPERATIONS ====================

func (e *Etcd) CreateJobFrequencyConfig(frequencyId string, frequencyConfig JobFrequencyConfig) error {
	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}

	// Get current frequencies directly from configuration
	currentFrequencies, err := e.GetJobFrequenciesAsString()
	if err != nil {
		// No frequencies exist yet, start with empty string
		log.Info().Msgf("No frequencies exist yet, will create new one")
		currentFrequencies = ""
	} else {
		log.Info().Msgf("Current frequencies in ETCD: %s", currentFrequencies)
	}

	// Check if frequency already exists to prevent duplicates
	if currentFrequencies != "" {
		frequencies := strings.Split(currentFrequencies, ",")
		for _, freq := range frequencies {
			if strings.TrimSpace(freq) == frequencyId {
				log.Info().Msgf("Frequency %s already exists, skipping append", frequencyId)
				return nil // Already exists, no need to add
			}
		}
	}

	// Append new frequency to the comma-separated string
	var updatedFrequencies string
	if currentFrequencies == "" {
		updatedFrequencies = frequencyId
	} else {
		updatedFrequencies = currentFrequencies + "," + frequencyId
	}

	log.Info().Msgf("Updated frequencies string: %s", updatedFrequencies)

	// Path for frequencies file: /config/{appName}/storage/frequencies
	frequenciesPath := fmt.Sprintf("/config/%s/storage/frequencies", e.appName)

	// Update or create the frequencies file in ETCD
	if currentFrequencies == "" {
		err = e.instance.CreateNode(frequenciesPath, updatedFrequencies)
	} else {
		err = e.instance.SetValue(frequenciesPath, updatedFrequencies)
	}

	if err != nil {
		log.Error().Err(err).Msgf("Failed to update frequencies file in ETCD at path: %s", frequenciesPath)
		return fmt.Errorf("failed to update frequencies file in ETCD: %w", err)
	}

	log.Info().Msgf("Successfully updated frequencies file in ETCD with frequency: %s", frequencyId)
	return nil
}

func (e *Etcd) UpdateJobFrequencyConfig(frequencyId string, frequencyConfig JobFrequencyConfig) error {
	return e.CreateJobFrequencyConfig(frequencyId, frequencyConfig)
}

func (e *Etcd) DeleteJobFrequencyConfig(frequencyId string) error {
	log.Info().Msgf("DeleteJobFrequencyConfig called for frequencyId: %s", frequencyId)

	if e.instance == nil {
		log.Error().Msg("ETCD instance is nil - not initialized properly")
		return fmt.Errorf("ETCD instance not initialized")
	}

	// Get current frequencies directly from configuration
	currentFrequencies, err := e.GetJobFrequenciesAsString()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get current frequencies from ETCD")
		return fmt.Errorf("failed to get current frequencies: %w", err)
	}

	log.Info().Msgf("Current frequencies in ETCD: %s", currentFrequencies)

	// Split frequencies and filter out the one to delete
	frequencies := strings.Split(currentFrequencies, ",")
	var updatedFrequencies []string
	found := false

	for _, freq := range frequencies {
		trimmedFreq := strings.TrimSpace(freq)
		if trimmedFreq != frequencyId && trimmedFreq != "" {
			updatedFrequencies = append(updatedFrequencies, trimmedFreq)
		} else if trimmedFreq == frequencyId {
			found = true
		}
	}

	if !found {
		log.Warn().Msgf("Frequency %s not found in ETCD, nothing to delete", frequencyId)
		return nil // Not found, but not an error
	}

	// Join the remaining frequencies back to comma-separated string
	updatedFrequenciesStr := strings.Join(updatedFrequencies, ",")
	log.Info().Msgf("Updated frequencies string after deletion: %s", updatedFrequenciesStr)

	// Path for frequencies file: /config/{appName}/storage/frequencies
	frequenciesPath := fmt.Sprintf("/config/%s/storage/frequencies", e.appName)

	// Update the frequencies file in ETCD
	err = e.instance.SetValue(frequenciesPath, updatedFrequenciesStr)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to update frequencies file in ETCD after deletion")
		return fmt.Errorf("failed to update frequencies file after deletion: %w", err)
	}

	log.Info().Msgf("Successfully deleted frequency %s from ETCD", frequencyId)
	return nil
}
