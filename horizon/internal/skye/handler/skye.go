package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/scylla"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/entity_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/filter_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/job_frequency_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/model_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/qdrant_cluster_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/store_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_tasks"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_promotion_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye"
	skyeEtcd "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

const (
	StatusPending    = "PENDING"
	StatusApproved   = "APPROVED"
	StatusRejected   = "REJECTED"
	StatusInProgress = "IN_PROGRESS"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
	StatusSuccess    = "SUCCESS"
)

type skyeConfig struct {
	StoreRequestRepo             store_requests.StoreRequestRepository
	EntityRequestRepo            entity_requests.EntityRequestRepository
	ModelRequestRepo             model_requests.ModelRequestRepository
	VariantRequestRepo           variant_requests.VariantRequestRepository
	FilterRequestRepo            filter_requests.FilterRequestRepository
	JobFrequencyRequestRepo      job_frequency_requests.JobFrequencyRequestRepository
	QdrantClusterRequestRepo     qdrant_cluster_requests.QdrantClusterRequestRepository
	VariantPromotionRequestRepo  variant_promotion_requests.VariantPromotionRequestRepository
	VariantOnboardingRequestRepo variant_onboarding_requests.VariantOnboardingRequestRepository
	VariantOnboardingTaskRepo    variant_onboarding_tasks.VariantOnboardingTaskRepository
	EtcdConfig                   skyeEtcd.Manager
	ScyllaStores                 map[int]scylla.Store
	ActiveScyllaConfigIDs        map[int]bool
	MQIdTopicsMapping            map[int]string
	VariantsList                 []string
	AppConfig                    configs.Configs
}

