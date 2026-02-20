package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/counter"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationjob"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationlock"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/prototext"

	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	predclient "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/predator"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorrequest"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/random"
	"github.com/Meesho/BharatMLStack/horizon/pkg/serializer"
	"github.com/rs/zerolog/log"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator/proto/modelconfig"
)

type Predator struct {
	GcsClient               externalcall.GCSClientInterface
	Repo                    predatorrequest.PredatorRequestRepository
	ServiceDeployableRepo   servicedeployableconfig.ServiceDeployableRepository
	PredatorConfigRepo      predatorconfig.PredatorConfigRepository
	ServiceDiscoveryRepo    discoveryconfig.DiscoveryConfigRepository
	groupIdCounter          counter.GroupIdCounterRepository
	infrastructureHandler   infrastructurehandler.InfrastructureHandler
	workingEnv              string
	featureValidationClient externalcall.FeatureValidationClient
	validationLockRepo      validationlock.Repository // Distributed lock repository for validation
	validationJobRepo       validationjob.Repository  // Repository for tracking validation jobs
}

func InitV1ConfigHandler() (Config, error) {
	var initErr error

	predatorOnce.Do(func() {
		connection, err := infra.SQL.GetConnection()
		if err != nil {
			log.Error().Err(err).Msg(errMsgCreateConnection)
			initErr = err
			return
		}

		sqlConn, ok := connection.(*infra.SQLConnection)
		if !ok {
			err := errors.New(errMsgTypeAssertion)
			log.Error().Err(err).Msg(errMsgTypeAssertionLog)
			initErr = err
			return
		}

		repo, err := predatorrequest.NewRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg(errMsgCreateRequestRepo)
			initErr = err
			return
		}

		serviceDeployableRepo, err := servicedeployableconfig.NewRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg(errMsgCreateDeployableRepo)
			initErr = err
			return
		}

		predatorConfigRepo, err := predatorconfig.NewRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg(errMsgCreateConfigRepo)
			initErr = err
			return
		}

		serviceDiscoveryRepo, err := discoveryconfig.NewRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg(errMsgCreateDiscoveryRepo)
			initErr = err
			return
		}
		groupIdCounter, err := counter.NewCounterRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg(errMsgCreateGroupIdCounterRepo)
			initErr = err
			return
		}

		validationLockRepo, err := validationlock.NewRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create validation lock repository")
			initErr = err
			return
		}

		validationJobRepo, err := validationjob.NewRepository(sqlConn)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create validation job repository")
			initErr = err
			return
		}

		infrastructureHandler := infrastructurehandler.InitInfrastructureHandler()
		// Use generalized working environment
		workingEnv := viper.GetString("WORKING_ENV")

		predator = &Predator{
			GcsClient:               externalcall.CreateGCSClient(),
			ServiceDeployableRepo:   serviceDeployableRepo,
			Repo:                    repo,
			PredatorConfigRepo:      predatorConfigRepo,
			ServiceDiscoveryRepo:    serviceDiscoveryRepo,
			infrastructureHandler:   infrastructureHandler,
			workingEnv:              workingEnv,
			groupIdCounter:          groupIdCounter,
			featureValidationClient: externalcall.Client,
			validationLockRepo:      validationLockRepo,
			validationJobRepo:       validationJobRepo,
		}

		// Cleanup any expired locks on startup
		go func() {
			if cleanupErr := predator.CleanupExpiredValidationLocks(); cleanupErr != nil {
				log.Error().Err(cleanupErr).Msg("Failed to cleanup expired locks on startup")
			}
		}()
	})

	return predator, initErr
}

func (p *Predator) HandleModelRequest(req ModelRequest, requestType string) (string, int, error) {
	var newRequests []predatorrequest.PredatorRequest
	var modelNameList []string
	for _, payload := range req.Payload {
		modelName, ok := payload[fieldModelName].(string)
		if !ok || modelName == "" {
			return constant.EmptyString, http.StatusBadRequest, fmt.Errorf("invalid or missing model name")
		}
		modelNameList = append(modelNameList, modelName)
	}

	var payloadObjects []Payload
	derivedModelNames := make([]string, len(modelNameList))

	for i, payload := range req.Payload {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return constant.EmptyString, http.StatusInternalServerError, errors.New(errMsgProcessPayload)
		}

		var payloadObject Payload
		if err := json.Unmarshal(payloadBytes, &payloadObject); err != nil {
			return constant.EmptyString, http.StatusInternalServerError, errors.New(errMsgProcessPayload)
		}
		derivedModelName, err := p.GetDerivedModelName(payloadObject, requestType)
		if err != nil {
			return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf("failed to fetch derived model name: %w", err)
		}
		if requestType == ScaleUpRequestType {
			payloadObject.ConfigMapping.SourceModelName = payloadObject.ModelName
		}
		payloadObject.ModelName = derivedModelName
		derivedModelNames[i] = derivedModelName
		payloadObjects = append(payloadObjects, payloadObject)
	}

	exist, err := p.Repo.ActiveModelRequestExistForRequestType(derivedModelNames, requestType)
	if err != nil {
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf("failed to check existing models: %w", err)
	}
	if exist {
		return constant.EmptyString, http.StatusConflict, fmt.Errorf("active model request already exists for one or more requested models")
	}

	predatorConfigList, err := p.PredatorConfigRepo.GetActiveModelByModelNameList(derivedModelNames)

	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("failed to fetch predator configs: %v", err))
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf(errMsgFetchConfigs, err)
	}

	if len(predatorConfigList) != 0 {
		return constant.EmptyString, http.StatusConflict, fmt.Errorf("active model already exists for this request")
	}

	groupID, err := p.groupIdCounter.GetAndIncrementCounter(1)
	if err != nil {
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf("failed to get group id: %w", err)
	}

	for i := range req.Payload {
		payloadObject := payloadObjects[i]
		payloadBytes, err := json.Marshal(payloadObject)
		if err != nil {
			return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf("failed to marshal payload: %w", err)
		}

		if payloadObject.ConfigMapping.ServiceDeployableID == 0 {
			return constant.EmptyString, http.StatusBadRequest, fmt.Errorf("service deployable id is required")
		}

		if requestType == OnboardRequestType && payloadObject.MetaData.InstanceCount > 1 {
			return constant.EmptyString, http.StatusBadRequest, fmt.Errorf("instance count should be 1 for onboard environment")
		}

		modelName := payloadObject.ModelName
		newRequests = append(newRequests, predatorrequest.PredatorRequest{
			ModelName:    modelName,
			GroupId:      groupID,
			Payload:      string(payloadBytes),
			CreatedBy:    req.CreatedBy,
			RequestType:  requestType,
			Status:       statusPendingApproval,
			RejectReason: constant.EmptyString,
			Reviewer:     constant.EmptyString,
			Active:       true,
		})
	}

	if err := p.Repo.CreateMany(newRequests); err != nil {
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf(errMsgCreateRequestFormat, strings.ToLower(requestType))
	}

	return fmt.Sprintf(successMsgFormat, requestType), http.StatusOK, nil
}

