package manualcb

import (
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	fscb "github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/rs/zerolog/log"
)

// failsafeBreaker is a wrapper around a failsafe-go circuit breaker that
// implements both the CircuitBreaker and ManualCircuitBreaker interfaces.
type failsafeBreaker struct {
	breaker fscb.CircuitBreaker[any]
}

// newFailsafeBreaker builds a circuit breaker from a validated configuration.
func NewManualFailsafeBreaker(config *CBConfig) *failsafeBreaker {
	cb := fscb.Builder[any]().
		WithFailureRateThreshold(uint(config.FailureRateThreshold), uint(config.FailureExecutionThreshold), time.Duration(config.FailureThresholdingPeriodInMS)*time.Millisecond).
		WithSuccessThresholdRatio(uint(config.SuccessRatioThreshold), uint(config.SuccessThresholdingCapacity)).
		WithDelay(time.Duration(config.WithDelayInMS) * time.Millisecond).
		OnStateChanged(func(event fscb.StateChangedEvent) {
			log.Debug().Msgf("Circuit Breaker '%s' changed state from %s to %s\n", config.CBName, event.OldState, event.NewState)
			metric.Incr("DAG_CB_STATE_CHANGED", []string{"name", config.CBName, "from", event.OldState.String(), "to", event.NewState.String()})
		}).
		Build()
	f := &failsafeBreaker{
		breaker: cb,
	}
	return f
}

// IsAllowed returns true if a request is permitted.
func (b *failsafeBreaker) IsAllowed() bool {
	return b.breaker.TryAcquirePermit()
}

// RecordSuccess records a successful execution.
func (b *failsafeBreaker) RecordSuccess() {
	b.breaker.RecordSuccess()
}

// RecordFailure records a failed execution.
func (b *failsafeBreaker) RecordFailure() {
	b.breaker.RecordFailure()
}