func InitV1ConfigHandler(appConfig configs.Configs) Config {
	if config == nil {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn := conn.(*infra.SQLConnection)

		// Initialize all repositories
		storeRequestRepo, err := store_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create store request repository")
		}
		variantOnboardingTaskRepo, err := variant_onboarding_tasks.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create variant onboarding task repository")
		}

		entityRequestRepo, err := entity_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create entity request repository")
		}

		modelRequestRepo, err := model_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create model request repository")
		}

		variantRequestRepo, err := variant_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create variant request repository")
		}

		filterRequestRepo, err := filter_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create filter request repository")
		}

		jobFrequencyRequestRepo, err := job_frequency_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create job frequency request repository")
		}

		qdrantClusterRequestRepo, err := qdrant_cluster_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create qdrant cluster request repository")
		}

		variantPromotionRequestRepo, err := variant_promotion_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create variant promotion request repository")
		}

		variantOnboardingRequestRepo, err := variant_onboarding_requests.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create variant onboarding request repository")
		}

		// Initialize Scylla stores
		scyllaStores := make(map[int]scylla.Store)
		activeScyllaConfigIDs := make(map[int]bool)

		// Get active Scylla config IDs from environment
		activeScyllaConfIds := skye.ScyllaActiveConfigIds
		if activeScyllaConfIds != "" {
			confIds := strings.Split(activeScyllaConfIds, ",")
			for _, confIdStr := range confIds {
				confIdStr = strings.TrimSpace(confIdStr)
				confId, err := strconv.Atoi(confIdStr)
				if err != nil {
					log.Error().Err(err).Msgf("Error converting Scylla config ID %s to int", confIdStr)
					continue
				}

				// Add to active config IDs set for validation
				activeScyllaConfigIDs[confId] = true

				if infra.Scylla != nil {
					connFacade, err := infra.Scylla.GetConnection(confId)
					if err != nil {
						log.Error().Err(err).Msgf("Error getting Scylla connection for config ID %d", confId)
						continue
					}

					conn := connFacade.(*infra.ScyllaClusterConnection)
					scyllaStore, err := scylla.NewRepository(conn)
					if err != nil {
						log.Error().Err(err).Msgf("Error creating Scylla store for config ID %d", confId)
						continue
					}

					scyllaStores[confId] = scyllaStore
					log.Info().Msgf("Successfully initialized Scylla store for config ID %d", confId)
				}
			}
		} else {
			log.Warn().Msg("SCYLLA_ACTIVE_CONFIG_IDS not configured, running without Scylla stores")
		}

		mqIdTopicsMapping := make(map[int]string)
		mqIdTopicsMappingStr := skye.MQIdTopicsMapping
		if mqIdTopicsMappingStr != "" {
			var mqIdTopicsList []struct {
				MQID  int    `json:"mq_id"`
				Topic string `json:"topic"`
			}
			if err := json.Unmarshal([]byte(mqIdTopicsMappingStr), &mqIdTopicsList); err != nil {
				log.Error().Err(err).Msg("Error parsing MQ_ID_TOPICS_MAPPING JSON, running without MQ ID to topic mapping")
			} else {
				for _, mapping := range mqIdTopicsList {
					mqIdTopicsMapping[mapping.MQID] = mapping.Topic
					log.Info().Msgf("Loaded MQ ID %d with topic %s", mapping.MQID, mapping.Topic)
				}
			}
		} else {
			log.Warn().Msg("MQ_ID_TOPICS_MAPPING not configured, running without MQ ID to topic mapping")
		}

		// Initialize Variants List
		variantsList := []string{}
		variantsListStr := skye.VariantsList
		if variantsListStr != "" {
			if err := json.Unmarshal([]byte(variantsListStr), &variantsList); err != nil {
				log.Error().Err(err).Msg("Error parsing VARIANTS_LIST JSON, running without variants list")
			} else {
				log.Info().Msgf("Loaded %d variants", len(variantsList))
			}
		} else {
			log.Warn().Msg("VARIANTS_LIST not configured, running without variants list")
		}

		config = &skyeConfig{
			StoreRequestRepo:             storeRequestRepo,
			EntityRequestRepo:            entityRequestRepo,
			ModelRequestRepo:             modelRequestRepo,
			VariantRequestRepo:           variantRequestRepo,
			FilterRequestRepo:            filterRequestRepo,
			JobFrequencyRequestRepo:      jobFrequencyRequestRepo,
			QdrantClusterRequestRepo:     qdrantClusterRequestRepo,
			VariantPromotionRequestRepo:  variantPromotionRequestRepo,
			VariantOnboardingRequestRepo: variantOnboardingRequestRepo,
			VariantOnboardingTaskRepo:    variantOnboardingTaskRepo,
			EtcdConfig:                   skyeEtcd.NewEtcdConfig(),
			ScyllaStores:                 scyllaStores,
			ActiveScyllaConfigIDs:        activeScyllaConfigIDs,
			AppConfig:                    appConfig,
			MQIdTopicsMapping:            mqIdTopicsMapping,
			VariantsList:                 variantsList,
		}
		log.Info().Msg("Config initialization completed")
	} else {
		log.Info().Msg("Config already exists, reusing...")
	}
	return config
}