func (p *Predator) HandleDeleteModel(deleteRequest DeleteRequest, createdBy string) (string, uint, int, error) {
	predatorConfigList, err := p.PredatorConfigRepo.GetActiveModelByIds(deleteRequest.IDs)
	if err != nil {
		return constant.EmptyString, 0, http.StatusNotFound, errors.New(errModelNotFound)
	}
	isValid, err := p.ValidateDeleteRequest(predatorConfigList, deleteRequest.IDs)
	if err != nil {
		return constant.EmptyString, 0, http.StatusBadRequest, err
	}
	if !isValid {
		return constant.EmptyString, 0, http.StatusBadRequest, fmt.Errorf("issue while validating request")
	}

	groupID, err := p.groupIdCounter.GetAndIncrementCounter(1)

	if err != nil {
		return constant.EmptyString, 0, http.StatusInternalServerError, fmt.Errorf("failed to get group id: %w", err)
	}
	predatorDeleteRequestList := make([]predatorrequest.PredatorRequest, 0)
	for _, config := range predatorConfigList {
		discoveryConfig, err := p.ServiceDiscoveryRepo.GetById(config.DiscoveryConfigID)
		if err != nil {
			return constant.EmptyString, 0, http.StatusInternalServerError, errors.New(errFetchDiscoveryConfig)
		}
		serviceDeployable, err := p.ServiceDeployableRepo.GetById(discoveryConfig.ServiceDeployableID)
		if err != nil {
			return constant.EmptyString, 0, http.StatusInternalServerError, errors.New(errFetchDeployableConfig)
		}
		var deployableConfig PredatorDeployableConfig

		if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err != nil {
			return constant.EmptyString, 0, http.StatusInternalServerError, errors.New(errUnmarshalDeployableConfig)
		}

		payload := map[string]interface{}{
			fieldModelName:         config.ModelName,
			fieldModelSourcePath:   strings.TrimSuffix(deployableConfig.GCSBucketPath, "/*") + slashConstant + config.ModelName,
			fieldMetaData:          config.MetaData,
			fieldDiscoveryConfigID: discoveryConfig.ID,
			fieldConfigMapping:     ConfigMapping{ServiceDeployableID: uint(discoveryConfig.ServiceDeployableID)},
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return constant.EmptyString, 0, http.StatusInternalServerError, errors.New(errMarshalPayload)
		}

		predatorDeleteRequestList = append(predatorDeleteRequestList, predatorrequest.PredatorRequest{
			ModelName:    config.ModelName,
			GroupId:      groupID,
			Payload:      string(payloadBytes),
			CreatedBy:    createdBy,
			RequestType:  DeleteRequestType,
			Status:       pendingApproval,
			RejectReason: constant.EmptyString,
			Reviewer:     constant.EmptyString,
			IsValid:      true,
		})

	}

	if err := p.Repo.CreateMany(predatorDeleteRequestList); err != nil {
		return constant.EmptyString, 0, http.StatusInternalServerError, errors.New(errCreateDeleteRequest)
	}

	return successDeleteRequestMsg, groupID, http.StatusOK, nil
}

func (p *Predator) FetchModelConfig(req FetchModelConfigRequest) (ModelParamsResponse, int, error) {
	if err := validateModelPath(req.ModelPath); err != nil {
		return ModelParamsResponse{}, http.StatusBadRequest, err
	}

	preProdBucket, preProdObjectPath := parseModelPath(req.ModelPath)
	metaDataPath := path.Join(preProdObjectPath, "metadata.json")
	modelName := path.Base(preProdObjectPath)
	preProdConfigPath := path.Join(preProdObjectPath, configFile)

	// Read config.pbtxt
	var configData []byte
	var err error
	if p.isNonProductionEnvironment() {
		configData, err = p.GcsClient.ReadFile(preProdBucket, preProdConfigPath)
	} else {
		prodConfigPath := path.Join(pred.GcsConfigBasePath, modelName, configFile)
		configData, err = p.GcsClient.ReadFile(pred.GcsConfigBucket, prodConfigPath)
	}
	if err != nil {
		return ModelParamsResponse{}, http.StatusInternalServerError, fmt.Errorf(errReadConfigFileFormat, err)
	}

	// Read feature_meta.json
	metaData, err := p.GcsClient.ReadFile(preProdBucket, metaDataPath)
	var featureMeta *FeatureMetadata
	if err == nil && metaData != nil {
		if err := json.Unmarshal(metaData, &featureMeta); err != nil {
			log.Error().Err(err).Msg("failed to parse feature_meta.json")
		}
	}

	// Create feature map for quick lookup
	featureMap := make(map[string][]string)

	if featureMeta != nil && featureMeta.Inputs != nil {
		for _, input := range featureMeta.Inputs {
			featureMap[input.Name] = input.Features
		}
	}

	var modelConfig modelconfig.ModelConfig
	if err := prototext.Unmarshal(configData, &modelConfig); err != nil {
		return ModelParamsResponse{}, http.StatusInternalServerError, fmt.Errorf(errUnmarshalProtoFormat, err)
	}

	if err := validateModelConfig(&modelConfig); err != nil {
		return ModelParamsResponse{}, http.StatusBadRequest, err
	}

	// Modify convertInput to include features
	inputs := convertInputWithFeatures(modelConfig.Input, featureMap)
	outputs := convertOutput(modelConfig.Output)
	if inputs == nil {
		inputs = []IO{}
	}
	if outputs == nil {
		outputs = []IO{}
	}

	return createModelParamsResponse(&modelConfig, preProdObjectPath, inputs, outputs), http.StatusOK, nil
}

func validateModelPath(modelPath string) error {
	if modelPath == constant.EmptyString || !strings.HasPrefix(modelPath, slashConstant) {
		return errors.New(errModelPathPrefix)
	}
	parts := strings.SplitN(modelPath[1:], slashConstant, 2)
	if len(parts) != 2 {
		return errors.New(errModelPathFormat)
	}
	return nil
}

