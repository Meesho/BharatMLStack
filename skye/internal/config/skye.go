package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/pkg/etcd"
)

type SkyeManager struct {
	etcd    etcd.Etcd
	appName string
}

// NewSkyeManager creates a SkyeManager with the given etcd client and app name.
// Used for testing with a mock etcd.
func NewSkyeManager(etcd etcd.Etcd, appName string) *SkyeManager {
	return &SkyeManager{etcd: etcd, appName: appName}
}

func initSkyeManager() Manager {
	if manager == nil {
		once.Do(func() {
			manager = &SkyeManager{
				etcd:    etcd.Instance()[appName],
				appName: appName,
			}
		})
	}
	return manager
}

func (s *SkyeManager) GetSkyeConfig() (*Skye, error) {
	etcdConfigInstance, ok := s.etcd.GetConfigInstance().(*Skye)
	if !ok {
		return nil, errors.New("failed to cast etcd config instance to Skye type")
	}
	etcdConf := etcdConfigInstance
	if etcdConf == nil {
		return nil, errors.New("etcdConf not found in configuration")
	}
	return etcdConf, nil
}

// GetEntities retrieves all entities from the configuration.
// Returns a map of entity names to their models or an error if entities are not found.
func (s *SkyeManager) GetEntities() (map[string]Models, error) {
	etcdConfigInstance, ok := s.etcd.GetConfigInstance().(*Skye)
	if !ok {
		return nil, errors.New("failed to cast etcd config instance to Skye type")
	}
	entities := etcdConfigInstance.Entity
	if entities == nil {
		return nil, errors.New("entities not found in configuration")
	}
	return entities, nil
}

func (s *SkyeManager) GetEntityConfig(entity string) (*Models, error) {
	skye, err := s.GetSkyeConfig()
	if err != nil {
		return nil, err
	}
	entityConf, exists := skye.Entity[entity]
	if !exists {
		return nil, fmt.Errorf("entity '%s' not found", entity)
	}
	return &entityConf, nil
}

func (s *SkyeManager) GetModelConfig(entity, model string) (*Model, error) {
	entityConf, err := s.GetEntityConfig(entity)
	if err != nil {
		return nil, err
	}
	modelConf, exists := entityConf.Models[model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found in entity '%s'", model, entity)
	}
	return &modelConf, nil
}

func (s *SkyeManager) GetVariantConfig(entity, model, variant string) (*Variant, error) {
	modelConf, err := s.GetModelConfig(entity, model)
	if err != nil {
		return nil, err
	}
	variantConf, exists := modelConf.Variants[variant]
	if !exists {
		return nil, fmt.Errorf("variant '%s' not found in model '%s' of entity '%s'", variant, model, entity)
	}
	return &variantConf, nil
}

// GetAllFiltersForActiveVariants retrieves all filters for each model and variant under the specified entity.
// Returns a nested map of criteria: model name -> variant name -> slice of Criteria structs or an error if entity not found.
func (s *SkyeManager) GetAllFiltersForActiveVariants(entity string) (map[string]map[string][]Criteria, error) {
	entityConf, err := s.GetEntityConfig(entity)
	if err != nil {
		return nil, err
	}
	criteriaMap := make(map[string]map[string][]Criteria)
	for modelName, modelConfig := range entityConf.Models {
		variantMap := make(map[string][]Criteria)
		for variantName, variantConfig := range modelConfig.Variants {
			if variantConfig.Enabled && variantConfig.Onboarded {
				variantMap[variantName] = variantConfig.Filter["criteria"]
			}
		}
		criteriaMap[modelName] = variantMap
	}
	return criteriaMap, nil
}

// SetVariantOnboarded UpdateVariantOnboarded updates the onboarded for the vector database of a specified variant.
// Returns an error if the update fails.
func (s *SkyeManager) SetVariantOnboarded(entity string, model string, variant string, onboarded bool) error {
	path := s.getVariantPath(entity, model, variant, "onboarded")
	err := s.etcd.SetValue(path, onboarded)
	return err
}

// UpdateVariantState updates the state of a specified variant in etcd.
// Returns an error if the update fails.
func (s *SkyeManager) UpdateVariantState(entity string, model string, variant string, variantState enums.VariantState) error {
	path := s.getVariantPath(entity, model, variant, "variant-state")
	return s.etcd.SetValue(path, variantState)
}

// UpdateVariantReadVersion updates the read version for the vector database of a specified variant.
// Returns an error if the update fails.
func (s *SkyeManager) UpdateVariantReadVersion(entity string, model string, variant string, version int) error {
	path := s.getVariantPath(entity, model, variant, "vector-db-read-version")
	return s.etcd.SetValue(path, version)
}

