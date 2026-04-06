package manualcb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPassThroughBreaker(t *testing.T) {
	breaker := NewPassThroughBreaker()

	assert.NotNil(t, breaker, "NewPassThroughBreaker should not return nil")

	// IsAllowed should always be true
	assert.True(t, breaker.IsAllowed(), "IsAllowed should always return true")

	// RecordSuccess and RecordFailure should not panic
	assert.NotPanics(t, func() {
		breaker.RecordSuccess()
	}, "RecordSuccess should not panic")

	assert.NotPanics(t, func() {
		breaker.RecordFailure()
	}, "RecordFailure should not panic")
}
