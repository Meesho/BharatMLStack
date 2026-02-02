package handler

import (
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye"
	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/scylla"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/entity_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/filter_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/job_frequency_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/model_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/qdrant_cluster_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/store_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_tasks"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_promotion_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_scaleup_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_scaleup_tasks"
	skyeEtcd "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd/enums"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type skyeConfig struct {
	StoreRequestRepo            store_requests.StoreRequestRepository
	EntityRequestRepo           entity_requests.EntityRequestRepository
	ModelRequestRepo            model_requests.ModelRequestRepository
	VariantRequestRepo          variant_requests.VariantRequestRepository
	FilterRequestRepo           filter_requests.FilterRequestRepository
	JobFrequencyRequestRepo     job_frequency_requests.JobFrequencyRequestRepository
	QdrantClusterRequestRepo    qdrant_cluster_requests.QdrantClusterRequestRepository
	VariantPromotionRequestRepo variant_promotion_requests.VariantPromotionRequestRepository
	VariantOnboardingTaskRepo   variant_onboarding_tasks.VariantOnboardingTaskRepository
	VariantScaleUpRequestRepo   variant_scaleup_requests.VariantScaleUpRequestRepository
	VariantScaleUpTaskRepo      variant_scaleup_tasks.VariantScaleUpTaskRepository
	EtcdConfig                  skyeEtcd.Manager
	ScyllaStores                map[int]scylla.SkyeStore
	AppConfig                   configs.Configs
	MQIdTopicsMapping           map[int]string
	VariantsList                []string
	SkyeClient                  skye.SkyeClient
}

func InitV1ConfigHandler(appConfig configs.Configs) Config {
	configOnce.Do(func() {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn := conn.(*infra.SQLConnection)
		config = &skyeConfig{
			StoreRequestRepo:            initRepository("store request", store_requests.Repository, sqlConn),
			EntityRequestRepo:           initRepository("entity request", entity_requests.Repository, sqlConn),
			ModelRequestRepo:            initRepository("model request", model_requests.Repository, sqlConn),
			VariantRequestRepo:          initRepository("variant request", variant_requests.Repository, sqlConn),
			FilterRequestRepo:           initRepository("filter request", filter_requests.Repository, sqlConn),
			JobFrequencyRequestRepo:     initRepository("job frequency request", job_frequency_requests.Repository, sqlConn),
			QdrantClusterRequestRepo:    initRepository("qdrant cluster request", qdrant_cluster_requests.Repository, sqlConn),
			VariantPromotionRequestRepo: initRepository("variant promotion request", variant_promotion_requests.Repository, sqlConn),
			VariantOnboardingTaskRepo:   initRepository("variant onboarding task", variant_onboarding_tasks.Repository, sqlConn),
			VariantScaleUpRequestRepo:   initRepository("variant scale up request", variant_scaleup_requests.Repository, sqlConn),
			VariantScaleUpTaskRepo:      initRepository("variant scale up task", variant_scaleup_tasks.Repository, sqlConn),
			EtcdConfig:                  skyeEtcd.NewEtcdConfig(appConfig),
			ScyllaStores:                initScyllaStores(appConfig),
			AppConfig:                   appConfig,
			MQIdTopicsMapping:           parseMQIdTopicsMapping(appConfig.MQIdTopicsMapping),
			VariantsList:                parseVariantsList(appConfig.VariantsList),
			SkyeClient: skye.GetSkyeClientFromConfig(1, skye.ClientConfig{
				Host:             appConfig.SkyeHost,
				Port:             appConfig.SkyePort,
				AuthToken:        appConfig.SkyeAuthToken,
				PlainText:        true,
				DeadlineExceedMS: appConfig.SkyeDeadlineExceedMS,
			}, appConfig.AppName),
		}
	})
	return config
}

func initScyllaStores(appConfig configs.Configs) map[int]scylla.SkyeStore {
	scyllaStores := make(map[int]scylla.SkyeStore)
	activeScyllaConfIds := appConfig.SkyeScyllaActiveConfigIds
	if activeScyllaConfIds == "" {
		log.Fatal().Msg("skye_scylla_active_config_ids not configured, running without Scylla stores")
		return scyllaStores
	}

	// Parse the horizon to skye conf id mapping
	confIdMapping := parseHorizonToSkyeScyllaConfIdMap(appConfig.HorizonToSkyeScyllaConfIdMap)

	for _, confIdStr := range strings.Split(activeScyllaConfIds, ",") {
		confIdStr = strings.TrimSpace(confIdStr)
		confId, err := strconv.Atoi(confIdStr)
		if err != nil {
			log.Error().Err(err).Msgf("Error converting Scylla config ID %s to int", confIdStr)
			continue
		}
		if infra.Scylla == nil {
			continue
		}
		connFacade, err := infra.Scylla.GetConnection(confId)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting Scylla connection for config ID %d", confId)
			continue
		}
		scyllaStore, err := scylla.NewSkyeRepository(connFacade.(*infra.ScyllaClusterConnection))
		if err != nil {
			log.Error().Err(err).Msgf("Error creating Scylla store for config ID %d", confId)
			continue
		}
		// Use mapped key if mapping exists, otherwise use original confId
		storeKey := confId
		if mappedKey, exists := confIdMapping[confId]; exists {
			storeKey = mappedKey
			log.Info().
				Int("original_conf_id", confId).
				Int("mapped_store_key", storeKey).
				Msg("Using mapped Scylla store key from horizon_to_skye_scylla_conf_id_map")
		}
		scyllaStores[storeKey] = scyllaStore
	}
	return scyllaStores
}

