package embedding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/handler/indexer"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/aggregator"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	"github.com/Meesho/BharatMLStack/skye/pkg/httpclient"
	skafka "github.com/Meesho/BharatMLStack/skye/pkg/kafka"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

var (
	embeddingConsumer Consumer
	embeddingOnce     sync.Once
)

type EmbeddingConsumer struct {
	qdrantIndexerHandler indexer.Handler
	aggregatorDb         aggregator.Database
	embeddingStore       embedding.Store
	configManager        config.Manager
	httpClient           *httpclient.HTTPClient
	AppConfig            *structs.AppConfig
}

func newEmbeddingConsumer() Consumer {
	if embeddingConsumer == nil {
		embeddingOnce.Do(func() {
			qdrantIndexerHandler := indexer.NewHandler(indexer.QDRANT)
			aggregatorDb := aggregator.NewRepository(aggregator.DefaultVersion)
			configManager := config.NewManager(config.DefaultVersion)
			embeddingStore := embedding.NewRepository(embedding.DefaultVersion)
			embeddingConsumer = &EmbeddingConsumer{
				qdrantIndexerHandler: qdrantIndexerHandler,
				aggregatorDb:         aggregatorDb,
				embeddingStore:       embeddingStore,
				configManager:        configManager,
				httpClient:           httpclient.NewConn("ADMIN"),
				AppConfig:            structs.GetAppConfig(),
			}
		})
	}
	return embeddingConsumer
}

func (e *EmbeddingConsumer) produceFailureEvents(failedEvents []Event) {
	for _, failedEvent := range failedEvents {
		metric.Incr("embedding_consumer_event_error", []string{"entity_name", failedEvent.Entity, "model_name", failedEvent.Model})
		modelConf, err := e.configManager.GetModelConfig(failedEvent.Entity, failedEvent.Model)
		if err != nil {
			log.Error().Err(err).Msg("Error getting model config")
			continue
		}
		failureProducerKafkaId := modelConf.FailureProducerKafkaId
		skafka.InitProducer(failureProducerKafkaId) // idempotent — ensures producer exists for this dynamic ID
		jsonBytes, err := json.Marshal(failedEvent)
		if err != nil {
			log.Error().Msgf("Error marshalling failed event: %v", err)
			return
		}
		keyStr := ""
		msgs := []skafka.ProducerMessage{
			{
				Key:     &keyStr,
				Value:   jsonBytes,
				Headers: make(map[string][]byte),
			},
		}
		sendErr := skafka.SendAndForget(failureProducerKafkaId, msgs)
		if sendErr != nil {
			log.Error().Err(sendErr).Int("failed_count", len(msgs)).Int("producer_kafka_id", failureProducerKafkaId).Msg("Error producing failed embedding events batch to failure topic")
			metric.Incr("embedding_failure_producer_event_error", []string{"entity_name", failedEvent.Entity, "model_name", failedEvent.Model})
		} else {
			log.Info().Int("produced_count", len(msgs)).Int("producer_kafka_id", failureProducerKafkaId).Msg("Successfully produced failed embedding events batch to failure topic")
			metric.Incr("embedding_failure_producer_event_success", []string{"entity_name", failedEvent.Entity, "model_name", failedEvent.Model})
		}
	}
}

func (e *EmbeddingConsumer) Process(event []Event) error {
	go func(event []Event) {
		var err error
		defer func() {
			if r := recover(); r != nil {
				metric.Count("embedding_consumer_panic_event", int64(len(event)), []string{})
				panicErr := fmt.Errorf("panic occurred: %v", r)
				log.Error().Msgf("%s", panicErr)
				if err == nil {
					err = panicErr
				} else {
					err = errors.Join(err, panicErr)
				}
			}
			if err != nil {
				e.produceFailureEvents(event)
			}
		}()

		indexerEvent := indexer.Event{
			Data: make(map[indexer.EventType][]indexer.Data),
		}

		embeddingErr := e.processEmbeddingEvent(event, indexerEvent)
		if embeddingErr != nil {
			log.Error().Err(embeddingErr).Msg("Error processing embedding events")
			err = embeddingErr
		}

		indexerErr := e.qdrantIndexerHandler.Process(indexerEvent)
		if indexerErr != nil {
			log.Error().Err(indexerErr).Msg("Error processing indexer event")
			err = indexerErr
		}

		log.Info().Msgf("Successfully processed embedding batch of size %d", len(event))
	}(event)
	return nil
}

