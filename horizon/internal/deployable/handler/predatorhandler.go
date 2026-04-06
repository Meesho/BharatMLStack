package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/workflow/activities"
	workflowHandler "github.com/Meesho/BharatMLStack/horizon/internal/workflow/handler"
	"github.com/spf13/viper"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/serviceconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
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
	repo                servicedeployableconfig.ServiceDeployableRepository
	serviceConfigRepo   serviceconfig.ServiceConfigRepository // Deprecated: kept for backward compatibility
	serviceConfigLoader configs.ServiceConfigLoader           // New: config-as-code loader
	workflowHandler     workflowHandler.Handler               // New: workflow handler for async onboarding
}

func NewHandler(
	repo servicedeployableconfig.ServiceDeployableRepository,
	serviceConfigRepo serviceconfig.ServiceConfigRepository,
) *Handler {
	return &Handler{
		repo:              repo,
		serviceConfigRepo: serviceConfigRepo,
		workflowHandler:   workflowHandler.GetWorkflowHandler(), // Initialize workflow handler
	}
}

// SetServiceConfigLoader sets the service config loader (config-as-code)
// This should be called during initialization if config-as-code is enabled
func (h *Handler) SetServiceConfigLoader(loader configs.ServiceConfigLoader) {
	h.serviceConfigLoader = loader
}

// loadServiceConfig loads service configuration from config-as-code (preferred) or database (fallback)
func (h *Handler) loadServiceConfig(serviceName, workingEnv string) (*serviceConfigData, error) {
	// Validate environment whitelist first (before attempting to load config)
	if err := ValidateEnvironmentWhitelist(workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("workingEnv", workingEnv).
			Msg("Environment whitelist validation failed - blocking onboarding")
		return nil, fmt.Errorf("onboarding blocked: %w", err)
	}

	// Atomic onboarding: Each environment MUST have its own config.yaml
	// Fail immediately if config-as-code loader is available but config doesn't exist
	// This enforces atomic onboarding logic - no fallback to shared configs
	if h.serviceConfigLoader != nil {
		log.Info().
			Str("serviceName", serviceName).
			Str("workingEnv", workingEnv).
			Msg("Loading service config from config-as-code (atomic onboarding - each environment must have its own config.yaml)")

		// Validate config exists (this will fail if config.yaml is missing)
		if err := ValidateEnvironmentConfig(serviceName, workingEnv, h.serviceConfigLoader); err != nil {
			log.Error().
				Err(err).
				Str("serviceName", serviceName).
				Str("workingEnv", workingEnv).
				Msg("Environment config validation failed - blocking onboarding (atomic behavior)")
			return nil, fmt.Errorf("onboarding blocked: %w", err)
		}

		cacConfig, err := h.serviceConfigLoader.LoadServiceConfig(serviceName, workingEnv)
		if err != nil {
			log.Error().
				Err(err).
				Str("serviceName", serviceName).
				Str("workingEnv", workingEnv).
				Msg("Failed to load service config from config-as-code - atomic onboarding requires each environment to have its own config.yaml")
			return nil, fmt.Errorf("atomic onboarding violation: config.yaml not found for environment %s (expected: horizon/configs/services/%s/%s/config.yaml): %w", workingEnv, serviceName, workingEnv, err)
		}

		log.Info().
			Str("serviceName", serviceName).
			Str("workingEnv", workingEnv).
			Msg("Service config loaded successfully from config-as-code")
		return convertCACToServiceConfig(cacConfig, workingEnv), nil
	}

	// Fallback to database only if config-as-code loader is not available (backward compatibility)
	if h.serviceConfigRepo != nil {
		log.Warn().
			Str("serviceName", serviceName).
			Str("workingEnv", workingEnv).
			Msg("Config-as-code loader not available, falling back to database (not recommended for multi-environment onboarding)")

		dbConfig, err := h.serviceConfigRepo.GetByName(serviceName)
		if err != nil {
			log.Error().
				Err(err).
				Str("serviceName", serviceName).
				Msg("Failed to load service config from database")
			return nil, fmt.Errorf("failed to get service config: %w", err)
		}

		log.Info().
			Str("serviceName", serviceName).
			Msg("Service config loaded successfully from database")
		return convertDBToServiceConfig(dbConfig, workingEnv), nil
	}

	return nil, fmt.Errorf("no service config loader or repository available")
}

// serviceConfigData represents the unified service config structure
type serviceConfigData struct {
	RepoName                  string
	BranchName                string
	HealthCheck               string
	AppPort                   int
	IngressClass              string
	Domain                    string // Domain name for host generation (e.g., "meesho.int", "example.com")
	TritonRepository          string // Triton inference server image repository path without tag (from config.yaml, required)
	TelegrafImage             string // Complete Telegraf image path with tag (from config.yaml, optional)
	InitContainerImage        string // Complete init container image path with tag for GCS operations (from config.yaml, required if gcs_triton_path is enabled)
	AppType                   string
	PrimaryOwner              string
	SecondaryOwner            string
	Team                      string
	BU                        string
	PriorityV2                string
	ServiceType               map[string]bool
	LivenessFailureThreshold  string
	LivenessPeriodSeconds     string
	LivenessSuccessThreshold  string
	LivenessTimeoutSeconds    string
	ReadinessFailureThreshold string
	ReadinessPeriodSeconds    string
	ReadinessSuccessThreshold string
	ReadinessTimeoutSeconds   string
	NodeSelector              string
	// Autoscaling configuration
	ASEnabled          string
	ASPoll             string
	ASDownPeriod       string
	ASUpPeriod         string
	ASUpStableWindow   string
	ASDownStableWindow string
	ASTriggerType      string
	ASTriggerMetric    string
	ASTriggerValue     string
	ASDownPodCount     string
	ASUpPodCount       string
	ASUpPodPercentage  string
	CPUThreshold       string
	// Deployment configuration
	MaxSurge                      string
	TerminationGracePeriodSeconds string
	ContourResponseTimeout        string
	PodDistributionSkew           string
	EnableWebsocket               string
	AddHeadless                   string
	CreateContourGateway          string
	PDBMinAvailable               string
	PDBMaxUnavailable             string
	PodAnnotations                map[string]string
}

