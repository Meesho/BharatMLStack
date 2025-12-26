package handler

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	"github.com/rs/zerolog/log"
)

// WorkflowHandler implements the Handler interface (same pattern as numerix Numerix struct)
type WorkflowHandler struct {
	orchestrator *workflow.Orchestrator
	worker       *workflow.Worker
}

// StartOnboardingWorkflow starts a new onboarding workflow
func (w *WorkflowHandler) StartOnboardingWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	appName := getStringFromPayload(payload, "appName")
	log.Info().
		Str("appName", appName).
		Str("createdBy", createdBy).
		Str("workingEnv", workingEnv).
		Int("payloadKeys", len(payload)).
		Msg("WorkflowHandler: Starting onboarding workflow creation")

	// Create workflow in etcd
	workflowID, err := w.orchestrator.StartOnboardingWorkflow(payload, createdBy, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("createdBy", createdBy).
			Str("workingEnv", workingEnv).
			Msg("WorkflowHandler: Failed to create workflow in etcd via orchestrator")
		return "", fmt.Errorf("failed to create workflow in etcd: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("WorkflowHandler: Workflow created in etcd successfully, attempting to enqueue")

	// Enqueue workflow for async execution
	if err := w.worker.Enqueue(workflowID); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("WorkflowHandler: Failed to enqueue workflow to worker pool - workflow exists in etcd but may not execute")
		// Workflow is already created in etcd, so we return the ID even if enqueue fails
		// The workflow can be manually retried later
		return workflowID, nil
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("createdBy", createdBy).
		Msg("WorkflowHandler: Onboarding workflow started and enqueued successfully")

	return workflowID, nil
}

// StartThresholdUpdateWorkflow starts a new threshold update workflow
func (w *WorkflowHandler) StartThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	appName := getStringFromPayload(payload, "appName")
	log.Info().
		Str("appName", appName).
		Str("createdBy", createdBy).
		Str("workingEnv", workingEnv).
		Int("payloadKeys", len(payload)).
		Msg("WorkflowHandler: Starting threshold update workflow creation")

	// Create workflow in etcd
	workflowID, err := w.orchestrator.StartThresholdUpdateWorkflow(payload, createdBy, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("createdBy", createdBy).
			Str("workingEnv", workingEnv).
			Msg("WorkflowHandler: Failed to create threshold update workflow in etcd via orchestrator")
		return "", fmt.Errorf("failed to create workflow in etcd: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("WorkflowHandler: Threshold update workflow created in etcd successfully, attempting to enqueue")

	// Enqueue workflow for async execution
	if err := w.worker.Enqueue(workflowID); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("WorkflowHandler: Failed to enqueue threshold update workflow to worker pool - workflow exists in etcd but may not execute")
		// Workflow is already created in etcd, so we return the ID even if enqueue fails
		// The workflow can be manually retried later
		return workflowID, nil
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("createdBy", createdBy).
		Msg("WorkflowHandler: Threshold update workflow started and enqueued successfully")

	return workflowID, nil
}

// StartCPUThresholdUpdateWorkflow starts a new CPU threshold update workflow
func (w *WorkflowHandler) StartCPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	return w.startInfrastructureWorkflow("CPU threshold update", w.orchestrator.StartCPUThresholdUpdateWorkflow, payload, createdBy, workingEnv)
}

// StartGPUThresholdUpdateWorkflow starts a new GPU threshold update workflow
func (w *WorkflowHandler) StartGPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	return w.startInfrastructureWorkflow("GPU threshold update", w.orchestrator.StartGPUThresholdUpdateWorkflow, payload, createdBy, workingEnv)
}

// StartSharedMemoryUpdateWorkflow starts a new shared memory update workflow
func (w *WorkflowHandler) StartSharedMemoryUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	return w.startInfrastructureWorkflow("shared memory update", w.orchestrator.StartSharedMemoryUpdateWorkflow, payload, createdBy, workingEnv)
}

// StartPodAnnotationsUpdateWorkflow starts a new pod annotations update workflow
func (w *WorkflowHandler) StartPodAnnotationsUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	return w.startInfrastructureWorkflow("pod annotations update", w.orchestrator.StartPodAnnotationsUpdateWorkflow, payload, createdBy, workingEnv)
}

// StartRestartDeploymentWorkflow starts a new restart deployment workflow
func (w *WorkflowHandler) StartRestartDeploymentWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	return w.startInfrastructureWorkflow("restart deployment", w.orchestrator.StartRestartDeploymentWorkflow, payload, createdBy, workingEnv)
}

// StartAutoscalingTriggersUpdateWorkflow starts a new autoscaling triggers update workflow
func (w *WorkflowHandler) StartAutoscalingTriggersUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	return w.startInfrastructureWorkflow("autoscaling triggers update", w.orchestrator.StartAutoscalingTriggersUpdateWorkflow, payload, createdBy, workingEnv)
}

// startInfrastructureWorkflow is a helper function to start infrastructure workflows
func (w *WorkflowHandler) startInfrastructureWorkflow(operationName string, startFn func(map[string]interface{}, string, string) (string, error), payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	appName := getStringFromPayload(payload, "appName")
	log.Info().
		Str("appName", appName).
		Str("createdBy", createdBy).
		Str("workingEnv", workingEnv).
		Str("operation", operationName).
		Msg("WorkflowHandler: Starting infrastructure workflow creation")

	// Create workflow in etcd
	workflowID, err := startFn(payload, createdBy, workingEnv)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("operation", operationName).
			Msg("WorkflowHandler: Failed to create infrastructure workflow in etcd via orchestrator")
		return "", fmt.Errorf("failed to create workflow in etcd: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("operation", operationName).
		Msg("WorkflowHandler: Infrastructure workflow created in etcd successfully, attempting to enqueue")

	// Enqueue workflow for async execution
	if err := w.worker.Enqueue(workflowID); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", appName).
			Str("operation", operationName).
			Msg("WorkflowHandler: Failed to enqueue infrastructure workflow to worker pool - workflow exists in etcd but may not execute")
		// Workflow is already created in etcd, so we return the ID even if enqueue fails
		return workflowID, nil
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("createdBy", createdBy).
		Str("operation", operationName).
		Msg("WorkflowHandler: Infrastructure workflow started and enqueued successfully")

	return workflowID, nil
}

// GetWorkflowStatus retrieves workflow status from etcd
func (w *WorkflowHandler) GetWorkflowStatus(workflowID string) (map[string]interface{}, error) {
	return w.orchestrator.GetWorkflowStatus(workflowID)
}

// Helper function
func getStringFromPayload(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
