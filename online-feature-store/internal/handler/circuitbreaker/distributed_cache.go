package circuitbreaker

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/circuitbreaker"
	"github.com/rs/zerolog/log"
)

type DFCircuitBreakerHandler struct {
	CBManager circuitbreaker.Manager
}

func NewDFCircuitBreakerHandler(cbManager circuitbreaker.Manager) *DFCircuitBreakerHandler {
	return &DFCircuitBreakerHandler{
		CBManager: cbManager,
	}
}

func (h *DFCircuitBreakerHandler) GetCB(cbKey string) (circuitbreaker.ManualCircuitBreaker, error) {
	return h.CBManager.GetOrCreateManualCB(cbKey)
}

func (h *DFCircuitBreakerHandler) IsCBEnabled(cbKey string) bool {
	return h.CBManager.IsCBEnabled(cbKey)
}

func (h *DFCircuitBreakerHandler) IsForcedOpen() bool {
	return h.CBManager.IsForcedOpen()
}

func (h *DFCircuitBreakerHandler) RecordFailure(cbKey string) {
	cb, cbErr := h.GetCB(cbKey)
	if cbErr != nil {
		log.Error().Err(cbErr).Msgf("Error getting circuit breaker %s", cbKey)
		return
	}
	cb.RecordFailure()
}
func (h *DFCircuitBreakerHandler) RecordSuccess(cbKey string) {
	cb, cbErr := h.GetCB(cbKey)
	if cbErr != nil {
		log.Error().Err(cbErr).Msgf("Error getting circuit breaker %s", cbKey)
		return
	}
	cb.RecordSuccess()
}

// IsCallAllowed checks if the circuit breaker allows the call and handles all CB-related logic
func (h *DFCircuitBreakerHandler) IsCallAllowed(cbKey string) bool {
	cb, cbErr := h.GetCB(cbKey)
	if cbErr != nil {
		log.Error().Err(cbErr).Msgf("Error getting circuit breaker %s", cbKey)
		return false
	}
	if !cb.IsAllowed() {
		log.Info().Msgf("Circuit breaker %s is not allowed", cbKey)
		return false
	}
	return true
}
