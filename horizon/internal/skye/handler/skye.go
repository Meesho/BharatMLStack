package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/scylla"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/entity_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/filter_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/job_frequency_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/model_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/qdrant_cluster_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/store_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_promotion_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_requests"
	skyeEtcd "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
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
	EtcdConfig                   skyeEtcd.Manager
	ScyllaStores                 map[int]scylla.Store
	ActiveScyllaConfigIDs        map[int]bool // Set of active Scylla config IDs for validation
}

func InitV1ConfigHandler() Config {
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

		log.Info().Msg("About to initialize ETCD config...")
		etcdConfig := skyeEtcd.NewEtcdInstance()
		log.Info().Msgf("ETCD config initialized: %+v", etcdConfig)

		// Initialize Scylla stores
		log.Info().Msg("Initializing Scylla stores...")
		scyllaStores := make(map[int]scylla.Store)
		activeScyllaConfigIDs := make(map[int]bool)

		// Get active Scylla config IDs from environment
		activeScyllaConfIds := viper.GetString("SCYLLA_ACTIVE_CONFIG_IDS")
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

		log.Info().Msgf("Initialized %d Scylla stores", len(scyllaStores))

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
			EtcdConfig:                   etcdConfig,
			ScyllaStores:                 scyllaStores,
			ActiveScyllaConfigIDs:        activeScyllaConfigIDs,
		}
		log.Info().Msg("Config initialization completed")
	} else {
		log.Info().Msg("Config already exists, reusing...")
	}
	return config
}

// ==================== STORE OPERATIONS ====================

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
	validApprovalDecisions := map[string]bool{
		StatusApproved:       true,
		StatusRejected:       true,
		"NEEDS_MODIFICATION": true,
	}

	if approval.ApprovalDecision == "" {
		return ApprovalResponse{}, fmt.Errorf("approval_decision cannot be empty. Valid values: APPROVED, REJECTED, NEEDS_MODIFICATION")
	}

	if !validApprovalDecisions[approval.ApprovalDecision] {
		return ApprovalResponse{}, fmt.Errorf("invalid approval_decision: %s. Valid values: APPROVED, REJECTED, NEEDS_MODIFICATION", approval.ApprovalDecision)
	}

	// Get and validate the request
	req, err := s.StoreRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get store request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	var storeId string // For rollback tracking

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload StoreRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse store request payload: %w", err)
		}

		// Check if Scylla store exists for the config ID (if DB type is scylla)
		if strings.ToLower(payload.DB) == "scylla" {
			_, exists := s.ScyllaStores[payload.ConfID]
			if !exists {
				// Get list of available Scylla config IDs for better error message
				availableConfigIds := make([]string, 0, len(s.ScyllaStores))
				for confId := range s.ScyllaStores {
					availableConfigIds = append(availableConfigIds, fmt.Sprintf("%d", confId))
				}

				var availableIdsMsg string
				if len(availableConfigIds) > 0 {
					availableIdsMsg = fmt.Sprintf(" Available Scylla config IDs: %s. Please ensure config_id %d is in SCYLLA_ACTIVE_CONFIG_IDS and connection details are configured.", strings.Join(availableConfigIds, ", "), payload.ConfID)
				} else {
					availableIdsMsg = " No Scylla config IDs are currently active. Please configure SCYLLA_ACTIVE_CONFIG_IDS environment variable with config_id and connection details."
				}

				log.Error().Msgf("Scylla store not found for config ID: %d.%s", payload.ConfID, availableIdsMsg)
				return ApprovalResponse{}, fmt.Errorf("scylla store not found for config ID %d.%s", payload.ConfID, availableIdsMsg)
			}
			log.Info().Msgf("Scylla store validated for config ID: %d", payload.ConfID)
		}

		// Prepare store configuration
		storeConfig := skyeEtcd.StoreConfig{
			ConfID:          payload.ConfID,
			DB:              payload.DB,
			EmbeddingsTable: payload.EmbeddingsTable,
			AggregatorTable: payload.AggregatorTable,
		}

		// Generate next incremental store ID
		nextStoreId, err := s.getNextStoreId()
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate next store ID")
			return ApprovalResponse{}, fmt.Errorf("failed to generate next store ID: %w", err)
		}

		storeId = fmt.Sprintf("%d", nextStoreId)
		log.Info().Msgf("Assigned store_id: %s for request_id: %d", storeId, requestID)

		// Create ETCD configuration
		err = s.EtcdConfig.CreateStoreConfig(storeId, storeConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create store config in ETCD for store_id: %s", storeId)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD store configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD store configuration for store_id: %s (using conf_id: %d)", storeId, payload.ConfID)

		// Create Scylla tables
		if strings.ToLower(payload.DB) == "scylla" {
			log.Info().Msgf("Creating Scylla tables for store_id: %s (using Scylla config_id: %d)", storeId, payload.ConfID)
			scyllaStore := s.ScyllaStores[payload.ConfID] // Safe to access - already validated above

			// Create embeddings table
			if payload.EmbeddingsTable != "" {
				primaryKeys := []string{"entity_id", "embedding_id"} // Default primary keys for embeddings
				tableTTL := 0                                        // Default TTL (0 means no TTL)

				log.Info().Msgf("Creating Scylla embeddings table: %s", payload.EmbeddingsTable)
				err = scyllaStore.CreateTable(payload.EmbeddingsTable, primaryKeys, tableTTL)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create Scylla embeddings table: %s", payload.EmbeddingsTable)
					// Rollback ETCD config
					if rollbackErr := s.EtcdConfig.DeleteStoreConfig(storeId); rollbackErr != nil {
						log.Error().Err(rollbackErr).Msgf("Failed to rollback ETCD store config for store_id: %s", storeId)
					}
					return ApprovalResponse{}, fmt.Errorf("failed to create Scylla embeddings table %s: %w", payload.EmbeddingsTable, err)
				}
				log.Info().Msgf("Successfully created Scylla embeddings table: %s", payload.EmbeddingsTable)
			}

			// Create aggregator table
			if payload.AggregatorTable != "" {
				primaryKeys := []string{"entity_id", "timestamp"} // Default primary keys for aggregator
				tableTTL := 0                                     // Default TTL (0 means no TTL)

				log.Info().Msgf("Creating Scylla aggregator table: %s", payload.AggregatorTable)
				err = scyllaStore.CreateTable(payload.AggregatorTable, primaryKeys, tableTTL)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create Scylla aggregator table: %s", payload.AggregatorTable)
					// Rollback ETCD config
					if rollbackErr := s.EtcdConfig.DeleteStoreConfig(storeId); rollbackErr != nil {
						log.Error().Err(rollbackErr).Msgf("Failed to rollback ETCD store config for store_id: %s", storeId)
					}
					return ApprovalResponse{}, fmt.Errorf("failed to create Scylla aggregator table %s: %w", payload.AggregatorTable, err)
				}
				log.Info().Msgf("Successfully created Scylla aggregator table: %s", payload.AggregatorTable)
			}

			log.Info().Msgf("Successfully created all Scylla tables for store_id: %s (using Scylla config_id: %d)", storeId, payload.ConfID)
		} else {
			log.Info().Msgf("DB type is '%s', skipping Scylla table creation", payload.DB)
		}
	}

	// Double-check approval decision before database update
	if approval.ApprovalDecision == "" {
		log.Error().Msgf("CRITICAL: ApprovalDecision is empty before database update for request_id: %d", requestID)
		return ApprovalResponse{}, fmt.Errorf("internal error: approval decision became empty before database update")
	}

	// TRANSACTIONAL: Update database status ONLY if ETCD operations succeed
	log.Info().Msgf("Updating store request status: request_id=%d, status=%s, admin_id=%s", requestID, approval.ApprovalDecision, approval.AdminID)
	err = s.StoreRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to update store request status in database: request_id=%d, status=%s", requestID, approval.ApprovalDecision)
		// ROLLBACK: If DB update fails and we created ETCD config, delete it
		if approval.ApprovalDecision == StatusApproved && storeId != "" {
			log.Error().Err(err).Msgf("DB update failed, rolling back ETCD store config for store_id: %s", storeId)
			if rollbackErr := s.EtcdConfig.DeleteStoreConfig(storeId); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msgf("Failed to rollback ETCD store config for store_id: %s", storeId)
			} else {
				log.Info().Msgf("Successfully rolled back ETCD store config for store_id: %s", storeId)
			}
		}
		return ApprovalResponse{}, fmt.Errorf("failed to update store request status: %w", err)
	}

	log.Info().Msgf("Successfully updated store request status in database: request_id=%d, status=%s", requestID, approval.ApprovalDecision)

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was created
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Store request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