func (s *skyeConfig) RegisterStore(request StoreRegisterRequest) (RequestStatus, error) {
	// Validate that the config_id is in the list of active Scylla config IDs
	if !s.ActiveScyllaConfigIDs[request.Payload.ConfID] {
		// Build list of allowed config IDs for error message
		allowedConfigIDs := make([]string, 0, len(s.ActiveScyllaConfigIDs))
		for confID := range s.ActiveScyllaConfigIDs {
			allowedConfigIDs = append(allowedConfigIDs, fmt.Sprintf("%d", confID))
		}

		var allowedMsg string
		if len(allowedConfigIDs) > 0 {
			allowedMsg = fmt.Sprintf(" Allowed config_ids: %s", strings.Join(allowedConfigIDs, ", "))
		} else {
			allowedMsg = " No active Scylla config IDs are configured. Please set SCYLLA_ACTIVE_CONFIG_IDS environment variable."
		}

		return RequestStatus{}, fmt.Errorf("invalid config_id: %d.%s", request.Payload.ConfID, allowedMsg)
	}

	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	storeRequest := &store_requests.StoreRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: request.RequestType,
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}

	err = s.StoreRequestRepo.Create(storeRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create store request: %w", err)
	}

	return RequestStatus{
		RequestID: storeRequest.RequestID,
		Status:    StatusPending,
		Message:   "Store registration request submitted successfully. Awaiting admin approval.",
		CreatedAt: storeRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveStoreRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.StoreRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get store request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusRejected {
		err = s.StoreRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
		if err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to update store request status: %w", err)
		}
		log.Info().Msgf("Successfully updated store request status in database: request_id=%d, status=%s", requestID, approval.ApprovalDecision)
		return ApprovalResponse{
			RequestID:        requestID,
			Status:           approval.ApprovalDecision,
			Message:          "Store request rejected.",
			ApprovedBy:       approval.AdminID,
			ApprovedAt:       time.Now(),
			ProcessingStatus: StatusCompleted,
		}, nil
	}

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload StoreRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse store request payload: %w", err)
		}

		// Create ETCD configuration
		err = s.EtcdConfig.RegisterStore(payload.ConfID, payload.DB, payload.EmbeddingsTable, payload.AggregatorTable)
		if err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD store configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD store configuration for conf_id: %d", payload.ConfID)

		// Create Scylla tables
		if strings.ToLower(payload.DB) == "scylla" {
			scyllaStore := s.ScyllaStores[payload.ConfID]
			// Create embeddings table
			if payload.EmbeddingsTable != "" {
				// TODO: why are we using entity_id and embedding_id as primary keys? @atishay
				primaryKeys := []string{"entity_id", "embedding_id"}
				tableTTL := 0
				log.Info().Msgf("Creating Scylla embeddings table: %s", payload.EmbeddingsTable)
				err = scyllaStore.CreateTable(payload.EmbeddingsTable, primaryKeys, tableTTL)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create Scylla embeddings table: %s", payload.EmbeddingsTable)
					// TODO: Rollback ETCD config
					return ApprovalResponse{}, fmt.Errorf("failed to create Scylla embeddings table %s: %w", payload.EmbeddingsTable, err)
				}
				log.Info().Msgf("Successfully created Scylla embeddings table: %s", payload.EmbeddingsTable)
			}

			// Create aggregator table
			if payload.AggregatorTable != "" {
				// TODO: why are we using entity_id and embedding_id as primary keys? @atishay
				primaryKeys := []string{"entity_id", "timestamp"}
				tableTTL := 0

				log.Info().Msgf("Creating Scylla aggregator table: %s", payload.AggregatorTable)
				err = scyllaStore.CreateTable(payload.AggregatorTable, primaryKeys, tableTTL)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create Scylla aggregator table: %s", payload.AggregatorTable)
					// TODO: Rollback ETCD config
					return ApprovalResponse{}, fmt.Errorf("failed to create Scylla aggregator table %s: %w", payload.AggregatorTable, err)
				}
				log.Info().Msgf("Successfully created Scylla aggregator table: %s", payload.AggregatorTable)
			}
			log.Info().Msgf("Successfully created all Scylla tables for conf_id: %d", payload.ConfID)
		} else {
			log.Info().Msgf("DB type is '%s', skipping Scylla table creation", payload.DB)
		}
	}

	log.Info().Msgf("Updating store request status: request_id=%d, status=%s, admin_id=%s", requestID, approval.ApprovalDecision, approval.AdminID)
	err = s.StoreRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to update store request status in database: request_id=%d, status=%s", requestID, approval.ApprovalDecision)
		return ApprovalResponse{}, fmt.Errorf("failed to update store request status: %w", err)
	}

	log.Info().Msgf("Successfully updated store request status in database: request_id=%d, status=%s", requestID, approval.ApprovalDecision)

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Store request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) GetAllStoreRequests() (StoreRequestListResponse, error) {
	requests, err := s.StoreRequestRepo.GetAll()
	if err != nil {
		return StoreRequestListResponse{}, fmt.Errorf("failed to get store requests: %w", err)
	}
	return StoreRequestListResponse{
		StoreRequests: requests,
		TotalCount:    len(requests),
	}, nil
}

