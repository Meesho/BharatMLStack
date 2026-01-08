package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/workflow/activities"
	"github.com/rs/zerolog/log"
)

// Orchestrator manages workflow execution using etcd for state management
type Orchestrator struct {
	etcdRepo EtcdRepository
}

// EtcdRepository interface for workflow storage (avoids import cycle)
type EtcdRepository interface {
	CreateWorkflow(workflowID string, workflowData map[string]interface{}) error
	UpdateWorkflow(workflowID string, workflowData map[string]interface{}) error
	GetWorkflow(workflowID string) (map[string]interface{}, error)
}

// NewOrchestrator creates a new workflow orchestrator
// etcdRepo is passed in to avoid import cycle (same pattern as numerix)
func NewOrchestrator(etcdRepo EtcdRepository) *Orchestrator {
	return &Orchestrator{
		etcdRepo: etcdRepo,
	}
}

// StartOnboardingWorkflow creates a new onboarding workflow in etcd and returns workflow ID
// This is non-blocking - the actual execution happens in a worker
func (o *Orchestrator) StartOnboardingWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateWorkflowID()

	now := time.Now().Format(time.RFC3339)

	// Build workflow steps dynamically based on payload
	// IamBinding is only added if serviceAccount is provided (similar to RingMaster's triton_image_tag check)
	steps := []map[string]interface{}{
		{"name": "CreateValuesYaml", "status": "PENDING", "error": ""},
		{"name": "CreateApplicationYaml", "status": "PENDING", "error": ""},
		{"name": "UpdateValuesProperties", "status": "PENDING", "error": ""},
		{"name": "CreateCloudDNS", "status": "PENDING", "error": ""},
		{"name": "CreateCoreDNS", "status": "PENDING", "error": ""},
	}

	// Add IamBinding step if serviceAccount is provided in payload
	// This matches RingMaster's behavior where IamBinding is only executed when triton_image_tag is present
	// In Horizon, we check for serviceAccount instead
	if serviceAccount, ok := payload["serviceAccount"].(string); ok && serviceAccount != "" {
		steps = append(steps, map[string]interface{}{
			"name":   "IamBinding",
			"status": "PENDING",
			"error":  "",
		})
		log.Info().
			Str("workflowID", workflowID).
			Str("serviceAccount", serviceAccount).
			Msg("Orchestrator: Service account found in payload - adding IamBinding step to workflow")
	}

	workflowData := map[string]interface{}{
		"id":          workflowID,
		"state":       "PENDING",
		"payload":     payload,
		"steps":       steps,
		"currentStep": 0,
		"retryCount":  0,
		"createdBy":   createdBy,
		"workingEnv":  workingEnv,
		"createdAt":   now,
		"updatedAt":   now,
	}

	// Create workflow in etcd (same pattern as numerix)
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", getStringFromPayload(payload, "appName")).
		Str("workingEnv", workingEnv).
		Str("createdBy", createdBy).
		Int("stepsCount", len(steps)).
		Msg("Orchestrator: Creating workflow entry in etcd")

	if err := o.etcdRepo.CreateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", getStringFromPayload(payload, "appName")).
			Str("workingEnv", workingEnv).
			Str("createdBy", createdBy).
			Msg("Orchestrator: Failed to create workflow in etcd - etcd connection or write operation failed")
		return "", fmt.Errorf("failed to create workflow in etcd: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", getStringFromPayload(payload, "appName")).
		Str("workingEnv", workingEnv).
		Msg("Orchestrator: Workflow created in etcd successfully")

	return workflowID, nil
}

