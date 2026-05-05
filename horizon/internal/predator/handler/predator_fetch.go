package handler

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
)

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
