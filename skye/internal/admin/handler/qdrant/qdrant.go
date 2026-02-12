package qdrant

import (
	"fmt"
	"math/rand"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/workflow"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

var (
	once sync.Once
	db   Db
)

type Qdrant struct {
	configManager config.Manager
	StateMachine  workflow.StateMachine
}

func initQdrantHandler() Db {
	if db == nil {
		once.Do(func() {
			db = &Qdrant{
				configManager: config.NewManager(config.DefaultVersion),
				StateMachine:  workflow.NewStateMachine(1),
			}
		})
	}
	return db
}

func (q *Qdrant) CreateCollection(request *CreateCollectionRequest) error {
	err := vector.GetRepository(enums.QDRANT).CreateCollection(request.Entity, request.Model, request.Variant, request.Version)
	if err != nil {
		log.Error().Msgf("error creating collection for %s , %s, %s", request.Entity, request.Model, request.Variant)
		return fmt.Errorf("error creating collection for %s , %s, %s", request.Entity, request.Model, request.Variant)
	}
	return nil
}

func (q *Qdrant) TriggerIndexing(request *TriggerIndexingRequest) error {
	log.Error().Msgf("Triggered Indexing for Partition for %s , %s, %s", request.Entity, request.Model, request.Partition)
	err := q.configManager.UpdatePartitionState(request.Entity, request.Model, request.Partition, 1)
	if err != nil {
		log.Error().Msgf("error updating partition state for %s , %s, %s", request.Entity, request.Model, request.Partition)
		return err
	}
	currentPartition, err := strconv.Atoi(request.Partition)
	if err != nil {
		log.Error().Msgf("error converting partition to int for %s , %s, %s", request.Entity, request.Model, request.Partition)
		return err
	}
	sleepDuration := time.Duration(5+rand.Intn(6)) * time.Second // random between 5 and 10 seconds
	time.Sleep(sleepDuration)
	modelConfig, _ := q.configManager.GetModelConfig(request.Entity, request.Model)
	isAllPartitionsTrue := true
	for partition := range modelConfig.NumberOfPartitions {
		if partition != currentPartition && modelConfig.PartitionStates[strconv.Itoa(partition)] == 0 {
			isAllPartitionsTrue = false
			break
		}
	}
	if !isAllPartitionsTrue {
		return nil
	}
	log.Error().Msgf("Data Ingestion Completed from All Partitions %s , %s", request.Entity, request.Model)
	for variant, version := range request.VariantsVersionMap {
		variantConf, err := q.configManager.GetVariantConfig(request.Entity, request.Model, variant)
		if err != nil {
			return err
		}
		if variantConf.VariantState != enums.DATA_INGESTION_STARTED {
			log.Error().Msgf("Indexing Workflow has already started for %s , %s, %s", request.Entity, request.Model, variant)
			return nil
		}
		payload := workflow.ModelStateExecutorPayload{
			Entity:                request.Entity,
			Model:                 request.Model,
			Variant:               variant,
			Version:               version,
			VariantState:          variantConf.VariantState,
			EmbeddingStoreVersion: request.EmbeddingStoreVersion,
		}
		go q.StateMachine.ProcessStates(&payload)
		log.Error().Msgf("State Machine Processed for %s , %s, %s", request.Entity, request.Model, variant)
	}
	return nil
}

// validateVariantsCompleted checks if all variants in the model are completed
func (q *Qdrant) validateVariantsCompleted(entity, model string) error {
	modelConfig, _ := q.configManager.GetModelConfig(entity, model)
	for variantName, variantConfig := range modelConfig.Variants {
		if variantConfig.VariantState != enums.COMPLETED {
			log.Error().Msgf("variant %s is not completed for %s , %s", variantName, entity, model)
			return fmt.Errorf("variant %s is not completed for %s , %s", variantName, entity, model)
		}
	}
	return nil
}

// resetAllPartitions sets all partitions to false for a given model
func (q *Qdrant) resetAllPartitions(entity, model string) error {
	modelConfig, _ := q.configManager.GetModelConfig(entity, model)
	for partition := range modelConfig.NumberOfPartitions {
		err := q.configManager.UpdatePartitionState(entity, model, strconv.Itoa(partition), 0)
		if err != nil {
			return err
		}
	}
	return nil
}

// prepareEmbeddingStoreVersion handles embedding store version based on model type
func (q *Qdrant) prepareEmbeddingStoreVersion(entity, model string) (int, error) {
	modelConfig, _ := q.configManager.GetModelConfig(entity, model)
	embeddingStoreVersion := modelConfig.EmbeddingStoreVersion

	if modelConfig.ModelType == enums.RESET {
		embeddingStoreVersion++
		err := q.configManager.UpdateEmbeddingVersion(entity, model, embeddingStoreVersion)
		if err != nil {
			return 0, err
		}
	}
	return embeddingStoreVersion, nil
}

func (q *Qdrant) ProcessModel(request *ProcessModelRequest) (*ProcessModelResponse, error) {
	// Validate all variants are completed
	if err := q.validateVariantsCompleted(request.Entity, request.Model); err != nil {
		return nil, err
	}

	// Reset all partitions
	if err := q.resetAllPartitions(request.Entity, request.Model); err != nil {
		return nil, err
	}

	// Prepare embedding store version
	embeddingStoreVersion, err := q.prepareEmbeddingStoreVersion(request.Entity, request.Model)
	if err != nil {
		return nil, err
	}

	// Process variants with matching vector DB type
	variantsToProcess := make(map[string]int)
	modelConfig, _ := q.configManager.GetModelConfig(request.Entity, request.Model)

	for variantName, variantConfig := range modelConfig.Variants {

		if variantConfig.Enabled && variantConfig.VectorDbType == request.VectorDbType {
			if err := q.processVariant(variantName, modelConfig.JobFrequency, request.Entity, request.Model, "", embeddingStoreVersion, modelConfig.ModelType, variantsToProcess, false, false); err != nil {
				log.Error().Msgf("Error processing variant %s for %s , %s", variantName, request.Entity, request.Model)
				metric.Count("variant_processing_error", 1, []string{"entity_name", request.Entity, "model_name", request.Model, "variant_name", variantName})
				continue
			}
		}

	}

	return q.buildProcessModelResponse(request.Entity, request.Model, embeddingStoreVersion, variantsToProcess)
}

func (q *Qdrant) ProcessModelsWithFrequency(request *ProcessModelsWithFrequencyRequest) (*ProcessModelsWithFrequencyResponse, error) {
	entity, _ := q.configManager.GetEntityConfig(request.Entity)
	response := &ProcessModelsWithFrequencyResponse{
		ProcessModelResponse: nil,
	}
	for model, modelConfig := range entity.Models {
		if modelConfig.JobFrequency != request.Frequency {
			continue
		}
		response.ProcessModelResponse = append(response.ProcessModelResponse, ProcessFreqModelResponse{
			Model: model,
		})
	}
	return response, nil
}

func (q *Qdrant) ProcessMultiVariant(request *ProcessMultiVariantRequest) (*ProcessModelResponse, error) {

	modelConfig, _ := q.configManager.GetModelConfig(request.Entity, request.Model)
	for _, variant := range request.Variants {
		if _, ok := modelConfig.Variants[variant]; !ok {
			log.Error().Msgf("variant %s is not present for %s , %s", variant, request.Entity, request.Model)
			return nil, fmt.Errorf("variant %s is not present for %s , %s", variant, request.Entity, request.Model)
		}
		variantConfig, _ := q.configManager.GetVariantConfig(request.Entity, request.Model, variant)
		if !variantConfig.Enabled {
			continue
		}
		if variantConfig.VariantState != enums.COMPLETED {
			log.Error().Msgf("variant %s is not completed for %s , %s", variant, request.Entity, request.Model)
			return nil, fmt.Errorf("variant %s is not completed for %s , %s", variant, request.Entity, request.Model)
		}
	}

	variantsToProcess := make(map[string]int)
	if err := q.resetAllPartitions(request.Entity, request.Model); err != nil {
		return nil, err
	}
	// Prepare embedding store version
	embeddingStoreVersion, err := q.prepareEmbeddingStoreVersion(request.Entity, request.Model)
	if err != nil {
		return nil, err
	}

	for _, variant := range request.Variants {
		variantConfig, _ := q.configManager.GetVariantConfig(request.Entity, request.Model, variant)
		if !variantConfig.Enabled {
			continue
		}
		// Process the specific variant
		if err := q.processVariant(variant, modelConfig.JobFrequency, request.Entity, request.Model, "", embeddingStoreVersion, modelConfig.ModelType, variantsToProcess, false, false); err != nil {
			log.Error().Msgf("Error processing variant %s for %s , %s", variant, request.Entity, request.Model)
			metric.Count("variant_processing_error", 1, []string{"entity_name", request.Entity, "model_name", request.Model, "variant_name", variant})
			continue
		}
	}

	return q.buildProcessModelResponse(request.Entity, request.Model, embeddingStoreVersion, variantsToProcess)
}

func (q *Qdrant) PromoteVariant(request *PromoteVariantRequest) (*ProcessModelResponse, error) {
	// Validate specific variant is completed
	variantConfig, _ := q.configManager.GetVariantConfig(request.Entity, request.Model, request.Variant)
	if !variantConfig.Enabled {
		return nil, fmt.Errorf("variant %s is not enabled for %s , %s", request.Variant, request.Entity, request.Model)
	}
	if variantConfig.VariantState != enums.COMPLETED {
		log.Error().Msgf("variant %s is not completed for %s , %s", request.Variant, request.Entity, request.Model)
		return nil, fmt.Errorf("variant %s is not completed for %s , %s", request.Variant, request.Entity, request.Model)
	}
	if variantConfig.Type != enums.EXPERIMENT {
		log.Error().Msgf("variant %s is not in experiment state for %s , %s", request.Variant, request.Entity, request.Model)
		return nil, fmt.Errorf("variant %s is not in experiment state for %s , %s", request.Variant, request.Entity, request.Model)
	}
	vectorDbConfig := variantConfig.VectorDbConfig
	vectorDbConfig.WriteHost = request.Host
	if err := q.configManager.UpdateVectorDbConfig(request.Entity, request.Model, request.Variant, vectorDbConfig); err != nil {
		log.Error().Msgf("Error updating vector DB config for %s, %s, %s", request.Entity, request.Model, request.Variant)
		return nil, err
	}
	time.Sleep(10 * time.Second)

	// Reset all partitions
	if err := q.resetAllPartitions(request.Entity, request.Model); err != nil {
		return nil, err
	}

	// Prepare embedding store version
	modelConfig, _ := q.configManager.GetModelConfig(request.Entity, request.Model)
	embeddingStoreVersion := modelConfig.EmbeddingStoreVersion

	// Process the specific variant
	variantsToProcessFreqMap := make(map[string]int)

	if err := q.processVariant(request.Variant, modelConfig.JobFrequency, request.Entity, request.Model, "", embeddingStoreVersion, modelConfig.ModelType, variantsToProcessFreqMap, true, false); err != nil {
		return nil, err
	}

	return q.buildProcessModelResponse(request.Entity, request.Model, embeddingStoreVersion, variantsToProcessFreqMap)
}

func (q *Qdrant) ProcessMultiVariantForceReset(request *ProcessMultiVariantRequest) (*ProcessModelResponse, error) {

	modelConfig, _ := q.configManager.GetModelConfig(request.Entity, request.Model)

	if modelConfig.ModelType == enums.RESET {
		return nil, fmt.Errorf("model type is reset for %s , %s. Force reset is not supported for reset model type", request.Entity, request.Model)
	}

	for _, variant := range request.Variants {
		if _, ok := modelConfig.Variants[variant]; !ok {
			log.Error().Msgf("variant %s is not present for %s , %s", variant, request.Entity, request.Model)
			return nil, fmt.Errorf("variant %s is not present for %s , %s", variant, request.Entity, request.Model)
		}
		variantConfig, _ := q.configManager.GetVariantConfig(request.Entity, request.Model, variant)
		if !variantConfig.Enabled {
			continue
		}
		if variantConfig.VariantState != enums.COMPLETED {
			log.Error().Msgf("variant %s is not completed for %s , %s", variant, request.Entity, request.Model)
			return nil, fmt.Errorf("variant %s is not completed for %s , %s", variant, request.Entity, request.Model)
		}
	}

	variantsToProcess := make(map[string]int)

	// Reset all partitions
	if err := q.resetAllPartitions(request.Entity, request.Model); err != nil {
		return nil, err
	}

	// Prepare embedding store version
	embeddingStoreVersion, err := q.prepareEmbeddingStoreVersion(request.Entity, request.Model)
	if err != nil {
		return nil, err
	}

	for _, variant := range request.Variants {
		variantConfig, _ := q.configManager.GetVariantConfig(request.Entity, request.Model, variant)
		if !variantConfig.Enabled {
			continue
		}
		// Process the specific variant
		if err := q.processVariant(variant, modelConfig.JobFrequency, request.Entity, request.Model, "", embeddingStoreVersion, modelConfig.ModelType, variantsToProcess, false, true); err != nil {
			log.Error().Msgf("Error processing variant %s for %s , %s", variant, request.Entity, request.Model)
			metric.Count("variant_processing_error", 1, []string{"entity_name", request.Entity, "model_name", request.Model, "variant_name", variant})
			continue
		}
	}
	return q.buildProcessModelResponse(request.Entity, request.Model, embeddingStoreVersion, variantsToProcess)
}

func (q *Qdrant) processVariant(variant, variantFrequency, entity, model, requestFrequency string, embeddingStoreVersion int, modelType enums.ModelType, variantsToProcess map[string]int, isPromote bool, isForceReset bool) (err error) {
	originalConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting original config for %s, %s, %s", entity, model, variant)
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			q.revertConfig(originalConfig, entity, model, variant)
			if e, ok := r.(error); ok {
				err = fmt.Errorf("panic while processing variant (entity=%s model=%s variant=%s): %w", entity, model, variant, e)
			} else {
				err = fmt.Errorf("panic while processing variant (entity=%s model=%s variant=%s): %v", entity, model, variant, r)
			}
			err = fmt.Errorf("%w\n%s", err, string(debug.Stack()))
		}

		if err != nil {
			q.revertConfig(originalConfig, entity, model, variant)
		}
	}()

	if originalConfig.VectorDbConfig.Params["after_collection_index_payload"] != "" && originalConfig.VectorDbConfig.Params["after_collection_index_payload"] == "true" {
		err = q.configManager.UpdateRateLimiter(entity, model, variant, 0, 0)
		if err != nil {
			log.Error().Msgf("Error updating rate limiter for %s, %s, %s", entity, model, variant)
			return err
		}
	}

	err = q.configManager.UpdateVariantEmbeddingStoreWriteVersion(entity, model, variant, embeddingStoreVersion)
	if err != nil {
		log.Error().Msgf("Error updating variant embedding store write version for %s, %s, %s", entity, model, variant)
		return err
	}
	if variantFrequency == requestFrequency || requestFrequency == "" {
		variantConfig, _ := q.configManager.GetVariantConfig(entity, model, variant)
		version := variantConfig.VectorDbReadVersion
		if !isPromote && variantConfig.VectorDbConfig.WriteHost != variantConfig.VectorDbConfig.ReadHost {
			log.Error().Msgf("write host and read host are not the same for %s, %s, %s", entity, model, variant)
			metric.Count("vector_db_host_mismatch", 1, []string{"entity_name", entity, "model_name", model, "variant_name", variant})
			return nil
		}

		// Handle model type specific logic
		if modelType == enums.RESET {
			err = q.handleResetModelType(entity, model, variant, version, variantsToProcess, isPromote)
			if err != nil {
				log.Error().Msgf("Error handling reset model type for %s, %s, %s", entity, model, variant)
				return err
			}
		} else if modelType == enums.DELTA {
			err = q.handleDeltaModelType(variant, entity, model, version, variantsToProcess, isPromote, isForceReset)
			if err != nil {
				log.Error().Msgf("Error handling delta model type for %s, %s, %s", entity, model, variant)
				return err
			}
		}
		// Update variant and job state
		err = q.updateVariantAndJobState(entity, model, variant)
		if err != nil {
			log.Error().Msgf("Error updating variant and job state for %s, %s, %s", entity, model, variant)
			delete(variantsToProcess, variant)
			return err
		}
	}
	return nil
}