// convertCACToServiceConfig converts config-as-code ServiceConfig to serviceConfigData
func convertCACToServiceConfig(cacConfig *configs.ServiceConfig, workingEnv string) *serviceConfigData {
	// Read labels from Labels section first (preferred), then fall back to legacy fields
	var bu, team, primaryOwner, secondaryOwner, priorityV2 string
	if cacConfig.Labels != nil {
		bu = cacConfig.Labels.BU
		team = cacConfig.Labels.Team
		primaryOwner = cacConfig.Labels.PrimaryOwner
		secondaryOwner = cacConfig.Labels.SecondaryOwner
		priorityV2 = cacConfig.Labels.PriorityV2
	}
	// Fall back to legacy fields if Labels section is not present or fields are empty
	if bu == "" {
		bu = cacConfig.BU
	}
	if team == "" {
		team = cacConfig.Team
	}
	if primaryOwner == "" {
		primaryOwner = cacConfig.PrimaryOwner
	}
	if secondaryOwner == "" {
		secondaryOwner = cacConfig.SecondaryOwner
	}
	if priorityV2 == "" {
		priorityV2 = cacConfig.PriorityV2
	}

	result := &serviceConfigData{
		RepoName:                  cacConfig.RepoName,
		BranchName:                configs.GetBranchName(workingEnv),
		HealthCheck:               cacConfig.HealthCheck,
		AppPort:                   cacConfig.AppPort,
		IngressClass:              cacConfig.IngressClass,
		Domain:                    cacConfig.Domain,
		TritonRepository:          cacConfig.TritonRepository,   // Triton image repository path without tag from config.yaml (required)
		InitContainerImage:        cacConfig.InitContainerImage, // Init container image path from config.yaml (required if gcs_triton_path is enabled)
		AppType:                   cacConfig.AppType,
		PrimaryOwner:              primaryOwner,
		SecondaryOwner:            secondaryOwner,
		Team:                      team,
		BU:                        bu,
		PriorityV2:                priorityV2,
		ServiceType:               cacConfig.ServiceType,
		LivenessFailureThreshold:  cacConfig.LivenessFailureThreshold,
		LivenessPeriodSeconds:     cacConfig.LivenessPeriodSeconds,
		LivenessSuccessThreshold:  cacConfig.LivenessSuccessThreshold,
		LivenessTimeoutSeconds:    cacConfig.LivenessTimeoutSeconds,
		ReadinessFailureThreshold: cacConfig.ReadinessFailureThreshold,
		ReadinessPeriodSeconds:    cacConfig.ReadinessPeriodSeconds,
		ReadinessSuccessThreshold: cacConfig.ReadinessSuccessThreshold,
		ReadinessTimeoutSeconds:   cacConfig.ReadinessTimeoutSeconds,
		NodeSelector:              cacConfig.NodeSelector,
		// Autoscaling configuration
		ASEnabled:          cacConfig.ASEnabled,
		ASPoll:             cacConfig.ASPoll,
		ASDownPeriod:       cacConfig.ASDownPeriod,
		ASUpPeriod:         cacConfig.ASUpPeriod,
		ASUpStableWindow:   cacConfig.ASUpStableWindow,
		ASDownStableWindow: cacConfig.ASDownStableWindow,
		ASTriggerType:      cacConfig.ASTriggerType,
		ASTriggerMetric:    cacConfig.ASTriggerMetric,
		ASTriggerValue:     cacConfig.ASTriggerValue,
		ASDownPodCount:     cacConfig.ASDownPodCount,
		ASUpPodCount:       cacConfig.ASUpPodCount,
		ASUpPodPercentage:  cacConfig.ASUpPodPercentage,
		CPUThreshold:       cacConfig.CPUThreshold,
		// Deployment configuration
		MaxSurge:                      cacConfig.MaxSurge,
		TerminationGracePeriodSeconds: cacConfig.TerminationGracePeriodSeconds,
		ContourResponseTimeout:        cacConfig.ContourResponseTimeout,
		PodDistributionSkew:           cacConfig.PodDistributionSkew,
		EnableWebsocket:               cacConfig.EnableWebsocket,
		AddHeadless:                   cacConfig.AddHeadless,
		CreateContourGateway:          cacConfig.CreateContourGateway,
		PDBMinAvailable:               cacConfig.PDBMinAvailable,
		PDBMaxUnavailable:             cacConfig.PDBMaxUnavailable,
		PodAnnotations:                cacConfig.PodAnnotations,
	}

	log.Info().
		Str("bu", result.BU).
		Str("team", result.Team).
		Str("priorityV2", result.PriorityV2).
		Str("primaryOwner", result.PrimaryOwner).
		Str("secondaryOwner", result.SecondaryOwner).
		Msg("convertCACToServiceConfig: Converted config-as-code to serviceConfigData")

	return result
}