func (e *EmbeddingConsumer) ProcessInSequence(events []Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			metric.Count("embedding_consumer_panic_event", int64(len(events)), []string{})
			panicErr := fmt.Errorf("panic occurred: %v", r)
			log.Error().Msgf("%s", panicErr)
			if err == nil {
				err = panicErr
			} else {
				err = errors.Join(err, panicErr)
			}
		}
		if err != nil {
			e.produceFailureEvents(events)
		}
	}()

	indexerEvent := indexer.Event{
		Data: make(map[indexer.EventType][]indexer.Data),
	}

	embeddingErr := e.processEmbeddingEvent(events, indexerEvent)
	if embeddingErr != nil {
		log.Error().Err(embeddingErr).Msg("Error processing embedding events")
		return embeddingErr
	}

	indexerErr := e.qdrantIndexerHandler.Process(indexerEvent)
	if indexerErr != nil {
		log.Error().Err(indexerErr).Msg("Error processing indexer event")
		return indexerErr
	}

	log.Info().Msgf("Successfully processed embedding batch of size %d", len(events))
	return nil
}

func (e *EmbeddingConsumer) processEmbeddingEvent(events []Event, indexerEvent indexer.Event) error {
	for _, event := range events {
		embeddingStorePayload, err := e.mapEventToEmbeddingStorePayload(event)
		if err != nil {
			log.Error().Msgf("Error mapping event to EmbeddingStorePayload: %v", err)
			return err
		}
		entityConfig, err := e.configManager.GetEntityConfig(event.Entity)
		if err != nil {
			log.Error().Msgf("Error getting entity config for entity %s: %v", event.Entity, err)
			return err
		}
		modelConfig, err := e.configManager.GetModelConfig(event.Entity, event.Model)
		if err != nil {
			log.Error().Msgf("Error getting model config for entity %s and model %s: %v", event.Entity, event.Model, err)
			return err
		}
		storeId := entityConfig.StoreId
		embeddingStoreTtl := modelConfig.EmbeddingStoreTtl
		isEmbeddingStoreEnabled := modelConfig.EmbeddingStoreEnabled
		if e.AppConfig.Configs.AppEnv == "int" {
			isEmbeddingStoreEnabled = false
		}
		if isEmbeddingStoreEnabled {
			if err = e.embeddingStore.Persist(storeId, embeddingStoreTtl, embeddingStorePayload); err != nil {
				metric.Incr("embedding_store_persist_error", []string{"entity", event.Entity, "model", event.Model, "store_id", storeId})
				log.Error().Msgf("Error persisting embedding store data: %v", err)
				return err
			}
		}
		if event.CandidateId == "EOF" {
			if err = e.triggerIndexing(event, event.IndexSpace.VariantsVersionMap, event.Partition); err != nil {
				log.Error().Msgf("Error triggering indexing for event: %v", err)
				// return err
			}
			continue
		}
		aggregatorData, err := e.aggregatorDb.Query(storeId, &aggregator.Query{CandidateId: event.CandidateId})
		if err != nil {
			log.Error().Msgf("Error querying aggregator for candidate %s: %v", event.CandidateId, err)
			return err
		}
		for variant, version := range event.IndexSpace.VariantsVersionMap {
			data := indexer.Data{
				Entity:  event.Entity,
				Model:   event.Model,
				Variant: variant,
				Version: version,
				Id:      event.CandidateId,
				Payload: nil,
				Vectors: event.IndexSpace.Embedding,
			}
			switch event.IndexSpace.Operation {
			case "DELETE":
				indexerEvent.Data[indexer.Delete] = append(indexerEvent.Data[indexer.Delete], data)
			case "ADD":
				if shouldBeIndexed := event.IndexSpace.VariantsIndexMap[variant]; shouldBeIndexed {
					if err = e.processAddOperation(event, data, indexerEvent, aggregatorData); err != nil {
						log.Error().Msgf("Error processing add operation: %v", err)
						return err
					}
				}
			case "UPSERT_PAYLOAD":
				if shouldBeIndexed := event.IndexSpace.VariantsIndexMap[variant]; shouldBeIndexed {
					if err = e.processUpsertPayloadOperation(event, data, indexerEvent, aggregatorData); err != nil {
						log.Error().Msgf("Error processing upsert payload operation: %v", err)
						return err
					}
				}
			default:
				log.Error().Msgf("Invalid operation: %s", event.IndexSpace.Operation)
				return fmt.Errorf("invalid operation: %s", event.IndexSpace.Operation)
			}
		}
	}
	return nil
}

