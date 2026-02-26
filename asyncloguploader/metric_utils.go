package asyncloguploader

import (
	"github.com/Meesho/go-core/metric"
)

const (
	// Metric names
	MetricLogBytes                   = "logBytes"
	MetricLogBytesDropped            = "logBytesDropped"
	MetricLogBytesSuccess            = "logBytesSuccess"
	MetricLogBytesTimeout            = "logBytesTimeout"
	MetricLogBytesError              = "logBytesError"
	MetricLogBytesClose              = "logBytesClose"
	MetricLoggerInitialized          = "loggerInitialized"
	MetricLoggerInitializationFailed = "loggerInitializationFailed"
	MetricLogBytesSwap               = "logBytesSwap"
	MetricLogBytesFlush              = "logBytesFlush"
	MetricLogBytesFlushDuration      = "logBytesFlushDuration"
	MetricUploadFileDuration         = "uploadFileDuration"
	MetricUploadFile                 = "uploadFile"
)

func GetLoggerTags(service string, modelId string) []string {
	return metric.BuildTag(metric.NewTag("service", service),
		metric.NewTag("model_id", modelId))
}

func GetEventNameTags(eventName string) []string {
	return metric.BuildTag(metric.NewTag("event_name", eventName))
}
