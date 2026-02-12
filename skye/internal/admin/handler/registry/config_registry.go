package registry

import (
	"errors"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/rs/zerolog/log"
)

var (
	manager Manager
	once    sync.Once
)

type RegistryManager struct {
	config config.Manager
}

func initRegistryHandler() Manager {
	if manager == nil {
		once.Do(func() {
			manager = &RegistryManager{
				config: config.NewManager(config.DefaultVersion),
			}
		})
	}
	return manager
}

func (r *RegistryManager) RegisterStore(request *CreateStoreRequest) error {
	err := r.config.RegisterStore(request.ConfId, request.Db, request.EmbeddingsTable, request.AggregatorTable)
	if err != nil {
		log.Error().Msgf("Error Registering Store for %s , %s, %s", request.Db, request.EmbeddingsTable, request.AggregatorTable)
		return err
	}
	return nil
}

func (r *RegistryManager) RegisterFrequency(request *CreateFrequencyRequest) error {
	err := r.config.RegisterFrequency(request.Frequency)
	if err != nil {
		log.Error().Msgf("Error Registering Store for %s ", request.Frequency)
		return err
	}
	return nil
}

func (r *RegistryManager) RegisterEntity(request *RegisterEntityRequest) error {
	err := r.config.RegisterEntity(request.Entity, request.StoreId)
	if err != nil {
		log.Error().Msgf("Error Registering Entity for %s , %s", request.Entity, request.StoreId)
		return err
	}
	return nil
}

func (r *RegistryManager) RegisterModel(request *RegisterModelRequest) error {
	err := r.config.RegisterModel(request.Entity, request.Model, request.EmbeddingStoreEnabled, request.EmbeddingStoreTtl, request.ModelConfig, request.ModelType, request.KafkaId, request.TrainingDataPath, request.Metadata, request.JobFrequency, request.NumberOfPartitions, request.FailureProducerKafkaId, request.TopicName)
	if err != nil {
		log.Error().Msgf("Error Registering Model for %s , %s, %v", request.Entity, request.Model, err)
		return err
	}
	return nil
}

func (r *RegistryManager) RegisterVariant(request *RegisterVariantRequest) error {
	if request.RtPartition == 0 {
		log.Error().Msgf("RTPartition is 0 for entity %s, model %s, variant %s", request.Entity, request.Model, request.Variant)
		return errors.New("RTPartition is 0")
	}
	if request.RateLimiters.RateLimit == 0 {
		log.Error().Msgf("RateLimit is 0 for entity %s, model %s, variant %s", request.Entity, request.Model, request.Variant)
		return errors.New("RateLimit is 0")
	}
	if request.RateLimiters.BurstLimit == 0 {
		log.Error().Msgf("BurstLimit is 0 for entity %s, model %s, variant %s", request.Entity, request.Model, request.Variant)
		return errors.New("BurstLimit is 0")
	}
	err := r.config.RegisterVariant(request.Entity, request.Model, request.Variant, request.VectorDbConfig, request.VectorDbType, request.Filter, request.Type, request.DistributedCachingEnabled, request.DistributedCacheTTLSeconds, request.InMemoryCachingEnabled, request.InMemoryCacheTTLSeconds, request.RtPartition, request.RateLimiters)
	if err != nil {
		log.Error().Msgf("Error Registering Variant for %s , %s, %s", request.Entity, request.Model, request.Variant)
		return err
	}
	return nil
}
