package github

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// GetBasePath returns the base path for helm chart files using environment-based directory structure
// New structure: {workingEnv}/deployables/{appName}/values.yaml
// Uses workingEnv as-is (no normalization) - environment-specific config comes from .env file
// This replaces the old bu/team-based structure for vendor-agnostic support
func GetBasePath(bu string, team string, workingEnv string, applicationName string) string {
	// Use workingEnv as-is (no normalization)
	// Environment-specific configuration should be provided via .env file
	if workingEnv == "" {
		log.Warn().Msg("GetBasePath: workingEnv is empty")
		return ""
	}

	// Validate application name
	if applicationName == "" {
		log.Warn().Msg("GetBasePath: Application name is empty")
		return ""
	}

	// New structure: {workingEnv}/deployables/{appName}
	// Files are grouped under deployables/ directory per environment
	basePath := fmt.Sprintf("%s/deployables/%s", workingEnv, applicationName)
	return basePath
}

// GetArgoApplicationPath returns the path for ArgoCD application YAML using environment-based structure
// New structure: {workingEnv}/applications/{appName}.yaml
// Uses workingEnv as-is (no normalization) - environment-specific config comes from .env file
// This replaces the old bu/team/cluster-based structure for vendor-agnostic support
// ArgoCD filename constraints: Keep total path length reasonable (Kubernetes resource names have 63 char limit)
func GetArgoApplicationPath(workingEnv string, applicationName string) string {
	// Use workingEnv as-is (no normalization)
	// Environment-specific configuration should be provided via .env file
	if workingEnv == "" {
		log.Warn().Msg("GetArgoApplicationPath: workingEnv is empty")
		return ""
	}

	// Validate application name
	if applicationName == "" {
		log.Warn().Msg("GetArgoApplicationPath: Application name is empty")
		return ""
	}

	// New structure: {workingEnv}/applications/{appName}.yaml
	// This is vendor-agnostic and doesn't depend on bu/team/cluster
	argoPath := fmt.Sprintf("%s/applications/%s.yaml", workingEnv, applicationName)
	return argoPath
}

// GetArgoApplicationPathLegacy returns the legacy ArgoCD application path for backward compatibility
// Structure: applications_v2/k8s-{bu}-{env}-ase1/{team}-{appName}.yaml
// This is kept for migration purposes but uses vendor-agnostic logic
func GetArgoApplicationPathLegacy(bu string, team string, workingEnv string, applicationName string) string {
	// Use BU and Team as-is (no normalization)
	buNorm := bu
	teamNorm := team

	// Get cluster_env from GetEnvConfig (vendor-agnostic)
	envConfig := GetEnvConfig(workingEnv)
	clusterEnv := envConfig["cluster_env"]
	if clusterEnv == "" {
		// Fallback: use workingEnv as cluster_env
		clusterEnv = workingEnv
	}

	// Ensure no empty segments
	if buNorm == "" || teamNorm == "" || applicationName == "" {
		return ""
	}

	argoPath := fmt.Sprintf("applications_v2/k8s-%s-%s-ase1/%s-%s.yaml",
		buNorm, clusterEnv, teamNorm, applicationName)
	return argoPath
}

// GetBasePathLegacy returns the legacy base path for backward compatibility
// Structure: {helm_path}/{bu}/{team}/{appName}
// This is kept for migration purposes but uses vendor-agnostic logic
func GetBasePathLegacy(bu string, team string, workingEnv string, applicationName string) string {
	// Use BU and Team as-is (no normalization for vendor-agnostic support)
	buShort := bu
	teamShort := team

	// Get helm_path from GetEnvConfig (vendor-agnostic)
	envConfig := GetEnvConfig(workingEnv)
	helmPath := envConfig["helm_path"]
	if helmPath == "" {
		helmPath = "values_v2" // Default fallback
	}

	// Ensure no empty segments in path to avoid malformed paths like "values_v2///app"
	if buShort == "" || teamShort == "" || applicationName == "" {
		return ""
	}

	basePath := fmt.Sprintf("%s/%s/%s/%s", helmPath, buShort, teamShort, applicationName)
	return basePath
}