func parseModelPath(modelPath string) (bucket, objectPath string) {
	parts := strings.SplitN(modelPath[1:], slashConstant, 2)
	return parts[0], parts[1]
}

func validateModelConfig(cfg *modelconfig.ModelConfig) error {
	switch {
	case cfg.Name == constant.EmptyString:
		return errors.New(errModelNameMissing)
	case len(cfg.Input) == 0:
		return errors.New(errNoInputDefinitions)
	case len(cfg.Output) == 0:
		return errors.New(errNoOutputDefinitions)
	}
	return nil
}

func convertFields(name string, dims []int64, dataType string) (IO, bool) {
	if name == constant.EmptyString || len(dims) == 0 || dataType == constant.EmptyString {
		return IO{}, false
	}

	var dtype string
	if dataType == typeString {
		dtype = bytesKeys
	} else {
		dtype = strings.TrimPrefix(dataType, typePrefix)
	}

	return IO{
		Name:     name,
		Dims:     dims,
		DataType: dtype,
		Features: nil, // Initialize empty features, will be populated by convertInputWithFeatures if available
	}, true
}

func convertInputWithFeatures(fields []*modelconfig.ModelInput, featureMap map[string][]string) []IO {
	ios := make([]IO, 0, len(fields))
	for _, f := range fields {
		if io, ok := convertFields(f.Name, f.Dims, f.DataType.String()); ok {
			// Add features from metadata if available
			if features, exists := featureMap[f.Name]; exists {
				io.Features = features
			}
			ios = append(ios, io)
		}
	}
	return ios
}

func convertOutput(fields []*modelconfig.ModelOutput) []IO {
	ios := make([]IO, 0, len(fields))
	for _, f := range fields {
		if io, ok := convertFields(f.Name, f.Dims, f.DataType.String()); ok {
			ios = append(ios, io)
		}
	}
	return ios
}

func createModelParamsResponse(modelConfig *modelconfig.ModelConfig, objectPath string, inputs, outputs []IO) ModelParamsResponse {
	var resp ModelParamsResponse

	if len(modelConfig.InstanceGroup) > 0 {
		resp.InstanceCount = modelConfig.InstanceGroup[0].Count
		resp.InstanceType = modelConfig.InstanceGroup[0].Kind.String()
	}
	if modelConfig.MaxBatchSize > 0 {
		resp.BatchSize = modelConfig.MaxBatchSize
	}

	resp.Backend = modelConfig.Backend

	if modelConfig.GetDynamicBatching() != nil {
		resp.DynamicBatchingEnabled = true
	} else {
		resp.DynamicBatchingEnabled = false
	}

	if ensembleScheduling := modelConfig.GetEnsembleScheduling(); ensembleScheduling != nil {
		resp.EnsembleScheduling = ensembleScheduling
	}

	resp.Platform = modelConfig.Platform

	resp.ModelName = path.Base(objectPath)
	resp.Inputs = inputs
	resp.Outputs = outputs

	return resp
}

func (p *Predator) ProcessRequest(req ApproveRequest) error {
	if req.Status == statusRejected && req.RejectReason == constant.EmptyString {
		return errors.New("reject reason is required when status is rejected")
	}
	idUint64, err := strconv.ParseUint(req.GroupID, 10, 32)
	if err != nil {
		return errors.New(errInvalidRequestIDFormat)
	}

	groupID := uint(idUint64)

	predatorRequestList, err := p.Repo.GetAllByGroupID(groupID)
	if err != nil {
		return fmt.Errorf(errFailedToFetchRequest, fmt.Sprintf("%d", groupID))
	}
	if len(predatorRequestList) == 0 {
		return fmt.Errorf("no request found for group id %s", req.GroupID)
	}

	requestIdPayloadMap := make(map[uint]*Payload)
	requestTypeValidationMap := make(map[string]bool)
	requestTypeCount := 0
	for _, predatorRequest := range predatorRequestList {
		if predatorRequest.Status == statusApproved {
			return fmt.Errorf("some request already approved for group id %d", groupID)
		}

		// Skip IsValid check in two cases:
		// 1. When req.Status != statusApproved (not approving the request)
		// 2. When request type is DeleteRequestType
		skipValidCheck := req.Status != statusApproved || predatorRequestList[0].RequestType == DeleteRequestType

		if !skipValidCheck && !predatorRequest.IsValid {
			return fmt.Errorf("some Request is not validated or Request validation failed for group id %d", groupID)
		}

		if _, exists := requestTypeValidationMap[predatorRequest.RequestType]; exists {

		} else {
			requestTypeCount++
		}
		requestTypeValidationMap[predatorRequest.RequestType] = true

		payload, err := p.processPayload(predatorRequest)
		if err != nil {
			p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRequestPayloadError)
			return fmt.Errorf(errMsgParsePayload, predatorRequest.RequestID, err)
		}
		requestIdPayloadMap[predatorRequest.RequestID] = payload
	}

	if requestTypeCount > 1 {
		return fmt.Errorf("multiple request types are not allowed in a single group id")
	}

	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRequestPayloadError)
		return err
	}

	go p.processRequest(requestIdPayloadMap, predatorRequestList, req)

	return nil
}

func (p *Predator) FetchModels() ([]ModelResponse, error) {
	predatorConfigs, err := p.PredatorConfigRepo.FindAllActiveConfig()
	if err != nil {
		return nil, fmt.Errorf(errMsgFetchConfigs, err)
	}

	if len(predatorConfigs) == 0 {
		return []ModelResponse{}, nil
	}

	// Phase 1: Batch fetch all required data to avoid N+1 queries
	discoveryConfigs, serviceDeployables, err := p.batchFetchRelatedData(predatorConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch related data: %w", err)
	}

	// Phase 2: Concurrently fetch deployable configs
	deployableConfigs, err := p.batchFetchDeployableConfigs(serviceDeployables)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch deployable configs: %w", err)
	}

	// Phase 3: Build response objects
	results := p.buildModelResponses(predatorConfigs, discoveryConfigs, serviceDeployables, deployableConfigs)

	return results, nil
}

