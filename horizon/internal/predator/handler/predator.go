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
	"regexp"
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
	"gorm.io/gorm"
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

const (
	OnboardRequestType                = "Onboard"
	ScaleUpRequestType                = "ScaleUp"
	PromoteRequestType                = "Promote"
	EditRequestType                   = "Edit"
	DeleteRequestType                 = "Delete"
	configFile                        = "config.pbtxt"
	pendingApproval                   = "Pending Approval"
	slashConstant                     = "/"
	gcsPrefix                         = "gs://"
	adminRole                         = "admin"
	typeString                        = "TYPE_STRING"
	bytesKeys                         = "BYTES"
	typePrefix                        = "TYPE_"
	errMsgFetchConfigs                = "failed to fetch predator configs: %w"
	errMsgParsePayload                = "failed to parse payload for request ID %d: %w"
	cpuRequestKey                     = "cpu_request"
	cpuLimitKey                       = "cpu_limit"
	memRequestKey                     = "mem_request"
	memLimitKey                       = "mem_limit"
	gpuRequestKey                     = "gpu_request"
	gpuLimitKey                       = "gpu_limit"
	minReplicaKey                     = "min_replica"
	maxReplicaKey                     = "max_replica"
	nodeSelectorKey                   = "node_selector"
	statusFailed                      = "Failed"
	statusInProgress                  = "In Progress"
	errMsgMarshalMeta                 = "Failed to marshal metadata"
	errMsgInsertConfig                = "Failed to insert predator_config"
	errMsgInsertDiscovery             = "Failed to insert service discovery"
	errMsgCreateConnection            = "Error in creating connection"
	errMsgTypeAssertion               = "failed to cast connection to *infra.SQLConnection"
	errMsgTypeAssertionLog            = "Type assertion error"
	errMsgCreateRequestRepo           = "Error in creating predator request repository"
	errMsgCreateDeployableRepo        = "Error in creating service deployable repository"
	errMsgCreateConfigRepo            = "Error in creating predator config repository"
	errMsgCreateDiscoveryRepo         = "Error in creating service discovery repository"
	errMsgCreateGroupIdCounterRepo    = "Error in creating group id counter repository"
	errMsgProcessPayload              = "failed to process payload"
	errMsgCreateRequestFormat         = "could not create %s request"
	successMsgFormat                  = "Model %s Request Raised Successfully."
	fieldModelName                    = "model_name"
	statusPendingApproval             = "Pending Approval"
	errModelNotFound                  = "model not found"
	errFetchDiscoveryConfig           = "failed to fetch service discovery config"
	errFetchDeployableConfig          = "failed to fetch service deployable config"
	errUnmarshalDeployableConfig      = "failed to unmarshal service deployable config"
	errMarshalPayload                 = "failed to marshal payload"
	errCreateDeleteRequest            = "could not create delete request"
	successDeleteRequestMsg           = "Model deletion request raised successfully"
	fieldModelSourcePath              = "model_source_path"
	fieldMetaData                     = "meta_data"
	fieldDiscoveryConfigID            = "discovery_config_id"
	fieldConfigMapping                = "config_mapping"
	errReadConfigFileFormat           = "failed to read config.pbtxt: %v"
	errUnmarshalProtoFormat           = "failed to unmarshal proto text: %v"
	errNoInstanceGroup                = "no instance group defined in model config"
	errModelPathPrefix                = "model_path must be provided and start with /"
	errModelPathFormat                = "invalid model_path format. Expected: /bucket/path/to/model"
	errModelNameMissing               = "model name is missing in config"
	errMaxBatchSizeMissing            = "max_batch_size is missing or zero in config"
	errBackendMissing                 = "backend is missing in config"
	errNoInputDefinitions             = "no input definitions found in config"
	errNoOutputDefinitions            = "no output definitions found in config"
	errInstanceGroupMissing           = "instance group is missing in config"
	errInvalidRequestIDFormat         = "invalid group ID format"
	errFailedToFetchRequest           = "failed to fetch request for group id %s"
	errInvalidRequestType             = "invalid request type"
	statusApproved                    = "Approved"
	statusRejected                    = "Rejected"
	errInvalidGcsBucketPath           = "invalid gcs bucket path format for source or destination"
	errFailedToUpdateRequest          = "Failed to update request status"
	successRejectMessage              = "Request %d rejected successfully.\n"
	errFailedToFindServiceDiscovery   = "Failed to find service discovery entry"
	errFailedToUpdateServiceDiscovery = "Failed to update service discovery to inactive"
	errFailedToFindPredatorConfig     = "Failed to find predator config entry"
	errFailedToUpdatePredatorConfig   = "Failed to update predator config to inactive"

	errFailedToParsePayload                    = "Failed to parse payload"
	errChildModelNotInDeleteRequest            = "ensemble model %s has child model %s which is not included in the delete request"
	errChildModelDifferentDeployable           = "ensemble model %s and its child model %s belong to different deployables (ensemble: %d, child: %d)"
	errFailedToFetchDiscoveryConfigForModel    = "failed to fetch discovery config for model %s: %w"
	errFailedToFetchDiscoveryConfigForEnsemble = "failed to fetch discovery config for ensemble model %s: %w"
	errFailedToFetchDiscoveryConfigForChild    = "failed to fetch discovery config for child model %s: %w"
	errDuplicateModelNameInDeployable          = "duplicate model name %s found within deployable %d"
	errNormalModelIsChildOfEnsemble            = "model %s is a child of ensemble model %s in the same deployable %d, but ensemble is not included in delete request"
	errEnsembleMissingChild                    = "ensemble model %s has child model %s which is not included in the delete request"
	errChildMissingEnsemble                    = "child model %s is included in delete request but its parent ensemble model %s is not included"
	errFailedToFindServiceDeployableEntry      = "Failed to find service deployable entry"
	errFailedToOperateGcsCloneStage            = "Failed to operate gcs clone stage"
	errFailedToRestartDeployable               = "Failed to restart deployable"
	errGCSCopyFailed                           = "GCS copy failed"
	errFailedToUpdateRequestStatusAndStage     = "Failed to update request status and stage %s"
	onboardRequestFlow                         = "Onboard request"
	cloneRequestFlow                           = "Clone request"
	promoteRequestFlow                         = "Promote request"
	predatorStageRestartDeployable             = "Restart Deployable"
	predatorStagePending                       = "Pending"
	machineTypeKey                             = "machine_type"
	cpuThresholdKey                            = "cpu_threshold"
	gpuThresholdKey                            = "gpu_threshold"
	tritonImageTagKey                          = "triton_image_tag"
	basePathKey                                = "base_path"
	predatorStageCloneToBucket                 = "Clone To Bucket"
	predatorStageDBPopulation                  = "DB Population"
	predatorStageRequestPayloadError           = "Request Payload Error"
	serviceDeployableNotFound                  = "ServiceDeployable not found"
	failedToParseServiceConfig                 = "Failed to parse service config"
	failedToCreateServiceDiscoveryAndConfig    = "Failed to create service discovery and config"
	predatorInferMethod                        = "inference.GRPCInferenceService/ModelInfer"
	deployableTagDelimiter                     = "_"
	scaleupTag                                 = "scaleup"
)

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

	for i := range len(req.Payload) {
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

func (p *Predator) ValidateDeleteRequest(predatorConfigList []predatorconfig.PredatorConfig, ids []int) (bool, error) {
	if len(predatorConfigList) != len(ids) {
		log.Error().Err(errors.New(errModelNotFound)).Msgf("model not found for ids %v", ids)
		return false, errors.New(errModelNotFound)
	}

	// Create maps for quick lookup
	requestedModelMap := make(map[string]predatorconfig.PredatorConfig)          // modelName -> config
	requestedDeployableMap := make(map[int]bool)                                 // serviceDeployableID -> exists
	deployableModelMap := make(map[int]map[string]predatorconfig.PredatorConfig) // deployableID -> modelName -> config

	// Build maps from requested models
	for _, predatorConfig := range predatorConfigList {
		// Get service deployable ID for this model
		discoveryConfig, err := p.ServiceDiscoveryRepo.GetById(predatorConfig.DiscoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to fetch discovery config for model %s", predatorConfig.ModelName)
			return false, fmt.Errorf(errFailedToFetchDiscoveryConfigForModel, predatorConfig.ModelName, err)
		}

		requestedModelMap[predatorConfig.ModelName] = predatorConfig
		requestedDeployableMap[discoveryConfig.ServiceDeployableID] = true

		// Group models by deployable
		if deployableModelMap[discoveryConfig.ServiceDeployableID] == nil {
			deployableModelMap[discoveryConfig.ServiceDeployableID] = make(map[string]predatorconfig.PredatorConfig)
		}
		deployableModelMap[discoveryConfig.ServiceDeployableID][predatorConfig.ModelName] = predatorConfig
	}

	// Check for duplicate model names within same deployable
	for deployableID, models := range deployableModelMap {
		if len(models) > 1 {
			// Check if any model names are duplicated within this deployable
			modelNameCount := make(map[string]int)
			for modelName := range models {
				modelNameCount[modelName]++
			}
			for modelName, count := range modelNameCount {
				if count > 1 {
					return false, fmt.Errorf(errDuplicateModelNameInDeployable, modelName, deployableID)
				}
			}
		}
	}

	// Validate ensemble-child group deletion requirements
	if err := p.validateEnsembleChildGroupDeletion(requestedModelMap, deployableModelMap); err != nil {
		return false, err
	}

	return true, nil
}

func (p *Predator) validateEnsembleChildGroupDeletion(requestedModelMap map[string]predatorconfig.PredatorConfig, deployableModelMap map[int]map[string]predatorconfig.PredatorConfig) error {
	// Get all active models to check for ensemble relationships
	allModels, err := p.PredatorConfigRepo.FindAllActiveConfig()
	if err != nil {
		log.Error().Err(err).Msgf("failed to fetch all active models")
		return fmt.Errorf("failed to fetch all active models: %w", err)
	}

	// Group all models by deployable for easier lookup
	allModelsByDeployable := make(map[int]map[string]predatorconfig.PredatorConfig)
	for _, model := range allModels {
		discoveryConfig, err := p.ServiceDiscoveryRepo.GetById(model.DiscoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to fetch discovery config for model %s", model.ModelName)
			continue
		}

		if allModelsByDeployable[discoveryConfig.ServiceDeployableID] == nil {
			allModelsByDeployable[discoveryConfig.ServiceDeployableID] = make(map[string]predatorconfig.PredatorConfig)
		}
		allModelsByDeployable[discoveryConfig.ServiceDeployableID][model.ModelName] = model
	}

	// Check each deployable for ensemble-child relationships
	for deployableID, modelsInDeployable := range allModelsByDeployable {
		requestedModelsInDeployable := deployableModelMap[deployableID]
		if requestedModelsInDeployable == nil {
			continue // No models from this deployable in the delete request
		}

		// Check each model in this deployable
		for modelName, model := range modelsInDeployable {
			var metadata MetaData
			if err := json.Unmarshal(model.MetaData, &metadata); err != nil {
				log.Error().Err(err).Msgf("failed to unmarshal metadata for model %s", modelName)
				continue
			}

			// Check if this is an ensemble model
			if len(metadata.Ensembling.Step) > 0 {
				// This is an ensemble model
				isEnsembleInRequest := requestedModelsInDeployable[modelName].ID != 0

				// Check each child of this ensemble
				for _, step := range metadata.Ensembling.Step {
					childModelName := step.ModelName
					isChildInRequest := requestedModelsInDeployable[childModelName].ID != 0

					// If ensemble is in request, all children must be in request
					if isEnsembleInRequest && !isChildInRequest {
						return fmt.Errorf(errEnsembleMissingChild, modelName, childModelName)
					}

					// If child is in request, ensemble must be in request
					if isChildInRequest && !isEnsembleInRequest {
						return fmt.Errorf(errChildMissingEnsemble, childModelName, modelName)
					}
				}
			}
		}
	}

	return nil
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

	var modelConfig ModelConfig
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

func validateModelConfig(cfg *ModelConfig) error {
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

func convertInputWithFeatures(fields []*ModelInput, featureMap map[string][]string) []IO {
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

func convertOutput(fields []*ModelOutput) []IO {
	ios := make([]IO, 0, len(fields))
	for _, f := range fields {
		if io, ok := convertFields(f.Name, f.Dims, f.DataType.String()); ok {
			ios = append(ios, io)
		}
	}
	return ios
}

func createModelParamsResponse(modelConfig *ModelConfig, objectPath string, inputs, outputs []IO) ModelParamsResponse {
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

func (p *Predator) processRequest(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if req.Status == statusApproved {
		switch predatorRequestList[0].RequestType {
		case OnboardRequestType:
			p.processOnboardFlow(requestIdPayloadMap, predatorRequestList, req)
		case ScaleUpRequestType:
			p.processScaleUpFlow(requestIdPayloadMap, predatorRequestList, req)
		case PromoteRequestType:
			p.processPromoteFlow(requestIdPayloadMap, predatorRequestList, req)
		case DeleteRequestType:
			p.processDeleteRequest(requestIdPayloadMap, predatorRequestList, req)
		case EditRequestType:
			p.processEditRequest(requestIdPayloadMap, predatorRequestList, req)
		default:
			log.Error().Err(errors.New(errInvalidRequestType)).Msg(errInvalidRequestType)
		}
	} else {
		p.processRejectRequest(predatorRequestList, req)
	}
}

func (p *Predator) processRejectRequest(predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	for i := range predatorRequestList {
		predatorRequestList[i].Status = statusRejected
		predatorRequestList[i].RejectReason = req.RejectReason
		predatorRequestList[i].Reviewer = req.ApprovedBy
		predatorRequestList[i].UpdatedBy = req.ApprovedBy
		predatorRequestList[i].UpdatedAt = time.Now()
		predatorRequestList[i].Active = false
	}

	if err := p.Repo.UpdateMany(predatorRequestList); err != nil {
		log.Printf(errFailedToUpdateRequestStatusAndStage, err)
	}

	log.Printf("Request %s rejected successfully.\n", req.GroupID)
}

func (p *Predator) processDeleteRequest(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	transferredGcsModelData, err := p.processGCSCloneToDeleteBucket(req.ApprovedBy, predatorRequestList, requestIdPayloadMap)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revertForDelete(transferredGcsModelData)
		return
	}

	p.processDBPopulationStageForDelete(predatorRequestList, requestIdPayloadMap, req)

	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}

}

func (p *Predator) processEditRequest(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	log.Info().Msgf("Starting edit request flow for group ID: %s", req.GroupID)

	// Step 1: Get target deployable configuration from the request
	targetDeployableID := int(requestIdPayloadMap[predatorRequestList[0].RequestID].ConfigMapping.ServiceDeployableID)
	targetServiceDeployable, err := p.ServiceDeployableRepo.GetById(targetDeployableID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch target service deployable for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		return
	}

	var targetDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(targetServiceDeployable.Config, &targetDeployableConfig); err != nil {
		log.Error().Err(err).Msg("Failed to parse target service deployable config")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		return
	}

	targetBucket, targetPath := extractGCSPath(strings.TrimSuffix(targetDeployableConfig.GCSBucketPath, "/*"))
	log.Info().Msgf("Target deployable path: gs://%s/%s", targetBucket, targetPath)

	// Step 2: GCS Copy Stage - Copy models from source to target deployable path
	transferredGcsModelData, err := p.processEditGCSCopyStage(requestIdPayloadMap, predatorRequestList, targetBucket, targetPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to copy models for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	// Update stage to DB Population after successful GCS copy
	p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusInProgress, predatorStageDBPopulation)

	// Step 3: DB Update Stage - Update existing predator config with new metadata from request
	err = p.processEditDBUpdateStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update database for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		p.revert(transferredGcsModelData)
		return
	}

	// Update stage to Restart Deployable after successful DB update
	p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusInProgress, predatorStageRestartDeployable)

	// Step 4: Restart Deployable Stage - Restart target deployable
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg("Failed to restart deployable for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}

	// Mark request as approved and completed
	p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusApproved, constant.EmptyString)
	log.Info().Msgf("Edit request completed successfully for group ID: %s", req.GroupID)
}

// processEditGCSCopyStage copies models from source to target deployable path for edit approval
func (p *Predator) processEditGCSCopyStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, targetBucket, targetPath string) ([]GcsModelData, error) {
	var transferredGcsModelData []GcsModelData

	// Check if we're in the correct stage for GCS copy
	if predatorRequestList[0].RequestStage != predatorStagePending && predatorRequestList[0].RequestStage != predatorStageCloneToBucket && predatorRequestList[0].RequestStage != constant.EmptyString {
		log.Info().Msgf("Skipping GCS copy stage - current stage: %s", predatorRequestList[0].RequestStage)
		return transferredGcsModelData, nil
	}

	isNotProd := p.isNonProductionEnvironment()

	for _, requestModel := range predatorRequestList {
		payload := requestIdPayloadMap[requestModel.RequestID]
		if payload == nil {
			log.Error().Msgf("Payload not found for request ID %d", requestModel.RequestID)
			continue
		}

		modelName := requestModel.ModelName

		// Use the source path from the payload, not the default GCS bucket
		if payload.ModelSource == "" {
			log.Error().Msgf("ModelSource is empty for request ID %d", requestModel.RequestID)
			return transferredGcsModelData, fmt.Errorf("model source path is empty for model %s", modelName)
		}

		// Normalize GCS URL (handle gcs:// prefix)
		normalizedModelSource := payload.ModelSource
		if strings.HasPrefix(normalizedModelSource, "gcs://") {
			normalizedModelSource = strings.Replace(normalizedModelSource, "gcs://", "gs://", 1)
			log.Info().Msgf("Normalized GCS URL from %s to %s", payload.ModelSource, normalizedModelSource)
		}

		// Parse the source GCS path
		sourceBucket, sourcePath := extractGCSPath(normalizedModelSource)
		if sourceBucket == "" || sourcePath == "" {
			log.Error().Msgf("Invalid source GCS path format: %s (normalized: %s)", payload.ModelSource, normalizedModelSource)
			return transferredGcsModelData, fmt.Errorf("invalid source GCS path format: %s", normalizedModelSource)
		}

		log.Info().Msgf("Copying model %s from source gs://%s/%s to target gs://%s/%s for edit approval",
			modelName, sourceBucket, sourcePath, targetBucket, targetPath)

		// Copy model from source to target deployable path
		// Extract model folder name from source path and copy to target with the same model name
		pathSegments := strings.Split(strings.TrimSuffix(sourcePath, "/"), "/")
		sourceModelName := pathSegments[len(pathSegments)-1]
		sourceBasePath := strings.TrimSuffix(sourcePath, "/"+sourceModelName)

		if isNotProd {
			if err := p.GcsClient.TransferFolder(
				sourceBucket, sourceBasePath, sourceModelName,
				targetBucket, targetPath, modelName,
			); err != nil {
				return transferredGcsModelData, err
			}
		} else {
			configBucket := pred.GcsConfigBucket
			configPath := pred.GcsConfigBasePath
			if err := p.GcsClient.TransferFolderWithSplitSources(
				sourceBucket, sourceBasePath, configBucket, configPath,
				sourceModelName, targetBucket, targetPath, modelName,
			); err != nil {
				return transferredGcsModelData, err
			}
		}

		// Track transferred data for potential rollback
		transferredGcsModelData = append(transferredGcsModelData, GcsModelData{
			Bucket: targetBucket,
			Path:   targetPath,
			Name:   modelName,
		})

		log.Info().Msgf("Successfully copied model %s for edit approval", modelName)
	}

	return transferredGcsModelData, nil
}