func (s *skyeConfig) GetStores() (StoreListResponse, error) {
	storeConfigs, err := s.EtcdConfig.GetStores()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stores from ETCD")
		return StoreListResponse{}, fmt.Errorf("failed to get stores from ETCD: %w", err)
	}
	var stores []StoreInfo
	for storeId, storeConfig := range storeConfigs {
		store := StoreInfo{
			ID:              storeId, // Use the actual store ID from ETCD
			ConfID:          storeConfig.ConfId,
			DB:              storeConfig.Db,
			EmbeddingsTable: storeConfig.EmbeddingTable,
			AggregatorTable: storeConfig.AggregatorTable,
		}
		stores = append(stores, store)
	}
	return StoreListResponse{
		Stores: stores,
	}, nil
}

func (s *skyeConfig) RegisterEntity(request EntityRegisterRequest) (RequestStatus, error) {
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	entityRequest := &entity_requests.EntityRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: request.RequestType,
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.EntityRequestRepo.Create(entityRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create entity request: %w", err)
	}
	return RequestStatus{
		RequestID: entityRequest.RequestID,
		Status:    StatusPending,
		Message:   "Entity registration request submitted successfully. Awaiting admin approval.",
		CreatedAt: entityRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveEntityRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.EntityRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get entity request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		var payload EntityRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse entity request payload: %w", err)
		}
		err = s.EtcdConfig.RegisterEntity(payload.Entity, payload.StoreID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to register entity in ETCD for entity: %s, store_id: %s", payload.Entity, payload.StoreID)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD entity configuration: %w", err)
		}
		log.Info().Msgf("Successfully registered entity in ETCD for entity: %s, store_id: %s", payload.Entity, payload.StoreID)
	}

	err = s.EntityRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update entity request status: %w", err)
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Entity request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) GetEntities() (EntityListResponse, error) {
	entityConfigs, err := s.EtcdConfig.GetEntities()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get entities from ETCD")
		return EntityListResponse{}, fmt.Errorf("failed to get entities from ETCD: %w", err)
	}
	var entities []EntityInfo
	for entityId, entityConfig := range entityConfigs {
		entities = append(entities, EntityInfo{
			Name:    entityId,
			StoreID: entityConfig.StoreId,
		})
	}
	return EntityListResponse{
		Entities: entities,
	}, nil
}

func (s *skyeConfig) GetMQIdTopics() (MQIdTopicsResponse, error) {
	var mappings []MQIdTopicMapping
	for mqID, topic := range s.MQIdTopicsMapping {
		mappings = append(mappings, MQIdTopicMapping{
			MQID:  mqID,
			Topic: topic,
		})
	}
	return MQIdTopicsResponse{
		Mappings:   mappings,
		TotalCount: len(mappings),
	}, nil
}

func (s *skyeConfig) GetVariantsList() (VariantsListResponse, error) {
	return VariantsListResponse{
		Variants:   s.VariantsList,
		TotalCount: len(s.VariantsList),
	}, nil
}

func (s *skyeConfig) GetAllEntityRequests() (EntityRequestListResponse, error) {
	requests, err := s.EntityRequestRepo.GetAll()
	if err != nil {
		return EntityRequestListResponse{}, fmt.Errorf("failed to get entity requests: %w", err)
	}
	return EntityRequestListResponse{
		EntityRequests: requests,
		TotalCount:     len(requests),
	}, nil
}

