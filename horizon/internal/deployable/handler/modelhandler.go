package handler

import (
	"encoding/json"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
)

type InferflowHandler struct {
	repo servicedeployableconfig.ServiceDeployableRepository
}

func NewInferflowHandler(
	repo servicedeployableconfig.ServiceDeployableRepository,
) *InferflowHandler {
	return &InferflowHandler{
		repo: repo,
	}
}

func (h *InferflowHandler) CreateDeployable(request *DeployableRequest) error {
	configJSON, err := json.Marshal(map[string]interface{}{
		"cpuRequest":        request.CPURequest,
		"cpuRequestUnit":    request.CPURequestUnit,
		"cpuLimit":          request.CPULimit,
		"cpuLimitUnit":      request.CPULimitUnit,
		"memoryRequest":     request.MemoryRequest,
		"memoryRequestUnit": request.MemoryRequestUnit,
		"memoryLimit":       request.MemoryLimit,
		"memoryLimitUnit":   request.MemoryLimitUnit,
		"gpu_request":       request.GPURequest,
		"gpu_limit":         request.GPULimit,
		"min_replica":       request.MinReplica,
		"max_replica":       request.MaxReplica,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	deployableConfig := &servicedeployableconfig.ServiceDeployableConfig{
		Name:                    request.AppName,
		Service:                 request.ServiceName,
		Host:                    request.AppName + "." + hostUrlSuffix,
		Active:                  true,
		CreatedBy:               request.CreatedBy,
		Config:                  configJSON,
		DeployableRunningStatus: false,
		DeployableHealth:        "DEPLOYMENT_REASON_ARGO_APP_HEALTHY",
		DeployableWorkFlowId:    "",
		MonitoringUrl: fmt.Sprintf("%s/d/dQ1Gux-Vk/inferflow?orgId=1&refresh=1m&var-service=%s",
			grafanaBaseUrl, request.AppName),
		WorkFlowStatus: "WORKFLOW_COMPLETED",
	}

	if err := h.repo.Create(deployableConfig); err != nil {
		return fmt.Errorf("failed to create deployable config: %w", err)
	}
	return nil
}

func (h *InferflowHandler) UpdateDeployable(request *DeployableRequest) error {
	return fmt.Errorf("UpdateDeployable is not implemented for InferflowHandler, use UpdateDeployableDBOnly instead")
}
