package handler

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	mainHandler "github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/deployablemetadata"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/serviceconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	workflowHandler "github.com/Meesho/BharatMLStack/horizon/internal/workflow/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
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
	serviceConfigLoader    configs.ServiceConfigLoader // Config-as-code loader
	ringMasterClient       mainHandler.RingmasterClient
	infrastructureHandler  infrastructurehandler.InfrastructureHandler
	workflowHandler        workflowHandler.Handler // Workflow handler for async operations
	workingEnv             string
}

func InitV1ConfigHandler(appConfig configs.Configs) Config {
	var handler Config
	deployableOnce.Do(func() {
		ringMasterClient := mainHandler.GetRingmasterClient()
		infrastructureHandler := infrastructurehandler.InitInfrastructureHandler()
		workflowHandler := workflowHandler.GetWorkflowHandler()
		workingEnv := mainHandler.GetWorkingEnvironment()
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

		// Initialize service config loader (config-as-code)
		// This will be nil if SERVICE_CONFIG_SOURCE is not set or invalid
		var serviceConfigLoader configs.ServiceConfigLoader
		configSource := appConfig.ServiceConfigSource
		if configSource != "" {
			basePath := appConfig.ServiceConfigPath
			if basePath == "" {
				basePath = "./configs" // Default path
			}
			loader, err := configs.NewServiceConfigLoader(
				configSource,
				basePath,
				appConfig.GitHubOwner,
				appConfig.ServiceConfigRepo,
			)
			if err != nil {
				log.Warn().
					Err(err).
					Str("source", configSource).
					Str("basePath", basePath).
					Str("repo", appConfig.ServiceConfigRepo).
					Msg("Failed to create service config loader, will fallback to database")
			} else {
				serviceConfigLoader = loader
				log.Info().
					Str("source", configSource).
					Str("basePath", basePath).
					Str("repo", appConfig.ServiceConfigRepo).
					Msg("Service config loader initialized successfully")
			}
		} else {
			log.Info().
				Msg("SERVICE_CONFIG_SOURCE not set, will use database for service config")
		}

		handler = &Deployable{
			repo:                   repo,
			deployableMetaDataRepo: deployableMetaDataRepo,
			serviceConfigRepo:      serviceConfigRepo,
			serviceConfigLoader:    serviceConfigLoader,
			ringMasterClient:       ringMasterClient,
			infrastructureHandler:  infrastructureHandler,
			workflowHandler:        workflowHandler,
			workingEnv:             workingEnv,
		}
	})

	return handler
}

// SetServiceConfigLoader sets the service config loader on the deployable handler
// This allows updating the loader after initialization if needed
func (d *Deployable) SetServiceConfigLoader(loader configs.ServiceConfigLoader) {
	d.serviceConfigLoader = loader
}

func (d *Deployable) GetMetaData() (map[string][]string, error) {
	deployablesMetaDataReponse, err := d.deployableMetaDataRepo.GetGroupedActiveMetadata()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get deployable metadata")
		return nil, fmt.Errorf("%s: %w", ErrFailedToFetchDeployable, err)
	}

	return deployablesMetaDataReponse, nil
}

func (d *Deployable) CreateDeployable(request *DeployableRequest, workingEnv string) (string, error) {
	switch request.ServiceName {
	case "predator":
		handler := NewHandler(d.repo, d.serviceConfigRepo, d.ringMasterClient)
		// Set service config loader if available
		if d.serviceConfigLoader != nil {
			handler.SetServiceConfigLoader(d.serviceConfigLoader)
		}
		return handler.CreateDeployable(request, workingEnv)
	case "inferflow":
		handler := NewInferflowHandler(d.repo)
		// Inferflow handler doesn't return workflow ID yet, return empty string for now
		err := handler.CreateDeployable(request)
		return "", err
	default:
		return "", fmt.Errorf("unsupported service type: %s", request.ServiceName)
	}
}