func (s *skyeConfig) GetAllStoreRequests() (StoreRequestListResponse, error) {
	requests, err := s.StoreRequestRepo.GetAll()
	if err != nil {
		return StoreRequestListResponse{}, fmt.Errorf("failed to get store requests: %w", err)
	}

	var storeRequestInfos []StoreRequestInfo
	for _, req := range requests {
		var payload StoreRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal store request payload")
			continue
		}

		storeRequestInfo := StoreRequestInfo{
			RequestID:   req.RequestID,
			Reason:      req.Reason,
			Payload:     payload,
			RequestType: req.RequestType,
			CreatedBy:   req.CreatedBy,
			ApprovedBy:  req.ApprovedBy,
			Status:      req.Status,
			CreatedAt:   req.CreatedAt,
			UpdatedAt:   req.UpdatedAt,
		}
		storeRequestInfos = append(storeRequestInfos, storeRequestInfo)
	}

	return StoreRequestListResponse{
		StoreRequests: storeRequestInfos,
		TotalCount:    len(storeRequestInfos),
	}, nil
}

func (s *skyeConfig) GetStores() (StoreListResponse, error) {
	log.Info().Msg("Fetching stores from ETCD (source of truth)")

	// Get all stores directly from ETCD configuration
	storeConfigs, err := s.EtcdConfig.GetStores()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stores from ETCD")
		return StoreListResponse{}, fmt.Errorf("failed to get stores from ETCD: %w", err)
	}

	log.Info().Msgf("Found %d stores in ETCD", len(storeConfigs))

	var stores []StoreInfo
	for storeId, storeConfig := range storeConfigs {
		// Create StoreInfo from ETCD data
		store := StoreInfo{
			ID:              storeId, // Use the actual store ID from ETCD
			ConfID:          storeConfig.ConfID,
			DB:              storeConfig.DB,
			EmbeddingsTable: storeConfig.EmbeddingsTable,
			AggregatorTable: storeConfig.AggregatorTable,
		}
		stores = append(stores, store)
	}

	log.Info().Msgf("Successfully retrieved %d stores from ETCD", len(stores))
	return StoreListResponse{
		Stores: stores,
	}, nil
}

// ==================== ENTITY OPERATIONS ====================

func (s *skyeConfig) RegisterEntity(request EntityRegisterRequest) (RequestStatus, error) {
	storeExists, err := s.EtcdConfig.StoreExists(request.Payload.StoreID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check if store_id '%s' exists", request.Payload.StoreID)
		return RequestStatus{}, fmt.Errorf("failed to validate store_id '%s': %w", request.Payload.StoreID, err)
	}

	if !storeExists {
		log.Error().Msgf("Store_id '%s' does not exist in ETCD. Cannot register entity '%s'", request.Payload.StoreID, request.Payload.Entity)
		return RequestStatus{}, fmt.Errorf("store_id '%s' does not exist in ETCD. Please ensure the store is created before registering entities", request.Payload.StoreID)
	}

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
	// Get and validate the request
	req, err := s.EntityRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get entity request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	var entityId string // For rollback tracking

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload EntityRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse entity request payload: %w", err)
		}

		// Create ETCD configuration FIRST
		entityConfig := skyeEtcd.EntityConfig{
			Entity:  payload.Entity,
			StoreID: payload.StoreID,
		}

		entityId = fmt.Sprintf("%s_%s", payload.Entity, payload.StoreID)
		err = s.EtcdConfig.CreateEntityConfig(entityId, entityConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create entity config in ETCD for entity_id: %s", entityId)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD entity configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD entity configuration for entity_id: %s", entityId)
	}

	// TRANSACTIONAL: Update database status ONLY if ETCD operations succeed
	err = s.EntityRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		// ROLLBACK: If DB update fails and we created ETCD config, delete it
		if approval.ApprovalDecision == StatusApproved && entityId != "" {
			log.Error().Err(err).Msgf("DB update failed, rolling back ETCD entity config for entity_id: %s", entityId)
			if rollbackErr := s.EtcdConfig.DeleteEntityConfig(entityId); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msgf("Failed to rollback ETCD entity config for entity_id: %s", entityId)
			} else {
				log.Info().Msgf("Successfully rolled back ETCD entity config for entity_id: %s", entityId)
			}
		}
		return ApprovalResponse{}, fmt.Errorf("failed to update entity request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was created
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Entity request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

func (s *skyeConfig) GetEntities() (EntityListResponse, error) {
	log.Info().Msg("Fetching entities from ETCD (source of truth)")

	// Get all entities directly from ETCD configuration
	entityConfigs, err := s.EtcdConfig.GetEntities()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get entities from ETCD")
		return EntityListResponse{}, fmt.Errorf("failed to get entities from ETCD: %w", err)
	}

	log.Info().Msgf("Found %d entities in ETCD", len(entityConfigs))

	var entities []EntityInfo
	for entityId, entityConfig := range entityConfigs {
		// Entity ID format: {entity}_{storeId}
		// Extract entity name and store ID from the config
		entities = append(entities, EntityInfo{
			Name:    entityConfig.Entity,
			StoreID: entityConfig.StoreID,
		})
		log.Debug().Msgf("Loaded entity from ETCD: ID=%s, Entity=%s, StoreID=%s", entityId, entityConfig.Entity, entityConfig.StoreID)
	}

	log.Info().Msgf("Successfully retrieved %d entities from ETCD", len(entities))
	return EntityListResponse{
		Entities: entities,
	}, nil
}

func (s *skyeConfig) GetAllEntityRequests() (EntityRequestListResponse, error) {
	requests, err := s.EntityRequestRepo.GetAll()
	if err != nil {
		return EntityRequestListResponse{}, fmt.Errorf("failed to get entity requests: %w", err)
	}

	var entityRequestInfos []EntityRequestInfo
	for _, request := range requests {
		var payload EntityRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		entityRequestInfos = append(entityRequestInfos, EntityRequestInfo{
			RequestID:   request.RequestID,
			Reason:      request.Reason,
			Payload:     payload,
			RequestType: request.RequestType,
			CreatedBy:   request.CreatedBy,
			ApprovedBy:  request.ApprovedBy,
			Status:      request.Status,
			CreatedAt:   request.CreatedAt,
			UpdatedAt:   request.UpdatedAt,
		})
	}

	return EntityRequestListResponse{
		EntityRequests: entityRequestInfos,
		TotalCount:     len(entityRequestInfos),
	}, nil
}

// ==================== MODEL OPERATIONS ====================

