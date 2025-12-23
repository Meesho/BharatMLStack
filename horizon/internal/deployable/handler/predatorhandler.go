package handler

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/serviceconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/rs/zerolog/log"
)

var (
	initOnce            sync.Once
	defaultCPUThreshold string
	defaultGPUThreshold string
	hostUrlSuffix       string
	grafanaBaseUrl      string
)

func Init(config configs.Configs) {
	initOnce.Do(func() {
		defaultCPUThreshold = config.DefaultCpuThreshold
		defaultGPUThreshold = config.DefaultGpuThreshold
		hostUrlSuffix = config.HostUrlSuffix
		grafanaBaseUrl = config.GrafanaBaseUrl
	})
}

type Handler struct {
	repo              servicedeployableconfig.ServiceDeployableRepository
	serviceConfigRepo serviceconfig.ServiceConfigRepository
	ringMasterClient  externalcall.RingmasterClient
}

func NewHandler(
	repo servicedeployableconfig.ServiceDeployableRepository,
	serviceConfigRepo serviceconfig.ServiceConfigRepository,
	ringMasterClient externalcall.RingmasterClient,
) *Handler {
	return &Handler{
		repo:              repo,
		serviceConfigRepo: serviceConfigRepo,
		ringMasterClient:  ringMasterClient,
	}
}