func (s *skyeConfig) RegisterStore(request StoreRegisterRequest) (RequestStatus, error) {
	return createRequest(request.Payload, request.Reason, request.Requestor, request.RequestType, "Store registration",
		func(payloadJSON string) (int, time.Time, error) {
			req := &store_requests.StoreRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: request.RequestType,
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.StoreRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		func(payload interface{}) error {
			storePayload, ok := payload.(StoreRequestPayload)
			if !ok {
				return fmt.Errorf("invalid payload type for store registration")
			}
			activeConfigIds := strings.Split(s.AppConfig.SkyeScyllaActiveConfigIds, ",")
			if !slices.Contains(activeConfigIds, fmt.Sprintf("%d", storePayload.ConfID)) {
				return fmt.Errorf("invalid config_id: %d. Allowed config_ids: %s", storePayload.ConfID, strings.Join(activeConfigIds, ", "))
			}
			return nil
		})
}

func (s *skyeConfig) ApproveStoreRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Store",
		func(id int) (RequestInfo, error) {
			req, err := s.StoreRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.StoreRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			payload, err := parsePayload[StoreRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse store request payload: %w", err)
			}
			if err := s.EtcdConfig.RegisterStore(payload.ConfID, payload.DB, payload.EmbeddingsTable, payload.AggregatorTable); err != nil {
				return fmt.Errorf("failed to create ETCD store configuration: %w", err)
			}
			scyllaStore := s.ScyllaStores[payload.ConfID]
			if err := scyllaStore.CreateEmbeddingTable(payload.EmbeddingsTable, 0, s.VariantsList); err != nil {
				log.Error().Err(err).Msgf("Failed to create Scylla embeddings table: %s", payload.EmbeddingsTable)
				return fmt.Errorf("failed to create Scylla embeddings table %s: %w", payload.EmbeddingsTable, err)
			}
			if err := scyllaStore.CreateAggregatorTable(payload.AggregatorTable, 0); err != nil {
				log.Error().Err(err).Msgf("Failed to create Scylla aggregator table: %s", payload.AggregatorTable)
				return fmt.Errorf("failed to create Scylla aggregator table %s: %w", payload.AggregatorTable, err)
			}
			return nil
		})
}

func (s *skyeConfig) GetAllStoreRequests() (StoreRequestListResponse, error) {
	requests, err := s.StoreRequestRepo.GetAll()
	if err != nil {
		return StoreRequestListResponse{}, fmt.Errorf("failed to get store requests: %w", err)
	}
	return StoreRequestListResponse{StoreRequests: requests, TotalCount: len(requests)}, nil
}

func (s *skyeConfig) GetStores() (StoreListResponse, error) {
	storeConfigs, err := s.EtcdConfig.GetStores()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stores from ETCD")
		return StoreListResponse{}, fmt.Errorf("failed to get stores from ETCD: %w", err)
	}

	var stores []StoreInfo
	for storeId, storeConfig := range storeConfigs {
		stores = append(stores, StoreInfo{
			ID:              storeId,
			ConfID:          storeConfig.ConfId,
			DB:              storeConfig.Db,
			EmbeddingsTable: storeConfig.EmbeddingTable,
			AggregatorTable: storeConfig.AggregatorTable,
		})
	}
	return StoreListResponse{Stores: stores}, nil
}

func (s *skyeConfig) RegisterEntity(request EntityRegisterRequest) (RequestStatus, error) {
	return createRequest(request.Payload, request.Reason, request.Requestor, request.RequestType, "Entity registration",
		func(payloadJSON string) (int, time.Time, error) {
			req := &entity_requests.EntityRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: request.RequestType,
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.EntityRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		func(payload interface{}) error {
			entityPayload, ok := payload.(EntityRequestPayload)
			if !ok {
				return fmt.Errorf("invalid payload type for entity registration")
			}
			// Validate that the store exists in etcd
			stores, err := s.EtcdConfig.GetStores()
			if err != nil {
				return fmt.Errorf("failed to get stores from etcd: %w", err)
			}
			if _, exists := stores[entityPayload.StoreID]; !exists {
				return fmt.Errorf("store with id '%s' is not registered in etcd", entityPayload.StoreID)
			}
			entities, err := s.EtcdConfig.GetEntities()
			if err != nil {
				return fmt.Errorf("failed to get entities from etcd: %w", err)
			}
			if _, exists := entities[entityPayload.Entity]; exists {
				return fmt.Errorf("entity with name '%s' is already registered in etcd", entityPayload.Entity)
			}
			return nil
		})
}

func (s *skyeConfig) ApproveEntityRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Entity",
		func(id int) (RequestInfo, error) {
			req, err := s.EntityRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.EntityRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			payload, err := parsePayload[EntityRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse entity request payload: %w", err)
			}
			if err := s.EtcdConfig.RegisterEntity(payload.Entity, payload.StoreID); err != nil {
				log.Error().Err(err).Msgf("Failed to register entity in ETCD for entity: %s, store_id: %s", payload.Entity, payload.StoreID)
				return fmt.Errorf("failed to create ETCD entity configuration: %w", err)
			}
			log.Info().Msgf("Successfully registered entity in ETCD for entity: %s, store_id: %s", payload.Entity, payload.StoreID)
			return nil
		})
}

