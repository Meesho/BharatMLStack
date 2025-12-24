package etcd

import (
	"bytes"
	"encoding/json"
	"fmt"

	workflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
)

type Etcd struct {
	instance etcd.Etcd
	appName  string
	env      string
}

// NewEtcdInstance creates a new etcd instance for workflow management
// Follows the exact same pattern as numerix NewEtcdInstance
func NewEtcdInstance() *Etcd {
	return &Etcd{
		instance: etcd.Instance()[workflowPkg.WorkflowAppName],
		appName:  workflowPkg.WorkflowAppName,
		env:      workflowPkg.AppEnv,
	}
}

// CreateWorkflow creates a new workflow in etcd
// Follows the exact same pattern as numerix CreateConfig
func (e *Etcd) CreateWorkflow(workflowID string, workflowData map[string]interface{}) error {
	// Use JSON encoder with HTML escaping disabled (same as numerix)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(workflowData)
	if err != nil {
		return err
	}

	// Remove the trailing newline added by Encode (same as numerix)
	workflowJSON := bytes.TrimSpace(buf.Bytes())

	// Path pattern: /config/{appName}/workflows/{workflowID}
	// Same pattern as numerix: /config/{appName}/expression-config/{configId}
	path := fmt.Sprintf("/config/%s/workflows/%s", e.appName, workflowID)
	return e.instance.CreateNode(path, string(workflowJSON))
}

// UpdateWorkflow updates an existing workflow in etcd
// Follows the exact same pattern as numerix UpdateConfig
func (e *Etcd) UpdateWorkflow(workflowID string, workflowData map[string]interface{}) error {
	// Use JSON encoder with HTML escaping disabled (same as numerix)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(workflowData)
	if err != nil {
		return err
	}

	// Remove the trailing newline added by Encode (same as numerix)
	workflowJSON := bytes.TrimSpace(buf.Bytes())

	// Path pattern: /config/{appName}/workflows/{workflowID}
	// Same pattern as numerix: /config/{appName}/expression-config/{configId}
	path := fmt.Sprintf("/config/%s/workflows/%s", e.appName, workflowID)
	return e.instance.SetValue(path, string(workflowJSON))
}

// GetWorkflow retrieves a workflow from etcd
func (e *Etcd) GetWorkflow(workflowID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/config/%s/workflows/%s", e.appName, workflowID)

	value, err := e.instance.GetValue(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow from etcd: %w", err)
	}

	if value == "" {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	var workflowData map[string]interface{}
	if err := json.Unmarshal([]byte(value), &workflowData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	return workflowData, nil
}

// DeleteWorkflow deletes a workflow from etcd
// Follows the exact same pattern as numerix DeleteConfig
func (e *Etcd) DeleteWorkflow(workflowID string) error {
	// Path pattern: /config/{appName}/workflows/{workflowID}
	// Same pattern as numerix: /config/{appName}/expression-config/{configId}
	path := fmt.Sprintf("/config/%s/workflows/%s", e.appName, workflowID)
	return e.instance.DeleteNode(path)
}