// UpdateVariantWriteVersion updates the write version for the vector database of a specified variant.
// Returns an error if the update fails.
func (s *SkyeManager) UpdateVariantWriteVersion(entity string, model string, variant string, version int) error {
	path := s.getVariantPath(entity, model, variant, "vector-db-write-version")
	return s.etcd.SetValue(path, version)
}

// RegisterStore registers a new store in etcd with the provided config.
// Returns an error if the registration fails.
func (s *SkyeManager) RegisterStore(confId int, db string, embeddingTable string, aggregatorTable string) error {
	skye, err := s.GetSkyeConfig()
	if err != nil {
		return err
	}
	stores := skye.Storage.Stores
	storeId := len(stores) + 1
	path := fmt.Sprintf("/config/%s/storage/stores/%v", s.appName, storeId)
	data := Data{
		ConfId:          confId,
		EmbeddingTable:  embeddingTable,
		AggregatorTable: aggregatorTable,
		Db:              db,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.etcd.CreateNode(path, string(jsonData))
}

// RegisterFrequency registers a new frequency in etcd with the provided config.
// Returns an error if the registration fails.
func (s *SkyeManager) RegisterFrequency(frequency string) error {
	skye, err := s.GetSkyeConfig()
	if err != nil {
		return err
	}
	registeredFrequencies := skye.Storage.Frequencies
	if strings.Contains(registeredFrequencies, frequency) {
		return fmt.Errorf("frequency is already registered")
	}
	path := fmt.Sprintf("/config/%s/storage/frequencies", s.appName)
	registeredFrequencies = strings.Join([]string{registeredFrequencies, frequency}, ",")
	return s.etcd.SetValue(path, registeredFrequencies)
}

// RegisterEntity registers a new entity in etcd with the specified store ID.
// Returns an error if the registration fails.
func (s *SkyeManager) RegisterEntity(entity string, storeId string) error {
	paths := map[string]interface{}{
		fmt.Sprintf("/config/%s/entity/%s/store-id", s.appName, entity): storeId,
	}
	return s.etcd.CreateNodes(paths)
}

// RegisterModel registers a new model configuration in etcd under the specified entity.
// It creates the necessary nodes and sets various properties for the model.
//
// Parameters:
// - entity: The name of the entity to which the model belongs.
// - model: The name of the model being registered.
// - embeddingStoreEnabled: Indicates whether the embedding store is enabled.
// - embeddingStoreTtl: The time-to-live for the embedding store.
// - modelConfig: A map containing configuration settings for the model.
// - modelType: The type of the model being registered.
// - trainingDataPath: The path to the training data for the model.
//
// Returns an error if the registration fails.
func (s *SkyeManager) RegisterModel(entity string, model string, embeddingStoreEnabled bool, embeddingStoreTtl int,
	modelConfig map[string]interface{}, modelType string, trainingDataPath string, metadata Metadata, jobFrequency string, numberOfPartitions int, topicName string) error {
	paths := make(map[string]interface{})

	skye, err := s.GetSkyeConfig()
	if err != nil {
		return err
	}
	registeredFrequencies := skye.Storage.Frequencies
	if !strings.Contains(registeredFrequencies, jobFrequency) {
		return fmt.Errorf("frequency is not registered, please register frequency first")
	}

	for _, models := range skye.Entity {
		for etcdModel, _ := range models.Models {
			if etcdModel == model {
				return fmt.Errorf("model already registered")
			}
		}
	}
	modelConfigJson, err := json.Marshal(modelConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal model config: %w", err)
	}

	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal operation mapping: %w", err)
	}

	// Create model node
	modelPath := fmt.Sprintf("/config/%s/entity/%s/models/%s", s.appName, entity, model)

	// Set properties
	paths[fmt.Sprintf("%s/job-frequency", modelPath)] = jobFrequency
	paths[fmt.Sprintf("%s/embedding-store-enabled", modelPath)] = embeddingStoreEnabled
	paths[fmt.Sprintf("%s/embedding-store-version", modelPath)] = 1
	paths[fmt.Sprintf("%s/embedding-store-ttl", modelPath)] = embeddingStoreTtl
	paths[fmt.Sprintf("%s/model-config", modelPath)] = string(modelConfigJson)
	paths[fmt.Sprintf("%s/model-type", modelPath)] = modelType
	paths[fmt.Sprintf("%s/topic-name", modelPath)] = topicName
	paths[fmt.Sprintf("%s/training-data-path", modelPath)] = trainingDataPath
	paths[fmt.Sprintf("%s/metadata", modelPath)] = string(metadataJson)
	for i := 0; i < numberOfPartitions; i++ {
		paths[fmt.Sprintf("%s/partition-states/%s", modelPath, strconv.Itoa(i))] = 0
	}
	paths[fmt.Sprintf("%s/number-of-partitions", modelPath)] = numberOfPartitions

	// Create nodes with properties
	if err := s.etcd.CreateNodes(paths); err != nil {
		return fmt.Errorf("failed to create model properties: %w", err)
	}
	return nil
}