func (s *skyeConfig) RegisterModel(request ModelRegisterRequest) (RequestStatus, error) {
	entityExists, err := s.EtcdConfig.EntityExists(request.Payload.Entity)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check if entity '%s' exists", request.Payload.Entity)
		return RequestStatus{}, fmt.Errorf("failed to validate entity '%s': %w", request.Payload.Entity, err)
	}

	if !entityExists {
		log.Error().Msgf("Entity '%s' does not exist in ETCD. Cannot register model '%s'", request.Payload.Entity, request.Payload.Model)
		return RequestStatus{}, fmt.Errorf("entity '%s' does not exist in ETCD. Please ensure the entity is created and approved before registering models", request.Payload.Entity)
	}

	log.Info().Msgf("✅ Entity '%s' validated successfully. Proceeding with model registration", request.Payload.Entity)

	// Validation: model_type can only be RESET or DELTA
	if request.Payload.ModelType != "RESET" && request.Payload.ModelType != "DELTA" {
		return RequestStatus{}, fmt.Errorf("invalid model_type: %s. Only 'RESET' or 'DELTA' are allowed", request.Payload.ModelType)
	}

	// Force partitions to be 24 regardless of user input
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
	// Validation: mq_id is not editable
	if _, exists := request.Payload.Updates["mq_id"]; exists {
		return RequestStatus{}, fmt.Errorf("mq_id is not editable and cannot be updated")
	}

	// Validation: topic_name is not editable (tied to MQ configuration)
	if _, exists := request.Payload.Updates["topic_name"]; exists {
		return RequestStatus{}, fmt.Errorf("topic_name is not editable and cannot be updated")
	}

	// Validation: if model_type is being updated, ensure it's RESET or DELTA
	if modelType, exists := request.Payload.Updates["model_type"]; exists {
		if modelTypeStr, ok := modelType.(string); ok {
			if modelTypeStr != "RESET" && modelTypeStr != "DELTA" {
				return RequestStatus{}, fmt.Errorf("invalid model_type: %s. Only 'RESET' or 'DELTA' are allowed", modelTypeStr)
			}
		}
	}

	// Force partitions to be 24 if being updated
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
	// Get and validate the request
	req, err := s.ModelRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get model request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload ModelRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse model request payload: %w", err)
		}

		// Create ETCD configuration FIRST
		modelConfig := skyeEtcd.ModelConfig{
			Entity:                payload.Entity,
			Model:                 payload.Model,
			EmbeddingStoreEnabled: payload.EmbeddingStoreEnabled,
			EmbeddingStoreVersion: 1, // Default version
			EmbeddingStoreTTL:     payload.EmbeddingStoreTTL,
			ModelConfigDetails: skyeEtcd.ModelConfigDetails{
				DistanceFunction: payload.ModelConfig.DistanceFunction,
				VectorDimension:  payload.ModelConfig.VectorDimension,
			},
			ModelType:           payload.ModelType,
			MQID:                payload.MQID,
			JobFrequency:        payload.JobFrequency,
			TrainingDataPath:    payload.TrainingDataPath,
			NumberOfPartitions:  payload.NumberOfPartitions,
			TopicName:           payload.TopicName,
			Metadata:            "{}",                 // Default empty metadata
			FailureProducerMqId: 0,                    // Default failure producer MQ ID
			PartitionStatus:     make(map[string]int), // Initialize empty partition status map
		}

		modelId := fmt.Sprintf("%s_%s", payload.Entity, payload.Model)
		err = s.EtcdConfig.CreateModelConfig(modelId, modelConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create model config in ETCD for model_id: %s", modelId)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD model configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD model configuration for model_id: %s", modelId)
	}

	// Update database status ONLY if ETCD operations succeed
	err = s.ModelRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update model request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was created
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Model request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
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

		// Update ETCD configuration FIRST
		modelConfig := skyeEtcd.ModelConfig{
			Entity:                payload.Entity,
			Model:                 payload.Model,
			EmbeddingStoreEnabled: payload.EmbeddingStoreEnabled,
			EmbeddingStoreVersion: 1, // Default version (could be updated based on payload if needed)
			EmbeddingStoreTTL:     payload.EmbeddingStoreTTL,
			ModelConfigDetails: skyeEtcd.ModelConfigDetails{
				DistanceFunction: payload.ModelConfig.DistanceFunction,
				VectorDimension:  payload.ModelConfig.VectorDimension,
			},
			ModelType:           payload.ModelType,
			MQID:                payload.MQID,
			JobFrequency:        payload.JobFrequency,
			TrainingDataPath:    payload.TrainingDataPath,
			NumberOfPartitions:  payload.NumberOfPartitions,
			TopicName:           payload.TopicName,
			Metadata:            "{}",                 // Default empty metadata
			FailureProducerMqId: 0,                    // Default failure producer MQ ID
			PartitionStatus:     make(map[string]int), // Initialize empty partition status map
		}

		modelId := fmt.Sprintf("%s_%s", payload.Entity, payload.Model)
		err = s.EtcdConfig.UpdateModelConfig(modelId, modelConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to update model config in ETCD for model_id: %s", modelId)
			return ApprovalResponse{}, fmt.Errorf("failed to update ETCD model configuration: %w", err)
		}
		log.Info().Msgf("Successfully updated ETCD model configuration for model_id: %s", modelId)
	}

	// Update database status ONLY if ETCD operations succeed
	err = s.ModelRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update model request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was updated
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Model edit request processed and ETCD updated successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

func (s *skyeConfig) GetModels() (ModelListResponse, error) {
	log.Info().Msg("Fetching models from ETCD (source of truth)")

	// Get all models directly from ETCD configuration
	modelConfigs, err := s.EtcdConfig.GetModels()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get models from ETCD")
		return ModelListResponse{}, fmt.Errorf("failed to get models from ETCD: %w", err)
	}

	log.Info().Msgf("Found %d models in ETCD", len(modelConfigs))

	var models []ModelInfo
	for modelId, modelConfig := range modelConfigs {
		// Model ID format: {entity}_{model}
		// Calculate variant count from variants
		variantCount := 0
		variants, err := s.EtcdConfig.GetVariants()
		if err == nil {
			for _, variantConfig := range variants {
				if variantConfig.Entity == modelConfig.Entity && variantConfig.Model == modelConfig.Model {
					variantCount++
				}
			}
		}

		models = append(models, ModelInfo{
			Entity:                modelConfig.Entity,
			Model:                 modelConfig.Model,
			ModelType:             modelConfig.ModelType,
			EmbeddingStoreEnabled: modelConfig.EmbeddingStoreEnabled,
			EmbeddingStoreTTL:     modelConfig.EmbeddingStoreTTL,
			VectorDimension:       modelConfig.ModelConfigDetails.VectorDimension,
			DistanceFunction:      modelConfig.ModelConfigDetails.DistanceFunction,
			Status:                "active",
			VariantCount:          variantCount,
			MQID:                  modelConfig.MQID,
			TrainingDataPath:      modelConfig.TrainingDataPath,
			TopicName:             modelConfig.TopicName,
			Metadata:              modelConfig.Metadata,
			FailureProducerMqId:   modelConfig.FailureProducerMqId,
		})
		log.Debug().Msgf("Loaded model from ETCD: ID=%s, Entity=%s, Model=%s", modelId, modelConfig.Entity, modelConfig.Model)
	}

	log.Info().Msgf("Successfully retrieved %d models from ETCD", len(models))
	return ModelListResponse{
		Models: models,
	}, nil
}

func (s *skyeConfig) GetAllModelRequests() (ModelRequestListResponse, error) {
	requests, err := s.ModelRequestRepo.GetAll()
	if err != nil {
		return ModelRequestListResponse{}, fmt.Errorf("failed to get model requests: %w", err)
	}

	var modelRequestInfos []ModelRequestInfo
	for _, request := range requests {
		var payload ModelRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		modelRequestInfos = append(modelRequestInfos, ModelRequestInfo{
			RequestID:   request.RequestID,
			Reason:      request.Reason,
			Payload:     payload,
			RequestType: request.RequestType,
			CreatedBy:   request.CreatedBy,
			ApprovedBy:  request.ApprovedBy,
			Status:      request.Status,
			CreatedAt:   request.CreatedAt,
			UpdatedAt:   request.UpdatedAt,
		})
	}

	return ModelRequestListResponse{
		ModelRequests: modelRequestInfos,
		TotalCount:    len(modelRequestInfos),
	}, nil
}

// ==================== VARIANT OPERATIONS ====================