// CreateDeployableMultiEnvironment creates deployables for multiple environments in a single request
// Each environment gets its own workflow ID, allowing independent tracking and execution
func (d *Deployable) CreateDeployableMultiEnvironment(request *DeployableRequest) (map[string]string, error) {
	if len(request.Environments) == 0 {
		return nil, fmt.Errorf("environments array is required for multi-environment onboarding")
	}

	log.Info().
		Str("appName", request.AppName).
		Str("serviceName", request.ServiceName).
		Int("environmentCount", len(request.Environments)).
		Msg("CreateDeployableMultiEnvironment: Starting multi-environment onboarding")

	// Log all environments being processed
	for i, envConfig := range request.Environments {
		log.Info().
			Str("appName", request.AppName).
			Int("index", i).
			Str("workingEnv", envConfig.WorkingEnv).
			Str("machineType", envConfig.MachineType).
			Str("serviceType", envConfig.ServiceType).
			Msg("CreateDeployableMultiEnvironment: Environment in request")
	}

	// Atomic validation: Validate all environments before proceeding
	// This ensures strict atomicity - all environments must be whitelisted and have configs
	// If any environment fails validation, the entire request is rejected
	if err := ValidateMultiEnvironmentOnboarding(request.ServiceName, request.Environments, d.serviceConfigLoader); err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("serviceName", request.ServiceName).
			Int("environmentCount", len(request.Environments)).
			Msg("CreateDeployableMultiEnvironment: Multi-environment validation failed - blocking onboarding (atomic behavior)")
		return nil, fmt.Errorf("onboarding blocked: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Str("serviceName", request.ServiceName).
		Int("validatedEnvironments", len(request.Environments)).
		Msg("CreateDeployableMultiEnvironment: All environments validated successfully - proceeding with onboarding")

	// Check ArgoCD configuration for all environments (optional - logs warnings if not found)
	// ArgoCD configuration is optional - system will use defaults if not configured
	for _, envConfig := range request.Environments {
		if err := argocd.ValidateArgoCDConfiguration(envConfig.WorkingEnv); err != nil {
			log.Warn().
				Err(err).
				Str("appName", request.AppName).
				Str("workingEnv", envConfig.WorkingEnv).
				Msg("CreateDeployableMultiEnvironment: ArgoCD configuration not found for environment (optional - will use defaults if available)")
		}
	}

	log.Info().
		Str("appName", request.AppName).
		Int("checkedEnvironments", len(request.Environments)).
		Msg("CreateDeployableMultiEnvironment: ArgoCD configuration check completed (optional)")

	workflowIDs := make(map[string]string)
	var errors []string

	log.Info().
		Str("appName", request.AppName).
		Int("totalEnvironments", len(request.Environments)).
		Msg("CreateDeployableMultiEnvironment: Starting to process environments")

	for i, envConfig := range request.Environments {
		log.Info().
			Str("appName", request.AppName).
			Int("index", i).
			Int("totalEnvironments", len(request.Environments)).
			Str("workingEnv", envConfig.WorkingEnv).
			Str("machineType", envConfig.MachineType).
			Str("serviceType", envConfig.ServiceType).
			Msg("CreateDeployableMultiEnvironment: Processing environment")

		// Create a single-environment request from the environment config
		singleEnvRequest := &DeployableRequest{
			AppName:            request.AppName,
			ServiceName:        request.ServiceName,
			CreatedBy:          request.CreatedBy,
			MachineType:        envConfig.MachineType,
			CPURequest:         envConfig.CPURequest,
			CPURequestUnit:     envConfig.CPURequestUnit,
			CPULimit:           envConfig.CPULimit,
			CPULimitUnit:       envConfig.CPULimitUnit,
			MemoryRequest:      envConfig.MemoryRequest,
			MemoryRequestUnit:  envConfig.MemoryRequestUnit,
			MemoryLimit:        envConfig.MemoryLimit,
			MemoryLimitUnit:    envConfig.MemoryLimitUnit,
			GPURequest:         envConfig.GPURequest,
			GPULimit:           envConfig.GPULimit,
			MinReplica:         envConfig.MinReplica,
			MaxReplica:         envConfig.MaxReplica,
			GCSBucketPath:      envConfig.GCSBucketPath,
			GCSTritonPath:      envConfig.GCSTritonPath,
			TritonImageTag:     envConfig.TritonImageTag,
			ServiceAccount:     envConfig.ServiceAccount,
			NodeSelectorValue:  envConfig.NodeSelectorValue,
			DeploymentStrategy: envConfig.DeploymentStrategy,
			ServiceType:        envConfig.ServiceType,    // Use environment-specific service_type if provided
			PodAnnotations:     envConfig.PodAnnotations, // Use environment-specific podAnnotations if provided
		}
		// If podAnnotations not provided in environment config, use from parent request
		if len(singleEnvRequest.PodAnnotations) == 0 {
			if len(request.PodAnnotations) > 0 {
				singleEnvRequest.PodAnnotations = request.PodAnnotations
			}
		}
		// If service_type not provided in environment config, use from parent request or default
		if singleEnvRequest.ServiceType == "" {
			if request.ServiceType != "" {
				singleEnvRequest.ServiceType = request.ServiceType
			} else {
				singleEnvRequest.ServiceType = "httpstateless" // Default
			}
		}

		log.Info().
			Str("appName", request.AppName).
			Str("workingEnv", envConfig.WorkingEnv).
			Str("finalServiceType", singleEnvRequest.ServiceType).
			Int("podAnnotationsCount", len(singleEnvRequest.PodAnnotations)).
			Msg("CreateDeployableMultiEnvironment: Calling CreateDeployable for environment")

		// Create deployable for this environment (creates separate workflow per environment)
		workflowID, err := d.CreateDeployable(singleEnvRequest, envConfig.WorkingEnv)
		if err != nil {
			log.Error().
				Err(err).
				Str("appName", request.AppName).
				Str("workingEnv", envConfig.WorkingEnv).
				Int("index", i).
				Int("totalEnvironments", len(request.Environments)).
				Msg("CreateDeployableMultiEnvironment: Failed to create deployable for environment")
			errors = append(errors, fmt.Sprintf("%s: %v", envConfig.WorkingEnv, err))
			log.Info().
				Str("appName", request.AppName).
				Str("workingEnv", envConfig.WorkingEnv).
				Int("errorCount", len(errors)).
				Int("successCount", len(workflowIDs)).
				Msg("CreateDeployableMultiEnvironment: Continuing to next environment after error")
			continue
		}

		workflowIDs[envConfig.WorkingEnv] = workflowID
		log.Info().
			Str("appName", request.AppName).
			Str("workingEnv", envConfig.WorkingEnv).
			Str("workflowID", workflowID).
			Int("index", i).
			Int("totalEnvironments", len(request.Environments)).
			Int("successCount", len(workflowIDs)).
			Int("errorCount", len(errors)).
			Msg("CreateDeployableMultiEnvironment: Successfully created deployable for environment")
	}

	log.Info().
		Str("appName", request.AppName).
		Int("totalEnvironments", len(request.Environments)).
		Int("successCount", len(workflowIDs)).
		Int("errorCount", len(errors)).
		Interface("workflowIDs", workflowIDs).
		Strs("errors", errors).
		Msg("CreateDeployableMultiEnvironment: Finished processing all environments")

	if len(errors) > 0 {
		if len(workflowIDs) == 0 {
			// All environments failed
			log.Error().
				Strs("errors", errors).
				Str("appName", request.AppName).
				Msg("CreateDeployableMultiEnvironment: All environments failed")
			return nil, fmt.Errorf("failed to create deployables for all environments: %v", errors)
		}
		// Some environments succeeded, some failed
		log.Warn().
			Strs("errors", errors).
			Int("successCount", len(workflowIDs)).
			Int("failureCount", len(errors)).
			Str("appName", request.AppName).
			Msg("CreateDeployableMultiEnvironment: Partial success - some environments failed")
	}

	log.Info().
		Str("appName", request.AppName).
		Int("totalEnvironments", len(request.Environments)).
		Int("successfulEnvironments", len(workflowIDs)).
		Int("failedEnvironments", len(errors)).
		Interface("workflowIDs", workflowIDs).
		Strs("errors", errors).
		Msg("CreateDeployableMultiEnvironment: Multi-environment onboarding completed")

	// Log each workflow ID for debugging
	for env, workflowID := range workflowIDs {
		log.Info().
			Str("appName", request.AppName).
			Str("environment", env).
			Str("workflowID", workflowID).
			Msg("CreateDeployableMultiEnvironment: Workflow ID for environment")
	}

	return workflowIDs, nil
}

func (d *Deployable) UpdateDeployable(request *DeployableRequest, workingEnv string) error {
	switch request.ServiceName {
	case "predator":
		handler := NewHandler(d.repo, d.serviceConfigRepo, d.ringMasterClient)
		// Set service config loader if available
		if d.serviceConfigLoader != nil {
			handler.SetServiceConfigLoader(d.serviceConfigLoader)
		}
		return handler.UpdateDeployable(request, workingEnv)
	case "inferflow":
		handler := NewInferflowHandler(d.repo)
		// Inferflow handler doesn't use workingEnv yet, but we pass it for consistency
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

func (d *Deployable) RefreshDeployable(appName, serviceType, workingEnv string) (*servicedeployableconfig.ServiceDeployableConfig, error) {
	log.Info().
		Str("appName", appName).
		Str("serviceType", serviceType).
		Str("workingEnv", workingEnv).
		Msg("RefreshDeployable: Starting refresh - received parameters")

	// Check if appName already starts with workingEnv prefix (backward compatibility)
	// Standard case: appName is just the app name (without env prefix)
	// Construct database entry name using standard format: {workingEnv}-{appName}
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("dbEntryName", appName).
		Msg("RefreshDeployable: Using standard format - constructed database entry name")

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("dbEntryName", appName).
		Str("serviceType", serviceType).
		Msg("RefreshDeployable: Looking up deployable in database")

	// Get deployable config using the database entry name
	deployable, err := d.repo.GetByNameAndService(appName, serviceType)
	if err != nil {
		log.Error().
			Err(err).
			Str("dbEntryName", appName).
			Str("serviceType", serviceType).
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("RefreshDeployable: Failed to get deployable by name and service from database")
		return nil, fmt.Errorf("%s: %w", ErrFailedToFetchDeployable, err)
	}

	if deployable == nil {
		log.Error().
			Str("dbEntryName", appName).
			Str("serviceType", serviceType).
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("RefreshDeployable: Deployable not found in database")
		return nil, fmt.Errorf("deployable not found with name: %s (searched as: %s) and service: %s in environment: %s", appName, appName, serviceType, workingEnv)
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("dbEntryName", appName).
		Int("deployableID", deployable.ID).
		Str("deployableName", deployable.Name).
		Msg("RefreshDeployable: Found deployable in database, now fetching resource detail from ArgoCD")

	// Use the actual app name (without env prefix) and workingEnv for ArgoCD lookup
	// ArgoCD functions will construct the application name as {workingEnv}-{actualAppName}
	log.Info().
		Str("workingEnv", workingEnv).
		Str("expectedArgoCDAppName", fmt.Sprintf("%s-%s", workingEnv, appName)).
		Msg("RefreshDeployable: Calling GetResourceDetail to connect to ArgoCD")

	result, err := d.infrastructureHandler.GetResourceDetail(appName, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("workingEnv", workingEnv).
			Str("expectedArgoCDAppName", fmt.Sprintf("%s-%s", workingEnv, appName)).
			Msg("RefreshDeployable: Failed to get resource detail from ArgoCD - ArgoCD connection or lookup failed")
		return nil, fmt.Errorf("failed to get resource detail from ArgoCD: %w", err)
	}

	log.Info().
		Str("workingEnv", workingEnv).
		Int("nodeCount", len(result.Nodes)).
		Msg("RefreshDeployable: Successfully connected to ArgoCD and retrieved resource detail")

	// Check if there are healthy pods
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
	infraConfig := d.infrastructureHandler.GetConfig(appName, d.workingEnv)
	// Convert to mainHandler.Config for compatibility
	return mainHandler.Config{
		MinReplica:    infraConfig.MinReplica,
		MaxReplica:    infraConfig.MaxReplica,
		RunningStatus: infraConfig.RunningStatus,
	}
}

func (d *Deployable) TuneThresholds(request *TuneThresholdsRequest, workingEnv string) (string, error) {
	// Construct database entry name using standard format: {env}-{appName}
	// This matches the naming pattern used in CreateDeployable and UpdateDeployable
	if workingEnv == "" {
		return "", fmt.Errorf("workingEnv is required for TuneThresholds")
	}

	log.Info().
		Str("appName", request.AppName).
		Str("dbEntryName", request.AppName).
		Str("workingEnv", workingEnv).
		Msg("TuneThresholds: Looking up deployable with environment-prefixed name")

	deployable, err := d.repo.GetByNameAndService(request.AppName, request.ServiceName)
	if err != nil {
		log.Error().Err(err).Str("dbEntryName", request.AppName).Msg("Failed to get deployable by name and service")
		return "", fmt.Errorf("failed to get deployable: %w", err)
	}

	if deployable == nil {
		return "", fmt.Errorf("deployable config not found for app: %s (searched as: %s) in environment: %s", request.AppName, request.AppName, workingEnv)
	}

	// Use deployable.CreatedBy as email for commit author, fallback to empty string if not available
	commitEmail := deployable.CreatedBy
	if commitEmail == "" {
		commitEmail = "horizon-system"
	}

	// Prepare workflow payload for threshold update
	workflowPayload := map[string]interface{}{
		"appName":      request.AppName,
		"machine_type": request.MachineType,
		"created_by":   commitEmail,
	}

	// Add thresholds to payload if provided
	if request.CPUThreshold != "" {
		workflowPayload["cpu_threshold"] = request.CPUThreshold
	}
	if request.GPUThreshold != "" {
		workflowPayload["gpu_threshold"] = request.GPUThreshold
	}

	// Validate that at least one threshold is provided
	if request.CPUThreshold == "" && request.GPUThreshold == "" {
		return "", fmt.Errorf("at least one of cpu_threshold or gpu_threshold must be provided")
	}

	// Validate machine type for GPU threshold
	// Check if machine type is GPU (case-insensitive for consistency with other parts of the codebase)
	machineTypeUpper := strings.ToUpper(strings.TrimSpace(request.MachineType))
	if request.GPUThreshold != "" && machineTypeUpper != "GPU" {
		return "", fmt.Errorf("gpu_threshold can only be updated for GPU machine type, got: %s", request.MachineType)
	}

	log.Info().
		Str("appName", request.AppName).
		Str("workingEnv", workingEnv).
		Str("cpuThreshold", request.CPUThreshold).
		Str("gpuThreshold", request.GPUThreshold).
		Str("machineType", request.MachineType).
		Msg("TuneThresholds: Starting threshold update workflow")

	// Start threshold update workflow (non-blocking, returns workflow ID)
	workflowID, err := d.workflowHandler.StartThresholdUpdateWorkflow(workflowPayload, commitEmail, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("workingEnv", workingEnv).
			Msg("TuneThresholds: Failed to start threshold update workflow")
		return "", fmt.Errorf("failed to start threshold update workflow: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", request.AppName).
		Str("workingEnv", workingEnv).
		Msg("TuneThresholds: Threshold update workflow started successfully")

	return workflowID, nil
}
