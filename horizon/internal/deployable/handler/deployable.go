package handler

import (
	"encoding/json"
	"fmt"
	"sync"

	mainHandler "github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/deployablemetadata"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/serviceconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

const (
	ErrFailedToFetchDeployable = "failed to fetch service deployable"
)

var (
	deployableOnce sync.Once
)

type Deployable struct {
	repo                   servicedeployableconfig.ServiceDeployableRepository
	deployableMetaDataRepo deployablemetadata.DeployableMetadataRepository
	serviceConfigRepo      serviceconfig.ServiceConfigRepository
	ringMasterClient       mainHandler.RingmasterClient
}

func InitV1ConfigHandler() Config {
	var handler Config
	deployableOnce.Do(func() {
		ringMasterClient := mainHandler.GetRingmasterClient()
		connection, err := infra.SQL.GetConnection()
		if err != nil {
			log.Panic().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn, ok := connection.(*infra.SQLConnection)
		if !ok {
			log.Panic().Msg("Failed to cast connection to SQLConnection")
		}
		repo, err := servicedeployableconfig.NewRepository(sqlConn)
		if err != nil {
			log.Panic().Err(err).Msg("Failed to create service deployable repository")
		}
		deployableMetaDataRepo, err := deployablemetadata.NewRepository(sqlConn)
		if err != nil {
			log.Panic().Err(err).Msg("Failed to create deployable metadata repository")
		}
		serviceConfigRepo, err := serviceconfig.NewRepository(sqlConn)
		if err != nil {
			log.Panic().Err(err).Msg("Failed to create service config repository")
		}

		handler = &Deployable{
			repo:                   repo,
			deployableMetaDataRepo: deployableMetaDataRepo,
			serviceConfigRepo:      serviceConfigRepo,
			ringMasterClient:       ringMasterClient,
		}
	})

	return handler
}

func (d *Deployable) GetMetaData() (map[string][]string, error) {
	deployablesMetaDataReponse, err := d.deployableMetaDataRepo.GetGroupedActiveMetadata()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get deployable metadata")
		return nil, fmt.Errorf("%s: %w", ErrFailedToFetchDeployable, err)
	}

	return deployablesMetaDataReponse, nil
}

func (d *Deployable) CreateDeployable(request *DeployableRequest) error {
	switch request.ServiceName {
	case "predator":
		handler := NewHandler(d.repo, d.serviceConfigRepo, d.ringMasterClient)
		return handler.CreateDeployable(request)
	case "inferflow":
		handler := NewInferflowHandler(d.repo)
		return handler.CreateDeployable(request)
	default:
		return fmt.Errorf("unsupported service type: %s", request.ServiceName)
	}
}

func (d *Deployable) UpdateDeployable(request *DeployableRequest) error {
	switch request.ServiceName {
	case "predator":
		handler := NewHandler(d.repo, d.serviceConfigRepo, d.ringMasterClient)
		return handler.UpdateDeployable(request)
	case "inferflow":
		handler := NewInferflowHandler(d.repo)
		return handler.UpdateDeployable(request)
	default:
		return fmt.Errorf("unsupported service type: %s", request.ServiceName)
	}
}

func (d *Deployable) GetDeployablesByService(serviceName string) ([]servicedeployableconfig.ServiceDeployableConfig, error) {
	deployables, err := d.repo.GetByService(serviceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get deployables by service")
		return nil, fmt.Errorf("%s: %w", ErrFailedToFetchDeployable, err)
	}
	return deployables, nil
}

func (d *Deployable) RefreshDeployable(appName, serviceType string) (*servicedeployableconfig.ServiceDeployableConfig, error) {
	// Get deployable config
	deployable, err := d.repo.GetByNameAndService(appName, serviceType)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get deployable by name and service")
		return nil, fmt.Errorf("%s: %w", ErrFailedToFetchDeployable, err)
	}

	if deployable == nil {
		return nil, fmt.Errorf("deployable not found with name: %s and service: %s", appName, serviceType)
	}

	result, err := d.ringMasterClient.GetResourceDetail(appName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get workflow result from Ring Master")
		return nil, fmt.Errorf("failed to get workflow result: %w", err)
	}

	// Check if there are activities
	if len(result.Nodes) > 0 {
		healthyPodCount := 0
		for _, node := range result.Nodes {
			if node.Kind == "Pod" && node.Health.Status == "Healthy" {
				for _, info := range node.Info {
					if info.Value == "Running" {
						healthyPodCount += 1
					}
				}
			}
		}

		if healthyPodCount > 0 {
			deployable.DeployableHealth = "DEPLOYMENT_REASON_ARGO_APP_HEALTHY"
			deployable.WorkFlowStatus = "WORKFLOW_COMPLETED"
			deployable.DeployableRunningStatus = true
		}
	}

	// Update in database
	if err := d.repo.Update(deployable); err != nil {
		log.Error().Err(err).Msg("Failed to update deployable status")
		return nil, fmt.Errorf("failed to update deployable status: %w", err)
	}

	return deployable, nil
}

