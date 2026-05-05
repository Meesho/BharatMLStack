package handler

// Handler interface for workflow operations (same pattern as numerix Config interface)
type Handler interface {
	// StartOnboardingWorkflow starts a new onboarding workflow and returns workflow ID
	// This is non-blocking - execution happens asynchronously
	StartOnboardingWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartThresholdUpdateWorkflow starts a new threshold update workflow and returns workflow ID
	// This is non-blocking - execution happens asynchronously
	StartThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartCPUThresholdUpdateWorkflow starts a new CPU threshold update workflow and returns workflow ID
	StartCPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartGPUThresholdUpdateWorkflow starts a new GPU threshold update workflow and returns workflow ID
	StartGPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartSharedMemoryUpdateWorkflow starts a new shared memory update workflow and returns workflow ID
	StartSharedMemoryUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartPodAnnotationsUpdateWorkflow starts a new pod annotations update workflow and returns workflow ID
	StartPodAnnotationsUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartRestartDeploymentWorkflow starts a new restart deployment workflow and returns workflow ID
	StartRestartDeploymentWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// StartAutoscalingTriggersUpdateWorkflow starts a new autoscaling triggers update workflow and returns workflow ID
	StartAutoscalingTriggersUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error)
	
	// GetWorkflowStatus retrieves the current status of a workflow
	GetWorkflowStatus(workflowID string) (map[string]interface{}, error)
}