func (e *EmbeddingConsumer) processAddOperation(event Event, data indexer.Data, indexerEvent indexer.Event, aggregatorData map[string]interface{}) error {
	var err error
	data.Payload, err = e.preparePayloadIndexMap(event, aggregatorData, data.Variant)
	if err != nil {
		log.Error().Msgf("Error preparing payload index map: %v", err)
		return err
	}
	if shouldIndex, err := e.shouldIndex(event, data.Variant, aggregatorData); err != nil {
		log.Error().Msgf("Error determining if should index: %v", err)
		return err
	} else if shouldIndex {
		indexerEvent.Data[indexer.Upsert] = append(indexerEvent.Data[indexer.Upsert], data)
	}
	return nil
}

func (e *EmbeddingConsumer) processUpsertPayloadOperation(event Event, data indexer.Data, indexerEvent indexer.Event, aggregatorData map[string]interface{}) error {
	var err error
	data.Payload, err = e.preparePayloadIndexMap(event, aggregatorData, data.Variant)
	if err != nil {
		log.Error().Msgf("Error preparing payload index map: %v", err)
		return err
	}
	if shouldIndex, err := e.shouldIndex(event, data.Variant, aggregatorData); err != nil {
		log.Error().Msgf("Error determining if should index: %v", err)
		return err
	} else if shouldIndex {
		indexerEvent.Data[indexer.UpsertPayload] = append(indexerEvent.Data[indexer.UpsertPayload], data)
	}
	return nil
}

func (e *EmbeddingConsumer) mapEventToEmbeddingStorePayload(event Event) (embedding.Payload, error) {
	var searchEmbedding []float32
	if event.SearchSpace.Embedding != nil {
		searchEmbedding = event.SearchSpace.Embedding
	} else {
		searchEmbedding = event.IndexSpace.Embedding
	}
	payload := embedding.Payload{
		CandidateId:      event.CandidateId,
		Embedding:        event.IndexSpace.Embedding,
		SearchEmbedding:  searchEmbedding,
		Model:            event.Model,
		Version:          event.EmbeddingStoreVersion,
		VariantsIndexMap: event.IndexSpace.VariantsIndexMap,
	}
	return payload, nil
}

func (e *EmbeddingConsumer) shouldIndex(event Event, variant string, aggregatorData map[string]interface{}) (bool, error) {
	variantConfig, err := e.configManager.GetVariantConfig(event.Entity, event.Model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", event.Entity, event.Model, variant, err)
		return false, err
	}
	aggregatorFilters := variantConfig.Filter
	if aggregatorFilters == nil {
		return true, nil
	}
	for _, criteria := range aggregatorFilters {
		for _, filter := range criteria {
			filterData := filter.DefaultValue
			if dataValue, exists := aggregatorData[filter.ColumnName]; exists {
				filterData = dataValue.(string)
			}
			if filterData != filter.FilterValue {
				return false, nil
			}
		}
	}
	return true, nil
}

