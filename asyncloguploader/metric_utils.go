package asyncloguploader

import (
	"github.com/Meesho/go-core/metric"
)

const (
	// Metric names
	MetricLogBytes                   = "logBytes"
	MetricLogBytesDropped            = "logBytesDropped"
	MetricLogBytesSuccess            = "logBytesSuccess"
	MetricLogBytesWritten            = "logBytesWritten"
	MetricLogBytesTimeout            = "logBytesTimeout"
	MetricLogBytesError              = "logBytesError"
	MetricLogBytesClose              = "logBytesClose"
	MetricLoggerInitialized          = "loggerInitialized"
	MetricLoggerInitializationFailed = "loggerInitializationFailed"
	MetricLogBytesSwap               = "logBytesSwap"
	MetricLogBytesFlushAttempts      = "logBytesFlushAttempts"
	MetricLogBytesFlushSuccess       = "logBytesFlushSuccess"
	MetricLogBytesFlushFailure       = "logBytesFlushFailure"
	MetricLogBytesFlushDuration      = "logBytesFlushDuration"
	MetricFileWriterWriteDuration    = "fileWriterWriteDuration"
	MetricFileWriterRotationCount    = "fileWriterRotationCount"
	MetricUploadFile                 = "uploadFile"
	MetricUploadFileFailed           = "uploadFileFailed"
	MetricUploadFileDuration         = "uploadFileDuration"
	MetricUploadBytes                = "uploadBytes"
	MetricFileRenameFailed           = "fileRenameFailed"

	// SSD lifecycle metrics
	MetricSSDClaimSuccess      = "ssdClaimSuccess"
	MetricSSDClaimFailed       = "ssdClaimFailed"
	MetricSSDRenewalFailed     = "ssdRenewalFailed"
	MetricSSDOrphanTmpRecovered = "ssdOrphanTmpRecovered"
	MetricSSDReleased          = "ssdReleased"
)

func GetLoggerTags(service string, modelId string) []string {
	return metric.BuildTag(metric.NewTag("service", service),
		metric.NewTag("model_id", modelId))
}

func GetEventNameTags(eventName string) []string {
	return metric.BuildTag(metric.NewTag("event_name", eventName))
}

// getBaseTags returns tags or empty slice if nil/empty (for use when no event context)
func getBaseTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	return tags
}

// MergeMetricTags merges base tags with event_name tag for event-scoped metrics.
// If eventName is empty, returns a copy of baseTags.
func MergeMetricTags(baseTags []string, eventName string) []string {
	if eventName == "" {
		if len(baseTags) == 0 {
			return []string{}
		}
		merged := make([]string, len(baseTags))
		copy(merged, baseTags)
		return merged
	}
	eventTags := GetEventNameTags(eventName)
	if len(baseTags) == 0 {
		return eventTags
	}
	merged := make([]string, 0, len(baseTags)+len(eventTags))
	merged = append(merged, baseTags...)
	merged = append(merged, eventTags...)
	return merged
}