func (s *skyeConfig) RegisterVariant(request VariantRegisterRequest) (RequestStatus, error) {
	entityExists, err := s.EtcdConfig.EntityExists(request.Payload.Entity)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check if entity '%s' exists", request.Payload.Entity)
		return RequestStatus{}, fmt.Errorf("failed to validate entity '%s': %w", request.Payload.Entity, err)
	}

	if !entityExists {
		log.Error().Msgf("Entity '%s' does not exist in ETCD. Cannot register variant '%s'", request.Payload.Entity, request.Payload.Variant)
		return RequestStatus{}, fmt.Errorf("entity '%s' does not exist in ETCD. Please ensure the entity is created and approved before registering variants", request.Payload.Entity)
	}

	log.Info().Msgf("✅ Entity '%s' validated successfully", request.Payload.Entity)

	modelExists, err := s.EtcdConfig.ModelExists(request.Payload.Entity, request.Payload.Model)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check if model '%s' exists in entity '%s'", request.Payload.Model, request.Payload.Entity)
		return RequestStatus{}, fmt.Errorf("failed to validate model '%s' in entity '%s': %w", request.Payload.Model, request.Payload.Entity, err)
	}

	if !modelExists {
		log.Error().Msgf("Model '%s' does not exist in entity '%s'. Cannot register variant '%s'", request.Payload.Model, request.Payload.Entity, request.Payload.Variant)
		return RequestStatus{}, fmt.Errorf("model '%s' does not exist in entity '%s'. Please ensure the model is created and approved before registering variants", request.Payload.Model, request.Payload.Entity)
	}

	log.Info().Msgf("✅ Model '%s' in entity '%s' validated successfully. Proceeding with variant registration", request.Payload.Model, request.Payload.Entity)

	// VALIDATION: Check if all filters referenced in FilterConfiguration exist in ETCD
	if len(request.Payload.FilterConfiguration.Criteria) > 0 {
		log.Info().Msgf("Validating %d filter criteria for variant registration in entity '%s'", len(request.Payload.FilterConfiguration.Criteria), request.Payload.Entity)

		for i, filterCriteria := range request.Payload.FilterConfiguration.Criteria {
			if filterCriteria.ColumnName == "" {
				log.Error().Msgf("Filter criteria %d is missing column_name field", i+1)
				return RequestStatus{}, fmt.Errorf("filter criteria %d is missing column_name field", i+1)
			}

			log.Info().Msgf("Validating filter criteria %d: column_name='%s' for entity '%s'", i+1, filterCriteria.ColumnName, request.Payload.Entity)

			filterExists, err := s.EtcdConfig.FilterExistsByColumnName(request.Payload.Entity, filterCriteria.ColumnName)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check if filter with column_name '%s' exists in entity '%s'", filterCriteria.ColumnName, request.Payload.Entity)
				return RequestStatus{}, fmt.Errorf("failed to validate filter with column_name '%s' in entity '%s': %w", filterCriteria.ColumnName, request.Payload.Entity, err)
			}

			if !filterExists {
				log.Error().Msgf("Filter with column_name '%s' does not exist in entity '%s'. Cannot register variant '%s'", filterCriteria.ColumnName, request.Payload.Entity, request.Payload.Variant)
				return RequestStatus{}, fmt.Errorf("filter with column_name '%s' does not exist in entity '%s'. Please ensure the filter is created and approved before registering variants that reference it", filterCriteria.ColumnName, request.Payload.Entity)
			}

			log.Info().Msgf("✅ Filter criteria %d validated successfully: column_name='%s' exists in entity '%s'", i+1, filterCriteria.ColumnName, request.Payload.Entity)
		}

		log.Info().Msgf("✅ All %d filter criteria validated successfully for variant registration", len(request.Payload.FilterConfiguration.Criteria))
	} else {
		log.Info().Msgf("No filter criteria specified in variant registration - skipping filter validation")
	}

	// Validation: RateLimiter should be empty during registration (set by admin during approval)
	if request.Payload.RateLimiter.RateLimit != 0 || request.Payload.RateLimiter.BurstLimit != 0 {
		return RequestStatus{}, fmt.Errorf("rate_limiter should be empty during registration and will be configured by admin during approval")
	}

	// Validation: VectorDBConfig should be empty during registration (set by admin during approval)
	if len(request.Payload.VectorDBConfig) > 0 {
		return RequestStatus{}, fmt.Errorf("vector_db_config should be empty during registration and will be configured by admin during approval")
	}

	// Validation: RTPartition should be 0 during registration (set by admin during approval)
	if request.Payload.RTPartition != 0 {
		return RequestStatus{}, fmt.Errorf("rt_partition should be 0 during registration and will be configured by admin during approval")
	}

	// Force fixed values regardless of user input
	request.Payload.VectorDBType = "QDRANT"
	request.Payload.Type = "EXPERIMENT"

	// Clear VectorDBConfig - should be empty during registration (filled by admin during approval)
	request.Payload.VectorDBConfig = make(VectorDBConfig)

	// Clear RateLimiter - should be empty during registration (filled by admin during approval)
	request.Payload.RateLimiter = RateLimiter{}

	// Clear RTPartition - should be 0 during registration (filled by admin during approval)
	request.Payload.RTPartition = 0

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
	// Validation: VectorDBConfig is not editable by users (only by admin during approval)
	if len(request.Payload.VectorDBConfig) > 0 {
		return RequestStatus{}, fmt.Errorf("vector_db_config is not editable and can only be configured by admin during approval")
	}

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
	// Get and validate the request
	req, err := s.VariantRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload VariantRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse variant request payload: %w", err)
		}

		// Admin must provide VectorDBConfig, RateLimiter, and RTPartition during approval
		if approval.AdminVectorDBConfig == nil {
			return ApprovalResponse{}, fmt.Errorf("admin must provide vector_db_config during variant approval")
		}
		if approval.AdminRateLimiter == nil {
			return ApprovalResponse{}, fmt.Errorf("admin must provide rate_limiter during variant approval")
		}
		if approval.AdminRTPartition == nil {
			return ApprovalResponse{}, fmt.Errorf("admin must provide rt_partition during variant approval")
		}

		// Create ETCD configuration FIRST
		variantConfig := skyeEtcd.VariantConfig{
			Entity:                     payload.Entity,
			Model:                      payload.Model,
			Variant:                    payload.Variant,
			VariantState:               "COMPLETED", // Default state
			VectorDBType:               payload.VectorDBType,
			Type:                       payload.Type,
			Enabled:                    true,  // Default enabled
			Onboarded:                  false, // Default not onboarded
			PartialHitEnabled:          true,  // Default enabled
			RTDeltaProcessing:          true,  // Default enabled
			EmbeddingStoreReadVersion:  1,     // Default version
			EmbeddingStoreWriteVersion: 1,     // Default version
			VectorDBReadVersion:        1,     // Default version
			VectorDBWriteVersion:       1,     // Default version
			CachingConfiguration: skyeEtcd.CachingConfiguration{
				InMemoryCachingEnabled:     payload.CachingConfiguration.InMemoryCachingEnabled,
				InMemoryCacheTTLSeconds:    payload.CachingConfiguration.InMemoryCacheTTLSeconds,
				DistributedCachingEnabled:  payload.CachingConfiguration.DistributedCachingEnabled,
				DistributedCacheTTLSeconds: payload.CachingConfiguration.DistributedCacheTTLSeconds,
			},
			FilterConfiguration: skyeEtcd.FilterConfiguration{
				Criteria: convertFilterCriteria(payload.FilterConfiguration.Criteria),
			},
			// Use admin-provided VectorDBConfig (any JSON structure)
			VectorDBConfig: skyeEtcd.VectorDBConfig(*approval.AdminVectorDBConfig),
			// Use admin-provided RateLimiter
			RateLimiter: skyeEtcd.RateLimiter{
				RateLimit:  approval.AdminRateLimiter.RateLimit,
				BurstLimit: approval.AdminRateLimiter.BurstLimit,
			},
			// Use admin-provided RTPartition
			RTPartition: *approval.AdminRTPartition,
		}

		variantId := fmt.Sprintf("%s_%s_%s", payload.Entity, payload.Model, payload.Variant)
		err = s.EtcdConfig.CreateVariantConfig(variantId, variantConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create variant config in ETCD for variant_id: %s", variantId)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD variant configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD variant configuration for variant_id: %s", variantId)
	}

	// Update database status ONLY if ETCD operations succeed
	err = s.VariantRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was created
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Variant request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
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
		// Parse the payload for edit request
		var payload VariantEditRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse variant edit request payload: %w", err)
		}

		// Update ETCD configuration FIRST
		variantConfig := skyeEtcd.VariantConfig{
			Entity:  payload.Entity,
			Model:   payload.Model,
			Variant: payload.Variant,
			CachingConfiguration: skyeEtcd.CachingConfiguration{
				InMemoryCachingEnabled:     payload.InMemoryCachingEnabled,
				InMemoryCacheTTLSeconds:    payload.InMemoryCacheTTLSeconds,
				DistributedCachingEnabled:  payload.DistributedCachingEnabled,
				DistributedCacheTTLSeconds: payload.DistributedCacheTTLSeconds,
			},
			FilterConfiguration: skyeEtcd.FilterConfiguration{
				Criteria: convertFilterCriteria(payload.Filter.Criteria),
			},
			RateLimiter: skyeEtcd.RateLimiter{
				RateLimit:  payload.RateLimiter.RateLimit,
				BurstLimit: payload.RateLimiter.BurstLimit,
			},
			RTPartition: payload.RTPartition,
		}

		if len(payload.VectorDBConfig) > 0 {
			variantConfig.VectorDBConfig = skyeEtcd.VectorDBConfig(payload.VectorDBConfig)
		}

		variantId := fmt.Sprintf("%s_%s_%s", payload.Entity, payload.Model, payload.Variant)
		err = s.EtcdConfig.UpdateVariantConfig(variantId, variantConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to update variant config in ETCD for variant_id: %s", variantId)
			return ApprovalResponse{}, fmt.Errorf("failed to update ETCD variant configuration: %w", err)
		}
		log.Info().Msgf("Successfully updated ETCD variant configuration for variant_id: %s", variantId)
	}

	// Update database status ONLY if ETCD operations succeed
	err = s.VariantRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was updated
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Variant edit request processed and ETCD updated successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

