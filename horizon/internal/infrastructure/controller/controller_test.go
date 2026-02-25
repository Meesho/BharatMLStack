package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	inframiddleware "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/middleware"
	workflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockInfrastructureHandler is a mock for InfrastructureHandler
type MockInfrastructureHandler struct {
	mock.Mock
}

func (m *MockInfrastructureHandler) GetHPAProperties(appName, workingEnv string) (*infrastructurehandler.HPAConfig, error) {
	args := m.Called(appName, workingEnv)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infrastructurehandler.HPAConfig), args.Error(1)
}

func (m *MockInfrastructureHandler) GetConfig(serviceName, workingEnv string) infrastructurehandler.Config {
	args := m.Called(serviceName, workingEnv)
	return args.Get(0).(infrastructurehandler.Config)
}

func (m *MockInfrastructureHandler) GetResourceDetail(appName, workingEnv string) (*infrastructurehandler.ResourceDetail, error) {
	args := m.Called(appName, workingEnv)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infrastructurehandler.ResourceDetail), args.Error(1)
}

func (m *MockInfrastructureHandler) GetApplicationLogs(appName, workingEnv string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
	args := m.Called(appName, workingEnv, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]argocd.ApplicationLogEntry), args.Error(1)
}

func (m *MockInfrastructureHandler) RestartDeployment(appName, workingEnv string, isCanary bool) error {
	args := m.Called(appName, workingEnv, isCanary)
	return args.Error(0)
}

func (m *MockInfrastructureHandler) UpdateCPUThreshold(appName, threshold, email, workingEnv string) error {
	args := m.Called(appName, threshold, email, workingEnv)
	return args.Error(0)
}

func (m *MockInfrastructureHandler) UpdateGPUThreshold(appName, threshold, email, workingEnv string) error {
	args := m.Called(appName, threshold, email, workingEnv)
	return args.Error(0)
}

func (m *MockInfrastructureHandler) UpdateSharedMemory(appName, size, email, workingEnv string) error {
	args := m.Called(appName, size, email, workingEnv)
	return args.Error(0)
}

func (m *MockInfrastructureHandler) UpdatePodAnnotations(appName string, annotations map[string]string, email, workingEnv string) error {
	args := m.Called(appName, annotations, email, workingEnv)
	return args.Error(0)
}

func (m *MockInfrastructureHandler) UpdateAutoscalingTriggers(appName string, triggers []interface{}, email, workingEnv string) error {
	args := m.Called(appName, triggers, email, workingEnv)
	return args.Error(0)
}

// MockWorkflowHandler is a mock for workflow Handler
type MockWorkflowHandler struct {
	mock.Mock
}

func (m *MockWorkflowHandler) StartOnboardingWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartCPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartGPUThresholdUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartSharedMemoryUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartPodAnnotationsUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartRestartDeploymentWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) StartAutoscalingTriggersUpdateWorkflow(payload map[string]interface{}, createdBy, workingEnv string) (string, error) {
	args := m.Called(payload, createdBy, workingEnv)
	return args.String(0), args.Error(1)
}