func (s *skyeConfig) GetEntities() (EntityListResponse, error) {
	entityConfigs, err := s.EtcdConfig.GetEntities()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get entities from ETCD")
		return EntityListResponse{}, fmt.Errorf("failed to get entities from ETCD: %w", err)
	}
	var entities []EntityInfo
	for entityId, entityConfig := range entityConfigs {
		entities = append(entities, EntityInfo{Name: entityId, StoreID: entityConfig.StoreId})
	}
	return EntityListResponse{Entities: entities}, nil
}

func (s *skyeConfig) GetAllEntityRequests() (EntityRequestListResponse, error) {
	requests, err := s.EntityRequestRepo.GetAll()
	if err != nil {
		return EntityRequestListResponse{}, fmt.Errorf("failed to get entity requests: %w", err)
	}
	return EntityRequestListResponse{EntityRequests: requests, TotalCount: len(requests)}, nil
}

func (s *skyeConfig) RegisterModel(request ModelRegisterRequest) (RequestStatus, error) {

	return createRequest(request.Payload, request.Reason, request.Requestor, RequestTypeCreate, "Model registration",
		func(payloadJSON string) (int, time.Time, error) {
			req := &model_requests.ModelRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: RequestTypeCreate,
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.ModelRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		func(payload interface{}) error {
			modelPayload, ok := payload.(ModelRequestPayload)
			if !ok {
				return fmt.Errorf("invalid payload type for model registration")
			}

			// 1. Check entity is registered
			entities, err := s.EtcdConfig.GetEntities()
			if err != nil {
				return fmt.Errorf("failed to get entities from etcd: %w", err)
			}
			if _, exists := entities[modelPayload.Entity]; !exists {
				return fmt.Errorf("entity with name '%s' is not registered in etcd", modelPayload.Entity)
			}

			// 2. Check model is new and not present
			_, err = s.EtcdConfig.GetModelConfig(modelPayload.Entity, modelPayload.Model)
			if err == nil {
				return fmt.Errorf("model '%s' for entity '%s' already exists in etcd", modelPayload.Model, modelPayload.Entity)
			}

			// 3. Check model_type is RESET or DELTA
			if modelPayload.ModelType != "RESET" && modelPayload.ModelType != "DELTA" {
				return fmt.Errorf("invalid model_type: %s. Only 'RESET' or 'DELTA' are allowed", modelPayload.ModelType)
			}

			// 4. Check frequency is registered and present
			frequencies, err := s.EtcdConfig.GetFrequencies()
			if err != nil {
				return fmt.Errorf("failed to get frequencies from etcd: %w", err)
			}
			if _, exists := frequencies[modelPayload.JobFrequency]; !exists {
				return fmt.Errorf("job frequency '%s' is not registered in etcd", modelPayload.JobFrequency)
			}

			// 5. Check mqId and topic_name match the mapping
			expectedTopic, exists := s.MQIdTopicsMapping[modelPayload.MQID]
			if !exists {
				return fmt.Errorf("mq_id %d is not present in mq_id_topics_mapping", modelPayload.MQID)
			}
			if expectedTopic != modelPayload.TopicName {
				return fmt.Errorf("topic_name '%s' does not match the mapping for mq_id %d (expected: '%s')", modelPayload.TopicName, modelPayload.MQID, expectedTopic)
			}

			if modelPayload.TrainingDataPath == "" {
				return fmt.Errorf("training_data_path must be provided")
			}

			if modelPayload.ModelConfig.DistanceFunction == "" {
				return fmt.Errorf("model_config must be provided and distance_function must be set")
			}
			allowedDistanceFunctions := map[string]bool{
				"DOT":       true,
				"EUCLIDEAN": true,
				"COSINE":    true,
				"MANHATTAN": true,
			}
			if !allowedDistanceFunctions[modelPayload.ModelConfig.DistanceFunction] {
				return fmt.Errorf("model_config.distance_function '%s' is invalid. Allowed values: DOT, EUCLIDEAN, COSINE, MANHATTAN", modelPayload.ModelConfig.DistanceFunction)
			}

			return nil
		})
}

func (s *skyeConfig) ApproveModelRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Model",
		func(id int) (RequestInfo, error) {
			req, err := s.ModelRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.ModelRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			numberOfPartitions := s.AppConfig.SkyeNumberOfPartitions
			if numberOfPartitions == 0 {
				numberOfPartitions = 24
			}
			failureProducerMqId := s.AppConfig.SkyeFailureProducerMqId
			payload, err := parsePayload[ModelRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse model request payload: %w", err)
			}
			if err := s.EtcdConfig.RegisterModel(payload.Entity, payload.Model, payload.EmbeddingStoreEnabled,
				payload.EmbeddingStoreTTL, payload.ModelConfig, payload.ModelType, payload.MQID, payload.TrainingDataPath,
				payload.Metadata, payload.JobFrequency, numberOfPartitions, failureProducerMqId, payload.TopicName); err != nil {
				log.Error().Err(err).Msgf("Failed to register model in ETCD for model: %s", payload.Model)
				return fmt.Errorf("failed to register model in ETCD: %w", err)
			}
			log.Info().Msgf("Successfully registered model in ETCD for model: %s", payload.Model)
			return nil
		})
}

