package handler

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// GetSupportedEnvironments returns the whitelist of supported environments
// Reads from environment variable SUPPORTED_ENVIRONMENTS (comma-separated)
// Example: SUPPORTED_ENVIRONMENTS="gcp_stg,gcp_int,gcp_prd,stg,int,prd"
// If not set, returns empty slice (no environments allowed)
func GetSupportedEnvironments() []string {
	envsStr := viper.GetString("SUPPORTED_ENVIRONMENTS")
	if envsStr == "" {
		log.Warn().Msg("SUPPORTED_ENVIRONMENTS not configured - no environments will be allowed for onboarding")
		return []string{}
	}

	// Split by comma and trim whitespace
	envs := strings.Split(envsStr, ",")
	supportedEnvs := make([]string, 0, len(envs))
	for _, env := range envs {
		trimmed := strings.TrimSpace(env)
		if trimmed != "" {
			supportedEnvs = append(supportedEnvs, trimmed)
		}
	}

	log.Info().
		Strs("supportedEnvironments", supportedEnvs).
		Msg("Loaded supported environments whitelist")

	return supportedEnvs
}

// ValidateEnvironmentWhitelist checks if the given environment is in the whitelist
// Returns error if environment is not supported
func ValidateEnvironmentWhitelist(workingEnv string) error {
	if workingEnv == "" {
		return fmt.Errorf("workingEnv cannot be empty")
	}

	supportedEnvs := GetSupportedEnvironments()
	if len(supportedEnvs) == 0 {
		return fmt.Errorf("no supported environments configured - set SUPPORTED_ENVIRONMENTS environment variable")
	}

	for _, supportedEnv := range supportedEnvs {
		if supportedEnv == workingEnv {
			return nil // Environment is whitelisted
		}
	}

	return fmt.Errorf("environment '%s' is not in the supported environments whitelist. Supported environments: %v", workingEnv, supportedEnvs)
}

// ValidateEnvironmentConfig checks if config.yaml exists for the given environment and service
// Returns error if config is missing
func ValidateEnvironmentConfig(serviceName, workingEnv string, serviceConfigLoader configs.ServiceConfigLoader) error {
	if serviceConfigLoader == nil {
		return fmt.Errorf("service config loader is not available - cannot validate config.yaml existence")
	}

	// Try to load the config - if it fails, the config doesn't exist
	_, err := serviceConfigLoader.LoadServiceConfig(serviceName, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("serviceName", serviceName).
			Str("workingEnv", workingEnv).
			Msg("Environment config validation failed - config.yaml not found")
		return fmt.Errorf("config.yaml not found for environment '%s' and service '%s' (expected: horizon/configs/services/%s/%s/config.yaml): %w", workingEnv, serviceName, serviceName, workingEnv, err)
	}

	log.Info().
		Str("serviceName", serviceName).
		Str("workingEnv", workingEnv).
		Msg("Environment config validation passed - config.yaml exists")

	return nil
}

// ValidateMultiEnvironmentOnboarding performs atomic validation for multi-environment onboarding
// Validates:
// 1. All environments are in the whitelist
// 2. All environments have config.yaml files
// Returns error if any validation fails (atomic behavior - all must pass)
func ValidateMultiEnvironmentOnboarding(serviceName string, environments []EnvironmentConfig, serviceConfigLoader configs.ServiceConfigLoader) error {
	if len(environments) == 0 {
		return fmt.Errorf("environments array cannot be empty")
	}

	log.Info().
		Str("serviceName", serviceName).
		Int("environmentCount", len(environments)).
		Msg("Validating multi-environment onboarding (atomic validation)")

	var validationErrors []string

	// Collect all unique environments for validation
	envSet := make(map[string]bool)
	for _, envConfig := range environments {
		workingEnv := envConfig.WorkingEnv
		if workingEnv == "" {
			validationErrors = append(validationErrors, "environment has empty workingEnv")
			continue
		}

		// Check for duplicates
		if envSet[workingEnv] {
			validationErrors = append(validationErrors, "duplicate environment '"+workingEnv+"' in request")
			continue
		}
		envSet[workingEnv] = true

		// Validate whitelist
		if err := ValidateEnvironmentWhitelist(workingEnv); err != nil {
			validationErrors = append(validationErrors, "environment '"+workingEnv+"': "+err.Error())
			continue
		}

		// Validate config exists
		if err := ValidateEnvironmentConfig(serviceName, workingEnv, serviceConfigLoader); err != nil {
			validationErrors = append(validationErrors, "environment '"+workingEnv+"': "+err.Error())
			continue
		}

		log.Info().
			Str("serviceName", serviceName).
			Str("workingEnv", workingEnv).
			Msg("Environment validation passed")
	}

	// If any validation failed, return combined error (atomic behavior)
	if len(validationErrors) > 0 {
		errorMsg := fmt.Sprintf("multi-environment onboarding validation failed for %d environment(s): %s", len(validationErrors), strings.Join(validationErrors, "; "))
		log.Error().
			Str("serviceName", serviceName).
			Int("failedEnvironments", len(validationErrors)).
			Strs("errors", validationErrors).
			Msg("Multi-environment onboarding validation failed - blocking all environments (atomic behavior)")
		return fmt.Errorf("onboarding blocked: %s", errorMsg)
	}

	log.Info().
		Str("serviceName", serviceName).
		Int("validatedEnvironments", len(environments)).
		Msg("Multi-environment onboarding validation passed - all environments are whitelisted and have config files")

	return nil
}
