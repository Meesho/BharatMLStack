package activities

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	"github.com/rs/zerolog/log"
)

// UpdateCPUThresholdActivity updates CPU threshold in GitHub
// This activity is used in the CPU threshold update workflow
func UpdateCPUThresholdActivity(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	cpuThreshold := getString(payload, "cpuThreshold")
	email := getString(payload, "email")
	if email == "" {
		email = "horizon-system"
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("cpuThreshold", cpuThreshold).
		Str("email", email).
		Msg("UpdateCPUThresholdActivity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	if err := infraHandler.UpdateCPUThreshold(appName, cpuThreshold, email, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("cpuThreshold", cpuThreshold).
			Msg("UpdateCPUThresholdActivity: Failed to update CPU threshold")
		return fmt.Errorf("failed to update CPU threshold: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("cpuThreshold", cpuThreshold).
		Msg("UpdateCPUThresholdActivity completed successfully")

	return nil
}

// UpdateGPUThresholdActivity updates GPU threshold in GitHub
// This activity is used in the GPU threshold update workflow
func UpdateGPUThresholdActivity(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	gpuThreshold := getString(payload, "gpuThreshold")
	email := getString(payload, "email")
	if email == "" {
		email = "horizon-system"
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("gpuThreshold", gpuThreshold).
		Str("email", email).
		Msg("UpdateGPUThresholdActivity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	if err := infraHandler.UpdateGPUThreshold(appName, gpuThreshold, email, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("gpuThreshold", gpuThreshold).
			Msg("UpdateGPUThresholdActivity: Failed to update GPU threshold")
		return fmt.Errorf("failed to update GPU threshold: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("gpuThreshold", gpuThreshold).
		Msg("UpdateGPUThresholdActivity completed successfully")

	return nil
}

// UpdateSharedMemoryActivity updates shared memory size in GitHub
// This activity is used in the shared memory update workflow
func UpdateSharedMemoryActivity(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	size := getString(payload, "size")
	email := getString(payload, "email")
	if email == "" {
		email = "horizon-system"
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("size", size).
		Str("email", email).
		Msg("UpdateSharedMemoryActivity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	if err := infraHandler.UpdateSharedMemory(appName, size, email, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("size", size).
			Msg("UpdateSharedMemoryActivity: Failed to update shared memory")
		return fmt.Errorf("failed to update shared memory: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("size", size).
		Msg("UpdateSharedMemoryActivity completed successfully")

	return nil
}

// UpdatePodAnnotationsActivity updates pod annotations in GitHub
// This activity is used in the pod annotations update workflow
func UpdatePodAnnotationsActivity(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	email := getString(payload, "email")
	if email == "" {
		email = "horizon-system"
	}

	// Extract annotations from payload
	annotationsInterface, ok := payload["annotations"]
	if !ok {
		return fmt.Errorf("annotations are required in payload")
	}

	annotations, ok := annotationsInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("annotations must be a map[string]interface{}")
	}

	// Convert to map[string]string
	annotationsMap := make(map[string]string)
	for k, v := range annotations {
		if strVal, ok := v.(string); ok {
			annotationsMap[k] = strVal
		} else {
			return fmt.Errorf("annotation value must be a string, got %T for key %s", v, k)
		}
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Int("annotationCount", len(annotationsMap)).
		Str("email", email).
		Msg("UpdatePodAnnotationsActivity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	if err := infraHandler.UpdatePodAnnotations(appName, annotationsMap, email, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Msg("UpdatePodAnnotationsActivity: Failed to update pod annotations")
		return fmt.Errorf("failed to update pod annotations: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Int("annotationCount", len(annotationsMap)).
		Msg("UpdatePodAnnotationsActivity completed successfully")

	return nil
}

// RestartDeploymentActivity restarts a deployment
// This activity is used in the restart deployment workflow
func RestartDeploymentActivity(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	isCanaryStr := getString(payload, "isCanary")
	isCanary := isCanaryStr == "true" || isCanaryStr == "True" || isCanaryStr == "1"

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Bool("isCanary", isCanary).
		Msg("RestartDeploymentActivity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	if err := infraHandler.RestartDeployment(appName, workingEnv, isCanary); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Bool("isCanary", isCanary).
			Msg("RestartDeploymentActivity: Failed to restart deployment")
		return fmt.Errorf("failed to restart deployment: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Bool("isCanary", isCanary).
		Msg("RestartDeploymentActivity completed successfully")

	return nil
}

// UpdateAutoscalingTriggersActivity updates autoscaling triggers in GitHub
// This activity is used in the autoscaling triggers update workflow
func UpdateAutoscalingTriggersActivity(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	email := getString(payload, "email")
	if email == "" {
		email = "horizon-system"
	}

	// Extract triggers from payload
	triggersInterface, ok := payload["triggers"]
	if !ok {
		return fmt.Errorf("triggers not found in payload")
	}

	triggers, ok := triggersInterface.([]interface{})
	if !ok {
		return fmt.Errorf("triggers must be an array")
	}

	if len(triggers) == 0 {
		return fmt.Errorf("triggers array cannot be empty")
	}

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Int("triggersCount", len(triggers)).
		Str("email", email).
		Msg("UpdateAutoscalingTriggersActivity started")

	// Initialize infrastructure handler
	infraHandler := handler.NewInfrastructureHandler()

	if err := infraHandler.UpdateAutoscalingTriggers(appName, triggers, email, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Int("triggersCount", len(triggers)).
			Msg("UpdateAutoscalingTriggersActivity: Failed to update autoscaling triggers")
		return fmt.Errorf("failed to update autoscaling triggers: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Int("triggersCount", len(triggers)).
		Msg("UpdateAutoscalingTriggersActivity completed successfully")

	return nil
}

