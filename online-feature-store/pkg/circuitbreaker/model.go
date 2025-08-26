package circuitbreaker

type Config struct {
	// Enabled determines whether the circuit breaker is active.
	// When set to false, the circuit breaker logic is bypassed, and all requests pass through.
	Enabled bool `json:"enabled"`

	// Name is a unique identifier for the circuit breaker.
	// It is useful for logging, monitoring, and distinguishing between multiple circuit breakers in the system.
	Name string `json:"name"`

	// Version specifies the configuration version for the circuit breaker.
	// This can help manage multiple circuit breaker implementations or allow rolling out new configurations
	// without modifying the serving code.
	Version int `json:"version"`

	// FailureCountThreshold defines the failure ratio used for count-based failure thresholding.
	// This is the numerator in the ratio of failures to total executions that must be met or exceeded
	// to trip the circuit breaker into the OpenState.
	// Example: Setting FailureCountThreshold to 5 and FailureCountWindow to 10 means that the circuit will open
	// if 5 out of the last 10 executions fail.
	FailureCountThreshold int `json:"failure-count-threshold"`

	// FailureCountWindow sets the denominator for the failure ratio threshold calculation.
	// This is the total number of recent executions considered when determining whether to trip the circuit.
	// Example: In combination with FailureCountThreshold, it ensures thresholds are evaluated over a defined capacity window.
	FailureCountWindow int `json:"failure-count-window"`

	// FailureRateThreshold specifies the percentage of failures (from 1 to 100) required for time-based failure thresholding.
	// The circuit breaker will trip into the OpenState if the failure rate exceeds this threshold within
	// the FailureRateWindowInMs, provided that the number of executions exceeds FailureRateMinimumWindow.
	// Example: If set to 50, the circuit breaker opens if the failure rate exceeds 50% of FailureRateMinimumWindow during the thresholding period.
	FailureRateThreshold int `json:"failure-rate-threshold"`

	// FailureRateMinimumWindow sets the minimum number of executions required before
	// evaluating the failure rate threshold. This prevents premature circuit tripping in low-traffic scenarios.
	// Example: If set to 20, the circuit breaker will not trip unless at least 20 executions are recorded in the thresholding period.
	FailureRateMinimumWindow int `json:"failure-rate-minimum-window"`

	// FailureRateWindowInMs defines the rolling time window (in milliseconds) over which
	// failures and successes are measured to calculate thresholds.
	// Example: A value of 10,000 (10 seconds) means the circuit breaker will calculate failure thresholds
	// based on executions occurring within this window.
	FailureRateWindowInMs int `json:"failure-rate-window-in-ms"`

	// SuccessCountThreshold specifies the success ratio used for recovery in HalfOpenState.
	// The circuit breaker will transition back to ClosedState if the success ratio during the recovery period
	// meets or exceeds this threshold.
	// Example: If set to 75, at least 75% of SuccessCountWindow executions must succeed for recovery.
	SuccessCountThreshold int `json:"success-count-threshold"`

	// SuccessCountWindow sets the total number of executions considered when evaluating the success ratio
	// in HalfOpenState.
	// Example: When combined with SuccessCountThreshold, it determines how many successful executions
	// must occur for the circuit breaker to transition back to ClosedState.
	SuccessCountWindow int `json:"success-count-window"`

	// WithDelayInMS specifies the delay (in milliseconds) for state transitions.
	// This delay applies when transitioning from one state to another, such as moving to HalfOpenState
	// or back to ClosedState, to allow for system stabilization.
	// Example: Setting this to 1,000 introduces a 1-second delay before retrying after a state change.
	WithDelayInMS int `json:"with-delay-in-ms"`

	// ActiveCBs maps circuit breaker names to their enabled status
	// Example: {"retrieve_from_distributed_cache": true, "write_to_cache": false}
	ActiveCBs map[string]bool `json:"active-cbs"`
}

func BuildConfig(serviceName string) *Config {
	cbConfig := Config{
		Enabled: true,
		Name:    serviceName,
		Version: 1,
	}

	return &cbConfig
}
