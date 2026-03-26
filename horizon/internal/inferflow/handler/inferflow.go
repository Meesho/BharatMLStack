package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	mainHandler "github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	discovery_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	inferflow_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/config"
	inferflow_request "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/request"
	service_deployable_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
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
			return Response{}, errors.New("this inferflow config has reached its version limit. Please create a clone to make further updates")
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
		return Response{}, errors.New("this inferflow config has reached its version limit. Please create a clone to make further updates")
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
	if latestSourceRequest.Error != emptyResponse {
		return Response{}, errors.New("failed to get latest request for the source configID: " + sourceConfigID + ": " + fmt.Sprint(latestSourceRequest.Error))
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
	if len(inferflowRequests) == 0 {
		return FeatureSchemaResponse{
			Data: []inferflow.SchemaComponents{},
		}, errors.New("no inferflow config found for model_config_id=" + request.ModelConfigId + " version=" + strconv.Itoa(version))
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