func (s *skyeConfig) RegisterModel(request ModelRegisterRequest) (RequestStatus, error) {
	// Validation: model_type can only be RESET or DELTA
	if request.Payload.ModelType != "RESET" && request.Payload.ModelType != "DELTA" {
		return RequestStatus{}, fmt.Errorf("invalid model_type: %s. Only 'RESET' or 'DELTA' are allowed", request.Payload.ModelType)
	}
	// TODO keep this configurable from cac
	request.Payload.NumberOfPartitions = 24
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	modelRequest := &model_requests.ModelRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "CREATE",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.ModelRequestRepo.Create(modelRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create model request: %w", err)
	}
	return RequestStatus{
		RequestID: modelRequest.RequestID,
		Status:    StatusPending,
		Message:   "Model registration request submitted successfully. Awaiting admin approval.",
		CreatedAt: modelRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) EditModel(request ModelEditRequest) (RequestStatus, error) {
	if _, exists := request.Payload.Updates["mq_id"]; exists {
		return RequestStatus{}, fmt.Errorf("mq_id is not editable and cannot be updated")
	}
	if _, exists := request.Payload.Updates["topic_name"]; exists {
		return RequestStatus{}, fmt.Errorf("topic_name is not editable and cannot be updated")
	}
	if modelType, exists := request.Payload.Updates["model_type"]; exists {
		if modelTypeStr, ok := modelType.(string); ok {
			if modelTypeStr != "RESET" && modelTypeStr != "DELTA" {
				return RequestStatus{}, fmt.Errorf("invalid model_type: %s. Only 'RESET' or 'DELTA' are allowed", modelTypeStr)
			}
		}
	}
	// TODO keep this configurable from cac
	if _, exists := request.Payload.Updates["number_of_partitions"]; exists {
		request.Payload.Updates["number_of_partitions"] = 24
	}
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	modelRequest := &model_requests.ModelRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "EDIT",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.ModelRequestRepo.Create(modelRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create model edit request: %w", err)
	}
	return RequestStatus{
		RequestID: modelRequest.RequestID,
		Status:    StatusPending,
		Message:   "Model edit request submitted successfully. Awaiting admin approval.",
		CreatedAt: modelRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveModelRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.ModelRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get model request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		var payload ModelRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse model request payload: %w", err)
		}
		err = s.EtcdConfig.RegisterModel(payload.Entity, payload.Model, payload.EmbeddingStoreEnabled,
			payload.EmbeddingStoreTTL, payload.ModelConfig, payload.ModelType, payload.MQID, payload.TrainingDataPath, payload.Metadata, payload.JobFrequency, payload.NumberOfPartitions, payload.FailureProducerMqId, payload.TopicName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to register model in ETCD for model: %s", payload.Model)
			return ApprovalResponse{}, fmt.Errorf("failed to register model in ETCD: %w", err)
		}
		log.Info().Msgf("Successfully registered model in ETCD for model: %s", payload.Model)
	}
	err = s.ModelRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update model request status: %w", err)
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Model request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) ApproveModelEditRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	// Get and validate the request
	req, err := s.ModelRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get model request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload for edit request
		var payload ModelRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse model edit request payload: %w", err)
		}
		// TODO for now calling Register Again, but we should call an EditModel and only Update the fields that are being updated
		err = s.EtcdConfig.RegisterModel(payload.Entity, payload.Model, payload.EmbeddingStoreEnabled,
			payload.EmbeddingStoreTTL, payload.ModelConfig, payload.ModelType, payload.MQID, payload.TrainingDataPath, payload.Metadata, payload.JobFrequency, payload.NumberOfPartitions, payload.FailureProducerMqId, payload.TopicName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to update model in ETCD for model: %s", payload.Model)
			return ApprovalResponse{}, fmt.Errorf("failed to update model in ETCD: %w", err)
		}
		log.Info().Msgf("Successfully updated model in ETCD for model: %s", payload.Model)
	}
	err = s.ModelRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update model request status: %w", err)
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Model edit request processed and ETCD updated successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) GetModels() (ModelListResponse, error) {
	modelsMap, err := s.EtcdConfig.GetEntities()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get models from ETCD")
		return ModelListResponse{}, fmt.Errorf("failed to get models from ETCD: %w", err)
	}
	return ModelListResponse{
		Models: modelsMap,
	}, nil
}

func (s *skyeConfig) GetAllModelRequests() (ModelRequestListResponse, error) {
	requests, err := s.ModelRequestRepo.GetAll()
	if err != nil {
		return ModelRequestListResponse{}, fmt.Errorf("failed to get model requests: %w", err)
	}
	return ModelRequestListResponse{
		ModelRequests: requests,
		TotalCount:    len(requests),
	}, nil
}