func (s *skyeConfig) GetModels() (ModelListResponse, error) {
	modelsMap, err := s.EtcdConfig.GetEntities()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get models from ETCD")
		return ModelListResponse{}, fmt.Errorf("failed to get models from ETCD: %w", err)
	}
	return ModelListResponse{Models: modelsMap}, nil
}

func (s *skyeConfig) GetAllModelRequests() (ModelRequestListResponse, error) {
	requests, err := s.ModelRequestRepo.GetAll()
	if err != nil {
		return ModelRequestListResponse{}, fmt.Errorf("failed to get model requests: %w", err)
	}
	return ModelRequestListResponse{ModelRequests: requests, TotalCount: len(requests)}, nil
}

func (s *skyeConfig) RegisterVariant(request VariantRegisterRequest) (RequestStatus, error) {
	return createRequest(request.Payload, request.Reason, request.Requestor, RequestTypeCreate, "Variant registration",
		func(payloadJSON string) (int, time.Time, error) {
			req := &variant_requests.VariantRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: RequestTypeCreate,
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.VariantRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		func(payload interface{}) error {

			variantPayload, ok := payload.(VariantRequestPayload)
			if !ok {
				return fmt.Errorf("invalid payload type for variant registration")
			}

			// Entity should be valid (exists)
			entities, err := s.EtcdConfig.GetEntities()
			if err != nil {
				return fmt.Errorf("failed to get entities from etcd: %w", err)
			}
			models, ok := entities[variantPayload.Entity]
			if !ok {
				return fmt.Errorf("entity with name '%s' does not exist", variantPayload.Entity)
			}

			// Model name should be valid (exists under entity)
			_, ok = models.Models[variantPayload.Model]
			if !ok {
				return fmt.Errorf("model with name '%s' does not exist for entity '%s'", variantPayload.Model, variantPayload.Entity)
			}

			// Variant should be present from variantList in config (i.e., must be pre-defined as allowed)
			allowedVariants := s.VariantsList
			if !slices.Contains(allowedVariants, variantPayload.Variant) {
				return fmt.Errorf("variant '%s' is not in allowed variant list: %v", variantPayload.Variant, allowedVariants)
			}

			// Check that all filters specified in payload.FilterConfiguration.Criteria exist in etcd for the specified entity
			filtersMap := models.Filters
			for _, filters := range variantPayload.FilterConfiguration.Criteria {
				if _, ok := filtersMap[filters.ColumnName]; !ok {
					return fmt.Errorf("filter '%s' does not exist for entity '%s'", filters.ColumnName, variantPayload.Entity)
				}
			}

			return nil
		})
}

func (s *skyeConfig) ApproveVariantRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Variant",
		func(id int) (RequestInfo, error) {
			req, err := s.VariantRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.VariantRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			payload, err := parsePayload[VariantRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse variant request payload: %w", err)
			}

			// Admin must provide VectorDBConfig, RateLimiter, and RTPartition during approval
			if approval.AdminVectorDBConfig.ReadHost == "" || approval.AdminVectorDBConfig.WriteHost == "" || approval.AdminVectorDBConfig.Port == "" {
				return fmt.Errorf("admin must provide vector_db_config during variant approval")
			}
			if approval.AdminRateLimiter.RateLimit == 0 || approval.AdminRateLimiter.BurstLimit == 0 {
				return fmt.Errorf("admin must provide rate_limiter during variant approval")
			}
			if approval.AdminCachingConfiguration.DistributedCachingEnabled || approval.AdminCachingConfiguration.DistributedCacheTTLSeconds == 0 || approval.AdminCachingConfiguration.InMemoryCachingEnabled || approval.AdminCachingConfiguration.InMemoryCacheTTLSeconds == 0 {
				return fmt.Errorf("admin must provide caching_configuration during variant approval")
			}
			// For RT Partition, collect all RT partitions for all models across all entities
			rtPartitions := make(map[int]bool)
			entities, err := s.EtcdConfig.GetEntities()
			if err != nil {
				return fmt.Errorf("failed to get entities from etcd: %w", err)
			}
			for _, models := range entities {
				for _, modelConfig := range models.Models {
					for _, variant := range modelConfig.Variants {
						if variant.RTPartition > 0 {
							rtPartitions[variant.RTPartition] = true
						}
					}
				}
			}
			// Choose a random partition between 1 and 256 not present in the used list
			availablePartitions := []int{}
			for i := 1; i <= 256; i++ {
				if _, exists := rtPartitions[i]; !exists {
					availablePartitions = append(availablePartitions, i)
				}
			}
			if len(availablePartitions) == 0 {
				return fmt.Errorf("no RT partitions available (all 1-256 are used)")
			}
			// Seed RNG with nanoseconds
			seed := time.Now().UnixNano()
			rnd := int(seed % int64(len(availablePartitions)))
			if rnd < 0 {
				rnd = -rnd
			}
			// Fetch filters from etcd for the provided entity
			filtersFromEtcd, err := s.EtcdConfig.GetFilters(payload.Entity)
			if err != nil {
				return fmt.Errorf("failed to fetch filters from etcd: %w", err)
			}
			criteria := []skyeEtcd.Criteria{}
			for _, filter := range payload.FilterConfiguration.Criteria {
				if _, ok := filtersFromEtcd[filter.ColumnName]; ok {
					filterValue := filtersFromEtcd[filter.ColumnName].FilterValue
					defaultValue := filtersFromEtcd[filter.ColumnName].DefaultValue
					if filter.Condition == enums.FilterCondition(enums.NOT_EQUALS) {
						filterValue = filtersFromEtcd[filter.ColumnName].DefaultValue
						defaultValue = filtersFromEtcd[filter.ColumnName].FilterValue

					}
					criteria = append(criteria, skyeEtcd.Criteria{
						ColumnName:   filter.ColumnName,
						FilterValue:  filterValue,
						DefaultValue: defaultValue,
					})
				}
			}

			if err := s.EtcdConfig.RegisterVariant(payload.Entity, payload.Model, payload.Variant,
				approval.AdminVectorDBConfig, payload.VectorDBType, criteria,
				payload.Type, approval.AdminCachingConfiguration.DistributedCachingEnabled, approval.AdminCachingConfiguration.DistributedCacheTTLSeconds,
				approval.AdminCachingConfiguration.InMemoryCachingEnabled, approval.AdminCachingConfiguration.InMemoryCacheTTLSeconds,
				approval.AdminCachingConfiguration.EmbeddingRetrievalInMemoryConfig.Enabled, approval.AdminCachingConfiguration.EmbeddingRetrievalInMemoryConfig.TTL,
				approval.AdminCachingConfiguration.EmbeddingRetrievalDistributedConfig.Enabled, approval.AdminCachingConfiguration.EmbeddingRetrievalDistributedConfig.TTL,
				approval.AdminCachingConfiguration.DotProductInMemoryConfig.Enabled, approval.AdminCachingConfiguration.DotProductInMemoryConfig.TTL,
				approval.AdminCachingConfiguration.DotProductDistributedConfig.Enabled, approval.AdminCachingConfiguration.DotProductDistributedConfig.TTL,
				availablePartitions[rnd], approval.AdminRateLimiter); err != nil {
				log.Error().Err(err).Msgf("Failed to register variant in ETCD for variant: %s", payload.Variant)
				return fmt.Errorf("failed to register variant in ETCD: %w", err)
			}

			task := &variant_onboarding_tasks.VariantOnboardingTask{
				Entity:  payload.Entity,
				Model:   payload.Model,
				Variant: payload.Variant,
				Payload: "{}",
				Status:  StatusPending,
			}
			if err := s.VariantOnboardingTaskRepo.Create(task); err != nil {
				log.Error().Err(err).Msgf("Failed to create variant onboarding task for request %d", requestID)
				return fmt.Errorf("failed to create variant onboarding task: %w", err)
			}
			log.Info().Int("request_id", requestID).Int("task_id", task.TaskID).
				Str("entity", payload.Entity).Str("model", payload.Model).Str("variant", payload.Variant).
				Msg("Created variant onboarding task")
			return nil
		})
}