func (d *Deployable) GetRingMasterConfig(appName, workflowID, runID string) mainHandler.Config {
	return d.ringMasterClient.GetConfig(appName, workflowID, runID)
}

func (d *Deployable) TuneThresholds(request *TuneThresholdsRequest) error {
	// Make a copy of request for goroutine safety
	req := *request

	go func() {
		deployable, err := d.repo.GetByNameAndService(req.AppName, req.ServiceName)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get deployable by name and service")
			return
		}

		if deployable == nil {
			log.Error().Msgf("Deployable config not found for app: %s", req.AppName)
			return
		}

		var config DeployableConfigPayload
		if err := json.Unmarshal(deployable.Config, &config); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal existing config")
			return
		}

		if req.MachineType == "GPU" {
			if req.CPUThreshold != "" {
				if err := d.ringMasterClient.UpdateCPUThreshold(req.AppName, req.CPUThreshold); err != nil {
					log.Error().Err(err).Msg("Failed to update CPU threshold")
					return
				}
				config.CPUThreshold = req.CPUThreshold
			}

			if req.GPUThreshold != "" {
				if err := d.ringMasterClient.UpdateGPUThreshold(req.AppName, req.GPUThreshold); err != nil {
					log.Error().Err(err).Msg("Failed to update GPU threshold")
					return
				}
				config.GPUThreshold = req.GPUThreshold
			}
		} else {
			if req.CPUThreshold != "" {
				if err := d.ringMasterClient.UpdateCPUThreshold(req.AppName, req.CPUThreshold); err != nil {
					log.Error().Err(err).Msg("Failed to update CPU threshold")
					return
				}
				config.CPUThreshold = req.CPUThreshold
			}
		}

		configJSON, err := json.Marshal(map[string]interface{}{
			"machine_type":      config.MachineType,
			"cpuRequest":        config.CPURequest,
			"cpuRequestUnit":    config.CPURequestUnit,
			"cpuLimit":          config.CPULimit,
			"cpuLimitUnit":      config.CPULimitUnit,
			"memoryRequest":     config.MemoryRequest,
			"memoryRequestUnit": config.MemoryRequestUnit,
			"memoryLimit":       config.MemoryLimit,
			"memoryLimitUnit":   config.MemoryLimitUnit,
			"gpu_request":       config.GPURequest,
			"gpu_limit":         config.GPULimit,
			"min_replica":       config.MinReplica,
			"max_replica":       config.MaxReplica,
			"gcs_bucket_path":   config.GCSBucketPath,
			"triton_image_tag":  config.TritonImageTag,
			"serviceAccount":    config.ServiceAccount,
			"nodeSelectorValue": config.NodeSelectorValue,
			"cpu_threshold":     config.CPUThreshold,
			"gpu_threshold":     config.GPUThreshold,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal updated config")
			return
		}

		deployable.Config = configJSON

		if err := d.repo.Update(deployable); err != nil {
			log.Error().Err(err).Msg("Failed to update deployable config")
			return
		}

		log.Info().Msg("Successfully updated thresholds and config")
	}()

	// Return immediately, while processing happens in background
	return nil
}