// createCollectionIfNeeded handles collection creation for onboarded and non-onboarded variants
func (q *Qdrant) createCollectionIfNeeded(entity, model, variant string, version int) error {
	createCollectionRequest := &CreateCollectionRequest{
		Entity:  entity,
		Model:   model,
		Variant: variant,
		Version: version,
	}
	return q.CreateCollection(createCollectionRequest)
}

// handleVariantOnboarding handles the common onboarding logic for variants
func (q *Qdrant) handleVariantOnboarding(entity, model, variant string, writeVersion int, isPromote bool) error {
	if err := q.createCollectionIfNeeded(entity, model, variant, writeVersion); err != nil {
		log.Error().Msgf("Error creating collection for %s, %s, %s", entity, model, variant)
		return err
	}
	if !isPromote {
		if err := q.configManager.SetVariantOnboarded(entity, model, variant, true); err != nil {
			log.Error().Msgf("Error updating variant Onboarded for %s, %s, %s", entity, model, variant)
			return err
		}
	}
	return nil
}

func (q *Qdrant) handleResetModelType(entity, model, variant string, version int, variantsToProcess map[string]int, isPromote bool) error {
	variantConfig, _ := q.configManager.GetVariantConfig(entity, model, variant)
	writeVersion := version

	if !variantConfig.Onboarded || isPromote {
		// Handle variant onboarding
		if err := q.handleVariantOnboarding(entity, model, variant, writeVersion, isPromote); err != nil {
			log.Error().Msgf("Error handling variant onboarding for %s, %s, %s", entity, model, variant)
			return err
		}
	} else {
		// Increment version and create new collection
		writeVersion++
		if err := q.createCollectionIfNeeded(entity, model, variant, writeVersion); err != nil {
			log.Error().Msgf("Error creating collection for %s, %s, %s", entity, model, variant)
			return err
		}
	}

	variantsToProcess[variant] = writeVersion
	return q.configManager.UpdateVariantWriteVersion(entity, model, variant, writeVersion)
}

