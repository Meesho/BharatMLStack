package configs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// LabelsConfig represents optional labels configuration
// If provided, these values drive directory structure, file names, and ArgoCD labels
// If absent, defaults are used (see ApplyDefaults method)
// Note: ArgoCD has filename length constraints (Kubernetes resource names: 63 chars max)
// Keep bu, team, and other label values concise to avoid exceeding limits
type LabelsConfig struct {
	// Optional: Business unit (used as-is, no normalization)
	// Max recommended length: 20 chars to avoid ArgoCD filename issues
	BU string `yaml:"bu" json:"bu"`

	// Optional: Team name (used as-is, no normalization)
	// Max recommended length: 20 chars to avoid ArgoCD filename issues
	Team string `yaml:"team" json:"team"`

	// Optional: Primary owner email (username extracted for labels)
	// Max recommended length: 50 chars (email format)
	PrimaryOwner string `yaml:"primary_owner" json:"primary_owner"`

	// Optional: Secondary owner email (username extracted for labels)
	// Max recommended length: 50 chars (email format)
	SecondaryOwner string `yaml:"secondary_owner" json:"secondary_owner"`

	// Optional: Priority level (e.g., "cp1", "p0", "p1")
	// Max recommended length: 10 chars
	PriorityV2 string `yaml:"priority_v2" json:"priority_v2"`

	// Optional: Custom labels for vendor extensibility
	CustomLabels map[string]string `yaml:"custom_labels" json:"custom_labels"`
}

