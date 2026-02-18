package circuitbreaker

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupViper(configs map[string]interface{}) {
	viper.Reset()
	for key, value := range configs {
		viper.Set(key, value)
	}
}

func TestBuildConfig_DisabledByDefault(t *testing.T) {
	viper.Reset()
	config := BuildConfig("testService")
	assert.False(t, config.Enabled, "Circuit breaker should be disabled by default")
}

func TestBuildConfig_EnabledWithValidConfig(t *testing.T) {
	setupViper(map[string]interface{}{
		"testService_CB_ENABLED":                     true,
		"testService_CB_NAME":                        "testCB",
		"testService_CB_FAILURE_RATE_THRESHOLD":      50,
		"testService_CB_FAILURE_RATE_MINIMUM_WINDOW": 20,
		"testService_CB_FAILURE_RATE_WINDOW_IN_MS":   10000,
		"testService_CB_FAILURE_COUNT_THRESHOLD":     5,
		"testService_CB_FAILURE_COUNT_WINDOW":        10,
		"testService_CB_SUCCESS_COUNT_THRESHOLD":     75,
		"testService_CB_SUCCESS_COUNT_WINDOW":        10,
		"testService_CB_WITH_DELAY_IN_MS":            1000,
		"testService_CB_VERSION":                     1,
	})

	config := BuildConfig("testService")
	assert.True(t, config.Enabled, "Circuit breaker should be enabled")
	assert.Equal(t, "testCB", config.Name)
	assert.Equal(t, 50, config.FailureRateThreshold)
	assert.Equal(t, 20, config.FailureRateMinimumWindow)
	assert.Equal(t, 10000, config.FailureRateWindowInMs)
	assert.Equal(t, 5, config.FailureCountThreshold)
	assert.Equal(t, 10, config.FailureCountWindow)
	assert.Equal(t, 75, config.SuccessCountThreshold)
	assert.Equal(t, 10, config.SuccessCountWindow)
	assert.Equal(t, 1000, config.WithDelayInMS)
	assert.Equal(t, 1, config.Version)
}

func TestBuildConfig_MissingMandatoryConfigs(t *testing.T) {
	setupViper(map[string]interface{}{
		"testService_CB_ENABLED": true,
	})

	assert.Panics(t, func() { BuildConfig("testService") }, "Should panic due to missing mandatory configs")
}

func TestValidateConfigs_Panics(t *testing.T) {
	baseConfig := map[string]interface{}{
		"testService_CB_ENABLED":                     true,
		"testService_CB_NAME":                        "test",
		"testService_CB_FAILURE_RATE_THRESHOLD":      50,
		"testService_CB_FAILURE_RATE_MINIMUM_WINDOW": 10,
		"testService_CB_FAILURE_RATE_WINDOW_IN_MS":   1000,
		"testService_CB_SUCCESS_COUNT_THRESHOLD":     5,
		"testService_CB_SUCCESS_COUNT_WINDOW":        10,
		"testService_CB_WITH_DELAY_IN_MS":            100,
		"testService_CB_VERSION":                     1,
	}

	testCases := []struct {
		name        string
		missingKey  string
		expectPanic bool
	}{
		{"Missing CB_NAME", "testService_CB_NAME", true},
		{"Missing CB_FAILURE_RATE_THRESHOLD", "testService_CB_FAILURE_RATE_THRESHOLD", true},
		{"Missing CB_FAILURE_RATE_MINIMUM_WINDOW", "testService_CB_FAILURE_RATE_MINIMUM_WINDOW", true},
		{"Missing CB_FAILURE_RATE_WINDOW_IN_MS", "testService_CB_FAILURE_RATE_WINDOW_IN_MS", true},
		{"Missing CB_SUCCESS_COUNT_THRESHOLD", "testService_CB_SUCCESS_COUNT_THRESHOLD", true},
		{"Missing CB_SUCCESS_COUNT_WINDOW", "testService_CB_SUCCESS_COUNT_WINDOW", true},
		{"Missing CB_WITH_DELAY_IN_MS", "testService_CB_WITH_DELAY_IN_MS", true},
		{"Missing CB_VERSION", "testService_CB_VERSION", true},
		{"Valid Config", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// copy base config
			config := make(map[string]interface{})
			for k, v := range baseConfig {
				config[k] = v
			}

			if tc.missingKey != "" {
				delete(config, tc.missingKey)
			}

			// Special case for thresholding logic
			if tc.name == "Missing CB_FAILURE_RATE_THRESHOLD" {
				config["testService_CB_FAILURE_COUNT_THRESHOLD"] = 0 // ensure count based is also not set
			}

			setupViper(config)

			if tc.expectPanic {
				assert.Panics(t, func() { BuildConfig("testService") })
			} else {
				assert.NotPanics(t, func() { BuildConfig("testService") })
			}
		})
	}

	t.Run("Neither time nor count based threshold", func(t *testing.T) {
		config := make(map[string]interface{})
		for k, v := range baseConfig {
			config[k] = v
		}
		delete(config, "testService_CB_FAILURE_RATE_THRESHOLD")
		delete(config, "testService_CB_FAILURE_COUNT_THRESHOLD")
		setupViper(config)
		assert.Panics(t, func() { BuildConfig("testService") })
	})
}
