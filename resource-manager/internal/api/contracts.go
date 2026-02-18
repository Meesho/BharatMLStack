package api

import rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"

type ErrorResponse struct {
	Error string `json:"error"`
}

type ListShadowDeployablesResponse struct {
	Data []ShadowDeployableView `json:"data"`
}

type ShadowDeployableView struct {
	Name     string `json:"name"`
	DNS      string `json:"dns"`
	PodCount int    `json:"pod_count"`
	InUse    bool   `json:"in_use"`
	NodePool string `json:"node_pool"`
}

type ShadowDeployableCommandRequest struct {
	Action        rmtypes.Action `json:"action"`
	Name          string         `json:"name"`
	WorkflowRunID string         `json:"workflow_run_id"`
	WorkflowPlan  string         `json:"workflow_plan"`
}

type ShadowDeployableCommandResponse struct {
	Status string `json:"status"`
	Delay  int    `json:"delay,omitempty"`
}

type MinPodCountRequest struct {
	Action         rmtypes.Action `json:"action"`
	Count          int            `json:"count"`
	Name           string         `json:"name"`
	WorkflowRunID  string         `json:"workflow_run_id"`
	WorkflowTaskID string         `json:"workflow_task_id"`
	WorkflowPlan   string         `json:"workflow_plan"`
}

type CallbackInput struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type AsyncAcceptedResponse struct {
	Message string `json:"message"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status"`
}

type CreateDeployableRequest struct {
	Name           string        `json:"name"`
	Image          string        `json:"image"`
	NodeSelector   string        `json:"node_selector"`
	MinPodCount    int           `json:"min_pod_count"`
	MaxPodCount    int           `json:"max_pod_count"`
	Resources      ResourceSpec  `json:"resources"`
	ModelsToLoad   []ModelConfig `json:"modelsToLoad"`
	WorkflowRunID  string        `json:"workflow_run_id"`
	WorkflowPlan   string        `json:"workflow_plan"`
	WorkflowTaskID string        `json:"workflow_task_id"`
	Callback       CallbackInput `json:"callback"`
}

type LoadModelRequest struct {
	DeployableName string        `json:"deployable_name"`
	Model          LoadModelSpec `json:"model"`
	WorkflowRunID  string        `json:"workflow_run_id"`
	WorkflowPlan   string        `json:"workflow_plan"`
	WorkflowTaskID string        `json:"workflow_task_id"`
	Callback       CallbackInput `json:"callback"`
}

type TriggerJobRequest struct {
	JobName        string         `json:"job_name"`
	Container      JobContainer   `json:"container"`
	Payload        map[string]any `json:"payload"`
	WorkflowRunID  string         `json:"workflow_run_id"`
	WorkflowPlan   string         `json:"workflow_plan"`
	WorkflowTaskID string         `json:"workflow_task_id"`
	Callback       CallbackInput  `json:"callback"`
}

type RestartDeployableRequest struct {
	Namespace      string        `json:"namespace"`
	WorkflowRunID  string        `json:"workflow_run_id"`
	WorkflowPlan   string        `json:"workflow_plan"`
	WorkflowTaskID string        `json:"workflow_task_id"`
	Callback       CallbackInput `json:"callback"`
}

type ScalarResource struct {
	Request string `json:"request"`
	Limit   string `json:"limit"`
}

type GPUResource struct {
	Request string         `json:"request"`
	Limit   string         `json:"limit"`
	Memory  ScalarResource `json:"memory"`
}

type ResourceSpec struct {
	CPU    ScalarResource `json:"cpu"`
	Memory ScalarResource `json:"memory"`
	GPU    GPUResource    `json:"gpu"`
}

type ModelConfig struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Runtime         string `json:"runtime"`
	Precision       string `json:"precision"`
	BatchSize       string `json:"batchSize"`
	ArtifactVersion string `json:"artifactVersion"`
}

type LoadModelSpec struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	ArtifactLocation string `json:"artifact_location"`
}

type JobContainer struct {
	Image string `json:"image"`
}
