package argocd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// GetArgoCDNamespace returns the ArgoCD namespace for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_NAMESPACE
// Falls back to ARGOCD_NAMESPACE if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_NAMESPACE
func GetArgoCDNamespace(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_NAMESPACE")
	namespace := viper.GetString(envSpecificKey)

	if namespace != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("namespace", namespace).
			Msg("Using environment-specific ArgoCD namespace")
		return namespace
	}

	// Fallback to default ArgoCD namespace if environment-specific not set
	namespace = viper.GetString("ARGOCD_NAMESPACE")
	if namespace != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("namespace", namespace).
			Msg("Environment-specific ArgoCD namespace not found, using default ARGOCD_NAMESPACE")
	} else {
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Neither environment-specific nor default ArgoCD namespace is configured")
	}
	return namespace
}

// GetArgoCDDestinationName returns the ArgoCD destination cluster name pattern for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_DESTINATION_NAME
// Falls back to ARGOCD_DESTINATION_NAME if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_DESTINATION_NAME
// The value can contain template variables like {bu_norm}, {bu}, {team} which will be processed
func GetArgoCDDestinationName(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_DESTINATION_NAME")
	destinationName := viper.GetString(envSpecificKey)

	if destinationName != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("destinationName", destinationName).
			Msg("Using environment-specific ArgoCD destination name")
		return destinationName
	}

	// Fallback to default ArgoCD destination name if environment-specific not set
	destinationName = viper.GetString("ARGOCD_DESTINATION_NAME")
	if destinationName != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("destinationName", destinationName).
			Msg("Environment-specific ArgoCD destination name not found, using default ARGOCD_DESTINATION_NAME")
	} else {
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Neither environment-specific nor default ArgoCD destination name is configured")
	}
	return destinationName
}

// GetArgoCDDestinationNamespace returns the ArgoCD destination namespace pattern for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_DESTINATION_NAMESPACE
// Falls back to ARGOCD_DESTINATION_NAMESPACE if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_DESTINATION_NAMESPACE
// The value can contain template variables like {env}, {appName}, {bu_norm}, {bu}, {team} which will be processed
func GetArgoCDDestinationNamespace(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_DESTINATION_NAMESPACE")
	destinationNamespace := viper.GetString(envSpecificKey)

	if destinationNamespace != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("destinationNamespace", destinationNamespace).
			Msg("Using environment-specific ArgoCD destination namespace")
		return destinationNamespace
	}

	// Fallback to default ArgoCD destination namespace if environment-specific not set
	destinationNamespace = viper.GetString("ARGOCD_DESTINATION_NAMESPACE")
	if destinationNamespace != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("destinationNamespace", destinationNamespace).
			Msg("Environment-specific ArgoCD destination namespace not found, using default ARGOCD_DESTINATION_NAMESPACE")
	} else {
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Neither environment-specific nor default ArgoCD destination namespace is configured")
	}
	return destinationNamespace
}

// GetArgoCDProject returns the ArgoCD project name for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_PROJECT
// Falls back to ARGOCD_PROJECT if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_PROJECT
// This is a hardcoded value (not a pattern) - no template variable processing
func GetArgoCDProject(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_PROJECT")
	project := viper.GetString(envSpecificKey)

	if project != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("project", project).
			Msg("Using environment-specific ArgoCD project")
		return project
	}

	// Fallback to default ArgoCD project if environment-specific not set
	project = viper.GetString("ARGOCD_PROJECT")
	if project != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("project", project).
			Msg("Environment-specific ArgoCD project not found, using default ARGOCD_PROJECT")
	} else {
		// Default fallback - use a sensible default
		project = "default"
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("project", project).
			Msg("Neither environment-specific nor default ArgoCD project found, using default: default")
	}
	return project
}

// GetArgoCDHelmChartPath returns the ArgoCD helm chart path for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_HELMCHART_PATH
// Falls back to ARGOCD_HELMCHART_PATH if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_HELMCHART_PATH
// Returns empty string if not configured (caller should use default "values_v2")
func GetArgoCDHelmChartPath(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_HELMCHART_PATH")
	helmChartPath := viper.GetString(envSpecificKey)

	if helmChartPath != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("helmChartPath", helmChartPath).
			Msg("Using environment-specific ArgoCD helm chart path")
		return helmChartPath
	}

	// Fallback to default ArgoCD helm chart path if environment-specific not set
	helmChartPath = viper.GetString("ARGOCD_HELMCHART_PATH")
	if helmChartPath != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("helmChartPath", helmChartPath).
			Msg("Environment-specific ArgoCD helm chart path not found, using default ARGOCD_HELMCHART_PATH")
	}
	// Return empty string if not configured - caller will use default "values_v2"
	return helmChartPath
}