// Helper function to safely extract cluster host from flexible VectorDBConfig
func getClusterHostFromConfig(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	// Try common host key names
	hostKeys := []string{"read_host", "cluster_host", "host", "endpoint", "url"}
	for _, key := range hostKeys {
		if value, exists := config[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}

	return "" // Default empty string if no host found
}

// Helper functions for type conversions
func convertFilterCriteria(criteria []FilterCriteria) []skyeEtcd.FilterCriteria {
	result := make([]skyeEtcd.FilterCriteria, len(criteria))
	for i, c := range criteria {
		result[i] = skyeEtcd.FilterCriteria{
			ColumnName:   c.ColumnName,
			FilterValue:  c.FilterValue,
			DefaultValue: c.DefaultValue,
		}
	}
	return result
}

func (s *skyeConfig) GetVariants() (VariantListResponse, error) {
	log.Info().Msg("Fetching variants from ETCD (source of truth)")

	// Get all variants directly from ETCD configuration
	variantConfigs, err := s.EtcdConfig.GetVariants()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get variants from ETCD")
		return VariantListResponse{}, fmt.Errorf("failed to get variants from ETCD: %w", err)
	}

	log.Info().Msgf("Found %d variants in ETCD", len(variantConfigs))

	var variants []VariantInfo
	for variantId, variantConfig := range variantConfigs {
		// Variant ID format: {entity}_{model}_{variant}
		// Note(TBD): CreatedAt and LastUpdated are not stored in ETCD config,
		// so we use zero time or current time as fallback
		variants = append(variants, VariantInfo{
			Entity:                    variantConfig.Entity,
			Model:                     variantConfig.Model,
			Variant:                   variantConfig.Variant,
			Type:                      variantConfig.Type,
			VectorDBType:              variantConfig.VectorDBType,
			Status:                    "active",
			CreatedAt:                 time.Time{}, // Not stored in ETCD config
			LastUpdated:               time.Time{}, // Not stored in ETCD config
			ClusterHost:               getClusterHostFromConfig(map[string]interface{}(variantConfig.VectorDBConfig)),
			InMemoryCachingEnabled:    variantConfig.CachingConfiguration.InMemoryCachingEnabled,
			DistributedCachingEnabled: variantConfig.CachingConfiguration.DistributedCachingEnabled,
		})
		log.Debug().Msgf("Loaded variant from ETCD: ID=%s, Entity=%s, Model=%s, Variant=%s", variantId, variantConfig.Entity, variantConfig.Model, variantConfig.Variant)
	}

	log.Info().Msgf("Successfully retrieved %d variants from ETCD", len(variants))
	return VariantListResponse{
		Variants: variants,
	}, nil
}

func (s *skyeConfig) GetAllVariantRequests() (VariantRequestListResponse, error) {
	requests, err := s.VariantRequestRepo.GetAll()
	if err != nil {
		return VariantRequestListResponse{}, fmt.Errorf("failed to get variant requests: %w", err)
	}

	var variantRequestInfos []VariantRequestInfo
	for _, request := range requests {
		var payload VariantRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		variantRequestInfos = append(variantRequestInfos, VariantRequestInfo{
			RequestID:   request.RequestID,
			Reason:      request.Reason,
			Payload:     payload,
			RequestType: request.RequestType,
			CreatedBy:   request.CreatedBy,
			ApprovedBy:  request.ApprovedBy,
			Status:      request.Status,
			CreatedAt:   request.CreatedAt,
			UpdatedAt:   request.UpdatedAt,
		})
	}

	return VariantRequestListResponse{
		VariantRequests: variantRequestInfos,
		TotalCount:      len(variantRequestInfos),
	}, nil
}

// ==================== FILTER OPERATIONS ====================

func (s *skyeConfig) RegisterFilter(request FilterRegisterRequest) (RequestStatus, error) {
	entityExists, err := s.EtcdConfig.EntityExists(request.Payload.Entity)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check if entity '%s' exists", request.Payload.Entity)
		return RequestStatus{}, fmt.Errorf("failed to validate entity '%s': %w", request.Payload.Entity, err)
	}

	if !entityExists {
		log.Error().Msgf("Entity '%s' does not exist in ETCD. Cannot register filter", request.Payload.Entity)
		return RequestStatus{}, fmt.Errorf("entity '%s' does not exist in ETCD. Please ensure the entity is created and approved before registering filters", request.Payload.Entity)
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
	// Get and validate the request
	req, err := s.FilterRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get filter request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	var filterId string // For rollback tracking

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload FilterRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse filter request payload: %w", err)
		}

		if payload.Filter.ColumnName == "" {
			return ApprovalResponse{}, fmt.Errorf("filter is missing column_name field")
		}

		// Create ETCD configuration FIRST
		filterConfig := skyeEtcd.FilterConfig{
			Entity:       payload.Entity, // Use entity from payload
			ColumnName:   payload.Filter.ColumnName,
			FilterValue:  payload.Filter.FilterValue,
			DefaultValue: payload.Filter.DefaultValue,
		}

		filterId = fmt.Sprintf("%s_%s", payload.Filter.ColumnName, payload.Filter.FilterValue)
		err = s.EtcdConfig.CreateFilterConfig(filterId, filterConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create filter config in ETCD for filter_id: %s", filterId)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD filter configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD filter configuration for filter_id: %s", filterId)
	}

	// TRANSACTIONAL: Update database status ONLY if ETCD operations succeed
	err = s.FilterRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		// Note: ETCD filter rollback not implemented for now
		if approval.ApprovalDecision == StatusApproved && filterId != "" {
			log.Error().Err(err).Msgf("DB update failed, ETCD filter config for filter_id: %s will remain (rollback not implemented)", filterId)
		}
		return ApprovalResponse{}, fmt.Errorf("failed to update filter request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD config was created
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Filter request processed and ETCD configured successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

func (s *skyeConfig) GetFilters(options FilterQueryOptions) (FilterListResponse, error) {
	log.Info().Msg("Fetching filters from ETCD (source of truth)")

	// Get all filters directly from ETCD configuration
	filterConfigs, err := s.EtcdConfig.GetFilters()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get filters from ETCD")
		return FilterListResponse{}, fmt.Errorf("failed to get filters from ETCD: %w", err)
	}

	log.Info().Msgf("Found %d filters in ETCD", len(filterConfigs))

	// Group filters by entity
	entityFiltersMap := make(map[string][]FilterCriteriaResponse)

	for filterId, filterConfig := range filterConfigs {
		// Apply filters based on query options
		if options.Entity != "" && filterConfig.Entity != options.Entity {
			continue
		}
		if options.ColumnName != "" && filterConfig.ColumnName != options.ColumnName {
			continue
		}
		if options.DataType != "" {
			// Simple data type inference
			dataType := "string"
			if filterConfig.FilterValue == "true" || filterConfig.FilterValue == "false" {
				dataType = "boolean"
			}
			if dataType != options.DataType {
				continue
			}
		}
		// Note: active_only is always true since we only fetch from ETCD (which contains active configs)

		// Create filter criteria response
		filterCriteria := FilterCriteriaResponse{
			ColumnName:   filterConfig.ColumnName,
			FilterValue:  filterConfig.FilterValue,
			DefaultValue: filterConfig.DefaultValue,
		}

		// Group by entity
		entityFiltersMap[filterConfig.Entity] = append(entityFiltersMap[filterConfig.Entity], filterCriteria)
		log.Debug().Msgf("Loaded filter from ETCD: ID=%s, Entity=%s, ColumnName=%s", filterId, filterConfig.Entity, filterConfig.ColumnName)
	}

	// Convert map to slice for response
	var filterGroups []EntityFilterGroup
	for entity, filters := range entityFiltersMap {
		filterGroups = append(filterGroups, EntityFilterGroup{
			Entity:  entity,
			Filters: filters,
		})
	}

	log.Info().Msgf("Successfully retrieved %d filter groups from ETCD", len(filterGroups))
	return FilterListResponse{
		FilterGroups: filterGroups,
	}, nil
}

