package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAutoScalinGitHubCPUThreshold_GetThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold AutoScalinGitHubCPUThreshold
		expected  string
	}{
		{"Valid threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "80"}, "80"},
		{"Zero threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "0"}, "0"},
		{"Empty threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: ""}, ""},
		{"Large threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "100"}, "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.threshold.GetThreshold()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAutoScalinGitHubCPUThreshold_GetTriggerType(t *testing.T) {
	threshold := AutoScalinGitHubCPUThreshold{CPUThreshold: "80"}
	assert.Equal(t, "cpu", threshold.GetTriggerType())
}

func TestAutoScalinGitHubCPUThreshold_Validate(t *testing.T) {
	tests := []struct {
		name        string
		threshold   AutoScalinGitHubCPUThreshold
		expectError bool
		errorMsg    string
	}{
		// Valid cases
		{"Valid positive threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "80"}, false, ""},
		{"Valid single digit", AutoScalinGitHubCPUThreshold{CPUThreshold: "1"}, false, ""},
		{"Valid large threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "100"}, false, ""},
		{"Valid three digit", AutoScalinGitHubCPUThreshold{CPUThreshold: "150"}, false, ""},

		// Invalid cases
		{"Empty threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: ""}, true, "why supplying empty values?"},
		{"Zero threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "0"}, true, "cpu threshold cannot be zero or negative"},
		{"Negative threshold", AutoScalinGitHubCPUThreshold{CPUThreshold: "-10"}, true, "cpu threshold cannot be zero or negative"},
		{"Non-numeric string", AutoScalinGitHubCPUThreshold{CPUThreshold: "abc"}, true, ""},
		{"Float string", AutoScalinGitHubCPUThreshold{CPUThreshold: "80.5"}, true, ""},
		{"Whitespace only", AutoScalinGitHubCPUThreshold{CPUThreshold: "   "}, true, "why supplying empty values?"},
		{"Leading zeros", AutoScalinGitHubCPUThreshold{CPUThreshold: "007"}, false, ""}, // Valid, parsed as 7
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.threshold.Validate()
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

func TestAutoScalinGitHubGPUThreshold_GetThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold AutoScalinGitHubGPUThreshold
		expected  string
	}{
		{"Valid threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "70"}, "70"},
		{"Zero threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "0"}, "0"},
		{"Empty threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: ""}, ""},
		{"Large threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "95"}, "95"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.threshold.GetThreshold()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAutoScalinGitHubGPUThreshold_GetTriggerType(t *testing.T) {
	threshold := AutoScalinGitHubGPUThreshold{GPUThreshold: "70"}
	assert.Equal(t, "gpu", threshold.GetTriggerType())
}

func TestAutoScalinGitHubGPUThreshold_Validate(t *testing.T) {
	tests := []struct {
		name        string
		threshold   AutoScalinGitHubGPUThreshold
		expectError bool
		errorMsg    string
	}{
		// Valid cases
		{"Valid positive threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "70"}, false, ""},
		{"Valid single digit", AutoScalinGitHubGPUThreshold{GPUThreshold: "1"}, false, ""},
		{"Valid large threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "100"}, false, ""},
		{"Valid three digit", AutoScalinGitHubGPUThreshold{GPUThreshold: "200"}, false, ""},

		// Invalid cases
		{"Empty threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: ""}, true, "why supplying empty values?"},
		{"Zero threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "0"}, true, "gpu threshold cannot be zero or negative"},
		{"Negative threshold", AutoScalinGitHubGPUThreshold{GPUThreshold: "-5"}, true, "gpu threshold cannot be zero or negative"},
		{"Non-numeric string", AutoScalinGitHubGPUThreshold{GPUThreshold: "xyz"}, true, ""},
		{"Float string", AutoScalinGitHubGPUThreshold{GPUThreshold: "70.5"}, true, ""},
		{"Whitespace only", AutoScalinGitHubGPUThreshold{GPUThreshold: "   "}, true, "why supplying empty values?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.threshold.Validate()
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

func TestAutoScalinGitHubProperties_ValidateAutoScalinGitHubProperties(t *testing.T) {
	tests := []struct {
		name        string
		props       AutoScalinGitHubProperties
		expectError bool
		errorMsg    string
	}{
		// Valid cases
		{"Valid min max", AutoScalinGitHubProperties{MinReplicas: "2", MaxReplicas: "10"}, false, ""},
		{"Equal min max", AutoScalinGitHubProperties{MinReplicas: "5", MaxReplicas: "5"}, false, ""},
		{"Single digit", AutoScalinGitHubProperties{MinReplicas: "1", MaxReplicas: "9"}, false, ""},
		{"Large values", AutoScalinGitHubProperties{MinReplicas: "10", MaxReplicas: "100"}, false, ""},
		{"Min is 1", AutoScalinGitHubProperties{MinReplicas: "1", MaxReplicas: "50"}, false, ""},

		// Invalid cases - empty values
		{"Empty min", AutoScalinGitHubProperties{MinReplicas: "", MaxReplicas: "10"}, true, "why supplying empty values?"},
		{"Empty max", AutoScalinGitHubProperties{MinReplicas: "2", MaxReplicas: ""}, true, "why supplying empty values?"},
		{"Both empty", AutoScalinGitHubProperties{MinReplicas: "", MaxReplicas: ""}, true, "why supplying empty values?"},

		// Invalid cases - zero or negative
		{"Zero min", AutoScalinGitHubProperties{MinReplicas: "0", MaxReplicas: "10"}, true, "min/max cannot be zero or negative"},
		{"Zero max", AutoScalinGitHubProperties{MinReplicas: "2", MaxReplicas: "0"}, true, "min/max cannot be zero or negative"},
		{"Both zero", AutoScalinGitHubProperties{MinReplicas: "0", MaxReplicas: "0"}, true, "min/max cannot be zero or negative"},
		{"Negative min", AutoScalinGitHubProperties{MinReplicas: "-1", MaxReplicas: "10"}, true, "min/max cannot be zero or negative"},
		{"Negative max", AutoScalinGitHubProperties{MinReplicas: "2", MaxReplicas: "-5"}, true, "min/max cannot be zero or negative"},

		// Invalid cases - min > max
		{"Min greater than max", AutoScalinGitHubProperties{MinReplicas: "10", MaxReplicas: "5"}, true, "min cannot be greater than max"},
		{"Min much greater", AutoScalinGitHubProperties{MinReplicas: "100", MaxReplicas: "50"}, true, "min cannot be greater than max"},

		// Invalid cases - non-numeric
		{"Non-numeric min", AutoScalinGitHubProperties{MinReplicas: "abc", MaxReplicas: "10"}, true, ""},
		{"Non-numeric max", AutoScalinGitHubProperties{MinReplicas: "2", MaxReplicas: "xyz"}, true, ""},
		{"Both non-numeric", AutoScalinGitHubProperties{MinReplicas: "abc", MaxReplicas: "xyz"}, true, ""},
		{"Float min", AutoScalinGitHubProperties{MinReplicas: "2.5", MaxReplicas: "10"}, true, ""},
		{"Float max", AutoScalinGitHubProperties{MinReplicas: "2", MaxReplicas: "10.5"}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.props.ValidateAutoScalinGitHubProperties()
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

func TestAutoScalinGitHubCronProperties_ValidateCronGitHubProperties(t *testing.T) {
	tests := []struct {
		name        string
		props       AutoScalinGitHubCronProperties
		expectError bool
		errorMsg    string
	}{
		// Valid cases
		{"Valid cron properties", AutoScalinGitHubCronProperties{Start: "09:00", End: "17:00", DesiredReplicas: "5"}, false, ""},
		{"Single replica", AutoScalinGitHubCronProperties{Start: "00:00", End: "23:59", DesiredReplicas: "1"}, false, ""},
		{"Large replicas", AutoScalinGitHubCronProperties{Start: "10:00", End: "18:00", DesiredReplicas: "100"}, false, ""},

		// Invalid cases - empty values
		{"Empty start", AutoScalinGitHubCronProperties{Start: "", End: "17:00", DesiredReplicas: "5"}, true, "why supplying empty values?"},
		{"Empty end", AutoScalinGitHubCronProperties{Start: "09:00", End: "", DesiredReplicas: "5"}, true, "why supplying empty values?"},
		{"Empty desired replicas", AutoScalinGitHubCronProperties{Start: "09:00", End: "17:00", DesiredReplicas: ""}, true, "why supplying empty values?"},
		{"All empty", AutoScalinGitHubCronProperties{Start: "", End: "", DesiredReplicas: ""}, true, "why supplying empty values?"},

		// Invalid cases - zero or negative replicas
		{"Zero replicas", AutoScalinGitHubCronProperties{Start: "09:00", End: "17:00", DesiredReplicas: "0"}, true, "desiredReplicas cannot be zero"},
		{"Negative replicas", AutoScalinGitHubCronProperties{Start: "09:00", End: "17:00", DesiredReplicas: "-1"}, true, "desiredReplicas cannot be zero"},

		// Invalid cases - non-numeric replicas
		{"Non-numeric replicas", AutoScalinGitHubCronProperties{Start: "09:00", End: "17:00", DesiredReplicas: "abc"}, true, ""},
		{"Float replicas", AutoScalinGitHubCronProperties{Start: "09:00", End: "17:00", DesiredReplicas: "5.5"}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.props.ValidateCronGitHubProperties()
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

func TestThresholdProperties_Interface(t *testing.T) {
	// Test CPU threshold implements interface correctly
	cpuThreshold := AutoScalinGitHubCPUThreshold{CPUThreshold: "80"}
	var thresholdProps ThresholdProperties = cpuThreshold
	assert.Equal(t, "80", thresholdProps.GetThreshold())
	assert.Equal(t, "cpu", thresholdProps.GetTriggerType())
	assert.NoError(t, thresholdProps.Validate())

	// Test GPU threshold implements interface correctly
	gpuThreshold := AutoScalinGitHubGPUThreshold{GPUThreshold: "70"}
	thresholdProps = gpuThreshold
	assert.Equal(t, "70", thresholdProps.GetThreshold())
	assert.Equal(t, "gpu", thresholdProps.GetTriggerType())
	assert.NoError(t, thresholdProps.Validate())
}
