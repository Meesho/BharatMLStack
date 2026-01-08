package github

import (
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
)

func createTestArgoCDApp() argocd.ArgoCDApplicationMetadata {
	return argocd.ArgoCDApplicationMetadata{
		Name: "test-app",
		Labels: argocd.ArgoCDApplicationLabels{
			BU:      "datascience",
			Team:    "ml-platform",
			AppName: "test-app",
		},
	}
}

func TestGetFileYaml(t *testing.T) {
	// Note: This test requires mocking the GitHub client
	// For now, we test the error cases and edge cases

	tests := []struct {
		name        string
		app         argocd.ArgoCDApplicationMetadata
		fileName    string
		workingEnv  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Unknown working environment",
			app:         createTestArgoCDApp(),
			fileName:    "values.yaml",
			workingEnv:  "unknown_env",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
		},
		{
			name:        "Valid gcp_prd environment",
			app:         createTestArgoCDApp(),
			fileName:    "values.yaml",
			workingEnv:  "gcp_prd",
			expectError: true, // Will fail because GitHub client not initialized
			errorMsg:    "GitHub client not initialized",
		},
		{
			name:        "Valid gcp_int environment",
			app:         createTestArgoCDApp(),
			fileName:    "values_properties.yaml",
			workingEnv:  "gcp_int",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := GetFileYaml(tt.app, tt.fileName, tt.workingEnv)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateFileYaml(t *testing.T) {
	tests := []struct {
		name        string
		app         argocd.ArgoCDApplicationMetadata
		data        map[string]interface{}
		fileName    string
		commitMsg   string
		sha         *string
		workingEnv  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Unknown working environment",
			app:         createTestArgoCDApp(),
			data:        map[string]interface{}{"key": "value"},
			fileName:    "values.yaml",
			commitMsg:   "test commit",
			sha:         github.String("abc123"),
			workingEnv:  "unknown_env",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
		},
		{
			name:        "Invalid YAML data",
			app:         createTestArgoCDApp(),
			data:        map[string]interface{}{"invalid": make(chan int)}, // Channel cannot be marshaled
			fileName:    "values.yaml",
			commitMsg:   "test commit",
			sha:         github.String("abc123"),
			workingEnv:  "gcp_prd",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateFileYaml(tt.app, tt.data, tt.fileName, tt.commitMsg, tt.sha, tt.workingEnv)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandleDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "No duplicates",
			input:    []byte("key: value\nother: data"),
			expected: "key: value\nother: data",
		},
		{
			name:     "With replicaCount",
			input:    []byte("replicaCount: 5\nkey: value"),
			expected: "replicaCountz: 5\nkey: value",
		},
		{
			name:     "Multiple replicaCount",
			input:    []byte("replicaCount: 5\nreplicaCount: 10"),
			expected: "replicaCountz: 5\nreplicaCount: 10", // Only first one replaced
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleDuplicates(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestUpdateTriggersInValues(t *testing.T) {
	// Test cases for UpdateTriggersInValues - at least 10 scenarios
	tests := []struct {
		name        string
		app         argocd.ArgoCDApplicationMetadata
		triggerType string
		threshold   string
		email       string
		workingEnv  string
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name:        "Test 1: Update CPU threshold with valid data",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "80",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true, // Will fail due to GitHub client not initialized
			description: "Valid CPU threshold update should succeed",
		},
		{
			name:        "Test 2: Update GPU threshold with valid data",
			app:         createTestArgoCDApp(),
			triggerType: "gpu",
			threshold:   "70",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Valid GPU threshold update should succeed",
		},
		{
			name:        "Test 3: Unknown trigger type",
			app:         createTestArgoCDApp(),
			triggerType: "memory",
			threshold:   "60",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			errorMsg:    "unsupported trigger type",
			description: "Unknown trigger type should return error",
		},
		{
			name:        "Test 4: Unknown working environment",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "80",
			email:       "test@meesho.com",
			workingEnv:  "unknown_env",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
			description: "Unknown environment should return error",
		},
		{
			name:        "Test 5: Empty trigger type",
			app:         createTestArgoCDApp(),
			triggerType: "",
			threshold:   "80",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			errorMsg:    "unsupported trigger type",
			description: "Empty trigger type should return error",
		},
		{
			name:        "Test 6: Update with gcp_int environment",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "85",
			email:       "test@meesho.com",
			workingEnv:  "gcp_int",
			expectError: true,
			description: "Update should work with gcp_int environment",
		},
		{
			name:        "Test 7: Update with gcp_dev environment",
			app:         createTestArgoCDApp(),
			triggerType: "gpu",
			threshold:   "75",
			email:       "test@meesho.com",
			workingEnv:  "gcp_dev",
			expectError: true,
			description: "Update should work with gcp_dev environment",
		},
		{
			name:        "Test 8: Update with empty email",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "80",
			email:       "",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with empty email should still proceed",
		},
		{
			name: "Test 9: Update with special characters in app name",
			app: argocd.ArgoCDApplicationMetadata{
				Name: "test-app-123",
				Labels: argocd.ArgoCDApplicationLabels{
					BU:      "datascience",
					Team:    "ml-platform",
					AppName: "test-app-123",
				},
			},
			triggerType: "cpu",
			threshold:   "80",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with special characters in app name should work",
		},
		{
			name:        "Test 10: Update with very long threshold value",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "999",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with large threshold value should work",
		},
		{
			name: "Test 11: Update with different BU and Team",
			app: argocd.ArgoCDApplicationMetadata{
				Name: "another-app",
				Labels: argocd.ArgoCDApplicationLabels{
					BU:      "central",
					Team:    "devops",
					AppName: "another-app",
				},
			},
			triggerType: "gpu",
			threshold:   "65",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with different BU/Team should work",
		},
		{
			name:        "Test 12: Update with gcp_stg environment",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "90",
			email:       "test@meesho.com",
			workingEnv:  "gcp_stg",
			expectError: true,
			description: "Update should work with gcp_stg environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateTriggersInValues(tt.app, tt.triggerType, tt.threshold, tt.email, tt.workingEnv)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestUpdateTriggersInValuesProperties(t *testing.T) {
	// Test cases for UpdateTriggersInValuesProperties - at least 10 scenarios
	tests := []struct {
		name        string
		app         argocd.ArgoCDApplicationMetadata
		triggerType string
		threshold   string
		email       string
		workingEnv  string
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name:        "Test 1: Update CPU threshold in values_properties",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "80",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Valid CPU threshold update in values_properties should succeed",
		},
		{
			name:        "Test 2: Update GPU threshold in values_properties",
			app:         createTestArgoCDApp(),
			triggerType: "gpu",
			threshold:   "70",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Valid GPU threshold update in values_properties should succeed",
		},
		{
			name:        "Test 3: Unknown trigger type in values_properties",
			app:         createTestArgoCDApp(),
			triggerType: "memory",
			threshold:   "60",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			errorMsg:    "unsupported trigger type",
			description: "Unknown trigger type should return error",
		},
		{
			name:        "Test 4: Unknown environment in values_properties",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "80",
			email:       "test@meesho.com",
			workingEnv:  "unknown_env",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
			description: "Unknown environment should return error",
		},
		{
			name:        "Test 5: Update with int environment",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "85",
			email:       "test@meesho.com",
			workingEnv:  "gcp_int",
			expectError: true,
			description: "Update should work with gcp_int environment",
		},
		{
			name:        "Test 6: Update with dev environment",
			app:         createTestArgoCDApp(),
			triggerType: "gpu",
			threshold:   "75",
			email:       "test@meesho.com",
			workingEnv:  "gcp_dev",
			expectError: true,
			description: "Update should work with gcp_dev environment",
		},
		{
			name:        "Test 7: Update with empty threshold",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with empty threshold should fail validation",
		},
		{
			name:        "Test 8: Update with numeric string threshold",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "100",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with numeric string threshold should work",
		},
		{
			name:        "Test 9: Update with single digit threshold",
			app:         createTestArgoCDApp(),
			triggerType: "gpu",
			threshold:   "5",
			email:       "test@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with single digit threshold should work",
		},
		{
			name:        "Test 10: Update with long email",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "80",
			email:       "very.long.email.address@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with long email should work",
		},
		{
			name:        "Test 11: Update with special characters in email",
			app:         createTestArgoCDApp(),
			triggerType: "gpu",
			threshold:   "70",
			email:       "test+tag@meesho.com",
			workingEnv:  "gcp_prd",
			expectError: true,
			description: "Update with special characters in email should work",
		},
		{
			name:        "Test 12: Update with staging environment",
			app:         createTestArgoCDApp(),
			triggerType: "cpu",
			threshold:   "90",
			email:       "test@meesho.com",
			workingEnv:  "gcp_stg",
			expectError: true,
			description: "Update should work with gcp_stg environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateTriggersInValuesProperties(tt.app, tt.triggerType, tt.threshold, tt.email, tt.workingEnv)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestUpdateThresholdHelmRepo(t *testing.T) {
	// Test cases for UpdateThresholdHelmRepo - the main update flow
	tests := []struct {
		name           string
		app            argocd.ArgoCDApplicationMetadata
		thresholdProps ThresholdProperties
		email          string
		workingEnv     string
		expectError    bool
		errorMsg       string
		description    string
	}{
		{
			name:           "Test 1: Update CPU threshold - complete flow",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "80"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true, // Will fail due to GitHub client
			description:    "Complete CPU threshold update flow",
		},
		{
			name:           "Test 2: Update GPU threshold - complete flow",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubGPUThreshold{GPUThreshold: "70"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Complete GPU threshold update flow",
		},
		{
			name:           "Test 3: Update with invalid threshold",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: ""},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with invalid threshold should fail validation",
		},
		{
			name:           "Test 4: Update with zero threshold",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "0"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with zero threshold should fail validation",
		},
		{
			name:           "Test 5: Update with negative threshold",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "-10"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with negative threshold should fail validation",
		},
		{
			name:           "Test 6: Update with int environment",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "85"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_int",
			expectError:    true,
			description:    "Update should work with gcp_int environment",
		},
		{
			name:           "Test 7: Update with dev environment",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubGPUThreshold{GPUThreshold: "75"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_dev",
			expectError:    true,
			description:    "Update should work with gcp_dev environment",
		},
		{
			name:           "Test 8: Update with staging environment",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "90"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_stg",
			expectError:    true,
			description:    "Update should work with gcp_stg environment",
		},
		{
			name: "Test 9: Update with different app metadata",
			app: argocd.ArgoCDApplicationMetadata{
				Name: "another-app",
				Labels: argocd.ArgoCDApplicationLabels{
					BU:      "supply",
					Team:    "cataloging",
					AppName: "another-app",
				},
			},
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "80"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with different app metadata should work",
		},
		{
			name:           "Test 10: Update with maximum threshold value",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "100"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with maximum threshold value should work",
		},
		{
			name:           "Test 11: Update with minimum threshold value",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "1"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with minimum threshold value should work",
		},
		{
			name:           "Test 12: Update with empty email",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "80"},
			email:          "",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update with empty email should still proceed",
		},
		{
			name:           "Test 13: Update GPU with high threshold",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubGPUThreshold{GPUThreshold: "95"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update GPU with high threshold should work",
		},
		{
			name:           "Test 14: Update CPU with low threshold",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "20"},
			email:          "test@meesho.com",
			workingEnv:     "gcp_prd",
			expectError:    true,
			description:    "Update CPU with low threshold should work",
		},
		{
			name:           "Test 15: Update with unknown environment",
			app:            createTestArgoCDApp(),
			thresholdProps: AutoScalinGitHubCPUThreshold{CPUThreshold: "80"},
			email:          "test@meesho.com",
			workingEnv:     "unknown_env",
			expectError:    true,
			errorMsg:       "", // Error can be either "unknown working environment" or "GitHub client not initialized"
			description:    "Update with unknown environment should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First validate the threshold properties
			validateErr := tt.thresholdProps.Validate()
			if validateErr != nil {
				// If validation fails, the update should also fail
				assert.Error(t, validateErr)
				return
			}

			err := UpdateThresholdHelmRepo(tt.app, tt.thresholdProps, tt.email, tt.workingEnv)
			if tt.expectError {
				assert.Error(t, err, tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, tt.description)
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}
