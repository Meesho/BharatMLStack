package circuitbreaker

type ManualCircuitBreaker interface {
	IsAllowed() bool
	RecordSuccess()
	RecordFailure()
	ForceOpen()
	ForceClose()
	NormalExecutionMode()
}
