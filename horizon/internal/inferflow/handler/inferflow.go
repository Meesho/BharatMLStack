package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	mainHandler "github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	inferflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/inferflow"
	etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	pb "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/handler/proto/protogen"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	discovery_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	inferflow_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/config"
	inferflow_request "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/request"
	service_deployable_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/grpc"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/random"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

type InferFlow struct {
	EtcdConfig                  etcd.Manager
	InferFlowRequestRepo        inferflow_request.Repository
	InferFlowConfigRepo         inferflow_config.Repository
	DiscoveryConfigRepo         discovery_config.DiscoveryConfigRepository
	ServiceDeployableConfigRepo service_deployable_config.ServiceDeployableRepository
	infrastructureHandler       infrastructurehandler.InfrastructureHandler
	workingEnv                  string
}

const (
	emptyResponse                     = ""
	rejected                          = "REJECTED"
	approved                          = "APPROVED"
	pendingApproval                   = "PENDING APPROVAL"
	promoteRequestType                = "PROMOTE"
	onboardRequestType                = "ONBOARD"
	editRequestType                   = "EDIT"
	cloneRequestType                  = "CLONE"
	scaleUpRequestType                = "SCALE UP"
	deleteRequestType                 = "DELETE"
	cancelled                         = "CANCELLED"
	adminRole                         = "ADMIN"
	activeTrue                        = true
	activeFalse                       = false
	inferFlowRetrieveModelScoreMethod = "/Inferflow/RetrieveModelScore"
	setFunctionalTest                 = "FunctionalTest"
	defaultLoggingTTL                 = 30
	maxConfigVersion                  = 15
	defaultModelSchemaPerc            = 0
	deployableTagDelimiter            = "_"
	scaleupTag                        = "scaleup"
	defaultVersion                    = 1
)

func InitV1ConfigHandler() Config {
	if config == nil {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn := conn.(*infra.SQLConnection)

		InferFlowRequestRepo, err := inferflow_request.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create infer flow request repository")
		}

		InferFlowConfigRepo, err := inferflow_config.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create infer flow config repository")
		}

		DiscoveryConfigRepo, err := discovery_config.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create deployable config repository")
		}

		ServiceDeployableConfigRepo, err := service_deployable_config.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create service deployable config repository")
		}

		infrastructureHandler := infrastructurehandler.InitInfrastructureHandler()
		// Use generalized working environment
		workingEnv := mainHandler.GetWorkingEnvironment()

		config = &InferFlow{
			EtcdConfig:                  etcd.NewEtcdInstance(),
			InferFlowRequestRepo:        InferFlowRequestRepo,
			InferFlowConfigRepo:         InferFlowConfigRepo,
			DiscoveryConfigRepo:         DiscoveryConfigRepo,
			ServiceDeployableConfigRepo: ServiceDeployableConfigRepo,
			infrastructureHandler:       infrastructureHandler,
			workingEnv:                  workingEnv,
		}
	}
	return config
}

func (m *InferFlow) Onboard(request InferflowOnboardRequest, token string) (Response, error) {

	configId := fmt.Sprintf("%s-%s-%s", request.Payload.RealEstate, request.Payload.Tenant, request.Payload.ConfigIdentifier)

	response, err := m.ValidateOnboardRequest(request.Payload)
	if err != nil {
		return Response{}, errors.New("failed to validate onboard request: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("inferflow config is invalid: " + response.Error)
	}

	exists, err := m.InferFlowConfigRepo.DoesConfigIDExist(configId)
	if err != nil {
		return Response{}, errors.New("failed to check if config already exists: " + err.Error())
	}
	if exists {
		return Response{}, errors.New("Config ID already exists")
	}

	inferFlowConfig, err := m.GetInferflowConfig(request, token)
	if err != nil {
		return Response{}, errors.New("failed to generate inferflow config: " + err.Error())
	}

	response, err = ValidateInferFlowConfig(inferFlowConfig, token)
	if err != nil {
		return Response{}, errors.New("failed to validate inferflow config: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("infer flow config is invalid: " + response.Error)
	}

	payload, err := AdaptOnboardRequestToDBPayload(request, inferFlowConfig, request.Payload)
	if err != nil {
		return Response{}, errors.New("failed to adapt onboard request to db payload: " + err.Error())
	}
	log.Info().Msgf("payload: %+v", payload)

	table := &inferflow_request.Table{
		ConfigID:    configId,
		CreatedBy:   request.CreatedBy,
		Payload:     payload,
		Status:      pendingApproval,
		RequestType: onboardRequestType,
		Active:      activeTrue,
	}

	err = m.InferFlowRequestRepo.Create(table)
	if err != nil {
		return Response{}, errors.New("failed to create infer flow config: " + err.Error())
	}
	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("InferFlowConfig creation request created successfully for Request Id %d and Config Id %s", table.RequestID, configId)},
	}, nil
}

func (m *InferFlow) Promote(request PromoteConfigRequest) (Response, error) {
	exists, err := m.InferFlowRequestRepo.DoesConfigIdExistWithRequestType(request.Payload.ConfigID, promoteRequestType)
	if err != nil {
		return Response{}, errors.New("failed to check if config id exists with request type in db: " + err.Error())
	}

	if exists {
		return Response{}, errors.New("request with this Config Id is already raised")
	}

	modelNameToEndPointMap := make(map[string]string)
	for _, proposedModelEndpoint := range request.Payload.ProposedModelEndPoints {
		modelNameToEndPointMap[proposedModelEndpoint.ModelName] = proposedModelEndpoint.EndPointID
	}

	for i := range request.Payload.ConfigValue.ComponentConfig.PredatorComponents {
		modelName := request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelName
		request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelEndPoint = modelNameToEndPointMap[modelName]
	}
	for i := range request.Payload.LatestRequest.Payload.RequestPayload.Rankers {
		modelName := request.Payload.LatestRequest.Payload.RequestPayload.Rankers[i].ModelName
		request.Payload.LatestRequest.Payload.RequestPayload.Rankers[i].EndPoint = modelNameToEndPointMap[modelName]
	}
	request.Payload.ConfigValue.ResponseConfig.LoggingTTL = defaultLoggingTTL
	request.Payload.ConfigValue.ResponseConfig.ModelSchemaPerc = defaultModelSchemaPerc

	destinationDeployableID := request.Payload.ConfigMapping.DeployableID
	request.Payload.LatestRequest.Payload.ConfigMapping.DeployableID = destinationDeployableID
	request.Payload.LatestRequest.Payload.RequestPayload.ConfigMapping.DeployableID = destinationDeployableID
	request.Payload.LatestRequest.Payload.RequestPayload.Response.RankerSchemaFeaturesInResponsePerc = defaultModelSchemaPerc

	newVersion := defaultVersion
	configIDExists, err := m.InferFlowConfigRepo.DoesConfigIDExist(request.Payload.ConfigID)
	if err != nil {
		return Response{}, errors.New("failed to check if config id exists in config table " + err.Error())
	}
	if configIDExists {
		log.Info().Msgf("config already exists, bumping version")
		latestRequests, retrieveErr := m.InferFlowRequestRepo.GetApprovedRequestsByConfigID(request.Payload.ConfigID)
		if retrieveErr != nil {
			return Response{}, errors.New("failed to fetch config from DB")
		}
		if len(latestRequests) > 0 {
			newVersion = latestRequests[0].Version + 1
		}
		if newVersion > maxConfigVersion {
			return Response{}, errors.New("This inferflow config has reached its version limit. Please create a clone to make further updates.")
		}
		request.Payload.ConfigValue.ComponentConfig.CacheVersion = newVersion
	} else {
		request.Payload.ConfigValue.ComponentConfig.CacheVersion = newVersion
	}

	payload, err := AdaptPromoteRequestToDBPayload(request, request.Payload.LatestRequest)
	if err != nil {
		return Response{}, errors.New("failed to adapt promote request to db payload: " + err.Error())
	}

	table := &inferflow_request.Table{
		ConfigID:    request.Payload.ConfigID,
		Payload:     payload,
		CreatedBy:   request.CreatedBy,
		RequestType: promoteRequestType,
		Status:      pendingApproval,
		Active:      activeTrue,
		Version:     newVersion,
	}

	err = m.InferFlowRequestRepo.Create(table)
	if err != nil {
		return Response{}, errors.New("failed to create infer flow config request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Infer Flow Config Promote request raised successfully with Request Id %d and Config Id %s", table.RequestID, request.Payload.ConfigID)},
	}, nil

}