func (s *skyeConfig) GetVariants(entity string, model string) (VariantListResponse, error) {
	modelConfig, err := s.EtcdConfig.GetModelConfig(entity, model)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get variants from ETCD")
		return VariantListResponse{}, fmt.Errorf("failed to get variants from ETCD: %w", err)
	}
	return VariantListResponse{Variants: modelConfig.Variants}, nil
}

func (s *skyeConfig) GetAllVariantRequests() (VariantRequestListResponse, error) {
	requests, err := s.VariantRequestRepo.GetAll()
	if err != nil {
		return VariantRequestListResponse{}, fmt.Errorf("failed to get variant requests: %w", err)
	}
	return VariantRequestListResponse{VariantRequests: requests, TotalCount: len(requests)}, nil
}

func (s *skyeConfig) RegisterFilter(request FilterRegisterRequest) (RequestStatus, error) {
	return createRequest(request.Payload, request.Reason, request.Requestor, RequestTypeCreate, "Filter registration",
		func(payloadJSON string) (int, time.Time, error) {
			req := &filter_requests.FilterRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: RequestTypeCreate,
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.FilterRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		func(payload interface{}) error {
			// Check if the entity is registered in ETCD
			filterPayload, ok := payload.(FilterRequestPayload)
			if !ok {
				return fmt.Errorf("invalid payload type for filter registration")
			}
			_, err := s.EtcdConfig.GetEntityConfig(filterPayload.Entity)
			if err != nil {
				return fmt.Errorf("entity '%s' not found in ETCD, cannot register filter: %w", filterPayload.Entity, err)
			}
			// Check if the column name already exists for the given entity
			filters, err := s.EtcdConfig.GetFilters(filterPayload.Entity)
			if err != nil {
				return fmt.Errorf("failed to fetch filters for entity %s: %w", filterPayload.Entity, err)
			}
			for _, f := range filters {
				if f.ColumnName == filterPayload.Filter.ColumnName {
					return fmt.Errorf("column_name '%s' is already registered for entity '%s'", filterPayload.Filter.ColumnName, filterPayload.Entity)
				}
			}

			if filterPayload.Filter.FilterValue == "" {
				return fmt.Errorf("filter_value is required")
			}
			if filterPayload.Filter.DefaultValue == "" {
				return fmt.Errorf("default_value is required")
			}
			return nil
		})
}

func (s *skyeConfig) ApproveFilterRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Filter",
		func(id int) (RequestInfo, error) {
			req, err := s.FilterRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.FilterRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			payload, err := parsePayload[FilterRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse filter request payload: %w", err)
			}
			if err := s.EtcdConfig.RegisterFilter(payload.Entity, payload.Filter.ColumnName, payload.Filter.FilterValue, payload.Filter.DefaultValue); err != nil {
				log.Error().Err(err).Msgf("Failed to create filter config in ETCD for filter_id: %s", payload.Filter.ColumnName)
				return fmt.Errorf("failed to create ETCD filter configuration: %w", err)
			}
			log.Info().Msgf("Successfully created ETCD filter configuration for filter_id: %s", payload.Filter.ColumnName)
			entityConfig, err := s.EtcdConfig.GetEntityConfig(payload.Entity)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to fetch entity ETCD config for '%s' while adding column to aggregator table", payload.Entity)
				return fmt.Errorf("failed to fetch entity ETCD config: %w", err)
			}
			storeId := entityConfig.StoreId
			storePayload, err := s.EtcdConfig.GetStores()
			if err != nil {
				log.Error().Err(err).Msgf("Failed to fetch store config for store '%s' while adding column to aggregator table", storeId)
				return fmt.Errorf("failed to fetch store config: %w", err)
			}
			aggregatorTable := storePayload[storeId].AggregatorTable
			scyllaStore := s.ScyllaStores[storePayload[storeId].ConfId]
			if err := scyllaStore.AddAggregatorColumn(aggregatorTable, payload.Filter.ColumnName); err != nil {
				log.Error().Err(err).Msgf("Failed to add column '%s' to aggregator table '%s' for store '%s'", payload.Filter.ColumnName, aggregatorTable, storeId)
				return fmt.Errorf("failed to add column '%s' to aggregator table '%s': %w", payload.Filter.ColumnName, aggregatorTable, err)
			}
			return nil
		})
}