func (p *Predator) FetchAllPredatorRequests(role, email string) ([]map[string]interface{}, error) {
	var requests []predatorrequest.PredatorRequest
	var err error

	if role == adminRole {
		requests, err = p.Repo.GetAll()
	} else {
		requests, err = p.Repo.GetAllByEmail(email)
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching predator requests: %v", err)
	}

	groupedRequests := make(map[uint][]PredatorRequestResponse)

	for _, req := range requests {
		var parsedPayload map[string]interface{}
		if err := json.Unmarshal([]byte(req.Payload), &parsedPayload); err != nil {
			return nil, fmt.Errorf("error parsing payload for request ID %d: %v", req.RequestID, err)
		}

		// Initialize response with default values
		requestResponse := PredatorRequestResponse{
			RequestID:    req.RequestID,
			GroupID:      req.GroupId,
			Payload:      parsedPayload,
			CreatedBy:    req.CreatedBy,
			UpdatedBy:    req.UpdatedBy,
			Reviewer:     req.Reviewer,
			RequestStage: req.RequestStage,
			RequestType:  req.RequestType,
			Status:       req.Status,
			RejectReason: req.RejectReason,
			CreatedAt:    req.CreatedAt,
			UpdatedAt:    req.UpdatedAt,
			IsValid:      req.IsValid,
			HasNilData:   false,
			TestResults:  json.RawMessage("{}"),
		}

		// Extract model name from payload and fetch predator config
		// Skip predator config lookup for edit requests as models might not exist in DB yet
		if modelName, ok := parsedPayload["model_name"].(string); ok && modelName != "" {
			if predatorConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(modelName); err == nil {
				requestResponse.HasNilData = predatorConfig.HasNilData
				if predatorConfig.TestResults != nil {
					requestResponse.TestResults = predatorConfig.TestResults
				}
			}
		}

		groupedRequests[req.GroupId] = append(groupedRequests[req.GroupId], requestResponse)
	}

	var response []map[string]interface{}

	for groupID, groupRequests := range groupedRequests {
		groupData := map[string]interface{}{
			"group_id": groupID,
			"groups":   groupRequests,
		}
		response = append(response, groupData)
	}

	return response, nil
}

func (p *Predator) ValidateRequest(groupId string) (string, int) {
	// Validate input and basic checks first (before acquiring lock)
	id, err := strconv.ParseUint(groupId, 10, 32)
	if err != nil {
		return "Invalid request ID format", http.StatusBadRequest
	}

	request, err := p.Repo.GetAllByGroupID(uint(id))
	if err != nil {
		return "Request not found", http.StatusNotFound
	}

	if len(request) == 0 {
		return "Request Validation Failed. No requests found", http.StatusNotFound
	}

	payload, err := p.processPayload(request[0])
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse payload for validation")
		return "Request Validation Failed. Failed to parse request payload", http.StatusBadRequest
	}

	// Determine test deployable ID based on machine type
	testDeployableID, err := p.getTestDeployableID(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to determine test deployable ID")
		return "Request Validation Failed. Failed to determine test deployable ID", http.StatusInternalServerError
	}

	// Create deployable-specific lock key (allows parallel processing for different deployables)
	lockKey := fmt.Sprintf("validation-deployable-%d", testDeployableID)

	// Try to acquire deployable-specific distributed lock
	lock, err := p.validationLockRepo.AcquireLock(lockKey, 30*time.Minute)
	if err != nil {
		log.Warn().Err(err).Msgf("Validation request for group ID %s rejected - failed to acquire lock for deployable %d", groupId, testDeployableID)
		return fmt.Sprintf("Request Validation Failed. Another validation is already in progress for %s deployable. Please try again later.",
			map[int]string{pred.TestDeployableID: "CPU", pred.TestGpuDeployableID: "GPU"}[testDeployableID]), http.StatusConflict
	}

	log.Info().Msgf("Starting validation for group ID: %s on deployable %d (lock acquired by %s)", groupId, testDeployableID, lock.LockedBy)

	// Validate request status
	for _, req := range request {
		if req.Status == statusApproved {
			p.releaseLockWithError(lock.ID, groupId, "Request already approved")
			return "Request Validation Failed. Request is already approved", http.StatusBadRequest
		}
		if req.Status == statusRejected {
			p.releaseLockWithError(lock.ID, groupId, "Request already rejected")
			return "Request Validation Failed. Request is already rejected", http.StatusBadRequest
		}
	}

	// Get service name from deployable config
	serviceName, err := p.getServiceNameFromDeployable(testDeployableID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get service name from deployable")
		p.releaseLockWithError(lock.ID, groupId, "Failed to get service name")
		return "Request Validation Failed. Failed to get service name", http.StatusInternalServerError
	}

	// Create validation job
	validationJob := &validationjob.Table{
		GroupID:             groupId,
		LockID:              lock.ID,
		TestDeployableID:    testDeployableID,
		ServiceName:         serviceName,
		Status:              validationjob.StatusPending,
		MaxHealthChecks:     15,
		HealthCheckInterval: 60,
	}

	if err := p.validationJobRepo.Create(validationJob); err != nil {
		log.Error().Err(err).Msg("Failed to create validation job")
		p.releaseLockWithError(lock.ID, groupId, "Failed to create validation job")
		return "Request Validation Failed. Failed to create validation job", http.StatusInternalServerError
	}

	// Start asynchronous validation process
	go p.performAsyncValidation(validationJob, request, payload, testDeployableID)

	log.Info().Msgf("Validation job created for group ID: %s, job ID: %d", groupId, validationJob.ID)
	return "Request Validation Started. The validation will run asynchronously and update the request status when complete.", http.StatusOK
}

// CleanupExpiredValidationLocks removes expired validation locks
// This method can be called periodically to clean up stale locks
func (p *Predator) CleanupExpiredValidationLocks() error {
	if p.validationLockRepo == nil {
		return errors.New("validation lock repository not initialized")
	}

	log.Info().Msg("Starting cleanup of expired validation locks")

	if err := p.validationLockRepo.CleanupExpiredLocks(); err != nil {
		log.Error().Err(err).Msg("Failed to cleanup expired validation locks")
		return err
	}

	log.Info().Msg("Successfully cleaned up expired validation locks")
	return nil
}

// GetValidationStatus returns the current validation lock status
func (p *Predator) GetValidationStatus() (bool, *validationlock.Table, error) {
	if p.validationLockRepo == nil {
		return false, nil, errors.New("validation lock repository not initialized")
	}

	isLocked, err := p.validationLockRepo.IsLocked(validationlock.ValidationLockKey)
	if err != nil {
		return false, nil, err
	}

	if !isLocked {
		return false, nil, nil
	}

	activeLock, err := p.validationLockRepo.GetActiveLock(validationlock.ValidationLockKey)
	if err != nil {
		return false, nil, err
	}

	return true, activeLock, nil
}