func (s *skyeConfig) RegisterVariant(request VariantRegisterRequest) (RequestStatus, error) {
	if request.Payload.RateLimiter.RateLimit != 0 || request.Payload.RateLimiter.BurstLimit != 0 {
		return RequestStatus{}, fmt.Errorf("rate_limiter should be empty during registration and will be configured by admin during approval")
	}
	if len(request.Payload.VectorDBConfig) > 0 {
		return RequestStatus{}, fmt.Errorf("vector_db_config should be empty during registration and will be configured by admin during approval")
	}
	if request.Payload.RTPartition != 0 {
		return RequestStatus{}, fmt.Errorf("rt_partition should be 0 during registration and will be configured by admin during approval")
	}
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	variantRequest := &variant_requests.VariantRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "CREATE",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.VariantRequestRepo.Create(variantRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create variant request: %w", err)
	}
	return RequestStatus{
		RequestID: variantRequest.RequestID,
		Status:    StatusPending,
		Message:   "Variant registration request submitted successfully. Awaiting admin approval.",
		CreatedAt: variantRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) EditVariant(request VariantEditRequest) (RequestStatus, error) {
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	variantRequest := &variant_requests.VariantRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "EDIT",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.VariantRequestRepo.Create(variantRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create variant edit request: %w", err)
	}
	return RequestStatus{
		RequestID: variantRequest.RequestID,
		Status:    StatusPending,
		Message:   "Variant edit request submitted successfully. Awaiting admin approval.",
		CreatedAt: variantRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveVariantRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.VariantRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		var payload VariantRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse variant request payload: %w", err)
		}
		// Admin must provide VectorDBConfig, RateLimiter, and RTPartition during approval
		if approval.AdminVectorDBConfig.ReadHost == "" || approval.AdminVectorDBConfig.WriteHost == "" || approval.AdminVectorDBConfig.Port == "" {
			return ApprovalResponse{}, fmt.Errorf("admin must provide vector_db_config during variant approval")
		}
		if approval.AdminRateLimiter.RateLimit == 0 || approval.AdminRateLimiter.BurstLimit == 0 {
			return ApprovalResponse{}, fmt.Errorf("admin must provide rate_limiter during variant approval")
		}
		if approval.AdminRTPartition == 0 {
			return ApprovalResponse{}, fmt.Errorf("admin must provide rt_partition during variant approval")
		}
		err = s.EtcdConfig.RegisterVariant(payload.Entity, payload.Model, payload.Variant,
			approval.AdminVectorDBConfig, payload.VectorDBType, payload.FilterConfiguration.Criteria,
			payload.Type, payload.CachingConfiguration.DistributedCachingEnabled, payload.CachingConfiguration.DistributedCacheTTLSeconds,
			payload.CachingConfiguration.InMemoryCachingEnabled, payload.CachingConfiguration.InMemoryCacheTTLSeconds,
			approval.AdminRTPartition, approval.AdminRateLimiter)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to register variant in ETCD for variant: %s", payload.Variant)
			return ApprovalResponse{}, fmt.Errorf("failed to register variant in ETCD: %w", err)
		}
	}
	err = s.VariantRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant request status: %w", err)
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Variant request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) ApproveVariantEditRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	// Get and validate the request
	req, err := s.VariantRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		var payload VariantEditRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse variant edit request payload: %w", err)
		}
		// TODO Correct this, here also create a EditVariant function and use it here
		// err = s.EtcdConfig.RegisterVariant(payload.Entity, payload.Model, payload.Variant,
		// 	approval.AdminVectorDBConfig, payload.VectorDBType, payload.Filter,
		// 	payload.Type, payload.CachingConfiguration.DistributedCachingEnabled, payload.CachingConfiguration.DistributedCacheTTLSeconds,
		// 	payload.CachingConfiguration.InMemoryCachingEnabled, payload.CachingConfiguration.InMemoryCacheTTLSeconds,
		// 	approval.AdminRTPartition, approval.AdminRateLimiter)
		// if err != nil {
		// 	log.Error().Err(err).Msgf("Failed to register variant in ETCD for variant: %s", payload.Variant)
		// 	return ApprovalResponse{}, fmt.Errorf("failed to register variant in ETCD: %w", err)
		// }
	}
	err = s.VariantRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant request status: %w", err)
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Variant edit request processed and ETCD updated successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) GetVariants(entity string, model string) (VariantListResponse, error) {
	modelConfig, err := s.EtcdConfig.GetModelConfig(entity, model)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get variants from ETCD")
		return VariantListResponse{}, fmt.Errorf("failed to get variants from ETCD: %w", err)
	}
	return VariantListResponse{
		Variants: modelConfig.Variants,
	}, nil
}