func (m *InferFlow) Edit(request EditConfigOrCloneConfigRequest, token string) (Response, error) {
	configId := fmt.Sprintf("%s-%s-%s", request.Payload.RealEstate, request.Payload.Tenant, request.Payload.ConfigIdentifier)

	response, err := m.ValidateOnboardRequest(request.Payload)
	if err != nil {
		return Response{}, errors.New("failed to validate edit request: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("onboard request is invalid: " + response.Error)
	}

	exists, err := m.InferFlowConfigRepo.DoesConfigIDExist(configId)
	if err != nil {
		return Response{}, errors.New("failed to check if config id exists in db: " + err.Error())
	}
	if !exists {
		return Response{}, errors.New("Config ID does not exist")
	}

	latestUnapprovedRequests, err := m.InferFlowRequestRepo.GetLatestPendingRequestByConfigID(configId)
	if err == nil {
		if len(latestUnapprovedRequests) > 0 {
			if latestUnapprovedRequests[0].RequestType == editRequestType {
				return Response{}, errors.New("Pending edit request with request ID: " + strconv.Itoa(int(latestUnapprovedRequests[0].RequestID)))
			}
		}
	}

	existingConfigs, err := m.InferFlowRequestRepo.GetApprovedRequestsByConfigID(configId)
	if err != nil {
		return Response{}, errors.New("failed to get existing configs: " + err.Error())
	}

	newVersion := defaultVersion
	prevLoggingTTL := defaultLoggingTTL
	if len(existingConfigs) > 0 {
		newVersion = existingConfigs[0].Version + 1
		prevLoggingTTL = existingConfigs[0].Payload.ConfigValue.ResponseConfig.LoggingTTL
	}
	if request.Payload.Response.LoggingTTL == 0 {
		request.Payload.Response.LoggingTTL = prevLoggingTTL
	}

	if newVersion > maxConfigVersion {
		return Response{}, errors.New("This inferflow config has reached its version limit. Please create a clone to make further updates.")
	}

	onboardRequest := InferflowOnboardRequest(request)

	inferFlowConfig, err := m.GetInferflowConfig(onboardRequest, token)
	if err != nil {
		return Response{}, errors.New("failed to get infer flow config: " + err.Error())
	}

	response, err = ValidateInferFlowConfig(inferFlowConfig, token)
	if err != nil {
		return Response{}, errors.New("failed to validate inferflow config: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("infer flow config is invalid: " + response.Error)
	}

	inferFlowConfig.ComponentConfig.CacheVersion = newVersion

	payload, err := AdaptEditRequestToDBPayload(request, inferFlowConfig, request.Payload)
	if err != nil {
		return Response{}, errors.New("failed to adapt edit request to db payload: " + err.Error())
	}

	table := &inferflow_request.Table{
		ConfigID:    configId,
		Payload:     payload,
		CreatedBy:   request.CreatedBy,
		Version:     newVersion,
		RequestType: editRequestType,
		Status:      pendingApproval,
		Active:      activeTrue,
	}

	err = m.InferFlowRequestRepo.Create(table)
	if err != nil {
		return Response{}, errors.New("failed to create infer flow config edit request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Infer Flow Config edit request created successfully with Request Id %d and Config Id %s", table.RequestID, configId)},
	}, nil
}

func (m *InferFlow) Clone(request EditConfigOrCloneConfigRequest, token string) (Response, error) {
	configId := fmt.Sprintf("%s-%s-%s", request.Payload.RealEstate, request.Payload.Tenant, request.Payload.ConfigIdentifier)

	response, err := m.ValidateOnboardRequest(request.Payload)
	if err != nil {
		return Response{}, errors.New("failed to validate onboard request: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("onboard request is invalid: " + response.Error)
	}

	exists, err := m.InferFlowRequestRepo.DoesConfigIDExist(configId)
	if err != nil {
		return Response{}, errors.New("failed to check if config id exists in db: " + err.Error())
	}
	if exists {
		return Response{}, errors.New("Config ID already exists")
	}

	// remove routing config from request
	for _, ranker := range request.Payload.Rankers {
		if len(ranker.RoutingConfig) > 0 {
			ranker.RoutingConfig = make([]RoutingConfig, 0)
		}
	}

	if request.Payload.Response.LoggingTTL == 0 {
		request.Payload.Response.LoggingTTL = defaultLoggingTTL
	}
	request.Payload.Response.RankerSchemaFeaturesInResponsePerc = defaultModelSchemaPerc

	onboardRequest := InferflowOnboardRequest(request)

	inferFlowConfig, err := m.GetInferflowConfig(onboardRequest, token)
	if err != nil {
		return Response{}, errors.New("failed to get infer flow config: " + err.Error())
	}

	response, err = ValidateInferFlowConfig(inferFlowConfig, token)
	if err != nil {
		return Response{}, errors.New("failed to validate infer flow config: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("infer flow config is invalid: " + response.Error)
	}

	payload, err := AdaptCloneConfigRequestToDBPayload(request, inferFlowConfig, request.Payload)
	if err != nil {
		return Response{}, errors.New("failed to adapt edit request to db payload: " + err.Error())
	}

	table := &inferflow_request.Table{
		ConfigID:    configId,
		Payload:     payload,
		CreatedBy:   request.CreatedBy,
		RequestType: cloneRequestType,
		Status:      pendingApproval,
		Active:      activeTrue,
	}

	err = m.InferFlowRequestRepo.Create(table)
	if err != nil {
		return Response{}, errors.New("failed to create infer flow config clone request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Infer Flow Config cloning request created successfully with Request Id %d and Config Id %s", table.RequestID, configId)},
	}, nil
}

func (m *InferFlow) ScaleUp(request ScaleUpConfigRequest) (Response, error) {
	sourceConfigID := request.Payload.ConfigID
	derivedConfigID, err := m.GetDerivedConfigID(request.Payload.ConfigID, request.GetConfigMapping().DeployableID)
	if err != nil {
		return Response{}, errors.New("failed to create derived config ID: " + err.Error())
	}
	request.Payload.ConfigID = derivedConfigID
	exists, err := m.InferFlowRequestRepo.DoesConfigIdExistWithRequestType(request.Payload.ConfigID, scaleUpRequestType)
	if err != nil {
		return Response{}, errors.New("failed to check if config id exists in db: " + err.Error())
	}
	if exists {
		return Response{}, errors.New("Config ID already exists with scale up request")
	}

	var latestSourceRequest GetLatestRequestResponse
	latestSourceRequest, err = m.GetLatestRequest(sourceConfigID)
	if err != nil {
		return Response{}, errors.New("failed to get latest request for the source configID: " + sourceConfigID + ": " + err.Error())
	}
	request.Payload.ConfigMapping.SourceConfigID = sourceConfigID

	modelNameToEndPointMap := make(map[string]ModelNameToEndPointMap)
	for _, proposedModelEndpoint := range request.Payload.ModelNameToEndPointMap {
		modelNameToEndPointMap[proposedModelEndpoint.CurrentModelName] = proposedModelEndpoint
	}

	for i := range request.Payload.ConfigValue.ComponentConfig.PredatorComponents {
		modelName := request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelName
		request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelEndPoint = modelNameToEndPointMap[modelName].EndPointID
		request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelName = modelNameToEndPointMap[modelName].NewModelName
	}

	for i := range latestSourceRequest.Data.Payload.RequestPayload.Rankers {
		modelName := latestSourceRequest.Data.Payload.RequestPayload.Rankers[i].ModelName
		latestSourceRequest.Data.Payload.RequestPayload.Rankers[i].EndPoint = modelNameToEndPointMap[modelName].EndPointID
		latestSourceRequest.Data.Payload.RequestPayload.Rankers[i].ModelName = modelNameToEndPointMap[modelName].NewModelName
	}

	// Set Request Payload name
	parts := strings.Split(request.Payload.ConfigID, "-")
	if len(parts) >= 3 {
		latestSourceRequest.Data.Payload.RequestPayload.RealEstate = parts[0]
		latestSourceRequest.Data.Payload.RequestPayload.Tenant = parts[1]
		latestSourceRequest.Data.Payload.RequestPayload.ConfigIdentifier = strings.Join(parts[2:], "-")

	}

	latestSourceRequest.Data.ConfigID = request.Payload.ConfigID
	latestSourceRequest.Data.Payload.RequestPayload.ConfigMapping.DeployableID = request.Payload.ConfigMapping.DeployableID
	latestSourceRequest.Data.Payload.RequestPayload.Response.RankerSchemaFeaturesInResponsePerc = defaultModelSchemaPerc

	request.Payload.ConfigValue.ResponseConfig.ModelSchemaPerc = defaultModelSchemaPerc
	request.Payload.ConfigValue.ResponseConfig.LoggingPerc = request.Payload.LoggingPerc
	request.Payload.ConfigValue.ResponseConfig.LoggingTTL = request.Payload.LoggingTTL
	request.Payload.ConfigValue.ComponentConfig.CacheVersion = defaultVersion

	payload, err := AdaptScaleUpRequestToDBPayload(request, latestSourceRequest.Data)
	if err != nil {
		return Response{}, errors.New("failed to adapt scale up request to db payload: " + err.Error())
	}
	payload.ConfigMapping.SourceConfigID = request.Payload.ConfigMapping.SourceConfigID

	table := &inferflow_request.Table{
		ConfigID:    request.Payload.ConfigID,
		Payload:     payload,
		CreatedBy:   request.CreatedBy,
		RequestType: scaleUpRequestType,
		Status:      pendingApproval,
		Active:      activeTrue,
	}

	err = m.InferFlowRequestRepo.Create(table)
	if err != nil {
		return Response{}, errors.New("failed to create infer flow config scale up request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Infer Flow Config Scale Up request updated successfully with Request Id %d and Config Id %s", table.RequestID, request.Payload.ConfigID)},
	}, nil
}

func (m *InferFlow) Delete(request DeleteConfigRequest) (Response, error) {

	InferFlowConfigTable, err := m.InferFlowConfigRepo.GetByID(request.ConfigID)
	if err != nil {
		return Response{}, errors.New("failed to get infer flow config by id in db: " + err.Error())
	}
	if InferFlowConfigTable == nil {
		return Response{}, errors.New("inferflow config: " + request.ConfigID + " does not exist in db")
	}

	Discoverytable, err := m.DiscoveryConfigRepo.GetById(int(InferFlowConfigTable.DiscoveryID))
	if err != nil {
		return Response{}, errors.New("failed to get discovery config by id in db: " + err.Error())
	}

	payload := &inferflow_request.Table{
		ConfigID: request.ConfigID,
		Payload: inferflow.Payload{
			ConfigMapping: inferflow.ConfigMapping{
				DeployableID:          Discoverytable.ServiceDeployableID,
				AppToken:              Discoverytable.AppToken,
				ConnectionConfigID:    Discoverytable.ServiceConnectionID,
				ResponseDefaultValues: nil,
			},
			ConfigValue: InferFlowConfigTable.ConfigValue,
		},
		CreatedBy:   request.CreatedBy,
		RequestType: deleteRequestType,
		Status:      pendingApproval,
		Active:      activeTrue,
	}

	err = m.InferFlowRequestRepo.Create(payload)
	if err != nil {
		return Response{}, errors.New("failed to create infer flow config delete request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Config deletion request raised successfully with Request Id %d and Config Id %s", payload.RequestID, request.ConfigID)},
	}, nil

}

