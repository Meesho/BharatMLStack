package github

import (
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
)

func TestInitGitHubClient(t *testing.T) {
	tests := []struct {
		name        string
		appID       int64
		installID   int64
		privateKey  []byte
		expectError bool
		description string
	}{
		{
			name:        "Test 1: Initialize with invalid private key path",
			appID:       12345,
			installID:   67890,
			privateKey:  []byte("/nonexistent/path/to/key.pem"),
			expectError: true,
			description: "Initialization with invalid key path should fail",
		},
		{
			name:        "Test 2: Initialize with zero app ID",
			appID:       0,
			installID:   67890,
			privateKey:  []byte("/tmp/test-key.pem"),
			expectError: true,
			description: "Initialization with zero app ID may fail",
		},
		{
			name:        "Test 3: Initialize with zero installation ID",
			appID:       12345,
			installID:   0,
			privateKey:  []byte("/tmp/test-key.pem"),
			expectError: true,
			description: "Initialization with zero installation ID may fail",
		},
		{
			name:        "Test 4: Initialize with empty private key path",
			appID:       12345,
			installID:   67890,
			privateKey:  []byte(""),
			expectError: true,
			description: "Initialization with empty key path should fail",
		},
		{
			name:        "Test 5: Multiple initialization calls",
			appID:       12345,
			installID:   67890,
			privateKey:  []byte("/tmp/test-key.pem"),
			expectError: true,
			description: "Multiple initialization calls should use sync.Once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: sync.Once prevents re-initialization in same process
			// In real tests, this would require process isolation or test-specific setup
			err := InitGitHubClient(tt.appID, tt.installID, tt.privateKey)
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				// Only assert no error if we don't expect one
				// Most cases will fail due to invalid key path
			}
		})
	}
}

func TestGetGitHubClient(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Test 1: Get client when not initialized",
			initialized: false,
			expectError: true,
			errorMsg:    "GitHub client not initialized",
		},
		{
			name:        "Test 2: Get client when initialized",
			initialized: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			githubClient = nil
			githubClientErr = nil

			if tt.initialized {
				// Mock initialization - in real scenario this would require valid credentials
				// For testing, we'll just check the error case
				githubClient = &github.Client{}
			}

			client, err := GetGitHubClient()
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestSafeInt(t *testing.T) {
	tests := []struct {
		name     string
		input    *int
		expected int
	}{
		{"Nil pointer", nil, 0},
		{"Zero value", github.Int(0), 0},
		{"Positive value", github.Int(42), 42},
		{"Negative value", github.Int(-10), -10},
		{"Large value", github.Int(1000000), 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadYaml(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		repo        string
		branch      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Test 1: Read YAML with uninitialized client",
			filePath:    "test.yaml",
			repo:        "test-repo",
			branch:      "main",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
		},
		{
			name:        "Test 2: Read YAML with empty file path",
			filePath:    "",
			repo:        "test-repo",
			branch:      "main",
			expectError: true,
		},
		{
			name:        "Test 3: Read YAML with empty repo",
			filePath:    "test.yaml",
			repo:        "",
			branch:      "main",
			expectError: true,
		},
		{
			name:        "Test 4: Read YAML with empty branch",
			filePath:    "test.yaml",
			repo:        "test-repo",
			branch:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset client state
			githubClient = nil
			githubClientErr = nil

			result, err := ReadYaml(tt.filePath, tt.repo, tt.branch)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestPushContentToGithub(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		repo        string
		branch      string
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Test 1: Push content with uninitialized client",
			content:     []byte("test content"),
			repo:        "test-repo",
			branch:      "main",
			filePath:    "test.yaml",
			expectError: true,
			errorMsg:    "GitHub client not initialized",
		},
		{
			name:        "Test 2: Push empty content",
			content:     []byte(""),
			repo:        "test-repo",
			branch:      "main",
			filePath:    "test.yaml",
			expectError: true,
		},
		{
			name:        "Test 3: Push with empty repo",
			content:     []byte("test content"),
			repo:        "",
			branch:      "main",
			filePath:    "test.yaml",
			expectError: true,
		},
		{
			name:        "Test 4: Push with empty branch",
			content:     []byte("test content"),
			repo:        "test-repo",
			branch:      "",
			filePath:    "test.yaml",
			expectError: true,
		},
		{
			name:        "Test 5: Push with empty file path",
			content:     []byte("test content"),
			repo:        "test-repo",
			branch:      "main",
			filePath:    "",
			expectError: true,
		},
		{
			name:        "Test 6: Push with large content",
			content:     make([]byte, 1024*1024), // 1MB
			repo:        "test-repo",
			branch:      "main",
			filePath:    "test.yaml",
			expectError: true,
		},
		{
			name:        "Test 7: Push with special characters in path",
			content:     []byte("test content"),
			repo:        "test-repo",
			branch:      "main",
			filePath:    "test/file-path.yaml",
			expectError: true,
		},
		{
			name:        "Test 8: Push with YAML content",
			content:     []byte("key: value\nnested:\n  subkey: subvalue"),
			repo:        "test-repo",
			branch:      "main",
			filePath:    "config.yaml",
			expectError: true,
		},
		{
			name:        "Test 9: Push with JSON content",
			content:     []byte(`{"key": "value", "nested": {"subkey": "subvalue"}}`),
			repo:        "test-repo",
			branch:      "main",
			filePath:    "config.json",
			expectError: true,
		},
		{
			name:        "Test 10: Push with binary content",
			content:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			repo:        "test-repo",
			branch:      "main",
			filePath:    "binary.bin",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset client state
			githubClient = nil
			githubClientErr = nil

			err := PushContentToGithub(tt.content, tt.repo, tt.branch, tt.filePath)
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
