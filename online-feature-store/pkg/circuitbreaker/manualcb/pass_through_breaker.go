package manualcb

// passThroughBreaker is an implementation of ManualCircuitBreaker that does nothing.
// It is used when a circuit breaker is disabled via configuration. It allows all requests to pass through.
type passThroughBreaker struct{}

func NewPassThroughBreaker() *passThroughBreaker {
	return &passThroughBreaker{}
}

// IsAllowed always returns true.
func (nb *passThroughBreaker) IsAllowed() bool {
	return true
}

// RecordSuccess does nothing.
func (nb *passThroughBreaker) RecordSuccess() {}

// RecordFailure does nothing.
func (nb *passThroughBreaker) RecordFailure() {}

// ForceOpen does nothing for pass-through breaker.
func (nb *passThroughBreaker) ForceOpen() {}

// ForceClose does nothing for pass-through breaker.
func (nb *passThroughBreaker) ForceClose() {}

// Execute does nothing for pass-through breaker.
func (nb *passThroughBreaker) NormalExecutionMode() {}