// GetValidationJobStatus returns the status of a validation job for a given group ID
func (p *Predator) GetValidationJobStatus(groupId string) (*validationjob.Table, error) {
	if p.validationJobRepo == nil {
		return nil, errors.New("validation job repository not initialized")
	}

	job, err := p.validationJobRepo.GetByGroupID(groupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get validation job for group %s: %w", groupId, err)
	}

	return job, nil
}

func (p *Predator) GenerateFunctionalTestRequest(req RequestGenerationRequest) (RequestGenerationResponse, error) {

	modelConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(req.ModelName)

	if err != nil {
		return RequestGenerationResponse{}, err
	}

	metadata := modelConfig.MetaData

	var md MetaData

	err = json.Unmarshal(metadata, &md)
	if err != nil {
		return RequestGenerationResponse{}, err
	}
	response := RequestGenerationResponse{
		ModelName: req.ModelName,
		RequestBody: RequestBody{
			Inputs: []Input{},
		},
	}

	intBatchSize, err := strconv.Atoi(req.BatchSize)

	if err != nil {
		return RequestGenerationResponse{}, err
	}

	inputs := md.Inputs

	for _, input := range inputs {
		var data []any

		// Convert input.Dims to []int, handling nested interfaces and various types
		shapeInt, err := convertDimsToIntSlice(input.Dims)
		if err != nil {
			return RequestGenerationResponse{}, fmt.Errorf("failed to convert dims for input %s: %w", input.Name, err)
		}

		for i := 0; i < intBatchSize; i++ {
			data = append(data, random.GenerateRandom(shapeInt, input.DataType))
		}

		// Create new shape with batch size prepended
		newshape := make([]int64, len(shapeInt)+1)
		newshape[0] = int64(intBatchSize) // newshape = batchsize + original shape
		for i, dim := range shapeInt {
			newshape[i+1] = int64(dim)
		}

		response.RequestBody.Inputs = append(response.RequestBody.Inputs, Input{
			Name:     input.Name,
			Dims:     newshape,
			DataType: input.DataType,
			Data:     data,
			Features: input.Features,
		})
	}

	return response, nil
}