func (m *InferFlow) Cancel(request CancelConfigRequest) (Response, error) {

	exists, err := m.InferFlowRequestRepo.DoesRequestIDExistWithStatus(request.RequestID, pendingApproval)
	if err != nil {
		return Response{}, errors.New("failed to check if request id exists with status in db: " + err.Error())
	}
	if !exists {
		return Response{}, errors.New("request ID does not exist or is not pending approval")
	}

	table := &inferflow_request.Table{
		RequestID: request.RequestID,
		Status:    cancelled,
		UpdatedBy: request.UpdatedBy,
		Active:    activeFalse,
	}

	err = m.InferFlowRequestRepo.Update(table)
	if err != nil {
		return Response{}, errors.New("failed to update infer flow config onboarding request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Infer Flow Config onboarding request cancelled successfully for Request Id %d", table.RequestID)},
	}, nil
}

func (m *InferFlow) Review(request ReviewRequest) (Response, error) {
	request.Status = strings.ToUpper(request.Status)

	if request.Status != approved && request.Status != rejected {
		log.Error().Msgf("invalid status for request id: %d", request.RequestID)
		return Response{}, errors.New("invalid status for request")
	}

	if request.Status == rejected && request.RejectReason == emptyResponse {
		log.Error().Msgf("request reason not specified for request id: %d", request.RequestID)
		return Response{}, errors.New("rejection reason is required")
	}

	exists, err := m.InferFlowRequestRepo.DoesRequestIDExistWithStatus(request.RequestID, pendingApproval)
	if err != nil {
		log.Error().Msgf("failed to check if request id: %d exists with status in db: %s", request.RequestID, err)
		return Response{}, errors.New("failed to check if request id exists with status in db: " + err.Error())
	}
	if !exists {
		log.Error().Msgf("request id: %d does not exist or request is not pending approval", request.RequestID)
		return Response{}, errors.New("request id does not exist or request is not pending approval")
	}

	if request.Status == rejected {
		return m.handleRejectedRequest(request)
	}

	return m.handleApprovedRequest(request)
}

func (m *InferFlow) handleRejectedRequest(request ReviewRequest) (Response, error) {
	requestEntry := &inferflow_request.Table{
		RequestID:    request.RequestID,
		Status:       request.Status,
		RejectReason: request.RejectReason,
		Reviewer:     request.Reviewer,
		Active:       activeFalse,
	}

	if err := m.InferFlowRequestRepo.Update(requestEntry); err != nil {
		return Response{}, errors.New("failed to update inferflow config request in db: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("inferflow config request rejected successfully for Request Id %d", request.RequestID)},
	}, nil
}

func (m *InferFlow) handleApprovedRequest(request ReviewRequest) (Response, error) {
	var requestEntry *inferflow_request.Table
	var discoveryID int
	var discoveryConfig *discovery_config.DiscoveryConfig

	tempRequest := inferflow_request.Table{}
	tempRequest, err := m.InferFlowRequestRepo.GetRequestByID(request.RequestID)
	if err != nil {
		return Response{}, fmt.Errorf("failed to fetch latest unapproved request for request id: %d: %w", request.RequestID, err)
	}

	var configExistedBeforeTx bool
	if tempRequest.RequestType == promoteRequestType {
		existingConfig, _ := m.InferFlowConfigRepo.GetByID(tempRequest.ConfigID)
		configExistedBeforeTx = existingConfig != nil
	}

	err = m.InferFlowRequestRepo.Transaction(func(tx *gorm.DB) error {
		requestEntry = &inferflow_request.Table{
			RequestID:    request.RequestID,
			Status:       request.Status,
			RejectReason: request.RejectReason,
			Reviewer:     request.Reviewer,
		}
		if err := tx.First(requestEntry, request.RequestID).Error; err != nil {
			return fmt.Errorf("failed to get request: %w", err)
		}
		requestEntry.Reviewer = request.Reviewer
		requestEntry.RejectReason = request.RejectReason

		var err error
		discoveryID, discoveryConfig, err = m.createOrUpdateDiscoveryConfig(tx, requestEntry, configExistedBeforeTx)
		if err != nil {
			return fmt.Errorf("failed to handle discovery config: %w", err)
		}

		if err := m.createOrUpdateInferFlowConfig(tx, requestEntry, discoveryID, configExistedBeforeTx); err != nil {
			return fmt.Errorf("failed to handle inferflow config: %w", err)
		}

		requestEntry.Status = approved
		err = m.InferFlowRequestRepo.UpdateTx(tx, requestEntry)
		if err != nil {
			return errors.New("failed to update inferflow config request in db: " + err.Error())
		}

		return nil
	})

	if err != nil {
		return Response{}, fmt.Errorf("failed to review config (DB rolled back): %w", err)
	}

	if err := m.createOrUpdateEtcdConfig(requestEntry, discoveryConfig, configExistedBeforeTx); err != nil {
		if rollBackErr := m.rollbackApprovedRequest(request, requestEntry, discoveryID, configExistedBeforeTx); rollBackErr != nil {
			log.Error().Err(rollBackErr).Msg("Failed to rollback DB changes after ETCD failure")
			return Response{}, fmt.Errorf("ETCD sync failed and DB rollback also failed: etcd=%w, rollback=%v", err, rollBackErr)
		}
		log.Warn().Msgf("Successfully rolled back the request: %d", request.RequestID)
		return Response{}, fmt.Errorf("ETCD sync failed: %w", err)
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: "Mp Config reviewed successfully."},
	}, nil
}