// ServiceConfig represents the service configuration loaded from config-as-code
// This replaces the database-based service_config table for open-source compatibility
type ServiceConfig struct {
	// Mandatory fields (must be provided)
	RepoName     string `yaml:"repo_name" json:"repo_name"`         // Mandatory: GitHub repository name
	AppPort      int    `yaml:"app_port" json:"app_port"`           // Mandatory: Application port
	IngressClass string `yaml:"ingress_class" json:"ingress_class"` // Mandatory: Ingress class
	Domain       string `yaml:"domain" json:"domain"`               // Mandatory: Domain name for host generation (e.g., "meesho.int", "example.com")

	// Optional fields (with defaults)
	HealthCheck        string `yaml:"health_check" json:"health_check"`                 // Optional: Health check endpoint (default: /health)
	AppType            string `yaml:"app_type" json:"app_type"`                         // Optional: Application type (python, java, go, etc.)
	TritonRepository   string `yaml:"triton_repository" json:"triton_repository"`       // Optional: Triton inference server image repository path without tag (e.g., "asia-southeast1-docker.pkg.dev/meesho-devops-admin-0622/prd/ml/triton-inference-server/triton-inference-server")
	TelegrafImage      string `yaml:"telegraf_image" json:"telegraf_image"`             // Optional: Complete Telegraf image path with tag (e.g., "asia-southeast1-docker.pkg.dev/meesho-devops-admin-0622/telegraf:1.24.4-arm64")
	InitContainerImage string `yaml:"init_container_image" json:"init_container_image"` // Optional: Complete init container image path with tag for GCS operations (e.g., "asia-southeast1-docker.pkg.dev/meesho-devops-admin-0622/admin/devops/build-tools:lunar-v1.0.3-slim")

	// Labels section (optional, vendor-agnostic)
	// If provided, these values drive directory structure, file names, and ArgoCD labels
	// If absent, defaults are used (see ApplyDefaults method)
	Labels *LabelsConfig `yaml:"labels" json:"labels"` // Optional: Labels configuration section

	// Legacy fields (deprecated, kept for backward compatibility)
	// These are now read from Labels section if present
	PrimaryOwner   string          `yaml:"primary_owner" json:"primary_owner"`     // Deprecated: Use labels.primary_owner
	SecondaryOwner string          `yaml:"secondary_owner" json:"secondary_owner"` // Deprecated: Use labels.secondary_owner
	Team           string          `yaml:"team" json:"team"`                       // Deprecated: Use labels.team
	BU             string          `yaml:"bu" json:"bu"`                           // Deprecated: Use labels.bu
	PriorityV2     string          `yaml:"priority_v2" json:"priority_v2"`         // Deprecated: Use labels.priority_v2
	ServiceType    map[string]bool `yaml:"service_type" json:"service_type"`       // Optional: Service type flags (Meesho-specific)

	// Custom labels (for vendor extensibility)
	CustomLabels map[string]string `yaml:"custom_labels" json:"custom_labels"` // Optional: Custom key-value pairs for labels

	// Probe configuration (optional, with defaults)
	LivenessFailureThreshold  string `yaml:"liveness_failure_threshold" json:"liveness_failure_threshold"`   // Optional: Liveness probe failure threshold (default: "5")
	LivenessPeriodSeconds     string `yaml:"liveness_period_seconds" json:"liveness_period_seconds"`         // Optional: Liveness probe period in seconds (default: "10")
	LivenessSuccessThreshold  string `yaml:"liveness_success_threshold" json:"liveness_success_threshold"`   // Optional: Liveness probe success threshold (default: "1")
	LivenessTimeoutSeconds    string `yaml:"liveness_timeout_seconds" json:"liveness_timeout_seconds"`       // Optional: Liveness probe timeout in seconds (default: "2")
	ReadinessFailureThreshold string `yaml:"readiness_failure_threshold" json:"readiness_failure_threshold"` // Optional: Readiness probe failure threshold (default: "5")
	ReadinessPeriodSeconds    string `yaml:"readiness_period_seconds" json:"readiness_period_seconds"`       // Optional: Readiness probe period in seconds (default: "10")
	ReadinessSuccessThreshold string `yaml:"readiness_success_threshold" json:"readiness_success_threshold"` // Optional: Readiness probe success threshold (default: "1")
	ReadinessTimeoutSeconds   string `yaml:"readiness_timeout_seconds" json:"readiness_timeout_seconds"`     // Optional: Readiness probe timeout in seconds (default: "2")

	// Node selector configuration (optional, with environment-based defaults)
	NodeSelector string `yaml:"node_selector" json:"node_selector"` // Optional: Kubernetes node selector key (default: "dedicated" for most envs, "cloud.google.com/compute-class" for int)

	// Autoscaling configuration (optional, with defaults)
	ASEnabled          string `yaml:"as_enabled" json:"as_enabled"`                       // Optional: Enable autoscaling (default: "true")
	ASPoll             string `yaml:"as_poll" json:"as_poll"`                             // Optional: Autoscaling polling interval in seconds (default: "30")
	ASDownPeriod       string `yaml:"as_down_period" json:"as_down_period"`               // Optional: Autoscaling down period in seconds (default: "300")
	ASUpPeriod         string `yaml:"as_up_period" json:"as_up_period"`                   // Optional: Autoscaling up period in seconds (default: "60")
	ASUpStableWindow   string `yaml:"as_up_stable_window" json:"as_up_stable_window"`     // Optional: Autoscaling up stable window in seconds (default: "300")
	ASDownStableWindow string `yaml:"as_down_stable_window" json:"as_down_stable_window"` // Optional: Autoscaling down stable window in seconds (default: "1800")
	ASTriggerType      string `yaml:"as_trigger_type" json:"as_trigger_type"`             // Optional: Autoscaling trigger type (default: "AverageValue")
	ASTriggerMetric    string `yaml:"as_trigger_metric" json:"as_trigger_metric"`         // Optional: Autoscaling trigger metric (default: "cpu")
	ASTriggerValue     string `yaml:"as_trigger_value" json:"as_trigger_value"`           // Optional: Autoscaling trigger value (default: "50")
	ASDownPodCount     string `yaml:"as_down_pod_count" json:"as_down_pod_count"`         // Optional: Autoscaling down pod count (default: "2")
	ASUpPodCount       string `yaml:"as_up_pod_count" json:"as_up_pod_count"`             // Optional: Autoscaling up pod count (default: "2")
	ASUpPodPercentage  string `yaml:"as_up_pod_percentage" json:"as_up_pod_percentage"`   // Optional: Autoscaling up pod percentage (default: "10")
	CPUThreshold       string `yaml:"cpu_threshold" json:"cpu_threshold"`                 // Optional: CPU threshold for autoscaling (default: "50")

	// Deployment configuration (optional, with defaults)
	MaxSurge                      string            `yaml:"max_surge" json:"max_surge"`                                               // Optional: Max surge percentage for rolling update (default: "50")
	TerminationGracePeriodSeconds string            `yaml:"termination_grace_period_seconds" json:"termination_grace_period_seconds"` // Optional: Termination grace period in seconds (default: "300")
	ContourResponseTimeout        string            `yaml:"contour_response_timeout" json:"contour_response_timeout"`                 // Optional: Contour response timeout (default: "false")
	PodDistributionSkew           string            `yaml:"pod_distribution_skew" json:"pod_distribution_skew"`                       // Optional: Pod distribution skew (default: "false")
	EnableWebsocket               string            `yaml:"enable_websocket" json:"enable_websocket"`                                 // Optional: Enable websocket (default: "false")
	AddHeadless                   string            `yaml:"add_headless" json:"add_headless"`                                         // Optional: Add headless service (default: "false")
	CreateContourGateway          string            `yaml:"create_contour_gateway" json:"create_contour_gateway"`                     // Optional: Create contour gateway (default: "false")
	PDBMinAvailable               string            `yaml:"pdb_min_available" json:"pdb_min_available"`                               // Optional: PDB min available (default: "")
	PDBMaxUnavailable             string            `yaml:"pdb_max_unavailable" json:"pdb_max_unavailable"`                           // Optional: PDB max unavailable (default: "10%")
	PodAnnotations                map[string]string `yaml:"pod_annotations" json:"pod_annotations"`                                   // Optional: Pod annotations (default: {})
}