func (q *Qdrant) handleDeltaModelType(variant, entity, model string, version int, variantsToProcessFreqMap map[string]int, isPromote bool, isForceReset bool) error {
	variantConfig, _ := q.configManager.GetVariantConfig(entity, model, variant)
	writeVersion := version
	if isForceReset {
		writeVersion++
	}

	if !variantConfig.Onboarded || isPromote || isForceReset {
		// Handle variant onboarding
		if err := q.handleVariantOnboarding(entity, model, variant, writeVersion, isPromote); err != nil {
			return err
		}
	} else {
		// Update indexing threshold for existing collection
		if err := vector.GetRepository(enums.QDRANT).UpdateIndexingThreshold(entity, model, variant, writeVersion, "0"); err != nil {
			log.Error().Msgf("Error updating indexing threshold for %s, %s, %s", entity, model, variant)
			return err
		}
	}

	variantsToProcessFreqMap[variant] = writeVersion
	return q.configManager.UpdateVariantWriteVersion(entity, model, variant, writeVersion)
}

func (q *Qdrant) updateVariantAndJobState(entity, model, variant string) error {
	if err := q.configManager.UpdateVariantState(entity, model, variant, enums.DATA_INGESTION_STARTED); err != nil {
		log.Error().Msgf("Error updating variant state for %s, %s, %s", entity, model, variant)
		return err
	}
	metric.Gauge("variant_state", 1, []string{"entity_name", entity, "model_name", model, "variant_name", variant})
	return nil
}