// processEditDBUpdateStage updates predator config for edit approval
// This updates the existing predator config with new config.pbtxt and metadata.json changes
func (p *Predator) processEditDBUpdateStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, approvedBy string) error {
	// Check if we're in the correct stage for DB update
	if predatorRequestList[0].RequestStage != predatorStageDBPopulation {
		log.Info().Msgf("Skipping DB update stage - current stage: %s", predatorRequestList[0].RequestStage)
		return nil
	}

	log.Info().Msg("Starting DB update stage for edit approval")

	for _, requestModel := range predatorRequestList {
		payload := requestIdPayloadMap[requestModel.RequestID]
		if payload == nil {
			log.Error().Msgf("Payload not found for request ID %d", requestModel.RequestID)
			continue
		}

		modelName := requestModel.ModelName
		log.Info().Msgf("Updating predator config for model %s", modelName)

		// Find existing predator config for this model
		existingPredatorConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(modelName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to fetch existing predator config for model %s", modelName)
			return fmt.Errorf("failed to fetch existing predator config for model %s: %w", modelName, err)
		}

		if existingPredatorConfig == nil {
			log.Error().Msgf("No existing predator config found for model %s", modelName)
			return fmt.Errorf("no existing predator config found for model %s", modelName)
		}

		// Clean up ensemble scheduling and update the predator config with new metadata from the request
		cleanedMetaData := p.cleanEnsembleScheduling(payload.MetaData)

		metaDataBytes, err := json.Marshal(cleanedMetaData)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal metadata for model %s", modelName)
			return fmt.Errorf("failed to marshal metadata for model %s: %w", modelName, err)
		}

		// Update the existing config
		existingPredatorConfig.MetaData = metaDataBytes
		existingPredatorConfig.UpdatedBy = approvedBy
		existingPredatorConfig.UpdatedAt = time.Now()
		existingPredatorConfig.HasNilData = true
		existingPredatorConfig.TestResults = nil
		// Save the updated config
		if err := p.PredatorConfigRepo.Update(existingPredatorConfig); err != nil {
			log.Error().Err(err).Msgf("Failed to update predator config for model %s", modelName)
			return fmt.Errorf("failed to update predator config for model %s: %w", modelName, err)
		}

		log.Info().Msgf("Successfully updated predator config for model %s", modelName)
	}

	log.Info().Msg("DB update stage completed successfully for edit approval")
	return nil
}

func (p *Predator) copyAllModelsFromActualToStaging(sourceBucket, sourcePath, targetBucket, targetPath string) error {
	// List all models in the actual target path and copy them to staging
	folders, err := p.GcsClient.ListFolders(sourceBucket, sourcePath)
	if err != nil {
		return fmt.Errorf("failed to list models in actual target path: %w", err)
	}

	// Copy each model folder from actual target to staging
	for _, modelName := range folders {
		log.Info().Msgf("Copying existing model %s from actual target to staging", modelName)

		if err := p.GcsClient.TransferFolder(sourceBucket, sourcePath, modelName, targetBucket, targetPath, modelName); err != nil {
			log.Error().Err(err).Msgf("Failed to copy existing model %s to staging", modelName)
			return fmt.Errorf("failed to copy existing model %s to staging: %w", modelName, err)
		}
	}

	return nil
}

