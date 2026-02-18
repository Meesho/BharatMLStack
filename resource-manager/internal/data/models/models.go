package models

import (
	"time"

	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

type Owner struct {
	WorkflowRunID string    `json:"workflow_run_id"`
	WorkflowPlan  string    `json:"workflow_plan"`
	ProcuredAt    time.Time `json:"procured_at"`
}

type ShadowDeployable struct {
	Name          string              `json:"name"`
	NodePool      string              `json:"node_pool"`
	DNS           string              `json:"dns"`
	State         rmtypes.ShadowState `json:"state"`
	Owner         *Owner              `json:"owner,omitempty"`
	MinPodCount   int                 `json:"pod_count"`
	LastUpdatedAt time.Time           `json:"last_updated_at"`
	Version       int64               `json:"version"`
}

type Callback struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type WorkflowContext struct {
	RunID  string `json:"workflow_run_id"`
	Plan   string `json:"workflow_plan"`
	TaskID string `json:"workflow_task_id"`
}

type WatchResource struct {
	Kind          string `json:"kind"`
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector"`
	Name          string `json:"name,omitempty"`
}

type WatchIntent struct {
	SchemaVersion    string          `json:"schemaVersion"`
	RequestID        string          `json:"requestId"`
	Operation        string          `json:"operation"`
	Resource         WatchResource   `json:"resource"`
	DesiredCondition string          `json:"desiredCondition"`
	Callback         Callback        `json:"callback"`
	Workflow         WorkflowContext `json:"workflow"`
	CreatedAt        time.Time       `json:"createdAt"`
}

type ShadowFilter struct {
	InUse    *bool
	NodePool string
}

type DeployableSpec struct {
	Name         string
	Namespace    string
	NodeSelector string
	MinPodCount  int
	MaxPodCount  int
}

type ModelLoadSpec struct {
	DeployableName string
	ModelName      string
	ModelVersion   string
}

type JobSpec struct {
	JobName string
}

type RestartSpec struct {
	Namespace string
}

type PublishResult struct {
	MessageID string
}

type IdempotencyRecord struct {
	RequestHash  string
	StatusCode   int
	ResponseBody []byte
	ContentType  string
}