// GetBranchName returns the branch name based on working environment
// This replaces the database branch_name field
func GetBranchName(workingEnv string) string {
	switch workingEnv {
	case "gcp_stg", "stg":
		return "develop"
	case "gcp_int", "int":
		return "develop"
	case "gcp_prd", "prd":
		return "main"
	case "gcp_ftr", "ftr":
		return "develop"
	case "gcp_dev", "dev":
		return "develop"
	default:
		return "main" // Default to main
	}
}

// ServiceConfigLoader loads service configuration from config-as-code
type ServiceConfigLoader interface {
	LoadServiceConfig(serviceName, workingEnv string) (*ServiceConfig, error)
}

// GitHubServiceConfigLoader loads service config from GitHub repository
type GitHubServiceConfigLoader struct {
	repoOwner string
	repoName  string
}

// NewGitHubServiceConfigLoader creates a new GitHub-based service config loader
func NewGitHubServiceConfigLoader(repoOwner, repoName string) *GitHubServiceConfigLoader {
	return &GitHubServiceConfigLoader{
		repoOwner: repoOwner,
		repoName:  repoName,
	}
}

// LoadServiceConfig loads service configuration from GitHub
// Path format: services/{serviceName}/{env}/config.yaml
func (l *GitHubServiceConfigLoader) LoadServiceConfig(serviceName, workingEnv string) (*ServiceConfig, error) {
	// Map workingEnv to config env name (with defaults for unknown environments)
	envConfig := github.GetEnvConfig(workingEnv)
	envName := envConfig["config_env"]
	if envName == "" {
		envName = workingEnv
	}

	// Construct file path
	configPath := fmt.Sprintf("horizon/configs/services/%s/%s/config.yaml", serviceName, envName)
	branchName := GetBranchName(workingEnv)

	log.Info().
		Str("serviceName", serviceName).
		Str("workingEnv", workingEnv).
		Str("configEnv", envName).
		Str("configPath", configPath).
		Str("branch", branchName).
		Str("repo", l.repoName).
		Msg("Loading service config from GitHub")

	// Get file content from GitHub
	client, err := github.GetGitHubClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	ctx := context.Background()
	fileContent, err := github.GetFile(ctx, client, l.repoName, configPath, branchName)
	if err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Str("branch", branchName).
			Msg("Failed to load service config from GitHub")
		return nil, fmt.Errorf("failed to load service config from GitHub: %w", err)
	}

	// Decode base64 content
	content, err := fileContent.GetContent()
	if err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Msg("Failed to decode service config content from GitHub")
		return nil, fmt.Errorf("failed to decode service config content: %w", err)
	}
	fileContentBytes := []byte(content)

	// Parse YAML
	var config ServiceConfig
	if err := yaml.Unmarshal(fileContentBytes, &config); err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Msg("Failed to parse service config YAML")
		return nil, fmt.Errorf("failed to parse service config YAML: %w", err)
	}

	// Log podAnnotations immediately after YAML parsing (before ApplyDefaults)
	if len(config.PodAnnotations) > 0 {
		log.Info().
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Int("podAnnotationsCount", len(config.PodAnnotations)).
			Interface("podAnnotations", config.PodAnnotations).
			Msg("Service config: podAnnotations found in YAML from GitHub (after parsing, before ApplyDefaults)")
	} else {
		log.Warn().
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Interface("podAnnotationsIsNil", config.PodAnnotations == nil).
			Interface("podAnnotations", config.PodAnnotations).
			Msg("Service config: podAnnotations is empty or nil after YAML parsing from GitHub")
	}

	// Validate mandatory fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service config: %w", err)
	}

	// Apply defaults
	config.ApplyDefaults()

	logFields := log.Info().
		Str("serviceName", serviceName).
		Str("repoName", config.RepoName).
		Int("appPort", config.AppPort).
		Str("ingressClass", config.IngressClass)

	// Log podAnnotations if present
	if len(config.PodAnnotations) > 0 {
		logFields = logFields.
			Int("podAnnotationsCount", len(config.PodAnnotations)).
			Interface("podAnnotations", config.PodAnnotations)
	}

	logFields.Msg("Service config loaded successfully from GitHub")

	return &config, nil
}