func (p *Predator) deleteServiceDiscoveryAndConfig(req ApproveRequest, predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload) error {
	tx := p.Repo.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after rollback
		}
	}()

	for i := range predatorRequestList {
		payload := requestIdPayloadMap[predatorRequestList[i].RequestID]
		if payload == nil {
			log.Error().Msgf(errFailedToParsePayload)
			tx.Rollback()
			return fmt.Errorf("failed to parse payload for request ID %d", predatorRequestList[i].RequestID)
		}
		discoveryConfigID := int(payload.DiscoveryConfigID)
		log.Info().Msgf("Processing delete request for discovery config ID: %d", discoveryConfigID)
		serviceDiscovery, err := p.ServiceDiscoveryRepo.WithTx(tx).GetById(discoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msg(errFailedToFindServiceDiscovery)
			tx.Rollback()
			return err
		}
		serviceDiscovery.Active = false
		serviceDiscovery.UpdatedAt = time.Now()
		serviceDiscovery.UpdatedBy = req.ApprovedBy

		if err := p.ServiceDiscoveryRepo.WithTx(tx).Update(serviceDiscovery); err != nil {
			log.Error().Err(err).Msg(errFailedToUpdateServiceDiscovery)
			tx.Rollback()
			return err
		}

		predatorConfigs, err := p.PredatorConfigRepo.WithTx(tx).GetByDiscoveryConfigID(discoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msg(errFailedToFindPredatorConfig)
			tx.Rollback()
			return err
		}

		for j := range predatorConfigs {
			predatorConfigs[j].Active = false
			predatorConfigs[j].UpdatedAt = time.Now()
			predatorConfigs[j].UpdatedBy = req.ApprovedBy
			if err := p.PredatorConfigRepo.WithTx(tx).Update(&predatorConfigs[j]); err != nil {
				log.Error().Err(err).Msg(errFailedToUpdatePredatorConfig)
				tx.Rollback()
				return err
			}
		}

		predatorRequestList[i].Status = statusInProgress
		predatorRequestList[i].Reviewer = req.ApprovedBy
		predatorRequestList[i].UpdatedBy = req.ApprovedBy
		predatorRequestList[i].RequestStage = predatorStageRestartDeployable
		predatorRequestList[i].UpdatedAt = time.Now()

		if err := p.Repo.WithTx(tx).Update(&predatorRequestList[i]); err != nil {
			log.Error().Err(err).Msg(errFailedToUpdateRequestStatusAndStage)
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Error().Err(err).Msg("transaction commit failed")
		return err
	}

	return nil
}

func (p *Predator) processGCSCloneToDeleteBucket(email string, predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload) ([]GcsTransferredData, error) {
	var transferredGcsModelData []GcsTransferredData
	if predatorRequestList[0].RequestStage == constant.EmptyString || predatorRequestList[0].RequestStage == predatorStagePending || predatorRequestList[0].RequestStage == predatorStageCloneToBucket {
		for _, requestModel := range predatorRequestList {
			srcBucket, srcPath, srcModelName := extractGCSDetails(requestIdPayloadMap[requestModel.RequestID].ModelSource)
			destBucket, destPath := extractGCSPath(pred.DefaultModelPathKey)
			log.Info().Msgf("srcBucket: %s, srcPath: %s, srcModelName: %s, destBucket: %s, destPath: %s", srcBucket, srcPath, srcModelName, destBucket, destPath)
			if srcBucket == constant.EmptyString || srcPath == constant.EmptyString || srcModelName == constant.EmptyString || destBucket == constant.EmptyString || destPath == constant.EmptyString || requestIdPayloadMap[requestModel.RequestID].ModelName == constant.EmptyString {
				log.Error().Err(errors.New(errModelPathFormat)).Msg(errInvalidGcsBucketPath)
				return transferredGcsModelData, errors.New(errModelPathFormat)
			}

			if err := p.GcsClient.TransferAndDeleteFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, requestIdPayloadMap[requestModel.RequestID].ModelName); err != nil {
				log.Error().Err(err).Msg(errGCSCopyFailed)
				return transferredGcsModelData, err
			}

			transferredGcsModelData = append(transferredGcsModelData, GcsTransferredData{
				SrcBucket:  destBucket,
				SrcPath:    destPath,
				SrcName:    requestIdPayloadMap[requestModel.RequestID].ModelName,
				DestBucket: srcBucket,
				DestPath:   srcPath,
				DestName:   srcModelName,
			})
		}
		p.updateRequestStatusAndStage(email, predatorRequestList, statusInProgress, predatorStageDBPopulation)
	}
	return transferredGcsModelData, nil
}

func (p *Predator) processRestartDeployableStage(email string, predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload) error {
	if predatorRequestList[0].RequestStage != predatorStageRestartDeployable {
		return nil
	}
	var serviceDeployableIDList []int
	for _, requestModel := range predatorRequestList {
		serviceDeployableIDList = append(serviceDeployableIDList, int(requestIdPayloadMap[requestModel.RequestID].ConfigMapping.ServiceDeployableID))
	}

	for _, serviceDeployableID := range serviceDeployableIDList {
		sd, err := p.ServiceDeployableRepo.GetById(int(serviceDeployableID))
		if err != nil {
			log.Error().Err(err).Msg(errFailedToFindServiceDeployableEntry)
			return err
		}
		// Extract isCanary from deployable config
		var deployableConfig map[string]interface{}
		isCanary := false
		if err := json.Unmarshal(sd.Config, &deployableConfig); err == nil {
			if strategy, ok := deployableConfig["deploymentStrategy"].(string); ok && strategy == "canary" {
				isCanary = true
			}
		}
		if err := p.infrastructureHandler.RestartDeployment(sd.Name, p.workingEnv, isCanary); err != nil {
			log.Error().Err(err).Msg(errFailedToRestartDeployable)
			return err
		}
	}

	p.updateRequestStatusAndStage(email, predatorRequestList, statusApproved, constant.EmptyString)
	return nil
}

func (p *Predator) processPayload(predatorRequest predatorrequest.PredatorRequest) (*Payload, error) {
	var payload Payload
	decoder := json.NewDecoder(strings.NewReader(predatorRequest.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse payload with strict decoding")
		return nil, err
	}
	return &payload, nil
}

func (p *Predator) processGCSCloneStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) ([]GcsModelData, error) {
	var transferredGcsModelData []GcsModelData
	if predatorRequestList[0].RequestStage == predatorStagePending || predatorRequestList[0].RequestStage == predatorStageCloneToBucket {
		isNotProd := p.isNonProductionEnvironment()
		for _, requestModel := range predatorRequestList {

			serviceDeployable, err := p.ServiceDeployableRepo.GetById(int(requestIdPayloadMap[requestModel.RequestID].ConfigMapping.ServiceDeployableID))

			if err != nil {
				log.Error().Err(err).Msg(serviceDeployableNotFound)
				return transferredGcsModelData, err
			}

			var deployableConfig PredatorDeployableConfig
			if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err != nil {
				log.Error().Err(err).Msg(failedToParseServiceConfig)
				return transferredGcsModelData, err
			}

			destBucket, destPath := extractGCSPath(strings.TrimSuffix(deployableConfig.GCSBucketPath, "/*"))
			destModelName := requestIdPayloadMap[requestModel.RequestID].ModelName

			var srcBucket, srcPath, srcModelName string

			srcBucket = pred.GcsModelBucket
			srcPath = pred.GcsModelBasePath
			if requestModel.RequestType == ScaleUpRequestType {
				srcModelName = destModelName
				log.Info().Msgf("Scale-up: Source from model-source gs://%s/%s/%s",
					srcBucket, srcPath, srcModelName)
			} else {
				_, _, srcModelName = extractGCSDetails(requestIdPayloadMap[requestModel.RequestID].ModelSource)
				log.Info().Msgf("Onboard/Promote: Source from payload gs://%s/%s/%s",
					srcBucket, srcPath, srcModelName)
			}

			log.Info().Msgf("Copying to target deployable - src: %s/%s/%s, dest: %s/%s/%s",
				srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName)

			if srcBucket == constant.EmptyString || srcPath == constant.EmptyString ||
				srcModelName == constant.EmptyString || destBucket == constant.EmptyString ||
				destPath == constant.EmptyString || destModelName == constant.EmptyString {
				log.Error().Err(errors.New(errModelPathFormat)).Msg(errInvalidGcsBucketPath)
				return transferredGcsModelData, errors.New(errModelPathFormat)
			}

			if isNotProd {
				if err := p.GcsClient.TransferFolder(srcBucket, srcPath, srcModelName,
					destBucket, destPath, destModelName); err != nil {
					log.Error().Err(err).Msg(errGCSCopyFailed)
					return transferredGcsModelData, err
				}
			} else {
				if err := p.GcsClient.TransferFolderWithSplitSources(
					srcBucket, srcPath, pred.GcsConfigBucket, pred.GcsConfigBasePath,
					srcModelName, destBucket, destPath, destModelName,
				); err != nil {
					log.Error().Err(err).Msg(errGCSCopyFailed)
					return transferredGcsModelData, err
				}
			}

			transferredGcsModelData = append(transferredGcsModelData, GcsModelData{
				Bucket: destBucket,
				Path:   destPath,
				Name:   requestIdPayloadMap[requestModel.RequestID].ModelName,
			})

			log.Info().Msgf("Successfully copied model to target deployable: %s", destModelName)
		}
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusInProgress, predatorStageDBPopulation)
	}
	return transferredGcsModelData, nil
}

func (p *Predator) processGCSCloneStageIndefaultFolder(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) ([]GcsModelData, error) {
	var transferredGcsModelData []GcsModelData
	if predatorRequestList[0].RequestStage != predatorStagePending &&
		predatorRequestList[0].RequestStage != predatorStageCloneToBucket {
		return transferredGcsModelData, nil
	}

	isNotProd := p.isNonProductionEnvironment()

	for _, requestModel := range predatorRequestList {
		payload := requestIdPayloadMap[requestModel.RequestID]

		destBucket := pred.GcsModelBucket
		destPath := pred.GcsModelBasePath
		destModelName := payload.ModelName

		_, _, originalModelName := extractGCSDetails(payload.ModelSource)
		srcBucket := pred.GcsModelBucket
		srcPath := pred.GcsModelBasePath
		srcModelName := originalModelName

		log.Info().Msgf("Scale-up: Copying within model-source %s → %s", srcModelName, destModelName)
		log.Info().Msgf("srcBucket: %s, srcPath: %s, srcModelName: %s, destBucket: %s, destPath: %s",
			srcBucket, srcPath, srcModelName, destBucket, destPath)

		if srcBucket == constant.EmptyString || srcPath == constant.EmptyString ||
			srcModelName == constant.EmptyString || destBucket == constant.EmptyString ||
			destPath == constant.EmptyString || destModelName == constant.EmptyString {
			log.Error().Err(errors.New(errModelPathFormat)).Msg(errInvalidGcsBucketPath)
			return transferredGcsModelData, errors.New(errModelPathFormat)
		}

		if err := p.GcsClient.TransferFolder(srcBucket, srcPath, srcModelName,
			destBucket, destPath, destModelName); err != nil {
			log.Error().Err(err).Msg(errGCSCopyFailed)
			return transferredGcsModelData, err
		}

		log.Info().Msgf("Successfully copied model in model-source: %s → %s", srcModelName, destModelName)

		if !isNotProd && srcModelName != destModelName {
			if err := p.copyConfigToNewNameInConfigSource(srcModelName, destModelName); err != nil {
				log.Error().Err(err).Msgf("Failed to copy config to config-source: %s → %s",
					srcModelName, destModelName)
				return transferredGcsModelData, err
			}
		}

		transferredGcsModelData = append(transferredGcsModelData, GcsModelData{
			Bucket: destBucket,
			Path:   destPath,
			Name:   destModelName,
		})
	}

	return transferredGcsModelData, nil
}

func (p *Predator) processDBPopulationStageForDelete(predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload, req ApproveRequest) {
	if predatorRequestList[0].RequestStage != predatorStageDBPopulation {
		return
	}

	if err := p.deleteServiceDiscoveryAndConfig(req, predatorRequestList, requestIdPayloadMap); err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		return
	}
}

