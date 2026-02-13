package workflow

import (
	"encoding/json"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	skafka "github.com/Meesho/BharatMLStack/skye/pkg/kafka"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

// sleepFunc is a package-level variable for time.Sleep, overridable in tests.
var sleepFunc = time.Sleep

type ModelStateMachine struct {
	configManager config.Manager
}

func initModelStateMachine() StateMachine {
	if machine == nil {
		once.Do(func() {
			machine = &ModelStateMachine{
				configManager: config.NewManager(config.DefaultVersion),
			}
		})
	}
	return machine
}

func (msm *ModelStateMachine) ProcessStates(payload *ModelStateExecutorPayload) error {
	currentState := payload.VariantState
	log.Error().Msgf("Processing %s State for %s %s %s", currentState, payload.Entity, payload.Model, payload.Variant)
	if payload.VariantState == "" {
		log.Error().Msgf("Job Completed for %s %s %s", payload.Entity, payload.Model, payload.Variant)
		return nil
	}
	err := msm.process(currentState, payload)
	if err != nil {
		return err
	}
	return nil
}

func (msm *ModelStateMachine) process(currentState enums.VariantState, payload *ModelStateExecutorPayload) error {
	newState, counter, err := msm.ProcessState(payload)
	if err != nil {
		log.Error().Msgf("Error in State Processing %s", err)
		return err
	}
	payload.VariantState = newState
	payload.Counter = counter
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Error().Msgf("Error in Marshalling %s", err)
		return err
	}
	keyStr := ""
	payloadToProduce := []skafka.ProducerMessage{
		{
			Key:     &keyStr,
			Value:   jsonPayload,
			Headers: make(map[string][]byte),
		},
	}
	err = skafka.SendAndForget(appConfig.ModelStateProducer, payloadToProduce)
	if err != nil {
		return err
	}
	log.Error().Msgf("%s State Processed for %s %s %s", currentState, payload.Entity, payload.Model, payload.Variant)
	return nil
}

func (msm *ModelStateMachine) ProcessState(payload *ModelStateExecutorPayload) (enums.VariantState, int, error) {
	variantConfig, err := msm.configManager.GetVariantConfig(payload.Entity, payload.Model, payload.Variant)
	if err != nil {
		return "", 0, err
	}
	vectorDbType := variantConfig.VectorDbType

	switch payload.VariantState {
	case enums.DATA_INGESTION_STARTED:
		return msm.handleDataIngestionStarted(payload)
	case enums.DATA_INGESTION_COMPLETED:
		return msm.handleDataIngestionCompleted(payload, vectorDbType)
	case enums.INDEXING_STARTED:
		return msm.handleIndexingStarted(payload, vectorDbType)
	case enums.INDEXING_IN_PROGRESS:
		return msm.handleIndexingInProgress(payload, vectorDbType)
	case enums.INDEXING_COMPLETED_WITH_RESET:
		return msm.handleIndexingCompletedWithReset(payload)
	case enums.MODEL_VERSION_UPDATED:
		return msm.handleModelVersionUpdated(payload, vectorDbType)
	case enums.INDEXING_COMPLETED:
		return msm.handleIndexingCompleted(payload)
	case enums.COMPLETED:
		return msm.handleCompleted(payload)
	default:
		return "", 0, nil
	}
}

func (msm *ModelStateMachine) handleDataIngestionStarted(payload *ModelStateExecutorPayload) (enums.VariantState, int, error) {
	log.Info().Msgf("Data Ingestion Started for %s %s %s", payload.Entity, payload.Model, payload.Variant)
	err := msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, enums.DATA_INGESTION_COMPLETED)
	if err != nil {
		return "", 0, err
	}
	metric.Gauge("variant_state", 2, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
	return enums.DATA_INGESTION_COMPLETED, 0, nil
}