func (s *skyeConfig) GetAllFilterRequests() (FilterRequestListResponse, error) {
	requests, err := s.FilterRequestRepo.GetAll()
	if err != nil {
		return FilterRequestListResponse{}, fmt.Errorf("failed to get filter requests: %w", err)
	}

	var filterRequestInfos []FilterRequestInfo
	for _, request := range requests {
		var payload FilterRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		filterRequestInfos = append(filterRequestInfos, FilterRequestInfo{
			RequestID:   request.RequestID,
			Reason:      request.Reason,
			Payload:     payload,
			RequestType: request.RequestType,
			CreatedBy:   request.CreatedBy,
			ApprovedBy:  request.ApprovedBy,
			Status:      request.Status,
			CreatedAt:   request.CreatedAt,
			UpdatedAt:   request.UpdatedAt,
		})
	}

	return FilterRequestListResponse{
		FilterRequests: filterRequestInfos,
		TotalCount:     len(filterRequestInfos),
	}, nil
}

// ==================== DEPLOYMENT OPERATIONS ====================

func (s *skyeConfig) CreateQdrantCluster(request QdrantClusterRequest) (RequestStatus, error) {
	payloadJSON, err := json.Marshal(request.Payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	clusterRequest := &qdrant_cluster_requests.QdrantClusterRequest{
		Reason:      request.Reason,
		Payload:     string(payloadJSON),
		RequestType: "CREATE",
		CreatedBy:   request.Requestor,
		Status:      StatusPending,
	}

	err = s.QdrantClusterRequestRepo.Create(clusterRequest)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create cluster request: %w", err)
	}

	return RequestStatus{
		RequestID: clusterRequest.RequestID,
		Status:    StatusPending,
		Message:   "Cluster creation request submitted successfully. Awaiting admin approval.",
		CreatedAt: clusterRequest.CreatedAt,
	}, nil
}

func (s *skyeConfig) ApproveQdrantClusterRequest(requestID int, approval ApprovalRequest) (ApprovalResponse, error) {
	// Get and validate the request
	req, err := s.QdrantClusterRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get cluster request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	// Update database status
	err = s.QdrantClusterRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update cluster request status: %w", err)
	}

	processingStatus := StatusInProgress
	message := "Qdrant cluster request processed successfully."

	if approval.ApprovalDecision == StatusApproved {
		message = "Qdrant cluster request approved successfully. Cluster creation will be handled by the backend system."
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          message,
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

func (s *skyeConfig) GetQdrantClusters() (ClusterListResponse, error) {
	// Fetch APPROVED Qdrant cluster requests from database
	approvedRequests, err := s.QdrantClusterRequestRepo.GetByStatus("APPROVED")
	if err != nil {
		return ClusterListResponse{}, fmt.Errorf("failed to get approved Qdrant clusters: %w", err)
	}

	var clusters []ClusterInfo
	totalInstances := 0
	totalStorageGB := 0
	environmentsMap := make(map[string]bool)
	projectsMap := make(map[string]bool)

	for _, request := range approvedRequests {
		var payload QdrantClusterRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		// Generate cluster ID from DNS subdomain
		clusterID := fmt.Sprintf("qdrant-%s", payload.DNSSubdomain)

		// Determine environment from DNS subdomain (basic heuristic)
		environment := "production"
		if strings.Contains(payload.DNSSubdomain, "dev") || strings.Contains(payload.DNSSubdomain, "staging") {
			if strings.Contains(payload.DNSSubdomain, "dev") {
				environment = "development"
			} else {
				environment = "staging"
			}
		}

		// Create monitoring URLs
		monitoring := MonitoringInfo{
			GrafanaDashboardURL:   fmt.Sprintf("https://grafana.meesho.int/d/qdrant-cluster/qdrant-cluster-monitoring?var-cluster=%s", payload.DNSSubdomain),
			ClusterHealthEndpoint: fmt.Sprintf("http://%s.prd.meesho.int:6333/cluster", payload.DNSSubdomain),
			HealthStatus:          "healthy", // Would be determined by actual health check
		}

		// For now, create mock instance details (would be populated from actual infrastructure)
		instanceDetails := make([]InstanceDetails, payload.NodeConf.Count)
		for i := 0; i < payload.NodeConf.Count; i++ {
			instanceDetails[i] = InstanceDetails{
				InstanceID:       fmt.Sprintf("i-%016x", request.RequestID*1000+i), // Mock instance ID
				PrivateIP:        fmt.Sprintf("10.0.%d.%d", (request.RequestID%10)+1, 100+i),
				PublicIP:         fmt.Sprintf("54.123.%d.%d", (request.RequestID % 256), 100+i),
				AvailabilityZone: fmt.Sprintf("us-east-1%c", 'a'+rune(i%3)),
				Status:           "running",
			}
		}

		clusters = append(clusters, ClusterInfo{
			ClusterID:         clusterID,
			DNSSubdomain:      payload.DNSSubdomain,
			Project:           payload.Project,
			Environment:       environment,
			Status:            "active", // Would be determined from actual cluster status
			NodeConfiguration: payload.NodeConf,
			QdrantVersion:     payload.QdrantVersion,
			InstanceDetails:   instanceDetails,
			Monitoring:        monitoring,
			CreatedAt:         request.CreatedAt,
			LastUpdated:       request.UpdatedAt,
		})

		// Aggregate metadata
		totalInstances += payload.NodeConf.Count
		if storageNum := extractStorageGB(payload.NodeConf.Storage); storageNum > 0 {
			totalStorageGB += storageNum * payload.NodeConf.Count
		}
		environmentsMap[environment] = true
		projectsMap[payload.Project] = true
	}

	// Convert maps to slices
	var environments []string
	for env := range environmentsMap {
		environments = append(environments, env)
	}
	var projects []string
	for proj := range projectsMap {
		projects = append(projects, proj)
	}

	// Format total storage
	totalStorage := fmt.Sprintf("%dGB", totalStorageGB)
	if totalStorageGB >= 1024 {
		totalStorage = fmt.Sprintf("%.1fTB", float64(totalStorageGB)/1024)
	}

	return ClusterListResponse{
		Clusters:      clusters,
		TotalClusters: len(clusters),
		Metadata: ClusterMetadata{
			TotalInstances: totalInstances,
			TotalStorage:   totalStorage,
			Environments:   environments,
			Projects:       projects,
		},
	}, nil
}

// Helper function to extract storage number from storage string (e.g., "2TB" -> 2048)
func extractStorageGB(storage string) int {
	storage = strings.ToUpper(strings.TrimSpace(storage))

	// Remove non-numeric characters except TB/GB
	var numStr string
	var unit string
	for i, r := range storage {
		if r >= '0' && r <= '9' || r == '.' {
			numStr += string(r)
		} else {
			unit = storage[i:]
			break
		}
	}

	if numStr == "" {
		return 0
	}

	// Parse the number
	num := 0
	if n, err := fmt.Sscanf(numStr, "%d", &num); err != nil || n != 1 {
		if f, err := fmt.Sscanf(numStr, "%f", &num); err != nil || f != 1 {
			return 0
		}
	}

	// Convert to GB
	if strings.Contains(unit, "TB") {
		return num * 1024
	}
	return num // Assume GB if no unit or GB unit
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
	// Get and validate the request
	req, err := s.VariantPromotionRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant promotion request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	// Update database status
	err = s.VariantPromotionRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant promotion request status: %w", err)
	}

	processingStatus := StatusInProgress
	message := "Variant promotion request processed successfully."

	if approval.ApprovalDecision == StatusApproved {
		message = "Variant promotion request approved successfully. Promotion will be handled by the backend system."
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          message,
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
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
	// Get and validate the request
	req, err := s.VariantOnboardingRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get variant onboarding request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	// Update database status first
	err = s.VariantOnboardingRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update variant onboarding request status: %w", err)
	}

	processingStatus := StatusInProgress
	message := "Variant onboarding request processed successfully."

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload VariantOnboardingRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse variant onboarding request payload: %w", err)
		}

		// TODO: Implement Airflow client integration
		// Trigger Airflow DAG for variant onboarding
		// airflowClient := externalcall.GetAirflowClient()
		// dagRunID := fmt.Sprintf("manual_run_%s_%s_%s_%d", payload.Entity, payload.Model, payload.Variant, time.Now().Unix())
		//
		// airflowResponse, err := airflowClient.TriggerDAG(dagRunID)
		// if err != nil {
		// 	log.Error().Err(err).Msgf("Failed to trigger Airflow DAG for variant onboarding request %d", requestID)
		// 	// Don't fail the approval, just log the error and continue
		// 	message = "Variant onboarding request approved but Airflow job trigger failed. Please check Airflow manually."
		// } else if airflowResponse.Status == "error" {
		// 	log.Error().Str("error", airflowResponse.Error).Msgf("Airflow DAG trigger returned error for request %d", requestID)
		// 	message = "Variant onboarding request approved but Airflow job trigger failed. Please check Airflow manually."
		// } else {
		// 	log.Info().Str("dag_run_id", dagRunID).Msgf("Successfully triggered Airflow DAG for variant onboarding request %d", requestID)
		// 	message = "Variant onboarding request approved and Airflow job triggered successfully. Task is in progress."
		// }
		message = "Variant onboarding request approved successfully. Airflow integration pending implementation."
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          message,
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

