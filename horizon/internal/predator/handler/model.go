package handler

import (
	"encoding/json"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/predator/proto/modelconfig"
)

type Payload struct {
	ModelName         string        `json:"model_name"`
	ModelSource       string        `json:"model_source_path,omitempty"`
	MetaData          MetaData      `json:"meta_data"`
	ConfigMapping     ConfigMapping `json:"config_mapping"`
	DiscoveryConfigID uint          `json:"discovery_config_id"`
}

type MetaData struct {
	Inputs                 []IOField  `json:"inputs"`
	Outputs                []IOField  `json:"outputs"`
	InstanceCount          int        `json:"instance_count"`
	InstanceType           string     `json:"instance_type"`
	BatchSize              int        `json:"batch_size"`
	Platform               string     `json:"platform"`
	Backend                string     `json:"backend"`
	DynamicBatchingEnabled bool       `json:"dynamic_batching_enabled"`
	Ensembling             Ensembling `json:"ensemble_scheduling"`
}

type Ensembling struct {
	Step []STEP `json:"step,omitempty"`
}

type STEP struct {
	ModelName    string            `json:"model_name"`
	ModelVersion int               `json:"model_version"`
	InputMap     map[string]string `json:"input_map"`
	OutputMap    map[string]string `json:"output_map"`
}

type IOField struct {
	Name     string      `json:"name"`
	Dims     interface{} `json:"dims"`
	DataType string      `json:"data_type"`
	Features []string    `json:"features,omitempty"`
}

type ConfigMapping struct {
	ServiceDeployableID uint `json:"service_deployable_id"`
	SourceModelName     string `json:"source_model_name,omitempty"`
}

type FetchModelConfigRequest struct {
	ModelPath string
}

type PredatorRequestDTO struct {
	Payload     Payload `json:"payload"`
	CreatedBy   string  `json:"craeted_by"`
	RequestType string  `json:"request_type"`
	UpdatedBy   string  `json:"updated_by"`
}

