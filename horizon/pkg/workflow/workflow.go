package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
)

const (
	workflowBasePath = "/config/horizon/workflows"
	appName          = "horizon"
)

// WorkflowState represents the current state of a workflow
type WorkflowState string

const (
	StatePending            WorkflowState = "PENDING"
	StateRunning            WorkflowState = "RUNNING"
	StateUpdatingDeployment WorkflowState = "UPDATING_DEPLOYMENT"
	StateCreatingApp        WorkflowState = "CREATING_APP"
	StateUpdatingValues     WorkflowState = "UPDATING_VALUES"
	StateCreatingDNS        WorkflowState = "CREATING_DNS"
	StateCompleted          WorkflowState = "COMPLETED"
	StateFailed             WorkflowState = "FAILED"
)

// StepStatus represents the status of a workflow step
type StepStatus string

const (
	StepStatusPending   StepStatus = "PENDING"
	StepStatusRunning   StepStatus = "RUNNING"
	StepStatusCompleted StepStatus = "COMPLETED"
	StepStatusFailed    StepStatus = "FAILED"
)

// Workflow represents an onboarding workflow
type Workflow struct {
	ID          string                 `json:"id"`
	State       WorkflowState          `json:"state"`
	Payload     map[string]interface{} `json:"payload"`
	Steps       []WorkflowStep         `json:"steps"`
	CurrentStep int                    `json:"currentStep"`
	RetryCount  int                    `json:"retryCount"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// WorkflowStep represents a single step in the workflow
type WorkflowStep struct {
	Name        string     `json:"name"`
	Status      StepStatus `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// WorkflowRepository handles workflow persistence in etcd
// Similar to how numerix uses etcd for configuration storage
type WorkflowRepository struct {
	etcdInstance etcd.Etcd
}

// NewWorkflowRepository creates a new workflow repository using etcd
// Follows the same pattern as numerix/inferflow etcd usage
func NewWorkflowRepository(etcdInstance etcd.Etcd) *WorkflowRepository {
	return &WorkflowRepository{
		etcdInstance: etcdInstance,
	}
}

// GetWorkflow retrieves a workflow from etcd
func (r *WorkflowRepository) GetWorkflow(workflowID string) (*Workflow, error) {
	path := fmt.Sprintf("%s/%s", workflowBasePath, workflowID)

	value, err := r.etcdInstance.GetValue(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow from etcd: %w", err)
	}

	if value == "" {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	var workflow Workflow
	if err := json.Unmarshal([]byte(value), &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	return &workflow, nil
}

// SaveWorkflow saves a workflow to etcd
// Uses CreateNode for new workflows, SetValue for updates (like numerix pattern)
func (r *WorkflowRepository) SaveWorkflow(workflow *Workflow) error {
	workflow.UpdatedAt = time.Now()

	path := fmt.Sprintf("%s/%s", workflowBasePath, workflow.ID)

	workflowJSON, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Check if workflow exists
	exists, err := r.etcdInstance.IsLeafNodeExist(path)
	if err != nil {
		return fmt.Errorf("failed to check if workflow exists: %w", err)
	}

	if exists {
		// Update existing workflow (like numerix UpdateConfig)
		if err := r.etcdInstance.SetValue(path, string(workflowJSON)); err != nil {
			return fmt.Errorf("failed to update workflow in etcd: %w", err)
		}
	} else {
		// Create new workflow (like numerix CreateConfig)
		if err := r.etcdInstance.CreateNode(path, string(workflowJSON)); err != nil {
			return fmt.Errorf("failed to create workflow in etcd: %w", err)
		}
	}

	return nil
}
