package realtime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/handler/aggregator"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/handler/indexer"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	skafka "github.com/Meesho/BharatMLStack/skye/pkg/kafka"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

var (
	rtConsumer Consumer
	rtOnce     sync.Once
)

type RealtimeConsumer struct {
	configManager        config.Manager
	aggregatorHandler    aggregator.Handler
	embeddingStore       embedding.Store
	qdrantIndexerHandler indexer.Handler
}

func newRealtimeConsumer() Consumer {
	if rtConsumer == nil {
		rtOnce.Do(func() {
			configManager := config.NewManager(config.DefaultVersion)
			aggregatorHandler := aggregator.NewHandler(aggregator.SCYLLA)
			qdrantIndexerHandler := indexer.NewHandler(indexer.QDRANT)
			embeddingStore := embedding.NewRepository(embedding.DefaultVersion)
			rtConsumer = &RealtimeConsumer{
				configManager:        configManager,
				aggregatorHandler:    aggregatorHandler,
				qdrantIndexerHandler: qdrantIndexerHandler,
				embeddingStore:       embeddingStore,
			}
		})
	}
	return rtConsumer
}

func (r *RealtimeConsumer) Process(events []Event) error {
	combinedIndexerEvent := indexer.Event{
		Data: make(map[indexer.EventType][]indexer.Data),
	}
	for _, event := range events {
		defer func() {
			if err := recover(); err != nil {
				metric.Count("realtime_consumer_panic_event", int64(len(event.Data)), []string{"type", event.Type})
				log.Error().Msgf("panic occurred: %v", err)
				r.ProduceMessage(event)
			}
		}()
		if err := r.ProcessRealtimeEvent(event, combinedIndexerEvent); err != nil {
			metric.Count("realtime_consumer_event_error", int64(len(event.Data)), []string{"type", event.Type})
			log.Error().Msgf("Error processing realtime event: %v", err)
			r.ProduceMessage(event)
			return err
		}
	}
	// err := r.qdrantIndexerHandler.Process(combinedIndexerEvent)
	// if err != nil {
	// 	log.Error().Msgf("Error processing combined indexer event: %v", err)
	// 	return err
	// }
	log.Info().Msgf("Realtime Combined Indexer Event: %v", combinedIndexerEvent)
	for eventType, event := range combinedIndexerEvent.Data {
		err := r.ProduceDeltaEvent(eventType, event)
		if err != nil {
			log.Error().Msgf("Error producing delta event: %v", err)
			return err
		}
	}
	return nil
}

func (r *RealtimeConsumer) ProduceDeltaEvent(eventType indexer.EventType, event []indexer.Data) error {
	keyStr := ""
	payloadToProduce := make([]skafka.ProducerMessage, 0, len(event))
	for _, eventData := range event {
		metric.Count("realtime_delta_producer_event", 1, []string{"type", string(eventType), "entity", eventData.Entity, "model", eventData.Model, "variant", eventData.Variant, "version", strconv.Itoa(eventData.Version)})
		if eventType == indexer.Upsert && len(eventData.Vectors) == 0 {
			metric.Count("embedding_empty_upsert_event", 1, []string{"type", string(eventType), "entity", eventData.Entity, "model", eventData.Model, "variant", eventData.Variant, "embedding_store_version", strconv.Itoa(eventData.Version)})
			continue
		}
		deltaEvent := DeltaEvent{
			Entity:    eventData.Entity,
			Model:     eventData.Model,
			Variant:   eventData.Variant,
			Version:   eventData.Version,
			Id:        eventData.Id,
			Payload:   eventData.Payload,
			Vectors:   eventData.Vectors,
			EventType: string(eventType),
		}
		jsonDeltaEvent, err := json.Marshal(deltaEvent)
		if err != nil {
			log.Error().Msgf("Error in Marshalling %s", err)
			return err
		}
		variantConfig, err := r.configManager.GetVariantConfig(deltaEvent.Entity, deltaEvent.Model, deltaEvent.Variant)
		if err != nil {
			log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", deltaEvent.Entity, deltaEvent.Model, deltaEvent.Variant, err)
			return err
		}
		if variantConfig.RTPartition == 0 {
			metric.Count("realtime_delta_producer_error", 1, []string{"type", string(eventType), "entity", eventData.Entity, "model", eventData.Model, "variant", eventData.Variant, "version", strconv.Itoa(eventData.Version)})
			log.Error().Msgf("RTPartition is 0 for entity %s, model %s, variant %s", deltaEvent.Entity, deltaEvent.Model, deltaEvent.Variant)
			return err
		}
		payloadToProduce = append(payloadToProduce, skafka.ProducerMessage{
			Partition: &variantConfig.RTPartition,
			Key:       &keyStr,
			Value:     jsonDeltaEvent,
			Headers:   make(map[string][]byte),
		})
	}
	err := skafka.SendAndForget(appConfig.RealTimeDeltaProducerKafkaId, payloadToProduce)
	if err != nil {
		log.Error().Msgf("Error in producing message %s", err)
		return err
	}
	return nil
}