// StartThresholdUpdateWorkflow creates a new threshold update workflow in etcd and returns workflow ID
// This is non-blocking - the actual execution happens in a worker
func (o *Orchestrator) StartThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateThresholdUpdateWorkflowID()

	now := time.Now().Format(time.RFC3339)

	// Define workflow steps for threshold update
	steps := []map[string]interface{}{
		{"name": "UpdateThresholds", "status": "PENDING", "error": ""},
	}

	workflowData := map[string]interface{}{
		"id":          workflowID,
		"state":       "PENDING",
		"payload":     payload,
		"steps":       steps,
		"currentStep": 0,
		"retryCount":  0,
		"createdBy":   createdBy,
		"workingEnv":  workingEnv,
		"createdAt":   now,
		"updatedAt":   now,
	}

	// Create workflow in etcd
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", getStringFromPayload(payload, "appName")).
		Str("workingEnv", workingEnv).
		Str("createdBy", createdBy).
		Int("stepsCount", len(steps)).
		Msg("Orchestrator: Creating threshold update workflow entry in etcd")

	if err := o.etcdRepo.CreateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", getStringFromPayload(payload, "appName")).
			Str("workingEnv", workingEnv).
			Str("createdBy", createdBy).
			Msg("Orchestrator: Failed to create threshold update workflow in etcd")
		return "", fmt.Errorf("failed to create workflow in etcd: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", getStringFromPayload(payload, "appName")).
		Str("workingEnv", workingEnv).
		Msg("Orchestrator: Threshold update workflow created in etcd successfully")

	return workflowID, nil
}

// ExecuteThresholdUpdateWorkflow executes a threshold update workflow by ID (called by worker)
func (o *Orchestrator) ExecuteThresholdUpdateWorkflow(ctx context.Context, workflowID string) error {
	log.Info().
		Str("workflowID", workflowID).
		Msg("Orchestrator: Starting threshold update workflow execution - retrieving workflow from etcd")

	// Get workflow from etcd
	workflowData, err := o.etcdRepo.GetWorkflow(workflowID)
	if err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Msg("Orchestrator: Failed to retrieve threshold update workflow from etcd")
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	appName := getStringFromPayload(workflowData["payload"].(map[string]interface{}), "appName")
	workingEnv := workflowData["workingEnv"].(string)
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("currentState", workflowData["state"].(string)).
		Msg("Orchestrator: Threshold update workflow retrieved from etcd successfully")

	// Update state to RUNNING
	workflowData["state"] = "RUNNING"
	workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
	if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Msg("Orchestrator: Failed to update threshold update workflow state to RUNNING")
		return fmt.Errorf("failed to update workflow state: %w", err)
	}

	payload := workflowData["payload"].(map[string]interface{})
	steps := workflowData["steps"].([]interface{})

	// Execute threshold update activity
	activity := struct {
		name string
		fn   func(map[string]interface{}, string) error
	}{
		"UpdateThresholds", activities.UpdateThresholds,
	}

	// Update current step
	workflowData["currentStep"] = 0
	stepMap := steps[0].(map[string]interface{})
	stepMap["status"] = "RUNNING"
	stepMap["startedAt"] = time.Now().Format(time.RFC3339)
	workflowData["updatedAt"] = time.Now().Format(time.RFC3339)

	if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
		log.Error().Err(err).Str("workflowID", workflowID).Str("step", activity.name).Msg("Failed to update workflow step")
	}

	// Execute activity with retry logic
	maxRetries := 3
	var activityErr error
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			log.Info().
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Int("retryAttempt", retry).
				Msg("Orchestrator: Retrying threshold update activity execution")
		}

		activityErr = activity.fn(payload, workingEnv)
		if activityErr == nil {
			log.Info().
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Msg("Orchestrator: Threshold update activity executed successfully")
			break
		}

		if retry < maxRetries-1 {
			backoffDuration := time.Duration(retry+1) * 5 * time.Second
			log.Warn().
				Err(activityErr).
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Int("retryAttempt", retry).
				Dur("backoff", backoffDuration).
				Msg("Orchestrator: Threshold update activity failed, retrying after backoff")
			time.Sleep(backoffDuration)
		}
	}

	// Update step status
	if activityErr != nil {
		stepMap["status"] = "FAILED"
		stepMap["error"] = activityErr.Error()
		stepMap["completedAt"] = time.Now().Format(time.RFC3339)
		workflowData["state"] = "FAILED"
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
		if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
			log.Error().Err(err).Str("workflowID", workflowID).Msg("Failed to update workflow with failure status")
		}
		return fmt.Errorf("threshold update activity failed: %w", activityErr)
	}

	stepMap["status"] = "COMPLETED"
	stepMap["completedAt"] = time.Now().Format(time.RFC3339)
	workflowData["state"] = "COMPLETED"
	workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
	if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
		log.Error().Err(err).Str("workflowID", workflowID).Msg("Failed to update workflow with completion status")
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Msg("Orchestrator: Threshold update workflow completed successfully")

	return nil
}

