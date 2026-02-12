package config

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	http2 "github.com/Meesho/go-core/v2/api/http"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name             string
	reqBody          interface{}
	headers          map[string]string
	configContent    string
	expectedCode     int
	expectedResponse string
}

func TestConfigValidator(t *testing.T) {
	testCases := []testCase{
		{
			name:    "Missing Headers",
			reqBody: map[string]interface{}{"key1": "value1"},
			headers: map[string]string{
				http2.HeaderContentType: "application/json",
			},
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "Headers 'environment' and 'deployable-name' are required",
		},
		{
			name:    "Invalid JSON",
			reqBody: "invalid json",
			headers: map[string]string{
				http2.HeaderContentType: "application/json",
				HeaderEnvironment:       "dev",
				HeaderDeployableName:    "app",
			},
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "Invalid JSON body",
		},
		{
			name:    "Missing Keys",
			reqBody: map[string]interface{}{"key1": "value1"},
			headers: map[string]string{
				http2.HeaderContentType: "application/json",
				HeaderEnvironment:       "dev",
				HeaderDeployableName:    "app",
			},
			configContent: `key4: value4
				key5: value5
				---
				deployable-name: deployable
				key5: override-value5
				key6: value6`,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: `{"mismatched_keys":null,"missing_keys":["key1"]}`,
		},
		{
			name: "Mismatched Keys",
			reqBody: map[string]interface{}{
				"key1": "incorrectValue",
				"key2": "value2",
			},
			headers: map[string]string{
				http2.HeaderContentType: "application/json",
				HeaderEnvironment:       "dev",
				HeaderDeployableName:    "app",
			},
			configContent: `key1: correctValue
				key2: value2
				---
				deployable-name: app
				key1: incorrectValue-1`,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: `{"mismatched_keys":["key1"],"missing_keys":null}`,
		},
		{
			name: "Success",
			reqBody: map[string]interface{}{
				"key1":       "value3",
				"key2":       "value2",
				"nested_key": "nested_value",
			},
			headers: map[string]string{
				http2.HeaderContentType: "application/json",
				HeaderEnvironment:       "dev",
				HeaderDeployableName:    "app",
			},
			configContent: `key1: value1
				key2: value2
				---
				deployable-name: app
				key1: value3
				nested:
				  key: nested_value`,
			expectedCode:     http.StatusOK,
			expectedResponse: `{"message":"Config is valid"}`,
		},
		{
			name: "Failed to Load Static Files",
			headers: map[string]string{
				http2.HeaderContentType: "application/json",
				HeaderEnvironment:       "dev",
				HeaderDeployableName:    "app",
			},
			configContent:    "",
			expectedCode:     http.StatusInternalServerError,
			expectedResponse: "Failed to load static configuration file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			directoryPath := "/tmp/config"
			defer os.RemoveAll(directoryPath)
			r := gin.Default()
			r.POST("/validate", ConfigValidator())

			body, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest("POST", "/validate", bytes.NewBuffer(body))
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			if tc.configContent != "" {
				viper.Reset()
				viper.Set(envConfigLocation, directoryPath)

				os.MkdirAll(directoryPath, os.ModePerm)

				configPath := directoryPath + "/application-dev.yml"
				if err := os.WriteFile(configPath, []byte(strings.ReplaceAll(tc.configContent, "\t", "")), os.ModePerm); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedResponse)
		})
	}
}
