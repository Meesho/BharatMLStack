package handler

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/rs/zerolog/log"
)

// getArgoCDApplicationName constructs the ArgoCD application name from appName and workingEnv
// ArgoCD applications are named with environment prefix: {env}-{appName}
// Examples: "stg-test-app", "int-test-app", "prd-test-app"
func getArgoCDApplicationName(appName, workingEnv string) string {
	// Use GetArgocdApplicationNameFromEnv to construct the proper ArgoCD application name
	return argocd.GetArgocdApplicationNameFromEnv(appName, workingEnv)
}

type InfrastructureHandler interface {
	GetHPAProperties(appName, workingEnv string) (*HPAConfig, error)
	GetConfig(serviceName, workingEnv string) Config
	GetResourceDetail(appName, workingEnv string) (*ResourceDetail, error)
	RestartDeployment(appName, workingEnv string, isCanary bool) error
	UpdateCPUThreshold(appName, threshold, email, workingEnv string) error
	UpdateGPUThreshold(appName, threshold, email, workingEnv string) error
	UpdateSharedMemory(appName, size, email, workingEnv string) error
	UpdatePodAnnotations(appName string, annotations map[string]string, email, workingEnv string) error
	UpdateAutoscalingTriggers(appName string, triggers []interface{}, email, workingEnv string) error
}

// Config matches the RingMaster client Config struct for compatibility
type Config struct {
	MinReplica    string `json:"min_replica"`
	MaxReplica    string `json:"max_replica"`
	RunningStatus string `json:"running_status"`
}

type infrastructureHandler struct {
	// Can add dependencies here if needed (e.g., GitHub client for threshold updates)
}

type HPAConfig struct {
	MinReplica    string `json:"min_replica"`
	MaxReplica    string `json:"max_replica"`
	RunningStatus string `json:"running_status"`
}

type ResourceDetail struct {
	Nodes []Node `json:"nodes"`
}

type Node struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Health Health `json:"health"`
	Info   []Info `json:"info"`
}

type Health struct {
	Status string `json:"status"`
}

type Info struct {
	Value string `json:"value"`
}

func NewInfrastructureHandler() InfrastructureHandler {
	return &infrastructureHandler{}
}

func (h *infrastructureHandler) GetHPAProperties(appName, workingEnv string) (*HPAConfig, error) {
	log.Info().Str("appName", appName).Str("workingEnv", workingEnv).Msg("Getting HPA properties")

	// Construct ArgoCD application name with environment prefix
	// Standard: All APIs accept just appName, ArgoCD functions construct {workingEnv}-{appName}
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("GetHPAProperties: Constructed ArgoCD application name")

	// Call ArgoCD directly with full application name
	hpa, err := argocd.GetHPAProperties(argocdAppName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to get HPA properties")
		return nil, fmt.Errorf("failed to get HPA properties: %w", err)
	}

	// Get resource detail to determine running status
	resourceDetail, err := argocd.GetArgoCDResourceDetail(appName, workingEnv)
	if err != nil {
		log.Warn().Err(err).Str("appName", appName).Msg("Failed to get resource detail, setting running status to false")
		return &HPAConfig{
			MinReplica:    fmt.Sprintf("%d", hpa.Min),
			MaxReplica:    fmt.Sprintf("%d", hpa.Max),
			RunningStatus: "false",
		}, nil
	}

	// Check if there are healthy pods
	healthyPodCount := 0
	for _, node := range resourceDetail.Nodes {
		if node.Kind == "Pod" && node.Health.Status == "Healthy" {
			for _, info := range node.Info {
				if info.Value == "Running" {
					healthyPodCount++
				}
			}
		}
	}

	runningStatus := "false"
	if healthyPodCount > 0 {
		runningStatus = "true"
	}

	return &HPAConfig{
		MinReplica:    fmt.Sprintf("%d", hpa.Min),
		MaxReplica:    fmt.Sprintf("%d", hpa.Max),
		RunningStatus: runningStatus,
	}, nil
}