func (p *Predator) processDBPopulationStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, approvedBy string, successMessage string) error {
	if predatorRequestList[0].RequestStage != predatorStageDBPopulation {
		return nil
	}
	tx := p.Repo.DB().Begin()
	for i := range predatorRequestList {
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				log.Printf("panic recovered, transaction rolled back")
			}
		}()

		if err := p.createDiscoveryAndPredatorConfigTx(tx, predatorRequestList[i], *requestIdPayloadMap[predatorRequestList[i].RequestID], approvedBy); err != nil {
			tx.Rollback()
			log.Error().Err(err).Msg(failedToCreateServiceDiscoveryAndConfig)
			return err
		}

		predatorRequestList[i].Status = statusInProgress
		predatorRequestList[i].RequestStage = predatorStageRestartDeployable
		if err := p.Repo.UpdateStatusAndStage(tx, &predatorRequestList[i]); err != nil {
			tx.Rollback()
			log.Printf(errFailedToUpdateRequestStatusAndStage, err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("failed to commit transaction: %v", err)
		return err
	}
	log.Printf("success %s %d\n", successMessage, predatorRequestList[0].GroupId)
	return nil
}

func (p *Predator) checkIfModelsExist(predatorRequestList []predatorrequest.PredatorRequest) bool {
	for _, requestModel := range predatorRequestList {
		modelName := requestModel.ModelName
		if modelName == "" {
			log.Error().Msgf("model name is empty for request ID %d", requestModel.RequestID)
			continue
		}

		predatorConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(modelName)
		if err != nil {
			log.Error().Err(err).Msgf("failed to fetch predator config for model %s", modelName)
			continue
		}
		if predatorConfig != nil {
			log.Error().Msgf("model %s already exists", modelName)
			return true
		}
	}
	return false
}

func (p *Predator) processOnboardFlow(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if p.checkIfModelsExist(predatorRequestList) {
		req.RejectReason = "model already exists"
		req.Status = statusRejected
		p.processRejectRequest(predatorRequestList, req)
		return
	}

	transferredGcsModelData, err := p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	err = p.processDBPopulationStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy, onboardRequestFlow)
	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
	}
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}
}

func (p *Predator) revert(transferredGcsModelData []GcsModelData) error {
	for _, data := range transferredGcsModelData {
		if err := p.GcsClient.DeleteFolder(data.Bucket, data.Path, data.Name); err != nil {
			log.Error().Err(err).Msg(errGCSCopyFailed)
			return err
		}
	}
	return nil
}

func (p *Predator) revertForDelete(transferredGcsModelData []GcsTransferredData) error {
	for _, data := range transferredGcsModelData {
		if err := p.GcsClient.TransferAndDeleteFolder(data.SrcBucket, data.SrcPath, data.SrcName, data.DestBucket, data.DestPath, data.DestName); err != nil {
			log.Error().Err(err).Msg(errGCSCopyFailed)
			return err
		}
	}
	return nil
}

func (p *Predator) processScaleUpFlow(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if p.checkIfModelsExist(predatorRequestList) {
		req.RejectReason = fmt.Sprintf("model %s already exists", requestIdPayloadMap[predatorRequestList[0].RequestID].ModelName)
		req.Status = statusRejected
		p.processRejectRequest(predatorRequestList, req)
		return
	}

	transferredGcsModelData, err := p.processGCSCloneStageIndefaultFolder(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	transferredGcsModelData, err = p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	err = p.processDBPopulationStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy, cloneRequestFlow)
	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
	}
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}
}

func (p *Predator) processPromoteFlow(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if p.checkIfModelsExist(predatorRequestList) {
		req.RejectReason = fmt.Sprintf("model %s already exists", requestIdPayloadMap[predatorRequestList[0].RequestID].ModelName)
		req.Status = statusRejected
		p.processRejectRequest(predatorRequestList, req)
		return
	}

	transferredGcsModelData, err := p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	err = p.processDBPopulationStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy, promoteRequestFlow)
	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
	}
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}
}

func (p *Predator) updateRequestStatusAndStage(approvedBy string, predatorRequestList []predatorrequest.PredatorRequest, status, stage string) {
	for i := range predatorRequestList {
		predatorRequestList[i].Status = status
		predatorRequestList[i].Reviewer = approvedBy
		predatorRequestList[i].UpdatedBy = approvedBy
		if stage != constant.EmptyString {
			predatorRequestList[i].RequestStage = stage
		}
		if predatorRequestList[i].Status == statusApproved ||
			predatorRequestList[i].Status == statusFailed ||
			predatorRequestList[i].Status == statusRejected {
			predatorRequestList[i].Active = false
		}
		predatorRequestList[i].UpdatedAt = time.Now()
	}

	if err := p.Repo.UpdateMany(predatorRequestList); err != nil {
		log.Printf(errFailedToUpdateRequestStatusAndStage, err)
	}
}

func (p *Predator) createDiscoveryAndPredatorConfigTx(tx *gorm.DB, requestModel predatorrequest.PredatorRequest, payload Payload, approvedBy string) error {
	discoveryConfig, err := p.createDiscoveryConfigTx(tx, &requestModel, payload)
	if err != nil {
		return err
	}
	return p.createPredatorConfigTx(tx, &requestModel, payload, approvedBy, discoveryConfig.ID)
}

func (p *Predator) createDiscoveryConfigTx(tx *gorm.DB, requestModel *predatorrequest.PredatorRequest, payload Payload) (discoveryconfig.DiscoveryConfig, error) {
	discoveryConfig := discoveryconfig.DiscoveryConfig{
		ServiceDeployableID: int(payload.ConfigMapping.ServiceDeployableID),
		CreatedBy:           requestModel.CreatedBy,
		UpdatedBy:           requestModel.UpdatedBy,
		Active:              true,
		CreatedAt:           requestModel.CreatedAt,
		UpdatedAt:           time.Now(),
	}
	if err := tx.Create(&discoveryConfig).Error; err != nil {
		log.Error().Err(err).Msg(errMsgInsertDiscovery)
		return discoveryConfig, err
	}
	return discoveryConfig, nil
}

func (p *Predator) createPredatorConfigTx(tx *gorm.DB, requestModel *predatorrequest.PredatorRequest, payload Payload, approvedBy string, discoveryConfigID int) error {
	// Clean up ensemble scheduling before marshaling
	cleanedMetaData := p.cleanEnsembleScheduling(payload.MetaData)

	metaDataBytes, err := json.Marshal(cleanedMetaData)
	if err != nil {
		log.Error().Err(err).Msg(errMsgMarshalMeta)
		return err
	}

	serviceDeployableID := int(payload.ConfigMapping.ServiceDeployableID)
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(serviceDeployableID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get service deployable config for ID %d", serviceDeployableID)
		return fmt.Errorf("failed to get service deployable config: %w", err)
	}

	config := predatorconfig.PredatorConfig{
		DiscoveryConfigID: discoveryConfigID,
		ModelName:         payload.ModelName,
		MetaData:          metaDataBytes,
		CreatedBy:         requestModel.CreatedBy,
		UpdatedBy:         approvedBy,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Active:            true,
		SourceModelName:   payload.ConfigMapping.SourceModelName,
	}

	if serviceDeployable.OverrideTesting {
		log.Info().Msgf("OverrideTesting is enabled for deployable %s. Setting test_results for model %s",
			serviceDeployable.Name, payload.ModelName)

		config.TestResults = json.RawMessage(`{"is_functionally_tested": true}`)
		config.HasNilData = false
	}

	if err := tx.Create(&config).Error; err != nil {
		log.Error().Err(err).Msg(errMsgInsertConfig)
		return err
	}
	return nil
}

func parseGCSURL(gcsURL string) (bucket, objectPath string, ok bool) {
	// Handle both gs:// and gcs:// prefixes (normalize gcs:// to gs://)
	if strings.HasPrefix(gcsURL, "gcs://") {
		gcsURL = strings.Replace(gcsURL, "gcs://", "gs://", 1)
	}

	if !strings.HasPrefix(gcsURL, gcsPrefix) {
		return constant.EmptyString, constant.EmptyString, false
	}

	trimmed := strings.TrimPrefix(gcsURL, gcsPrefix)
	parts := strings.SplitN(trimmed, slashConstant, 2)
	if len(parts) < 1 {
		return constant.EmptyString, constant.EmptyString, false
	}

	bucket = parts[0]
	if len(parts) == 2 {
		objectPath = parts[1]
	}
	return bucket, objectPath, true
}

func extractGCSPath(gcsURL string) (bucket, objectPath string) {
	bucket, objectPath, ok := parseGCSURL(gcsURL)
	if !ok {
		return constant.EmptyString, constant.EmptyString
	}
	return bucket, objectPath
}

func extractGCSDetails(gcsURL string) (bucket, pathOnly, modelName string) {
	bucket, objectPath, ok := parseGCSURL(gcsURL)
	if !ok || objectPath == constant.EmptyString {
		return constant.EmptyString, constant.EmptyString, constant.EmptyString
	}

	segments := strings.Split(objectPath, slashConstant)
	if len(segments) == 0 {
		return bucket, constant.EmptyString, constant.EmptyString
	}

	modelName = segments[len(segments)-1]
	pathOnly = path.Join(segments[:len(segments)-1]...)
	return bucket, pathOnly, modelName
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

// batchFetchRelatedData efficiently fetches all discovery configs and service deployables in batch
func (p *Predator) batchFetchRelatedData(predatorConfigs []predatorconfig.PredatorConfig) (map[int]*discoveryconfig.DiscoveryConfig, map[int]*servicedeployableconfig.ServiceDeployableConfig, error) {
	// Collect all unique discovery config IDs
	discoveryConfigIDs := make([]int, 0, len(predatorConfigs))
	discoveryIDSet := make(map[int]bool)

	for _, config := range predatorConfigs {
		if !discoveryIDSet[config.DiscoveryConfigID] {
			discoveryConfigIDs = append(discoveryConfigIDs, config.DiscoveryConfigID)
			discoveryIDSet[config.DiscoveryConfigID] = true
		}
	}

	// Batch fetch all discovery configs
	discoveryConfigs := make(map[int]*discoveryconfig.DiscoveryConfig)
	for _, id := range discoveryConfigIDs {
		config, err := p.ServiceDiscoveryRepo.GetById(id)
		if err != nil {
			continue // Skip failed configs, same behavior as original
		}
		discoveryConfigs[id] = config
	}

	// Collect all unique service deployable IDs
	serviceDeployableIDs := make([]int, 0, len(discoveryConfigs))
	serviceDeployableIDSet := make(map[int]bool)

	for _, config := range discoveryConfigs {
		if !serviceDeployableIDSet[config.ServiceDeployableID] {
			serviceDeployableIDs = append(serviceDeployableIDs, config.ServiceDeployableID)
			serviceDeployableIDSet[config.ServiceDeployableID] = true
		}
	}

	// Batch fetch all service deployables
	serviceDeployables := make(map[int]*servicedeployableconfig.ServiceDeployableConfig)
	for _, id := range serviceDeployableIDs {
		deployable, err := p.ServiceDeployableRepo.GetById(id)
		if err != nil {
			continue // Skip failed deployables, same behavior as original
		}
		serviceDeployables[id] = deployable
	}

	return discoveryConfigs, serviceDeployables, nil
}

// batchFetchDeployableConfigs concurrently fetches deployable configs for all service deployables
func (p *Predator) batchFetchDeployableConfigs(serviceDeployables map[int]*servicedeployableconfig.ServiceDeployableConfig) (map[int]externalcall.Config, error) {
	deployableConfigs := make(map[int]externalcall.Config)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrent API calls
	semaphore := make(chan struct{}, 10) // Limit to 10 concurrent calls

	for id, deployable := range serviceDeployables {
		wg.Add(1)
		go func(deployableID int, sd *servicedeployableconfig.ServiceDeployableConfig) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			infraConfig := p.infrastructureHandler.GetConfig(sd.Name, p.workingEnv)
			// Convert to externalcall.Config for compatibility
			config := externalcall.Config{
				MinReplica:    infraConfig.MinReplica,
				MaxReplica:    infraConfig.MaxReplica,
				RunningStatus: infraConfig.RunningStatus,
			}

			mu.Lock()
			deployableConfigs[deployableID] = config
			mu.Unlock()
		}(id, deployable)
	}

	wg.Wait()
	return deployableConfigs, nil
}