func (q *Qdrant) buildProcessModelResponse(entity, model string, embeddingStoreVersion int, variantsToProcessFreqMap map[string]int) (*ProcessModelResponse, error) {
	modelConfig, _ := q.configManager.GetModelConfig(entity, model)
	kafkaId := modelConfig.KafkaId
	topicName := modelConfig.TopicName
	trainingDataPath := modelConfig.TrainingDataPath
	numberOfPartitions := modelConfig.NumberOfPartitions
	modelType := modelConfig.ModelType
	if len(variantsToProcessFreqMap) == 0 {
		return nil, nil
	}
	return &ProcessModelResponse{
		Variants:              variantsToProcessFreqMap,
		KafkaId:               kafkaId,
		TrainingDataPath:      trainingDataPath,
		EmbeddingStoreVersion: embeddingStoreVersion,
		Model:                 model,
		NumberOfPartitions:    numberOfPartitions,
		TopicName:             topicName,
		ModelType:             modelType,
	}, nil
}

func (q *Qdrant) PublishCollectionMetrics() error {
	isCollectionMetricEnabled := appConfig.CollectionMetricEnabled
	if isCollectionMetricEnabled {
		ticker := time.NewTicker(time.Duration(appConfig.CollectionMetricPublish) * time.Second)
		defer func() {
			if r := recover(); r != nil {
				panicErr := fmt.Errorf("panic occurred: %v", r)
				log.Error().Msgf("%s", panicErr)
			}
			ticker.Stop()
			q.startTicker(ticker)
		}()
		q.startTicker(ticker)
	}
	return nil
}

