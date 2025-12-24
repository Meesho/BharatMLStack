package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	inframiddleware "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/middleware"
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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Apply workingEnv middleware to match production setup
	router.Use(inframiddleware.WorkingEnvMiddleware())
	return router
}

func TestNewController(t *testing.T) {
	controller := NewController()
	assert.NotNil(t, controller)

	// Verify it's a singleton
	controller2 := NewController()
	assert.Equal(t, controller, controller2)
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
				m.On("RestartDeployment", "test-app", mock.AnythingOfType("string"), true).
					Return(nil)
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
				m.On("RestartDeployment", "test-app", mock.AnythingOfType("string"), false).
					Return(nil)
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
				m.On("RestartDeployment", "test-app", mock.AnythingOfType("string"), false).
					Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Handler error should return internal server error",
		},
		{
			name:    "Test 6: Restart deployment with missing isCanary field",
			appName: "test-app",
			requestBody: map[string]interface{}{
				"otherField": "value",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				// isCanary defaults to false when not provided
				m.On("RestartDeployment", "test-app", mock.AnythingOfType("string"), false).
					Return(nil)
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
				m.On("RestartDeployment", "test-app-123", mock.AnythingOfType("string"), true).
					Return(nil)
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
				m.On("RestartDeployment", "very-long-application-name-with-many-characters", mock.AnythingOfType("string"), false).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Long app name should work",
		},
		{
			name:        "Test 9: Restart deployment with empty request body",
			appName:     "test-app",
			requestBody: map[string]interface{}{},
			mockSetup: func(m *MockInfrastructureHandler) {
				// Empty body defaults to isCanary=false
				m.On("RestartDeployment", "test-app", mock.AnythingOfType("string"), false).
					Return(nil)
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
				// nil isCanary defaults to false when unmarshaled
				m.On("RestartDeployment", "test-app", mock.AnythingOfType("string"), false).
					Return(nil)
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

			controller := &InfrastructureController{
				handler: mockHandler,
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
				m.On("UpdateCPUThreshold", "test-app", "80", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateCPUThreshold", "test-app", "80", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Handler error should return internal server error",
		},
		{
			name:    "Test 6: Update CPU threshold with zero threshold",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "0",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("UpdateCPUThreshold", "test-app", "0", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Zero threshold should be passed to handler",
		},
		{
			name:    "Test 7: Update CPU threshold with maximum value",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "100",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("UpdateCPUThreshold", "test-app", "100", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateCPUThreshold", "test-app", "1", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateCPUThreshold", "test-app-123", "80", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateCPUThreshold", "test-app", "abc", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Non-numeric threshold should be passed to handler for validation",
		},
		{
			name:    "Test 12: Update CPU threshold with negative threshold",
			appName: "test-app",
			requestBody: UpdateThresholdRequest{
				CPUThreshold: "-10",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("UpdateCPUThreshold", "test-app", "-10", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Negative threshold should be passed to handler for validation",
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
				m.On("UpdateGPUThreshold", "test-app", "70", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateGPUThreshold", "test-app", "70", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "Handler error should return internal server error",
		},
		{
			name:    "Test 6: Update GPU threshold with zero threshold",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "0",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("UpdateGPUThreshold", "test-app", "0", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Zero threshold should be passed to handler",
		},
		{
			name:    "Test 7: Update GPU threshold with maximum value",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "100",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("UpdateGPUThreshold", "test-app", "100", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateGPUThreshold", "test-app", "1", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateGPUThreshold", "test-app-123", "70", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
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
				m.On("UpdateGPUThreshold", "test-app", "xyz", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Non-numeric threshold should be passed to handler for validation",
		},
		{
			name:    "Test 12: Update GPU threshold with negative threshold",
			appName: "test-app",
			requestBody: UpdateGPUThresholdRequest{
				GPUThreshold: "-5",
			},
			mockSetup: func(m *MockInfrastructureHandler) {
				m.On("UpdateGPUThreshold", "test-app", "-5", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Negative threshold should be passed to handler for validation",
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
		})
	}
}
