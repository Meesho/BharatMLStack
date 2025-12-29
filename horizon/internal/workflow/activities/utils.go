package activities

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
)

// GetIngressClass returns the ingress class based on priority and working environment
// This is a simplified version - organizations may need to customize this logic
func GetIngressClass(ingressClass, priorityV2, workingEnv string) string {
	// For now, return the ingress class as-is
	// Organizations can customize this logic based on their requirements
	if ingressClass != "" {
		return ingressClass
	}
	// Default ingress class based on priority
	if priorityV2 == "p0" || priorityV2 == "p1" {
		return "contour"
	}
	return "nginx"
}

// GetValuePropertyPath returns the path for values_properties.yaml file using environment-based structure
// New structure: {workingEnv}/deployables/{appName}/values_properties.yaml
// Uses workingEnv as-is (no normalization) - environment-specific config comes from .env file
// This is vendor-agnostic and doesn't depend on bu/team
func GetValuePropertyPath(payload map[string]interface{}, workingEnv string) string {
	appName := getString(payload, "appName")
	if appName == "" {
		return ""
	}

	// Use new environment-based path structure
	basePath := github.GetBasePath("", "", workingEnv, appName)
	if basePath == "" {
		return ""
	}
	valuePropertyPath := fmt.Sprintf("%s/values_properties.yaml", basePath)
	return valuePropertyPath
}

// GetValuePropertyPathLegacy returns the legacy path for values_properties.yaml file (for backward compatibility)
func GetValuePropertyPathLegacy(payload map[string]interface{}, workingEnv string) string {
	bu := getString(payload, "bu")
	team := getString(payload, "team")
	appName := getString(payload, "appName")

	// Normalize BU and team
	buNorm := getString(payload, "bu_norm")
	if buNorm == "" {
		// Use BU as-is (no normalization)
		buNorm = bu
	}

	teamNorm := getString(payload, "team_norm")
	if teamNorm == "" {
		// Use Team as-is (no normalization)
		teamNorm = team
	}

	basePath := github.GetBasePathLegacy(buNorm, teamNorm, workingEnv, appName)
	if basePath == "" {
		return ""
	}
	valuePropertyPath := fmt.Sprintf("%s/values_properties.yaml", basePath)
	return valuePropertyPath
}

// IsServiceBeingOnboardedGrpc checks if the service is a gRPC service
func IsServiceBeingOnboardedGrpc(payload map[string]interface{}) bool {
	// Check service_type field first (new format: "grpc", "httpstateless", or comma-separated like "httpstateless,grpc")
	serviceType, ok := payload["service_type"]
	if ok {
		switch v := serviceType.(type) {
		case string:
			// Handle comma-separated values like "httpstateless,grpc"
			serviceTypes := strings.Split(strings.ReplaceAll(v, " ", ""), ",")
			for _, st := range serviceTypes {
				if strings.EqualFold(st, "grpc") {
					return true
				}
			}
		}
	}

	// Check service_type_grpc flag (backward compatibility)
	if grpcFlag, ok := payload["service_type_grpc"]; ok {
		switch v := grpcFlag.(type) {
		case bool:
			if v {
				return true
			}
		case string:
			if v == "true" || v == "True" || v == "TRUE" {
				return true
			}
		}
	}

	// Legacy: Handle array format (backward compatibility)
	serviceType, ok = payload["service_type"]
	if ok {
		switch v := serviceType.(type) {
		case []string:
			for _, st := range v {
				if st == "grpc" {
					return true
				}
			}
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok && str == "grpc" {
					return true
				}
			}
		}
	}

	return false
}

// EnableDeploymentBackwardCompatibility adds backward compatibility fields to payload
func EnableDeploymentBackwardCompatibility(payload map[string]interface{}) {
	// Add default values if not present
	if _, ok := payload["replica_count"]; !ok {
		payload["replica_count"] = 1
	}

	if _, ok := payload["asEnabled"]; !ok {
		payload["asEnabled"] = true
	}

	if _, ok := payload["appMetrics"]; !ok {
		payload["appMetrics"] = false
	}

	if _, ok := payload["initialDelaySeconds"]; !ok {
		payload["initialDelaySeconds"] = 30
	}
}