func (m *InferFlow) rollbackApprovedRequest(request ReviewRequest, fullTable *inferflow_request.Table, discoveryID int, configExistedBeforeTx bool) error {
	return m.InferFlowRequestRepo.Transaction(func(tx *gorm.DB) error {
		table := &inferflow_request.Table{
			RequestID: request.RequestID,
			Status:    pendingApproval,
			Reviewer:  emptyResponse,
		}
		if err := m.InferFlowRequestRepo.UpdateTx(tx, table); err != nil {
			return fmt.Errorf("failed to revert request status: %w", err)
		}

		switch fullTable.RequestType {
		case onboardRequestType, cloneRequestType, scaleUpRequestType:
			if err := m.rollbackCreatedConfigs(tx, fullTable.ConfigID, discoveryID); err != nil {
				return err
			}

		case editRequestType:
			if err := m.rollbackEditRequest(tx, fullTable, discoveryID); err != nil {
				return err
			}

		case deleteRequestType:
			updatedBy := fullTable.UpdatedBy
			if updatedBy == "" {
				updatedBy = fullTable.CreatedBy
			}
			if err := m.rollbackDeletedConfigs(tx, fullTable.ConfigID, discoveryID, updatedBy); err != nil {
				return err
			}

		case promoteRequestType:
			if err := m.rollbackPromoteRequest(tx, fullTable, discoveryID, configExistedBeforeTx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (m *InferFlow) rollbackPromoteRequest(tx *gorm.DB, currentRequest *inferflow_request.Table, discoveryID int, configExistedBeforeTx bool) error {
	if !configExistedBeforeTx {
		if err := m.rollbackCreatedConfigs(tx, currentRequest.ConfigID, discoveryID); err != nil {
			return err
		}
	}
	return nil
}

func (m *InferFlow) rollbackEditRequest(tx *gorm.DB, currentRequest *inferflow_request.Table, discoveryID int) error {
	approvedRequests, err := m.InferFlowRequestRepo.GetApprovedRequestsByConfigID(currentRequest.ConfigID)
	if err != nil {
		return fmt.Errorf("Failed to retrieve approved requests: %w", err)
	}

	var previousRequest *inferflow_request.Table
	if len(approvedRequests) > 0 {
		if approvedRequests[0].RequestID == currentRequest.RequestID {
			if len(approvedRequests) > 1 {
				previousRequest = &approvedRequests[1]
			} else {
				return fmt.Errorf("no other request to revert back to: Requires manual intervention")
			}
		} else {
			previousRequest = &approvedRequests[0]
		}
	} else {
		return fmt.Errorf("no other request to revert back to: Requires manual intervention")
	}

	existingConfig, err := m.InferFlowConfigRepo.GetByID(currentRequest.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to get inferflow config: %w", err)
	}
	if existingConfig == nil {
		return errors.New("inferflow config not found")
	}

	restoredConfig := &inferflow_config.Table{
		ConfigID:    currentRequest.ConfigID,
		DiscoveryID: discoveryID,
		ConfigValue: previousRequest.Payload.ConfigValue,
		Active:      activeTrue,
		UpdatedBy:   currentRequest.UpdatedBy,
	}

	if err := m.InferFlowConfigRepo.UpdateTx(tx, restoredConfig); err != nil {
		return fmt.Errorf("failed to restore inferflow config: %w", err)
	}

	restoredDiscovery := &discovery_config.DiscoveryConfig{
		ID:                  discoveryID,
		ServiceDeployableID: previousRequest.Payload.ConfigMapping.DeployableID,
		AppToken:            previousRequest.Payload.ConfigMapping.AppToken,
		ServiceConnectionID: previousRequest.Payload.ConfigMapping.ConnectionConfigID,
		Active:              activeTrue,
		UpdatedBy:           currentRequest.UpdatedBy,
	}
	if err := m.DiscoveryConfigRepo.UpdateTx(tx, restoredDiscovery); err != nil {
		return fmt.Errorf("failed to restore discovery config: %w", err)
	}

	return nil
}

func (m *InferFlow) rollbackCreatedConfigs(tx *gorm.DB, configID string, discoveryID int) error {
	if err := m.InferFlowConfigRepo.DeleteByConfigIDTx(tx, configID); err != nil {
		return fmt.Errorf("failed to rollback inferflow config: %w", err)
	}

	if err := m.DiscoveryConfigRepo.DeleteByIDTx(tx, discoveryID); err != nil {
		return fmt.Errorf("failed to rollback discovery config: %w", err)
	}

	return nil
}

func (m *InferFlow) rollbackDeletedConfigs(tx *gorm.DB, configID string, discoveryID int, updatedby string) error {
	latestConfig, err := m.InferFlowConfigRepo.GetLatestInactiveByConfigID(tx, configID)
	if err != nil {
		return fmt.Errorf("failed to find soft-deleted inferflow config: %w", err)
	}
	if latestConfig == nil {
		return errors.New("no soft-deleted inferflow config found")
	}

	if err := m.InferFlowConfigRepo.ReactivateByIDTx(tx, int(latestConfig.ID), updatedby); err != nil {
		return fmt.Errorf("failed to reactivate inferflow config: %w", err)
	}

	if err := m.DiscoveryConfigRepo.ReactivateByIDTx(tx, discoveryID); err != nil {
		return fmt.Errorf("failed to reactivate discovery config: %w", err)
	}

	return nil
}

func (m *InferFlow) createOrUpdateDiscoveryConfig(tx *gorm.DB, requestEntry *inferflow_request.Table, configExistedBeforeTx bool) (int, *discovery_config.DiscoveryConfig, error) {
	discovery := &discovery_config.DiscoveryConfig{
		ServiceDeployableID: requestEntry.Payload.ConfigMapping.DeployableID,
		AppToken:            requestEntry.Payload.ConfigMapping.AppToken,
		ServiceConnectionID: requestEntry.Payload.ConfigMapping.ConnectionConfigID,
		Active:              activeTrue,
	}

	switch requestEntry.RequestType {
	case onboardRequestType, cloneRequestType, scaleUpRequestType:
		if requestEntry.UpdatedBy != "" {
			discovery.CreatedBy = requestEntry.UpdatedBy
		} else {
			discovery.CreatedBy = requestEntry.CreatedBy
		}
		err := m.DiscoveryConfigRepo.CreateTx(tx, discovery)
		if err != nil {
			return 0, nil, errors.New("failed to create discovery config: " + err.Error())
		}
	case promoteRequestType:
		if !configExistedBeforeTx {
			if requestEntry.UpdatedBy != "" {
				discovery.CreatedBy = requestEntry.UpdatedBy
			} else {
				discovery.CreatedBy = requestEntry.CreatedBy
			}
			err := m.DiscoveryConfigRepo.CreateTx(tx, discovery)
			if err != nil {
				return 0, nil, errors.New("failed to create discovery config: " + err.Error())
			}
		} else {
			existingConfig, err := m.InferFlowConfigRepo.GetByID(requestEntry.ConfigID)
			if err != nil {
				return 0, nil, errors.New("failed to query inferflow config repo: " + err.Error())
			}
			if requestEntry.UpdatedBy != "" {
				discovery.UpdatedBy = requestEntry.UpdatedBy
			} else {
				discovery.UpdatedBy = requestEntry.CreatedBy
			}
			discovery.ID = int(existingConfig.DiscoveryID)
			err = m.DiscoveryConfigRepo.UpdateTx(tx, discovery)
			if err != nil {
				return 0, nil, errors.New("failed to update discovery config: " + err.Error())
			}
		}
	case editRequestType:
		if requestEntry.UpdatedBy != "" {
			discovery.UpdatedBy = requestEntry.UpdatedBy
		} else {
			discovery.UpdatedBy = requestEntry.CreatedBy
		}
		config, err := m.InferFlowConfigRepo.GetByID(requestEntry.ConfigID)
		if err != nil {
			return 0, nil, errors.New("failed to get inferflow config by id: " + err.Error())
		}
		if config == nil {
			return 0, nil, errors.New("failed to get inferflow config by id")
		}
		discovery.ID = int(config.DiscoveryID)
		err = m.DiscoveryConfigRepo.UpdateTx(tx, discovery)
		if err != nil {
			return 0, nil, errors.New("failed to update discovery config: " + err.Error())
		}
	case deleteRequestType:
		config, err := m.InferFlowConfigRepo.GetByID(requestEntry.ConfigID)
		if err != nil {
			return 0, nil, errors.New("failed to get inferflow config by id: " + err.Error())
		}
		if config == nil {
			return 0, nil, errors.New("failed to get inferflow config by id")
		}
		if requestEntry.UpdatedBy != "" {
			discovery.UpdatedBy = requestEntry.UpdatedBy
		} else {
			discovery.UpdatedBy = requestEntry.CreatedBy
		}
		discovery.ID = int(config.DiscoveryID)
		discovery.Active = activeFalse
		err = m.DiscoveryConfigRepo.UpdateTx(tx, discovery)
		if err != nil {
			return 0, nil, errors.New("failed to update discovery config: " + err.Error())
		}
	default:
		return 0, nil, errors.New("invalid request type")
	}

	return discovery.ID, discovery, nil
}

func (m *InferFlow) createOrUpdateInferFlowConfig(tx *gorm.DB, requestEntry *inferflow_request.Table, discoveryID int, configExistedBeforeTx bool) error {
	newConfig := &inferflow_config.Table{
		DiscoveryID: discoveryID,
		ConfigID:    requestEntry.ConfigID,
		Active:      activeTrue,
		ConfigValue: requestEntry.Payload.ConfigValue,
	}

	switch requestEntry.RequestType {
	case onboardRequestType, cloneRequestType:
		if requestEntry.UpdatedBy != "" {
			newConfig.CreatedBy = requestEntry.UpdatedBy
		} else {
			newConfig.CreatedBy = requestEntry.CreatedBy
		}
		return m.InferFlowConfigRepo.CreateTx(tx, newConfig)
	case scaleUpRequestType:
		if requestEntry.UpdatedBy != "" {
			newConfig.CreatedBy = requestEntry.UpdatedBy
		} else {
			newConfig.CreatedBy = requestEntry.CreatedBy
		}
		newConfig.SourceConfigID = requestEntry.Payload.ConfigMapping.SourceConfigID
		return m.InferFlowConfigRepo.CreateTx(tx, newConfig)
	case promoteRequestType:
		if !configExistedBeforeTx {
			if requestEntry.UpdatedBy != "" {
				newConfig.CreatedBy = requestEntry.UpdatedBy
			} else {
				newConfig.CreatedBy = requestEntry.CreatedBy
			}
			return m.InferFlowConfigRepo.CreateTx(tx, newConfig)
		} else {
			existingConfig, err := m.InferFlowConfigRepo.GetByID(requestEntry.ConfigID)
			if err != nil {
				return errors.New("failed to query inferflow config repo: " + err.Error())
			}
			newConfig.ID = existingConfig.ID
			if requestEntry.UpdatedBy != "" {
				newConfig.UpdatedBy = requestEntry.UpdatedBy
			} else {
				newConfig.UpdatedBy = requestEntry.CreatedBy
			}
			return m.InferFlowConfigRepo.UpdateTx(tx, newConfig)
		}
	case editRequestType:
		existingConfig, err := m.InferFlowConfigRepo.GetByID(requestEntry.ConfigID)
		if err != nil {
			return errors.New("failed to get inferflow config by id: " + err.Error())
		}
		if existingConfig == nil {
			return errors.New("failed to get inferflow config by id")
		}
		newConfig.ID = existingConfig.ID
		if requestEntry.UpdatedBy != "" {
			newConfig.UpdatedBy = requestEntry.UpdatedBy
		} else {
			newConfig.UpdatedBy = requestEntry.CreatedBy
		}
		return m.InferFlowConfigRepo.UpdateTx(tx, newConfig)
	case deleteRequestType:
		existingConfig, err := m.InferFlowConfigRepo.GetByID(requestEntry.ConfigID)
		if err != nil {
			return errors.New("failed to get inferflow config by id: " + err.Error())
		}
		if existingConfig == nil {
			return errors.New("failed to get inferflow config by id")
		}
		newConfig.ID = existingConfig.ID
		if requestEntry.UpdatedBy != "" {
			newConfig.UpdatedBy = requestEntry.UpdatedBy
		} else {
			newConfig.UpdatedBy = requestEntry.CreatedBy
		}
		newConfig.Active = activeFalse
		return m.InferFlowConfigRepo.UpdateTx(tx, newConfig)
	default:
		return errors.New("invalid request type")
	}
}

func (m *InferFlow) createOrUpdateEtcdConfig(table *inferflow_request.Table, discovery *discovery_config.DiscoveryConfig, configExistedBeforeTx bool) error {
	serviceDeployableTable, err := m.ServiceDeployableConfigRepo.GetById(int(discovery.ServiceDeployableID))
	if err != nil {
		return errors.New("failed to get service deployable config by id: " + err.Error())
	}
	serviceName := strings.ToLower(serviceDeployableTable.Name)
	configId := table.ConfigID
	inferFlowConfig := AdaptToEtcdInferFlowConfig(table.Payload.ConfigValue)

	switch table.RequestType {
	case onboardRequestType, cloneRequestType, scaleUpRequestType:
		return m.EtcdConfig.CreateConfig(serviceName, configId, inferFlowConfig)
	case promoteRequestType:
		if !configExistedBeforeTx {
			return m.EtcdConfig.CreateConfig(serviceName, configId, inferFlowConfig)
		}
		return m.EtcdConfig.UpdateConfig(serviceName, configId, inferFlowConfig)
	case editRequestType:
		return m.EtcdConfig.UpdateConfig(serviceName, configId, inferFlowConfig)
	case deleteRequestType:
		return m.EtcdConfig.DeleteConfig(serviceName, configId)
	default:
		return errors.New("invalid request type")
	}
}

func (m *InferFlow) GetAllRequests(request GetAllRequestConfigsRequest) (GetAllRequestConfigsResponse, error) {

	var tables []inferflow_request.Table
	var err error
	request.Role = strings.ToUpper(request.Role)
	if request.Role == adminRole {

		tables, err = m.InferFlowRequestRepo.GetAll()
		if err != nil {
			return GetAllRequestConfigsResponse{}, errors.New("failed to get all infer flow request configs: " + err.Error())
		}
	} else {

		tables, err = m.InferFlowRequestRepo.GetByUser(request.Email)
		if err != nil {
			return GetAllRequestConfigsResponse{}, errors.New("failed to get all infer flow request configs by user: " + err.Error())
		}
	}

	deployableIDsMap := make(map[int]bool)
	for _, table := range tables {
		payload := table.Payload
		if payload.ConfigMapping.DeployableID > 0 {
			deployableIDsMap[payload.ConfigMapping.DeployableID] = true
		}
	}

	deployableIDs := make([]int, 0, len(deployableIDsMap))
	for id := range deployableIDsMap {
		deployableIDs = append(deployableIDs, id)
	}

	serviceDeployables, err := m.ServiceDeployableConfigRepo.GetByIds(deployableIDs)
	if err != nil {
		return GetAllRequestConfigsResponse{}, errors.New("failed to get service deployable configs: " + err.Error())
	}

	deployableMap := make(map[int]string)
	for _, sd := range serviceDeployables {
		deployableMap[sd.ID] = sd.Name
	}

	requestConfigs := make([]RequestConfig, len(tables))
	for i, table := range tables {

		payload := table.Payload
		ConfigMapping := AdaptFromDbToConfigMapping(payload.ConfigMapping)

		ConfigValue := AdaptFromDbToInferFlowConfig(payload.ConfigValue)

		if name, exists := deployableMap[ConfigMapping.DeployableID]; exists {
			ConfigMapping.DeployableName = name
		} else {
			ConfigMapping.DeployableName = "Unknown"
		}

		requestConfigs[i] = RequestConfig{
			RequestID: table.RequestID,
			Payload: Payload{
				ConfigMapping: ConfigMapping,
				ConfigValue:   ConfigValue,
			},
			ConfigID:     table.ConfigID,
			CreatedBy:    table.CreatedBy,
			CreatedAt:    table.CreatedAt,
			UpdatedBy:    table.UpdatedBy,
			UpdatedAt:    table.UpdatedAt,
			RequestType:  table.RequestType,
			RejectReason: table.RejectReason,
			Status:       table.Status,
			Reviewer:     table.Reviewer,
		}
	}

	response := GetAllRequestConfigsResponse{
		Error: emptyResponse,
		Data:  requestConfigs,
	}
	return response, nil
}

func (m *InferFlow) GetAll() (GetAllResponse, error) {

	tables, err := m.InferFlowConfigRepo.GetAll()
	if err != nil {
		return GetAllResponse{}, errors.New("failed to get all inferflow configs: " + err.Error())
	}

	discoveryIDs := make([]int, 0, len(tables))
	for _, table := range tables {
		discoveryIDs = append(discoveryIDs, int(table.DiscoveryID))
	}

	discoveryMap, serviceDeployableMap, err := m.batchFetchDiscoveryConfigs(discoveryIDs)
	if err != nil {
		return GetAllResponse{}, errors.New(err.Error())
	}

	ringMasterConfigs, err := m.batchFetchRingMasterConfigs(serviceDeployableMap)
	if err != nil {
		return GetAllResponse{}, errors.New("failed to batch fetch ringmaster configs: " + err.Error())
	}

	responseConfigs := make([]ConfigTable, len(tables))
	for i, table := range tables {
		disocveryTable, exists := discoveryMap[int(table.DiscoveryID)]
		if !exists {
			return GetAllResponse{}, errors.New("failed to find discovery config by id: " + strconv.Itoa(int(table.DiscoveryID)))
		}

		serviceDeployableTable, exists := serviceDeployableMap[disocveryTable.ServiceDeployableID]
		if !exists {
			return GetAllResponse{}, errors.New("failed to find service deployable config by id: " + strconv.Itoa(disocveryTable.ServiceDeployableID))
		}

		ConfigValue := AdaptFromDbToInferFlowConfig(table.ConfigValue)

		ringMasterConfig := ringMasterConfigs[serviceDeployableTable.ID]

		responseConfigs[i] = ConfigTable{
			ConfigID:                table.ConfigID,
			ConfigValue:             ConfigValue,
			Host:                    serviceDeployableTable.Host,
			DeployableRunningStatus: ringMasterConfig.RunningStatus == "true",
			MonitoringUrl:           serviceDeployableTable.MonitoringUrl,
			CreatedBy:               table.CreatedBy,
			UpdatedBy:               table.UpdatedBy,
			CreatedAt:               table.CreatedAt,
			UpdatedAt:               table.UpdatedAt,
			TestResults: TestResults{
				Tested:  table.TestResults.Tested,
				Message: table.TestResults.Message,
			},
			SourceConfigID: table.SourceConfigID,
		}

	}

	response := GetAllResponse{
		Error: emptyResponse,
		Data:  responseConfigs,
	}
	return response, nil
}

func (m *InferFlow) batchFetchDiscoveryConfigs(discoveryIDs []int) (
	map[int]*discovery_config.DiscoveryConfig,
	map[int]*service_deployable_config.ServiceDeployableConfig,
	error,
) {
	emptyDiscoveryMap := make(map[int]*discovery_config.DiscoveryConfig)
	emptyServiceDeployableMap := make(map[int]*service_deployable_config.ServiceDeployableConfig)

	if len(discoveryIDs) == 0 {
		return emptyDiscoveryMap, emptyServiceDeployableMap, nil
	}

	discoveryConfigs, err := m.DiscoveryConfigRepo.GetByServiceDeployableIDs(discoveryIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get discovery configs: %w", err)
	}

	discoveryMap := make(map[int]*discovery_config.DiscoveryConfig)
	for i := range discoveryConfigs {
		discoveryMap[discoveryConfigs[i].ID] = &discoveryConfigs[i]
	}

	serviceDeployableIDsMap := make(map[int]bool)
	for _, dc := range discoveryConfigs {
		serviceDeployableIDsMap[dc.ServiceDeployableID] = true
	}

	serviceDeployableIDs := make([]int, 0, len(serviceDeployableIDsMap))
	for id := range serviceDeployableIDsMap {
		serviceDeployableIDs = append(serviceDeployableIDs, id)
	}

	if len(serviceDeployableIDs) == 0 {
		return discoveryMap, emptyServiceDeployableMap, nil
	}

	serviceDeployables, err := m.ServiceDeployableConfigRepo.GetByIds(serviceDeployableIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get service deployable configs: %w", err)
	}

	serviceDeployableMap := make(map[int]*service_deployable_config.ServiceDeployableConfig)
	for i := range serviceDeployables {
		serviceDeployableMap[serviceDeployables[i].ID] = &serviceDeployables[i]
	}

	return discoveryMap, serviceDeployableMap, nil
}

func (m *InferFlow) batchFetchRingMasterConfigs(serviceDeployables map[int]*service_deployable_config.ServiceDeployableConfig) (map[int]infrastructurehandler.Config, error) {
	ringMasterConfigs := make(map[int]infrastructurehandler.Config)
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 10)

	for id, deployable := range serviceDeployables {
		wg.Add(1)
		go func(deployableID int, sd *service_deployable_config.ServiceDeployableConfig) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			config := m.infrastructureHandler.GetConfig(sd.Name, inferflowPkg.AppEnv)

			mu.Lock()
			ringMasterConfigs[deployableID] = config
			mu.Unlock()
		}(id, deployable)
	}

	wg.Wait()
	return ringMasterConfigs, nil
}

func (m *InferFlow) ValidateRequest(request ValidateRequest, token string) (Response, error) {
	tables, err := m.InferFlowRequestRepo.GetAll()
	if err != nil {
		return Response{
			Error: "failed to get all infer flow request configs: " + err.Error(),
			Data:  Message{Message: emptyResponse},
		}, err
	}

	configTable := inferflow_request.Table{}
	for _, table := range tables {
		if table.ConfigID == request.ConfigID {
			configTable = table
		}
	}

	configValue := AdaptFromDbToInferFlowConfig(configTable.Payload.ConfigValue)

	return ValidateInferFlowConfig(configValue, token)
}

func ValidateInferFlowConfig(config InferflowConfig, token string) (Response, error) {
	ComponentConfig := config.ComponentConfig
	if ComponentConfig != nil {
		for _, featureComponent := range ComponentConfig.FeatureComponents {
			entity := featureComponent.FSRequest.Label
			if entity == "dummy" {
				continue
			}
			response, err := mainHandler.Client.ValidateOnlineFeatures(entity, token)
			if err != nil {
				return Response{
					Error: "failed to validate feature exists: " + err.Error(),
					Data:  Message{Message: emptyResponse},
				}, err
			}
			for _, fg := range featureComponent.FSRequest.FeatureGroups {
				featureMap := make(map[string]bool)
				for _, feature := range fg.Features {
					if _, exists := featureMap[feature]; exists {
						return Response{
							Error: "feature " + feature + " is duplicated",
							Data:  Message{Message: emptyResponse},
						}, errors.New("feature " + feature + " is duplicated")
					}
					featureMap[feature] = true
					if !mainHandler.ValidateFeatureExists(fg.Label+COLON_DELIMITER+feature, response) {
						return Response{
							Error: "feature \"" + entity + COLON_DELIMITER + fg.Label + COLON_DELIMITER + feature + "\" does not exist",
							Data:  Message{Message: emptyResponse},
						}, errors.New("feature \"" + entity + COLON_DELIMITER + fg.Label + COLON_DELIMITER + feature + "\" does not exist")
					}
				}
			}
		}

		for _, predatorComponent := range config.ComponentConfig.PredatorComponents {
			outputMap := make(map[string]bool)
			for _, output := range predatorComponent.Outputs {
				for _, modelScore := range output.ModelScores {
					if _, exists := outputMap[modelScore]; exists {
						return Response{
							Error: "model score " + modelScore + " is duplicated for component " + predatorComponent.Component,
							Data:  Message{Message: emptyResponse},
						}, errors.New("model score " + modelScore + " is duplicated for component " + predatorComponent.Component)
					}
					outputMap[modelScore] = true
				}
			}
		}
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: "Request validated successfully"},
	}, nil
}

func (m *InferFlow) ValidateOnboardRequest(request OnboardPayload) (Response, error) {
	outputs := mapset.NewSet[string]()
	deployableConfig, err := m.ServiceDeployableConfigRepo.GetById(request.ConfigMapping.DeployableID)
	if err != nil {
		return Response{
			Error: "Failed to fetch deployable config for the request",
			Data:  Message{Message: emptyResponse},
		}, errors.New("Failed to fetch deployable config for the request")
	}
	permissibleEndpoints := m.EtcdConfig.GetConfiguredEndpoints(deployableConfig.Name)
	for _, ranker := range request.Rankers {
		if len(ranker.EntityID) == 0 {
			return Response{
				Error: "Entity ID is not set for model: " + ranker.ModelName,
				Data:  Message{Message: emptyResponse},
			}, errors.New("Entity ID is not set for model: " + ranker.ModelName)
		}
		if !permissibleEndpoints.Contains(ranker.EndPoint) {
			errorMsg := fmt.Sprintf(
				"invalid endpoint: %s chosen for service deployable: %s for model: %s",
				ranker.EndPoint, deployableConfig.Name, ranker.ModelName,
			)
			return Response{
				Error: errorMsg,
				Data:  Message{Message: emptyResponse},
			}, errors.New(errorMsg)
		}
		for _, output := range ranker.Outputs {
			if len(output.ModelScores) != len(output.ModelScoresDims) {
				return Response{
					Error: "model scores and model scores dims are not equal for model: " + ranker.ModelName,
					Data:  Message{Message: emptyResponse},
				}, errors.New("model scores and model scores dims are not equal for model: " + ranker.ModelName)
			}
			for _, modelScore := range output.ModelScores {
				if outputs.Contains(modelScore) {
					return Response{
						Error: "duplicate model scores: " + modelScore + " for model: " + ranker.ModelName,
						Data:  Message{Message: emptyResponse},
					}, errors.New("duplicate model scores: " + modelScore + " for model: " + ranker.ModelName)
				}
				outputs.Add(modelScore)
			}
		}
	}

	for _, reRanker := range request.ReRankers {
		if len(reRanker.EntityID) == 0 {
			return Response{
				Error: "Entity ID is not set for re ranker: " + reRanker.Score,
				Data:  Message{Message: emptyResponse},
			}, errors.New("Entity ID is not set for re ranker: " + reRanker.Score)
		}
		for _, value := range reRanker.EqVariables {
			parts := strings.Split(value, PIPE_DELIMITER)
			if len(parts) != 2 {
				return Response{
					Error: "invalid eq variable: " + value,
					Data:  Message{Message: emptyResponse},
				}, errors.New("invalid eq variable: " + value)
			}
			if parts[1] == "" {
				return Response{
					Error: "invalid eq variable: " + value,
					Data:  Message{Message: emptyResponse},
				}, errors.New("invalid eq variable: " + value)
			}
		}
		if outputs.Contains(reRanker.Score) {
			return Response{
				Error: "duplicate score: " + reRanker.Score + " for reRanker: " + reRanker.Score,
				Data:  Message{Message: emptyResponse},
			}, errors.New("duplicate score: " + reRanker.Score + " for reRanker: " + reRanker.Score)
		}
		outputs.Add(reRanker.Score)
	}

	// Validate MODEL_FEATURE list
	for _, ranker := range request.Rankers {
		for _, input := range ranker.Inputs {
			for _, feature := range input.Features {
				featureParts := strings.Split(feature, PIPE_DELIMITER)
				if len(featureParts) != 2 {
					return Response{
						Error: "invalid feature: " + feature + " in input features of ranker: " + ranker.ModelName,
						Data:  Message{Message: emptyResponse},
					}, errors.New("invalid feature: " + feature + " in input features of ranker: " + ranker.ModelName)
				}
				if strings.Contains(featureParts[0], MODEL_FEATURE) {
					if !outputs.Contains(featureParts[1]) {
						return Response{
							Error: "model score " + featureParts[1] + " is not found in other model scores of ranker: " + ranker.ModelName,
							Data:  Message{Message: emptyResponse},
						}, errors.New("model score " + featureParts[1] + " is not found in other model scores of ranker: " + ranker.ModelName)
					}
				}
			}
		}
	}

	for _, reRanker := range request.ReRankers {
		for _, feature := range reRanker.EqVariables {
			featureParts := strings.Split(feature, PIPE_DELIMITER)
			if len(featureParts) != 2 {
				return Response{
					Error: "invalid feature: " + feature,
					Data:  Message{Message: emptyResponse},
				}, errors.New("invalid feature: " + feature)
			}
			if strings.Contains(featureParts[0], MODEL_FEATURE) {
				if !outputs.Contains(featureParts[1]) {
					return Response{
						Error: "model score " + featureParts[1] + " is not found in other model scores of re ranker: " + strconv.Itoa(reRanker.EqID),
						Data:  Message{Message: emptyResponse},
					}, errors.New("model score " + featureParts[1] + " is not found in other model scores of re ranker: " + strconv.Itoa(reRanker.EqID))
				}
			}
		}
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: "Request validated successfully"},
	}, nil
}

func (m *InferFlow) GenerateFunctionalTestRequest(request GenerateRequestFunctionalTestingRequest) (GenerateRequestFunctionalTestingResponse, error) {

	response := GenerateRequestFunctionalTestingResponse{
		RequestBody: RequestBody{
			Entities: []Entity{
				{
					Entity:   request.Entity + "_id",
					Ids:      []string{},
					Features: []FeatureValue{},
				},
			},
			ModelConfigID: request.ModelConfigID,
		},
	}

	batchSize, err := strconv.Atoi(request.BatchSize)
	if err != nil {
		response.Error = fmt.Errorf("invalid batch size: %w", err).Error()
		return response, errors.New("invalid batch size: " + err.Error())
	}

	response.RequestBody.Entities[0].Entity = request.Entity + "_id"
	response.RequestBody.Entities[0].Ids = random.GenerateRandomIntSliceWithRange(batchSize, 100000, 1000000)

	for feature, value := range request.DefaultFeatures {
		featureValues := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			featureValues[i] = value
		}
		response.RequestBody.Entities[0].Features = append(response.RequestBody.Entities[0].Features, FeatureValue{
			Name:            feature,
			IdsFeatureValue: featureValues,
		})
	}

	response.MetaData = request.MetaData

	return response, nil
}