func (h *infrastructureHandler) GetConfig(serviceName, workingEnv string) Config {
	log.Info().Str("serviceName", serviceName).Str("workingEnv", workingEnv).Msg("Getting config")

	// Standard: Accept just appName (without env prefix) for consistency
	// If serviceName contains env prefix (e.g., "stg-appname"), extract just the appName
	// Otherwise, use serviceName as-is (it's already just the appName)
	appName := serviceName
	parts := strings.Split(serviceName, "-")
	if len(parts) > 1 {
		// Check if first part matches workingEnv (could be env-prefixed format)
		// For backward compatibility, try to extract appName if format matches env-appname
		// But prefer to use serviceName as-is if it doesn't start with workingEnv
		expectedPrefix := fmt.Sprintf("%s-", workingEnv)
		if strings.HasPrefix(serviceName, expectedPrefix) {
			// Extract appName by removing workingEnv prefix
			appName = strings.TrimPrefix(serviceName, expectedPrefix)
			log.Info().
				Str("serviceName", serviceName).
				Str("workingEnv", workingEnv).
				Str("extractedAppName", appName).
				Msg("GetConfig: Extracted appName from env-prefixed serviceName")
		}
		// If it doesn't match workingEnv prefix, use serviceName as-is (it's already just appName)
	}

	// Use GetHPAProperties to get the config
	hpaConfig, err := h.GetHPAProperties(appName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("serviceName", serviceName).Msg("Failed to get config, returning defaults")
		return Config{
			MinReplica:    "0",
			MaxReplica:    "0",
			RunningStatus: "false",
		}
	}

	return Config{
		MinReplica:    hpaConfig.MinReplica,
		MaxReplica:    hpaConfig.MaxReplica,
		RunningStatus: hpaConfig.RunningStatus,
	}
}