func (s *skyeConfig) GetFilters(entity string) (FilterListResponse, error) {
	filters, err := s.EtcdConfig.GetFilters(entity)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get filters from ETCD")
		return FilterListResponse{}, fmt.Errorf("failed to get filters from ETCD: %w", err)
	}
	return FilterListResponse{Filters: filters}, nil
}

func (s *skyeConfig) GetAllFilters() (AllFiltersListResponse, error) {
	entities, err := s.EtcdConfig.GetEntities()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get entities from ETCD")
		return AllFiltersListResponse{}, fmt.Errorf("failed to get entities from ETCD: %w", err)
	}
	filters := make(map[string]map[string]skyeEtcd.Criteria)
	for entityName := range entities {
		filtersForEntity, err := s.EtcdConfig.GetFilters(entityName)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get filters from ETCD")
			return AllFiltersListResponse{}, fmt.Errorf("failed to get filters from ETCD: %w", err)
		}
		filters[entityName] = filtersForEntity
	}
	return AllFiltersListResponse{Filters: filters}, nil
}

func (s *skyeConfig) GetAllFilterRequests() (FilterRequestListResponse, error) {
	requests, err := s.FilterRequestRepo.GetAll()
	if err != nil {
		return FilterRequestListResponse{}, fmt.Errorf("failed to get filter requests: %w", err)
	}
	return FilterRequestListResponse{FilterRequests: requests, TotalCount: len(requests)}, nil
}

