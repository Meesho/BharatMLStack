package circuitbreaker

const (
	CBEnabled                  = "_CB_ENABLED"
	CBName                     = "_CB_NAME"
	CBFailureCountThreshold    = "_CB_FAILURE_COUNT_THRESHOLD"
	CBFailureRateThreshold     = "_CB_FAILURE_RATE_THRESHOLD"
	CBFailureRateMinimumWindow = "_CB_FAILURE_RATE_MINIMUM_WINDOW"
	CBFailureRateWindowInMs    = "_CB_FAILURE_RATE_WINDOW_IN_MS"
	CBFailureCountWindow       = "_CB_FAILURE_COUNT_WINDOW"
	CBSuccessCountThreshold    = "_CB_SUCCESS_COUNT_THRESHOLD"
	CBSuccessCountWindow       = "_CB_SUCCESS_COUNT_WINDOW"
	CBVersion                  = "_CB_VERSION"
	CBWithDelayInMS            = "_CB_WITH_DELAY_IN_MS"
)