// convertDBToServiceConfig converts database ServiceConfig to serviceConfigData
func convertDBToServiceConfig(dbConfig *serviceconfig.ServiceConfig, workingEnv string) *serviceConfigData {
	// Parse service type config from JSON
	serviceType := make(map[string]bool)
	if len(dbConfig.Config) > 0 {
		var serviceTypeConfig map[string]interface{}
		if err := json.Unmarshal(dbConfig.Config, &serviceTypeConfig); err == nil {
			for k, v := range serviceTypeConfig {
				if boolVal, ok := v.(bool); ok {
					serviceType[k] = boolVal
				} else if strVal, ok := v.(string); ok {
					serviceType[k] = strVal == "true"
				}
			}
		}
	}

	// Use branch name from config or derive from workingEnv
	branchName := dbConfig.BranchName
	if branchName == "" {
		branchName = configs.GetBranchName(workingEnv)
	}

	// Set default probe values for database configs (backward compatibility)
	// These defaults match the ones in ServiceConfig.ApplyDefaults()
	livenessFailureThreshold := "5"
	livenessPeriodSeconds := "10"
	livenessSuccessThreshold := "1"
	livenessTimeoutSeconds := "2"
	readinessFailureThreshold := "5"
	readinessPeriodSeconds := "10"
	readinessSuccessThreshold := "1"
	readinessTimeoutSeconds := "2"

	// Set default autoscaling values for database configs (backward compatibility)
	asEnabled := "true"
	asPoll := "30"
	asDownPeriod := "300"
	asUpPeriod := "60"
	asUpStableWindow := "300"
	asDownStableWindow := "1800"
	asTriggerType := "AverageValue"
	asTriggerMetric := "cpu"
	asTriggerValue := "50"
	asDownPodCount := "2"
	asUpPodCount := "2"
	asUpPodPercentage := "10"
	cpuThreshold := "50"

	// Set default deployment values for database configs (backward compatibility)
	maxSurge := "50"
	terminationGracePeriodSeconds := "300"
	contourResponseTimeout := "false"
	podDistributionSkew := "false"
	enableWebsocket := "false"
	addHeadless := "false"
	createContourGateway := "false"
	pdbMinAvailable := ""
	pdbMaxUnavailable := "10%"

	// Set default nodeSelector based on environment (matches RingMaster logic)
	// Use GetEnvConfig to handle unknown environments gracefully
	envConfig := github.GetEnvConfig(workingEnv)
	configEnv := envConfig["config_env"]
	nodeSelector := "dedicated" // Default
	if configEnv == "int" {
		nodeSelector = "cloud.google.com/compute-class"
	}

	return &serviceConfigData{
		RepoName:                  dbConfig.RepoName,
		BranchName:                branchName,
		HealthCheck:               dbConfig.HealthCheck,
		AppPort:                   dbConfig.AppPort,
		IngressClass:              dbConfig.IngressClass,
		AppType:                   dbConfig.AppType,
		PrimaryOwner:              dbConfig.PrimaryOwner,
		SecondaryOwner:            dbConfig.SecondaryOwner,
		Team:                      dbConfig.Team,
		BU:                        dbConfig.BU,
		PriorityV2:                dbConfig.PriorityV2,
		ServiceType:               serviceType,
		LivenessFailureThreshold:  livenessFailureThreshold,
		LivenessPeriodSeconds:     livenessPeriodSeconds,
		LivenessSuccessThreshold:  livenessSuccessThreshold,
		LivenessTimeoutSeconds:    livenessTimeoutSeconds,
		ReadinessFailureThreshold: readinessFailureThreshold,
		ReadinessPeriodSeconds:    readinessPeriodSeconds,
		ReadinessSuccessThreshold: readinessSuccessThreshold,
		ReadinessTimeoutSeconds:   readinessTimeoutSeconds,
		NodeSelector:              nodeSelector,
		// Autoscaling configuration
		ASEnabled:          asEnabled,
		ASPoll:             asPoll,
		ASDownPeriod:       asDownPeriod,
		ASUpPeriod:         asUpPeriod,
		ASUpStableWindow:   asUpStableWindow,
		ASDownStableWindow: asDownStableWindow,
		ASTriggerType:      asTriggerType,
		ASTriggerMetric:    asTriggerMetric,
		ASTriggerValue:     asTriggerValue,
		ASDownPodCount:     asDownPodCount,
		ASUpPodCount:       asUpPodCount,
		ASUpPodPercentage:  asUpPodPercentage,
		CPUThreshold:       cpuThreshold,
		// Deployment configuration
		MaxSurge:                      maxSurge,
		TerminationGracePeriodSeconds: terminationGracePeriodSeconds,
		ContourResponseTimeout:        contourResponseTimeout,
		PodDistributionSkew:           podDistributionSkew,
		EnableWebsocket:               enableWebsocket,
		AddHeadless:                   addHeadless,
		CreateContourGateway:          createContourGateway,
		PDBMinAvailable:               pdbMinAvailable,
		PDBMaxUnavailable:             pdbMaxUnavailable,
		PodAnnotations:                make(map[string]string), // Database doesn't store podAnnotations, use empty map
	}
}

