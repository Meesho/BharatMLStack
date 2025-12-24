package etcd

import (
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEtcdWorkflowOperations tests basic etcd operations for workflows
// This test requires etcd to be running locally on localhost:2379
func TestEtcdWorkflowOperations(t *testing.T) {
	// Skip if etcd is not available (for CI/CD environments)
	if testing.Short() {
		t.Skip("Skipping etcd integration test in short mode")
	}

	// Initialize etcd with WorkflowRegistry (same pattern as numerix)
	registry := &WorkflowRegistry{
		Workflows: make(map[string]WorkflowData),
	}
	
	// Create a test config
	testConfig := configs.Configs{
		EtcdServer:         "localhost:2379",
		EtcdUsername:       "",
		EtcdPassword:       "",
		EtcdWatcherEnabled: false,
		HorizonAppName:     "horizon",
	}
	
	// Initialize etcd (same as numerix initialization in main.go)
	etcd.InitFromAppName(registry, testConfig.HorizonAppName, testConfig)
	
	// Create etcd instance (same pattern as numerix)
	workflowEtcd := NewEtcdInstance()
	require.NotNil(t, workflowEtcd)
	
	// Test data
	workflowID := "test-workflow-123"
	workflowData := map[string]interface{}{
		"id":          workflowID,
		"state":       "PENDING",
		"payload":     map[string]interface{}{"appName": "test-app"},
		"currentStep": 0,
		"retryCount":  0,
		"createdAt":   time.Now().Format(time.RFC3339),
		"updatedAt":   time.Now().Format(time.RFC3339),
	}
	
	// Test CreateWorkflow (same as numerix CreateConfig)
	t.Run("CreateWorkflow", func(t *testing.T) {
		err := workflowEtcd.CreateWorkflow(workflowID, workflowData)
		assert.NoError(t, err, "CreateWorkflow should succeed")
	})
	
	// Test GetWorkflow (new method using GetValue)
	t.Run("GetWorkflow", func(t *testing.T) {
		retrieved, err := workflowEtcd.GetWorkflow(workflowID)
		require.NoError(t, err, "GetWorkflow should succeed")
		assert.Equal(t, workflowID, retrieved["id"], "Workflow ID should match")
		assert.Equal(t, "PENDING", retrieved["state"], "Workflow state should match")
	})
	
	// Test UpdateWorkflow (same as numerix UpdateConfig)
	t.Run("UpdateWorkflow", func(t *testing.T) {
		workflowData["state"] = "RUNNING"
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)
		
		err := workflowEtcd.UpdateWorkflow(workflowID, workflowData)
		assert.NoError(t, err, "UpdateWorkflow should succeed")
		
		// Verify update
		retrieved, err := workflowEtcd.GetWorkflow(workflowID)
		require.NoError(t, err)
		assert.Equal(t, "RUNNING", retrieved["state"], "Workflow state should be updated")
	})
	
	// Test DeleteWorkflow (same as numerix DeleteConfig)
	t.Run("DeleteWorkflow", func(t *testing.T) {
		err := workflowEtcd.DeleteWorkflow(workflowID)
		assert.NoError(t, err, "DeleteWorkflow should succeed")
		
		// Verify deletion
		_, err = workflowEtcd.GetWorkflow(workflowID)
		assert.Error(t, err, "GetWorkflow should fail after deletion")
	})
}