// LocalFileServiceConfigLoader loads service config from local filesystem
// Useful for development and testing
// Standard path format: configs/services/{serviceName}/{env}/config.yaml
type LocalFileServiceConfigLoader struct {
	basePath string
}

// NewLocalFileServiceConfigLoader creates a new local file-based service config loader
func NewLocalFileServiceConfigLoader(basePath string) *LocalFileServiceConfigLoader {
	return &LocalFileServiceConfigLoader{
		basePath: basePath,
	}
}

// LoadServiceConfig loads service configuration from local filesystem
// Standard path format: configs/services/{serviceName}/{env}/config.yaml
func (l *LocalFileServiceConfigLoader) LoadServiceConfig(serviceName, workingEnv string) (*ServiceConfig, error) {
	// Map workingEnv to config env name using GetEnvConfig which handles unknown environments gracefully
	// This allows any environment name to be used without blocking onboarding
	configEnv := github.GetEnvConfig(workingEnv)
	envName := configEnv["config_env"]
	if envName == "" {
		envName = workingEnv
	}

	// Standard path format: configs/services/{serviceName}/{env}/config.yaml
	// basePath is the root directory (e.g., "." or "./configs")
	// If basePath already ends with "configs", use services/... directly
	// Otherwise, use configs/services/...
	var configPath string
	if filepath.Base(l.basePath) == "configs" || strings.HasSuffix(l.basePath, "/configs") || strings.HasSuffix(l.basePath, "\\configs") {
		// basePath already includes "configs", so use services/... directly
		configPath = filepath.Join(l.basePath, "services", serviceName, envName, "config.yaml")
	} else {
		// basePath is root (e.g., "."), use configs/services/...
		configPath = filepath.Join(l.basePath, "configs", "services", serviceName, envName, "config.yaml")
	}

	log.Info().
		Str("serviceName", serviceName).
		Str("workingEnv", workingEnv).
		Str("configEnv", envName).
		Str("configPath", configPath).
		Msg("Loading service config from standard path (configs/services/{service}/{env}/config.yaml)")

	fileContent, err := os.ReadFile(configPath)
	if err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Msg("Failed to read service config file")
		return nil, fmt.Errorf("failed to read service config file at %s: %w", configPath, err)
	}

	log.Info().
		Str("serviceName", serviceName).
		Str("configPath", configPath).
		Int("fileSize", len(fileContent)).
		Msg("Service config file found")

	// Parse YAML
	var config ServiceConfig
	if err := yaml.Unmarshal(fileContent, &config); err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Msg("Failed to parse service config YAML")
		return nil, fmt.Errorf("failed to parse service config YAML: %w", err)
	}

	// Log podAnnotations immediately after YAML parsing (before ApplyDefaults)
	if len(config.PodAnnotations) > 0 {
		log.Info().
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Int("podAnnotationsCount", len(config.PodAnnotations)).
			Interface("podAnnotations", config.PodAnnotations).
			Msg("Service config: podAnnotations found in YAML (after parsing, before ApplyDefaults)")
	} else {
		log.Warn().
			Str("serviceName", serviceName).
			Str("configPath", configPath).
			Interface("podAnnotationsIsNil", config.PodAnnotations == nil).
			Interface("podAnnotations", config.PodAnnotations).
			Msg("Service config: podAnnotations is empty or nil after YAML parsing")
	}

	// Validate mandatory fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service config: %w", err)
	}

	// Apply defaults
	config.ApplyDefaults()

	logFields := log.Info().
		Str("serviceName", serviceName).
		Str("repoName", config.RepoName).
		Int("appPort", config.AppPort).
		Str("ingressClass", config.IngressClass).
		Str("bu", config.BU).
		Str("team", config.Team).
		Str("priorityV2", config.PriorityV2).
		Str("primaryOwner", config.PrimaryOwner).
		Str("secondaryOwner", config.SecondaryOwner)

	// Log podAnnotations if present
	if len(config.PodAnnotations) > 0 {
		logFields = logFields.
			Int("podAnnotationsCount", len(config.PodAnnotations)).
			Interface("podAnnotations", config.PodAnnotations)
	}

	logFields.Msg("Service config loaded successfully from local file")

	return &config, nil
}

