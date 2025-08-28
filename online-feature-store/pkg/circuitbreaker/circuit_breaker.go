package circuitbreaker

type ManualCircuitBreaker interface {
	IsAllowed() bool
	RecordSuccess()
	RecordFailure()
}