func (q *Qdrant) startTicker(ticker *time.Ticker) error {
	for range ticker.C {
		entities, err := q.configManager.GetEntities()
		if err != nil {
			log.Error().Msgf("Error getting entities")
			continue
		}
		for entity, modelMap := range entities {
			for model, models := range modelMap.Models {
				metric.Count("model_config", 1, []string{"entity_name", entity, "model_name", model, "kafka_id", strconv.Itoa(models.KafkaId), "model_type", string(models.ModelType)})
				for Variant, variant := range models.Variants {
					variantConfig, _ := q.configManager.GetVariantConfig(entity, model, Variant)
					variantType := variantConfig.Type
					metric.Count("variant_cache_config_similarity_search", 1, []string{"entity_name", entity, "model_name", model, "variant_name", Variant,
						"in_memory_enabled", strconv.FormatBool(variantConfig.InMemoryCachingEnabled), "in_memory_ttl", strconv.Itoa(variantConfig.InMemoryCacheTTLSeconds),
						"distributed_enabled", strconv.FormatBool(variantConfig.DistributedCachingEnabled), "distributed_ttl", strconv.Itoa(variantConfig.DistributedCacheTTLSeconds)})
					if variant.VectorDbType == enums.QDRANT {
						if variant.Enabled && variant.Onboarded {
							readCollectionInfo, err := vector.GetRepository(enums.QDRANT).GetReadCollectionInfo(entity, model, Variant, variant.VectorDbReadVersion)
							metricTags := []string{"entity_name", entity, "model_name", model, "variant_name", Variant, "variant_type", string(variantType)}
							if err != nil || readCollectionInfo == nil {
								log.Error().Msgf("error getting read collection info for %s, %s, %s", entity, model, Variant)
								continue
							}
							metric.Gauge("qdrant_read_points_count", readCollectionInfo.PointsCount, metricTags)
							metric.Gauge("qdrant_read_indexed_vector_count", readCollectionInfo.IndexedVectorsCount, metricTags)
						}
						metric.Count("variant_config", 1, []string{"entity_name", entity, "model_name", model, "variant_name", Variant, "read_host", variant.VectorDbConfig.ReadHost, "write_host", variant.VectorDbConfig.WriteHost, "variant_type", string(variantType)})
					}
				}
			}
		}
	}
	return nil
}