// ==================== JOB FREQUENCY OPERATIONS ====================

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
	// Get and validate the request
	req, err := s.JobFrequencyRequestRepo.GetByID(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get job frequency request: %w", err)
	}
	if req.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	var frequencyId string // For rollback tracking

	if approval.ApprovalDecision == StatusApproved {
		// Parse the payload
		var payload JobFrequencyRequestPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to parse job frequency request payload: %w", err)
		}

		// Create ETCD configuration FIRST
		jobFrequencyConfig := skyeEtcd.JobFrequencyConfig{
			JobFrequency: payload.JobFrequency,
			Description:  generateJobFrequencyDescription(payload.JobFrequency),
		}

		frequencyId = payload.JobFrequency
		err = s.EtcdConfig.CreateJobFrequencyConfig(frequencyId, jobFrequencyConfig)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create job frequency config in ETCD for frequency_id: %s", frequencyId)
			return ApprovalResponse{}, fmt.Errorf("failed to create ETCD job frequency configuration: %w", err)
		}
		log.Info().Msgf("Successfully created ETCD job frequency configuration for frequency_id: %s", frequencyId)

		// Append to file storage
		err = appendJobFrequencyToFile(payload.JobFrequency)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to append job frequency to file: %s", payload.JobFrequency)
			return ApprovalResponse{}, fmt.Errorf("failed to append job frequency to file: %w", err)
		}
		log.Info().Msgf("Successfully appended job frequency to file: %s", payload.JobFrequency)
	}

	// TRANSACTIONAL: Update database status ONLY if ETCD and file operations succeed
	err = s.JobFrequencyRequestRepo.UpdateStatus(requestID, approval.ApprovalDecision, approval.AdminID)
	if err != nil {
		// ROLLBACK: If DB update fails and we created ETCD config, delete it
		if approval.ApprovalDecision == StatusApproved && frequencyId != "" {
			log.Error().Err(err).Msgf("DB update failed, rolling back ETCD job frequency config for frequency_id: %s", frequencyId)
			if rollbackErr := s.EtcdConfig.DeleteJobFrequencyConfig(frequencyId); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msgf("Failed to rollback ETCD job frequency config for frequency_id: %s", frequencyId)
			} else {
				log.Info().Msgf("Successfully rolled back ETCD job frequency config for frequency_id: %s", frequencyId)
			}
			// Note: File rollback is complex; consider it a known limitation for now
			log.Warn().Msgf("File rollback for frequency '%s' not implemented - manual cleanup may be required", frequencyId)
		}
		return ApprovalResponse{}, fmt.Errorf("failed to update job frequency request status: %w", err)
	}

	processingStatus := StatusInProgress
	if approval.ApprovalDecision == StatusApproved {
		processingStatus = StatusCompleted // Actually completed since ETCD and file were updated
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          "Job frequency request processed, ETCD configured and file updated successfully.",
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: processingStatus,
	}, nil
}

func (s *skyeConfig) GetJobFrequencies() (JobFrequencyListResponse, error) {
	// Fetch APPROVED job frequency requests from database
	approvedRequests, err := s.JobFrequencyRequestRepo.GetByStatus("APPROVED")
	if err != nil {
		return JobFrequencyListResponse{}, fmt.Errorf("failed to get approved job frequencies: %w", err)
	}

	var jobFrequencies []JobFrequencyInfo
	for _, request := range approvedRequests {
		var payload JobFrequencyRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		jobFrequencies = append(jobFrequencies, JobFrequencyInfo{
			ID:          fmt.Sprintf("freq_%d", request.RequestID),
			Frequency:   payload.JobFrequency,
			Description: generateJobFrequencyDescription(payload.JobFrequency),
			IsActive:    true,
			CreatedAt:   request.CreatedAt,
		})
	}

	return JobFrequencyListResponse{
		JobFrequencies: jobFrequencies,
		TotalCount:     len(jobFrequencies),
	}, nil
}

func (s *skyeConfig) GetAllJobFrequencyRequests() (JobFrequencyRequestListResponse, error) {
	requests, err := s.JobFrequencyRequestRepo.GetAll()
	if err != nil {
		return JobFrequencyRequestListResponse{}, fmt.Errorf("failed to get job frequency requests: %w", err)
	}

	var jobFrequencyRequestInfos []JobFrequencyRequestInfo
	for _, request := range requests {
		var payload JobFrequencyRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		jobFrequencyRequestInfos = append(jobFrequencyRequestInfos, JobFrequencyRequestInfo{
			RequestID:   request.RequestID,
			Reason:      request.Reason,
			Payload:     payload,
			RequestType: request.RequestType,
			CreatedBy:   request.CreatedBy,
			ApprovedBy:  request.ApprovedBy,
			Status:      request.Status,
			CreatedAt:   request.CreatedAt,
			UpdatedAt:   request.UpdatedAt,
		})
	}

	return JobFrequencyRequestListResponse{
		JobFrequencyRequests: jobFrequencyRequestInfos,
		TotalCount:           len(jobFrequencyRequestInfos),
	}, nil
}

// ==================== JOB FREQUENCY HELPER FUNCTIONS ====================