// RegisterVariant registers a new variant for a specified model in etcd.
// It creates the necessary nodes and sets various properties for the variant.
//
// Parameters:
// - entity: The name of the entity to which the model belongs.
// - model: The name of the model to which the variant belongs.
// - variant: The name of the variant being registered.
// - jobFrequency: The frequency at which the job related to this variant should run.
// - vectorDbConfig: Configuration settings for the vector database.
// - vectorDbType: The type of vector database being used.
// - filter: A list of criteria used to filter the data for the variant.
//
// Returns an error if the registration fails.
func (s *SkyeManager) RegisterVariant(entity string, model string, variant string,
	vectorDbConfig VectorDbConfig, vectorDbType string, filter []Criteria, variantType enums.Type,
	distributedCacheEnabled bool, distributedCacheTtl int, inMemoryCacheEnabled bool, inMemoryCacheTtl int, rtPartition int, rateLimiters RateLimiter) error {
	paths := make(map[string]interface{})

	// Marshal config
	vectorDbConfigJson, err := json.Marshal(vectorDbConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal vector DB config: %w", err)
	}
	filterMap := map[string][]Criteria{"criteria": filter}
	filterJson, err := json.Marshal(filterMap)
	if err != nil {
		return fmt.Errorf("failed to marshal filter: %w", err)
	}

	// Create variant node
	variantPath := fmt.Sprintf("/config/%s/entity/%s/models/%s/variants/%s", s.appName, entity, model, variant)
	s.etcd.CreateNode(fmt.Sprintf("%s/enabled", variantPath), true)
	s.etcd.CreateNode(fmt.Sprintf("%s/vector-db-type", variantPath), vectorDbType)

	// Set variant properties
	paths[fmt.Sprintf("%s/filter", variantPath)] = string(filterJson)
	//paths[fmt.Sprintf("%s/enabled", variantPath)] = true
	paths[fmt.Sprintf("%s/embedding-store-read-version", variantPath)] = s.etcd.GetConfigInstance().(*Skye).Entity[entity].Models[model].EmbeddingStoreVersion
	paths[fmt.Sprintf("%s/embedding-store-write-version", variantPath)] = s.etcd.GetConfigInstance().(*Skye).Entity[entity].Models[model].EmbeddingStoreVersion
	paths[fmt.Sprintf("%s/onboarded", variantPath)] = false
	paths[fmt.Sprintf("%s/vector-db-read-version", variantPath)] = 1
	paths[fmt.Sprintf("%s/vector-db-write-version", variantPath)] = 1
	paths[fmt.Sprintf("%s/rt-delta-processing", variantPath)] = true
	paths[fmt.Sprintf("%s/partial-hit-enabled", variantPath)] = true
	paths[fmt.Sprintf("%s/variant-state", variantPath)] = "COMPLETED"
	paths[fmt.Sprintf("%s/vector-db-config", variantPath)] = string(vectorDbConfigJson)
	//paths[fmt.Sprintf("%s/vector-db-type", variantPath)] = vectorDbType
	paths[fmt.Sprintf("%s/type", variantPath)] = variantType
	paths[fmt.Sprintf("%s/distributed-caching-enabled", variantPath)] = distributedCacheEnabled
	paths[fmt.Sprintf("%s/distributed-cache-TTL-seconds", variantPath)] = distributedCacheTtl
	paths[fmt.Sprintf("%s/in-memory-caching-enabled", variantPath)] = inMemoryCacheEnabled
	paths[fmt.Sprintf("%s/in-memory-cache-TTL-seconds", variantPath)] = inMemoryCacheTtl
	paths[fmt.Sprintf("%s/rt-partition", variantPath)] = rtPartition
	paths[fmt.Sprintf("%s/rate-limiters/burst-limit", variantPath)] = rateLimiters.BurstLimit
	paths[fmt.Sprintf("%s/rate-limiters/rate-limit", variantPath)] = rateLimiters.RateLimit
	// Create variant properties nodes
	if err := s.etcd.CreateNodes(paths); err != nil {
		return fmt.Errorf("failed to create variant properties: %w", err)
	}
	return nil
}