func (e *EmbeddingConsumer) preparePayloadIndexMap(event Event, rtColumns map[string]interface{}, variant string) (map[string]interface{}, error) {
	variantConfig, err := e.configManager.GetVariantConfig(event.Entity, event.Model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", event.Entity, event.Model, variant, err)
		return nil, err
	}
	variantPayload := variantConfig.VectorDbConfig.Payload
	payloadIndexMap := make(map[string]interface{})
	for key, variantPayloadValue := range variantPayload {
		// Event
		eventPayloadValue := event.IndexSpace.Payload[key]

		if eventPayloadValue != "" {
			payloadValue, err := adaptToPayloadValue(eventPayloadValue, variantPayloadValue.FieldSchema)
			if err != nil {
				log.Error().Msgf(
					"Error adapting payload value for key=%s schema=%s value=%q: %v",
					key, variantPayloadValue.FieldSchema, eventPayloadValue, err,
				)
				return nil, err
			}
			payloadIndexMap[key] = payloadValue
		} else {
			// RT
			if rtColumns[key] != nil {
				payloadIndexMap[key] = rtColumns[key]
			} else {
				payloadValue, err := adaptToPayloadValue(variantPayloadValue.DefaultValue, variantPayloadValue.FieldSchema)
				if err != nil {
					log.Error().Msgf("Error adapting payload value for key=%s schema=%s value=%q: %v", key, variantPayloadValue.FieldSchema, variantPayloadValue.DefaultValue, err)
					return nil, err
				}
				payloadIndexMap[key] = payloadValue
			}
		}
	}
	return payloadIndexMap, nil
}

func adaptToPayloadValue(rawValue string, fieldSchema string) (interface{}, error) {
	rawValue = strings.TrimSpace(rawValue)
	fieldSchema = strings.ToLower(strings.TrimSpace(fieldSchema))

	isArray := strings.HasPrefix(rawValue, "[") && strings.HasSuffix(rawValue, "]")

	if rawValue == "" {
		switch fieldSchema {
		case "keyword":
			if isArray {
				return []string{""}, nil
			}
			return "", nil
		case "integer":
			if isArray {
				return []int{0}, nil
			}
			return 0, nil
		case "boolean":
			if isArray {
				return []bool{false}, nil
			}
			return false, nil
		default:
			return nil, fmt.Errorf("unsupported field_schema for empty value: %s", fieldSchema)
		}
	}

	switch fieldSchema {
	case "integer":
		if isArray {
			var v []int
			if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
				return nil, err
			}
			return v, nil
		}
		return strconv.Atoi(rawValue)

	case "keyword":
		if isArray {
			var v []string
			if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
				return nil, err
			}
			return v, nil
		}
		return rawValue, nil

	case "boolean":
		if isArray {
			var v []bool
			if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
				return nil, err
			}
			return v, nil
		}
		return strconv.ParseBool(rawValue)

	default:
		return nil, fmt.Errorf("unsupported field_schema: %s", fieldSchema)
	}
}

func (e *EmbeddingConsumer) triggerIndexing(event Event, variantsVersionMap map[string]int, partition string) error {
	jsonData := map[string]interface{}{
		"entity":                  event.Entity,
		"model":                   event.Model,
		"variants_version_map":    variantsVersionMap,
		"embedding_store_version": event.EmbeddingStoreVersion,
		"partition":               partition,
	}
	log.Error().Msgf("Triggering indexing for event: %v", jsonData)
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		// return err
	}
	req, err := http.NewRequest(http.MethodPost, e.httpClient.Endpoint+"/api/v1/qdrant/trigger-indexing", bytes.NewReader(jsonBytes))
	if err != nil {
		log.Error().Msg("Error while creating http request")
		return err
	}
	resp, err := e.httpClient.CoreClient.Do(req)
	if err != nil {
		log.Error().Msg("Error while sending http request")
		// return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msg("Error while closing response body")
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Unexpected status code: %d", resp.StatusCode)
		// return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
