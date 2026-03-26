package manualcb

type CBConfig struct {
	CBName                        string
	FailureRateThreshold          int
	FailureExecutionThreshold     int
	FailureThresholdingPeriodInMS int
	SuccessRatioThreshold         int
	SuccessThresholdingCapacity   int
	WithDelayInMS                 int
}
