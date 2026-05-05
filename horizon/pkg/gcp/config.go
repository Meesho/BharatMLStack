package gcp

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// GetNamespace returns the Kubernetes namespace for an application
// Format: {env}-{appName}
// Example: workingEnv="gcp_stg", appName="test-app" -> namespace="stg-test-app"
func GetNamespace(appName, workingEnv string) string {
	// Get environment config to extract config_env
	envConfig := github.GetEnvConfig(workingEnv)
	configEnv := envConfig["config_env"]
	if configEnv == "" {
		// Fallback: extract from workingEnv if config_env not found
		if strings.Contains(workingEnv, "_") {
			parts := strings.Split(workingEnv, "_")
			configEnv = parts[len(parts)-1] // Get last part (e.g., "stg" from "gcp_stg")
		} else {
			configEnv = workingEnv
		}
	}

	namespace := fmt.Sprintf("%s-%s", configEnv, appName)
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("configEnv", configEnv).
		Str("namespace", namespace).
		Msg("GetNamespace: Constructed Kubernetes namespace")

	return namespace
}

// GetGCPProjectID returns the GCP project ID for a given working environment
// Reads from environment variable: {WORKING_ENV}_GCP_PROJECT_ID or GCP_PROJECT_ID
// Example: GCP_STG_GCP_PROJECT_ID or GCP_PROJECT_ID
func GetGCPProjectID(workingEnv string) string {
	// Try environment-specific first: {WORKING_ENV}_GCP_PROJECT_ID
	workingEnvUpper := strings.ToUpper(workingEnv)
	envSpecificKey := workingEnvUpper + "_GCP_PROJECT_ID"
	projectID := viper.GetString(envSpecificKey)

	if projectID != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("envKey", envSpecificKey).
			Str("projectID", projectID).
			Msg("GetGCPProjectID: Found environment-specific GCP project ID")
		return projectID
	}

	// Fallback to generic GCP_PROJECT_ID
	projectID = viper.GetString("GCP_PROJECT_ID")
	if projectID != "" {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("projectID", projectID).
			Msg("GetGCPProjectID: Using generic GCP_PROJECT_ID")
		return projectID
	}

	log.Warn().
		Str("workingEnv", workingEnv).
		Msg("GetGCPProjectID: GCP project ID not configured - workload identity binding will fail")
	return ""
}

// GetServiceAccountFromPayload extracts service account from payload
// Service account is provided during onboarding and should be in the payload
// Returns empty string if not found in payload
func GetServiceAccountFromPayload(payload map[string]interface{}, workingEnv string) string {
	// Get service account from payload (it's already provided during onboarding)
	if serviceAccount, ok := payload["serviceAccount"].(string); ok && serviceAccount != "" {
		log.Info().
			Str("serviceAccount", serviceAccount).
			Str("workingEnv", workingEnv).
			Msg("GetServiceAccountFromPayload: Found service account in payload")
		return serviceAccount
	}

	log.Warn().
		Str("workingEnv", workingEnv).
		Msg("GetServiceAccountFromPayload: Service account not found in payload")
	return ""
}

