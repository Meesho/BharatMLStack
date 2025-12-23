package handler

// DeployableRequest represents the request body for creating a new deployable
type DeployableRequest struct {
	AppName            string `json:"appName"`
	ServiceName        string `json:"service_name"`
	MachineType        string `json:"machine_type"`
	CPURequest         string `json:"cpuRequest"`
	CPURequestUnit     string `json:"cpuRequestUnit"`
	CPULimit           string `json:"cpuLimit"`
	CPULimitUnit       string `json:"cpuLimitUnit"`
	MemoryRequest      string `json:"memoryRequest"`
	MemoryRequestUnit  string `json:"memoryRequestUnit"`
	MemoryLimit        string `json:"memoryLimit"`
	MemoryLimitUnit    string `json:"memoryLimitUnit"`
	GPURequest         string `json:"gpu_request"`
	GPULimit           string `json:"gpu_limit"`
	MinReplica         string `json:"min_replica"`
	MaxReplica         string `json:"max_replica"`
	GCSBucketPath      string `json:"gcs_bucket_path"`
	GCSTritonPath      string `json:"gcs_triton_path"`
	TritonImageTag     string `json:"triton_image_tag"`
	CreatedBy          string `json:"created_by"`
	ServiceAccount     string `json:"serviceAccount"`
	NodeSelectorValue  string `json:"nodeSelectorValue"`
	DeploymentStrategy string `json:"deploymentStrategy"`
}

// OnboardResponse represents the response from ringmaster onboarding API
type OnboardResponse struct {
	DeploymentTriggered  bool   `json:"deployment-triggered"`
	DeploymentRunID      string `json:"deploymentRunID"`
	DeploymentWorkflowID string `json:"deploymentWorkflowID"`
	Onboarded            bool   `json:"onboarded"`
	ServiceAccountBound  bool   `json:"serviceAccount-bound"`
}

type DeployableConfigPayload struct {
	MachineType        string `json:"machine_type"`
	CPURequest         string `json:"cpuRequest"`
	CPURequestUnit     string `json:"cpuRequestUnit"`
	CPULimit           string `json:"cpuLimit"`
	CPULimitUnit       string `json:"cpuLimitUnit"`
	MemoryRequest      string `json:"memoryRequest"`
	MemoryRequestUnit  string `json:"memoryRequestUnit"`
	MemoryLimit        string `json:"memoryLimit"`
	MemoryLimitUnit    string `json:"memoryLimitUnit"`
	GPURequest         string `json:"gpu_request"`
	GPULimit           string `json:"gpu_limit"`
	MinReplica         string `json:"min_replica"`
	MaxReplica         string `json:"max_replica"`
	GCSBucketPath      string `json:"gcs_bucket_path"`
	TritonImageTag     string `json:"triton_image_tag"`
	ServiceAccount     string `json:"serviceAccount"`
	NodeSelectorValue  string `json:"nodeSelectorValue"`
	GCSTritonPath      string `json:"gcs_triton_path"`
	CPUThreshold       string `json:"cpu_threshold"`
	GPUThreshold       string `json:"gpu_threshold"`
	DeploymentStrategy string `json:"deploymentStrategy"`
}

type TuneThresholdsRequest struct {
	AppName      string `json:"appName" binding:"required"`
	ServiceName  string `json:"service_name" binding:"required"`
	MachineType  string `json:"machine_type" binding:"required"`
	CPUThreshold string `json:"cpu_threshold"`
	GPUThreshold string `json:"gpu_threshold"`
}