func (m *InferFlow) ExecuteFuncitonalTestRequest(request ExecuteRequestFunctionalTestingRequest) (ExecuteRequestFunctionalTestingResponse, error) {
	response := ExecuteRequestFunctionalTestingResponse{}

	normalizedEndpoint := func(raw string) string {
		ep := strings.TrimSpace(raw)
		if strings.HasPrefix(ep, "http://") {
			ep = strings.TrimPrefix(ep, "http://")
		} else if strings.HasPrefix(ep, "https://") {
			ep = strings.TrimPrefix(ep, "https://")
		}
		ep = strings.TrimSuffix(ep, "/")
		if idx := strings.LastIndex(ep, ":"); idx != -1 {
			if idx < len(ep)-1 {
				ep = ep[:idx]
			}
		}

		port := ":8080"
		env := strings.ToLower(strings.TrimSpace(inferflowPkg.AppEnv))
		if env == "stg" || env == "int" {
			port = ":80"
		}
		ep = ep + port
		return ep
	}(request.EndPoint)

	conn, err := grpc.GetConnection(normalizedEndpoint)
	if err != nil {
		response.Error = err.Error()
		return response, errors.New("failed to get connection: " + err.Error())
	}

	protoRequest := &pb.InferflowRequestProto{}
	protoRequest.ModelConfigId = request.RequestBody.ModelConfigID

	md := metadata.New(nil)
	if len(request.MetaData) > 0 {
		for key, value := range request.MetaData {
			md.Set(key, value)
		}
	}

	md.Set(setFunctionalTest, "true")

	protoRequest.Entities = make([]*pb.InferflowRequestProto_Entity, len(request.RequestBody.Entities))
	for i, entity := range request.RequestBody.Entities {

		protoFeatures := make([]*pb.InferflowRequestProto_Entity_Feature, len(entity.Features))

		for j, feature := range entity.Features {
			protoFeatures[j] = &pb.InferflowRequestProto_Entity_Feature{
				Name:            feature.Name,
				IdsFeatureValue: feature.IdsFeatureValue,
			}
		}

		protoRequest.Entities[i] = &pb.InferflowRequestProto_Entity{
			Entity:   entity.Entity,
			Ids:      entity.Ids,
			Features: protoFeatures,
		}
	}

	protoResponse := &pb.InferflowResponseProto{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()
	err = grpc.SendGRPCRequest(ctx, conn, inferFlowRetrieveModelScoreMethod, protoRequest, protoResponse, md)
	if err != nil {
		response.Error = err.Error()
		log.Error().Msgf("error: %v", err)
		return response, errors.New("failed to send grpc request: " + err.Error())
	}

	for _, compData := range protoResponse.GetComponentData() {
		response.ComponentData = append(response.ComponentData, ComponentData{
			Data: compData.GetData(),
		})
	}

	for i, compData := range response.ComponentData {
		if i == 0 {
			continue
		}
		for j, data := range compData.Data {
			if data == "" {
				response.Error = fmt.Sprintf("response data is empty for field: %s", response.ComponentData[0].Data[j])
				break
			}
		}
	}

	if protoResponse.GetError() != nil {
		response.Error = protoResponse.GetError().GetMessage()
	}

	inferFlowConfig, err := m.InferFlowConfigRepo.GetByID(request.RequestBody.ModelConfigID)

	if err != nil {
		fmt.Println("Error getting inferflow config: ", err)
	} else if inferFlowConfig == nil {
		log.Error().Msgf("inferflow config '%s' does not exist in DB", request.RequestBody.ModelConfigID)
	} else {
		if response.Error != emptyResponse {
			inferFlowConfig.TestResults = inferflow.TestResults{
				Tested:  false,
				Message: response.Error,
			}
		} else {
			inferFlowConfig.TestResults = inferflow.TestResults{
				Tested:  true,
				Message: "Functional test request executed successfully",
			}
		}
		err = m.InferFlowConfigRepo.Update(inferFlowConfig)
		if err != nil {
			fmt.Println("Error updating inferflow config: ", err)
		}
	}

	return response, nil
}

func (m *InferFlow) GetLatestRequest(requestID string) (GetLatestRequestResponse, error) {
	requests, err := m.InferFlowRequestRepo.GetApprovedRequestsByConfigID(requestID)
	if err != nil {
		return GetLatestRequestResponse{
			Error: "failed to get latest request: " + err.Error(),
			Data:  RequestConfig{},
		}, err
	}

	latestRequest := inferflow_request.Table{}
	if len(requests) > 0 {
		latestRequest = requests[0]
	}

	if latestRequest.Payload.RequestPayload.ConfigIdentifier == "" {
		return GetLatestRequestResponse{
			Error: "failed to find latest request",
			Data:  RequestConfig{},
		}, err
	}

	return GetLatestRequestResponse{
		Error: emptyResponse,
		Data: RequestConfig{
			RequestID: latestRequest.RequestID,
			Payload: Payload{
				ConfigMapping:  AdaptFromDbToConfigMapping(latestRequest.Payload.ConfigMapping),
				RequestPayload: AdaptFromDbToOnboardPayload(latestRequest.Payload.RequestPayload),
			},
			ConfigID:     latestRequest.ConfigID,
			CreatedBy:    latestRequest.CreatedBy,
			CreatedAt:    latestRequest.CreatedAt,
			UpdatedBy:    latestRequest.UpdatedBy,
			UpdatedAt:    latestRequest.UpdatedAt,
			RequestType:  latestRequest.RequestType,
			RejectReason: latestRequest.RejectReason,
			Status:       latestRequest.Status,
			Reviewer:     latestRequest.Reviewer,
		},
	}, nil
}

func (m *InferFlow) GetDerivedConfigID(configID string, deployableID int) (string, error) {
	serviceDeployableConfig, err := m.ServiceDeployableConfigRepo.GetById(deployableID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch service service deployable config for name generation: %w", err)
	}
	deployableTag := serviceDeployableConfig.DeployableTag
	if deployableTag == "" {
		return configID, nil
	}

	derivedConfigID := configID + deployableTagDelimiter + deployableTag + deployableTagDelimiter + scaleupTag
	return derivedConfigID, nil
}

func (m *InferFlow) GetLoggingTTL() (GetLoggingTTLResponse, error) {
	return GetLoggingTTLResponse{
		Data: []int{30, 60, 90},
	}, nil
}

func (m *InferFlow) GetFeatureSchema(request FeatureSchemaRequest) (FeatureSchemaResponse, error) {
	version, err := strconv.Atoi(request.Version)
	if err != nil {
		return FeatureSchemaResponse{
			Data: []inferflow.SchemaComponents{},
		}, err
	}
	inferflowRequests, err := m.InferFlowRequestRepo.GetByConfigIDandVersion(request.ModelConfigId, version)
	if err != nil {
		log.Error().Err(err).Str("model_config_id", request.ModelConfigId).Msg("Failed to get inferflow config")
		return FeatureSchemaResponse{
			Data: []inferflow.SchemaComponents{},
		}, err
	}
	inferflowConfig := inferflowRequests[0].Payload
	componentConfig := &inferflowConfig.ConfigValue.ComponentConfig
	responseConfig := &inferflowConfig.ConfigValue.ResponseConfig

	response := BuildFeatureSchemaFromInferflow(componentConfig, responseConfig)

	if responseConfig.LogSelectiveFeatures {
		responseSchemaComponents := ProcessResponseConfigFromInferflow(responseConfig, response)
		log.Info().Str("model_config_id", request.ModelConfigId).Int("schema_components_count", len(responseSchemaComponents)).Msg("Successfully generated feature schema")
		return FeatureSchemaResponse{
			Data: responseSchemaComponents,
		}, nil
	}

	log.Info().Str("model_config_id", request.ModelConfigId).Int("schema_components_count", len(response)).Msg("Successfully generated feature schema")
	return FeatureSchemaResponse{
		Data: response,
	}, nil
}
