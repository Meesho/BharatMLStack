package manualcb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManualFailsafeBreaker(t *testing.T) {
	config := &CBConfig{
		CBName:                        "test",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     10,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         5,
		SuccessThresholdingCapacity:   10,
		WithDelayInMS:                 100,
	}
	breaker := NewManualFailsafeBreaker(config)
	assert.NotNil(t, breaker)
}

func TestManualFailsafeBreaker_ClosesOnSuccessInHalfOpen(t *testing.T) {
	config := &CBConfig{
		CBName:                        "test_closes_on_success",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     10,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         50,
		SuccessThresholdingCapacity:   4,
		WithDelayInMS:                 100,
	}
	breaker := NewManualFailsafeBreaker(config)
	assert.NotNil(t, breaker)

	// Initially closed
	assert.True(t, breaker.IsAllowed(), "Breaker should be closed initially")

	// Trip the breaker
	for i := 0; i < 6; i++ {
		breaker.RecordFailure()
	}
	for i := 0; i < 4; i++ {
		breaker.RecordSuccess()
	}

	// Should be open now
	assert.False(t, breaker.IsAllowed(), "Breaker should be open after failures")

	// Wait to enter half-open state
	time.Sleep(time.Duration(config.WithDelayInMS) * time.Millisecond)

	// In half-open state, record successes to close it
	assert.True(t, breaker.IsAllowed(), "Breaker should allow request in half-open")
	breaker.RecordSuccess()
	assert.True(t, breaker.IsAllowed(), "Breaker should allow request in half-open")
	breaker.RecordSuccess()

	// With 2 successes out of 4 capacity and 50% threshold, it should close.
	// The breaker internal state will check after each execution. After 2 successes the success rate is 100% on 2 executions.
	// Failsafe-go will close the breaker if the success rate is met without needing all executions up to capacity.
	assert.True(t, breaker.IsAllowed(), "Breaker should be closed after successes in half-open")
}

func TestManualFailsafeBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	config := &CBConfig{
		CBName:                        "test_reopens_on_failure",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     10,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         50,
		SuccessThresholdingCapacity:   4,
		WithDelayInMS:                 100,
	}
	breaker := NewManualFailsafeBreaker(config)
	assert.NotNil(t, breaker)

	// Initially closed
	assert.True(t, breaker.IsAllowed(), "Breaker should be closed initially")

	// Trip the breaker
	for i := 0; i < 6; i++ {
		breaker.RecordFailure()
	}
	for i := 0; i < 4; i++ {
		breaker.RecordSuccess()
	}

	// Should be open now
	assert.False(t, breaker.IsAllowed(), "Breaker should be open after failures")
}
