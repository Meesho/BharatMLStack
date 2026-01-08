package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTriggerUpdater(t *testing.T) {
	tests := []struct {
		name        string
		triggerType string
		appName     string
		expected    TriggerUpdater
		expectNil   bool
	}{
		{"CPU trigger", "cpu", "test-app", &CPUTriggerUpdater{}, false},
		{"GPU trigger", "gpu", "test-app", &GPUTriggerUpdater{appName: "test-app"}, false},
		{"Unknown trigger", "unknown", "test-app", nil, true},
		{"Empty trigger type", "", "test-app", nil, true},
		{"Memory trigger (not supported)", "memory", "test-app", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTriggerUpdater(tt.triggerType, tt.appName)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				// For GPU triggers, GetTriggerType returns "prometheus", not "gpu"
				// So we check the actual type instead of comparing triggerType
				if tt.triggerType == "gpu" {
					_, ok := result.(*GPUTriggerUpdater)
					assert.True(t, ok, "Expected GPUTriggerUpdater for GPU trigger type")
					assert.Equal(t, "prometheus", result.GetTriggerType())
				} else {
					assert.Equal(t, tt.triggerType, result.GetTriggerType())
				}
			}
		})
	}
}

func TestCPUTriggerUpdater_GetTriggerType(t *testing.T) {
	updater := &CPUTriggerUpdater{}
	assert.Equal(t, "cpu", updater.GetTriggerType())
}

func TestCPUTriggerUpdater_GetTriggerIdentifier(t *testing.T) {
	updater := &CPUTriggerUpdater{}
	assert.Equal(t, "type", updater.GetTriggerIdentifier())
}

