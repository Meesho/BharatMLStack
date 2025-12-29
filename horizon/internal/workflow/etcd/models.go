package etcd

// WorkflowRegistry represents the workflow registry structure in etcd
// Similar to NumerixConfigRegistery pattern
type WorkflowRegistry struct {
	Workflows map[string]WorkflowData `json:"workflows"`
}

// WorkflowData represents a single workflow stored in etcd
type WorkflowData struct {
	ID          string                 `json:"id"`
	State       string                 `json:"state"`
	Payload     map[string]interface{} `json:"payload"`
	Steps       []WorkflowStep         `json:"steps"`
	CurrentStep int                    `json:"currentStep"`
	RetryCount  int                    `json:"retryCount"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// WorkflowStep represents a single step in the workflow
type WorkflowStep struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

