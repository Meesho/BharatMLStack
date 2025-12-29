package handler

// DeployableRequest represents the request body for creating a new deployable
// Supports both single environment (legacy) and multi-environment (new) formats
type DeployableRequest struct {
	AppName     string `json:"appName"`
	ServiceName string `json:"service_name"`
	CreatedBy   string `json:"created_by"`
	// Legacy fields (used when environments array is not provided)
	MachineType        string            `json:"machine_type,omitempty"`
	CPURequest         string            `json:"cpuRequest,omitempty"`
	CPURequestUnit     string            `json:"cpuRequestUnit,omitempty"`
	CPULimit           string            `json:"cpuLimit,omitempty"`
	CPULimitUnit       string            `json:"cpuLimitUnit,omitempty"`
	MemoryRequest      string            `json:"memoryRequest,omitempty"`
	MemoryRequestUnit  string            `json:"memoryRequestUnit,omitempty"`
	MemoryLimit        string            `json:"memoryLimit,omitempty"`
	MemoryLimitUnit    string            `json:"memoryLimitUnit,omitempty"`
	GPURequest         string            `json:"gpu_request,omitempty"`
	GPULimit           string            `json:"gpu_limit,omitempty"`
	MinReplica         string            `json:"min_replica,omitempty"`
	MaxReplica         string            `json:"max_replica,omitempty"`
	GCSBucketPath      string            `json:"gcs_bucket_path,omitempty"`
	GCSTritonPath      bool              `json:"gcs_triton_path,omitempty"`
	TritonImageTag     string            `json:"triton_image_tag,omitempty"`
	ServiceAccount     string            `json:"serviceAccount,omitempty"`
	NodeSelectorValue  string            `json:"nodeSelectorValue,omitempty"`
	DeploymentStrategy string            `json:"deploymentStrategy,omitempty"`
	ServiceType        string            `json:"service_type,omitempty"`   // Service type: "httpstateless" (default) or "grpc"
	PodAnnotations     map[string]string `json:"podAnnotations,omitempty"` // Pod annotations (higher precedence than config.yaml)
	// New multi-environment support
	Environments []EnvironmentConfig `json:"environments,omitempty"`
}

// EnvironmentConfig represents configuration for a specific environment
type EnvironmentConfig struct {
	WorkingEnv         string            `json:"workingEnv" binding:"required"` // e.g., "gcp_stg", "gcp_prod", "gcp_int"
	MachineType        string            `json:"machine_type,omitempty"`
	CPURequest         string            `json:"cpuRequest,omitempty"`
	CPURequestUnit     string            `json:"cpuRequestUnit,omitempty"`
	CPULimit           string            `json:"cpuLimit,omitempty"`
	CPULimitUnit       string            `json:"cpuLimitUnit,omitempty"`
	MemoryRequest      string            `json:"memoryRequest,omitempty"`
	MemoryRequestUnit  string            `json:"memoryRequestUnit,omitempty"`
	MemoryLimit        string            `json:"memoryLimit,omitempty"`
	MemoryLimitUnit    string            `json:"memoryLimitUnit,omitempty"`
	GPURequest         string            `json:"gpu_request,omitempty"`
	GPULimit           string            `json:"gpu_limit,omitempty"`
	MinReplica         string            `json:"min_replica,omitempty"`
	MaxReplica         string            `json:"max_replica,omitempty"`
	GCSBucketPath      string            `json:"gcs_bucket_path,omitempty"`
	GCSTritonPath      bool              `json:"gcs_triton_path,omitempty"`
	TritonImageTag     string            `json:"triton_image_tag,omitempty"`
	ServiceAccount     string            `json:"serviceAccount,omitempty"`
	NodeSelectorValue  string            `json:"nodeSelectorValue,omitempty"`
	DeploymentStrategy string            `json:"deploymentStrategy,omitempty"`
	ServiceType        string            `json:"service_type,omitempty"`   // Service type: "httpstateless" (default) or "grpc"
	PodAnnotations     map[string]string `json:"podAnnotations,omitempty"` // Pod annotations (higher precedence than config.yaml)
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
	GCSTritonPath      bool   `json:"gcs_triton_path"`
	CPUThreshold       string `json:"cpu_threshold"`
	GPUThreshold       string `json:"gpu_threshold"`
	DeploymentStrategy string `json:"deploymentStrategy"`
}

// TuneThresholdsRequest represents a request to update CPU and/or GPU thresholds
// Both cpu_threshold and gpu_threshold are optional, but at least one must be provided
// machine_type is required and determines which thresholds can be updated:
//   - For "GPU" machine type: Both CPU and GPU thresholds can be updated
//   - For other machine types: Only CPU threshold can be updated (GPU threshold will be rejected)
type TuneThresholdsRequest struct {
	AppName      string `json:"appName" binding:"required"`      // Application name (required)
	ServiceName  string `json:"service_name" binding:"required"` // Service name (required)
	MachineType  string `json:"machine_type" binding:"required"` // Machine type: "GPU" or other (required, determines if GPU threshold can be updated)
	CPUThreshold string `json:"cpu_threshold"`                   // CPU threshold value (optional, but at least one threshold must be provided)
	GPUThreshold string `json:"gpu_threshold"`                   // GPU threshold value (optional, only valid for "GPU" machine type)
}
