package circuitbreaker

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetCircuitBreaker_Version1(t *testing.T) {
	config := &Config{
		Version:                  1,
		Name:                     "testCB",
		FailureRateThreshold:     50,
		FailureRateMinimumWindow: 20,
		FailureRateWindowInMs:    10000,
		FailureCountThreshold:    5,
		FailureCountWindow:       10,
		SuccessCountThreshold:    75,
		SuccessCountWindow:       10,
		WithDelayInMS:            1000,
	}

	cb := GetCircuitBreaker[int, int](config)
	assert.NotNil(t, cb, "Circuit breaker should not be nil")
}

func TestGetCircuitBreaker_UnsupportedVersion(t *testing.T) {
	config := &Config{
		Version: 999,
	}

	assert.Panics(t, func() { GetCircuitBreaker[int, int](config) }, "Should panic for unsupported version")
}

func TestFailSafeCB_Initialization(t *testing.T) {
	config := &Config{
		Name:                     "testCB",
		FailureRateThreshold:     50,
		FailureRateMinimumWindow: 20,
		FailureRateWindowInMs:    10000,
		SuccessCountThreshold:    75,
		SuccessCountWindow:       10,
		WithDelayInMS:            1000,
	}

	cb := failSafeCB[int, int](config)
	assert.NotNil(t, cb, "FailSafeCB should not be nil")
}

func TestGetManualCircuitBreaker_NilConfig(t *testing.T) {
	cb := GetManualCircuitBreaker(nil)
	assert.Nil(t, cb, "Circuit breaker should be nil for nil config")
}

func TestGetManualCircuitBreaker_Disabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}
	cb := GetManualCircuitBreaker(config)
	assert.NotNil(t, cb)
	assert.True(t, cb.IsAllowed(), "pass through breaker should always allow calls")
	cb.RecordFailure() // should not panic
	cb.RecordSuccess() // should not panic
}

func TestGetManualCircuitBreaker_Version1(t *testing.T) {
	config := &Config{
		Enabled:                  true,
		Version:                  1,
		Name:                     "testManualCB",
		FailureRateThreshold:     50,
		FailureRateMinimumWindow: 20,
		FailureRateWindowInMs:    10000,
		SuccessCountThreshold:    75,
		SuccessCountWindow:       10,
		WithDelayInMS:            1000,
	}

	cb := GetManualCircuitBreaker(config)
	assert.NotNil(t, cb, "Manual circuit breaker should not be nil")
}

func TestGetManualCircuitBreaker_UnsupportedVersion(t *testing.T) {
	config := &Config{
		Enabled: true,
		Version: 999, // unsupported
	}
	assert.Panics(t, func() { GetManualCircuitBreaker(config) }, "Should panic for unsupported version")
}

func TestManualFailSafeCB_Initialization(t *testing.T) {
	config := &Config{
		Enabled:                  true,
		Version:                  1,
		Name:                     "testManualCB",
		FailureRateThreshold:     50,
		FailureRateMinimumWindow: 20,
		FailureRateWindowInMs:    10000,
		SuccessCountThreshold:    75,
		SuccessCountWindow:       10,
		WithDelayInMS:            1000,
	}

	cb := manualFailSafeCB(config)
	assert.NotNil(t, cb, "manualFailSafeCB should not be nil")
}
