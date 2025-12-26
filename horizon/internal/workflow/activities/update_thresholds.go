package activities

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	"github.com/rs/zerolog/log"
)

// UpdateThresholds updates CPU and/or GPU thresholds in GitHub
// This activity is used in the threshold update workflow
func UpdateThresholds(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	cpuThreshold := getString(payload, "cpu_threshold")
	gpuThreshold := getString(payload, "gpu_threshold")
	machineType := getString(payload, "machine_type")
	email := getString(payload, "created_by")
	if email == "" {
		email = "horizon-system"
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("machineType", machineType).
		Str("cpuThreshold", cpuThreshold).
		Str("gpuThreshold", gpuThreshold).
		Str("email", email).
		Msg("UpdateThresholds activity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	// Update CPU threshold if provided
	if cpuThreshold != "" {
		if err := infraHandler.UpdateCPUThreshold(appName, cpuThreshold, email, workingEnv); err != nil {
			log.Error().
				Err(err).
				Str("appName", appName).
				Str("cpuThreshold", cpuThreshold).
				Msg("UpdateThresholds: Failed to update CPU threshold")
			return fmt.Errorf("failed to update CPU threshold: %w", err)
		}
		log.Info().
			Str("appName", appName).
			Str("cpuThreshold", cpuThreshold).
			Msg("UpdateThresholds: CPU threshold updated successfully")
	}

	// Update GPU threshold if provided and machine type is GPU
	// Check if machine type is GPU (case-insensitive for consistency with other parts of the codebase)
	machineTypeUpper := strings.ToUpper(strings.TrimSpace(machineType))
	if gpuThreshold != "" && machineTypeUpper == "GPU" {
		if err := infraHandler.UpdateGPUThreshold(appName, gpuThreshold, email, workingEnv); err != nil {
			log.Error().
				Err(err).
				Str("appName", appName).
				Str("gpuThreshold", gpuThreshold).
				Msg("UpdateThresholds: Failed to update GPU threshold")
			return fmt.Errorf("failed to update GPU threshold: %w", err)
		}
		log.Info().
			Str("appName", appName).
			Str("gpuThreshold", gpuThreshold).
			Msg("UpdateThresholds: GPU threshold updated successfully")
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("UpdateThresholds activity completed successfully")

	return nil
}