// NewServiceConfigLoader creates a service config loader based on the source type
// source: "local" or "github"
// basePath: Base path for local files (ignored for GitHub)
// repoOwner: GitHub repository owner (required for GitHub source)
// repoName: GitHub repository name (required for GitHub source)
func NewServiceConfigLoader(source, basePath, repoOwner, repoName string) (ServiceConfigLoader, error) {
	switch source {
	case "local":
		if basePath == "" {
			basePath = "./configs" // Default path
		}
		return NewLocalFileServiceConfigLoader(basePath), nil
	case "github":
		if repoOwner == "" || repoName == "" {
			return nil, fmt.Errorf("repoOwner and repoName are required for GitHub source")
		}
		return NewGitHubServiceConfigLoader(repoOwner, repoName), nil
	default:
		return nil, fmt.Errorf("unknown service config source: %s (must be 'local' or 'github')", source)
	}
}

// Validate validates that mandatory fields are present
func (c *ServiceConfig) Validate() error {
	if c.RepoName == "" {
		return fmt.Errorf("repo_name is mandatory")
	}
	if c.AppPort == 0 {
		return fmt.Errorf("app_port is mandatory and must be > 0")
	}
	if c.IngressClass == "" {
		return fmt.Errorf("ingress_class is mandatory")
	}
	if c.Domain == "" {
		return fmt.Errorf("domain is mandatory (used for host generation: <appname>.<domain>)")
	}
	return nil
}