func (p *Predator) ExecuteFunctionalTestRequest(req ExecuteRequestFunctionalRequest) (ExecuteRequestFunctionalResponse, error) {

	if len(req.RequestBody.Inputs) == 0 {
		return ExecuteRequestFunctionalResponse{}, fmt.Errorf("no inputs found in request body")
	}

	modelConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(req.ModelName)
	if err != nil {
		return ExecuteRequestFunctionalResponse{}, err
	}

	// Build helix predator client config from endpoint and env
	host, port := func(endpoint string) (string, string) {
		parts := strings.Split(endpoint, ":")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
		return endpoint, "8080"
	}(req.EndPoint)

	callerID := os.Getenv("PREDATOR_CALLER_ID")
	if callerID == constant.EmptyString {
		callerID = "HORIZON"
	}
	callerToken := os.Getenv("PREDATOR_AUTH_TOKEN")
	if callerToken == constant.EmptyString {
		callerToken = "HORIZON"
	}

	client := predclient.NewClientV1(&predclient.Config{
		Host:        host,
		Port:        port,
		PlainText:   true,
		CallerId:    callerID,
		CallerToken: callerToken,
	})

	// Prepare helix predator request
	// Load outputs metadata to request by name
	var md MetaData
	if err := json.Unmarshal(modelConfig.MetaData, &md); err != nil {
		return ExecuteRequestFunctionalResponse{}, err
	}

	helixReq := &predclient.PredatorRequest{
		ModelName:    req.ModelName,
		ModelVersion: "1",
		Inputs:       []predclient.Input{},
		Outputs:      []predclient.Output{},
		BatchSize:    1,
		Deadline:     30000,
	}

	for _, out := range md.Outputs {
		helixReq.Outputs = append(helixReq.Outputs, predclient.Output{
			Name: out.Name,
		})
	}

	// Map inputs: convert to [][][]byte per helix format [batch][feature][bytes]
	for _, in := range req.RequestBody.Inputs {
		normalizedDT := strings.ToUpper(strings.TrimPrefix(in.DataType, "TYPE_"))
		if normalizedDT == "STRING" {
			normalizedDT = "BYTES"
		}

		// Extract dims without batch dimension for helix client
		inputDims := make([]int, 0)
		if len(in.Dims) > 1 {
			for _, d := range in.Dims[1:] {
				inputDims = append(inputDims, int(d))
			}
		}

		// Determine batch size from dims[0] if present
		batchSize := 1
		if len(in.Dims) > 0 {
			batchSize = int(in.Dims[0])
		}
		if batchSize <= 0 {
			batchSize = 1
		}
		if batchSize > helixReq.BatchSize {
			helixReq.BatchSize = batchSize
		}

		// Build data3D
		data3D := make([][][]byte, 0, batchSize)
		// Expect in.Data as []any with length batchSize
		if batch, ok := in.Data.([]any); ok {
			for _, sample := range batch {
				// Flatten sample to per-feature bytes
				features, ferr := func(sample any, dt string) ([][]byte, error) {
					flat, err := serializer.FlattenMatrixByType(sample, dt)
					if err != nil {
						return nil, err
					}
					switch dt {
					case "FP16":
						vals, ok := flat.([]float32)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for FP16")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b, err := serializer.Float32ToFloat16Bytes(v)
							if err != nil {
								return nil, err
							}
							out[i] = b
						}
						return out, nil
					case "FP32":
						vals, ok := flat.([]float32)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for FP32")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 4)
							binary.LittleEndian.PutUint32(b, math.Float32bits(v))
							out[i] = b
						}
						return out, nil
					case "INT64":
						vals, ok := flat.([]int64)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for INT64")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 8)
							binary.LittleEndian.PutUint64(b, uint64(v))
							out[i] = b
						}
						return out, nil
					case "INT32":
						vals, ok := flat.([]int32)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for INT32")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 4)
							binary.LittleEndian.PutUint32(b, uint32(v))
							out[i] = b
						}
						return out, nil
					case "INT16":
						vals, ok := flat.([]int16)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for INT16")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 2)
							binary.LittleEndian.PutUint16(b, uint16(v))
							out[i] = b
						}
						return out, nil
					case "INT8":
						// FlattenMatrixByType returns []int8 for INT8
						vals, ok := flat.([]int8)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for INT8")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							out[i] = []byte{byte(v)}
						}
						return out, nil
					case "UINT64":
						vals, ok := flat.([]uint64)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for UINT64")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 8)
							binary.LittleEndian.PutUint64(b, v)
							out[i] = b
						}
						return out, nil
					case "UINT32":
						vals, ok := flat.([]uint32)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for UINT32")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 4)
							binary.LittleEndian.PutUint32(b, v)
							out[i] = b
						}
						return out, nil
					case "UINT16":
						vals, ok := flat.([]uint16)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for UINT16")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							b := make([]byte, 2)
							binary.LittleEndian.PutUint16(b, v)
							out[i] = b
						}
						return out, nil
					case "UINT8":
						vals, ok := flat.([]uint8)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for UINT8")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							out[i] = []byte{v}
						}
						return out, nil
					case "BOOL":
						vals, ok := flat.([]bool)
						if !ok {
							return nil, fmt.Errorf("unexpected flat type for BOOL")
						}
						out := make([][]byte, len(vals))
						for i, v := range vals {
							if v {
								out[i] = []byte{1}
							} else {
								out[i] = []byte{0}
							}
						}
						return out, nil
					case "BYTES":
						if bs, ok := flat.([][]byte); ok {
							return bs, nil
						}
						if ss, ok := flat.([]string); ok {
							out := make([][]byte, len(ss))
							for i, s := range ss {
								out[i] = []byte(s)
							}
							return out, nil
						}
						return nil, fmt.Errorf("unexpected flat type for BYTES")
					default:
						return nil, fmt.Errorf("unsupported datatype %s", dt)
					}
				}(sample, normalizedDT)
				if ferr != nil {
					return ExecuteRequestFunctionalResponse{}, fmt.Errorf("failed to prepare input %s: %w", in.Name, ferr)
				}
				data3D = append(data3D, features)
			}
		} else {
			// Fallback: use existing helper for entire data (single batch)
			flattenedData, ferr := p.flattenInputTo3DByteSlice(in.Data, normalizedDT)
			if ferr != nil {
				return ExecuteRequestFunctionalResponse{}, fmt.Errorf("failed to flatten input %s: %w", in.Name, ferr)
			}
			data3D = flattenedData
		}

		helixReq.Inputs = append(helixReq.Inputs, predclient.Input{
			Name:     in.Name,
			DataType: normalizedDT,
			Dims:     inputDims,
			Data:     data3D,
		})
	}

	// Execute via helix client (doesn't accept context; deadline is set in request)
	helixResp, err := client.GetInferenceScoreV2(helixReq)

	modelConfig.TestResults = json.RawMessage(`{"is_functionally_tested": true}`)

	if err != nil {
		modelConfig.HasNilData = true
		p.PredatorConfigRepo.Update(modelConfig)
		return ExecuteRequestFunctionalResponse{}, fmt.Errorf("failed to execute functional test: %w", err)
	}

	// Convert RawOutputContents to response format
	convertedOutputs := []Output{}

	if len(helixResp.RawOutputContents) > 0 {
		// Convert RawOutputContents to strings based on output data type and dims
		for i, outputBytes := range helixResp.RawOutputContents {
			if len(outputBytes) == 0 {
				modelConfig.HasNilData = true
				p.PredatorConfigRepo.Update(modelConfig)
				return ExecuteRequestFunctionalResponse{}, fmt.Errorf("functional test failed: empty output %d", i)
			}

			// Get output metadata for this output
			if i < len(md.Outputs) {
				outputMeta := md.Outputs[i]

				// Convert dims to []int64
				var dims []int64
				if dimsInterface, ok := outputMeta.Dims.([]interface{}); ok {
					for _, d := range dimsInterface {
						if intVal, ok := d.(float64); ok {
							dims = append(dims, int64(intVal))
						}
					}
				}

				// Calculate elements per batch (excluding batch dimension)
				elementsPerBatch := int64(1)
				for _, dim := range dims {
					elementsPerBatch *= dim
				}

				normalizedOutputDT := strings.ToUpper(strings.TrimPrefix(outputMeta.DataType, "TYPE_"))
				isStringType := normalizedOutputDT == "STRING" || normalizedOutputDT == "BYTES"

				elementSize := getElementSize(outputMeta.DataType)
				bytesPerBatch := int(elementsPerBatch * int64(elementSize))

				if isStringType {
					var allBatches [][]interface{}
					offset := 0
					for offset < len(outputBytes) {
						var batchSlice []interface{}
						for j := int64(0); j < elementsPerBatch && offset < len(outputBytes); j++ {
							if offset+4 > len(outputBytes) {
								modelConfig.HasNilData = true
								p.PredatorConfigRepo.Update(modelConfig)
								return ExecuteRequestFunctionalResponse{}, fmt.Errorf("functional test failed: insufficient bytes for string length at offset %d", offset)
							}

							length := binary.LittleEndian.Uint32(outputBytes[offset : offset+4])
							offset += 4

							if offset+int(length) > len(outputBytes) {
								modelConfig.HasNilData = true
								p.PredatorConfigRepo.Update(modelConfig)
								return ExecuteRequestFunctionalResponse{}, fmt.Errorf("functional test failed: insufficient bytes for string content at offset %d, expected %d bytes", offset, length)
							}

							stringContent := outputBytes[offset : offset+int(length)]
							offset += int(length)
							batchSlice = append(batchSlice, string(stringContent))
						}

						if len(batchSlice) > 0 {
							allBatches = append(allBatches, batchSlice)
						}

						if offset >= len(outputBytes) {
							break
						}
					}

					convertedOutputs = append(convertedOutputs, Output{
						Name:     outputMeta.Name,
						Dims:     dims,
						DataType: outputMeta.DataType,
						Data:     allBatches,
					})
				} else if elementSize > 0 && len(outputBytes) >= bytesPerBatch {
					// Calculate number of batches from total bytes
					numBatches := len(outputBytes) / bytesPerBatch

					// Process each batch
					var allBatches [][]interface{}
					for batchIdx := 0; batchIdx < numBatches; batchIdx++ {
						start := batchIdx * bytesPerBatch
						end := start + bytesPerBatch
						batchBytes := outputBytes[start:end]

						// Split batch bytes into individual elements
						var elementBytes [][]byte
						for j := int64(0); j < elementsPerBatch; j++ {
							elementStart := int(j * int64(elementSize))
							elementEnd := elementStart + elementSize
							if elementEnd <= len(batchBytes) {
								elementBytes = append(elementBytes, batchBytes[elementStart:elementEnd])
							}
						}

						// Convert this batch
						converted, convErr := serializer.ConvertFromRawBytes(elementBytes, outputMeta.DataType)
						if convErr != nil {
							log.Warn().Err(convErr).Str("output", outputMeta.Name).Int("batch", batchIdx).Msg("failed to convert output bytes")
							modelConfig.HasNilData = true
							p.PredatorConfigRepo.Update(modelConfig)
							return ExecuteRequestFunctionalResponse{}, fmt.Errorf("failed to convert output %s batch %d: %w", outputMeta.Name, batchIdx, convErr)
						}

						// Convert to slice for reshaping
						var batchSlice []interface{}
						switch v := converted.(type) {
						case []interface{}:
							batchSlice = v
						case []string:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []float32:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []float64:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []int32:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []int64:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []uint32:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []uint64:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []uint16:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						case []uint8:
							for _, item := range v {
								batchSlice = append(batchSlice, item)
							}
						default:
							// If we can't convert, use as-is
							batchSlice = []interface{}{converted}
						}

						allBatches = append(allBatches, batchSlice)
					}

					convertedOutputs = append(convertedOutputs, Output{
						Name:     outputMeta.Name,
						Dims:     dims,
						DataType: outputMeta.DataType,
						Data:     allBatches,
					})
					log.Info().Str("output", outputMeta.Name).Int("batches", numBatches).Interface("data", allBatches).Msg("converted output data")
				} else {
					// Fallback: try direct conversion
					converted, convErr := serializer.ConvertFromRawBytes([][]byte{outputBytes}, outputMeta.DataType)
					if convErr != nil {
						log.Warn().Err(convErr).Str("output", outputMeta.Name).Msg("failed to convert output bytes")
						modelConfig.HasNilData = true
						p.PredatorConfigRepo.Update(modelConfig)
						return ExecuteRequestFunctionalResponse{}, fmt.Errorf("failed to convert output %s: %w", outputMeta.Name, convErr)
					}

					// Reshape data to preserve batch dimension
					reshapedData := reshapeDataForBatch(converted, dims)

					convertedOutputs = append(convertedOutputs, Output{
						Name:     outputMeta.Name,
						Dims:     dims,
						DataType: outputMeta.DataType,
						Data:     reshapedData,
					})
					log.Info().Str("output", outputMeta.Name).Interface("data", reshapedData).Msg("converted output data")
				}
			}
		}
	} else {
		modelConfig.HasNilData = true
		p.PredatorConfigRepo.Update(modelConfig)
		return ExecuteRequestFunctionalResponse{}, fmt.Errorf("no raw output contents received from helix")
	}

	modelConfig.HasNilData = false
	p.PredatorConfigRepo.Update(modelConfig)

	// Return converted response
	return ExecuteRequestFunctionalResponse{
		ModelName:    req.ModelName,
		ModelVersion: helixResp.GetModelVersion(),
		Outputs:      convertedOutputs,
	}, nil
}