func (h *infrastructureHandler) GetResourceDetail(appName, workingEnv string) (*ResourceDetail, error) {
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("GetResourceDetail: Starting ArgoCD resource detail lookup")

	// Call ArgoCD directly
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("GetResourceDetail: Calling argocd.GetArgoCDResourceDetail")

	appResponse, err := argocd.GetArgoCDResourceDetail(appName, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("GetResourceDetail: Failed to get resource detail from ArgoCD - check ArgoCD connection and credentials")
		return nil, fmt.Errorf("failed to get resource detail from ArgoCD: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Int("nodeCount", len(appResponse.Nodes)).
		Msg("GetResourceDetail: Successfully retrieved resource detail from ArgoCD")

	// Transform to our model
	nodes := make([]Node, len(appResponse.Nodes))
	for i, node := range appResponse.Nodes {
		infoList := make([]Info, len(node.Info))
		for j, info := range node.Info {
			infoList[j] = Info{Value: info.Value}
		}
		nodes[i] = Node{
			Kind:   node.Kind,
			Name:   node.Name,
			Health: Health{Status: node.Health.Status},
			Info:   infoList,
		}
	}

	return &ResourceDetail{Nodes: nodes}, nil
}

func (h *infrastructureHandler) RestartDeployment(appName, workingEnv string, isCanary bool) error {
	log.Info().Str("appName", appName).Str("workingEnv", workingEnv).Bool("isCanary", isCanary).Msg("Restarting deployment")

	if appName == "" {
		return fmt.Errorf("appName is required")
	}

	// Construct ArgoCD application name with environment prefix
	// Standard: All APIs accept just appName, ArgoCD functions construct {workingEnv}-{appName}
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("RestartDeployment: Constructed ArgoCD application name")

	// Call ArgoCD directly with full application name
	err := argocd.RestartDeployment(argocdAppName, workingEnv, isCanary)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to restart deployment")
		return fmt.Errorf("failed to restart deployment: %w", err)
	}

	return nil
}

func (h *infrastructureHandler) UpdateCPUThreshold(appName, threshold, email, workingEnv string) error {
	log.Info().Str("appName", appName).Str("threshold", threshold).Str("workingEnv", workingEnv).Str("email", email).Msg("Updating CPU threshold")

	// Validate threshold
	cpuThreshold := github.AutoScalinGitHubCPUThreshold{CPUThreshold: threshold}
	if err := cpuThreshold.Validate(); err != nil {
		log.Error().Err(err).Str("threshold", threshold).Msg("Invalid CPU threshold")
		return fmt.Errorf("invalid CPU threshold: %w", err)
	}

	// Construct ArgoCD application name with environment prefix
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateCPUThreshold: Constructed ArgoCD application name")

	// Get ArgoCD application metadata
	argocdApp, err := argocd.GetArgoCDApplication(argocdAppName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to get ArgoCD application")
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Update threshold in GitHub
	// Use provided email or default to generic value
	commitAuthor := email
	if commitAuthor == "" {
		commitAuthor = "horizon-system"
	}
	err = github.UpdateThresholdHelmRepo(argocdApp.Metadata, cpuThreshold, commitAuthor, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to update CPU threshold in GitHub")
		return fmt.Errorf("failed to update CPU threshold: %w", err)
	}

	// Refresh ArgoCD application to pick up changes
	// Create a minimal ArgoCDObject for refresh
	argocdObj := argocd.ArgoCDObject{
		AppName: argocdAppName,
	}
	err = argocdObj.Refresh(workingEnv, "", "")
	if err != nil {
		log.Warn().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to refresh ArgoCD application, but threshold was updated in GitHub")
		// Don't fail the operation if refresh fails - the change is already in GitHub
	}

	log.Info().Str("appName", appName).Str("argocdAppName", argocdAppName).Str("threshold", threshold).Msg("Successfully updated CPU threshold")
	return nil
}

func (h *infrastructureHandler) UpdateGPUThreshold(appName, threshold, email, workingEnv string) error {
	log.Info().Str("appName", appName).Str("threshold", threshold).Str("workingEnv", workingEnv).Str("email", email).Msg("Updating GPU threshold")

	// Validate threshold
	gpuThreshold := github.AutoScalinGitHubGPUThreshold{GPUThreshold: threshold}
	if err := gpuThreshold.Validate(); err != nil {
		log.Error().Err(err).Str("threshold", threshold).Msg("Invalid GPU threshold")
		return fmt.Errorf("invalid GPU threshold: %w", err)
	}

	// Construct ArgoCD application name with environment prefix
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateGPUThreshold: Constructed ArgoCD application name")

	// Get ArgoCD application metadata
	argocdApp, err := argocd.GetArgoCDApplication(argocdAppName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to get ArgoCD application")
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Update threshold in GitHub
	// Use provided email or default to generic value
	commitAuthor := email
	if commitAuthor == "" {
		commitAuthor = "horizon-system"
	}
	err = github.UpdateThresholdHelmRepo(argocdApp.Metadata, gpuThreshold, commitAuthor, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to update GPU threshold in GitHub")
		return fmt.Errorf("failed to update GPU threshold: %w", err)
	}

	// Refresh ArgoCD application to pick up changes
	// Create a minimal ArgoCDObject for refresh
	argocdObj := argocd.ArgoCDObject{
		AppName: argocdAppName,
	}
	err = argocdObj.Refresh(workingEnv, "", "")
	if err != nil {
		log.Warn().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to refresh ArgoCD application, but threshold was updated in GitHub")
		// Don't fail the operation if refresh fails - the change is already in GitHub
	}

	log.Info().Str("appName", appName).Str("argocdAppName", argocdAppName).Str("threshold", threshold).Msg("Successfully updated GPU threshold")
	return nil
}

func (h *infrastructureHandler) UpdateSharedMemory(appName, size, email, workingEnv string) error {
	log.Info().Str("appName", appName).Str("size", size).Str("workingEnv", workingEnv).Str("email", email).Msg("Updating shared memory")

	// Validate size (should be a valid Kubernetes memory size, e.g., "1Gi", "2Gi")
	if size == "" {
		log.Error().Str("size", size).Msg("Invalid shared memory size")
		return fmt.Errorf("shared memory size is required")
	}

	// Construct ArgoCD application name with environment prefix
	// Standard: All APIs accept just appName, ArgoCD functions construct {workingEnv}-{appName}
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateSharedMemory: Constructed ArgoCD application name")

	// Get ArgoCD application metadata
	argocdApp, err := argocd.GetArgoCDApplication(argocdAppName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to get ArgoCD application")
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Get current values.yaml
	data, sha, err := github.GetFileYaml(argocdApp.Metadata, "values.yaml", workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve values.yaml")
		return fmt.Errorf("failed to retrieve values.yaml: %w", err)
	}

	// Update sharedMemory in values.yaml
	// Structure: sharedmemory: { size: "1Gi" }
	if data["sharedmemory"] == nil {
		data["sharedmemory"] = make(map[string]interface{})
	}
	sharedMemoryMap, ok := data["sharedmemory"].(map[string]interface{})
	if !ok {
		// If it's not a map, create a new one
		sharedMemoryMap = make(map[string]interface{})
		data["sharedmemory"] = sharedMemoryMap
	}
	sharedMemoryMap["size"] = size

	// Use provided email or default to generic value
	commitAuthor := email
	if commitAuthor == "" {
		commitAuthor = "horizon-system"
	}
	commitMsg := fmt.Sprintf("%s: Shared memory updated to %s by %s", argocdApp.Metadata.Name, size, commitAuthor)
	err = github.UpdateFileYaml(argocdApp.Metadata, data, "values.yaml", commitMsg, sha, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to update shared memory in GitHub")
		return fmt.Errorf("failed to update shared memory: %w", err)
	}

	// Refresh ArgoCD application to pick up changes
	argocdObj := argocd.ArgoCDObject{
		AppName: argocdAppName,
	}
	err = argocdObj.Refresh(workingEnv, "", "")
	if err != nil {
		log.Warn().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to refresh ArgoCD application, but shared memory was updated in GitHub")
		// Don't fail the operation if refresh fails - the change is already in GitHub
	}

	log.Info().Str("appName", appName).Str("argocdAppName", argocdAppName).Str("size", size).Msg("Successfully updated shared memory")
	return nil
}

func (h *infrastructureHandler) UpdatePodAnnotations(appName string, annotations map[string]string, email, workingEnv string) error {
	log.Info().
		Str("appName", appName).
		Int("annotationCount", len(annotations)).
		Str("workingEnv", workingEnv).
		Str("email", email).
		Msg("Updating pod annotations")

	// Validate annotations
	if len(annotations) == 0 {
		log.Error().Msg("Invalid pod annotations - annotations map is empty")
		return fmt.Errorf("pod annotations are required")
	}

	// Construct ArgoCD application name with environment prefix
	// Standard: All APIs accept just appName, ArgoCD functions construct {workingEnv}-{appName}
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("UpdatePodAnnotations: Constructed ArgoCD application name")

	// Get ArgoCD application metadata
	argocdApp, err := argocd.GetArgoCDApplication(argocdAppName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to get ArgoCD application")
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Get current values.yaml
	data, sha, err := github.GetFileYaml(argocdApp.Metadata, "values.yaml", workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve values.yaml")
		return fmt.Errorf("failed to retrieve values.yaml: %w", err)
	}

	// Update podAnnotations in values.yaml
	// Structure: podAnnotations: { key1: "value1", key2: "value2" }
	// Note: The YAML file uses camelCase "podAnnotations" to match the template structure
	// The template variable is .pod_annotations (snake_case), but the YAML key is podAnnotations (camelCase)
	// Convert annotations map[string]string to map[string]interface{} for YAML compatibility
	podAnnotationsMap := make(map[string]interface{})
	for k, v := range annotations {
		podAnnotationsMap[k] = v
	}

	// Check if podAnnotations already exists in values.yaml (camelCase)
	if existingPodAnnotations, ok := data["podAnnotations"].(map[string]interface{}); ok {
		// Merge with existing annotations (new annotations override existing ones)
		for k, v := range podAnnotationsMap {
			existingPodAnnotations[k] = v
		}
		data["podAnnotations"] = existingPodAnnotations
		log.Info().
			Str("appName", appName).
			Int("existingAnnotationCount", len(existingPodAnnotations)).
			Int("newAnnotationCount", len(annotations)).
			Msg("Merged new annotations with existing podAnnotations")
	} else {
		// Create new podAnnotations map
		data["podAnnotations"] = podAnnotationsMap
		log.Info().
			Str("appName", appName).
			Int("annotationCount", len(annotations)).
			Msg("Created new podAnnotations in values.yaml")
	}

	// Use provided email or default to generic value
	commitAuthor := email
	if commitAuthor == "" {
		commitAuthor = "horizon-system"
	}
	commitMsg := fmt.Sprintf("%s: Pod annotations updated by %s", argocdApp.Metadata.Name, commitAuthor)
	err = github.UpdateFileYaml(argocdApp.Metadata, data, "values.yaml", commitMsg, sha, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to update pod annotations in GitHub")
		return fmt.Errorf("failed to update pod annotations: %w", err)
	}

	// Refresh ArgoCD application to pick up changes
	argocdObj := argocd.ArgoCDObject{
		AppName: argocdAppName,
	}
	err = argocdObj.Refresh(workingEnv, "", "")
	if err != nil {
		log.Warn().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to refresh ArgoCD application, but pod annotations were updated in GitHub")
		// Don't fail the operation if refresh fails - the change is already in GitHub
	}

	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Int("annotationCount", len(annotations)).
		Msg("Successfully updated pod annotations")
	return nil
}

func (h *infrastructureHandler) UpdateAutoscalingTriggers(appName string, triggers []interface{}, email, workingEnv string) error {
	log.Info().
		Str("appName", appName).
		Int("triggersCount", len(triggers)).
		Str("workingEnv", workingEnv).
		Str("email", email).
		Msg("Updating autoscaling triggers")

	if appName == "" {
		return fmt.Errorf("appName is required")
	}

	if len(triggers) == 0 {
		return fmt.Errorf("triggers array cannot be empty")
	}

	// Construct ArgoCD application name with environment prefix
	argocdAppName := getArgoCDApplicationName(appName, workingEnv)
	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Str("workingEnv", workingEnv).
		Msg("UpdateAutoscalingTriggers: Constructed ArgoCD application name")

	// Get ArgoCD application metadata
	argocdApp, err := argocd.GetArgoCDApplication(argocdAppName, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to get ArgoCD application")
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Update triggers in GitHub
	commitAuthor := email
	if commitAuthor == "" {
		commitAuthor = "horizon-system"
	}
	err = github.UpdateAutoscalingTriggers(argocdApp.Metadata, triggers, commitAuthor, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("appName", appName).Msg("Failed to update autoscaling triggers in GitHub")
		return fmt.Errorf("failed to update autoscaling triggers: %w", err)
	}

	// Refresh ArgoCD application to pick up changes
	argocdObj := argocd.ArgoCDObject{
		AppName: argocdAppName,
	}
	err = argocdObj.Refresh(workingEnv, "", "")
	if err != nil {
		log.Warn().Err(err).Str("appName", appName).Str("argocdAppName", argocdAppName).Msg("Failed to refresh ArgoCD application, but triggers were updated in GitHub")
		// Don't fail the operation if refresh fails - the change is already in GitHub
	}

	log.Info().
		Str("appName", appName).
		Str("argocdAppName", argocdAppName).
		Int("triggersCount", len(triggers)).
		Msg("Successfully updated autoscaling triggers")
	return nil
}
