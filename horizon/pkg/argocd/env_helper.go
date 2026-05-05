package argocd

import (
	"strings"
)

// ExtractEnvironmentSuffix extracts the environment suffix from workingEnv
// Used for ArgoCD application name construction (e.g., "stg-test-app")
// Examples:
//   - "gcp_stg" -> "stg"
//   - "gcp_int" -> "int"
//   - "gcp_prd" -> "prd"
//   - "stg" -> "stg"
//   - "int" -> "int"
//   - "prd" -> "prd"
func ExtractEnvironmentSuffix(workingEnv string) string {
	// If workingEnv contains underscore, extract the last part
	if strings.Contains(workingEnv, "_") {
		parts := strings.Split(workingEnv, "_")
		return parts[len(parts)-1] // Get last part (e.g., "stg" from "gcp_stg")
	}
	// If no underscore, return as-is
	return workingEnv
}

// GetArgoCDConfigKey returns the ArgoCD configuration key for the environment
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Examples:
//   - workingEnv="gcp_stg" -> "GCP_STG_ARGOCD_API"
//   - workingEnv="gcp_int" -> "GCP_INT_ARGOCD_API"
//   - workingEnv="stg" -> "STG_ARGOCD_API"
func GetArgoCDConfigKey(workingEnv string, configType string) string {
	// Use workingEnv exactly as provided, just uppercase it
	// No parsing or transformation - use it as-is
	workingEnvUpper := strings.ToUpper(workingEnv)
	return workingEnvUpper + "_" + configType
}