func (s *SkyeManager) UpdateEmbeddingVersion(entity, model string, version int) error {
	path := s.getModelPath(entity, model, "embedding-store-version")
	return s.etcd.SetValue(path, version)
}

func (s *SkyeManager) UpdateVariantEmbeddingStoreReadVersion(entity string, model string, variant string, version int) error {
	path := s.getVariantPath(entity, model, variant, "embedding-store-read-version")
	return s.etcd.SetValue(path, version)
}

func (s *SkyeManager) UpdateVariantEmbeddingStoreWriteVersion(entity string, model string, variant string, version int) error {
	path := s.getVariantPath(entity, model, variant, "embedding-store-write-version")
	return s.etcd.SetValue(path, version)
}

// UpdateVariantStates UpdateVariantStates updates the onboarded for the vector database of a specified variant.
// Returns an error if the update fails.
func (s *SkyeManager) UpdateVariantStates(entity string, model string, variant map[string]string, state int) error {
	variantPath := s.getModelPath(entity, model, "variant-state")
	for variantName := range variant {
		path := fmt.Sprintf("%s/%s", variantPath, variantName)
		err := s.etcd.SetValue(path, state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SkyeManager) UpdateVectorDbConfig(entity string, model string, variant string, vectorDbConfig VectorDbConfig) error {
	variantPath := s.getVariantPath(entity, model, variant, "vector-db-config")
	vectorDbConfigJson, err := json.Marshal(vectorDbConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal vector DB config: %w", err)
	}
	err = s.etcd.SetValue(variantPath, string(vectorDbConfigJson))
	if err != nil {
		return err
	}
	return nil
}

func (s *SkyeManager) UpdatePartitionsForEof(entity string, model string, partition string, eofState int) error {
	variantPath := s.getModelPath(entity, model, "partition-eof-state")
	path := fmt.Sprintf("%s/%s", variantPath, partition)
	err := s.etcd.SetValue(path, eofState)
	if err != nil {
		return err
	}
	return nil
}

func (s *SkyeManager) UpdatePartitionState(entity string, model string, partition string, state int) error {
	partitionPath := s.getModelPath(entity, model, "partition-states")
	path := fmt.Sprintf("%s/%s", partitionPath, partition)
	err := s.etcd.SetValue(path, state)
	if err != nil {
		return err
	}
	return nil
}

// getVariantPath constructs the etcd path based on the provided parameters and the last node.
// The last node is the final node in the path, such as "onboarded".
func (s *SkyeManager) getVariantPath(entity, model, variant, lastNode string) string {
	return fmt.Sprintf("/config/%s/entity/%s/models/%s/variants/%s/%s", s.appName, entity, model, variant, lastNode)
}

// getModelPath constructs the etcd path based on the provided parameters and the last node.
// The last node is the final node in the path, such as "embedding-store-version" or "operation-mapping".
func (s *SkyeManager) getModelPath(entity, model, lastNode string) string {
	return fmt.Sprintf("/config/%s/entity/%s/models/%s/%s", s.appName, entity, model, lastNode)
}

func (s *SkyeManager) GetRateLimiters() map[int]RateLimiter {
	RateLimiters := make(map[int]RateLimiter)
	etcdConfigs := s.etcd.GetConfigInstance().(*Skye)
	for _, models := range etcdConfigs.Entity {
		for _, modelConfig := range models.Models {
			for _, variantConfig := range modelConfig.Variants {
				RateLimiters[variantConfig.RTPartition] = variantConfig.RateLimiter
			}
		}
	}
	return RateLimiters
}

func (s *SkyeManager) RegisterWatchPathCallbackWithEvent(path string, callback func(key, value, eventType string) error) error {
	return s.etcd.RegisterWatchPathCallbackWithEvent(path, callback)
}

func (s *SkyeManager) UpdateRateLimiter(entity string, model string, variant string, burstLimit int, rateLimit int) error {
	variantPath := fmt.Sprintf("/config/%s/entity/%s/models/%s/variants/%s", s.appName, entity, model, variant)
	if err := s.etcd.SetValue(fmt.Sprintf("%s/rate-limiter/burst-limit", variantPath), burstLimit); err != nil {
		return fmt.Errorf("failed to update rate limiter burst limit: %w", err)
	}
	if err := s.etcd.SetValue(fmt.Sprintf("%s/rate-limiter/rate-limit", variantPath), rateLimit); err != nil {
		return fmt.Errorf("failed to update rate limiter rate limit: %w", err)
	}
	return nil
}