func TestCPUTriggerUpdater_MatchesTrigger(t *testing.T) {
	updater := &CPUTriggerUpdater{}
	tests := []struct {
		name     string
		trigger  map[string]interface{}
		expected bool
	}{
		{"Matching CPU trigger", map[string]interface{}{"type": "cpu"}, true},
		{"Non-matching type", map[string]interface{}{"type": "gpu"}, false},
		{"Missing type field", map[string]interface{}{"metadata": map[string]interface{}{}}, false},
		{"Empty trigger", map[string]interface{}{}, false},
		{"Type as number", map[string]interface{}{"type": 123}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.MatchesTrigger(tt.trigger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCPUTriggerUpdater_CreateNewTrigger(t *testing.T) {
	updater := &CPUTriggerUpdater{}
	threshold := "80"

	trigger := updater.CreateNewTrigger(threshold)

	assert.NotNil(t, trigger)
	assert.Equal(t, "cpu", trigger["type"])
	assert.Equal(t, "Utilization", trigger["metricType"])

	metadata, ok := trigger["metadata"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, threshold, metadata["value"])
}

func TestCPUTriggerUpdater_UpdateTriggers(t *testing.T) {
	updater := &CPUTriggerUpdater{}
	newThreshold := "85"

	tests := []struct {
		name             string
		existingTriggers []interface{}
		expectedCount    int
		shouldUpdate     bool
		shouldAdd        bool
	}{
		{
			name: "Update existing CPU trigger",
			existingTriggers: []interface{}{
				map[string]interface{}{
					"type":     "cpu",
					"metadata": map[string]interface{}{"value": "80"},
				},
			},
			expectedCount: 1,
			shouldUpdate:  true,
			shouldAdd:     false,
		},
		{
			name: "Add CPU trigger when none exists",
			existingTriggers: []interface{}{
				map[string]interface{}{
					"type":     "gpu",
					"metadata": map[string]interface{}{"threshold": "70"},
				},
			},
			expectedCount: 2,
			shouldUpdate:  false,
			shouldAdd:     true,
		},
		{
			name:             "Add CPU trigger to empty list",
			existingTriggers: []interface{}{},
			expectedCount:    1,
			shouldUpdate:     false,
			shouldAdd:        true,
		},
		{
			name: "Preserve multiple non-CPU triggers",
			existingTriggers: []interface{}{
				map[string]interface{}{"type": "gpu", "metadata": map[string]interface{}{}},
				map[string]interface{}{"type": "memory", "metadata": map[string]interface{}{}},
			},
			expectedCount: 3,
			shouldUpdate:  false,
			shouldAdd:     true,
		},
		{
			name: "Update CPU trigger among multiple triggers",
			existingTriggers: []interface{}{
				map[string]interface{}{"type": "gpu", "metadata": map[string]interface{}{}},
				map[string]interface{}{
					"type":     "cpu",
					"metadata": map[string]interface{}{"value": "75"},
				},
				map[string]interface{}{"type": "memory", "metadata": map[string]interface{}{}},
			},
			expectedCount: 3,
			shouldUpdate:  true,
			shouldAdd:     false,
		},
		{
			name: "Handle invalid trigger format",
			existingTriggers: []interface{}{
				"invalid trigger",
				map[string]interface{}{
					"type":     "cpu",
					"metadata": map[string]interface{}{"value": "80"},
				},
			},
			expectedCount: 2,
			shouldUpdate:  true,
			shouldAdd:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.UpdateTriggers(tt.existingTriggers, newThreshold)

			assert.Equal(t, tt.expectedCount, len(result))

			// Verify CPU trigger exists with correct threshold
			cpuTriggerFound := false
			for _, trigger := range result {
				if triggerMap, ok := trigger.(map[string]interface{}); ok {
					if triggerType, exists := triggerMap["type"].(string); exists && triggerType == "cpu" {
						cpuTriggerFound = true
						metadata, ok := triggerMap["metadata"].(map[string]interface{})
						assert.True(t, ok)
						assert.Equal(t, newThreshold, metadata["value"])
					}
				}
			}
			assert.True(t, cpuTriggerFound, "CPU trigger should be present in result")
		})
	}
}

func TestGPUTriggerUpdater_GetTriggerType(t *testing.T) {
	updater := &GPUTriggerUpdater{appName: "test-app"}
	assert.Equal(t, "prometheus", updater.GetTriggerType())
}

func TestGPUTriggerUpdater_GetTriggerIdentifier(t *testing.T) {
	updater := &GPUTriggerUpdater{appName: "test-app"}
	assert.Equal(t, "metricName", updater.GetTriggerIdentifier())
}

func TestGPUTriggerUpdater_MatchesTrigger(t *testing.T) {
	updater := &GPUTriggerUpdater{appName: "test-app"}
	tests := []struct {
		name     string
		trigger  map[string]interface{}
		expected bool
	}{
		{
			name: "Matching GPU trigger",
			trigger: map[string]interface{}{
				"type": "prometheus",
				"metadata": map[string]interface{}{
					"metricName": "gpu_utilisation",
				},
			},
			expected: true,
		},
		{
			name: "Non-matching type",
			trigger: map[string]interface{}{
				"type": "cpu",
				"metadata": map[string]interface{}{
					"metricName": "gpu_utilisation",
				},
			},
			expected: false,
		},
		{
			name: "Non-matching metric name",
			trigger: map[string]interface{}{
				"type": "prometheus",
				"metadata": map[string]interface{}{
					"metricName": "cpu_utilisation",
				},
			},
			expected: false,
		},
		{
			name: "Missing metadata",
			trigger: map[string]interface{}{
				"type": "prometheus",
			},
			expected: false,
		},
		{
			name:     "Empty trigger",
			trigger:  map[string]interface{}{},
			expected: false,
		},
		{
			name: "Missing metricName in metadata",
			trigger: map[string]interface{}{
				"type": "prometheus",
				"metadata": map[string]interface{}{
					"threshold": "70",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.MatchesTrigger(tt.trigger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGPUTriggerUpdater_CreateNewTrigger(t *testing.T) {
	appName := "test-app"
	updater := &GPUTriggerUpdater{appName: appName}
	threshold := "70"

	trigger := updater.CreateNewTrigger(threshold)

	assert.NotNil(t, trigger)
	assert.Equal(t, "prometheus", trigger["type"])

	metadata, ok := trigger["metadata"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "gpu_utilisation", metadata["metricName"])
	assert.Equal(t, threshold, metadata["threshold"])
	assert.Contains(t, metadata["query"].(string), appName)
	assert.NotEmpty(t, metadata["serverAddress"])
}

func TestGPUTriggerUpdater_UpdateTriggers(t *testing.T) {
	appName := "test-app"
	updater := &GPUTriggerUpdater{appName: appName}
	newThreshold := "75"

	tests := []struct {
		name             string
		existingTriggers []interface{}
		expectedCount    int
		shouldUpdate     bool
		shouldAdd        bool
	}{
		{
			name: "Update existing GPU trigger",
			existingTriggers: []interface{}{
				map[string]interface{}{
					"type": "prometheus",
					"metadata": map[string]interface{}{
						"metricName": "gpu_utilisation",
						"threshold":  "70",
					},
				},
			},
			expectedCount: 1,
			shouldUpdate:  true,
			shouldAdd:     false,
		},
		{
			name: "Add GPU trigger when none exists",
			existingTriggers: []interface{}{
				map[string]interface{}{
					"type":     "cpu",
					"metadata": map[string]interface{}{"value": "80"},
				},
			},
			expectedCount: 2,
			shouldUpdate:  false,
			shouldAdd:     true,
		},
		{
			name:             "Add GPU trigger to empty list",
			existingTriggers: []interface{}{},
			expectedCount:    1,
			shouldUpdate:     false,
			shouldAdd:        true,
		},
		{
			name: "Preserve multiple non-GPU triggers",
			existingTriggers: []interface{}{
				map[string]interface{}{"type": "cpu", "metadata": map[string]interface{}{}},
				map[string]interface{}{"type": "memory", "metadata": map[string]interface{}{}},
			},
			expectedCount: 3,
			shouldUpdate:  false,
			shouldAdd:     true,
		},
		{
			name: "Update GPU trigger among multiple triggers",
			existingTriggers: []interface{}{
				map[string]interface{}{"type": "cpu", "metadata": map[string]interface{}{}},
				map[string]interface{}{
					"type": "prometheus",
					"metadata": map[string]interface{}{
						"metricName": "gpu_utilisation",
						"threshold":  "70",
					},
				},
				map[string]interface{}{"type": "memory", "metadata": map[string]interface{}{}},
			},
			expectedCount: 3,
			shouldUpdate:  true,
			shouldAdd:     false,
		},
		{
			name: "Handle invalid trigger format",
			existingTriggers: []interface{}{
				"invalid trigger",
				map[string]interface{}{
					"type": "prometheus",
					"metadata": map[string]interface{}{
						"metricName": "gpu_utilisation",
						"threshold":  "70",
					},
				},
			},
			expectedCount: 2,
			shouldUpdate:  true,
			shouldAdd:     false,
		},
		{
			name: "Handle trigger without type field",
			existingTriggers: []interface{}{
				map[string]interface{}{
					"metadata": map[string]interface{}{
						"metricName": "gpu_utilisation",
					},
				},
			},
			expectedCount: 2,
			shouldUpdate:  false,
			shouldAdd:     true,
		},
		{
			name: "Preserve other prometheus triggers",
			existingTriggers: []interface{}{
				map[string]interface{}{
					"type": "prometheus",
					"metadata": map[string]interface{}{
						"metricName": "cpu_utilisation",
						"threshold":  "80",
					},
				},
			},
			expectedCount: 2,
			shouldUpdate:  false,
			shouldAdd:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.UpdateTriggers(tt.existingTriggers, newThreshold)

			assert.Equal(t, tt.expectedCount, len(result))

			// Verify GPU trigger exists with correct threshold
			gpuTriggerFound := false
			for _, trigger := range result {
				if triggerMap, ok := trigger.(map[string]interface{}); ok {
					if triggerType, exists := triggerMap["type"].(string); exists && triggerType == "prometheus" {
						if metadata, ok := triggerMap["metadata"].(map[string]interface{}); ok {
							if metricName, exists := metadata["metricName"].(string); exists && metricName == "gpu_utilisation" {
								gpuTriggerFound = true
								assert.Equal(t, newThreshold, metadata["threshold"])
							}
						}
					}
				}
			}
			assert.True(t, gpuTriggerFound, "GPU trigger should be present in result")
		})
	}
}