func (r *RealtimeConsumer) ProduceMessage(event Event) {
	metric.Count("realtime_producer_event", int64(len(event.Data)), []string{"type", event.Type})
	jsonPayload, err := json.Marshal(event)
	if err != nil {
		log.Error().Msgf("Error in Marshalling %s", err)
	}
	keyStr := ""
	payloadToProduce := []skafka.ProducerMessage{
		{
			Key:     &keyStr,
			Value:   jsonPayload,
			Headers: make(map[string][]byte),
		},
	}
	err = skafka.SendAndForget(appConfig.RealtimeProducerKafkaId, payloadToProduce)
	if err != nil {
		log.Error().Msgf("Error in producing message %s", err)
		return
	}
}

func (r *RealtimeConsumer) ProcessRealtimeEvent(event Event, indexerEvent indexer.Event) error {
	entityConfig, err := r.configManager.GetEntityConfig(event.Entity)
	if err != nil {
		log.Error().Msgf("Error getting entity config for entity %s: %v", event.Entity, err)
		return err
	}
	storeId := entityConfig.StoreId
	criteriaMap, err := r.configManager.GetAllFiltersForActiveVariants(event.Entity)
	if err != nil {
		log.Error().Msgf("Error getting criteria for entity %s: %v", event.Entity, err)
		return err
	}
	err = r.process(event, indexerEvent, storeId, criteriaMap)
	if err != nil {
		log.Error().Msgf("Error processing realtime event: %v", err)
		return err
	}
	return nil
}

func (r *RealtimeConsumer) process(event Event, indexerEvent indexer.Event, storeId string, criteriaMap map[string]map[string][]config.Criteria) error {
	for _, data := range event.Data {
		aggregatorData := make(map[string]string)
		for _, value := range data.Value {
			aggregatorData[value.Label] = value.Value
		}
		aggregatorResponse, err := r.aggregatorHandler.Process(aggregator.Payload{CandidateId: data.Id, Entity: event.Entity, Columns: aggregatorData})
		if err != nil {
			log.Error().Msgf("Error querying aggregator for candidate %s: %v", data.Id, err)
			return err
		}
		log.Info().Msgf("Aggregator Value Types : %v", aggregatorResponse)
		err = r.processDelta(event, indexerEvent, storeId, criteriaMap, aggregatorResponse, data)
		if err != nil {
			log.Error().Msgf("Error processing delta values for candidate %s: %v", data.Id, err)
			return err
		}
	}
	return nil
}

func (r *RealtimeConsumer) processDelta(event Event, indexerEvent indexer.Event, storeId string, criteriaMap map[string]map[string][]config.Criteria, aggregatorResponse *aggregator.Response, data Data) error {
	if len(aggregatorResponse.DeltaData) > 0 {
		metric.Count("delta_realtime_consumer_event", int64(len(event.Data)), []string{"type", event.Type})
		err := r.processCriteria(event, indexerEvent, storeId, criteriaMap, aggregatorResponse, data)
		if err != nil {
			log.Error().Msgf("Error processing criteria for candidate %s: %v", data.Id, err)
			return err
		}
	}
	return nil
}