func (s *skyeConfig) GetAllVariantRequests() (VariantRequestListResponse, error) {
	requests, err := s.VariantRequestRepo.GetAll()
	if err != nil {
		return VariantRequestListResponse{}, fmt.Errorf("failed to get variant requests: %w", err)
	}
	return VariantRequestListResponse{
		VariantRequests: requests,
		TotalCount:      len(requests),
	}, nil
}

func (s *skyeConfig) RegisterFilter(request FilterRegisterRequest) (RequestStatus, error) {
	if request.Payload.Entity == "" {
		return RequestStatus{}, fmt.Errorf("entity is required")
	}
	if request.Payload.Filter.ColumnName == "" {
		return RequestStatus{}, fmt.Errorf("column_name is required")
	}
	if request.Payload.Filter.FilterValue == "" {
		return RequestStatus{}, fmt.Errorf("filter_value is required")
	}
	if request.Payload.Filter.DefaultValue == "" {
		return RequestStatus{}, fmt.Errorf("default_value is required")
	}
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	filterRequest := &filter_requests.FilterRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "CREATE",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.FilterRequestRepo.Create(filterRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create filter request: %w", err)
	}
	return RequestStatus{
		RequestID: filterRequest.RequestID,
		Status:    StatusPending,
		Message:   "Filter registration request submitted successfully. Awaiting admin approval.",
		CreatedAt: filterRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveFilterRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.FilterRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get filter request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		var payload FilterRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse filter request payload: %w", err)
		}
		err = s.EtcdConfig.RegisterFilter(payload.Entity, payload.Filter.ColumnName, payload.Filter.FilterValue, payload.Filter.DefaultValue)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create filter config in ETCD for filter_id: %s", payload.Filter.ColumnName)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD filter configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD filter configuration for filter_id: %s", payload.Filter.ColumnName)
	}

	err = s.FilterRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update filter request status: %w", err)
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Filter request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) GetFilters(entity string) (FilterListResponse, error) { // Get all filters directly from ETCD configuration
	filters, err := s.EtcdConfig.GetFilters(entity)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get filters from ETCD")
		return FilterListResponse{}, fmt.Errorf("failed to get filters from ETCD: %w", err)
	}
	return FilterListResponse{
		Filters: filters,
	}, nil
}

func (s *skyeConfig) GetAllFilterRequests() (FilterRequestListResponse, error) {
	requests, err := s.FilterRequestRepo.GetAll()
	if err != nil {
		return FilterRequestListResponse{}, fmt.Errorf("failed to get filter requests: %w", err)
	}
	return FilterRequestListResponse{
		FilterRequests: requests,
		TotalCount:     len(requests),
	}, nil
}