func (msm *ModelStateMachine) handleDataIngestionCompleted(payload *ModelStateExecutorPayload, vectorDbType enums.VectorDbType) (enums.VariantState, int, error) {
	log.Info().Msgf("Data Ingestion Completed for %s %s %s", payload.Entity, payload.Model, payload.Variant)
	variantConfig, _ := msm.configManager.GetVariantConfig(payload.Entity, payload.Model, payload.Variant)
	variantParams := variantConfig.VectorDbConfig.Params
	err := vector.GetRepository(vectorDbType).UpdateIndexingThreshold(payload.Entity, payload.Model, payload.Variant, payload.Version, variantParams["indexing_threshold"])
	if err != nil {
		return "", 0, err
	}
	sleepFunc(30 * time.Second)
	err = msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, enums.INDEXING_STARTED)
	if err != nil {
		return "", 0, err
	}
	metric.Gauge("variant_state", 3, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
	return enums.INDEXING_STARTED, 0, nil
}

func (msm *ModelStateMachine) handleIndexingStarted(payload *ModelStateExecutorPayload, vectorDbType enums.VectorDbType) (enums.VariantState, int, error) {
	log.Error().Msgf("Indexing Started for %s %s %s %v", payload.Entity, payload.Model, payload.Variant, payload.Counter)
	sleepFunc(30 * time.Second)
	counter := payload.Counter
	response, _ := vector.GetRepository(vectorDbType).GetCollectionInfo(payload.Entity, payload.Model, payload.Variant, payload.Version)
	log.Error().Msgf("Collection Info for %s %s %s is %v", payload.Entity, payload.Model, payload.Variant, response)
	isIndexedScaleUp := false
	if response != nil && response.IndexedVectorsCount/response.PointsCount > 0.95 {
		isIndexedScaleUp = true
	}
	if isIndexedScaleUp {
		counter++
	}
	if counter == 2 {
		sleepFunc(30 * time.Second)
		variantConfig, _ := msm.configManager.GetVariantConfig(payload.Entity, payload.Model, payload.Variant)
		if variantConfig.VectorDbConfig.Params["after_collection_index_payload"] != "" && variantConfig.VectorDbConfig.Params["after_collection_index_payload"] == "true" {
			vector.GetRepository(vectorDbType).UpdateIndexingThreshold(payload.Entity, payload.Model, payload.Variant, payload.Version, "0")
			sleepFunc(10 * time.Second)
			err := vector.GetRepository(vectorDbType).CreateFieldIndexes(payload.Entity, payload.Model, payload.Variant, payload.Version)
			if err != nil {
				return "", 0, err
			}
		}
		err := msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, enums.INDEXING_IN_PROGRESS)
		if err != nil {
			return "", 0, err
		}
		metric.Gauge("variant_state", 4, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
		return enums.INDEXING_IN_PROGRESS, 0, nil
	} else {
		return enums.INDEXING_STARTED, counter, nil
	}
}

func (msm *ModelStateMachine) handleIndexingInProgress(payload *ModelStateExecutorPayload, vectorDbType enums.VectorDbType) (enums.VariantState, int, error) {
	log.Info().Msgf("Indexing In Progress for %s %s %s", payload.Entity, payload.Model, payload.Variant)
	variantConfig, _ := msm.configManager.GetVariantConfig(payload.Entity, payload.Model, payload.Variant)
	variantParams := variantConfig.VectorDbConfig.Params
	if variantConfig.VectorDbConfig.Params["after_collection_index_payload"] != "" && variantConfig.VectorDbConfig.Params["after_collection_index_payload"] == "true" {
		counter := payload.Counter
		response, _ := vector.GetRepository(vectorDbType).GetCollectionInfo(payload.Entity, payload.Model, payload.Variant, payload.Version)
		isPayloadIndexedScaleUp := false
		for _, payloadPointsCount := range response.PayloadPointsCount {
			if response != nil && payloadPointsCount/response.PointsCount > 0.95 {
				isPayloadIndexedScaleUp = true
			}
		}
		if isPayloadIndexedScaleUp {
			counter++
		}
		if counter != 5 {
			return enums.INDEXING_IN_PROGRESS, counter, nil
		}
	}
	err := vector.GetRepository(vectorDbType).UpdateIndexingThreshold(payload.Entity, payload.Model, payload.Variant, payload.Version, variantParams["default_indexing_threshold"])
	if err != nil {
		return "", 0, err
	}
	modelConfig, _ := msm.configManager.GetModelConfig(payload.Entity, payload.Model)
	modelType := modelConfig.ModelType
	nextState := enums.INDEXING_COMPLETED
	if modelType == enums.RESET {
		nextState = enums.INDEXING_COMPLETED_WITH_RESET
	}
	err = msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, nextState)
	if err != nil {
		return "", 0, err
	}
	metric.Gauge("variant_state", 5, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
	return nextState, 0, nil
}