// 1. shouldIndexNonDelta = false -> continue (meaning acc to non delta columns data should not go to vector db)
// 2. shouldIndexNonDelta = true and shouldIndexDelta = false -> delete, meaning data is to be deleted acc to criteria
// 3. shouldIndexNonDelta = true and shouldIndexDelta = true -> Upsert, meaning data is to be added acc to criteria
// 4. criteria not present in delta columns -> skip
// 5. Payload Field is present in delta columns -> upsert payload
func (r *RealtimeConsumer) processCriteria(event Event, indexerEvent indexer.Event, storeId string, criteriaMap map[string]map[string][]config.Criteria, aggregatorResponse *aggregator.Response, data Data) error {
	for model, models := range criteriaMap {
		for variant, criteria := range models {
			if criteria == nil {
				continue
			}
			// log.Info().Msgf("Processing criteria for model %s, variant %s", model, variant)
			indexerData := indexer.Data{
				Entity:  event.Entity,
				Model:   model,
				Variant: variant,
				Id:      data.Id,
				Payload: nil,
			}
			variantConfig, err := r.configManager.GetVariantConfig(event.Entity, model, variant)
			if err != nil {
				log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", event.Entity, model, variant, err)
				return err
			}

			shouldIndex := r.shouldIndexNonDelta(aggregatorResponse, criteria)
			// log.Info().Msgf("Should index non delta for model %s, variant %s: %v", model, variant, shouldIndex)
			if !shouldIndex {
				continue
			}
			variantWriteVersion := variantConfig.VectorDbWriteVersion
			variantReadVersion := variantConfig.VectorDbReadVersion
			if !r.isCriteriaPresent(aggregatorResponse.DeltaData, criteria) {
				payload := variantConfig.VectorDbConfig.Payload
				if r.isPayloadPresent(aggregatorResponse.DeltaData, payload) {
					writeEmbeddingStoreResponse, readEmbeddingStoreResponse, err := r.getEmbeddingStoreResponses(event.Entity, storeId, model, variant, data.Id, variantReadVersion, variantWriteVersion)
					if err != nil {
						log.Error().Msgf("Error getting embedding store responses for entity %s, model %s, variant %s: %v", event.Entity, model, variant, err)
						return err
					}
					metric.Count("delta_payload_realtime_consumer_event", int64(len(event.Data)), []string{"type", event.Type})
					r.addUpsertPayloadEvent(event.Entity, model, data.Id, variant, indexerEvent, indexerData, writeEmbeddingStoreResponse, readEmbeddingStoreResponse, variantReadVersion, variantWriteVersion, aggregatorResponse.DeltaData)
				}
				continue
			}
			shouldIndex = r.shouldIndexDelta(aggregatorResponse.DeltaData, criteria)
			if shouldIndex {
				writeEmbeddingStoreResponse, readEmbeddingStoreResponse, err := r.getEmbeddingStoreResponses(event.Entity, storeId, model, variant, data.Id, variantReadVersion, variantWriteVersion)
				if err != nil {
					log.Error().Msgf("Error getting embedding store responses for entity %s, model %s, variant %s: %v", event.Entity, model, variant, err)
					return err
				}
				// log.Info().Msgf("Adding upsert event for model %s, variant %s", model, variant)
				r.addUpsertEvent(event.Entity, model, data.Id, variant, indexerEvent, indexerData, writeEmbeddingStoreResponse, readEmbeddingStoreResponse, variantReadVersion, variantWriteVersion, aggregatorResponse.CompleteData)
			} else {
				writeEmbeddingStoreResponse, readEmbeddingStoreResponse, err := r.getEmbeddingStoreResponses(event.Entity, storeId, model, variant, data.Id, variantReadVersion, variantWriteVersion)
				if err != nil {
					log.Error().Msgf("Error getting embedding store responses for entity %s, model %s, variant %s: %v", event.Entity, model, variant, err)
					return err
				}
				// log.Info().Msgf("Adding delete event for model %s, variant %s", model, variant)
				r.addDeleteEvent(indexerEvent, indexerData, writeEmbeddingStoreResponse, readEmbeddingStoreResponse, data.Id, variant, variantReadVersion, variantWriteVersion)
			}

		}
	}
	return nil
}