func (h *Handler) CreateDeployable(request *DeployableRequest, workingEnv string) (string, error) {
	log.Info().
		Str("appName", request.AppName).
		Str("serviceName", request.ServiceName).
		Str("workingEnv", workingEnv).
		Str("createdBy", request.CreatedBy).
		Msg("CreateDeployable: Starting deployable creation")

	// 1. Get service config (from config-as-code or database)
	serviceConfig, err := h.loadServiceConfig(request.ServiceName, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("serviceName", request.ServiceName).
			Str("workingEnv", workingEnv).
			Msg("CreateDeployable: Failed to get service config")
		return "", fmt.Errorf("failed to get service config: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Str("serviceName", request.ServiceName).
		Int("appPort", serviceConfig.AppPort).
		Str("repoName", serviceConfig.RepoName).
		Int("podAnnotationsCount", len(serviceConfig.PodAnnotations)).
		Interface("podAnnotations", serviceConfig.PodAnnotations).
		Msg("CreateDeployable: Service config retrieved successfully")

	// 2. Check ArgoCD configuration for the environment (optional - logs warning if not found)
	// ArgoCD configuration is optional - system will use defaults if not configured
	if err := argocd.ValidateArgoCDConfiguration(workingEnv); err != nil {
		log.Warn().
			Err(err).
			Str("appName", request.AppName).
			Str("serviceName", request.ServiceName).
			Str("workingEnv", workingEnv).
			Msg("CreateDeployable: ArgoCD configuration not found (optional - will use defaults if available)")
		// Continue - this is optional, not mandatory
	} else {
		log.Info().
			Str("appName", request.AppName).
			Str("workingEnv", workingEnv).
			Msg("CreateDeployable: ArgoCD configuration found")
	}

	// 3. Create initial DB entry
	log.Info().
		Str("appName", request.AppName).
		Msg("CreateDeployable: Marshaling deployable configuration to JSON")

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
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("serviceName", request.ServiceName).
			Msg("CreateDeployable: Failed to marshal deployable configuration to JSON")
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Int("configSize", len(configJSON)).
		Msg("CreateDeployable: Configuration marshaled successfully")

	// Standard format: Always use {env}-{appName} for database entry name
	// This applies to both single and multi-environment deployables
	// Extract environment suffix from workingEnv (e.g., "gcp_stg" -> "stg", "gcp_int" -> "int")
	if workingEnv == "" {
		return "", fmt.Errorf("workingEnv is required for deployable creation")
	}

	// Generate host using domain from config.yaml: <appname>.<domain>
	// Each environment defines its own domain in config.yaml, so no need to inject env into host
	domain := serviceConfig.Domain
	if domain == "" {
		// Fallback to hostUrlSuffix if domain not provided (backward compatibility)
		log.Warn().
			Str("appName", request.AppName).
			Str("workingEnv", workingEnv).
			Msg("Domain not found in config.yaml, using hostUrlSuffix fallback")
		domain = hostUrlSuffix
	}

	// Host format: <appname>.<domain> (domain comes from each environment's config.yaml)
	host := fmt.Sprintf("%s.%s", request.AppName, domain)

	log.Info().
		Str("appName", request.AppName).
		Str("dbEntryName", request.AppName).
		Str("host", host).
		Str("workingEnv", workingEnv).
		Msg("CreateDeployable: Using unique database entry name and host per environment")

	// if serviceConfig.AppPort != 0 {
	// 	host = host + ":" + strconv.Itoa(serviceConfig.AppPort)
	// }
	deployableConfig := &servicedeployableconfig.ServiceDeployableConfig{
		Name:                    request.AppName,
		Service:                 request.ServiceName,
		Host:                    host, // Use environment-prefixed host for database uniqueness
		Active:                  true,
		CreatedBy:               request.CreatedBy,
		Config:                  configJSON,
		DeployableRunningStatus: false,
		DeployableHealth:        "DEPLOYMENT_REASON_ARGO_APP_HEALTH_DEGRADED",
		DeployableWorkFlowId:    "",
		MonitoringUrl: fmt.Sprintf("%s/d/a2605923-52c4-4834-bdae-97570966b765/model-inference-service?orgId=1&var-service=%s&var-query0=",
			grafanaBaseUrl, request.AppName), // Monitoring URL uses original appName
		WorkFlowStatus: "WORKFLOW_NOT_STARTED",
	}

	log.Info().
		Str("appName", request.AppName).
		Str("host", deployableConfig.Host).
		Str("service", deployableConfig.Service).
		Msg("CreateDeployable: Creating deployable entry in database")

	if err := h.repo.Create(deployableConfig); err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("host", deployableConfig.Host).
			Str("service", deployableConfig.Service).
			Str("workingEnv", workingEnv).
			Msg("CreateDeployable: Failed to create deployable config in database")
		return "", fmt.Errorf("failed to create deployable config: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Int("deployableID", deployableConfig.ID).
		Msg("CreateDeployable: Deployable entry created in database successfully")

	// 3. Prepare workflow payload (same structure as ringmaster payload)
	log.Info().
		Str("appName", request.AppName).
		Str("bu", serviceConfig.BU).
		Str("team", serviceConfig.Team).
		Str("priorityV2", serviceConfig.PriorityV2).
		Msg("CreateDeployable: Preparing workflow payload with service config values")

	workflowPayload := map[string]interface{}{
		"appName":              request.AppName,
		"service_name":         request.ServiceName, // Add service_name for template conditional logic (e.g., skip externalSecret for predator)
		"primaryOwner":         serviceConfig.PrimaryOwner,
		"repoName":             serviceConfig.RepoName,
		"branchName":           serviceConfig.BranchName,
		"secondaryOwner":       serviceConfig.SecondaryOwner,
		"healthCheck":          serviceConfig.HealthCheck,
		"host":                 host,   // Use host generated from domain in config.yaml
		"domain":               domain, // Add domain to payload for UpdateValuesProperties activity
		"appPort":              serviceConfig.AppPort,
		"team":                 serviceConfig.Team,
		"bu":                   serviceConfig.BU,
		"priorityV2":           serviceConfig.PriorityV2,
		"appType":              serviceConfig.AppType,
		"ingress_class":        serviceConfig.IngressClass,
		"triton_repository":    serviceConfig.TritonRepository,   // Triton image repository path without tag from config.yaml (required)
		"init_container_image": serviceConfig.InitContainerImage, // Init container image path from config.yaml (required if gcs_triton_path is enabled)
		"machine_type":         request.MachineType,
		"cpuRequest":           request.CPURequest,
		"cpuRequestUnit":       request.CPURequestUnit,
		"cpuLimit":             request.CPULimit,
		"cpuLimitUnit":         request.CPULimitUnit,
		"memoryRequest":        request.MemoryRequest,
		"memoryRequestUnit":    request.MemoryRequestUnit,
		"memoryLimit":          request.MemoryLimit,
		"memoryLimitUnit":      request.MemoryLimitUnit,
		"gpu_request":          request.GPURequest,
		"gpu_limit":            request.GPULimit,
		"as_min":               request.MinReplica,
		"as_max":               request.MaxReplica,
		"gcs_bucket_path":      request.GCSBucketPath,
		"triton_image_tag":     request.TritonImageTag,
		"created_by":           request.CreatedBy,
	}

	// Handle service_type: Get from request payload, default to "httpstateless" if not provided
	// Priority: 1. Request payload (preserve exact value including comma-separated like "httpstateless,grpc"), 2. Service config from config.yaml, 3. Default "httpstateless"
	serviceType := request.ServiceType
	if serviceType == "" {
		// Check if service_type is set in config.yaml (as map[string]bool)
		// If grpc is true in config, use "grpc", otherwise default to "httpstateless"
		if serviceConfig.ServiceType != nil && serviceConfig.ServiceType["grpc"] {
			serviceType = "grpc"
		} else {
			serviceType = "httpstateless" // Default
		}
	}

	// Preserve the exact service_type value from request (can be "httpstateless", "grpc", or "httpstateless,grpc")
	// This allows comma-separated values to be passed through correctly
	workflowPayload["service_type"] = serviceType

	// Check if grpc is enabled (handles comma-separated values like "httpstateless,grpc")
	isGrpc := false
	if serviceType != "" {
		serviceTypes := strings.Split(strings.ReplaceAll(serviceType, " ", ""), ",")
		for _, st := range serviceTypes {
			if strings.EqualFold(st, "grpc") {
				isGrpc = true
				break
			}
		}
	}

	// Also set service_type_* flags for backward compatibility with templates
	// This maintains compatibility with existing template logic that checks service_type_grpc, etc.
	log.Info().
		Str("appName", request.AppName).
		Str("serviceType", serviceType).
		Bool("isGrpc", isGrpc).
		Int("serviceTypeCount", len(serviceConfig.ServiceType)).
		Msg("CreateDeployable: Adding service type configuration")

	// Set service_type_grpc flag if grpc is enabled (even in comma-separated format)
	workflowPayload["service_type_grpc"] = isGrpc
	// Set service_type_httpstateless flag
	hasHttpStateless := false
	if serviceType != "" {
		serviceTypes := strings.Split(strings.ReplaceAll(serviceType, " ", ""), ",")
		for _, st := range serviceTypes {
			if strings.EqualFold(st, "httpstateless") {
				hasHttpStateless = true
				break
			}
		}
	}
	workflowPayload["service_type_httpstateless"] = hasHttpStateless

	// Also preserve any other service_type flags from config.yaml
	for k, v := range serviceConfig.ServiceType {
		if k != "grpc" && k != "httpstateless" {
			workflowPayload[fmt.Sprintf("service_type_%s", k)] = v
		}
	}

	// Add remaining fields
	workflowPayload["serviceAccount"] = request.ServiceAccount
	workflowPayload["nodeSelectorValue"] = request.NodeSelectorValue
	workflowPayload["deploymentStrategy"] = request.DeploymentStrategy
	workflowPayload["gcs_triton_path"] = request.GCSTritonPath

	// Check if GCS fields are "NA" or empty, if so use localModelPath from config.yaml
	gcsBucketPath := strings.TrimSpace(request.GCSBucketPath)
	gcsTritonPath := strings.TrimSpace(request.GCSTritonPath)
	serviceAccount := strings.TrimSpace(request.ServiceAccount)

	// Check if any GCS-related field is "NA" or empty
	useLocalModelPath := false
	if gcsBucketPath == "" || strings.EqualFold(gcsBucketPath, "NA") ||
		gcsTritonPath == "" || strings.EqualFold(gcsTritonPath, "NA") ||
		serviceAccount == "" || strings.EqualFold(serviceAccount, "NA") {
		useLocalModelPath = true
		log.Info().
			Str("appName", request.AppName).
			Str("gcs_bucket_path", gcsBucketPath).
			Str("gcs_triton_path", gcsTritonPath).
			Str("serviceAccount", serviceAccount).
			Msg("GCS fields are NA or empty, will use localModelPath from config.yaml for local development")
	}

	if useLocalModelPath {
		workflowPayload["localModelPath"] = viper.GetString("LOCAL_MODEL_PATH")
		log.Info().
			Str("appName", request.AppName).
			Str("localModelPath", viper.GetString("LOCAL_MODEL_PATH")).
			Msg("Using localModelPath from config.yaml for local development")
	}

	// Add probe configuration from serviceConfig (read from config.yaml)
	// These values come from config.yaml and have defaults applied in ApplyDefaults()
	workflowPayload["liveness_failure_threshold"] = serviceConfig.LivenessFailureThreshold
	workflowPayload["liveness_period_seconds"] = serviceConfig.LivenessPeriodSeconds
	workflowPayload["liveness_success_threshold"] = serviceConfig.LivenessSuccessThreshold
	workflowPayload["liveness_timeout_seconds"] = serviceConfig.LivenessTimeoutSeconds
	workflowPayload["readiness_failure_threshold"] = serviceConfig.ReadinessFailureThreshold
	workflowPayload["readiness_period_seconds"] = serviceConfig.ReadinessPeriodSeconds
	workflowPayload["readiness_success_threshold"] = serviceConfig.ReadinessSuccessThreshold
	workflowPayload["readiness_timeout_seconds"] = serviceConfig.ReadinessTimeoutSeconds

	// Add nodeSelector from serviceConfig (read from config.yaml or set based on environment)
	// If specified in config.yaml, use it; otherwise, it's set based on environment in convertCACToServiceConfig/convertDBToServiceConfig
	if serviceConfig.NodeSelector != "" {
		workflowPayload["nodeSelector"] = serviceConfig.NodeSelector
	} else {
		// Set default based on environment (matches RingMaster logic)
		// Use GetEnvConfig to handle unknown environments gracefully
		envConfig := github.GetEnvConfig(workingEnv)
		configEnv := envConfig["config_env"]
		if configEnv == "int" {
			workflowPayload["nodeSelector"] = "cloud.google.com/compute-class"
		} else {
			workflowPayload["nodeSelector"] = "dedicated"
		}
	}

	// Add autoscaling configuration from serviceConfig (read from config.yaml)
	workflowPayload["as_enabled"] = serviceConfig.ASEnabled
	workflowPayload["as_poll"] = serviceConfig.ASPoll
	workflowPayload["as_down_period"] = serviceConfig.ASDownPeriod
	workflowPayload["as_up_period"] = serviceConfig.ASUpPeriod
	workflowPayload["as_up_stable_window"] = serviceConfig.ASUpStableWindow
	workflowPayload["as_down_stable_window"] = serviceConfig.ASDownStableWindow
	workflowPayload["as_trigger_type"] = serviceConfig.ASTriggerType
	workflowPayload["as_trigger_metric"] = serviceConfig.ASTriggerMetric
	workflowPayload["as_trigger_value"] = serviceConfig.ASTriggerValue
	workflowPayload["as_down_pod_count"] = serviceConfig.ASDownPodCount
	workflowPayload["as_up_pod_count"] = serviceConfig.ASUpPodCount
	workflowPayload["as_up_pod_percentage"] = serviceConfig.ASUpPodPercentage
	workflowPayload["cpuThreshold"] = serviceConfig.CPUThreshold

	// Add deployment configuration from serviceConfig (read from config.yaml)
	workflowPayload["maxSurge"] = serviceConfig.MaxSurge
	workflowPayload["terminationGracePeriodSeconds"] = serviceConfig.TerminationGracePeriodSeconds
	workflowPayload["contourResponseTimeout"] = serviceConfig.ContourResponseTimeout
	workflowPayload["podDistributionSkew"] = serviceConfig.PodDistributionSkew
	workflowPayload["enableWebsocket"] = serviceConfig.EnableWebsocket
	workflowPayload["addHeadless"] = serviceConfig.AddHeadless
	workflowPayload["createContourGateway"] = serviceConfig.CreateContourGateway
	workflowPayload["pdbMinAvailable"] = serviceConfig.PDBMinAvailable
	workflowPayload["pdbMaxUnavailable"] = serviceConfig.PDBMaxUnavailable

	// Add podAnnotations from serviceConfig (read from config.yaml)
	// These will be merged with payload overrides in CreateValuesYaml
	log.Info().
		Str("appName", request.AppName).
		Interface("podAnnotationsIsNil", serviceConfig.PodAnnotations == nil).
		Int("podAnnotationsLen", len(serviceConfig.PodAnnotations)).
		Interface("podAnnotations", serviceConfig.PodAnnotations).
		Msg("CreateDeployable: Checking podAnnotations from serviceConfig")

	if len(serviceConfig.PodAnnotations) > 0 {
		workflowPayload["pod_annotations_from_config"] = serviceConfig.PodAnnotations
		log.Info().
			Str("appName", request.AppName).
			Int("annotationCount", len(serviceConfig.PodAnnotations)).
			Interface("annotations", serviceConfig.PodAnnotations).
			Msg("CreateDeployable: Adding podAnnotations from config.yaml (will be used as defaults)")
	} else {
		log.Warn().
			Str("appName", request.AppName).
			Interface("podAnnotationsIsNil", serviceConfig.PodAnnotations == nil).
			Int("podAnnotationsLen", len(serviceConfig.PodAnnotations)).
			Msg("CreateDeployable: No podAnnotations found in config.yaml (empty or nil)")
	}

	// Add podAnnotations from request (higher precedence than config.yaml)
	// These will override config.yaml annotations in CreateValuesYaml merge logic
	if len(request.PodAnnotations) > 0 {
		workflowPayload["pod_annotations"] = request.PodAnnotations
		log.Info().
			Str("appName", request.AppName).
			Int("annotationCount", len(request.PodAnnotations)).
			Msg("CreateDeployable: Adding podAnnotations from request (will override config.yaml)")
	}

	// 4. Start workflow asynchronously (non-blocking, same pattern as RingMaster)
	log.Info().
		Str("appName", request.AppName).
		Str("workingEnv", workingEnv).
		Str("createdBy", request.CreatedBy).
		Int("payloadKeys", len(workflowPayload)).
		Msg("CreateDeployable: Starting onboarding workflow")

	workflowID, err := h.workflowHandler.StartOnboardingWorkflow(workflowPayload, request.CreatedBy, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("serviceName", request.ServiceName).
			Str("workingEnv", workingEnv).
			Str("createdBy", request.CreatedBy).
			Int("deployableID", deployableConfig.ID).
			Msg("CreateDeployable: Failed to start onboarding workflow - updating deployable status to WORKFLOW_START_FAILED")

		// Update deployable status to indicate workflow start failed
		deployableConfig.WorkFlowStatus = "WORKFLOW_START_FAILED"
		if updateErr := h.repo.Update(deployableConfig); updateErr != nil {
			log.Error().
				Err(updateErr).
				Str("appName", request.AppName).
				Int("deployableID", deployableConfig.ID).
				Str("workflowStatus", "WORKFLOW_START_FAILED").
				Msg("CreateDeployable: Failed to update deployable status after workflow start failure")
		} else {
			log.Info().
				Str("appName", request.AppName).
				Int("deployableID", deployableConfig.ID).
				Msg("CreateDeployable: Deployable status updated to WORKFLOW_START_FAILED")
		}
		return "", fmt.Errorf("failed to start onboarding workflow: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Str("workflowID", workflowID).
		Str("workingEnv", workingEnv).
		Msg("CreateDeployable: Onboarding workflow started successfully")

	// 5. Update deployable with workflow ID (non-blocking - workflow executes in background)
	log.Info().
		Str("appName", request.AppName).
		Str("workflowID", workflowID).
		Int("deployableID", deployableConfig.ID).
		Msg("CreateDeployable: Updating deployable with workflow ID")

	deployableConfig.DeployableWorkFlowId = workflowID
	deployableConfig.WorkFlowStatus = "WORKFLOW_RUNNING"
	if err := h.repo.Update(deployableConfig); err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("workflowID", workflowID).
			Int("deployableID", deployableConfig.ID).
			Str("workflowStatus", "WORKFLOW_RUNNING").
			Msg("CreateDeployable: Failed to update deployable with workflow ID in database")
		return "", fmt.Errorf("failed to update deployable with workflow ID: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Str("workflowID", workflowID).
		Int("deployableID", deployableConfig.ID).
		Msg("CreateDeployable: Deployable updated with workflow ID successfully")

	log.Info().
		Str("appName", request.AppName).
		Str("workflowID", workflowID).
		Str("workingEnv", workingEnv).
		Msg("Onboarding workflow started successfully (non-blocking)")

	return workflowID, nil
}

func (h *Handler) UpdateDeployable(request *DeployableRequest, workingEnv string) error {
	// 1. Get service config (from config-as-code or database)
	serviceConfig, err := h.loadServiceConfig(request.ServiceName, workingEnv)
	if err != nil {
		return fmt.Errorf("failed to get service config: %w", err)
	}

	// 2. Construct database entry name using standard format: {env}-{appName}
	// This applies to both single and multi-environment deployables
	if workingEnv == "" {
		return fmt.Errorf("workingEnv is required for deployable update")
	}

	log.Info().
		Str("appName", request.AppName).
		Str("dbEntryName", request.AppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateDeployable: Looking up deployable with environment-prefixed name")

	// 3. Get existing deployable configs by service using environment-prefixed name
	existingConfig, err := h.repo.GetByNameAndService(request.AppName, request.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get existing deployable configs: %w", err)
	}

	if existingConfig == nil {
		return fmt.Errorf("deployable config not found for app: %s (searched as: %s) in environment: %s", request.AppName, request.AppName, workingEnv)
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

	// Generate host using domain from config.yaml: <appname>.<domain>
	// Each environment defines its own domain in config.yaml, so no need to inject env into host
	domain := serviceConfig.Domain
	if domain == "" {
		// Fallback to hostUrlSuffix if domain not provided (backward compatibility)
		log.Warn().
			Str("appName", request.AppName).
			Str("workingEnv", workingEnv).
			Msg("Domain not found in config.yaml, using hostUrlSuffix fallback")
		domain = hostUrlSuffix
	}

	// Host format: <appname>.<domain> (domain comes from each environment's config.yaml)
	host := fmt.Sprintf("%s.%s", request.AppName, domain)

	// 4. Prepare workflow payload for updating GitHub files
	// This follows the same structure as CreateDeployable for consistency
	workflowPayload := map[string]interface{}{
		"appName":              request.AppName,
		"service_name":         request.ServiceName, // Add service_name for template conditional logic (e.g., skip externalSecret for predator)
		"primaryOwner":         serviceConfig.PrimaryOwner,
		"repoName":             serviceConfig.RepoName,
		"branchName":           serviceConfig.BranchName,
		"secondaryOwner":       serviceConfig.SecondaryOwner,
		"healthCheck":          serviceConfig.HealthCheck,
		"host":                 host,   // Use host generated from domain in config.yaml
		"domain":               domain, // Add domain to payload for UpdateValuesProperties activity
		"appPort":              serviceConfig.AppPort,
		"team":                 serviceConfig.Team,
		"bu":                   serviceConfig.BU,
		"priorityV2":           serviceConfig.PriorityV2,
		"appType":              serviceConfig.AppType,
		"ingress_class":        serviceConfig.IngressClass,
		"triton_repository":    serviceConfig.TritonRepository,   // Triton image repository path without tag from config.yaml (required)
		"init_container_image": serviceConfig.InitContainerImage, // Init container image path from config.yaml (required if gcs_triton_path is enabled)
		"machine_type":         request.MachineType,
		"cpuRequest":           request.CPURequest,
		"cpuRequestUnit":       request.CPURequestUnit,
		"cpuLimit":             request.CPULimit,
		"cpuLimitUnit":         request.CPULimitUnit,
		"memoryRequest":        request.MemoryRequest,
		"memoryRequestUnit":    request.MemoryRequestUnit,
		"memoryLimit":          request.MemoryLimit,
		"memoryLimitUnit":      request.MemoryLimitUnit,
		"gpu_request":          request.GPURequest,
		"gpu_limit":            request.GPULimit,
		"as_min":               request.MinReplica,
		"as_max":               request.MaxReplica,
		"gcs_bucket_path":      request.GCSBucketPath,
		"triton_image_tag":     request.TritonImageTag,
		"created_by":           request.CreatedBy,
		"serviceAccount":       request.ServiceAccount,
		"nodeSelectorValue":    request.NodeSelectorValue,
		"deploymentStrategy":   request.DeploymentStrategy,
		"gcs_triton_path":      request.GCSTritonPath,
	}

	// Handle service_type: Get from request payload, default to "httpstateless" if not provided
	serviceType := request.ServiceType
	if serviceType == "" {
		// Check if service_type is set in config.yaml (as map[string]bool)
		if serviceConfig.ServiceType != nil && serviceConfig.ServiceType["grpc"] {
			serviceType = "grpc"
		} else {
			serviceType = "httpstateless" // Default
		}
	}
	workflowPayload["service_type"] = serviceType

	// Check if grpc is enabled (handles comma-separated values like "httpstateless,grpc")
	isGrpc := false
	if serviceType != "" {
		serviceTypes := strings.Split(strings.ReplaceAll(serviceType, " ", ""), ",")
		for _, st := range serviceTypes {
			if strings.EqualFold(st, "grpc") {
				isGrpc = true
				break
			}
		}
	}
	workflowPayload["service_type_grpc"] = isGrpc

	// Set service_type_httpstateless flag
	hasHttpStateless := false
	if serviceType != "" {
		serviceTypes := strings.Split(strings.ReplaceAll(serviceType, " ", ""), ",")
		for _, st := range serviceTypes {
			if strings.EqualFold(st, "httpstateless") {
				hasHttpStateless = true
				break
			}
		}
	}
	workflowPayload["service_type_httpstateless"] = hasHttpStateless

	// Also preserve any other service_type flags from config.yaml
	for k, v := range serviceConfig.ServiceType {
		if k != "grpc" && k != "httpstateless" {
			workflowPayload[fmt.Sprintf("service_type_%s", k)] = v
		}
	}

	// Add probe configuration from serviceConfig (read from config.yaml)
	workflowPayload["liveness_failure_threshold"] = serviceConfig.LivenessFailureThreshold
	workflowPayload["liveness_period_seconds"] = serviceConfig.LivenessPeriodSeconds
	workflowPayload["liveness_success_threshold"] = serviceConfig.LivenessSuccessThreshold
	workflowPayload["liveness_timeout_seconds"] = serviceConfig.LivenessTimeoutSeconds
	workflowPayload["readiness_failure_threshold"] = serviceConfig.ReadinessFailureThreshold
	workflowPayload["readiness_period_seconds"] = serviceConfig.ReadinessPeriodSeconds
	workflowPayload["readiness_success_threshold"] = serviceConfig.ReadinessSuccessThreshold
	workflowPayload["readiness_timeout_seconds"] = serviceConfig.ReadinessTimeoutSeconds

	// Add nodeSelector from serviceConfig (read from config.yaml or set based on environment)
	if serviceConfig.NodeSelector != "" {
		workflowPayload["nodeSelector"] = serviceConfig.NodeSelector
	} else {
		// Set default based on environment
		envConfig := github.GetEnvConfig(workingEnv)
		configEnv := envConfig["config_env"]
		if configEnv == "int" {
			workflowPayload["nodeSelector"] = "cloud.google.com/compute-class"
		} else {
			workflowPayload["nodeSelector"] = "dedicated"
		}
	}

	// Add autoscaling configuration from serviceConfig (read from config.yaml)
	workflowPayload["as_enabled"] = serviceConfig.ASEnabled
	workflowPayload["as_poll"] = serviceConfig.ASPoll
	workflowPayload["as_down_period"] = serviceConfig.ASDownPeriod
	workflowPayload["as_up_period"] = serviceConfig.ASUpPeriod
	workflowPayload["as_up_stable_window"] = serviceConfig.ASUpStableWindow
	workflowPayload["as_down_stable_window"] = serviceConfig.ASDownStableWindow
	workflowPayload["as_trigger_type"] = serviceConfig.ASTriggerType
	workflowPayload["as_trigger_metric"] = serviceConfig.ASTriggerMetric
	workflowPayload["as_trigger_value"] = serviceConfig.ASTriggerValue
	workflowPayload["as_down_pod_count"] = serviceConfig.ASDownPodCount
	workflowPayload["as_up_pod_count"] = serviceConfig.ASUpPodCount
	workflowPayload["as_up_pod_percentage"] = serviceConfig.ASUpPodPercentage
	workflowPayload["cpuThreshold"] = serviceConfig.CPUThreshold

	// Add deployment configuration from serviceConfig (read from config.yaml)
	workflowPayload["maxSurge"] = serviceConfig.MaxSurge
	workflowPayload["terminationGracePeriodSeconds"] = serviceConfig.TerminationGracePeriodSeconds
	workflowPayload["contourResponseTimeout"] = serviceConfig.ContourResponseTimeout
	workflowPayload["podDistributionSkew"] = serviceConfig.PodDistributionSkew
	workflowPayload["enableWebsocket"] = serviceConfig.EnableWebsocket
	workflowPayload["addHeadless"] = serviceConfig.AddHeadless
	workflowPayload["createContourGateway"] = serviceConfig.CreateContourGateway
	workflowPayload["pdbMinAvailable"] = serviceConfig.PDBMinAvailable
	workflowPayload["pdbMaxUnavailable"] = serviceConfig.PDBMaxUnavailable

	// Add podAnnotations from serviceConfig (read from config.yaml)
	if len(serviceConfig.PodAnnotations) > 0 {
		workflowPayload["pod_annotations_from_config"] = serviceConfig.PodAnnotations
	}

	// Add podAnnotations from request if provided (higher precedence than config.yaml)
	if len(request.PodAnnotations) > 0 {
		workflowPayload["pod_annotations"] = request.PodAnnotations
	}

	// 5. Update GitHub files directly (values.yaml and values_properties.yaml)
	// This replaces the RingMaster API call with direct GitHub updates
	log.Info().
		Str("appName", request.AppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateDeployable: Updating GitHub files (values.yaml and values_properties.yaml)")

	// Update values.yaml
	if err := activities.CreateValuesYaml(workflowPayload, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("workingEnv", workingEnv).
			Msg("UpdateDeployable: Failed to update values.yaml")
		return fmt.Errorf("failed to update values.yaml: %w", err)
	}

	// Update values_properties.yaml
	if err := activities.UpdateValuesProperties(workflowPayload, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", request.AppName).
			Str("workingEnv", workingEnv).
			Msg("UpdateDeployable: Failed to update values_properties.yaml")
		return fmt.Errorf("failed to update values_properties.yaml: %w", err)
	}

	log.Info().
		Str("appName", request.AppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateDeployable: Successfully updated deployable and GitHub files")

	return nil
}
