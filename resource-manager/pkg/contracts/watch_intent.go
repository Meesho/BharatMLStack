package contracts

type WatchResource struct {
	Kind          string `json:"kind"`
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector"`
	Name          string `json:"name,omitempty"`
}

type Callback struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type Workflow struct {
	RunID  string `json:"runId"`
	Plan   string `json:"plan"`
	TaskID string `json:"taskId"`
}

type WatchIntentV1 struct {
	SchemaVersion    string        `json:"schemaVersion"`
	RequestID        string        `json:"requestId"`
	Operation        string        `json:"operation"`
	Resource         WatchResource `json:"resource"`
	DesiredCondition string        `json:"desiredCondition"`
	Callback         Callback      `json:"callback"`
	Workflow         Workflow      `json:"workflow"`
	CreatedAt        string        `json:"createdAt"`
}