// ExecuteWorkflow executes a workflow by ID (called by worker)
func (o *Orchestrator) ExecuteWorkflow(ctx context.Context, workflowID string) error {
	log.Info().
		Str("workflowID", workflowID).
		Msg("Orchestrator: Starting workflow execution - retrieving workflow from etcd")

	// Get workflow from etcd
	workflowData, err := o.etcdRepo.GetWorkflow(workflowID)
	if err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Msg("Orchestrator: Failed to retrieve workflow from etcd - workflow may not exist or etcd connection failed")
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	appName := getStringFromPayload(workflowData["payload"].(map[string]interface{}), "appName")
	workingEnv := workflowData["workingEnv"].(string)
	steps := workflowData["steps"].([]interface{})
	
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("currentState", workflowData["state"].(string)).
		Int("stepsCount", len(steps)).
		Msg("Orchestrator: Workflow retrieved from etcd successfully")

	// Determine workflow type based on steps
	// Infrastructure workflows have a single step with specific names
	// Onboarding workflows have multiple steps: "CreateValuesYaml", "CreateApplicationYaml", "UpdateValuesProperties"
	if len(steps) == 1 {
		stepMap := steps[0].(map[string]interface{})
		if stepName, ok := stepMap["name"].(string); ok {
			switch stepName {
			case "UpdateThresholds":
				// This is a threshold update workflow
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected threshold update workflow, routing to ExecuteThresholdUpdateWorkflow")
				return o.ExecuteThresholdUpdateWorkflow(ctx, workflowID)
			case "UpdateCPUThreshold":
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected CPU threshold update workflow")
				return o.ExecuteInfrastructureWorkflow(ctx, workflowID, "UpdateCPUThreshold", activities.UpdateCPUThresholdActivity)
			case "UpdateGPUThreshold":
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected GPU threshold update workflow")
				return o.ExecuteInfrastructureWorkflow(ctx, workflowID, "UpdateGPUThreshold", activities.UpdateGPUThresholdActivity)
			case "UpdateSharedMemory":
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected shared memory update workflow")
				return o.ExecuteInfrastructureWorkflow(ctx, workflowID, "UpdateSharedMemory", activities.UpdateSharedMemoryActivity)
			case "UpdatePodAnnotations":
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected pod annotations update workflow")
				return o.ExecuteInfrastructureWorkflow(ctx, workflowID, "UpdatePodAnnotations", activities.UpdatePodAnnotationsActivity)
			case "RestartDeployment":
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected restart deployment workflow")
				return o.ExecuteInfrastructureWorkflow(ctx, workflowID, "RestartDeployment", activities.RestartDeploymentActivity)
			case "UpdateAutoscalingTriggers":
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Msg("Orchestrator: Detected autoscaling triggers update workflow")
				return o.ExecuteInfrastructureWorkflow(ctx, workflowID, "UpdateAutoscalingTriggers", activities.UpdateAutoscalingTriggersActivity)
			}
		}
	}

	// Default to onboarding workflow execution
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Msg("Orchestrator: Detected onboarding workflow, routing to standard ExecuteWorkflow")

	// Update state to RUNNING
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Msg("Orchestrator: Updating workflow state to RUNNING in etcd")

	workflowData["state"] = "RUNNING"
	workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
	if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", appName).
			Str("newState", "RUNNING").
			Msg("Orchestrator: Failed to update workflow state to RUNNING in etcd")
		return fmt.Errorf("failed to update workflow state: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Msg("Orchestrator: Workflow state updated to RUNNING successfully")

	payload := workflowData["payload"].(map[string]interface{})
	// steps already retrieved above for workflow type detection

	// Build activities list dynamically based on workflow steps
	// IamBinding is only executed if it's in the steps (i.e., serviceAccount was provided)
	activityList := []struct {
		name string
		fn   func(map[string]interface{}, string) error
	}{
		{"CreateValuesYaml", activities.CreateValuesYaml},
		{"CreateApplicationYaml", activities.CreateApplicationYaml},
		{"UpdateValuesProperties", activities.UpdateValuesProperties},
		{"CreateCloudDNS", activities.CreateCloudDNS},
		{"CreateCoreDNS", activities.CreateCoreDNS},
	}

	// Add IamBinding activity if it's in the steps
	for _, step := range steps {
		stepMap := step.(map[string]interface{})
		if stepName, ok := stepMap["name"].(string); ok && stepName == "IamBinding" {
			activityList = append(activityList, struct {
				name string
				fn   func(map[string]interface{}, string) error
			}{"IamBinding", activities.IamBinding})
			log.Info().
				Str("workflowID", workflowID).
				Str("appName", appName).
				Msg("Orchestrator: IamBinding step found - adding IamBinding activity to execution list")
			break
		}
	}

	for i, activity := range activityList {
		// Check context cancellation
		select {
		case <-ctx.Done():
			workflowData["state"] = "CANCELLED"
			workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
			o.etcdRepo.UpdateWorkflow(workflowID, workflowData)
			return ctx.Err()
		default:
		}

		// Update current step
		workflowData["currentStep"] = i
		stepMap := steps[i].(map[string]interface{})
		stepMap["status"] = "RUNNING"
		stepMap["startedAt"] = time.Now().Format(time.RFC3339)
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)

		if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
			log.Error().Err(err).Str("workflowID", workflowID).Str("step", activity.name).Msg("Failed to update workflow step")
		}

		// Execute activity with retry logic
		maxRetries := 3
		var activityErr error
		log.Info().
			Str("workflowID", workflowID).
			Str("appName", appName).
			Str("step", activity.name).
			Str("workingEnv", workingEnv).
			Int("stepNumber", i+1).
			Int("totalSteps", len(activityList)).
			Int("maxRetries", maxRetries).
			Msg("Orchestrator: Executing activity with retry logic")

		for retry := 0; retry < maxRetries; retry++ {
			if retry > 0 {
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Str("step", activity.name).
					Int("retryAttempt", retry).
					Int("maxRetries", maxRetries).
					Msg("Orchestrator: Retrying activity execution")
			}

			log.Info().
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Str("workingEnv", workingEnv).
				Int("retryAttempt", retry).
				Msg("Orchestrator: Calling activity function")

			activityErr = activity.fn(payload, workingEnv)
			if activityErr == nil {
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Str("step", activity.name).
					Int("retryAttempt", retry).
					Msg("Orchestrator: Activity executed successfully")
				break
			}

			if retry < maxRetries-1 {
				backoffDuration := time.Duration(retry+1) * 5 * time.Second
				log.Warn().
					Err(activityErr).
					Str("workflowID", workflowID).
					Str("appName", appName).
					Str("step", activity.name).
					Int("retryAttempt", retry+1).
					Int("maxRetries", maxRetries).
					Dur("backoffDuration", backoffDuration).
					Msg("Orchestrator: Activity failed, will retry after backoff")
				time.Sleep(backoffDuration) // Exponential backoff
			} else {
				log.Error().
					Err(activityErr).
					Str("workflowID", workflowID).
					Str("appName", appName).
					Str("step", activity.name).
					Int("retryAttempt", retry+1).
					Int("maxRetries", maxRetries).
					Msg("Orchestrator: Activity failed after all retry attempts exhausted")
			}
		}

		if activityErr != nil {
			// Activity failed after retries
			log.Error().
				Err(activityErr).
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Int("stepNumber", i+1).
				Int("totalSteps", len(activityList)).
				Int("retryCount", maxRetries).
				Msg("Orchestrator: Activity failed after all retries - updating workflow state to FAILED")

			stepMap["status"] = "FAILED"
			stepMap["error"] = activityErr.Error()
			stepMap["completedAt"] = time.Now().Format(time.RFC3339)
			workflowData["state"] = "FAILED"
			workflowData["error"] = fmt.Sprintf("%s failed: %v", activity.name, activityErr)
			workflowData["retryCount"] = maxRetries
			workflowData["updatedAt"] = time.Now().Format(time.RFC3339)

			if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
				log.Error().
					Err(err).
					Str("workflowID", workflowID).
					Str("appName", appName).
					Str("step", activity.name).
					Str("workflowState", "FAILED").
					Msg("Orchestrator: Failed to update workflow state to FAILED in etcd")
			} else {
				log.Info().
					Str("workflowID", workflowID).
					Str("appName", appName).
					Str("step", activity.name).
					Msg("Orchestrator: Workflow state updated to FAILED in etcd")
			}

			return fmt.Errorf("activity %s failed after %d retries: %w", activity.name, maxRetries, activityErr)
		}

		// Activity succeeded
		stepMap["status"] = "COMPLETED"
		stepMap["completedAt"] = time.Now().Format(time.RFC3339)
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)

		if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
			log.Error().
				Err(err).
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Str("stepStatus", "COMPLETED").
				Msg("Orchestrator: Failed to update completed step status in etcd")
		} else {
			log.Info().
				Str("workflowID", workflowID).
				Str("appName", appName).
				Str("step", activity.name).
				Msg("Orchestrator: Step status updated to COMPLETED in etcd")
		}

		log.Info().
			Str("workflowID", workflowID).
			Str("step", activity.name).
			Int("stepNumber", i+1).
			Int("totalSteps", len(activityList)).
			Msg("Activity completed successfully")
	}

	// All activities completed
	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Int("totalSteps", len(activityList)).
		Msg("Orchestrator: All activities completed successfully - updating workflow state to COMPLETED")

	workflowData["state"] = "COMPLETED"
	workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
	if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("appName", appName).
			Str("workflowState", "COMPLETED").
			Msg("Orchestrator: Failed to update workflow state to COMPLETED in etcd")
		return fmt.Errorf("failed to update completed workflow: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Msg("Orchestrator: Workflow state updated to COMPLETED in etcd successfully")

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", getStringFromPayload(payload, "appName")).
		Msg("Onboarding workflow completed successfully")

	return nil
}