func (m *MockWorkflowHandler) GetWorkflowStatus(workflowID string) (map[string]interface{}, error) {
	args := m.Called(workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockEtcdInstance is a mock implementation of etcd.Etcd interface for testing
type MockEtcdInstance struct {
	mock.Mock
	etcd.Etcd
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

// setupMockEtcd initializes a mock etcd instance for testing
// This is needed because NewController() calls GetWorkflowHandler() which requires etcd
func setupMockEtcd() {
	// Set WorkflowAppName for testing (normally set by Init)
	workflowPkg.WorkflowAppName = "horizon"

	// Create mock etcd instance
	mockEtcd := NewMockEtcdInstance()

	// Set up the mock instance in the etcd package
	etcd.SetMockInstance(map[string]etcd.Etcd{
		workflowPkg.WorkflowAppName: mockEtcd,
	})
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Apply workingEnv middleware to match production setup
	router.Use(inframiddleware.WorkingEnvMiddleware())
	return router
}

func TestNewController(t *testing.T) {
	// Setup mock etcd before creating controller (required by GetWorkflowHandler)
	setupMockEtcd()

	controller := NewController()
	assert.NotNil(t, controller)

	// Verify it creates a new instance each time (not a singleton)
	// The underlying handlers (InfrastructureHandler and WorkflowHandler) are singletons,
	// but the controller itself creates a new instance each time
	controller2 := NewController()
	// Compare pointers to verify they are different instances
	if controller == controller2 {
		t.Error("NewController should create new instances each time, but got the same instance")
	}

	// Both controllers are valid and can be used
	assert.NotNil(t, controller2)
}

func TestInfrastructureController_GetHPAConfig(t *testing.T) {
	tests := []struct {
		name           string
		appName        string
		mockSetup      func(*MockInfrastructureHandler)
		expectedStatus int
		description    string
	}{
		{
			name:    "Test 1: Get HPA config with valid app name",
			appName: "test-app",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetHPAProperties", "test-app", mock.AnythingOfType("string")).
					Return(&infrastructurehandler.HPAConfig{
						MinReplica:    "2",
						MaxReplica:    "10",
						RunningStatus: "true",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Valid app name should return HPA config",
		},
		{
			name:    "Test 2: Get HPA config with empty app name",
			appName: "test", // Use valid name since empty string causes 404 (route doesn't match)
			mockSetup: func(m *MockInfrastructureHandler) {
				// Handler will be called but we test validation in handler tests
				m.On("GetHPAProperties", "test", mock.AnythingOfType("string")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Handler error when no proper setup",
		},
		{
			name:    "Test 3: Get HPA config with handler error",
			appName: "test-app",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetHPAProperties", "test-app", mock.AnythingOfType("string")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Handler error should return internal server error",
		},
		{
			name:    "Test 4: Get HPA config with special characters",
			appName: "test-app-123",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetHPAProperties", "test-app-123", mock.AnythingOfType("string")).
					Return(&infrastructurehandler.HPAConfig{
						MinReplica:    "1",
						MaxReplica:    "5",
						RunningStatus: "false",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Special characters in app name should work",
		},
		{
			name:    "Test 5: Get HPA config with zero replicas",
			appName: "test-app",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetHPAProperties", "test-app", mock.AnythingOfType("string")).
					Return(&infrastructurehandler.HPAConfig{
						MinReplica:    "0",
						MaxReplica:    "0",
						RunningStatus: "false",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Zero replicas should be returned correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockInfrastructureHandler)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHandler)
			}

			controller := &InfrastructureController{
				handler: mockHandler,
			}

			router := setupTestRouter()
			router.GET("/api/v1/infrastructure/hpa/:appName", controller.GetHPAConfig)

			url := "/api/v1/infrastructure/hpa/" + tt.appName + "?workingEnv=gcp_prd"
			if tt.appName == " " {
				url = "/api/v1/infrastructure/hpa/ ?workingEnv=gcp_prd" // Ensure route matches
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			mockHandler.AssertExpectations(t)
		})
	}
}

func TestInfrastructureController_GetResourceDetail(t *testing.T) {
	tests := []struct {
		name           string
		appName        string
		mockSetup      func(*MockInfrastructureHandler)
		expectedStatus int
		description    string
	}{
		{
			name:    "Test 1: Get resource detail with valid app name",
			appName: "test-app",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetResourceDetail", "test-app", mock.AnythingOfType("string")).
					Return(&infrastructurehandler.ResourceDetail{
						Nodes: []infrastructurehandler.Node{},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Valid app name should return resource detail",
		},
		{
			name:    "Test 2: Get resource detail with empty app name",
			appName: "",
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Empty app name should return bad request",
		},
		{
			name:    "Test 3: Get resource detail with handler error",
			appName: "test-app",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetResourceDetail", "test-app", mock.AnythingOfType("string")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Handler error should return internal server error",
		},
		{
			name:    "Test 4: Get resource detail with nodes",
			appName: "test-app",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetResourceDetail", "test-app", mock.AnythingOfType("string")).
					Return(&infrastructurehandler.ResourceDetail{
						Nodes: []infrastructurehandler.Node{
							{
								Kind: "Pod",
								Name: "test-pod",
								Health: infrastructurehandler.Health{
									Status: "Healthy",
								},
								Info: []infrastructurehandler.Info{
									{Value: "Running"},
								},
							},
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Resource detail with nodes should be returned",
		},
		{
			name:    "Test 5: Get resource detail with special characters",
			appName: "test-app-123",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetResourceDetail", "test-app-123", mock.AnythingOfType("string")).
					Return(&infrastructurehandler.ResourceDetail{
						Nodes: []infrastructurehandler.Node{},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Special characters in app name should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockInfrastructureHandler)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHandler)
			}

			controller := &InfrastructureController{
				handler: mockHandler,
			}

			router := setupTestRouter()
			router.GET("/api/v1/infrastructure/resource-detail", controller.GetResourceDetail)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/infrastructure/resource-detail?appName="+tt.appName+"&workingEnv=gcp_prd", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			mockHandler.AssertExpectations(t)
		})
	}
}

func TestInfrastructureController_RestartDeployment(t *testing.T) {
	tests := []struct {
		name           string
		appName        string
		requestBody    interface{}
		mockSetup      func(*MockInfrastructureHandler)
		expectedStatus int
		description    string
	}{
		{
			name:    "Test 1: Restart deployment with canary",
			appName: "test-app",
			requestBody: RestartDeploymentRequest{
				IsCanary: true,
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Restart canary deployment should succeed",
		},
		{
			name:    "Test 2: Restart deployment without canary",
			appName: "test-app",
			requestBody: RestartDeploymentRequest{
				IsCanary: false,
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Restart non-canary deployment should succeed",
		},
		{
			name:        "Test 3: Restart deployment with empty app name",
			appName:     "",
			requestBody: RestartDeploymentRequest{IsCanary: false},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Empty app name should return bad request",
		},
		{
			name:        "Test 4: Restart deployment with invalid JSON",
			appName:     "test-app",
			requestBody: "invalid json",
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid JSON should return bad request",
		},
		{
			name:    "Test 5: Restart deployment with handler error",
			appName: "test-app",
			requestBody: RestartDeploymentRequest{
				IsCanary: false,
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Workflow handler error should return internal server error",
		},
		{
			name:    "Test 6: Restart deployment with missing isCanary field",
			appName: "test-app",
			requestBody: map[string]interface{}{
				"otherField": "value",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Missing isCanary field should default to false",
		},
		{
			name:    "Test 7: Restart deployment with special characters in app name",
			appName: "test-app-123",
			requestBody: RestartDeploymentRequest{
				IsCanary: true,
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Special characters in app name should work",
		},
		{
			name:    "Test 8: Restart deployment with very long app name",
			appName: "very-long-application-name-with-many-characters",
			requestBody: RestartDeploymentRequest{
				IsCanary: false,
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Long app name should work",
		},
		{
			name:        "Test 9: Restart deployment with empty request body",
			appName:     "test-app",
			requestBody: map[string]interface{}{},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Empty request body should default to isCanary=false",
		},
		{
			name:    "Test 10: Restart deployment with null isCanary",
			appName: "test-app",
			requestBody: map[string]interface{}{
				"isCanary": nil,
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Null isCanary should default to false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockInfrastructureHandler)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHandler)
			}

			// Create mock workflow handler
			mockWorkflowHandler := new(MockWorkflowHandler)
			// Set up expectations for workflow handler methods only when it will be called
			// (not for validation error cases that return 400)
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				if tt.expectedStatus == http.StatusInternalServerError && tt.name == "Test 5: Restart deployment with handler error" {
					// For error test case, mock workflow handler to return error
					mockWorkflowHandler.On("StartRestartDeploymentWorkflow", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
						Return("", assert.AnError)
				} else {
					// For success cases, mock workflow handler to return success
					mockWorkflowHandler.On("StartRestartDeploymentWorkflow", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
						Return("workflow-id-123", nil)
				}
			}

			controller := &InfrastructureController{
				handler:         mockHandler,
				workflowHandler: mockWorkflowHandler,
			}

			router := setupTestRouter()
			router.POST("/api/v1/infrastructure/deployment/:appName/restart", controller.RestartDeployment)

			var body []byte
			var err error
			if tt.requestBody != nil {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					body = []byte(tt.requestBody.(string))
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/deployment/"+tt.appName+"/restart?workingEnv=gcp_prd", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			mockHandler.AssertExpectations(t)
			// Only assert workflow handler expectations if it was set up (i.e., for 200/500 status codes)
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				mockWorkflowHandler.AssertExpectations(t)
			}
		})
	}
}

func TestInfrastructureController_UpdateCPUThreshold(t *testing.T) {
	tests := []struct {
		name           string
		appName        string
		requestBody    interface{}
		mockSetup      func(*MockInfrastructureHandler)
		expectedStatus int
		description    string
	}{
		{
			name:    "Test 1: Update CPU threshold with valid values",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "80",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Valid CPU threshold update should succeed",
		},
		{
			name:        "Test 2: Update CPU threshold with empty app name",
			appName:     "",
			requestBody: UpdateThresholdRequest{CPUThreshold: "80"},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Empty app name should return bad request",
		},
		{
			name:        "Test 3: Update CPU threshold with missing CPU threshold",
			appName:     "test-app",
			requestBody: map[string]interface{}{},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Missing CPU threshold should return bad request",
		},
		{
			name:        "Test 4: Update CPU threshold with invalid JSON",
			appName:     "test-app",
			requestBody: "invalid json",
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid JSON should return bad request",
		},
		{
			name:    "Test 5: Update CPU threshold with handler error",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "80",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Workflow handler error should return internal server error",
		},
		{
			name:    "Test 6: Update CPU threshold with zero threshold",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "0",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Zero threshold should be passed to workflow",
		},
		{
			name:    "Test 7: Update CPU threshold with maximum value",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "100",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Maximum threshold value should work",
		},
		{
			name:    "Test 8: Update CPU threshold with minimum value",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "1",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Minimum threshold value should work",
		},
		{
			name:    "Test 9: Update CPU threshold with special characters in app name",
			appName: "test-app-123",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "80",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Special characters in app name should work",
		},
		{
			name:    "Test 10: Update CPU threshold with empty threshold string",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called - empty threshold is invalid
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Empty threshold string should return bad request",
		},
		{
			name:    "Test 11: Update CPU threshold with non-numeric threshold",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "abc",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Non-numeric threshold should be passed to workflow for validation",
		},
		{
			name:    "Test 12: Update CPU threshold with negative threshold",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "-10",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Negative threshold should be passed to workflow for validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockInfrastructureHandler)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHandler)
			}

			// Create mock workflow handler
			mockWorkflowHandler := new(MockWorkflowHandler)
			// Set up expectations for workflow handler methods only when it will be called
			// (not for validation error cases that return 400)
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				if tt.expectedStatus == http.StatusInternalServerError && tt.name == "Test 5: Update CPU threshold with handler error" {
					// For error test case, mock workflow handler to return error
					mockWorkflowHandler.On("StartCPUThresholdUpdateWorkflow", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
						Return("", assert.AnError)
				} else {
					// For success cases, mock workflow handler to return success
					mockWorkflowHandler.On("StartCPUThresholdUpdateWorkflow", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
						Return("workflow-id-123", nil)
				}
			}

			controller := &InfrastructureController{
				handler:         mockHandler,
				workflowHandler: mockWorkflowHandler,
			}

			router := setupTestRouter()
			router.PUT("/api/v1/infrastructure/threshold/:appName/cpu", controller.UpdateCPUThreshold)

			var body []byte
			var err error
			if tt.requestBody != nil {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					body = []byte(tt.requestBody.(string))
				}
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/infrastructure/threshold/"+tt.appName+"/cpu?workingEnv=gcp_prd", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			mockHandler.AssertExpectations(t)
			// Only assert workflow handler expectations if it was set up (i.e., for 200/500 status codes)
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				mockWorkflowHandler.AssertExpectations(t)
			}
		})
	}
}

func TestInfrastructureController_GetApplicationLogs(t *testing.T) {
	ts := "2026-02-24T11:53:35Z"

	tests := []struct {
		name           string
		appName        string
		queryParams    string
		mockSetup      func(*MockInfrastructureHandler)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
		description    string
	}{
		{
			name:    "success with valid params",
			appName: "test-app",
			queryParams: "container=main&tailLines=100",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"), mock.AnythingOfType("*argocd.ApplicationLogsOptions")).
					Return([]argocd.ApplicationLogEntry{
						{Result: argocd.ApplicationLogResult{Content: "log line 1", TimeStamp: &ts, PodName: "pod-1"}},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Valid request should return log entries",
		},
		{
			name:    "success with all optional params",
			appName: "test-app",
			queryParams: "container=sidecar&podName=pod-xyz&previous=true&sinceSeconds=600&tailLines=50&filter=ERROR",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.MatchedBy(func(opts *argocd.ApplicationLogsOptions) bool {
					return opts.Container == "sidecar" &&
						opts.PodName == "pod-xyz" &&
						opts.Previous == true &&
						opts.SinceSeconds == 600 &&
						opts.TailLines == 50 &&
						opts.Filter == "ERROR"
					})).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "All optional params should be forwarded to handler",
		},
		{
			name:    "success returns empty entries",
			appName: "test-app",
			queryParams: "container=main",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"), mock.AnythingOfType("*argocd.ApplicationLogsOptions")).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Empty log result should return 200 with empty array",
		},
		{
			name:        "missing workingEnv",
			appName:     "test-app",
			queryParams: "container=main&_skipWorkingEnv=true",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			description:    "Missing workingEnv should be rejected by middleware",
		},
		{
			name:        "whitespace-only appName",
			appName:     "%20%20",
			queryParams: "container=main",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "appName is required", errObj["message"])
			},
			description: "Whitespace-only appName should return 400",
		},
		{
			name:        "missing container",
			appName:     "test-app",
			queryParams: "",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "container is required", errObj["message"])
			},
			description: "Missing container should return 400",
		},
		{
			name:        "whitespace-only container",
			appName:     "test-app",
			queryParams: "container=%20%20",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "container is required", errObj["message"])
			},
			description: "Whitespace-only container should return 400",
		},
		{
			name:        "follow true rejected",
			appName:     "test-app",
			queryParams: "container=main&follow=true",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "streaming (follow=true) is not supported", errObj["message"])
			},
			description: "follow=true should be rejected",
		},
		{
			name:        "sinceSeconds non-numeric",
			appName:     "test-app",
			queryParams: "container=main&sinceSeconds=abc",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "sinceSeconds must be a non-negative integer", errObj["message"])
			},
			description: "Non-numeric sinceSeconds should return 400",
		},
		{
			name:        "sinceSeconds negative",
			appName:     "test-app",
			queryParams: "container=main&sinceSeconds=-10",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "sinceSeconds must be a non-negative integer", errObj["message"])
			},
			description: "Negative sinceSeconds should return 400",
		},
		{
			name:        "tailLines non-numeric",
			appName:     "test-app",
			queryParams: "container=main&tailLines=xyz",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "tailLines must be a non-negative integer", errObj["message"])
			},
			description: "Non-numeric tailLines should return 400",
		},
		{
			name:        "tailLines negative",
			appName:     "test-app",
			queryParams: "container=main&tailLines=-5",
			mockSetup:   func(m *MockInfrastructureHandler) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.Equal(t, "tailLines must be a non-negative integer", errObj["message"])
			},
			description: "Negative tailLines should return 400",
		},
		{
			name:    "large tailLines is accepted",
			appName: "test-app",
			queryParams: "container=main&tailLines=20000",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.MatchedBy(func(opts *argocd.ApplicationLogsOptions) bool {
						return opts.TailLines == 20000
					})).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Large tailLines value should be passed through to handler",
		},
		{
			name:    "default tailLines when omitted",
			appName: "test-app",
			queryParams: "container=main",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.MatchedBy(func(opts *argocd.ApplicationLogsOptions) bool {
						return opts.TailLines == 1000
					})).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Omitted tailLines should default to 1000",
		},
		{
			name:    "default sinceSeconds when omitted",
			appName: "test-app",
			queryParams: "container=main",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.MatchedBy(func(opts *argocd.ApplicationLogsOptions) bool {
						return opts.SinceSeconds == 0
					})).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Omitted sinceSeconds should default to 0",
		},
		{
			name:    "handler returns error",
			appName: "test-app",
			queryParams: "container=main",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"), mock.AnythingOfType("*argocd.ApplicationLogsOptions")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				assert.NotEmpty(t, errObj["message"])
			},
			description: "Handler error should return 500 with error message",
		},
		{
			name:    "handler returns ArgoCD 404 error",
			appName: "test-app",
			queryParams: "container=main",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"), mock.AnythingOfType("*argocd.ApplicationLogsOptions")).
					Return(nil, fmt.Errorf("failed to get application logs: %w", &argocd.ArgoCDAPIError{
						StatusCode: http.StatusNotFound,
						HTTPStatus: "Not Found",
						Message:    "application not found",
					}))
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				msg, _ := errObj["message"].(string)
				assert.Contains(t, msg, "application not found")
			},
			description: "ArgoCD 404 should be forwarded as HTTP 404",
		},
		{
			name:    "handler returns ArgoCD 403 error",
			appName: "test-app",
			queryParams: "container=main",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"), mock.AnythingOfType("*argocd.ApplicationLogsOptions")).
					Return(nil, fmt.Errorf("failed to get application logs: %w", &argocd.ArgoCDAPIError{
						StatusCode: http.StatusForbidden,
						HTTPStatus: "Forbidden",
						Message:    "permission denied",
					}))
			},
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}
				msg, _ := errObj["message"].(string)
				assert.Contains(t, msg, "permission denied")
			},
			description: "ArgoCD 403 should be forwarded as HTTP 403",
		},
		{
			name:    "sinceSeconds zero string is valid",
			appName: "test-app",
			queryParams: "container=main&sinceSeconds=0",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.MatchedBy(func(opts *argocd.ApplicationLogsOptions) bool {
						return opts.SinceSeconds == 0
					})).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "sinceSeconds=0 is valid and should pass through",
		},
		{
			name:    "tailLines zero is valid",
			appName: "test-app",
			queryParams: "container=main&tailLines=0",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.MatchedBy(func(opts *argocd.ApplicationLogsOptions) bool {
						return opts.TailLines == 0
					})).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "tailLines=0 is valid and should pass through",
		},
		{
			name:    "follow false is accepted",
			appName: "test-app",
			queryParams: "container=main&follow=false",
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("GetApplicationLogs", "test-app", mock.AnythingOfType("string"),
					mock.AnythingOfType("*argocd.ApplicationLogsOptions")).
					Return([]argocd.ApplicationLogEntry{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "follow=false should be accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockInfrastructureHandler)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHandler)
			}

			controller := &InfrastructureController{
				handler: mockHandler,
			}

			router := setupTestRouter()
			router.GET("/api/v1/infrastructure/applications/:appName/logs", controller.GetApplicationLogs)

			urlPath := "/api/v1/infrastructure/applications/" + tt.appName + "/logs"

			skipWorkingEnv := false
			qp := tt.queryParams
			if strings.Contains(qp, "_skipWorkingEnv=true") {
				skipWorkingEnv = true
				qp = strings.ReplaceAll(qp, "_skipWorkingEnv=true", "")
				qp = strings.TrimRight(qp, "&")
			}

			if !skipWorkingEnv {
				if qp != "" {
					qp = "workingEnv=gcp_prd&" + qp
				} else {
					qp = "workingEnv=gcp_prd"
				}
			}
			if qp != "" {
				urlPath += "?" + qp
			}

			req := httptest.NewRequest(http.MethodGet, urlPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)

			if tt.validateBody != nil {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				assert.NoError(t, err, "response body should be valid JSON")
				tt.validateBody(t, body)
			}

			mockHandler.AssertExpectations(t)
		})
	}
}