// GetArgoCDSourceRepoURL returns the ArgoCD source repository URL pattern for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_SOURCE_REPO_URL
// Falls back to ARGOCD_SOURCE_REPO_URL if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_SOURCE_REPO_URL
// The value can contain template variables like {owner}, {repo} which will be processed
func GetArgoCDSourceRepoURL(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_SOURCE_REPO_URL")
	repoURL := viper.GetString(envSpecificKey)

	if repoURL != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("repoURL", repoURL).
			Msg("Using environment-specific ArgoCD source repo URL")
		return repoURL
	}

	// Fallback to default ArgoCD source repo URL if environment-specific not set
	repoURL = viper.GetString("ARGOCD_SOURCE_REPO_URL")
	if repoURL != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("repoURL", repoURL).
			Msg("Environment-specific ArgoCD source repo URL not found, using default ARGOCD_SOURCE_REPO_URL")
	} else {
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Neither environment-specific nor default ArgoCD source repo URL is configured")
	}
	return repoURL
}

// GetArgoCDSourceTargetRevision returns the ArgoCD source target revision (branch/tag) for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_SOURCE_TARGET_REVISION
// Falls back to ARGOCD_SOURCE_TARGET_REVISION if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_SOURCE_TARGET_REVISION
func GetArgoCDSourceTargetRevision(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_SOURCE_TARGET_REVISION")
	targetRevision := viper.GetString(envSpecificKey)

	if targetRevision != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("targetRevision", targetRevision).
			Msg("Using environment-specific ArgoCD source target revision")
		return targetRevision
	}

	// Fallback to default ArgoCD source target revision if environment-specific not set
	targetRevision = viper.GetString("ARGOCD_SOURCE_TARGET_REVISION")
	if targetRevision != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("targetRevision", targetRevision).
			Msg("Environment-specific ArgoCD source target revision not found, using default ARGOCD_SOURCE_TARGET_REVISION")
	} else {
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Neither environment-specific nor default ArgoCD source target revision is configured")
	}
	return targetRevision
}

// GetArgoCDSyncPolicyOptions returns the ArgoCD sync policy options for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Pattern: {WORKING_ENV}_ARGOCD_SYNC_POLICY_OPTIONS
// Falls back to ARGOCD_SYNC_POLICY_OPTIONS if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_SYNC_POLICY_OPTIONS
// Returns comma-separated string that should be split into array
func GetArgoCDSyncPolicyOptions(workingEnv string) string {
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_SYNC_POLICY_OPTIONS")
	syncOptions := viper.GetString(envSpecificKey)

	if syncOptions != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("syncOptions", syncOptions).
			Msg("Using environment-specific ArgoCD sync policy options")
		return syncOptions
	}

	// Fallback to default ArgoCD sync policy options if environment-specific not set
	syncOptions = viper.GetString("ARGOCD_SYNC_POLICY_OPTIONS")
	if syncOptions != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("syncOptions", syncOptions).
			Msg("Environment-specific ArgoCD sync policy options not found, using default ARGOCD_SYNC_POLICY_OPTIONS")
	} else {
		// Default fallback
		syncOptions = "CreateNamespace=true"
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Str("syncOptions", syncOptions).
			Msg("Neither environment-specific nor default ArgoCD sync policy options found, using default: CreateNamespace=true")
	}
	return syncOptions
}

// ProcessArgoCDTemplateVars processes template variables in ArgoCD configuration values
// Replaces {bu_norm}, {bu}, {team}, {owner}, {repo} with actual values
func ProcessArgoCDTemplateVars(template string, buNorm, buNormalized, teamNormalized, githubOwner, repo string) string {
	result := strings.ReplaceAll(template, "{bu_norm}", buNorm)
	result = strings.ReplaceAll(result, "{bu}", buNormalized)
	result = strings.ReplaceAll(result, "{team}", teamNormalized)
	result = strings.ReplaceAll(result, "{owner}", githubOwner)
	result = strings.ReplaceAll(result, "{repo}", repo)
	return result
}

// ProcessArgoCDDestinationNamespaceTemplateVars processes template variables in destination namespace
// Replaces {env}, {appName}, {bu_norm}, {bu}, {team} with actual values
func ProcessArgoCDDestinationNamespaceTemplateVars(template string, env, appName, buNorm, buNormalized, teamNormalized string) string {
	result := strings.ReplaceAll(template, "{env}", env)
	result = strings.ReplaceAll(result, "{appName}", appName)
	result = strings.ReplaceAll(result, "{bu_norm}", buNorm)
	result = strings.ReplaceAll(result, "{bu}", buNormalized)
	result = strings.ReplaceAll(result, "{team}", teamNormalized)
	return result
}