// GetWorkflowStatus retrieves workflow status from etcd
// Returns a clean response without the payload to avoid exposing internal workflow data
func (o *Orchestrator) GetWorkflowStatus(workflowID string) (map[string]interface{}, error) {
	workflowData, err := o.etcdRepo.GetWorkflow(workflowID)
	if err != nil {
		return nil, err
	}

	// Build clean response without payload
	response := map[string]interface{}{
		"id":          workflowData["id"],
		"state":      workflowData["state"],
		"currentStep": workflowData["currentStep"],
		"retryCount":  workflowData["retryCount"],
		"createdAt":   workflowData["createdAt"],
		"createdBy":   workflowData["createdBy"],
		"updatedAt":   workflowData["updatedAt"],
		"workingEnv":  workflowData["workingEnv"],
		"steps":       workflowData["steps"],
	}

	// Only include appName from payload if needed for identification (not the full payload)
	if payload, ok := workflowData["payload"].(map[string]interface{}); ok {
		if appName, ok := payload["appName"].(string); ok && appName != "" {
			response["appName"] = appName
		}
	}

	return response, nil
}

// Helper functions
// generateWorkflowID generates a workflow ID with "onboarding-" prefix
func generateWorkflowID() string {
	return fmt.Sprintf("onboarding-%d", time.Now().UnixNano())
}