func (p *Predator) SendLoadTestRequest(req ExecuteRequestLoadTest) (PhoenixClientResponse, error) {
	phoenixclientbaseURL := pred.PhoenixServerBaseUrl

	executeURL := phoenixclientbaseURL + "/execute"

	tritonRequest, err := ConvertExecuteRequestToTritonRequest(req)

	if err != nil {
		return PhoenixClientResponse{}, fmt.Errorf("failed to convert request to Triton Request: %w", err)
	}

	tritonRequestBody, err := json.Marshal(tritonRequest)
	if err != nil {
		return PhoenixClientResponse{}, fmt.Errorf("failed to marshal Triton Request: %w", err)
	}

	phoenixLoadTestRequest := PhoenixLoadTestRequest{
		LoadTestConfig: req.LoadTestConfig,
		Service:        predatorInferMethod,
		Endpoint:       req.Endpoint,
		Data:           tritonRequestBody,
		Metadata:       req.Metadata,
	}

	reqBody, err := json.Marshal(phoenixLoadTestRequest)
	if err != nil {
		return PhoenixClientResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodGet, executeURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return PhoenixClientResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return PhoenixClientResponse{}, fmt.Errorf("failed to send request to Phoenix: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return PhoenixClientResponse{}, fmt.Errorf("phoenix server returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response PhoenixClientResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return PhoenixClientResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}

func ConvertExecuteRequestToTritonRequest(req ExecuteRequestLoadTest) (TritonInferenceRequest, error) {
	response := TritonInferenceRequest{
		ModelName:        req.ModelName,
		Inputs:           []TritonInputTensor{},
		Outputs:          []TritonOutputTensor{},
		RawInputContents: [][]byte{},
	}

	for _, input := range req.RequestBody.Inputs {

		tritonInput := TritonInputTensor{
			Name: input.Name,

			Datatype: input.DataType,
			Shape:    input.Dims,
		}
		response.Inputs = append(response.Inputs, tritonInput)

		flattenedData, err := serializer.FlattenMatrixByType(input.Data, input.DataType)
		if err != nil {
			return TritonInferenceRequest{}, fmt.Errorf("failed to flatten data for input %s: %w", input.Name, err)
		}

		rawByteData, err := serializer.ConvertToRawBytes(flattenedData)
		if err != nil {
			return TritonInferenceRequest{}, fmt.Errorf("failed to convert to raw bytes for input %s: %w", input.Name, err)
		}

		response.RawInputContents = append(response.RawInputContents, rawByteData)
	}

	response.Outputs = append(response.Outputs, TritonOutputTensor{
		Name: "output__0",
	})

	return response, nil
}

func (p *Predator) GetGCSModels() (*GCSFoldersResponse, error) {
	bucket := pred.GcsModelBucket
	basePath := pred.GcsModelBasePath

	// Remove trailing slash to avoid double slashes
	basePath = strings.TrimSuffix(basePath, "/")

	folders, err := p.GcsClient.ListFolders(bucket, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list GCS folders: %w", err)
	}

	if len(folders) == 0 {
		return &GCSFoldersResponse{Folders: []GCSFolder{}}, nil
	}

	// Use semaphore to limit concurrent GCS reads (max 10 at a time)
	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup

	// Use channel to collect results safely
	results := make(chan GCSFolder, len(folders))

	for _, folder := range folders {
		wg.Add(1)
		go func(folderName string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			gcsFolder := GCSFolder{
				Name: folderName,
				Path: fmt.Sprintf("gcs://%s/%s/%s", bucket, basePath, folderName),
			}

			// Try to download and parse metadata.json from the model folder
			metadataPath := path.Join(basePath, folderName, "metadata.json")
			metadataBytes, err := p.GcsClient.ReadFile(bucket, metadataPath)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to read metadata.json for model %s, skipping metadata", folderName)
			} else {
				var metadata FeatureMetadata
				if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
					log.Warn().Err(err).Msgf("Failed to parse metadata.json for model %s, skipping metadata", folderName)
				} else {
					gcsFolder.Metadata = &metadata
					log.Info().Msgf("Successfully loaded metadata for model %s", folderName)
				}
			}

			// Send result through channel (thread-safe)
			results <- gcsFolder
		}(folder)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results from channel
	response := &GCSFoldersResponse{
		Folders: make([]GCSFolder, 0, len(folders)),
	}

	for gcsFolder := range results {
		response.Folders = append(response.Folders, gcsFolder)
	}

	return response, nil
}

func (p *Predator) UploadJSONToGCS(bucketName, fileName string, data []byte) (string, error) {
	if err := p.GcsClient.UploadFile(bucketName, fileName, data); err != nil {
		log.Error().Err(err).Msg("Failed to upload JSON to GCS")
		return "", fmt.Errorf("failed to upload JSON to GCS: %w", err)
	}
	fileURL := fmt.Sprintf("gcs://%s/%s", bucketName, fileName)
	return fileURL, nil
}

func (p *Predator) HandleEditModel(req ModelRequest, createdBy string) (string, int, error) {
	// Extract model names for validation
	var modelNameList []string
	for _, payload := range req.Payload {
		modelName, ok := payload[fieldModelName].(string)
		if !ok || modelName == "" {
			return constant.EmptyString, http.StatusBadRequest, fmt.Errorf("invalid or missing model name")
		}
		modelNameList = append(modelNameList, modelName)
	}

	// Check if there are any active edit requests for these models
	exist, err := p.Repo.ActiveModelRequestExistForRequestType(modelNameList, EditRequestType)
	if err != nil {
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf("failed to check existing edit requests: %w", err)
	}
	if exist {
		return constant.EmptyString, http.StatusConflict, fmt.Errorf("active edit request already exists for one or more requested models")
	}

	var newRequests []predatorrequest.PredatorRequest

	groupID, err := p.groupIdCounter.GetAndIncrementCounter(1)
	if err != nil {
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf("failed to get group id: %w", err)
	}

	for _, payload := range req.Payload {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return constant.EmptyString, http.StatusInternalServerError, errors.New(errMsgProcessPayload)
		}

		var payloadObject Payload
		if err := json.Unmarshal(payloadBytes, &payloadObject); err != nil {
			return constant.EmptyString, http.StatusInternalServerError, errors.New(errMsgProcessPayload)
		}

		if payloadObject.MetaData.InstanceCount > 1 && p.isNonProductionEnvironment() {
			return constant.EmptyString, http.StatusBadRequest, fmt.Errorf("instance count should be 1 for non-production environment")
		}

		modelName, _ := payload[fieldModelName].(string)
		newRequests = append(newRequests, predatorrequest.PredatorRequest{
			ModelName:    modelName,
			GroupId:      groupID,
			Payload:      string(payloadBytes),
			CreatedBy:    req.CreatedBy,
			RequestType:  EditRequestType,
			Status:       statusPendingApproval,
			RejectReason: constant.EmptyString,
			Reviewer:     constant.EmptyString,
			Active:       true,
		})
	}

	if err := p.Repo.CreateMany(newRequests); err != nil {
		return constant.EmptyString, http.StatusInternalServerError, fmt.Errorf(errMsgCreateRequestFormat, strings.ToLower(EditRequestType))
	}

	return fmt.Sprintf(successMsgFormat, EditRequestType), http.StatusOK, nil

}

