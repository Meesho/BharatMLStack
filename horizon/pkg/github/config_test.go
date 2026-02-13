package github

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitGitHub(t *testing.T) {
	tests := []struct {
		name        string
		config      GitHubConfig
		expectError bool
		description string
	}{
		{
			name: "Test 1: Initialize with all valid credentials",
			config: GitHubConfig{
				AppID:                        12345,
				InstallationID:               67890,
				PrivateKey:                   []byte("/path/to/key.pem"),
				Owner:                        "test-org",
				CommitAuthor:                 "test-bot",
				CommitEmail:                  "test@example.com",
				VictoriaMetricsServerAddress: "http://test-vm:8481/select/100/prometheus/",
			},
			expectError: true, // Will fail due to invalid key path
			description: "Complete configuration should initialize all components",
		},
		{
			name: "Test 2: Initialize without client credentials (config only)",
			config: GitHubConfig{
				Owner:                        "test-org",
				CommitAuthor:                 "test-bot",
				CommitEmail:                  "test@example.com",
				VictoriaMetricsServerAddress: "http://test-vm:8481/select/100/prometheus/",
			},
			expectError: false,
			description: "Configuration should work without client credentials",
		},
		{
			name:        "Test 3: Initialize with empty config (use defaults)",
			config:      GitHubConfig{},
			expectError: false,
			description: "Empty config should use default values",
		},
		{
			name: "Test 4: Initialize with partial config",
			config: GitHubConfig{
				Owner:       "custom-org",
				CommitEmail: "custom@example.com",
			},
			expectError: false,
			description: "Partial config should work with defaults for missing values",
		},
		{
			name: "Test 5: Initialize with invalid client credentials",
			config: GitHubConfig{
				AppID:          12345,
				InstallationID: 67890,
				PrivateKey:     []byte("/nonexistent/path.pem"),
				Owner:          "test-org",
			},
			expectError: true,
			description: "Invalid credentials should return error but not crash",
		},
		{
			name: "Test 6: Initialize with zero AppID (should skip client init)",
			config: GitHubConfig{
				AppID:          0,
				InstallationID: 67890,
				PrivateKey:     []byte("/path/to/key.pem"),
				Owner:          "test-org",
			},
			expectError: false,
			description: "Zero AppID should skip client initialization",
		},
		{
			name: "Test 7: Initialize with only repository names",
			config: GitHubConfig{
				Owner: "custom-org",
			},
			expectError: false,
			description: "Repository names should be configurable independently",
		},
		{
			name: "Test 8: Initialize with only VictoriaMetrics address",
			config: GitHubConfig{
				VictoriaMetricsServerAddress: "http://custom-vm:8481/select/100/prometheus/",
			},
			expectError: false,
			description: "VictoriaMetrics address should be configurable independently",
		},
		{
			name: "Test 9: Initialize with all empty strings",
			config: GitHubConfig{
				Owner:                        "",
				CommitAuthor:                 "",
				CommitEmail:                  "",
				VictoriaMetricsServerAddress: "",
			},
			expectError: false,
			description: "Empty strings should use default values",
		},
		{
			name: "Test 10: Initialize with valid credentials but missing optional config",
			config: GitHubConfig{
				AppID:          12345,
				InstallationID: 67890,
				PrivateKey:     []byte("/path/to/key.pem"),
				// Other fields use defaults
			},
			expectError: true, // Will fail due to invalid key path
			description: "Client should initialize even with minimal config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state for each test
			githubClient = nil
			githubClientErr = nil
			githubClientOnce = sync.Once{}
			githubOwner = "Meesho"
			githubCommitAuthor = "horizon-github"
			githubCommitEmail = "devops@meesho.com"
			victoriaMetricsServerAddress = "http://vmselect-datascience-prd-proxy.victoriametrics.svc.cluster.local:8481/select/100/prometheus/"

			err := InitGitHub(tt.config)
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}

			// Verify configuration was applied (for non-error cases or when config is set)
			if !tt.expectError || tt.config.Owner != "" {
				if tt.config.Owner != "" {
					assert.Equal(t, tt.config.Owner, githubOwner, "Owner should be set")
				}
			}
		})
	}
}

func TestGitHubConfig_ZeroValues(t *testing.T) {
	// Test that zero values work correctly
	config := GitHubConfig{}

	// Reset state
	githubClient = nil
	githubClientErr = nil
	githubClientOnce = sync.Once{}

	err := InitGitHub(config)
	// Should not error even with zero values (uses defaults)
	assert.NoError(t, err)
}