// ApplyDefaults applies default values for optional fields
func (c *ServiceConfig) ApplyDefaults() {
	if c.HealthCheck == "" {
		c.HealthCheck = "/health"
	}
	if c.ServiceType == nil {
		c.ServiceType = make(map[string]bool)
	}
	if c.CustomLabels == nil {
		c.CustomLabels = make(map[string]string)
	}
	// Initialize PodAnnotations if nil (but don't override if it has values from YAML)
	if c.PodAnnotations == nil {
		c.PodAnnotations = make(map[string]string)
	}

	// Handle labels section: migrate legacy fields to Labels if Labels is nil
	// This provides backward compatibility while supporting the new structure
	if c.Labels == nil {
		c.Labels = &LabelsConfig{}
		// Migrate legacy fields to Labels section if present
		if c.BU != "" {
			c.Labels.BU = c.BU
		}
		if c.Team != "" {
			c.Labels.Team = c.Team
		}
		if c.PrimaryOwner != "" {
			c.Labels.PrimaryOwner = c.PrimaryOwner
		}
		if c.SecondaryOwner != "" {
			c.Labels.SecondaryOwner = c.SecondaryOwner
		}
		if c.PriorityV2 != "" {
			c.Labels.PriorityV2 = c.PriorityV2
		}
		if len(c.CustomLabels) > 0 {
			c.Labels.CustomLabels = c.CustomLabels
		}
	}

	// Apply defaults for labels if not provided
	// Use "ml" as fallback when bu/team are absent (maintains backward compatibility)
	if c.Labels.BU == "" {
		c.Labels.BU = "ml" // Fallback BU maintains backward compatibility
	}
	if c.Labels.Team == "" {
		c.Labels.Team = "ml" // Fallback team maintains backward compatibility
	}
	if c.Labels.PriorityV2 == "" {
		c.Labels.PriorityV2 = "p2" // Default priority (lower than p0/p1)
	}
	if c.Labels.CustomLabels == nil {
		c.Labels.CustomLabels = make(map[string]string)
	}

	// Probe configuration defaults (matches RingMaster defaults)
	if c.LivenessFailureThreshold == "" {
		c.LivenessFailureThreshold = "5"
	}
	if c.LivenessPeriodSeconds == "" {
		c.LivenessPeriodSeconds = "10"
	}
	if c.LivenessSuccessThreshold == "" {
		c.LivenessSuccessThreshold = "1"
	}
	if c.LivenessTimeoutSeconds == "" {
		c.LivenessTimeoutSeconds = "2"
	}
	if c.ReadinessFailureThreshold == "" {
		c.ReadinessFailureThreshold = "5"
	}
	if c.ReadinessPeriodSeconds == "" {
		c.ReadinessPeriodSeconds = "10"
	}
	if c.ReadinessSuccessThreshold == "" {
		c.ReadinessSuccessThreshold = "1"
	}
	if c.ReadinessTimeoutSeconds == "" {
		c.ReadinessTimeoutSeconds = "2"
	}

	// NodeSelector default is set in ToWorkflowPayload based on environment
	// We don't set it here because it depends on workingEnv

	// Autoscaling defaults
	if c.ASEnabled == "" {
		c.ASEnabled = "true"
	}
	if c.ASPoll == "" {
		c.ASPoll = "30"
	}
	if c.ASDownPeriod == "" {
		c.ASDownPeriod = "300"
	}
	if c.ASUpPeriod == "" {
		c.ASUpPeriod = "60"
	}
	if c.ASUpStableWindow == "" {
		c.ASUpStableWindow = "300"
	}
	if c.ASDownStableWindow == "" {
		c.ASDownStableWindow = "1800"
	}
	if c.ASTriggerType == "" {
		c.ASTriggerType = "AverageValue"
	}
	if c.ASTriggerMetric == "" {
		c.ASTriggerMetric = "cpu"
	}
	if c.ASTriggerValue == "" {
		c.ASTriggerValue = "50"
	}
	if c.ASDownPodCount == "" {
		c.ASDownPodCount = "2"
	}
	if c.ASUpPodCount == "" {
		c.ASUpPodCount = "2"
	}
	if c.ASUpPodPercentage == "" {
		c.ASUpPodPercentage = "10"
	}
	if c.CPUThreshold == "" {
		c.CPUThreshold = "50"
	}

	// Deployment defaults
	if c.MaxSurge == "" {
		c.MaxSurge = "50"
	}
	if c.TerminationGracePeriodSeconds == "" {
		c.TerminationGracePeriodSeconds = "300"
	}
	if c.ContourResponseTimeout == "" {
		c.ContourResponseTimeout = "false"
	}
	if c.PodDistributionSkew == "" {
		c.PodDistributionSkew = "false"
	}
	if c.EnableWebsocket == "" {
		c.EnableWebsocket = "false"
	}
	if c.AddHeadless == "" {
		c.AddHeadless = "false"
	}
	if c.CreateContourGateway == "" {
		c.CreateContourGateway = "false"
	}
	if c.PDBMinAvailable == "" {
		c.PDBMinAvailable = ""
	}
	if c.PDBMaxUnavailable == "" {
		c.PDBMaxUnavailable = "10%"
	}
}

