package indexer

import (
	"strconv"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/rs/zerolog/log"
)

var (
	qdrantHandler Handler
)

type QdrantIndexer struct {
	configManager config.Manager
}

func initQdrantIndexerHandler() Handler {
	if qdrantHandler == nil {
		once.Do(func() {
			qdrantHandler = &QdrantIndexer{
				configManager: config.NewManager(config.DefaultVersion),
			}
		})
	}
	return qdrantHandler
}

func (q *QdrantIndexer) Process(event Event) error {
	var err error
	for eventType, data := range event.Data {
		switch eventType {
		case Upsert:
			err = q.bulkUpsert(data)
		case Delete:
			err = q.bulkDelete(data)
		case UpsertPayload:
			err = q.bulkUpsertPayload(data)
		default:
			log.Error().Msgf("Invalid event type: %s", eventType)
		}
	}
	return err
}

func (q *QdrantIndexer) getKey(entity, model, variant string, version int) string {
	return entity + "|" + model + "|" + variant + "|" + strconv.Itoa(version)
}

func (q *QdrantIndexer) bulkUpsert(data []Data) error {
	upsertRequest := vector.UpsertRequest{
		Data: make(map[string][]vector.Data),
	}
	vectorDbRequest := make(map[enums.VectorDbType]vector.UpsertRequest)
	for _, d := range data {
		variantConfig, err := q.configManager.GetVariantConfig(d.Entity, d.Model, d.Variant)
		if err != nil {
			log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", d.Entity, d.Model, d.Variant, err)
			return err
		}
		if !variantConfig.Enabled {
			continue
		}
		vectorDbType := variantConfig.VectorDbType
		key := q.getKey(d.Entity, d.Model, d.Variant, d.Version)
		if upsertRequest.Data[key] == nil {
			upsertRequest.Data[key] = []vector.Data{}
		}
		upsertRequest.Data[key] = append(upsertRequest.Data[key], vector.Data{
			Id:      d.Id,
			Payload: d.Payload,
			Vectors: d.Vectors,
		})
		vectorDbRequest[vectorDbType] = upsertRequest
	}
	for vectorDbType, request := range vectorDbRequest {
		err := vector.GetRepository(vectorDbType).BulkUpsert(request)
		if err != nil {
			log.Error().Msgf("Error in bulk upsert: %s", err)
			return err
		}
	}

	return nil
}

func (q *QdrantIndexer) bulkDelete(data []Data) error {
	deleteRequest := vector.DeleteRequest{
		Data: make(map[string][]vector.Data),
	}
	vectorDbRequest := make(map[enums.VectorDbType]vector.DeleteRequest)
	for _, d := range data {
		variantConfig, err := q.configManager.GetVariantConfig(d.Entity, d.Model, d.Variant)
		if err != nil {
			log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", d.Entity, d.Model, d.Variant, err)
			return err
		}
		if !variantConfig.Enabled {
			continue
		}
		vectorDbType := variantConfig.VectorDbType
		key := q.getKey(d.Entity, d.Model, d.Variant, d.Version)
		if deleteRequest.Data[key] == nil {
			deleteRequest.Data[key] = []vector.Data{}
		}
		deleteRequest.Data[key] = append(deleteRequest.Data[key], vector.Data{
			Id:      d.Id,
			Payload: d.Payload,
			Vectors: d.Vectors,
		})
		vectorDbRequest[vectorDbType] = deleteRequest
	}
	for vectorDbType, request := range vectorDbRequest {
		err := vector.GetRepository(vectorDbType).BulkDelete(request)
		if err != nil {
			log.Error().Msgf("Error in bulk delete: %s", err)
			return err
		}
	}
	return nil
}

func (q *QdrantIndexer) bulkUpsertPayload(data []Data) error {
	upsertPayloadRequest := vector.UpsertPayloadRequest{
		Data: make(map[string][]vector.Data),
	}
	vectorDbRequest := make(map[enums.VectorDbType]vector.UpsertPayloadRequest)
	for _, d := range data {
		variantConfig, err := q.configManager.GetVariantConfig(d.Entity, d.Model, d.Variant)
		if err != nil {
			log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", d.Entity, d.Model, d.Variant, err)
			return err
		}
		if !variantConfig.Enabled {
			continue
		}
		vectorDbType := variantConfig.VectorDbType
		key := q.getKey(d.Entity, d.Model, d.Variant, d.Version)
		if upsertPayloadRequest.Data[key] == nil {
			upsertPayloadRequest.Data[key] = []vector.Data{}
		}
		upsertPayloadRequest.Data[key] = append(upsertPayloadRequest.Data[key], vector.Data{
			Id:      d.Id,
			Payload: d.Payload,
			Vectors: d.Vectors,
		})
		vectorDbRequest[vectorDbType] = upsertPayloadRequest
	}
	for vectorDbType, request := range vectorDbRequest {
		err := vector.GetRepository(vectorDbType).BulkUpsertPayload(request)
		if err != nil {
			log.Error().Msgf("Error in bulk upsert payload: %s", err)
			return err
		}
	}
	return nil
}
