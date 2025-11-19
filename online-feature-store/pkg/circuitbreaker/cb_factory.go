package circuitbreaker

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/circuitbreaker/manualcb"
	"github.com/rs/zerolog/log"
)

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