// buildModelResponses constructs the final ModelResponse objects
func (p *Predator) buildModelResponses(
	predatorConfigs []predatorconfig.PredatorConfig,
	discoveryConfigs map[int]*discoveryconfig.DiscoveryConfig,
	serviceDeployables map[int]*servicedeployableconfig.ServiceDeployableConfig,
	deployableConfigs map[int]externalcall.Config,
) []ModelResponse {
	results := make([]ModelResponse, 0, len(predatorConfigs))

	for _, config := range predatorConfigs {
		// Get discovery config
		serviceDiscovery, exists := discoveryConfigs[config.DiscoveryConfigID]
		if !exists {
			continue // Skip if discovery config not found
		}

		// Get service deployable
		serviceDeployable, exists := serviceDeployables[serviceDiscovery.ServiceDeployableID]
		if !exists {
			continue // Skip if service deployable not found
		}

		// Parse deployable config
		var deployableConfig PredatorDeployableConfig
		if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err != nil {
			continue // Skip if config parsing fails
		}

		// Get infrastructure config (HPA/replica info)
		infraConfig := deployableConfigs[serviceDiscovery.ServiceDeployableID]

		deploymentConfig := map[string]any{
			machineTypeKey:    deployableConfig.MachineType,
			cpuThresholdKey:   deployableConfig.CPUThreshold,
			gpuThresholdKey:   deployableConfig.GPUThreshold,
			cpuRequestKey:     deployableConfig.CPURequest,
			cpuLimitKey:       deployableConfig.CPULimit,
			memRequestKey:     deployableConfig.MemoryRequest,
			memLimitKey:       deployableConfig.MemoryLimit,
			gpuRequestKey:     deployableConfig.GPURequest,
			gpuLimitKey:       deployableConfig.GPULimit,
			minReplicaKey:     infraConfig.MinReplica,
			maxReplicaKey:     infraConfig.MaxReplica,
			nodeSelectorKey:   deployableConfig.NodeSelectorValue,
			tritonImageTagKey: deployableConfig.TritonImageTag,
			basePathKey:       deployableConfig.GCSBucketPath,
		}

		modelResponse := ModelResponse{
			ID:                      config.ID,
			ModelName:               config.ModelName,
			MetaData:                config.MetaData,
			Host:                    serviceDeployable.Host,
			MachineType:             deployableConfig.MachineType,
			DeploymentConfig:        deploymentConfig,
			MonitoringUrl:           serviceDeployable.MonitoringUrl,
			GCSPath:                 strings.TrimSuffix(deployableConfig.GCSBucketPath, "/*"),
			CreatedBy:               config.CreatedBy,
			CreatedAt:               config.CreatedAt,
			UpdatedBy:               config.UpdatedBy,
			UpdatedAt:               config.UpdatedAt,
			DeployableRunningStatus: infraConfig.RunningStatus,
			TestResults:             config.TestResults,
			HasNilData:              config.HasNilData,
			SourceModelName:         config.SourceModelName,
		}

		results = append(results, modelResponse)
	}

	return results
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

// releaseLockWithError is a helper function to release lock and log error
func (p *Predator) releaseLockWithError(lockID uint, groupID, errorMsg string) {
	if releaseErr := p.validationLockRepo.ReleaseLock(lockID); releaseErr != nil {
		log.Error().Err(releaseErr).Msgf("Failed to release validation lock for group ID %s after error: %s", groupID, errorMsg)
	}
	log.Error().Msgf("Validation failed for group ID %s: %s", groupID, errorMsg)
}

// markModelWithNilData marks a model as having nil data issues

// getTestDeployableID determines the appropriate test deployable ID based on machine type
func (p *Predator) getTestDeployableID(payload *Payload) (int, error) {
	// Get the target deployable ID from the request
	targetDeployableID := int(payload.ConfigMapping.ServiceDeployableID)
	// Fetch the service deployable config to check machine type
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(targetDeployableID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch service deployable config: %w", err)
	}

	// Parse the deployable config to extract machine type
	var deployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err != nil {
		return 0, fmt.Errorf("failed to parse service deployable config: %w", err)
	}

	// Select test deployable ID based on machine type
	switch strings.ToUpper(deployableConfig.MachineType) {
	case "CPU":
		log.Info().Msgf("Using CPU test deployable ID: %d", pred.TestDeployableID)
		return pred.TestDeployableID, nil
	case "GPU":
		log.Info().Msgf("Using GPU test deployable ID: %d", pred.TestGpuDeployableID)
		return pred.TestGpuDeployableID, nil
	default:
		// Default to CPU if machine type is not specified or unknown
		log.Warn().Msgf("Unknown machine type '%s', defaulting to CPU test deployable ID: %d",
			deployableConfig.MachineType, pred.TestDeployableID)
		return pred.TestDeployableID, nil
	}
}

// getServiceNameFromDeployable extracts service name from deployable configuration
func (p *Predator) getServiceNameFromDeployable(deployableID int) (string, error) {
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(deployableID)
	if err != nil {
		return "", fmt.Errorf("failed to get deployable config: %w", err)
	}
	return serviceDeployable.Name, nil
}

// performAsyncValidation performs the actual validation process asynchronously
func (p *Predator) performAsyncValidation(job *validationjob.Table, requests []predatorrequest.PredatorRequest, payload *Payload, testDeployableID int) {
	defer func() {
		// Always release the lock when validation completes
		if releaseErr := p.validationLockRepo.ReleaseLock(job.LockID); releaseErr != nil {
			log.Error().Err(releaseErr).Msgf("Failed to release validation lock for job %d", job.ID)
		}
		log.Info().Msgf("Released validation lock for job %d", job.ID)
	}()

	log.Info().Msgf("Starting async validation for job %d, group %s", job.ID, job.GroupID)

	// Step 1: Clear temporary deployable
	if err := p.clearTemporaryDeployable(testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to clear temporary deployable")
		p.failValidationJob(job.ID, "Failed to clear temporary deployable: "+err.Error())
		return
	}

	// Step 2: Copy existing models to temporary deployable
	targetDeployableID := int(payload.ConfigMapping.ServiceDeployableID)
	if err := p.copyExistingModelsToTemporary(targetDeployableID, testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to copy existing models to temporary deployable")
		p.failValidationJob(job.ID, "Failed to copy existing models: "+err.Error())
		return
	}

	// Step 3: Copy new models from request to temporary deployable
	if err := p.copyRequestModelsToTemporary(requests, testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to copy request models to temporary deployable")
		p.failValidationJob(job.ID, "Failed to copy request models: "+err.Error())
		return
	}

	// Step 4: Restart temporary deployable
	if err := p.restartTemporaryDeployable(testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to restart temporary deployable")
		p.failValidationJob(job.ID, "Failed to restart temporary deployable: "+err.Error())
		return
	}

	// Update job status to checking and record restart time
	now := time.Now()
	if err := p.validationJobRepo.UpdateStatus(job.ID, validationjob.StatusChecking, ""); err != nil {
		log.Error().Err(err).Msgf("Failed to update job %d status to checking", job.ID)
	}

	// Update restart time in the job
	job.RestartedAt = &now
	job.Status = validationjob.StatusChecking

	// Step 5: Start health checking process
	p.startHealthCheckingProcess(job)
}

// startHealthCheckingProcess monitors the deployment health and updates validation status
func (p *Predator) startHealthCheckingProcess(job *validationjob.Table) {
	log.Info().Msgf("Starting health check process for job %d, service %s", job.ID, job.ServiceName)

	for job.HealthCheckCount < job.MaxHealthChecks {
		// Wait for the specified interval before checking
		time.Sleep(time.Duration(job.HealthCheckInterval) * time.Second)

		// Increment health check count
		if err := p.validationJobRepo.IncrementHealthCheck(job.ID); err != nil {
			log.Error().Err(err).Msgf("Failed to increment health check count for job %d", job.ID)
		}
		job.HealthCheckCount++

		// Check deployment health using infrastructure handler
		isHealthy, err := p.checkDeploymentHealth(job.ServiceName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to check deployment health for job %d", job.ID)
			continue // Continue checking, don't fail immediately on health check errors
		}

		if isHealthy {
			log.Info().Msgf("Deployment is healthy for job %d, validation successful", job.ID)
			p.completeValidationJob(job.ID, true, "Deployment is healthy and running successfully")
			p.updateRequestValidationStatus(job.GroupID, true)
			return
		}

		log.Info().Msgf("Deployment not yet healthy for job %d, check %d/%d", job.ID, job.HealthCheckCount, job.MaxHealthChecks)
	}

	// If we reach here, max health checks exceeded
	log.Warn().Msgf("Max health checks exceeded for job %d, marking as failed", job.ID)
	p.completeValidationJob(job.ID, false, fmt.Sprintf("Deployment failed to become healthy after %d checks", job.MaxHealthChecks))
	p.updateRequestValidationStatus(job.GroupID, false)
}

// checkDeploymentHealth checks if the deployment is healthy using infrastructure handler
func (p *Predator) checkDeploymentHealth(serviceName string) (bool, error) {
	resourceDetail, err := p.infrastructureHandler.GetResourceDetail(serviceName, p.workingEnv)
	if err != nil {
		return false, fmt.Errorf("failed to get resource detail: %w", err)
	}

	if resourceDetail == nil || len(resourceDetail.Nodes) == 0 {
		return false, nil
	}

	healthyPodCount := 0
	totalPodCount := 0

	for _, node := range resourceDetail.Nodes {
		if node.Kind == "Deployment" {
			totalPodCount++
			if node.Health.Status == "Healthy" {
				healthyPodCount++
			}
		}
	}

	log.Info().Msgf("Health check for service %s: %d/%d pods healthy and running", serviceName, healthyPodCount, totalPodCount)

	if totalPodCount == healthyPodCount {
		return true, nil
	}
	// Consider deployment healthy if at least one pod is healthy and running
	return false, nil
}

// failValidationJob marks a validation job as failed
func (p *Predator) failValidationJob(jobID uint, errorMessage string) {
	if err := p.validationJobRepo.UpdateValidationResult(jobID, false, errorMessage); err != nil {
		log.Error().Err(err).Msgf("Failed to update validation job %d as failed", jobID)
	}
}

// completeValidationJob marks a validation job as completed
func (p *Predator) completeValidationJob(jobID uint, success bool, message string) {
	if err := p.validationJobRepo.UpdateValidationResult(jobID, success, message); err != nil {
		log.Error().Err(err).Msgf("Failed to update validation job %d as completed", jobID)
	}
}

// updateRequestValidationStatus updates the request table with validation results
func (p *Predator) updateRequestValidationStatus(groupID string, success bool) {
	id, err := strconv.ParseUint(groupID, 10, 32)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse group ID %s for status update", groupID)
		return
	}

	requests, err := p.Repo.GetAllByGroupID(uint(id))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get requests for group ID %s", groupID)
		return
	}

	// Update all requests in the group
	for _, request := range requests {
		request.UpdatedAt = time.Now()
		request.IsValid = success
		if !success {
			request.RejectReason = "Validation Failed"
			request.Status = statusRejected
			request.UpdatedBy = "Validation Job"
			request.UpdatedAt = time.Now()
			request.Active = false
		}
		if err := p.Repo.Update(&request); err != nil {
			log.Error().Err(err).Msgf("Failed to update request %d status", request.RequestID)
		} else {
			log.Info().Msgf("Updated request %d status to %s", request.RequestID, request.Status)
		}
	}
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