type PredatorDeployableConfig struct {
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

type ModelResponse struct {
	ID                      int             `json:"id"`
	ModelName               string          `json:"model_name"`
	MetaData                json.RawMessage `json:"meta_data"`
	Host                    string          `json:"host"`
	MachineType             string          `json:"machine_type"`
	DeploymentConfig        any             `json:"deployment_config"`
	AppToken                string          `json:"app_token"`
	MonitoringUrl           string          `json:"monitoring_url"`
	GCSPath                 string          `json:"gcs_path"`
	CreatedBy               string          `json:"created_by"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedBy               string          `json:"updated_by"`
	UpdatedAt               time.Time       `json:"updated_at"`
	DeployableRunningStatus string          `json:"deployable_running_status"`
	TestResults             json.RawMessage `json:"test_results"`
	HasNilData              bool            `json:"has_nil_data"`
	SourceModelName         string          `json:"source_model_name,omitempty"`
}

type PredatorRequestResponse struct {
	RequestID    uint                   `json:"request_id"`
	GroupID      uint                   `json:"group_id"`
	Payload      map[string]interface{} `json:"payload"`
	CreatedBy    string                 `json:"created_by"`
	UpdatedBy    string                 `json:"updated_by"`
	Reviewer     string                 `json:"reviewer"`
	RequestStage string                 `json:"request_stage"`
	RequestType  string                 `json:"request_type"`
	Status       string                 `json:"status"`
	RejectReason string                 `json:"reject_reason"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	IsValid      bool                   `json:"is_valid"`
	HasNilData   bool                   `json:"has_nil_data"`
	TestResults  json.RawMessage        `json:"test_results"`
}

type ModelRequest struct {
	Payload   []map[string]interface{} `json:"payload" binding:"required"`
	CreatedBy string                   `json:"created_by" binding:"required"`
}

type ApproveRequest struct {
	GroupID      string `json:"group_id"`
	Status       string `json:"status"`
	ApprovedBy   string `json:"approved_by"`
	RejectReason string `json:"reject_reason"`
}

type GcsModelData struct {
	Bucket string
	Path   string
	Name   string
}

type GcsTransferredData struct {
	SrcBucket  string
	SrcPath    string
	SrcName    string
	DestBucket string
	DestPath   string
	DestName   string
}

type IO struct {
	Name     string   `json:"name"`
	Dims     any      `json:"dims"`
	DataType string   `json:"data_type"`
	Features []string `json:"features,omitempty"`
}

type DeleteRequest struct {
	IDs []int `json:"ids"`
}

type ModelParamsResponse struct {
	ModelName              string           `json:"model_name"`
	Inputs                 []IO             `json:"inputs"`
	Outputs                []IO             `json:"outputs"`
	InstanceCount          int32            `json:"instance_count"`
	InstanceType           string           `json:"instance_type"`
	BatchSize              int32            `json:"batch_size"`
	Backend                string           `json:"backend"`
	DynamicBatchingEnabled bool             `json:"dynamic_batching_enabled"`
	Platform               string           `json:"platform"`
	EnsembleScheduling     *modelconfig.ModelEnsembling `json:"ensemble_scheduling,omitempty"`
}

type RequestGenerationRequest struct {
	BatchSize string `json:"batch_size"`
	ModelName string `json:"model_name"`
}

type RequestGenerationResponse struct {
	ModelName   string      `json:"model_name"`
	RequestBody RequestBody `json:"request_body"`
}

type RequestBody struct {
	Inputs  []Input   `json:"inputs"`
	Outputs []Outputs `json:"outputs"`
}

type Outputs struct {
	Name     string   `json:"name"`
	Dims     [][]int  `json:"dims"`
	Features []string `json:"features,omitempty"`
}

type Input struct {
	Name     string   `json:"name"`
	Dims     []int64  `json:"dims"`
	DataType string   `json:"data_type"`
	Data     any      `json:"data"` // Multi dimesnional array
	Features []string `json:"features,omitempty"`
}

type ExecuteRequestFunctionalRequest struct {
	EndPoint    string      `json:"end_point"`
	ModelName   string      `json:"model_name"`
	RequestBody RequestBody `json:"request_body"`
}

type ExecuteRequestFunctionalResponse struct {
	ModelName    string   `json:"model_name"`
	ModelVersion string   `json:"model_version"`
	Outputs      []Output `json:"outputs"`
}

type Output struct {
	Name     string  `json:"name"`
	Dims     []int64 `json:"dims"`
	DataType string  `json:"data_type"`
	Data     any     `json:"data"`
}

type ExecuteRequestLoadTest struct {
	LoadTestConfig LoadTestConfig    `json:"load_test_config"`
	Endpoint       string            `json:"end_point"`
	RequestBody    RequestBody       `json:"request_body"`
	Metadata       map[string]string `json:"metadata"`
	ModelName      string            `json:"model_name"`
}

type LoadTestConfig struct {
	TotalRequests string `json:"total_requests"`
	Concurrency   string `json:"concurrency"`
	Duration      string `json:"duration"`
	LoadStart     string `json:"load_start"`
	LoadStep      string `json:"load_step"`
	LoadEnd       string `json:"load_end"`
	Connection    string `json:"connection"`
	RPS           string `json:"rps"`
}

type PhoenixLoadTestRequest struct {
	LoadTestConfig LoadTestConfig    `json:"load_test_config"`
	Service        string            `json:"service"`
	Endpoint       string            `json:"end_point"`
	Data           json.RawMessage   `json:"data"`
	Metadata       map[string]string `json:"metadata"`
	ModelName      string            `json:"model_name"`
}

type PhoenixClientResponse struct {
	IsBusy  bool   `json:"isBusy"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type TritonInferenceRequest struct {
	ModelName        string               `json:"model_name"`
	Inputs           []TritonInputTensor  `json:"inputs"`
	Outputs          []TritonOutputTensor `json:"outputs"`
	RawInputContents [][]byte             `json:"raw_input_contents"`
}

type TritonInputTensor struct {
	Name     string  `json:"name"`
	Datatype string  `json:"datatype"`
	Shape    []int64 `json:"shape"`
}

type TritonOutputTensor struct {
	Name string `json:"name"`
}
