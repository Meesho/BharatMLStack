package argocd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// ValidateArgoCDConfiguration checks if ArgoCD API and Token are configured for the given workingEnv
// This is OPTIONAL - returns nil if configuration exists, nil if not (does not fail)
// The system will use default ARGOCD_API/ARGOCD_TOKEN if environment-specific config is not found
// This should be called before creating deployables to log warnings if ArgoCD is not configured
func ValidateArgoCDConfiguration(workingEnv string) error {
	log.Info().
		Str("workingEnv", workingEnv).
		Msg("ValidateArgoCDConfiguration: Checking ArgoCD configuration for environment")

	// Use workingEnv exactly as provided (just uppercased) - no transformation
	// Examples: "gcp_stg" -> "GCP_STG_ARGOCD_API", "stg" -> "STG_ARGOCD_API"
	apiKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_API")
	tokenKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_TOKEN")

	log.Info().
		Str("workingEnv", workingEnv).
		Str("apiKey", apiKey).
		Str("tokenKey", tokenKey).
		Msg("ValidateArgoCDConfiguration: Checking ArgoCD configuration keys")

	// Check for environment-specific ArgoCD API
	api := viper.GetString(apiKey)
	if api == "" {
		// Fallback to default ArgoCD API
		api = viper.GetString("ARGOCD_API")
		if api == "" {
			log.Warn().
				Str("workingEnv", workingEnv).
				Str("apiKey", apiKey).
				Msg("ValidateArgoCDConfiguration: ArgoCD API not configured (optional - will use default if available)")
			// Return nil - this is optional, not mandatory
			return nil
		}
		log.Info().
			Str("workingEnv", workingEnv).
			Str("apiKey", apiKey).
			Msg("ValidateArgoCDConfiguration: Environment-specific ArgoCD API not found, using default ARGOCD_API")
	}

	// Check for environment-specific ArgoCD Token
	token := viper.GetString(tokenKey)
	if token == "" {
		// Fallback to default ArgoCD Token
		token = viper.GetString("ARGOCD_TOKEN")
		if token == "" {
			log.Warn().
				Str("workingEnv", workingEnv).
				Str("tokenKey", tokenKey).
				Msg("ValidateArgoCDConfiguration: ArgoCD Token not configured (optional - will use default if available)")
			// Return nil - this is optional, not mandatory
			return nil
		}
		log.Info().
			Str("workingEnv", workingEnv).
			Str("tokenKey", tokenKey).
			Msg("ValidateArgoCDConfiguration: Environment-specific ArgoCD Token not found, using default ARGOCD_TOKEN")
	}

	log.Info().
		Str("workingEnv", workingEnv).
		Str("apiKey", apiKey).
		Str("tokenKey", tokenKey).
		Str("apiConfigured", "true").
		Str("tokenConfigured", "true").
		Msg("ValidateArgoCDConfiguration: ArgoCD configuration found")

	return nil
}