func (r *RealtimeConsumer) getEmbeddingStoreResponses(entity, storeId, model, variant, candidateId string, variantReadVersion, variantWriteVersion int) (map[string]map[string]interface{}, map[string]map[string]interface{}, error) {
	variantConfig, err := r.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", entity, model, variant, err)
		return nil, nil, err
	}
	embeddingStoreWriteVersion := variantConfig.EmbeddingStoreWriteVersion
	writeEmbeddingStoreResponse, err := r.embeddingStore.BulkQueryConsumer(storeId, &embedding.BulkQuery{
		CandidateIds: []string{candidateId},
		Model:        model,
		Version:      embeddingStoreWriteVersion,
	})
	if err != nil {
		log.Error().Msgf("Error querying embedding store for candidate %s: %v", candidateId, err)
		return nil, nil, err
	}
	readEmbeddingStoreResponse := writeEmbeddingStoreResponse
	if variantReadVersion != variantWriteVersion {
		embeddingStoreReadVersion := variantConfig.EmbeddingStoreReadVersion
		readEmbeddingStoreResponse, _ = r.embeddingStore.BulkQueryConsumer(storeId, &embedding.BulkQuery{
			CandidateIds: []string{candidateId},
			Model:        model,
			Version:      embeddingStoreReadVersion,
		})
	}
	checkEmbeddingResponse(writeEmbeddingStoreResponse, candidateId, "write", entity, model, variant, variantConfig.EmbeddingStoreWriteVersion)
	if variantConfig.EmbeddingStoreReadVersion != variantConfig.EmbeddingStoreWriteVersion {
		checkEmbeddingResponse(readEmbeddingStoreResponse, candidateId, "read", entity, model, variant, variantConfig.EmbeddingStoreReadVersion)
	}

	return writeEmbeddingStoreResponse, readEmbeddingStoreResponse, nil
}

func checkEmbeddingResponse(resp map[string]map[string]any, candidateId, prefix, entity, model, variant string, version int) {
	if len(resp) == 0 {
		metric.Incr(fmt.Sprintf("%s_embedding_store_response_null", prefix), []string{"entity", entity, "model", model, "variant", variant, "embedding_store_version", strconv.Itoa(version)})
		return
	}

	if candidateData, ok := resp[candidateId]; ok {
		if candidateData[variant+"_to_be_indexed"].(bool) {
			if _, ok := candidateData["embedding"].([]float32); !ok {
				metric.Incr(fmt.Sprintf("%s_embedding_store_emb_null", prefix), []string{"entity", entity, "model", model, "variant", variant, "embedding_store_version", strconv.Itoa(version)})
			} else if len(candidateData["embedding"].([]float32)) == 0 {
				metric.Incr(fmt.Sprintf("%s_embedding_store_emb_empty", prefix), []string{"entity", entity, "model", model, "variant", variant, "embedding_store_version", strconv.Itoa(version)})
			}
		}
		return
	}
	metric.Incr(fmt.Sprintf("%s_embedding_store_candidate_null", prefix), []string{"entity", entity, "model", model, "variant", variant, "embedding_store_version", strconv.Itoa(version)})
}

func (r *RealtimeConsumer) addUpsertPayloadEvent(entity, model, candidateId, variant string,
	indexerEvent indexer.Event, indexerData indexer.Data,
	writeEmbeddingStoreResponse, readEmbeddingStoreResponse map[string]map[string]interface{},
	variantReadVersion, variantWriteVersion int, deltaAggregatorData map[string]interface{}) {

	indexerData.Payload = r.preparePayloadIndexMap(entity, model, deltaAggregatorData, variant)
	if len(indexerData.Payload) > 0 {
		if len(writeEmbeddingStoreResponse) != 0 && writeEmbeddingStoreResponse[candidateId][variant+"_to_be_indexed"].(bool) {
			indexerData.Version = variantWriteVersion
			indexerEvent.Data[indexer.UpsertPayload] = append(indexerEvent.Data[indexer.UpsertPayload], indexerData)
		}
		if variantReadVersion != variantWriteVersion {
			if len(readEmbeddingStoreResponse) != 0 && readEmbeddingStoreResponse[candidateId][variant+"_to_be_indexed"].(bool) {
				indexerData.Version = variantReadVersion
				indexerEvent.Data[indexer.UpsertPayload] = append(indexerEvent.Data[indexer.UpsertPayload], indexerData)
			}
		}
	}
}