func (h *Handler) CreateDeployable(request *DeployableRequest) error {
	// 1. Get service config
	serviceConfig, err := h.serviceConfigRepo.GetByName(request.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get service config: %w", err)
	}

	// 2. Create initial DB entry
	configJSON, err := json.Marshal(map[string]interface{}{
		"machine_type":       request.MachineType,
		"cpuRequest":         request.CPURequest,
		"cpuRequestUnit":     request.CPURequestUnit,
		"cpuLimit":           request.CPULimit,
		"cpuLimitUnit":       request.CPULimitUnit,
		"memoryRequest":      request.MemoryRequest,
		"memoryRequestUnit":  request.MemoryRequestUnit,
		"memoryLimit":        request.MemoryLimit,
		"memoryLimitUnit":    request.MemoryLimitUnit,
		"gpu_request":        request.GPURequest,
		"gpu_limit":          request.GPULimit,
		"min_replica":        request.MinReplica,
		"max_replica":        request.MaxReplica,
		"deploymentStrategy": request.DeploymentStrategy,
		"cpu_threshold":      defaultCPUThreshold,
		"gpu_threshold":      defaultGPUThreshold,
		"gcs_bucket_path":    request.GCSBucketPath,
		"gcs_triton_path":    request.GCSTritonPath,
		"triton_image_tag":   request.TritonImageTag,
		"serviceAccount":     request.ServiceAccount,
		"nodeSelectorValue":  request.NodeSelectorValue,
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
		DeployableHealth:        "DEPLOYMENT_REASON_ARGO_APP_HEALTH_DEGRADED",
		DeployableWorkFlowId:    "",
		MonitoringUrl: fmt.Sprintf("%s/d/a2605923-52c4-4834-bdae-97570966b765/predator?orgId=1&var-service=%s&var-query0=",
			grafanaBaseUrl, request.AppName),
		WorkFlowStatus: "WORKFLOW_NOT_STARTED",
	}

	if err := h.repo.Create(deployableConfig); err != nil {
		return fmt.Errorf("failed to create deployable config: %w", err)
	}

	// 3. Prepare ringmaster payload
	ringmasterPayload := map[string]interface{}{
		"appName":           request.AppName,
		"primaryOwner":      serviceConfig.PrimaryOwner,
		"repoName":          serviceConfig.RepoName,
		"branchName":        serviceConfig.BranchName,
		"secondaryOwner":    serviceConfig.SecondaryOwner,
		"healthCheck":       serviceConfig.HealthCheck,
		"host":              request.AppName + "." + hostUrlSuffix,
		"appPort":           serviceConfig.AppPort,
		"team":              serviceConfig.Team,
		"bu":                serviceConfig.BU,
		"priorityV2":        serviceConfig.PriorityV2,
		"module":            serviceConfig.Module,
		"appType":           serviceConfig.AppType,
		"ingress_class":     serviceConfig.IngressClass,
		"machine_type":      request.MachineType,
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
		"as_min":            request.MinReplica,
		"as_max":            request.MaxReplica,
		"gcs_bucket_path":   request.GCSBucketPath,
		"buildNo":           serviceConfig.BuildNo,
		"triton_image_tag":  request.TritonImageTag,
		"created_by":        request.CreatedBy,
	}

	// Add service type config
	var serviceTypeConfig map[string]string
	if err := json.Unmarshal(serviceConfig.Config, &serviceTypeConfig); err != nil {
		return fmt.Errorf("failed to unmarshal service type config: %w", err)
	}
	for k, v := range serviceTypeConfig {
		ringmasterPayload[k] = v
	}

	// Add remaining fields
	ringmasterPayload["serviceAccount"] = request.ServiceAccount
	ringmasterPayload["nodeSelectorValue"] = request.NodeSelectorValue
	ringmasterPayload["deploymentStrategy"] = request.DeploymentStrategy
	ringmasterPayload["gcs_triton_path"] = request.GCSTritonPath

	// 4. Make API calls in sequence
	go func() {

		var rawBody []byte
		sleepTime := 30
		for attempt := 1; attempt <= 3; attempt++ {
			rawBody, err = h.ringMasterClient.CreateDeployable(ringmasterPayload)
			time.Sleep(time.Duration(sleepTime) * time.Second)
			sleepTime += 30
		}

		if err != nil {
			log.Error().Err(err).Msg("failed to create deployable in ringmaster")
			return
		}

		var onboardResp OnboardResponse
		if err := json.Unmarshal(rawBody, &onboardResp); err != nil {
			log.Error().Err(err).Msg("failed to parse ringmaster response")
		}

		// Update deployable with workflow IDs
		deployableConfig.DeploymentRunID = onboardResp.DeploymentRunID
		deployableConfig.DeployableWorkFlowId = onboardResp.DeploymentWorkflowID
		if err := h.repo.Update(deployableConfig); err != nil {
			log.Error().Err(err).Msg("failed to update deployable with workflow IDs")
			return
		}
	}()

	return nil
}

func (h *Handler) UpdateDeployable(request *DeployableRequest) error {
	// 1. Get service config
	serviceConfig, err := h.serviceConfigRepo.GetByName(request.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get service config: %w", err)
	}

	// 2. Get existing deployable configs by service
	existingConfig, err := h.repo.GetByNameAndService(request.AppName, request.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get existing deployable configs: %w", err)
	}

	if existingConfig == nil {
		return fmt.Errorf("deployable config not found for app: %s", request.AppName)
	}

	var config DeployableConfigPayload
	if err := json.Unmarshal(existingConfig.Config, &config); err != nil {
		return fmt.Errorf("failed to unmarshal existing config: %w", err)
	}

	// 3. Update DB entry
	configJSON, err := json.Marshal(map[string]interface{}{
		"machine_type":       request.MachineType,
		"cpuRequest":         request.CPURequest,
		"cpuRequestUnit":     request.CPURequestUnit,
		"cpuLimit":           request.CPULimit,
		"cpuLimitUnit":       request.CPULimitUnit,
		"memoryRequest":      request.MemoryRequest,
		"memoryRequestUnit":  request.MemoryRequestUnit,
		"memoryLimit":        request.MemoryLimit,
		"memoryLimitUnit":    request.MemoryLimitUnit,
		"gpu_request":        request.GPURequest,
		"gpu_limit":          request.GPULimit,
		"min_replica":        request.MinReplica,
		"max_replica":        request.MaxReplica,
		"gcs_bucket_path":    request.GCSBucketPath,
		"triton_image_tag":   request.TritonImageTag,
		"serviceAccount":     request.ServiceAccount,
		"nodeSelectorValue":  request.NodeSelectorValue,
		"cpu_threshold":      config.CPUThreshold,
		"deploymentStrategy": request.DeploymentStrategy,
		"gpu_threshold":      config.GPUThreshold,
		"gcs_triton_path":    request.GCSTritonPath,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	existingConfig.Config = configJSON
	existingConfig.CreatedBy = request.CreatedBy
	existingConfig.WorkFlowStatus = "WORKFLOW_NOT_STARTED"

	if err := h.repo.Update(existingConfig); err != nil {
		return fmt.Errorf("failed to update deployable config: %w", err)
	}

	// 4. Prepare ringmaster payload
	ringmasterPayload := map[string]interface{}{
		"appName":           request.AppName,
		"primaryOwner":      serviceConfig.PrimaryOwner,
		"repoName":          serviceConfig.RepoName,
		"branchName":        serviceConfig.BranchName,
		"secondaryOwner":    serviceConfig.SecondaryOwner,
		"healthCheck":       serviceConfig.HealthCheck,
		"appPort":           serviceConfig.AppPort,
		"team":              serviceConfig.Team,
		"bu":                serviceConfig.BU,
		"priorityV2":        serviceConfig.PriorityV2,
		"module":            serviceConfig.Module,
		"appType":           serviceConfig.AppType,
		"ingress_class":     serviceConfig.IngressClass,
		"machine_type":      request.MachineType,
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
		"as_min":            request.MinReplica,
		"as_max":            request.MaxReplica,
		"gcs_bucket_path":   request.GCSBucketPath,
		"host":              request.AppName + "." + hostUrlSuffix,
		"buildNo":           serviceConfig.BuildNo,
		"triton_image_tag":  request.TritonImageTag,
		"created_by":        request.CreatedBy,
	}

	// Add service type config
	var serviceTypeConfig map[string]string
	if err := json.Unmarshal(serviceConfig.Config, &serviceTypeConfig); err != nil {
		return fmt.Errorf("failed to unmarshal service type config: %w", err)
	}
	for k, v := range serviceTypeConfig {
		ringmasterPayload[k] = v
	}

	// Add remaining fields
	ringmasterPayload["serviceAccount"] = request.ServiceAccount
	ringmasterPayload["nodeSelectorValue"] = request.NodeSelectorValue
	ringmasterPayload["deploymentStrategy"] = request.DeploymentStrategy
	ringmasterPayload["gcs_triton_path"] = request.GCSTritonPath

	// 5. Make API calls in sequence
	go func() {
		// Update deployable using same create API as per requirement
		rawBody, err := h.ringMasterClient.CreateDeployable(ringmasterPayload)
		if err != nil {
			log.Error().Err(err).Msg("failed to create deployable in ringmaster")
			return
		}

		var onboardResp OnboardResponse
		if err := json.Unmarshal(rawBody, &onboardResp); err != nil {
			log.Error().Err(err).Msg("failed to parse ringmaster response")
			return
		}

		// Update deployable with workflow IDs
		existingConfig.DeploymentRunID = onboardResp.DeploymentRunID
		existingConfig.DeployableWorkFlowId = onboardResp.DeploymentWorkflowID
		if err := h.repo.Update(existingConfig); err != nil {
			log.Error().Err(err).Msg("failed to update deployable with workflow IDs")
			return
		}
	}()

	return nil
}
