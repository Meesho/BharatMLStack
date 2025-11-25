package metric

import "strings"

// Tag constants
const (
	TagEnv     = "env"
	TagService = "service"
	TagPath    = "path"
	TagMethod  = "method"
	// Deprecated: Use TagHttpStatusCode or TagGrpcStatusCode instead
	TagStatusCode                   = "status_code"
	TagHttpStatusCode               = "http_status_code"
	TagGrpcStatusCode               = "grpc_status_code"
	TagExternalService              = "external_service"
	TagExternalServicePath          = "external_service_path"
	TagExternalServiceMethod        = "external_service_method"
	TagExternalServiceStatusCode    = "external_service_status_code"
	TagZkRealtimeTotalUpdateEvent   = "zk_realtime_total_update_event"
	TagZkRealtimeFailureEvent       = "zk_realtime_failure_event"
	TagZkRealtimeSuccessEvent       = "zk_realtime_success_event"
	TagZkRealtimeEventUpdateLatency = "zk_realtime_event_update_latency"
	TagCommunicationProtocol        = "communication_protocol"
	TagUserContext                  = "user_context"

	TagValueCommunicationProtocolHttp = "http"
	TagValueCommunicationProtocolGrpc = "grpc"
)

type Tag struct {
	Name  string
	Value string
}

func NewTag(name, value string) Tag {
	return Tag{
		Name:  name,
		Value: value,
	}
}

// BuildTag builds a tag from the given name and value
func BuildTag(tags ...Tag) []string {
	allTags := make([]string, 0)
	for _, tag := range tags {
		allTags = append(allTags, TagAsString(tag.Name, tag.Value))
	}
	return allTags
}

// normalizeTagValue sanitizes tag values to prevent parsing issues
func normalizeTagValue(value string) string {
	// Replace problematic characters that could be misinterpreted by DogStatsD/Telegraf
	// Note: "/" is kept as-is to preserve URL paths
	problematicChars := []string{":", " ", "\\", ",", "|", "@", "#"}
	normalized := value
	for _, char := range problematicChars {
		normalized = strings.ReplaceAll(normalized, char, "_")
	}
	return normalized
}

func TagAsString(name string, value string) string {
	return name + ":" + normalizeTagValue(value)
}

func UpdateTags(tags *[]string, newTags ...Tag) {
	for _, tag := range newTags {
		*tags = append(*tags, TagAsString(tag.Name, tag.Value))
	}
}