// EnableValuesBackwardCompatibility adds backward compatibility fields for values properties
func EnableValuesBackwardCompatibility(payload map[string]interface{}) {
	// Add default autoscaling values if not present
	if _, ok := payload["as_poll"]; !ok {
		payload["as_poll"] = 30
	}

	if _, ok := payload["as_down_period"]; !ok {
		payload["as_down_period"] = 300
	}

	if _, ok := payload["as_up_period"]; !ok {
		payload["as_up_period"] = 60
	}

	if _, ok := payload["as_up_stable_window"]; !ok {
		payload["as_up_stable_window"] = 300
	}

	if _, ok := payload["as_trigger_type"]; !ok {
		payload["as_trigger_type"] = "AverageValue"
	}

	if _, ok := payload["as_trigger_metric"]; !ok {
		payload["as_trigger_metric"] = "cpu"
	}

	if _, ok := payload["cpuThreshold"]; !ok {
		payload["cpuThreshold"] = "50"
	}
}

// GetBranchName returns the branch name based on working environment
// Uses BRANCH_NAME environment variable (vendor-agnostic)
func GetBranchName(workingEnv string) string {
	// Use configured branch from BRANCH_NAME env var (vendor-agnostic)
	branch := github.GetBranchForCommits()
	if branch != "" {
		return branch
	}

	// Default branch mapping
	if strings.HasPrefix(workingEnv, "gcp_") || workingEnv == "prd" {
		return "main"
	}
	if workingEnv == "int" || workingEnv == "gcp_int" || workingEnv == "gcp_stg" {
		return "develop"
	}
	return "develop"
}

// GetValuesTemplateName returns the template name based on triton_image_tag
// Horizon only supports triton deployables, so we use triton-values.tmpl
// triton-values.tmpl includes common-values.tmpl at the top
func GetValuesTemplateName(appType string) string {
	// Horizon only deals with triton/predator deployables
	// triton-values.tmpl includes {{ template "common-values.tmpl" . }} at the top
	// So it automatically compiles with common-values.tmpl
	return "triton-values.tmpl"
}

// GetValuesYamlPath returns the path for values.yaml file using environment-based structure
// New structure: {workingEnv}/deployables/{appName}/values.yaml
// Uses workingEnv as-is (no normalization) - environment-specific config comes from .env file
// This is vendor-agnostic and doesn't depend on bu/team
func GetValuesYamlPath(payload map[string]interface{}, workingEnv string) string {
	appName := getString(payload, "appName")
	if appName == "" {
		return ""
	}

	// Use new environment-based path structure
	basePath := github.GetBasePath("", "", workingEnv, appName)
	if basePath == "" {
		return ""
	}
	valuesYamlPath := fmt.Sprintf("%s/values.yaml", basePath)
	return valuesYamlPath
}

// GetValuesYamlPathLegacy returns the legacy path for values.yaml file (for backward compatibility)
func GetValuesYamlPathLegacy(payload map[string]interface{}, workingEnv string) string {
	bu := getString(payload, "bu")
	team := getString(payload, "team")
	appName := getString(payload, "appName")

	// Normalize BU and team
	buNorm := getString(payload, "bu_norm")
	if buNorm == "" {
		// Use BU as-is (no normalization)
		buNorm = bu
	}

	teamNorm := getString(payload, "team_norm")
	if teamNorm == "" {
		// Use Team as-is (no normalization)
		teamNorm = team
	}

	basePath := github.GetBasePathLegacy(buNorm, teamNorm, workingEnv, appName)
	if basePath == "" {
		return ""
	}
	valuesYamlPath := fmt.Sprintf("%s/values.yaml", basePath)
	return valuesYamlPath
}