func (r *RealtimeConsumer) addUpsertEvent(entity, model, candidateId, variant string,
	indexerEvent indexer.Event, indexerData indexer.Data,
	writeEmbeddingStoreResponse, readEmbeddingStoreResponse map[string]map[string]interface{},
	variantReadVersion, variantWriteVersion int, allAggregatorData map[string]interface{}) {

	indexerData.Payload = r.preparePayloadIndexMap(entity, model, allAggregatorData, variant)
	// log.Info().Msgf("upsert event for model %s, variant %s embedding store response: %v", model, variant, writeEmbeddingStoreResponse)
	if len(writeEmbeddingStoreResponse) != 0 && writeEmbeddingStoreResponse[candidateId][variant+"_to_be_indexed"].(bool) {
		indexerData.Version = variantWriteVersion
		indexerData.Vectors = writeEmbeddingStoreResponse[candidateId]["embedding"].([]float32)
		indexerEvent.Data[indexer.Upsert] = append(indexerEvent.Data[indexer.Upsert], indexerData)
	}
	if variantReadVersion != variantWriteVersion {
		if len(readEmbeddingStoreResponse) != 0 && readEmbeddingStoreResponse[candidateId][variant+"_to_be_indexed"].(bool) {
			indexerData.Version = variantReadVersion
			indexerData.Vectors = readEmbeddingStoreResponse[candidateId]["embedding"].([]float32)
			indexerEvent.Data[indexer.Upsert] = append(indexerEvent.Data[indexer.Upsert], indexerData)
		}
	}
}

func (r *RealtimeConsumer) addDeleteEvent(indexerEvent indexer.Event, indexerData indexer.Data,
	writeEmbeddingStoreResponse, readEmbeddingStoreResponse map[string]map[string]interface{}, candidateId, variant string, variantReadVersion, variantWriteVersion int) {
	// log.Info().Msgf("delete event variant %s embedding store response: %v", variant, writeEmbeddingStoreResponse)
	if len(writeEmbeddingStoreResponse) != 0 && writeEmbeddingStoreResponse[candidateId][variant+"_to_be_indexed"].(bool) {
		indexerData.Version = variantWriteVersion
		indexerEvent.Data[indexer.Delete] = append(indexerEvent.Data[indexer.Delete], indexerData)
	}
	if variantReadVersion != variantWriteVersion && len(readEmbeddingStoreResponse) != 0 && readEmbeddingStoreResponse[candidateId][variant+"_to_be_indexed"].(bool) {
		indexerData.Version = variantReadVersion
		indexerEvent.Data[indexer.Delete] = append(indexerEvent.Data[indexer.Delete], indexerData)
	}
}

func (r *RealtimeConsumer) shouldIndexDelta(deltaData map[string]interface{}, criteria []config.Criteria) bool {
	shouldIndex := true
	for _, val := range criteria {
		if deltaData[val.ColumnName] != nil && deltaData[val.ColumnName].(string) != val.FilterValue {
			shouldIndex = false
			break
		}
	}
	return shouldIndex
}

func (r *RealtimeConsumer) isCriteriaPresent(deltaData map[string]interface{}, criteria []config.Criteria) bool {
	isCriteriaPresent := false
	for _, val := range criteria {
		if deltaData[val.ColumnName] != nil {
			isCriteriaPresent = true
			break
		}
	}
	return isCriteriaPresent
}

func (r *RealtimeConsumer) isPayloadPresent(deltaData map[string]interface{}, payload map[string]config.Payload) bool {
	if len(payload) == 0 {
		return false
	}
	for key := range payload {
		if _, exists := deltaData[key]; exists {
			return true
		}
	}
	return false
}

func (r *RealtimeConsumer) shouldIndexNonDelta(aggregatorResponse *aggregator.Response, criteria []config.Criteria) bool {
	deltaData := aggregatorResponse.DeltaData
	allData := aggregatorResponse.CompleteData
	shouldIndex := true
	for _, val := range criteria {
		if deltaData[val.ColumnName] == nil {
			if allData[val.ColumnName] == nil || allData[val.ColumnName].(string) != val.FilterValue {
				shouldIndex = false
				break
			}
		}
	}
	return shouldIndex
}

func (r *RealtimeConsumer) preparePayloadIndexMap(entity string, model string, rtColumns map[string]interface{}, variant string) map[string]interface{} {
	variantConfig, err := r.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", entity, model, variant, err)
		return nil
	}
	variantPayload := variantConfig.VectorDbConfig.Payload
	payloadIndexMap := make(map[string]interface{})
	for key, value := range rtColumns {
		if _, exists := variantPayload[key]; exists {
			payloadIndexMap[key] = value
		}
	}
	return payloadIndexMap
}
