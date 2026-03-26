package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBasePath(t *testing.T) {
	tests := []struct {
		name         string
		bu           string
		team         string
		workingEnv   string
		appName      string
		expectEmpty  bool
		expectedPath string
		description  string
	}{
		{
			name:         "Test 1: Valid path for gcp_prd",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "gcp_prd/deployables/test-app",
			description:  "Should generate correct path for gcp_prd",
		},
		{
			name:         "Test 2: Valid path for gcp_int",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_int",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "gcp_int/deployables/test-app",
			description:  "Should generate correct path for gcp_int",
		},
		{
			name:         "Test 3: Valid path for gcp_dev",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_dev",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "gcp_dev/deployables/test-app",
			description:  "Should generate correct path for gcp_dev",
		},
		{
			name:         "Test 4: Valid path for any environment",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "unknown_env",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "unknown_env/deployables/test-app",
			description:  "Any environment should generate path",
		},
		{
			name:        "Test 5: Empty workingEnv",
			bu:          "datascience",
			team:        "ml-platform",
			workingEnv:  "",
			appName:     "test-app",
			expectEmpty: true,
			description: "Empty workingEnv should return empty path",
		},
		{
			name:        "Test 6: Empty app name",
			bu:          "datascience",
			team:        "ml-platform",
			workingEnv:  "gcp_prd",
			appName:     "",
			expectEmpty: true,
			description: "Empty app name should return empty path",
		},
		{
			name:         "Test 7: App name with special characters",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_prd",
			appName:      "test-app-123",
			expectEmpty:  false,
			expectedPath: "gcp_prd/deployables/test-app-123",
			description:  "Special characters in app name should be preserved",
		},
		{
			name:         "Test 8: BU and team are ignored (vendor-agnostic)",
			bu:           "any-bu",
			team:         "any-team",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "gcp_prd/deployables/test-app",
			description:  "BU and team parameters are ignored in current implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBasePath(tt.bu, tt.team, tt.workingEnv, tt.appName)
			if tt.expectEmpty {
				assert.Empty(t, result, tt.description)
			} else {
				assert.NotEmpty(t, result, tt.description)
				if tt.expectedPath != "" {
					assert.Equal(t, tt.expectedPath, result, tt.description)
				}
			}
		})
	}
}

// Note: Tests for EnvToBranch, BU_NORMALIZED, TEAM_NORMALIZED, and WORKING_ENV_MAP
// have been removed as these mappings were removed during refactoring to make
// the codebase vendor-agnostic. The current implementation uses GetEnvConfig()
// which derives values from environment variables rather than hardcoded mappings.