func (q *Qdrant) revertConfig(originalConfig *config.Variant, entity string, model string, variant string) {
	if originalConfig == nil {
		log.Error().Msgf("original config is nil for %s, %s, %s", entity, model, variant)
		return
	}
	err := q.configManager.UpdateVariantEmbeddingStoreWriteVersion(entity, model, variant, originalConfig.EmbeddingStoreWriteVersion)
	if err != nil {
		log.Error().Err(err).Msgf("Error reverting variant embedding store write version for %s, %s, %s", entity, model, variant)
	}
	err = q.configManager.UpdateVariantWriteVersion(entity, model, variant, originalConfig.VectorDbWriteVersion)
	if err != nil {
		log.Error().Err(err).Msgf("Error reverting variant write version for %s, %s, %s", entity, model, variant)
	}
	err = q.configManager.UpdateVariantState(entity, model, variant, originalConfig.VariantState)
	if err != nil {
		log.Error().Err(err).Msgf("Error reverting variant state for %s, %s, %s", entity, model, variant)
	}
	err = q.configManager.UpdateRateLimiter(entity, model, variant, originalConfig.RateLimiter.BurstLimit, originalConfig.RateLimiter.RateLimit)
	if err != nil {
		log.Error().Err(err).Msgf("Error reverting rate limiter for %s, %s, %s", entity, model, variant)
	}
	if originalConfig.Onboarded {
		err = vector.GetRepository(enums.QDRANT).UpdateIndexingThreshold(entity, model, variant, originalConfig.VectorDbWriteVersion, "100")
		if err != nil {
			log.Error().Err(err).Msgf("Error reverting indexing threshold for %s, %s, %s", entity, model, variant)
		}
	}
}
