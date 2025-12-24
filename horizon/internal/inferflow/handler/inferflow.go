package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	mainHandler "github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	inferflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/inferflow"
	etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	pb "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/handler/proto/protogen"
	discovery_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	inferflow_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/config"
	inferflow_request "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/request"
	service_deployable_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/grpc"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/random"
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
	ringMasterClient            mainHandler.RingmasterClient
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

		ringMasterClient := mainHandler.GetRingmasterClient()

		config = &InferFlow{
			EtcdConfig:                  etcd.NewEtcdInstance(),
			InferFlowRequestRepo:        InferFlowRequestRepo,
			InferFlowConfigRepo:         InferFlowConfigRepo,
			DiscoveryConfigRepo:         DiscoveryConfigRepo,
			ServiceDeployableConfigRepo: ServiceDeployableConfigRepo,
			ringMasterClient:            ringMasterClient,
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
		return Response{}, errors.New("model proxy config is invalid: " + response.Error)
	}

	exists := false
	configs, err := m.InferFlowConfigRepo.GetByID(configId)
	if err == nil {
		exists = configs.Active
	}
	if exists {
		return Response{}, errors.New("Config ID already exists")
	}

	inferFlowConfig, err := m.GetInferflowConfig(request, token)
	if err != nil {
		return Response{}, errors.New("failed to generate model proxy config: " + err.Error())
	}

	response, err = ValidateInferFlowConfig(inferFlowConfig, token)
	if err != nil {
		return Response{}, errors.New("failed to validate model proxy config: " + err.Error())
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
		return Response{}, errors.New("failed to validate onboard request: " + err.Error())
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

	newVersion := 1
	if len(existingConfigs) > 0 {
		newVersion = existingConfigs[0].Version + 1
	}

	onboardRequest := InferflowOnboardRequest(request)

	inferFlowConfig, err := m.GetInferflowConfig(onboardRequest, token)
	if err != nil {
		return Response{}, errors.New("failed to get infer flow config: " + err.Error())
	}

	response, err = ValidateInferFlowConfig(inferFlowConfig, token)
	if err != nil {
		return Response{}, errors.New("failed to validate model proxy config: " + err.Error())
	}
	if response.Error != emptyResponse {
		return Response{}, errors.New("infer flow config is invalid: " + response.Error)
	}

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
	exists, err := m.InferFlowRequestRepo.DoesConfigIdExistWithRequestType(request.Payload.ConfigID, scaleUpRequestType)
	if err != nil {
		return Response{}, errors.New("failed to check if config id exists in db: " + err.Error())
	}
	if exists {
		return Response{}, errors.New("Config ID already exists with scale up request")
	}

	modelNameToEndPointMap := make(map[string]ModelNameToEndPointMap)
	for _, proposedModelEndpoint := range request.Payload.ModelNameToEndPointMap {
		modelNameToEndPointMap[proposedModelEndpoint.CurrentModelName] = proposedModelEndpoint
	}

	for i := range request.Payload.ConfigValue.ComponentConfig.PredatorComponents {
		modelName := request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelName
		request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelEndPoint = modelNameToEndPointMap[modelName].EndPointID
		request.Payload.ConfigValue.ComponentConfig.PredatorComponents[i].ModelName = modelNameToEndPointMap[modelName].NewModelName
	}

	request.Payload.ConfigValue.ResponseConfig.LoggingPerc = request.Payload.LoggingPerc

	payload, err := AdaptScaleUpRequestToDBPayload(request)
	if err != nil {
		return Response{}, errors.New("failed to adapt scale up request to db payload: " + err.Error())
	}

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

	err := m.InferFlowRequestRepo.Transaction(func(tx *gorm.DB) error {

		request.Status = strings.ToUpper(request.Status)

		if request.Status != approved && request.Status != rejected {
			return errors.New("invalid status")
		}

		if request.Status == rejected {
			if request.RejectReason == emptyResponse {
				return errors.New("rejecttion reason is needed")
			}
		}

		exists, err := m.InferFlowRequestRepo.DoesRequestIDExistWithStatus(request.RequestID, pendingApproval)
		if err != nil {
			return errors.New("failed to check if request id exists with status: " + err.Error())
		}
		if !exists {
			return errors.New("request id does not exist or is not pending approval")
		}

		fullTable := &inferflow_request.Table{}
		if err := tx.First(fullTable, request.RequestID).Error; err != nil {
			return errors.New("failed to get infer flow config request by id: " + err.Error())
		}

		table := &inferflow_request.Table{
			RequestID:    request.RequestID,
			Status:       request.Status,
			RejectReason: request.RejectReason,
			Reviewer:     request.Reviewer,
		}
		if request.Status == rejected {
			table.Active = activeFalse
		}

		err = m.InferFlowRequestRepo.UpdateTx(tx, table)
		if err != nil {
			return errors.New("failed to update infer flow config request in db: " + err.Error())
		}

		if request.Status == approved {
			discoveryID, discovery, err := m.createOrUpdateDiscoveryConfig(tx, fullTable)
			if err != nil {
				return err
			}

			err = m.createOrUpdateInferFlowConfig(tx, fullTable, discoveryID)
			if err != nil {
				return err
			}

			err = m.createOrUpdateEtcdConfig(fullTable, discovery)
			if err != nil {
				log.Error().Err(err).Msg("Failed to sync config to etcd")
				return errors.New("failed to sync config to etcd: " + err.Error())
			}
		}
		return nil
	})

	if err != nil {
		return Response{}, errors.New("failed to review config: " + err.Error())
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: "Mp Config reviewed successfully."},
	}, nil
}