func (s *skyeConfig) RegisterJobFrequency(request JobFrequencyRegisterRequest) (RequestStatus, error) {
	return createRequest(request.Payload, request.Reason, request.Requestor, RequestTypeCreate, "Job frequency registration",
		func(payloadJSON string) (int, time.Time, error) {
			req := &job_frequency_requests.JobFrequencyRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: RequestTypeCreate,
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.JobFrequencyRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		nil)
}

func (s *skyeConfig) ApproveJobFrequencyRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Job frequency",
		func(id int) (RequestInfo, error) {
			req, err := s.JobFrequencyRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.JobFrequencyRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			payload, err := parsePayload[JobFrequencyRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse job frequency request payload: %w", err)
			}
			if err := s.EtcdConfig.RegisterFrequency(payload.JobFrequency); err != nil {
				log.Error().Err(err).Msgf("Failed to create job frequency config in ETCD for frequency_id: %s", payload.JobFrequency)
				return fmt.Errorf("failed to create ETCD job frequency configuration: %w", err)
			}
			log.Info().Msgf("Successfully created ETCD job frequency configuration for frequency_id: %s", payload.JobFrequency)
			return nil
		})
}

func (s *skyeConfig) GetJobFrequencies() (JobFrequencyListResponse, error) {
	frequencies, err := s.EtcdConfig.GetFrequencies()
	if err != nil {
		return JobFrequencyListResponse{}, fmt.Errorf("failed to get job frequencies from ETCD: %w", err)
	}
	return JobFrequencyListResponse{Frequencies: frequencies, TotalCount: len(frequencies)}, nil
}

func (s *skyeConfig) GetAllJobFrequencyRequests() (JobFrequencyRequestListResponse, error) {
	requests, err := s.JobFrequencyRequestRepo.GetAll()
	if err != nil {
		return JobFrequencyRequestListResponse{}, fmt.Errorf("failed to get job frequency requests: %w", err)
	}
	return JobFrequencyRequestListResponse{JobFrequencyRequests: requests, TotalCount: len(requests)}, nil
}

// func (s *skyeConfig) OnboardVariant(request VariantOnboardingRequest) (RequestStatus, error) {
// 	return createRequest(request.Payload, request.Reason, request.Requestor, "ONBOARD", "Variant onboarding",
// 		func(payloadJSON string) (int, time.Time, error) {
// 			req := &variant_onboarding_requests.VariantOnboardingRequest{
// 				Reason:      request.Reason,
// 				Payload:     payloadJSON,
// 				RequestType: "ONBOARD",
// 				CreatedBy:   request.Requestor,
// 				Status:      StatusPending,
// 			}
// 			err := s.VariantOnboardingRequestRepo.Create(req)
// 			return req.RequestID, req.CreatedAt, err
// 		},
// 		nil)
// }

// func (s *skyeConfig) ApproveVariantOnboardingRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
// 	return approveRequest(requestID, approval, "Variant onboarding",
// 		func(id int) (RequestInfo, error) {
// 			req, err := s.VariantOnboardingRequestRepo.GetByID(id)
// 			if err != nil {
// 				return RequestInfo{}, err
// 			}
// 			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
// 		},
// 		s.VariantOnboardingRequestRepo.UpdateStatus,
// 		func(payloadJSON string) error {

// 		})
// }

// func (s *skyeConfig) GetAllVariantOnboardingRequests() (VariantOnboardingRequestListResponse, error) {
// 	requests, err := s.VariantOnboardingRequestRepo.GetAll()
// 	if err != nil {
// 		return VariantOnboardingRequestListResponse{}, fmt.Errorf("failed to get variant onboarding requests: %w", err)
// 	}
// 	return VariantOnboardingRequestListResponse{VariantOnboardingRequests: requests, TotalCount: len(requests)}, nil
// }

func (s *skyeConfig) GetVariantOnboardingTasks() (VariantOnboardingTaskListResponse, error) {
	tasks, err := s.VariantOnboardingTaskRepo.GetAll()
	if err != nil {
		return VariantOnboardingTaskListResponse{}, fmt.Errorf("failed to get variant onboarding tasks: %w", err)
	}
	return VariantOnboardingTaskListResponse{VariantOnboardingTasks: tasks, TotalCount: len(tasks)}, nil
}

func (s *skyeConfig) ScaleUpVariant(request VariantScaleUpRequest) (RequestStatus, error) {
	return createRequest(request.Payload, request.Reason, request.Requestor, "SCALE_UP", "Variant scale up",
		func(payloadJSON string) (int, time.Time, error) {
			req := &variant_scaleup_requests.VariantScaleUpRequest{
				Reason:      request.Reason,
				Payload:     payloadJSON,
				RequestType: "SCALE_UP",
				CreatedBy:   request.Requestor,
				Status:      StatusPending,
			}
			err := s.VariantScaleUpRequestRepo.Create(req)
			return req.RequestID, req.CreatedAt, err
		},
		nil)
}

