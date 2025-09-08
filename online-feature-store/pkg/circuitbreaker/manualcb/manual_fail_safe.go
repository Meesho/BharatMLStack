package manualcb

import (
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	fscb "github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/rs/zerolog/log"
)

// failsafeBreaker is a wrapper around a failsafe-go circuit breaker that
// implements both the CircuitBreaker and ManualCircuitBreaker interfaces.
type failsafeBreaker struct {
	breaker            fscb.CircuitBreaker[any]
	stateChangeAllowed bool
	forceStateMu       sync.RWMutex
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
		breaker:            cb,
		stateChangeAllowed: true,
	}
	return f
}

// IsAllowed returns true if a request is permitted.
func (b *failsafeBreaker) IsAllowed() bool {
	if !b.stateChangeAllowed {
		return b.breaker.IsClosed()
	}
	return b.breaker.TryAcquirePermit()
}

// RecordSuccess records a successful execution.
func (b *failsafeBreaker) RecordSuccess() {
	if b.stateChangeAllowed {
		b.breaker.RecordSuccess()
	}
}

// RecordFailure records a failed execution.
func (b *failsafeBreaker) RecordFailure() {
	if b.stateChangeAllowed {
		b.breaker.RecordFailure()
	}
}

// ForceOpen forces the circuit breaker to open state, denying all requests.
func (b *failsafeBreaker) ForceOpen() {
	b.forceStateMu.Lock()
	defer b.forceStateMu.Unlock()
	b.breaker.Open()
	b.stateChangeAllowed = false
	log.Info().Msg("Circuit breaker force opened")
}

// ForceClose forces the circuit breaker to closed state, allowing all requests.
func (b *failsafeBreaker) ForceClose() {
	b.forceStateMu.Lock()
	defer b.forceStateMu.Unlock()
	b.breaker.Close()
	b.stateChangeAllowed = false
	log.Info().Msg("Circuit breaker force closed")
}

// NormalExecutionMode removes any forced state and allows normal circuit breaker operation.
func (b *failsafeBreaker) NormalExecutionMode() {
	b.forceStateMu.Lock()
	defer b.forceStateMu.Unlock()
	b.breaker.Close()
	b.stateChangeAllowed = true
	log.Info().Msg("Circuit breaker in normal execution mode")
}