func generateJobFrequencyDescription(frequency string) string {
	descriptions := map[string]string{
		"FREQ_1D":  "Daily job frequency - runs once per day",
		"FREQ_2D":  "Bi-daily job frequency - runs every 2 days",
		"FREQ_3D":  "Tri-daily job frequency - runs every 3 days",
		"FREQ_7D":  "Weekly job frequency - runs every 7 days",
		"FREQ_1W":  "Weekly job frequency - runs once per week",
		"FREQ_2W":  "Bi-weekly job frequency - runs every 2 weeks",
		"FREQ_1M":  "Monthly job frequency - runs once per month",
		"FREQ_3M":  "Quarterly job frequency - runs every 3 months",
		"FREQ_6M":  "Semi-annual job frequency - runs every 6 months",
		"FREQ_1Y":  "Annual job frequency - runs once per year",
		"FREQ_1H":  "Hourly job frequency - runs once per hour",
		"FREQ_6H":  "Every 6 hours job frequency",
		"FREQ_12H": "Twice daily job frequency - runs every 12 hours",
	}

	if desc, exists := descriptions[frequency]; exists {
		return desc
	}
	return fmt.Sprintf("Custom job frequency: %s", frequency)
}

func appendJobFrequencyToFile(frequency string) error {
	// Create directory if it doesn't exist
	appName := viper.GetString("SKYE_APP_NAME")
	if appName == "" {
		return fmt.Errorf("SKYE_APP_NAME is not configured")
	}

	// Create directory if it doesn't exist
	storageDir := fmt.Sprintf("%s/storage", appName)
	err := createDirectoryIfNotExists(storageDir)
	if err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	filePath := fmt.Sprintf("%s/storage/frequencies", appName)

	// Check if file exists and read current contents
	var currentFrequencies []string
	if fileExists(filePath) {
		content, err := readFileContents(filePath)
		if err != nil {
			return fmt.Errorf("failed to read existing frequencies file: %w", err)
		}
		if content != "" {
			currentFrequencies = strings.Split(strings.TrimSpace(content), ",")
		}
	}

	// Check if frequency already exists (avoid duplicates)
	for _, existingFreq := range currentFrequencies {
		if strings.TrimSpace(existingFreq) == frequency {
			log.Info().Msgf("Frequency %s already exists in file, skipping duplicate", frequency)
			return nil // Not an error, just skip duplicate
		}
	}

	// Append new frequency
	currentFrequencies = append(currentFrequencies, frequency)

	// Write back to file (comma-separated)
	newContent := strings.Join(currentFrequencies, ",")
	err = writeFileContents(filePath, newContent)
	if err != nil {
		return fmt.Errorf("failed to write frequencies to file: %w", err)
	}

	log.Info().Msgf("Successfully appended frequency '%s' to file: %s", frequency, filePath)
	return nil
}

// getNextStoreId generates the next incremental store ID by finding the highest existing store ID
func (s *skyeConfig) getNextStoreId() (int, error) {
	log.Info().Msg("Generating next incremental store ID...")

	// Get all existing stores directly from ETCD configuration
	storeConfigs, err := s.EtcdConfig.GetStores()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to get existing stores from ETCD (might be first store): %v", err)
		// If no stores exist yet or there's an error, start with ID 1
		log.Info().Msg("No existing stores found, starting with store_id: 1")
		return 1, nil
	}

	if len(storeConfigs) == 0 {
		log.Info().Msg("No existing stores found, starting with store_id: 1")
		return 1, nil
	}

	// Find the highest existing store ID
	maxStoreId := 0
	for storeIdStr := range storeConfigs {
		storeId, err := strconv.Atoi(storeIdStr)
		if err != nil {
			log.Warn().Msgf("Invalid store ID format: %s, skipping", storeIdStr)
			continue
		}
		if storeId > maxStoreId {
			maxStoreId = storeId
		}
	}

	nextStoreId := maxStoreId + 1
	log.Info().Msgf("Found %d existing stores, highest store_id: %d, next store_id: %d", len(storeConfigs), maxStoreId, nextStoreId)

	return nextStoreId, nil
}

func sanitizeFilePath(path string) (string, error) {
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts after cleaning
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected in path: %s", path)
	}

	// Reject absolute paths (security: prevent access to system directories)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Ensure path doesn't start with / (Unix-style absolute path)
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("absolute paths not allowed: %s", path)
	}

	appName := viper.GetString("SKYE_APP_NAME")
	if appName == "" {
		return "", fmt.Errorf("SKYE_APP_NAME is not configured")
	}

	// Additional validation: ensure path is within allowed directory structure
	expectedPrefix := fmt.Sprintf("%s/", appName)
	if !strings.HasPrefix(cleaned, expectedPrefix) {
		return "", fmt.Errorf("path must be within %s/ directory: %s", appName, path)
	}

	return cleaned, nil
}

// File utility functions
func createDirectoryIfNotExists(dirPath string) error {
	sanitizedPath, err := sanitizeFilePath(dirPath)
	if err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	if _, err := os.Stat(sanitizedPath); os.IsNotExist(err) {
		return os.MkdirAll(sanitizedPath, 0755)
	}
	return nil
}

func fileExists(filePath string) bool {
	sanitizedPath, err := sanitizeFilePath(filePath)
	if err != nil {
		log.Error().Err(err).Msgf("Invalid file path: %s", filePath)
		return false
	}

	_, err = os.Stat(sanitizedPath)
	return !os.IsNotExist(err)
}

func readFileContents(filePath string) (string, error) {
	sanitizedPath, err := sanitizeFilePath(filePath)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	data, err := os.ReadFile(sanitizedPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeFileContents(filePath, content string) error {
	sanitizedPath, err := sanitizeFilePath(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	return os.WriteFile(sanitizedPath, []byte(content), 0644)
}

// ==================== VARIANT ONBOARDING DATA OPERATIONS ====================

func (s *skyeConfig) GetAllVariantOnboardingRequests() (VariantOnboardingRequestListResponse, error) {
	requests, err := s.VariantOnboardingRequestRepo.GetAll()
	if err != nil {
		return VariantOnboardingRequestListResponse{}, fmt.Errorf("failed to get variant onboarding requests: %w", err)
	}

	var variantOnboardingRequestInfos []VariantOnboardingRequestInfo
	for _, request := range requests {
		var payload VariantOnboardingRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		variantOnboardingRequestInfos = append(variantOnboardingRequestInfos, VariantOnboardingRequestInfo{
			RequestID:   request.RequestID,
			Reason:      request.Reason,
			Payload:     payload,
			RequestType: request.RequestType,
			CreatedBy:   request.CreatedBy,
			ApprovedBy:  request.ApprovedBy,
			Status:      request.Status,
			CreatedAt:   request.CreatedAt,
			UpdatedAt:   request.UpdatedAt,
		})
	}

	return VariantOnboardingRequestListResponse{
		VariantOnboardingRequests: variantOnboardingRequestInfos,
		TotalCount:                len(variantOnboardingRequestInfos),
	}, nil
}

func (s *skyeConfig) GetOnboardedVariants() (OnboardedVariantListResponse, error) {
	// Get all approved variant onboarding requests
	requests, err := s.VariantOnboardingRequestRepo.GetByStatus("APPROVED")
	if err != nil {
		return OnboardedVariantListResponse{}, fmt.Errorf("failed to get approved variant onboarding requests: %w", err)
	}

	var onboardedVariants []OnboardedVariantInfo
	for _, request := range requests {
		var payload VariantOnboardingRequestPayload
		if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
			continue // Skip invalid payloads
		}

		// For now, we'll consider all approved requests as "ONBOARDED"
		// we want to check the actual status
		// by querying the Qdrant cluster or checking Airflow job status
		onboardedVariants = append(onboardedVariants, OnboardedVariantInfo{
			Entity:       payload.Entity,
			Model:        payload.Model,
			Variant:      payload.Variant,
			VectorDBType: payload.VectorDBType,
			Status:       "ONBOARDED",
			OnboardedAt:  request.UpdatedAt,
			RequestID:    request.RequestID,
			CreatedBy:    request.CreatedBy,
			ApprovedBy:   request.ApprovedBy,
		})
	}

	return OnboardedVariantListResponse{
		OnboardedVariants: onboardedVariants,
		TotalCount:        len(onboardedVariants),
	}, nil
}