func (msm *ModelStateMachine) handleIndexingCompletedWithReset(payload *ModelStateExecutorPayload) (enums.VariantState, int, error) {
	log.Info().Msgf("Indexing Completed with Reset for %s %s %s", payload.Entity, payload.Model, payload.Variant)
	err := msm.configManager.UpdateVariantReadVersion(payload.Entity, payload.Model, payload.Variant, payload.Version)
	if err != nil {
		return "", 0, err
	}
	err = msm.configManager.UpdateVariantEmbeddingStoreReadVersion(payload.Entity, payload.Model, payload.Variant, payload.EmbeddingStoreVersion)
	if err != nil {
		return "", 0, err
	}
	err = msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, enums.MODEL_VERSION_UPDATED)
	if err != nil {
		return "", 0, err
	}
	metric.Gauge("variant_state", 7, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
	return enums.MODEL_VERSION_UPDATED, 0, nil
}

func (msm *ModelStateMachine) handleModelVersionUpdated(payload *ModelStateExecutorPayload, vectorDbType enums.VectorDbType) (enums.VariantState, int, error) {
	log.Info().Msgf("Model Version Updated for %s %s %s", payload.Entity, payload.Model, payload.Variant)
	err := msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, enums.INDEXING_COMPLETED)
	if err != nil {
		return "", 0, err
	}
	versionToDelete := payload.Version - 1
	log.Info().Msgf("Deleting Version %d for %s %s %s", versionToDelete, payload.Entity, payload.Model, payload.Variant)
	// Todo: Try to delete multiple times if it fails (mainly 3 times)
	err = vector.GetRepository(vectorDbType).DeleteCollection(payload.Entity, payload.Model, payload.Variant, versionToDelete)
	if err != nil {
		return "", 0, err
	}
	variantConfig, _ := msm.configManager.GetVariantConfig(payload.Entity, payload.Model, payload.Variant)
	if variantConfig.VectorDbConfig.Params["after_collection_index_payload"] != "" && variantConfig.VectorDbConfig.Params["after_collection_index_payload"] == "true" {
		err := msm.configManager.UpdateRateLimiter(payload.Entity, payload.Model, payload.Variant, 0, 0)
		if err != nil {
			return "", 0, err
		}
	}
	metric.Gauge("variant_state", 6, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
	return enums.INDEXING_COMPLETED, 0, nil
}

func (msm *ModelStateMachine) handleIndexingCompleted(payload *ModelStateExecutorPayload) (enums.VariantState, int, error) {
	log.Info().Msgf("Indexing Completed for %s %s %s", payload.Entity, payload.Model, payload.Variant)

	err := msm.configManager.UpdateVariantState(payload.Entity, payload.Model, payload.Variant, enums.COMPLETED)
	if err != nil {
		return "", 0, err
	}
	metric.Gauge("variant_state", 8, []string{"entity_name", payload.Entity, "model_name", payload.Model, "variant_name", payload.Variant})
	return enums.COMPLETED, 0, nil
}

func (msm *ModelStateMachine) handleCompleted(payload *ModelStateExecutorPayload) (enums.VariantState, int, error) {
	log.Info().Msgf("Processing Completed for %s %s %s", payload.Entity, payload.Model, payload.Variant)
	return "", 0, nil
}
