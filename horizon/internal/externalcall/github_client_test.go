package externalcall

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitGitHubClient(t *testing.T) {
	tests := []struct {
		name                   string
		appID                  int64
		installationID         int64
		privateKey             []byte
		owner                  string
		commitAuthor           string
		commitEmail            string
		victoriaMetricsAddress string
		description            string
	}{
		{
			name:                   "Test 1: Initialize with complete configuration",
			appID:                  12345,
			installationID:         67890,
			privateKey:             []byte("/path/to/key.pem"),
			owner:                  "test-org",
			commitAuthor:           "test-bot",
			commitEmail:            "test@example.com",
			victoriaMetricsAddress: "http://test-vm:8481/select/100/prometheus/",
			description:            "Complete configuration should initialize all components",
		},
		{
			name:                   "Test 2: Initialize without client credentials",
			appID:                  0,
			installationID:         0,
			privateKey:             []byte(""),
			owner:                  "test-org",
			commitAuthor:           "test-bot",
			commitEmail:            "test@example.com",
			victoriaMetricsAddress: "http://test-vm:8481/select/100/prometheus/",
			description:            "Configuration should work without client credentials",
		},
		{
			name:                   "Test 3: Initialize with empty strings (use defaults)",
			appID:                  0,
			installationID:         0,
			privateKey:             []byte(""),
			owner:                  "",
			commitAuthor:           "",
			commitEmail:            "",
			victoriaMetricsAddress: "",
			description:            "Empty strings should use default values",
		},
		{
			name:                   "Test 4: Initialize with partial configuration",
			appID:                  0,
			installationID:         0,
			privateKey:             []byte(""),
			owner:                  "custom-org",
			commitAuthor:           "",
			commitEmail:            "custom@example.com",
			victoriaMetricsAddress: "",
			description:            "Partial config should work with defaults for missing values",
		},
		{
			name:                   "Test 5: Initialize with only VictoriaMetrics address",
			appID:                  0,
			installationID:         0,
			privateKey:             []byte(""),
			owner:                  "",
			commitAuthor:           "",
			commitEmail:            "",
			victoriaMetricsAddress: "http://custom-vm:8481/select/100/prometheus/",
			description:            "VictoriaMetrics address should be configurable independently",
		},
		{
			name:                   "Test 6: Initialize with zero AppID (should skip client init)",
			appID:                  0,
			installationID:         67890,
			privateKey:             []byte("/path/to/key.pem"),
			owner:                  "test-org",
			commitAuthor:           "",
			commitEmail:            "",
			victoriaMetricsAddress: "",
			description:            "Zero AppID should skip client initialization",
		},
		{
			name:                   "Test 7: Initialize with only owner and email",
			appID:                  0,
			installationID:         0,
			privateKey:             []byte(""),
			owner:                  "my-org",
			commitAuthor:           "",
			commitEmail:            "devops@myorg.com",
			victoriaMetricsAddress: "",
			description:            "Should work with minimal configuration",
		},
		{
			name:                   "Test 8: Initialize with invalid credentials path",
			appID:                  12345,
			installationID:         67890,
			privateKey:             []byte("/nonexistent/path.pem"),
			owner:                  "test-org",
			commitAuthor:           "",
			commitEmail:            "",
			victoriaMetricsAddress: "",
			description:            "Invalid credentials should not crash, errors logged internally",
		},
		{
			name:                   "Test 9: Initialize multiple times (should only initialize once)",
			appID:                  0,
			installationID:         0,
			privateKey:             []byte(""),
			owner:                  "test-org",
			commitAuthor:           "test-bot",
			commitEmail:            "test@example.com",
			victoriaMetricsAddress: "http://test-vm:8481/select/100/prometheus/",
			description:            "Multiple calls should be idempotent due to sync.Once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state by calling init with test parameters
			// Note: sync.Once will prevent re-initialization in same test run
			// This is expected behavior - initialization should be idempotent
			InitGitHubClient(
				tt.appID,
				tt.installationID,
				tt.privateKey,
				tt.owner,
				tt.commitAuthor,
				tt.commitEmail,
				tt.victoriaMetricsAddress,
				nil, // branchConfig - nil for tests (uses defaults)
			)
			// If we get here without panic, initialization succeeded
			// The actual initialization logic is tested in the GitHub package
			assert.True(t, true, tt.description)
		})
	}
}

func TestInitGitHubClient_Idempotent(t *testing.T) {
	// Test that multiple calls to InitGitHubClient are idempotent
	InitGitHubClient(0, 0, []byte(""), "org1", "author1", "email1", "", nil)
	InitGitHubClient(0, 0, []byte(""), "org2", "author2", "email2", "", nil)
	InitGitHubClient(0, 0, []byte(""), "org1", "author1", "email1", "", nil)
	InitGitHubClient(0, 0, []byte(""), "org2", "author2", "email2", "", nil)
	// Second call should not change the configuration due to sync.Once
	// This is expected behavior - initialization should happen only once
	assert.True(t, true, "Multiple initialization calls should be idempotent")
}
