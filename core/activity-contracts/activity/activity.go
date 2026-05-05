package activity

import "context"

// GlobalActivityOutputs maps each action name to a list of its result maps.
// {"noop.log": [{"logged": "hello"}, {"logged": "world"}], "noop.echo": [...]}
type GlobalActivityOutputs map[string][]map[string]interface{}

// ActivityInput is the generic input passed to all activities.
type ActivityInput struct {
	Action              string                 `json:"action"`
	Params              map[string]interface{} `json:"params"`
	StepName            string                 `json:"step_name"`
	WorkflowExecutionId string                 `json:"workflow_execution_id"`
	Async               bool                   `json:"async"`
	PluginVersion       string                 `json:"plugin_version"`
	StepId              string                 `json:"step_id,omitempty"`
	// Attempt is the 1-based Temporal activity attempt number for this invocation.
	// Async plugins must include this in the payload they send to the external system
	// so that callbacks can be routed back to the exact attempt that triggered the work.
	Attempt int `json:"attempt,omitempty"`
	// Config holds plugin-specific configuration injected from etcd at call time.
	// Plugins should read environment-specific values from here instead of os.Getenv,
	// so that config can be updated in etcd without redeploying the orchestrator.
	Config map[string]string `json:"config,omitempty"`
}

// ActivityOutput is the generic output returned by all activities.
type ActivityOutput struct {
	Result map[string]interface{} `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

// ActivityFunc is the function signature that every registered activity must implement.
type ActivityFunc func(ctx context.Context, globalOutputs GlobalActivityOutputs, input ActivityInput) (ActivityOutput, error)
