package circuitbreaker

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config defines the configuration for a circuit breaker. It provides parameters
// for controlling circuit breaker behavior, including thresholds for failures and successes,
// state transition delays, and windowing periods for threshold calculations.
// This configuration is designed to be generic, enabling adaptability to various circuit breaker implementations.
type Config struct {
	// Enabled determines whether the circuit breaker is active.
	// When set to false, the circuit breaker logic is bypassed, and all requests pass through.
	Enabled bool

	// Name is a unique identifier for the circuit breaker.
	// It is useful for logging, monitoring, and distinguishing between multiple circuit breakers in the system.
	Name string

	// Version specifies the configuration version for the circuit breaker.
	// This can help manage multiple circuit breaker implementations or allow rolling out new configurations
	// without modifying the serving code.
	Version int

	// FailureCountThreshold defines the failure ratio used for count-based failure thresholding.
	// This is the numerator in the ratio of failures to total executions that must be met or exceeded
	// to trip the circuit breaker into the OpenState.
	// Example: Setting FailureCountThreshold to 5 and FailureCountWindow to 10 means that the circuit will open
	// if 5 out of the last 10 executions fail.
	FailureCountThreshold int

	// FailureCountWindow sets the denominator for the failure ratio threshold calculation.
	// This is the total number of recent executions considered when determining whether to trip the circuit.
	// Example: In combination with FailureCountThreshold, it ensures thresholds are evaluated over a defined capacity window.
	FailureCountWindow int

	// FailureRateThreshold specifies the percentage of failures (from 1 to 100) required for time-based failure thresholding.
	// The circuit breaker will trip into the OpenState if the failure rate exceeds this threshold within
	// the FailureRateWindowInMs, provided that the number of executions exceeds FailureRateMinimumWindow.
	// Example: If set to 50, the circuit breaker opens if the failure rate exceeds 50% of FailureRateMinimumWindow during the thresholding period.
	FailureRateThreshold int

	// FailureRateMinimumWindow sets the minimum number of executions required before
	// evaluating the failure rate threshold. This prevents premature circuit tripping in low-traffic scenarios.
	// Example: If set to 20, the circuit breaker will not trip unless at least 20 executions are recorded in the thresholding period.
	FailureRateMinimumWindow int

	// FailureRateWindowInMs defines the rolling time window (in milliseconds) over which
	// failures and successes are measured to calculate thresholds.
	// Example: A value of 10,000 (10 seconds) means the circuit breaker will calculate failure thresholds
	// based on executions occurring within this window.
	FailureRateWindowInMs int

	// SuccessCountThreshold specifies the success ratio used for recovery in HalfOpenState.
	// The circuit breaker will transition back to ClosedState if the success ratio during the recovery period
	// meets or exceeds this threshold.
	// Example: If set to 75, at least 75% of SuccessCountWindow executions must succeed for recovery.
	SuccessCountThreshold int

	// SuccessCountWindow sets the total number of executions considered when evaluating the success ratio
	// in HalfOpenState.
	// Example: When combined with SuccessCountThreshold, it determines how many successful executions
	// must occur for the circuit breaker to transition back to ClosedState.
	SuccessCountWindow int

	// WithDelayInMS specifies the delay (in milliseconds) for state transitions.
	// This delay applies when transitioning from one state to another, such as moving to HalfOpenState
	// or back to ClosedState, to allow for system stabilization.
	// Example: Setting this to 1,000 introduces a 1-second delay before retrying after a state change.
	WithDelayInMS int
}

func BuildConfig(serviceName string) *Config {
	cbConfig := Config{
		Enabled: false,
	}

	// Check if circuit breaker is enabled
	if viper.IsSet(serviceName+CBEnabled) && viper.GetBool(serviceName+CBEnabled) {
		cbConfig.Enabled = true
		validateConfigs(serviceName, cbConfig)
		// Load configuration properties
		cbConfig.Name = viper.GetString(serviceName + CBName)
		cbConfig.FailureRateThreshold = viper.GetInt(serviceName + CBFailureRateThreshold)
		cbConfig.FailureRateMinimumWindow = viper.GetInt(serviceName + CBFailureRateMinimumWindow)
		cbConfig.FailureRateWindowInMs = viper.GetInt(serviceName + CBFailureRateWindowInMs)
		cbConfig.FailureCountThreshold = viper.GetInt(serviceName + CBFailureCountThreshold)
		cbConfig.FailureCountWindow = viper.GetInt(serviceName + CBFailureCountWindow)
		cbConfig.SuccessCountThreshold = viper.GetInt(serviceName + CBSuccessCountThreshold)
		cbConfig.SuccessCountWindow = viper.GetInt(serviceName + CBSuccessCountWindow)
		cbConfig.WithDelayInMS = viper.GetInt(serviceName + CBWithDelayInMS)
		cbConfig.Version = viper.GetInt(serviceName + CBVersion)
		// Validation: Ensure either time-based or count-based failure threshold is set
		if (cbConfig.FailureRateThreshold == 0 || cbConfig.FailureRateMinimumWindow == 0 || cbConfig.FailureRateWindowInMs == 0) &&
			(cbConfig.FailureCountThreshold == 0 || cbConfig.FailureCountWindow == 0) {
			log.Panic().Msgf("%s: Configuration invalid, neither time-based nor count-based failure thresholds are fully defined", serviceName)
		}
	}

	return &cbConfig
}

func validateConfigs(serviceName string, cbConfig Config) {
	// Mandatory configuration checks
	if !viper.IsSet(serviceName + CBName) {
		log.Panic().Msgf("%s-%s not set", serviceName, CBName)
	}
	if !viper.IsSet(serviceName+CBFailureRateThreshold) && !viper.IsSet(serviceName+CBFailureCountThreshold) {
		log.Panic().Msgf("%s: Neither time-based nor count-based failure thresholds are set", serviceName)
	}
	if !viper.IsSet(serviceName+CBFailureRateMinimumWindow) && viper.IsSet(serviceName+CBFailureRateThreshold) {
		log.Panic().Msgf("%s-%s not set, required for time-based failure thresholding", serviceName, CBFailureRateMinimumWindow)
	}
	if !viper.IsSet(serviceName+CBFailureRateWindowInMs) && viper.IsSet(serviceName+CBFailureRateThreshold) {
		log.Panic().Msgf("%s-%s not set, required for time-based failure thresholding", serviceName, CBFailureRateWindowInMs)
	}
	if !viper.IsSet(serviceName + CBSuccessCountThreshold) {
		log.Panic().Msgf("%s-%s not set", serviceName, CBSuccessCountThreshold)
	}
	if !viper.IsSet(serviceName + CBSuccessCountWindow) {
		log.Panic().Msgf("%s-%s not set", serviceName, CBSuccessCountWindow)
	}
	if !viper.IsSet(serviceName + CBVersion) {
		log.Panic().Msgf("%s-%s not set", serviceName, CBVersion)
	}
	if !viper.IsSet(serviceName + CBWithDelayInMS) {
		log.Panic().Msgf("%s-%s not set", serviceName, CBWithDelayInMS)
	}
}