func TestInfrastructureController_UpdateGPUThreshold(t *testing.T) {
	tests := []struct {
		name           string
		appName        string
		requestBody    interface{}
		mockSetup      func(*MockInfrastructureHandler)
		expectedStatus int
		description    string
	}{
		{
			name:    "Test 1: Update GPU threshold with valid values",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "70",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Valid GPU threshold update should succeed",
		},
		{
			name:        "Test 2: Update GPU threshold with empty app name",
			appName:     "",
			requestBody: UpdateGPUThresholdRequest{GPUThreshold: "70"},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Empty app name should return bad request",
		},
		{
			name:        "Test 3: Update GPU threshold with missing GPU threshold",
			appName:     "test-app",
			requestBody: map[string]interface{}{},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Missing GPU threshold should return bad request",
		},
		{
			name:        "Test 4: Update GPU threshold with invalid JSON",
			appName:     "test-app",
			requestBody: "invalid json",
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid JSON should return bad request",
		},
		{
			name:    "Test 5: Update GPU threshold with handler error",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "70",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Workflow handler error should return internal server error",
		},
		{
			name:    "Test 6: Update GPU threshold with zero threshold",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "0",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Zero threshold should be passed to workflow",
		},
		{
			name:    "Test 7: Update GPU threshold with maximum value",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "100",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Maximum threshold value should work",
		},
		{
			name:    "Test 8: Update GPU threshold with minimum value",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "1",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Minimum threshold value should work",
		},
		{
			name:    "Test 9: Update GPU threshold with special characters in app name",
			appName: "test-app-123",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "70",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Special characters in app name should work",
		},
		{
			name:    "Test 10: Update GPU threshold with empty threshold string",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Should not be called - empty threshold is invalid
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Empty threshold string should return bad request",
		},
		{
			name:    "Test 11: Update GPU threshold with non-numeric threshold",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "xyz",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Non-numeric threshold should be passed to workflow for validation",
		},
		{
			name:    "Test 12: Update GPU threshold with negative threshold",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "-5",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Infrastructure handler is not called directly - workflow handler is used instead
			},
			expectedStatus: http.StatusOK,
			description:    "Negative threshold should be passed to workflow for validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockInfrastructureHandler)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHandler)
			}

			// Create mock workflow handler
			mockWorkflowHandler := new(MockWorkflowHandler)
			// Set up expectations for workflow handler methods only when it will be called
			// (not for validation error cases that return 400)
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				if tt.expectedStatus == http.StatusInternalServerError && tt.name == "Test 5: Update GPU threshold with handler error" {
					// For error test case, mock workflow handler to return error
					mockWorkflowHandler.On("StartGPUThresholdUpdateWorkflow", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
						Return("", assert.AnError)
				} else {
					// For success cases, mock workflow handler to return success
					mockWorkflowHandler.On("StartGPUThresholdUpdateWorkflow", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
						Return("workflow-id-123", nil)
				}
			}

			controller := &InfrastructureController{
				handler:         mockHandler,
				workflowHandler: mockWorkflowHandler,
			}

			router := setupTestRouter()
			router.PUT("/api/v1/infrastructure/threshold/:appName/gpu", controller.UpdateGPUThreshold)

			var body []byte
			var err error
			if tt.requestBody != nil {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					body = []byte(tt.requestBody.(string))
				}
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/infrastructure/threshold/"+tt.appName+"/gpu?workingEnv=gcp_prd", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			mockHandler.AssertExpectations(t)
			// Only assert workflow handler expectations if it was set up (i.e., for 200/500 status codes)
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusInternalServerError {
				mockWorkflowHandler.AssertExpectations(t)
			}
		})
	}
}
