package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWorkingEnvMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		workingEnv     string
		expectedStatus int
		description    string
	}{
		{
			name:           "Test 1: Valid workingEnv - gcp_prd",
			workingEnv:     "gcp_prd",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 2: Valid workingEnv - prd",
			workingEnv:     "prd",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 3: Valid workingEnv - dev",
			workingEnv:     "dev",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 4: Missing workingEnv",
			workingEnv:     "",
			expectedStatus: http.StatusBadRequest,
			description:    "Missing workingEnv should return bad request",
		},
		{
			name:           "Test 5: Any workingEnv should be accepted",
			workingEnv:     "invalid_env",
			expectedStatus: http.StatusOK,
			description:    "Any workingEnv should be accepted (environment-agnostic)",
		},
		{
			name:           "Test 5b: stg workingEnv should be accepted",
			workingEnv:     "stg",
			expectedStatus: http.StatusOK,
			description:    "stg workingEnv should be accepted",
		},
		{
			name:           "Test 6: Valid workingEnv - int",
			workingEnv:     "int",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 7: Valid workingEnv - gcp_int",
			workingEnv:     "gcp_int",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 8: Valid workingEnv - gcp_dev",
			workingEnv:     "gcp_dev",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 9: Valid workingEnv - gcp_stg",
			workingEnv:     "gcp_stg",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 10: Valid workingEnv - ftr",
			workingEnv:     "ftr",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
		{
			name:           "Test 11: Valid workingEnv - gcp_ftr",
			workingEnv:     "gcp_ftr",
			expectedStatus: http.StatusOK,
			description:    "Valid workingEnv should pass middleware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(WorkingEnvMiddleware())

			router.GET("/test", func(c *gin.Context) {
				workingEnv, exists := c.Get("workingEnv")
				assert.True(t, exists, "workingEnv should exist in context")
				if tt.workingEnv != "" {
					assert.Equal(t, tt.workingEnv, workingEnv.(string))
				}
				c.JSON(http.StatusOK, gin.H{"workingEnv": workingEnv})
			})

			url := "/test"
			if tt.workingEnv != "" {
				url += "?workingEnv=" + tt.workingEnv
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}