func (s *skyeConfig) PromoteVariant(request VariantPromotionRequest) (RequestStatus, error) {
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	promotionRequest := &variant_promotion_requests.VariantPromotionRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "PROMOTE",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.VariantPromotionRequestRepo.Create(promotionRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create variant promotion request: %w", err)
	}

	return RequestStatus{
		RequestID: promotionRequest.RequestID,
		Status:    StatusPending,
		Message:   "Variant promotion request submitted successfully. Awaiting admin approval.",
		CreatedAt: promotionRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveVariantPromotionRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.VariantPromotionRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant promotion request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	err = s.VariantPromotionRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant promotion request status: %w", err)
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Variant promotion request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) OnboardVariant(request VariantOnboardingRequest) (RequestStatus, error) {
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	onboardingRequest := &variant_onboarding_requests.VariantOnboardingRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "ONBOARD",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.VariantOnboardingRequestRepo.Create(onboardingRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create variant onboarding request: %w", err)
	}
	return RequestStatus{
		RequestID: onboardingRequest.RequestID,
		Status:    StatusPending,
		Message:   "Variant onboarding request submitted successfully. Awaiting admin approval.",
		CreatedAt: onboardingRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveVariantOnboardingRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.VariantOnboardingRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant onboarding request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	err = s.VariantOnboardingRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant onboarding request status: %w", err)
	}
	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload VariantOnboardingRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse variant onboarding request payload: %w", err)
		}
		// create task in the database which will be picked up by the variant onboarding job
		task := &variant_onboarding_tasks.VariantOnboardingTask{
			Entity:  payload.Entity,
			Model:   payload.Model,
			Variant: payload.Variant,
			Payload: "{}",
			Status:  "PENDING",
		}
		if err := s.VariantOnboardingTaskRepo.Create(task); err != nil {
			log.Error().Err(err).Msgf("Failed to create variant onboarding task for request %d", requestID)
			return ApprovalResponse{}, fmt.Errorf("failed to create variant onboarding task: %w", err)
		}
		log.Info().
			Int("request_id", requestID).
			Int("task_id", task.TaskID).
			Str("entity", payload.Entity).
			Str("model", payload.Model).
			Str("variant", payload.Variant).
			Msg("Created variant onboarding task")
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Variant onboarding request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) RegisterJobFrequency(request JobFrequencyRegisterRequest) (RequestStatus, error) {
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	jobFrequencyRequest := &job_frequency_requests.JobFrequencyRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "CREATE",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}
	err = s.JobFrequencyRequestRepo.Create(jobFrequencyRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create job frequency request: %w", err)
	}
	return RequestStatus{
		RequestID: jobFrequencyRequest.RequestID,
		Status:    StatusPending,
		Message:   "Job frequency registration request submitted successfully. Awaiting admin approval.",
		CreatedAt: jobFrequencyRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveJobFrequencyRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	req, err := s.JobFrequencyRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get job frequency request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}
	if approval.ApprovalDecision == StatusApproved {
		var payload JobFrequencyRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse job frequency request payload: %w", err)
		}
		err = s.EtcdConfig.RegisterFrequency(payload.JobFrequency)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create job frequency config in ETCD for frequency_id: %s", payload.JobFrequency)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD job frequency configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD job frequency configuration for frequency_id: %s", payload.JobFrequency)
	}
	err = s.JobFrequencyRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update job frequency request status: %w", err)
	}
	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Job frequency request processed, ETCD configured and file updated successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

func (s *skyeConfig) GetJobFrequencies() (JobFrequencyListResponse, error) {
	// Fetch job frequencies from ETCD
	frequencies, err := s.EtcdConfig.GetFrequencies()
	if err != nil {
		return JobFrequencyListResponse{}, fmt.Errorf("failed to get job frequencies from ETCD: %w", err)
	}

	return JobFrequencyListResponse{
		Frequencies: frequencies,
		TotalCount:  len(frequencies),
	}, nil
}

func (s *skyeConfig) GetAllJobFrequencyRequests() (JobFrequencyRequestListResponse, error) {
	requests, err := s.JobFrequencyRequestRepo.GetAll()
	if err != nil {
		return JobFrequencyRequestListResponse{}, fmt.Errorf("failed to get job frequency requests: %w", err)
	}
	return JobFrequencyRequestListResponse{
		JobFrequencyRequests: requests,
		TotalCount:           len(requests),
	}, nil
}

func (s *skyeConfig) GetAllVariantOnboardingRequests() (VariantOnboardingRequestListResponse, error) {
	requests, err := s.VariantOnboardingRequestRepo.GetAll()
	if err != nil {
		return VariantOnboardingRequestListResponse{}, fmt.Errorf("failed to get variant onboarding requests: %w", err)
	}
	return VariantOnboardingRequestListResponse{
		VariantOnboardingRequests: requests,
		TotalCount:                len(requests),
	}, nil
}

func (s *skyeConfig) GetVariantOnboardingTasks() (VariantOnboardingTaskListResponse, error) {
	tasks, err := s.VariantOnboardingTaskRepo.GetAll()
	if err != nil {
		return VariantOnboardingTaskListResponse{}, fmt.Errorf("failed to get variant onboarding tasks: %w", err)
	}

	return VariantOnboardingTaskListResponse{
		VariantOnboardingTasks: tasks,
		TotalCount:             len(tasks),
	}, nil
}
