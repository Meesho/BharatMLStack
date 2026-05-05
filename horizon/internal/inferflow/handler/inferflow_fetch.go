package handler

import (
	"fmt"
	"sync"

	inferflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/inferflow"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	discovery_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	service_deployable_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
)

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

	discoveryConfigs, err := m.DiscoveryConfigRepo.GetByDiscoveryIDs(discoveryIDs)
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
