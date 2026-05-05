package etcd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	workflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEtcdInstance is a mock implementation of etcd.Etcd interface
type MockEtcdInstance struct {
	mock.Mock
	etcd.Etcd
	// In-memory storage for testing
	storage map[string]string
}

func NewMockEtcdInstance() *MockEtcdInstance {
	return &MockEtcdInstance{
		storage: make(map[string]string),
	}
}

func (m *MockEtcdInstance) GetConfigInstance() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockEtcdInstance) SetValue(path string, value interface{}) error {
	args := m.Called(path, value)
	if args.Error(0) == nil {
		m.storage[path] = value.(string)
	}
	return args.Error(0)
}

func (m *MockEtcdInstance) GetValue(path string) (string, error) {
	args := m.Called(path)
	if val, ok := m.storage[path]; ok {
		return val, args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockEtcdInstance) CreateNode(path string, value interface{}) error {
	args := m.Called(path, value)
	if args.Error(0) == nil {
		m.storage[path] = value.(string)
	}
	return args.Error(0)
}

func (m *MockEtcdInstance) DeleteNode(path string) error {
	args := m.Called(path)
	if args.Error(0) == nil {
		delete(m.storage, path)
	}
	return args.Error(0)
}

// Implement other required methods with default behavior
func (m *MockEtcdInstance) SetValues(paths map[string]interface{}) error {
	args := m.Called(paths)
	return args.Error(0)
}

func (m *MockEtcdInstance) CreateNodes(paths map[string]interface{}) error {
	args := m.Called(paths)
	return args.Error(0)
}

func (m *MockEtcdInstance) IsNodeExist(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *MockEtcdInstance) IsLeafNodeExist(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *MockEtcdInstance) RegisterWatchPathCallback(path string, callback func() error) error {
	args := m.Called(path, callback)
	return args.Error(0)
}

func (m *MockEtcdInstance) Delete(path string) error {
	return m.DeleteNode(path)
}

// TestEtcdWorkflowOperations tests basic etcd operations for workflows using mocks
func TestEtcdWorkflowOperations(t *testing.T) {
	// Set WorkflowAppName for testing (normally set by Init)
	workflowPkg.WorkflowAppName = "horizon"

	// Create mock etcd instance
	mockEtcd := NewMockEtcdInstance()

	// Set up the mock instance in the etcd package
	etcd.SetMockInstance(map[string]etcd.Etcd{
		workflowPkg.WorkflowAppName: mockEtcd,
	})

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
		path := "/config/horizon/workflows/" + workflowID

		// Use mock.AnythingOfType to match the JSON string without exact comparison
		mockEtcd.On("CreateNode", path, mock.AnythingOfType("string")).Return(nil).Once()

		err := workflowEtcd.CreateWorkflow(workflowID, workflowData)
		assert.NoError(t, err, "CreateWorkflow should succeed")
		mockEtcd.AssertExpectations(t)
	})

	// Test GetWorkflow (new method using GetValue)
	t.Run("GetWorkflow", func(t *testing.T) {
		workflowJSON, _ := json.Marshal(workflowData)
		path := "/config/horizon/workflows/" + workflowID

		mockEtcd.On("GetValue", path).Return(string(workflowJSON), nil).Once()

		retrieved, err := workflowEtcd.GetWorkflow(workflowID)
		require.NoError(t, err, "GetWorkflow should succeed")
		assert.Equal(t, workflowID, retrieved["id"], "Workflow ID should match")
		assert.Equal(t, "PENDING", retrieved["state"], "Workflow state should match")
		mockEtcd.AssertExpectations(t)
	})

	// Test UpdateWorkflow (same as numerix UpdateConfig)
	t.Run("UpdateWorkflow", func(t *testing.T) {
		workflowData["state"] = "RUNNING"
		workflowData["updatedAt"] = time.Now().Format(time.RFC3339)

		// Encode the same way as UpdateWorkflow does (with SetEscapeHTML(false) and TrimSpace)
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)
		encoder.Encode(workflowData)
		workflowJSON := bytes.TrimSpace(buf.Bytes())

		path := "/config/horizon/workflows/" + workflowID

		mockEtcd.On("SetValue", path, mock.AnythingOfType("string")).Return(nil).Once()
		mockEtcd.On("GetValue", path).Return(string(workflowJSON), nil).Once()

		err := workflowEtcd.UpdateWorkflow(workflowID, workflowData)
		assert.NoError(t, err, "UpdateWorkflow should succeed")

		// Verify update
		retrieved, err := workflowEtcd.GetWorkflow(workflowID)
		require.NoError(t, err)
		assert.Equal(t, "RUNNING", retrieved["state"], "Workflow state should be updated")
		mockEtcd.AssertExpectations(t)
	})

	// Test DeleteWorkflow (same as numerix DeleteConfig)
	t.Run("DeleteWorkflow", func(t *testing.T) {
		path := "/config/horizon/workflows/" + workflowID

		mockEtcd.On("DeleteNode", path).Return(nil).Once()
		mockEtcd.On("GetValue", path).Return("", assert.AnError).Once()

		err := workflowEtcd.DeleteWorkflow(workflowID)
		assert.NoError(t, err, "DeleteWorkflow should succeed")

		// Verify deletion
		_, err = workflowEtcd.GetWorkflow(workflowID)
		assert.Error(t, err, "GetWorkflow should fail after deletion")
		mockEtcd.AssertExpectations(t)
	})
}