func (m *InferFlow) createOrUpdateDiscoveryConfig(tx *gorm.DB, table *inferflow_request.Table) (int, *discovery_config.DiscoveryConfig, error) {
	discovery := &discovery_config.DiscoveryConfig{
		ServiceDeployableID: table.Payload.ConfigMapping.DeployableID,
		AppToken:            table.Payload.ConfigMapping.AppToken,
		ServiceConnectionID: table.Payload.ConfigMapping.ConnectionConfigID,
		Active:              activeTrue,
	}

	switch table.RequestType {
	case onboardRequestType, cloneRequestType, promoteRequestType, scaleUpRequestType:
		if table.UpdatedBy != "" {
			discovery.CreatedBy = table.UpdatedBy
		} else {
			discovery.CreatedBy = table.CreatedBy
		}
		err := m.DiscoveryConfigRepo.CreateTx(tx, discovery)
		if err != nil {
			return 0, nil, errors.New("failed to create discovery config: " + err.Error())
		}
	case editRequestType:
		if table.UpdatedBy != "" {
			discovery.UpdatedBy = table.UpdatedBy
		} else {
			discovery.UpdatedBy = table.CreatedBy
		}
		config, err := m.InferFlowConfigRepo.GetByID(table.ConfigID)
		if err != nil {
			return 0, nil, errors.New("failed to get model proxy config by id: " + err.Error())
		}
		discovery.ID = int(config.DiscoveryID)
		err = m.DiscoveryConfigRepo.UpdateTx(tx, discovery)
		if err != nil {
			return 0, nil, errors.New("failed to update discovery config: " + err.Error())
		}
	case deleteRequestType:
		config, err := m.InferFlowConfigRepo.GetByID(table.ConfigID)
		if err != nil {
			return 0, nil, errors.New("failed to get model proxy config by id: " + err.Error())
		}
		if table.UpdatedBy != "" {
			discovery.UpdatedBy = table.UpdatedBy
		} else {
			discovery.UpdatedBy = table.CreatedBy
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

func (m *InferFlow) createOrUpdateInferFlowConfig(tx *gorm.DB, table *inferflow_request.Table, discoveryID int) error {
	newTable := &inferflow_config.Table{
		DiscoveryID: discoveryID,
		ConfigID:    table.ConfigID,
		Active:      activeTrue,
		ConfigValue: table.Payload.ConfigValue,
	}

	switch table.RequestType {
	case onboardRequestType, cloneRequestType, promoteRequestType, scaleUpRequestType:
		if table.UpdatedBy != "" {
			newTable.CreatedBy = table.UpdatedBy
		} else {
			newTable.CreatedBy = table.CreatedBy
		}
		return m.InferFlowConfigRepo.CreateTx(tx, newTable)
	case editRequestType:
		existingConfig, err := m.InferFlowConfigRepo.GetByID(table.ConfigID)
		if err != nil {
			return errors.New("failed to get model proxy config by id: " + err.Error())
		}
		newTable.ID = existingConfig.ID
		if table.UpdatedBy != "" {
			newTable.UpdatedBy = table.UpdatedBy
		} else {
			newTable.UpdatedBy = table.CreatedBy
		}
		return m.InferFlowConfigRepo.UpdateTx(tx, newTable)
	case deleteRequestType:
		existingConfig, err := m.InferFlowConfigRepo.GetByID(table.ConfigID)
		if err != nil {
			return errors.New("failed to get model proxy config by id: " + err.Error())
		}
		newTable.ID = existingConfig.ID
		if table.UpdatedBy != "" {
			newTable.UpdatedBy = table.UpdatedBy
		} else {
			newTable.UpdatedBy = table.CreatedBy
		}
		newTable.Active = activeFalse
		return m.InferFlowConfigRepo.UpdateTx(tx, newTable)
	default:
		return errors.New("invalid request type")
	}
}

func (m *InferFlow) createOrUpdateEtcdConfig(table *inferflow_request.Table, discovery *discovery_config.DiscoveryConfig) error {
	serviceDeployableTable, err := m.ServiceDeployableConfigRepo.GetById(int(discovery.ServiceDeployableID))
	if err != nil {
		return errors.New("failed to get service deployable config by id: " + err.Error())
	}
	serviceName := strings.ToLower(serviceDeployableTable.Name)
	configId := table.ConfigID
	inferFlowConfig := AdaptToEtcdInferFlowConfig(table.Payload.ConfigValue)

	switch table.RequestType {
	case onboardRequestType, cloneRequestType, promoteRequestType, scaleUpRequestType:
		return m.EtcdConfig.CreateConfig(serviceName, configId, inferFlowConfig)
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

	requestConfigs := make([]RequestConfig, len(tables))
	for i, table := range tables {

		payload := table.Payload
		ConfigMapping := AdaptFromDbToConfigMapping(payload.ConfigMapping)

		ConfigValue := AdaptFromDbToInferFlowConfig(payload.ConfigValue)

		serviceDeployableID := ConfigMapping.DeployableID
		serviceDeployableTable, err := m.ServiceDeployableConfigRepo.GetById(int(serviceDeployableID))
		if err != nil {
			return GetAllRequestConfigsResponse{}, errors.New("failed to get service deployable config by id: " + err.Error())
		}

		ConfigMapping.DeployableName = serviceDeployableTable.Name

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
		return GetAllResponse{}, errors.New("failed to get all infer flow configs: " + err.Error())
	}

	responseConfigs := make([]ConfigTable, len(tables))
	for i, table := range tables {
		disocveryTable, err := m.DiscoveryConfigRepo.GetById(int(table.DiscoveryID))
		if err != nil {
			return GetAllResponse{}, errors.New("failed to get discovery config by id: " + err.Error())
		}

		serviceDeployableID := disocveryTable.ServiceDeployableID
		serviceDeployableTable, err := m.ServiceDeployableConfigRepo.GetById(int(serviceDeployableID))
		if err != nil {
			return GetAllResponse{}, errors.New("failed to get service deployable config by id: " + err.Error())
		}

		ConfigValue := AdaptFromDbToInferFlowConfig(table.ConfigValue)

		ringMasterConfig := m.ringMasterClient.GetConfig(serviceDeployableTable.Name, serviceDeployableTable.DeployableWorkFlowId, serviceDeployableTable.DeploymentRunID)

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
	for _, ranker := range request.Rankers {
		if len(ranker.EntityID) == 0 {
			return Response{
				Error: "Entity ID is not set for model: " + ranker.ModelName,
				Data:  Message{Message: emptyResponse},
			}, errors.New("Entity ID is not set for model: " + ranker.ModelName)
		}
		for _, output := range ranker.Outputs {
			if len(output.ModelScores) != len(output.ModelScoresDims) {
				return Response{
					Error: "model scores and model scores dims are not equal for model: " + ranker.ModelName,
					Data:  Message{Message: emptyResponse},
				}, errors.New("model scores and model scores dims are not equal for model: " + ranker.ModelName)
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

		// Check if endpoint already has a port
		hasPort := strings.LastIndex(ep, ":") != -1

		// Only add port if it's missing
		if !hasPort {
			var port string
			if !inferflowPkg.IsMeeshoEnabled {
				port = ":8085"
			} else {
				port = ":8080"
			}
			env := strings.ToLower(strings.TrimSpace(inferflowPkg.AppEnv))
			if env == "stg" || env == "int" {
				port = ":80"
			}
			ep = ep + port
		}
		return ep
	}(request.EndPoint)

	conn, err := grpc.GetConnection(normalizedEndpoint)
	if err != nil {
		response.Error = err.Error()
		return response, errors.New("failed to get connection: " + err.Error())
	}
	defer conn.Close()

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err = grpc.SendGRPCRequest(ctx, conn, inferFlowRetrieveModelScoreMethod, protoRequest, protoResponse, md)
	if err != nil {
		response.Error = err.Error()
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
		fmt.Println("Error getting model proxy config: ", err)
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
			fmt.Println("Error updating model proxy config: ", err)
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

func (m *InferFlow) GetLoggingTTL() (GetLoggingTTLResponse, error) {
	return GetLoggingTTLResponse{
		Data: []int{30, 60, 90},
	}, nil
}
