package circuitbreaker

import (
	"github.com/Meesho/BharatMLStack/skye/pkg/circuitbreaker/failsafecb"
	"github.com/Meesho/BharatMLStack/skye/pkg/circuitbreaker/manualcb"
	"github.com/rs/zerolog/log"
)

func GetCircuitBreaker[T, R any](config *Config) CircuitBreaker[T, R] {
	switch config.Version {
	case 1:
		return failSafeCB[T, R](config)
	default:
		log.Panic().Msgf("Circuit breaker version %d not supported", config.Version)
	}
	return nil
}

func failSafeCB[T, R any](config *Config) CircuitBreaker[T, R] {
	fsCbConfig := &failsafecb.CBConfig{
		CBName:                        config.Name,
		FailureRateThreshold:          config.FailureRateThreshold,
		FailureExecutionThreshold:     config.FailureRateMinimumWindow,
		FailureThresholdingPeriodInMS: config.FailureRateWindowInMs,
		SuccessRatioThreshold:         config.SuccessCountThreshold,
		SuccessThresholdingCapacity:   config.SuccessCountWindow,
		WithDelayInMS:                 config.WithDelayInMS,
	}
	return failsafecb.NewFailSafe[T, R](fsCbConfig)
}

func GetManualCircuitBreaker(config *Config) ManualCircuitBreaker {
	if config == nil {
		return nil
	}

	if !config.Enabled {
		return manualcb.NewPassThroughBreaker()
	}

	switch config.Version {
	case 1:
		return manualFailSafeCB(config)
	default:
		log.Panic().Msgf("Circuit breaker version %d not supported", config.Version)
	}
	return nil
}

func manualFailSafeCB(config *Config) ManualCircuitBreaker {
	cbConfig := &manualcb.CBConfig{
		CBName:                        config.Name,
		FailureRateThreshold:          config.FailureRateThreshold,
		FailureExecutionThreshold:     config.FailureRateMinimumWindow,
		FailureThresholdingPeriodInMS: config.FailureRateWindowInMs,
		SuccessRatioThreshold:         config.SuccessCountThreshold,
		SuccessThresholdingCapacity:   config.SuccessCountWindow,
		WithDelayInMS:                 config.WithDelayInMS,
	}
	return manualcb.NewManualFailsafeBreaker(cbConfig)
}