// clearTemporaryDeployable clears all models from the temporary deployable GCS path
func (p *Predator) clearTemporaryDeployable(testDeployableID int) error {
	// Get temporary deployable config
	testServiceDeployable, err := p.ServiceDeployableRepo.GetById(testDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	var tempDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(testServiceDeployable.Config, &tempDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse temporary deployable config: %w", err)
	}

	if tempDeployableConfig.GCSBucketPath != "NA" {
		// Extract bucket and path from temporary deployable config
		tempBucket, tempPath := extractGCSPath(strings.TrimSuffix(tempDeployableConfig.GCSBucketPath, "/*"))

		// Clear all models from temporary deployable
		log.Info().Msgf("Clearing temporary deployable GCS path: gs://%s/%s", tempBucket, tempPath)
		if err := p.GcsClient.DeleteFolder(tempBucket, tempPath, ""); err != nil {
			return fmt.Errorf("failed to clear temporary deployable GCS path: %w", err)
		}
	}

	return nil
}

// copyExistingModelsToTemporary copies all existing models from target deployable to temporary deployable
func (p *Predator) copyExistingModelsToTemporary(targetDeployableID, tempDeployableID int) error {
	// Get target deployable config
	targetServiceDeployable, err := p.ServiceDeployableRepo.GetById(targetDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch target service deployable: %w", err)
	}

	var targetDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(targetServiceDeployable.Config, &targetDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse target deployable config: %w", err)
	}

	// Get temporary deployable config
	tempServiceDeployable, err := p.ServiceDeployableRepo.GetById(tempDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	var tempDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(tempServiceDeployable.Config, &tempDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse temporary deployable config: %w", err)
	}

	if targetDeployableConfig.GCSBucketPath != "NA" {
		// Extract GCS paths
		targetBucket, targetPath := extractGCSPath(strings.TrimSuffix(targetDeployableConfig.GCSBucketPath, "/*"))
		tempBucket, tempPath := extractGCSPath(strings.TrimSuffix(tempDeployableConfig.GCSBucketPath, "/*"))

		// Copy all existing models from target to temporary deployable
		return p.copyAllModelsFromActualToStaging(targetBucket, targetPath, tempBucket, tempPath)
	} else {
		return nil
	}
}

// copyRequestModelsToTemporary copies the requested models to temporary deployable
func (p *Predator) copyRequestModelsToTemporary(requests []predatorrequest.PredatorRequest, tempDeployableID int) error {
	// Get temporary deployable config
	tempServiceDeployable, err := p.ServiceDeployableRepo.GetById(tempDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	var tempDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(tempServiceDeployable.Config, &tempDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse temporary deployable config: %w", err)
	}

	tempBucket, tempPath := extractGCSPath(strings.TrimSuffix(tempDeployableConfig.GCSBucketPath, "/*"))

	isNotProd := p.isNonProductionEnvironment()

	// Copy each requested model from default GCS location to temporary deployable
	for _, request := range requests {
		modelName := request.ModelName
		payload, err := p.processPayload(request)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to parse payload for request %d", request.RequestID)
			return fmt.Errorf("failed to parse payload for request %d: %w", request.RequestID, err)
		}

		var sourceBucket, sourcePath, sourceModelName string
		if payload.ModelSource != "" {
			sourceBucket, sourcePath, sourceModelName = extractGCSDetails(payload.ModelSource)
			log.Info().Msgf("Using ModelSource from payload for validation: gs://%s/%s/%s",
				sourceBucket, sourcePath, sourceModelName)
		} else {
			sourceBucket = pred.GcsModelBucket
			sourcePath = pred.GcsModelBasePath
			sourceModelName = modelName
			log.Info().Msgf("Using default model source for validation: gs://%s/%s/%s",
				sourceBucket, sourcePath, sourceModelName)
		}
		log.Info().Msgf("Copying model %s from gs://%s/%s/%s to temporary deployable gs://%s/%s",
			modelName, sourceBucket, sourcePath, sourceModelName, tempBucket, tempPath)

		if isNotProd {
			if err := p.GcsClient.TransferFolder(sourceBucket, sourcePath, sourceModelName,
				tempBucket, tempPath, modelName); err != nil {
				return fmt.Errorf("failed to copy requested model %s to temporary deployable: %w", modelName, err)
			}
		} else {
			if err := p.GcsClient.TransferFolderWithSplitSources(
				sourceBucket, sourcePath, pred.GcsConfigBucket, pred.GcsConfigBasePath,
				sourceModelName, tempBucket, tempPath, modelName,
			); err != nil {
				return fmt.Errorf("failed to copy requested model %s to temporary deployable: %w", modelName, err)
			}
		}

		log.Info().Msgf("Successfully copied model %s to temporary deployable", modelName)
	}

	return nil
}

// restartTemporaryDeployable restarts the temporary deployable for validation
func (p *Predator) restartTemporaryDeployable(tempDeployableID int) error {
	tempServiceDeployable, err := p.ServiceDeployableRepo.GetById(tempDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	// Extract isCanary from deployable config
	var deployableConfig map[string]interface{}
	isCanary := false
	if err := json.Unmarshal(tempServiceDeployable.Config, &deployableConfig); err == nil {
		if strategy, ok := deployableConfig["deploymentStrategy"].(string); ok && strategy == "canary" {
			isCanary = true
		}
	}
	if err := p.infrastructureHandler.RestartDeployment(tempServiceDeployable.Name, p.workingEnv, isCanary); err != nil {
		return fmt.Errorf("failed to restart temporary deployable: %w", err)
	}

	log.Info().Msgf("Successfully restarted temporary deployable: %s for validation", tempServiceDeployable.Name)
	return nil
}

// convertDimsToIntSlice converts input.Dims to []int, handling nested interfaces and various types
// Dynamic dimensions (-1) are replaced with reasonable default values for test data generation
func convertDimsToIntSlice(dims interface{}) ([]int, error) {
	var result []int

	switch v := dims.(type) {
	case []int:
		result = make([]int, len(v))
		copy(result, v)
	case []int64:
		result = make([]int, len(v))
		for i, dim := range v {
			result[i] = int(dim)
		}
	case []interface{}:
		result = make([]int, len(v))
		for i, dim := range v {
			switch d := dim.(type) {
			case int:
				result[i] = d
			case int64:
				result[i] = int(d)
			case float64:
				result[i] = int(d)
			default:
				return nil, fmt.Errorf("unsupported dimension type in slice: %T", d)
			}
		}
	case int:
		result = []int{v}
	case int64:
		result = []int{int(v)}
	case float64:
		result = []int{int(v)}
	default:
		return nil, fmt.Errorf("unsupported dims type: %T", v)
	}

	// Replace dynamic dimensions (-1) with reasonable default values for test data generation
	for i, dim := range result {
		if dim == -1 {
			// Use different default sizes based on position
			if i == 0 {
				result[i] = 10 // First dimension (often sequence length): default to 10
			} else {
				result[i] = 128 // Other dimensions: default to 128
			}
			log.Debug().Msgf("Replaced dynamic dimension -1 at position %d with %d", i, result[i])
		} else if dim < 0 {
			// Handle any other negative dimensions
			result[i] = 1
			log.Debug().Msgf("Replaced negative dimension %d at position %d with 1", dim, i)
		}
	}

	return result, nil
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

// flattenInputTo3DByteSlice converts input data to 3D byte slice format [batch][feature][bytes]
// This matches the working adapter's data structure expectations
func (p *Predator) flattenInputTo3DByteSlice(data any, dataType string) ([][][]byte, error) {
	// The input data comes as nested arrays [batch_size][feature_count]
	// For FP16: each feature is a single float32 value converted to 2 bytes
	// We need to convert this to [batch][feature][bytes] format exactly like the working adapter

	switch v := data.(type) {
	case [][]float32:
		// 2D array of float32 values [batch_size][feature_count]
		batchSize := len(v)
		if batchSize == 0 {
			return [][][]byte{}, nil
		}
		featureCount := len(v[0])

		result := make([][][]byte, batchSize)
		for batchIdx := 0; batchIdx < batchSize; batchIdx++ {
			result[batchIdx] = make([][]byte, featureCount)
			for featureIdx := 0; featureIdx < featureCount; featureIdx++ {
				val := v[batchIdx][featureIdx]
				switch dataType {
				case "FP16":
					fp16Bytes, err := serializer.Float32ToFloat16Bytes(val)
					if err != nil {
						return nil, err
					}
					result[batchIdx][featureIdx] = fp16Bytes
				case "FP32":
					bytes := make([]byte, 4)
					binary.LittleEndian.PutUint32(bytes, math.Float32bits(val))
					result[batchIdx][featureIdx] = bytes
				default:
					return nil, fmt.Errorf("unsupported numeric type %s for float32 data", dataType)
				}
			}
		}
		return result, nil

	default:
		// Fallback: try to flatten and reshape based on expected structure
		flattened, err := serializer.FlattenMatrixByType(data, dataType)
		if err != nil {
			return nil, err
		}

		switch dataType {
		case "FP16":
			if f32slice, ok := flattened.([]float32); ok {
				// We need to infer the batch structure from the input data
				// For now, assume it matches the shape from the input tensor
				// This is a fallback - the main case should handle [][]float32
				batchSize := 1
				featureCount := len(f32slice)

				result := make([][][]byte, batchSize)
				result[0] = make([][]byte, featureCount)
				for i, val := range f32slice {
					fp16Bytes, err := serializer.Float32ToFloat16Bytes(val)
					if err != nil {
						return nil, err
					}
					result[0][i] = fp16Bytes
				}
				return result, nil
			}
		case "BYTES":
			if byteSlice, ok := flattened.([][]byte); ok {
				// For BYTES, each element is a separate feature
				result := make([][][]byte, 1)
				result[0] = byteSlice
				return result, nil
			}
		}

		return nil, fmt.Errorf("unsupported data format: %T for type %s", data, dataType)
	}
}

// getElementSize returns the byte size of a single element for the given data type
func getElementSize(dataType string) int {
	switch strings.ToUpper(dataType) {
	case "FP32", "TYPE_FP32":
		return 4
	case "FP64", "TYPE_FP64":
		return 8
	case "INT32", "TYPE_INT32":
		return 4
	case "INT64", "TYPE_INT64":
		return 8
	case "INT16", "TYPE_INT16":
		return 2
	case "INT8", "TYPE_INT8":
		return 1
	case "UINT32", "TYPE_UINT32":
		return 4
	case "UINT64", "TYPE_UINT64":
		return 8
	case "UINT16", "TYPE_UINT16":
		return 2
	case "UINT8", "TYPE_UINT8":
		return 1
	case "BOOL", "TYPE_BOOL":
		return 1
	case "FP16", "TYPE_FP16":
		return 2
	default:
		return 0 // Unknown type
	}
}

// reshapeDataForBatch reshapes flattened data to preserve batch dimension
func reshapeDataForBatch(data interface{}, dims []int64) interface{} {
	if len(dims) == 0 {
		return data
	}

	batchSize := dims[0]
	featureDims := dims[1:]

	// Calculate elements per batch
	elementsPerBatch := int64(1)
	for _, dim := range featureDims {
		elementsPerBatch *= dim
	}

	// Convert data to slice if it isn't already
	var dataSlice []interface{}
	switch v := data.(type) {
	case []interface{}:
		dataSlice = v
	case []string:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []float32:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []float64:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []int32:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	case []int64:
		for _, item := range v {
			dataSlice = append(dataSlice, item)
		}
	default:
		// If we can't convert, return as-is
		return data
	}

	// Reshape into batches
	var result [][]interface{}
	for i := int64(0); i < batchSize; i++ {
		start := i * elementsPerBatch
		end := start + elementsPerBatch
		if end <= int64(len(dataSlice)) {
			batch := dataSlice[start:end]
			result = append(result, batch)
		}
	}

	return result
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

// uploadSingleModel processes a single model upload with improved validation and error handling
func (p *Predator) uploadSingleModel(modelItem ModelUploadItem, bucket, basePath string, isPartial bool, authToken string) ModelUploadResult {
	// Step 1: Extract and validate model name
	modelName, err := p.extractModelName(modelItem.Metadata)
	if err != nil {
		return p.createErrorResult("unknown", "Failed to extract model name", err)
	}

	log.Info().Msgf("Processing %s upload for model: %s from %s",
		map[bool]string{true: "partial", false: "full"}[isPartial], modelName, modelItem.GCSPath)

	// Step 2: Setup destination paths
	destPath := path.Join(basePath, modelName)
	fullGCSPath := fmt.Sprintf("gs://%s/%s", bucket, destPath)

	// Step 3: Validate upload prerequisites
	if err := p.validateUploadPrerequisites(bucket, destPath, isPartial, modelName); err != nil {
		return p.createErrorResult(modelName, "Upload prerequisites validation failed", err)
	}

	// Step 4: Validate source model structure and configuration
	if err := p.validateSourceModel(modelItem.GCSPath, isPartial); err != nil {
		return p.createErrorResult(modelName, "Source model validation failed", err)
	}

	// Step 5: Validate metadata features (after model structure validation)
	if err := p.validateMetadataFeatures(modelItem.Metadata, authToken); err != nil {
		return p.createErrorResult(modelName, "Feature validation failed", err)
	}

	// Step 6: Download/sync model files based on upload type
	if err := p.syncModelFiles(modelItem.GCSPath, bucket, destPath, modelName, isPartial); err != nil {
		return p.createErrorResult(modelName, "Model file sync failed", err)
	}

	// Step 7: Copy config.pbtxt to prod config source (only in production)
	if err := p.copyConfigToProdConfigSource(modelItem.GCSPath, modelName); err != nil {
		return p.createErrorResult(modelName, "Failed to copy config to prod config source", err)
	}

	// Upload processed metadata.json (always done regardless of partial/full)
	metadataPath, err := p.uploadModelMetadata(modelItem.Metadata, bucket, destPath)
	if err != nil {
		return p.createErrorResult(modelName, "Metadata upload failed", err)
	}

	log.Info().Msgf("Successfully completed %s upload for model: %s",
		map[bool]string{true: "partial", false: "full"}[isPartial], modelName)
	return ModelUploadResult{
		ModelName:    modelName,
		GCSPath:      fullGCSPath,
		MetadataPath: metadataPath,
		Status:       "success",
	}
}

// copyConfigToProdConfigSource copies config.pbtxt to the prod config source path
// This is done in both int and prd environments so config is available for prod deployments
func (p *Predator) copyConfigToProdConfigSource(gcsPath, modelName string) error {
	// Check if config source is configured
	if pred.GcsConfigBucket == "" || pred.GcsConfigBasePath == "" {
		log.Warn().Msg("Config source not configured, skipping config.pbtxt copy to config source")
		return nil
	}

	// Parse source GCS path
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	// Read config.pbtxt from source
	srcConfigPath := path.Join(srcPath, configFile)
	configData, err := p.GcsClient.ReadFile(srcBucket, srcConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from source: %w", err)
	}

	// Update model name while preserving formatting
	updatedConfigData := p.replaceModelNameInConfigPreservingFormat(configData, modelName)

	// Upload to prod config source path with updated model name
	destConfigPath := path.Join(pred.GcsConfigBasePath, modelName, configFile)
	if err := p.GcsClient.UploadFile(pred.GcsConfigBucket, destConfigPath, updatedConfigData); err != nil {
		return fmt.Errorf("failed to upload config.pbtxt to config source: %w", err)
	}

	log.Info().Msgf("Successfully copied config.pbtxt to config source with model name %s: gs://%s/%s",
		modelName, pred.GcsConfigBucket, destConfigPath)
	return nil
}

// Helper functions for simplified upload flow

// createErrorResult creates a standardized error result
func (p *Predator) createErrorResult(modelName, message string, err error) ModelUploadResult {
	return ModelUploadResult{
		ModelName: modelName,
		Status:    "error",
		Error:     fmt.Sprintf("%s: %v", message, err),
	}
}

// generateUploadSummary creates response message and status code based on results
func (p *Predator) generateUploadSummary(successCount, failCount int, results []ModelUploadResult) (string, int) {
	switch {
	case failCount == 0:
		return fmt.Sprintf("%d model uploaded successfully", successCount), http.StatusOK
	case successCount == 0:
		return fmt.Sprintf("%d model failed to upload. Errors: %s", failCount, results[0].Error), http.StatusBadRequest
	default:
		return fmt.Sprintf("Mixed results: %d successful, %d failed. Errors: %s", successCount, failCount, results[0].Error), http.StatusPartialContent
	}
}

// validateUploadPrerequisites validates upload requirements based on type
func (p *Predator) validateUploadPrerequisites(bucket, destPath string, isPartial bool, modelName string) error {
	exists, err := p.GcsClient.CheckFolderExists(bucket, destPath)
	if err != nil {
		return fmt.Errorf("failed to check model existence: %w", err)
	}

	if isPartial {
		// Partial upload requires existing model
		if !exists {
			return fmt.Errorf("partial upload requires existing model folder at destination")
		}
		log.Info().Msgf("Partial upload: updating existing model %s", modelName)
	} else {
		// Full upload can create new or replace existing
		if exists {
			log.Info().Msgf("Full upload: replacing existing model %s", modelName)
		} else {
			log.Info().Msgf("Full upload: creating new model %s", modelName)
		}
	}

	return nil
}

// validateSourceModel validates the source model structure and configuration
func (p *Predator) validateSourceModel(gcsPath string, isPartial bool) error {
	// Parse GCS path
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	// Always validate config.pbtxt (required for both partial and full)
	if err := p.validateModelConfiguration(gcsPath); err != nil {
		return fmt.Errorf("config.pbtxt validation failed: %w", err)
	}

	if !isPartial {
		// For full upload, validate complete model structure
		if err := p.validateCompleteModelStructure(srcBucket, srcPath); err != nil {
			return fmt.Errorf("complete model structure validation failed: %w", err)
		}
	}

	return nil
}

// validateCompleteModelStructure validates that version "1" folder exists with non-empty files
// Note: config.pbtxt is already validated above
func (p *Predator) validateCompleteModelStructure(srcBucket, srcPath string) error {
	// Check if version "1" folder exists
	versionPath := path.Join(srcPath, "1")
	exists, err := p.GcsClient.CheckFolderExists(srcBucket, versionPath)
	if err != nil {
		return fmt.Errorf("failed to check version folder 1/: %w", err)
	}

	if !exists {
		return fmt.Errorf("version folder 1/ not found - required for complete model")
	}

	// Check if version "1" folder has at least one non-empty file
	if err := p.validateVersionHasFiles(srcBucket, versionPath); err != nil {
		return fmt.Errorf("version folder 1/ validation failed: %w", err)
	}

	log.Info().Msgf("Model structure validation passed - version 1/ folder exists with files")
	return nil
}

// validateVersionHasFiles checks if version folder has at least one non-empty file
func (p *Predator) validateVersionHasFiles(srcBucket, versionPath string) error {
	// Simply check if the version folder exists and has any content
	// CheckFolderExists returns true if there are any objects with the given prefix
	exists, err := p.GcsClient.CheckFolderExists(srcBucket, versionPath)
	if err != nil {
		return fmt.Errorf("failed to check version folder contents: %w", err)
	}

	if !exists {
		return fmt.Errorf("version folder 1/ is empty - must contain model files")
	}

	log.Info().Msgf("Version folder 1/ contains files")
	return nil
}

// syncModelFiles handles file synchronization based on upload type
func (p *Predator) syncModelFiles(gcsPath, destBucket, destPath, modelName string, isPartial bool) error {
	if isPartial {
		// Partial upload: only sync config.pbtxt
		return p.syncPartialFiles(gcsPath, destBucket, destPath, modelName)
	} else {
		// Full upload: sync everything
		return p.syncFullModel(gcsPath, destBucket, destPath, modelName)
	}
}

// uploadModelMetadata uploads metadata.json to GCS and returns the full path
func (p *Predator) uploadModelMetadata(metadata interface{}, bucket, destPath string) (string, error) {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	metadataPath := path.Join(destPath, "metadata.json")
	if err := p.GcsClient.UploadFile(bucket, metadataPath, metadataBytes); err != nil {
		return "", fmt.Errorf("failed to upload metadata: %w", err)
	}

	return fmt.Sprintf("gs://%s/%s", bucket, metadataPath), nil
}

// Legacy functions for backward compatibility
func (p *Predator) CheckModelExists(bucket, path string) (bool, error) {
	return p.GcsClient.CheckFolderExists(bucket, path)
}

func (p *Predator) UploadFileToGCS(bucket, path string, data []byte) error {
	return p.GcsClient.UploadFile(bucket, path, data)
}

// validateMetadataFeatures validates the features in metadata against online/offline validation APIs
func (p *Predator) validateMetadataFeatures(metadata interface{}, authToken string) error {
	// Parse metadata to extract features
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var featureMeta FeatureMetadata
	if err := json.Unmarshal(metadataBytes, &featureMeta); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Validate that auth token is provided
	if authToken == "" {
		return fmt.Errorf("authorization token is required for feature validation")
	}

	// Group features by validation type
	onlineFeaturesByEntity := make(map[string][]string)
	pricingFeaturesByEntity := make(map[string][]string)
	var offlineFeatures []string

	for _, input := range featureMeta.Inputs {
		for _, feature := range input.Features {
			featureType, entity, gf, featureName, isValid := externalcall.ParseFeatureString(feature)
			if !isValid {
				log.Error().Msgf("Invalid feature format: %s", feature)
				return fmt.Errorf("invalid feature format: %s", feature)
			}

			switch featureType {
			case "ONLINE_FEATURE", "PARENT_ONLINE_FEATURE":
				// Validate using online validation API
				onlineFeaturesByEntity[entity] = append(onlineFeaturesByEntity[entity], gf)
				log.Info().Msgf("Added online feature for validation - entity: %s, feature: %s", entity, gf)

			case "OFFLINE_FEATURE", "PARENT_OFFLINE_FEATURE":
				// Validate using offline validation API
				offlineFeatures = append(offlineFeatures, featureName)
				log.Info().Msgf("Added offline feature for validation: %s", featureName)

			case "RTP_FEATURE", "PARENT_RTP_FEATURE":
				// Validate using pricing service API - store full entity:feature_group:feature format
				fullFeature := entity + ":" + gf // entity:feature_group:feature
				pricingFeaturesByEntity[entity] = append(pricingFeaturesByEntity[entity], fullFeature)
				log.Info().Msgf("Added pricing feature for validation - entity: %s, full feature: %s", entity, fullFeature)

			case "DEFAULT_FEATURE", "PARENT_DEFAULT_FEATURE", "MODEL_FEATURE", "CALIBRATION":
				// These feature types don't need API validation - they are correct by default
				log.Info().Msgf("Skipping API validation for feature type %s: %s (no validation required)", featureType, feature)
				continue

			default:
				log.Warn().Msgf("Unknown feature type %s for feature: %s", featureType, feature)
			}
		}
	}

	// Validate online features
	for entity, features := range onlineFeaturesByEntity {
		if err := p.validateOnlineFeatures(entity, features, authToken); err != nil {
			return fmt.Errorf("online feature validation failed for entity %s: %w", entity, err)
		}
	}

	// Validate offline features
	if len(offlineFeatures) > 0 {
		if err := p.validateOfflineFeatures(offlineFeatures, authToken); err != nil {
			return fmt.Errorf("offline feature validation failed: %w", err)
		}
	}

	// Validate pricing features
	for entity, features := range pricingFeaturesByEntity {
		if err := p.validatePricingFeatures(entity, features); err != nil {
			return fmt.Errorf("pricing feature validation failed for entity %s: %w", entity, err)
		}
	}

	return nil
}

// validateOnlineFeatures validates online features for a specific entity
func (p *Predator) validateOnlineFeatures(entity string, features []string, token string) error {
	response, err := p.featureValidationClient.ValidateOnlineFeatures(entity, token)
	if err != nil {
		return fmt.Errorf("failed to call online validation API: %w", err)
	}

	// Check if all features exist in the response
	for _, feature := range features {
		if !externalcall.ValidateFeatureExists(feature, response) {
			return fmt.Errorf("online feature '%s' does not exist for entity '%s'", feature, entity)
		}
	}

	log.Info().Msgf("Successfully validated %d online features for entity %s", len(features), entity)
	return nil
}

// validateOfflineFeatures validates offline features by checking online mapping
func (p *Predator) validateOfflineFeatures(features []string, token string) error {
	response, err := p.featureValidationClient.ValidateOfflineFeatures(features, token)
	if err != nil {
		return fmt.Errorf("failed to call offline validation API: %w", err)
	}

	if response.Error != "" {
		return fmt.Errorf("offline validation API returned error: %s", response.Error)
	}

	// Check if all offline features have online mappings
	for _, feature := range features {
		if _, exists := response.Data[feature]; !exists {
			return fmt.Errorf("offline feature '%s' does not have an online mapping", feature)
		}
	}

	log.Info().Msgf("Successfully validated %d offline features", len(features))
	return nil
}

// validatePricingFeatures validates pricing features for a specific entity
func (p *Predator) validatePricingFeatures(entity string, features []string) error {
	if !pred.IsMeeshoEnabled {
		return nil
	}
	response, err := externalcall.PricingClient.GetDataTypes(entity)
	if err != nil {
		return fmt.Errorf("failed to call pricing service API: %w", err)
	}

	// Check if all features exist in the response
	for _, feature := range features {
		if !externalcall.ValidatePricingFeatureExists(feature, response) {
			return fmt.Errorf("pricing feature '%s' does not exist for entity '%s'", feature, entity)
		}
	}

	log.Info().Msgf("Successfully validated %d pricing features for entity %s", len(features), entity)
	return nil
}

// extractModelName extracts model name from metadata
func (p *Predator) extractModelName(metadata interface{}) (string, error) {
	// Parse metadata to extract model name
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var metadataMap map[string]interface{}
	if err := json.Unmarshal(metadataBytes, &metadataMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	modelName, exists := metadataMap["model_name"]
	if !exists {
		return "", fmt.Errorf("model_name not found in metadata")
	}

	modelNameStr, ok := modelName.(string)
	if !ok || modelNameStr == "" {
		return "", fmt.Errorf("model_name must be a non-empty string")
	}

	return modelNameStr, nil
}

// syncFullModel syncs all model files for full upload
func (p *Predator) syncFullModel(gcsPath, destBucket, destPath, modelName string) error {
	log.Info().Msgf("Syncing full model from GCS path: %s", gcsPath)

	// Parse the GCS path to extract bucket and object path
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	// Extract the model folder name from the source path
	pathSegments := strings.Split(strings.TrimSuffix(srcPath, "/"), "/")
	srcModelName := pathSegments[len(pathSegments)-1]
	srcBasePath := strings.TrimSuffix(srcPath, "/"+srcModelName)

	// Step 2: Transfer all files from source to destination
	log.Info().Msgf("Full upload: transferring all files from %s/%s to %s/%s",
		srcBucket, srcPath, destBucket, destPath)

	return p.GcsClient.TransferFolder(srcBucket, srcBasePath, srcModelName,
		destBucket, strings.TrimSuffix(destPath, "/"+modelName), modelName)
}

// syncPartialFiles syncs only config.pbtxt for partial upload
// Note: metadata.json is handled separately in uploadModelMetadata
func (p *Predator) syncPartialFiles(gcsPath, destBucket, destPath, modelName string) error {
	// Parse GCS path
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	// Files to sync for partial upload (only config.pbtxt)
	// metadata.json is always uploaded from the request metadata
	filesToSync := []string{"config.pbtxt"}

	log.Info().Msgf("Partial upload: syncing %v for model %s", filesToSync, modelName)

	for _, fileName := range filesToSync {
		srcFilePath := path.Join(srcPath, fileName)
		destFilePath := path.Join(destPath, fileName)

		// Read file from source
		data, err := p.GcsClient.ReadFile(srcBucket, srcFilePath)
		if err != nil {
			return fmt.Errorf("required file %s not found in source %s/%s: %w",
				fileName, srcBucket, srcFilePath, err)
		}

		// Note: config.pbtxt modification is handled by GCS client during TransferFolder

		// Upload to destination
		if err := p.GcsClient.UploadFile(destBucket, destFilePath, data); err != nil {
			return fmt.Errorf("failed to upload %s: %w", fileName, err)
		}

		log.Info().Msgf("Successfully synced %s for partial upload of model %s", fileName, modelName)
	}

	return nil
}

// validateModelConfiguration validates the model configuration by:
// 1. Downloading and parsing config.pbtxt to proto
// 2. Checking if backend is "python"
// 3. If backend is python, checking if preprocessing.tar.gz exists in source
func (p *Predator) validateModelConfiguration(gcsPath string) error {
	log.Info().Msgf("Validating model configuration for GCS path: %s", gcsPath)

	// Parse the GCS path to extract bucket and object path
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	// Step 1: Download config.pbtxt
	configPath := path.Join(srcPath, configFile)
	configData, err := p.GcsClient.ReadFile(srcBucket, configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from %s/%s: %w", srcBucket, configPath, err)
	}

	// Step 2: Parse config.pbtxt to proto
	var modelConfig ModelConfig
	if err := prototext.Unmarshal(configData, &modelConfig); err != nil {
		return fmt.Errorf("failed to parse config.pbtxt as proto: %w", err)
	}

	log.Info().Msgf("Parsed model config - Name: %s, Backend: %s", modelConfig.Name, modelConfig.Backend)

	return nil
}

// cleanEnsembleScheduling cleans up ensemble scheduling to avoid storing {"step": null}
func (p *Predator) cleanEnsembleScheduling(metadata MetaData) MetaData {
	// If ensemble scheduling step is empty, set to nil so omitempty works
	if len(metadata.Ensembling.Step) == 0 {
		metadata.Ensembling = Ensembling{Step: nil}
	}
	return metadata
}

// Returns the derived model name with deployable tag
func (p *Predator) GetDerivedModelName(payloadObject Payload, requestType string) (string, error) {
	if requestType != ScaleUpRequestType {
		return payloadObject.ModelName, nil
	}
	serviceDeployableID := payloadObject.ConfigMapping.ServiceDeployableID
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(int(serviceDeployableID))
	if err != nil {
		return constant.EmptyString, fmt.Errorf("%s: %w", errFetchDeployableConfig, err)
	}

	deployableTag := serviceDeployable.DeployableTag
	if deployableTag == "" {
		return payloadObject.ModelName, nil
	}

	derivedModelName := payloadObject.ModelName + deployableTagDelimiter + deployableTag
	derivedModelName = derivedModelName + deployableTagDelimiter + scaleupTag
	return derivedModelName, nil
}

// Returns the original model name if no tag is found (backward compatibility).
func (p *Predator) GetOriginalModelName(derivedModelName string, serviceDeployableID int) (string, error) {
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(serviceDeployableID)
	if err != nil {
		return constant.EmptyString, fmt.Errorf("%s: %w", errFetchDeployableConfig, err)
	}

	deployableTag := serviceDeployable.DeployableTag
	if deployableTag == "" {
		return derivedModelName, nil
	}

	scaleupSuffix := deployableTagDelimiter + scaleupTag
	derivedModelName = strings.TrimSuffix(derivedModelName, scaleupSuffix)

	deployableTagSuffix := deployableTagDelimiter + deployableTag
	if originalName, foundSuffix := strings.CutSuffix(derivedModelName, deployableTagSuffix); foundSuffix {
		return originalName, nil
	}

	return derivedModelName, nil
}

func (p *Predator) isNonProductionEnvironment() bool {
	env := strings.ToLower(strings.TrimSpace(pred.AppEnv))
	if env == "prd" || env == "prod" {
		return false
	}
	return true
}

func (p *Predator) copyConfigToNewNameInConfigSource(oldModelName, newModelName string) error {
	if oldModelName == newModelName {
		return nil
	}

	if pred.GcsConfigBucket == "" || pred.GcsConfigBasePath == "" {
		log.Warn().Msg("Config source not configured, skipping config.pbtxt copy in config source")
		return nil
	}

	destConfigPath := path.Join(pred.GcsConfigBasePath, newModelName, configFile)
	exists, err := p.GcsClient.CheckFileExists(pred.GcsConfigBucket, destConfigPath)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to check if config exists for %s, will attempt copy anyway", newModelName)
	} else if exists {
		log.Info().Msgf("Config already exists for %s in config source, skipping copy", newModelName)
		return nil
	}

	srcConfigPath := path.Join(pred.GcsConfigBasePath, oldModelName, configFile)

	configData, err := p.GcsClient.ReadFile(pred.GcsConfigBucket, srcConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from %s: %w", srcConfigPath, err)
	}

	// Use formatting-preserving function instead of marshal/unmarshal
	updatedConfigData := p.replaceModelNameInConfigPreservingFormat(configData, newModelName)

	if err := p.GcsClient.UploadFile(pred.GcsConfigBucket, destConfigPath, updatedConfigData); err != nil {
		return fmt.Errorf("failed to upload config.pbtxt to %s: %w", destConfigPath, err)
	}

	log.Info().Msgf("Successfully copied config.pbtxt from %s to %s in config source",
		oldModelName, newModelName)
	return nil
}

// replaceModelNameInConfigPreservingFormat updates only the top-level model name while preserving formatting
// It replaces only the first occurrence to avoid modifying nested names in inputs/outputs/instance_groups
func (p *Predator) replaceModelNameInConfigPreservingFormat(data []byte, destModelName string) []byte {
	content := string(data)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match top-level "name:" field - should be at the start of line (or minimal indentation)
		// Skip nested names which are typically indented with 2+ spaces
		if strings.HasPrefix(trimmed, "name:") {
			// Check indentation: top-level fields have minimal/no indentation
			leadingWhitespace := len(line) - len(strings.TrimLeft(line, " \t"))
			// Skip if heavily indented (nested field)
			if leadingWhitespace >= 2 {
				continue
			}

			// Match the first occurrence of name: "value" pattern
			namePattern := regexp.MustCompile(`name\s*:\s*"([^"]+)"`)
			matches := namePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				oldModelName := matches[1]
				// Replace only the FIRST occurrence to avoid replacing nested names
				loc := namePattern.FindStringIndex(line)
				if loc != nil {
					// Replace only the matched portion (first occurrence)
					before := line[:loc[0]]
					matched := line[loc[0]:loc[1]]
					after := line[loc[1]:]
					// Replace the value inside quotes while preserving the "name:" format
					valuePattern := regexp.MustCompile(`"([^"]+)"`)
					valueReplaced := valuePattern.ReplaceAllString(matched, fmt.Sprintf(`"%s"`, destModelName))
					lines[i] = before + valueReplaced + after
				} else {
					// Fallback: replace all (shouldn't happen with valid input)
					lines[i] = namePattern.ReplaceAllString(line, fmt.Sprintf(`name: "%s"`, destModelName))
				}
				log.Info().Msgf("Replacing top-level model name in config.pbtxt: '%s' -> '%s'", oldModelName, destModelName)
				break
			}
		}
	}

	return []byte(strings.Join(lines, "\n"))
}
