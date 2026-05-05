package handler

import (
	"errors"
	"fmt"
	"strings"

	discovery_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	inferflow_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/config"
	inferflow_request "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/request"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

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
		existingConfig, err := m.InferFlowConfigRepo.GetByID(tempRequest.ConfigID)
		if err != nil {
			return Response{}, fmt.Errorf("failed to check existing config for promote: %w", err)
		}
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
		Data:  Message{Message: "Inferflow Config reviewed successfully."},
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
	if configExistedBeforeTx {
		if err := m.rollbackEditRequest(tx, currentRequest, discoveryID); err != nil {
			return err
		}
	} else {
		if err := m.rollbackCreatedConfigs(tx, currentRequest.ConfigID, discoveryID); err != nil {
			return err
		}
	}
	return nil
}

func (m *InferFlow) rollbackEditRequest(tx *gorm.DB, currentRequest *inferflow_request.Table, discoveryID int) error {
	approvedRequests, err := m.InferFlowRequestRepo.GetApprovedRequestsByConfigID(currentRequest.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to retrieve approved requests: %w", err)
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