// UploadModelFolderFromLocal uploads models from GCS source paths to destination bucket
// This is a simplified and cleaner implementation of the upload flow
func (p *Predator) UploadModelFolderFromLocal(req UploadModelFolderRequest, isPartial bool, authToken string) (UploadModelFolderResponse, int, error) {
	log.Info().Msgf("Starting upload of %d models (partial: %v, request_type: %s)", len(req.Models), isPartial, req.RequestType)

	// Configuration
	bucket := pred.GcsModelBucket
	basePath := pred.GcsModelBasePath

	if req.RequestType == "create" {
		for _, modelItem := range req.Models {
			modelName, err := p.extractModelName(modelItem.Metadata)
			if err != nil {
				return UploadModelFolderResponse{
					Message: "Failed to extract model name for existence check",
					Results: []ModelUploadResult{},
				}, http.StatusBadRequest, fmt.Errorf("failed to extract model name: %w", err)
			}

			// Check if model exists in predator_source_model bucket
			exists, err := p.GcsClient.CheckFolderExists(bucket, path.Join(basePath, modelName))
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check model existence for %s", modelName)
				return UploadModelFolderResponse{
					Message: fmt.Sprintf("Failed to check if model %s already exists", modelName),
					Results: []ModelUploadResult{},
				}, http.StatusInternalServerError, fmt.Errorf("failed to check model existence: %w", err)
			}

			if exists {
				return UploadModelFolderResponse{
					Message: fmt.Sprintf("Model %s already exists in predator_source_model bucket", modelName),
					Results: []ModelUploadResult{},
				}, http.StatusConflict, fmt.Errorf("model %s already exists", modelName)
			}
		}
	}

	// Process all models
	results := make([]ModelUploadResult, 0, len(req.Models))
	successCount := 0

	for i, modelItem := range req.Models {
		log.Info().Msgf("Processing model %d/%d", i+1, len(req.Models))

		result := p.uploadSingleModel(modelItem, bucket, basePath, isPartial, authToken)
		results = append(results, result)

		if result.Status == "success" {
			successCount++
		}
	}

	// Generate response with detailed error information
	failCount := len(req.Models) - successCount
	message, statusCode := p.generateUploadSummary(successCount, failCount, results)

	return UploadModelFolderResponse{
		Message: message,
		Results: results,
	}, statusCode, nil
}

// Legacy functions for backward compatibility
func (p *Predator) CheckModelExists(bucket, path string) (bool, error) {
	return p.GcsClient.CheckFolderExists(bucket, path)
}

func (p *Predator) UploadFileToGCS(bucket, path string, data []byte) error {
	return p.GcsClient.UploadFile(bucket, path, data)
}