// generateThresholdUpdateWorkflowID generates a workflow ID with "threshold-update-" prefix
func generateThresholdUpdateWorkflowID() string {
	return fmt.Sprintf("threshold-update-%d", time.Now().UnixNano())
}

// generateInfrastructureWorkflowID generates a workflow ID with a descriptive prefix
func generateInfrastructureWorkflowID(operation string) string {
	return fmt.Sprintf("%s-%d", operation, time.Now().UnixNano())
}

// StartCPUThresholdUpdateWorkflow creates a new CPU threshold update workflow
func (o *Orchestrator) StartCPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateInfrastructureWorkflowID("cpu-threshold-update")
	return o.startInfrastructureWorkflow(workflowID, "UpdateCPUThreshold", payload, createdBy, workingEnv)
}

// StartGPUThresholdUpdateWorkflow creates a new GPU threshold update workflow
func (o *Orchestrator) StartGPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateInfrastructureWorkflowID("gpu-threshold-update")
	return o.startInfrastructureWorkflow(workflowID, "UpdateGPUThreshold", payload, createdBy, workingEnv)
}

// StartSharedMemoryUpdateWorkflow creates a new shared memory update workflow
func (o *Orchestrator) StartSharedMemoryUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateInfrastructureWorkflowID("shared-memory-update")
	return o.startInfrastructureWorkflow(workflowID, "UpdateSharedMemory", payload, createdBy, workingEnv)
}