func (s *skyeConfig) ApproveVariantScaleUpRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	return approveRequest(requestID, approval, "Variant scale up",
		func(id int) (RequestInfo, error) {
			req, err := s.VariantScaleUpRequestRepo.GetByID(id)
			if err != nil {
				return RequestInfo{}, err
			}
			return RequestInfo{RequestID: req.RequestID, Status: req.Status, Payload: req.Payload, CreatedAt: req.CreatedAt}, nil
		},
		s.VariantScaleUpRequestRepo.UpdateStatus,
		func(payloadJSON string) error {
			payload, err := parsePayload[VariantScaleUpRequestPayload](payloadJSON)
			if err != nil {
				return fmt.Errorf("failed to parse variant scale up request payload: %w", err)
			}

			task := &variant_scaleup_tasks.VariantScaleUpTask{
				Entity:           payload.Entity,
				ScaleUpHost:      approval.ScaleUpHost,
				Model:            payload.Model,
				Variant:          payload.Variant,
				VectorDBType:     payload.VectorDBType,
				TrainingDataPath: payload.TrainingDataPath,
				Payload:          "{}",
				Status:           StatusPending,
			}
			if err := s.VariantScaleUpTaskRepo.Create(task); err != nil {
				log.Error().Err(err).Msgf("Failed to create variant scale up task for request %d", requestID)
				return fmt.Errorf("failed to create variant scale up task: %w", err)
			}
			log.Info().Int("request_id", requestID).
				Str("entity", payload.Entity).Str("model", payload.Model).Str("variant", payload.Variant).
				Msg("Created variant scale up task")
			return nil
		})
}

func (s *skyeConfig) GetAllVariantScaleUpRequests() (VariantScaleUpRequestListResponse, error) {
	requests, err := s.VariantScaleUpRequestRepo.GetAll()
	if err != nil {
		return VariantScaleUpRequestListResponse{}, fmt.Errorf("failed to get variant scale up requests: %w", err)
	}
	return VariantScaleUpRequestListResponse{VariantScaleUpRequests: requests, TotalCount: len(requests)}, nil
}

func (s *skyeConfig) GetVariantScaleUpTasks() (VariantScaleUpTaskListResponse, error) {
	tasks, err := s.VariantScaleUpTaskRepo.GetAll()
	if err != nil {
		return VariantScaleUpTaskListResponse{}, fmt.Errorf("failed to get variant scale up tasks: %w", err)
	}
	return VariantScaleUpTaskListResponse{VariantScaleUpTasks: tasks, TotalCount: len(tasks)}, nil
}

func (s *skyeConfig) GetMQIdTopics() (MQIdTopicsResponse, error) {
	var mappings []MQIdTopicMapping
	for mqID, topic := range s.MQIdTopicsMapping {
		mappings = append(mappings, MQIdTopicMapping{MQID: mqID, Topic: topic})
	}
	return MQIdTopicsResponse{Mappings: mappings, TotalCount: len(mappings)}, nil
}

func (s *skyeConfig) GetVariantsList() (VariantsListResponse, error) {
	return VariantsListResponse{Variants: s.VariantsList, TotalCount: len(s.VariantsList)}, nil
}

func (s *skyeConfig) TestVariant(request VariantTestRequest) (VariantTestResponse, error) {
	grpcRequest := &grpc.SkyeRequest{
		Entity:        request.Entity,
		CandidateIds:  request.CandidateIds,
		Limit:         request.Limit,
		ModelName:     request.ModelName,
		Variant:       request.Variant,
		Filters:       request.Filters,
		Attribute:     request.Attribute,
		Embeddings:    request.Embeddings,
		GlobalFilters: request.GlobalFilters,
	}
	taskStatus, err := s.VariantOnboardingTaskRepo.GetByEntityAndModelAndVariant(request.Entity, request.ModelName, request.Variant)
	if err != nil {
		return VariantTestResponse{}, fmt.Errorf("failed to get variant onboarding task: %w", err)
	}
	if taskStatus.Status == "IN_PROGRESS" || taskStatus.Status == "PENDING" {
		return VariantTestResponse{}, fmt.Errorf("variant onboarding task is still in progress")
	}
	grpcResponse, err := s.SkyeClient.GetSimilarCandidates(grpcRequest)
	if err != nil {
		return VariantTestResponse{}, fmt.Errorf("failed to test variant: %w", err)
	}
	s.VariantOnboardingTaskRepo.UpdateStatus(taskStatus.TaskID, "TESTED_SUCCESSFULLY")
	return VariantTestResponse{Response: grpcResponse}, nil
}

func (s *skyeConfig) GenerateTestRequest(request TestRequestGenerationRequest) (TestRequestGenerationResponse, error) {
	variantTestRequest := VariantTestRequest{
		Entity:        request.Entity,
		ModelName:     request.Model,
		Variant:       request.Variant,
		Limit:         200,
		Filters:       nil,
		Attribute:     []string{},
		GlobalFilters: nil,
	}

	modelConfig, err := s.EtcdConfig.GetModelConfig(request.Entity, request.Model)
	if err != nil {
		return TestRequestGenerationResponse{}, fmt.Errorf("failed to get model config for model %s: %w", request.Model, err)
	}
	vectorDim := modelConfig.ModelConfig.VectorDimension
	embedding := make([]float64, vectorDim)
	for i := range vectorDim {
		embedding[i] = rand.Float64()
	}
	variantTestRequest.Embeddings = []*grpc.Embedding{{Embedding: embedding}}

	return TestRequestGenerationResponse{Request: variantTestRequest}, nil
}