// ToWorkflowPayload converts ServiceConfig to workflow payload format
// This maintains backward compatibility with existing workflow activities
func (c *ServiceConfig) ToWorkflowPayload(appName, workingEnv string) map[string]interface{} {
	payload := map[string]interface{}{
		"appName":       appName,
		"repoName":      c.RepoName,
		"branchName":    GetBranchName(workingEnv),
		"healthCheck":   c.HealthCheck,
		"appPort":       c.AppPort,
		"ingress_class": c.IngressClass,
		"appType":       c.AppType,
	}

	// Add labels (from Labels section if present, otherwise from legacy fields)
	// Labels section takes precedence for vendor-agnostic support
	var bu, team, primaryOwner, secondaryOwner, priorityV2 string
	if c.Labels != nil {
		bu = c.Labels.BU
		team = c.Labels.Team
		primaryOwner = c.Labels.PrimaryOwner
		secondaryOwner = c.Labels.SecondaryOwner
		priorityV2 = c.Labels.PriorityV2
	} else {
		// Fallback to legacy fields for backward compatibility
		bu = c.BU
		team = c.Team
		primaryOwner = c.PrimaryOwner
		secondaryOwner = c.SecondaryOwner
		priorityV2 = c.PriorityV2
	}

	// Add to payload (use defaults from ApplyDefaults if empty)
	if bu != "" {
		payload["bu"] = bu
	}
	if team != "" {
		payload["team"] = team
	}
	if primaryOwner != "" {
		payload["primaryOwner"] = primaryOwner
	}
	if secondaryOwner != "" {
		payload["secondaryOwner"] = secondaryOwner
	}
	if priorityV2 != "" {
		payload["priorityV2"] = priorityV2
	}

	// Add service type flags (convert map[string]bool to map[string]string for compatibility)
	if len(c.ServiceType) > 0 {
		for k, v := range c.ServiceType {
			payload[fmt.Sprintf("service_type_%s", k)] = v
		}
	}

	// Add custom labels
	for k, v := range c.CustomLabels {
		payload[k] = v
	}

	// Add probe configuration
	payload["liveness_failure_threshold"] = c.LivenessFailureThreshold
	payload["liveness_period_seconds"] = c.LivenessPeriodSeconds
	payload["liveness_success_threshold"] = c.LivenessSuccessThreshold
	payload["liveness_timeout_seconds"] = c.LivenessTimeoutSeconds
	payload["readiness_failure_threshold"] = c.ReadinessFailureThreshold
	payload["readiness_period_seconds"] = c.ReadinessPeriodSeconds
	payload["readiness_success_threshold"] = c.ReadinessSuccessThreshold
	payload["readiness_timeout_seconds"] = c.ReadinessTimeoutSeconds

	// Add autoscaling configuration
	payload["as_enabled"] = c.ASEnabled
	payload["as_poll"] = c.ASPoll
	payload["as_down_period"] = c.ASDownPeriod
	payload["as_up_period"] = c.ASUpPeriod
	payload["as_up_stable_window"] = c.ASUpStableWindow
	payload["as_down_stable_window"] = c.ASDownStableWindow
	payload["as_trigger_type"] = c.ASTriggerType
	payload["as_trigger_metric"] = c.ASTriggerMetric
	payload["as_trigger_value"] = c.ASTriggerValue
	payload["as_down_pod_count"] = c.ASDownPodCount
	payload["as_up_pod_count"] = c.ASUpPodCount
	payload["as_up_pod_percentage"] = c.ASUpPodPercentage
	payload["cpuThreshold"] = c.CPUThreshold

	// Add deployment configuration
	payload["maxSurge"] = c.MaxSurge
	payload["terminationGracePeriodSeconds"] = c.TerminationGracePeriodSeconds
	payload["contourResponseTimeout"] = c.ContourResponseTimeout
	payload["podDistributionSkew"] = c.PodDistributionSkew
	payload["enableWebsocket"] = c.EnableWebsocket
	payload["addHeadless"] = c.AddHeadless
	payload["createContourGateway"] = c.CreateContourGateway
	payload["pdbMinAvailable"] = c.PDBMinAvailable
	payload["pdbMaxUnavailable"] = c.PDBMaxUnavailable

	// Add nodeSelector (use from config if specified, otherwise set based on environment)
	if c.NodeSelector != "" {
		payload["nodeSelector"] = c.NodeSelector
	} else {
		// Set default based on environment (matches RingMaster logic)
		// Use GetEnvConfig to handle unknown environments gracefully
		envConfig := github.GetEnvConfig(workingEnv)
		configEnv := envConfig["config_env"]
		if configEnv == "int" {
			payload["nodeSelector"] = "cloud.google.com/compute-class"
		} else {
			payload["nodeSelector"] = "dedicated"
		}
	}

	return payload
}

// ToServiceConfigJSON converts ServiceConfig to JSON format for database storage
// This is used for backward compatibility when service_config table is still used
func (c *ServiceConfig) ToServiceConfigJSON() (json.RawMessage, error) {
	// Convert service_type map[string]bool to map[string]string for JSON compatibility
	serviceTypeStr := make(map[string]string)
	for k, v := range c.ServiceType {
		if v {
			serviceTypeStr[k] = "true"
		} else {
			serviceTypeStr[k] = "false"
		}
	}

	configMap := map[string]interface{}{
		"service_type_web":           c.ServiceType["web"],
		"service_type_grpc":          c.ServiceType["grpc"],
		"service_type_cache":         c.ServiceType["cache"],
		"service_type_worker":        c.ServiceType["worker"],
		"service_type_consumer":      c.ServiceType["consumer"],
		"service_type_database":      c.ServiceType["database"],
		"service_type_producer":      c.ServiceType["producer"],
		"service_type_scheduler":     c.ServiceType["scheduler"],
		"service_type_websocket":     c.ServiceType["websocket"],
		"service_type_httpstateless": c.ServiceType["httpstateless"],
	}

	return json.Marshal(configMap)
}