// StartPodAnnotationsUpdateWorkflow creates a new pod annotations update workflow
func (o *Orchestrator) StartPodAnnotationsUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateInfrastructureWorkflowID("pod-annotations-update")
	return o.startInfrastructureWorkflow(workflowID, "UpdatePodAnnotations", payload, createdBy, workingEnv)
}

// StartRestartDeploymentWorkflow creates a new restart deployment workflow
func (o *Orchestrator) StartRestartDeploymentWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateInfrastructureWorkflowID("restart-deployment")
	return o.startInfrastructureWorkflow(workflowID, "RestartDeployment", payload, createdBy, workingEnv)
}

// StartAutoscalingTriggersUpdateWorkflow creates a new autoscaling triggers update workflow
func (o *Orchestrator) StartAutoscalingTriggersUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	workflowID := generateInfrastructureWorkflowID("autoscaling-triggers-update")
	return o.startInfrastructureWorkflow(workflowID, "UpdateAutoscalingTriggers", payload, createdBy, workingEnv)
}

// startInfrastructureWorkflow is a helper function to create infrastructure workflows
func (o *Orchestrator) startInfrastructureWorkflow(workflowID, stepName string, payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	now := time.Now().Format(time.RFC3339)

	steps := []map[string]interface{}{
		{"name": stepName, "status": "PENDING", "error": ""},
	}

	workflowData := map[string]interface{}{
		"id":          workflowID,
		"state":       "PENDING",
		"payload":     payload,
		"steps":       steps,
		"currentStep": 0,
		"retryCount":  0,
		"createdBy":   createdBy,
		"workingEnv":  workingEnv,
		"createdAt":   now,
		"updatedAt":   now,
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("stepName", stepName).
		Str("appName", getStringFromPayload(payload, "appName")).
		Str("workingEnv", workingEnv).
		Str("createdBy", createdBy).
		Msg("Orchestrator: Creating infrastructure workflow entry in etcd")

	if err := o.etcdRepo.CreateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("stepName", stepName).
			Msg("Orchestrator: Failed to create infrastructure workflow in etcd")
		return "", fmt.Errorf("failed to create workflow in etcd: %w", err)
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("stepName", stepName).
		Str("appName", getStringFromPayload(payload, "appName")).
		Msg("Orchestrator: Infrastructure workflow created in etcd successfully")

	return workflowID, nil
}

// ExecuteInfrastructureWorkflow executes an infrastructure workflow by ID
func (o *Orchestrator) ExecuteInfrastructureWorkflow(ctx context.Context, workflowID string, activityName string, activityFn func(map[string]interface{}, string) error) error {
	log.Info().
		Str("workflowID", workflowID).
		Str("activityName", activityName).
		Msg("Orchestrator: Starting infrastructure workflow execution")

	workflowData, err := o.etcdRepo.GetWorkflow(workflowID)
	if err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Msg("Orchestrator: Failed to retrieve infrastructure workflow from etcd")
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	appName := getStringFromPayload(workflowData["payload"].(map[string]interface{}), "appName")
	workingEnv := workflowData["workingEnv"].(string)

	log.Info().
		Str("workflowID", workflowID).
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("activityName", activityName).
		Msg("Orchestrator: Infrastructure workflow retrieved from etcd successfully")

	// Update state to RUNNING
	workflowData["state"] = "RUNNING"
	workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
	if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Msg("Orchestrator: Failed to update infrastructure workflow state to RUNNING")
		return fmt.Errorf("failed to update workflow state: %w", err)
	}

	payload := workflowData["payload"].(map[string]interface{})
	steps := workflowData["steps"].([]interface{})

	// Update current step
	if len(steps) > 0 {
		stepMap := steps[0].(map[string]interface{})
		stepMap["status"] = "RUNNING"
		stepMap["startedAt"] = time.Now().Format(time.RFC3339)
		workflowData["currentStep"] = 0
		workflowData["steps"] = steps
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
		if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
			log.Error().
				Err(err).
				Str("workflowID", workflowID).
				Msg("Orchestrator: Failed to update infrastructure workflow step to RUNNING")
			return fmt.Errorf("failed to update workflow step: %w", err)
		}
	}

	// Execute activity
	if err := activityFn(payload, workingEnv); err != nil {
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Str("activityName", activityName).
			Msg("Orchestrator: Infrastructure workflow activity failed")

		// Update step with error
		if len(steps) > 0 {
			stepMap := steps[0].(map[string]interface{})
			stepMap["status"] = "FAILED"
			stepMap["error"] = err.Error()
			stepMap["completedAt"] = time.Now().Format(time.RFC3339)
			workflowData["steps"] = steps
			workflowData["state"] = "FAILED"
			workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
			_ = o.etcdRepo.UpdateWorkflow(workflowID, workflowData)
		}

		return fmt.Errorf("%s activity failed: %w", activityName, err)
	}

	// Update step to completed
	if len(steps) > 0 {
		stepMap := steps[0].(map[string]interface{})
		stepMap["status"] = "COMPLETED"
		stepMap["error"] = ""
		stepMap["completedAt"] = time.Now().Format(time.RFC3339)
		workflowData["steps"] = steps
		workflowData["state"] = "COMPLETED"
		workflowData["currentStep"] = len(steps)
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
		if err := o.etcdRepo.UpdateWorkflow(workflowID, workflowData); err != nil {
			log.Error().
				Err(err).
				Str("workflowID", workflowID).
				Msg("Orchestrator: Failed to update infrastructure workflow step to COMPLETED")
			return fmt.Errorf("failed to update workflow step: %w", err)
		}
	}

	log.Info().
		Str("workflowID", workflowID).
		Str("activityName", activityName).
		Str("appName", appName).
		Msg("Orchestrator: Infrastructure workflow completed successfully")

	return nil
}

func getStringFromPayload(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
