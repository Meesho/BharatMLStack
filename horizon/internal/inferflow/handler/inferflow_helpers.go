package handler

import (
	"fmt"
)

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
