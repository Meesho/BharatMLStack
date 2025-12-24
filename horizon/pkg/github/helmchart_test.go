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
			name:         "Test 1: Valid path for gcp_prd with datascience BU",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v3/dsci/ml/test-app",
			description:  "Should generate correct path for gcp_prd",
		},
		{
			name:         "Test 2: Valid path for gcp_int",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_int",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v2/dsci/ml/test-app",
			description:  "Should generate correct path for gcp_int",
		},
		{
			name:         "Test 3: Valid path for gcp_dev",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_dev",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v2/dsci/ml/test-app",
			description:  "Should generate correct path for gcp_dev",
		},
		{
			name:         "Test 4: Valid path for gcp_stg",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_stg",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v2/dsci/ml/test-app",
			description:  "Should generate correct path for gcp_stg",
		},
		{
			name:        "Test 5: Unknown environment",
			bu:          "datascience",
			team:        "ml-platform",
			workingEnv:  "unknown_env",
			appName:     "test-app",
			expectEmpty: true,
			description: "Unknown environment should return empty path",
		},
		{
			name:         "Test 6: Unknown BU (should use empty string)",
			bu:           "unknown_bu",
			team:         "ml-platform",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			description:  "Unknown BU should still generate path",
		},
		{
			name:         "Test 7: Unknown team (should use empty string)",
			bu:           "datascience",
			team:         "unknown_team",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			description:  "Unknown team should still generate path",
		},
		{
			name:         "Test 8: Central BU with devops team",
			bu:           "central",
			team:         "devops",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v3/cntr/devop/test-app",
			description:  "Should normalize BU and team correctly",
		},
		{
			name:         "Test 9: Supply BU with supplier-ads team",
			bu:           "supply",
			team:         "supplier-ads",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v3/supl/ads/test-app",
			description:  "Should normalize supply BU and team correctly",
		},
		{
			name:         "Test 10: Demand BU with experience team",
			bu:           "demand",
			team:         "experience",
			workingEnv:   "gcp_int",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v2/dmnd/xp/test-app",
			description:  "Should normalize demand BU and team correctly",
		},
		{
			name:         "Test 11: Data-platform BU with data-science team",
			bu:           "data-platform",
			team:         "data-science",
			workingEnv:   "gcp_prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "values_v3/deng/ds/test-app",
			description:  "Should normalize data-platform BU correctly",
		},
		{
			name:         "Test 12: Empty app name",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_prd",
			appName:      "",
			expectEmpty:  false,
			expectedPath: "values_v3/dsci/ml/",
			description:  "Empty app name should still generate path",
		},
		{
			name:         "Test 13: App name with special characters",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "gcp_prd",
			appName:      "test-app-123",
			expectEmpty:  false,
			expectedPath: "values_v3/dsci/ml/test-app-123",
			description:  "Special characters in app name should be preserved",
		},
		{
			name:         "Test 14: Prd environment (non-GCP)",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "prd",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "charts/dsci/ml/test-app",
			description:  "Non-GCP prd environment should use charts path",
		},
		{
			name:         "Test 15: Int environment (non-GCP)",
			bu:           "datascience",
			team:         "ml-platform",
			workingEnv:   "int",
			appName:      "test-app",
			expectEmpty:  false,
			expectedPath: "charts/dsci/ml/test-app",
			description:  "Non-GCP int environment should use charts path",
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

func TestEnvToBranch(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		expected    string
		shouldExist bool
		description string
	}{
		{"Test 1: gcp_prd to main", "gcp_prd", "main", true, "gcp_prd should map to main"},
		{"Test 2: prd to main", "prd", "main", true, "prd should map to main"},
		{"Test 3: gcp_int to pre-prod", "gcp_int", "pre-prod", true, "gcp_int should map to pre-prod"},
		{"Test 4: int to pre-prod", "int", "pre-prod", true, "int should map to pre-prod"},
		{"Test 5: gcp_dev to develop", "gcp_dev", "develop", true, "gcp_dev should map to develop"},
		{"Test 6: dev to develop", "dev", "develop", true, "dev should map to develop"},
		{"Test 7: gcp_stg to develop", "gcp_stg", "develop", true, "gcp_stg should map to develop"},
		{"Test 8: gcp_ftr to feature", "gcp_ftr", "feature", true, "gcp_ftr should map to feature"},
		{"Test 9: ftr to feature", "ftr", "feature", true, "ftr should map to feature"},
		{"Test 10: Unknown environment", "unknown", "", false, "Unknown environment should not exist"},
		{"Test 11: Empty environment", "", "", false, "Empty environment should not exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := EnvToBranch[tt.env]
			if tt.shouldExist {
				assert.True(t, exists, tt.description)
				assert.Equal(t, tt.expected, result, tt.description)
			} else {
				assert.False(t, exists, tt.description)
			}
		})
	}
}

func TestBU_NORMALIZED(t *testing.T) {
	tests := []struct {
		name     string
		bu       string
		expected string
		exists   bool
	}{
		{"Test 1: central", "central", "cntr", true},
		{"Test 2: supply", "supply", "supl", true},
		{"Test 3: demand", "demand", "dmnd", true},
		{"Test 4: data-platform", "data-platform", "deng", true},
		{"Test 5: dataengg", "dataengg", "deng", true},
		{"Test 6: datascience", "datascience", "dsci", true},
		{"Test 7: farmiso", "farmiso", "farm", true},
		{"Test 8: Unknown BU", "unknown", "", false},
		{"Test 9: Empty BU", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := BU_NORMALIZED[tt.bu]
			if tt.exists {
				assert.True(t, exists)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.False(t, exists)
			}
		})
	}
}

func TestTEAM_NORMALIZED(t *testing.T) {
	tests := []struct {
		name     string
		team     string
		expected string
		exists   bool
	}{
		{"Test 1: ml-platform", "ml-platform", "ml", true},
		{"Test 2: devops", "devops", "devop", true},
		{"Test 3: supplier-ads", "supplier-ads", "ads", true},
		{"Test 4: experience", "experience", "xp", true},
		{"Test 5: Unknown team", "unknown", "", false},
		{"Test 6: Empty team", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := TEAM_NORMALIZED[tt.team]
			if tt.exists {
				assert.True(t, exists)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.False(t, exists)
			}
		})
	}
}

func TestWORKING_ENV_MAP(t *testing.T) {
	envs := []string{"prd", "gcp_prd", "int", "gcp_int", "dev", "gcp_dev", "gcp_stg", "ftr", "gcp_ftr"}
	
	for _, env := range envs {
		t.Run("Test "+env+" environment config", func(t *testing.T) {
			config, exists := WORKING_ENV_MAP[env]
			assert.True(t, exists, "Environment %s should exist in WORKING_ENV_MAP", env)
			if exists {
				assert.NotEmpty(t, config["helm_path"], "helm_path should be set for %s", env)
				assert.NotEmpty(t, config["helm_branch"], "helm_branch should be set for %s", env)
				assert.NotEmpty(t, config["argo_branch"], "argo_branch should be set for %s", env)
			}
		})
	}
	
	// Test unknown environment
	t.Run("Test unknown environment", func(t *testing.T) {
		config, exists := WORKING_ENV_MAP["unknown"]
		assert.False(t, exists)
		assert.Nil(t, config)
	})
}

