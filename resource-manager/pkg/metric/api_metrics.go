package metric

import (
	"strconv"
	"time"
)

const (
	APIRequestCount   = "api_request_count"
	APIRequestLatency = "api_request_latency"
)

func ObserveAPIRequest(path, method string, statusCode int, latency time.Duration) {
	tags := BuildTag(
		NewTag(TagPath, path),
		NewTag(TagMethod, method),
		NewTag(TagStatusCode, strconv.Itoa(statusCode)),
		NewTag(TagHttpStatusCode, strconv.Itoa(statusCode)),
		NewTag(TagCommunicationProtocol, TagValueCommunicationProtocolHttp),
	)

	Incr(APIRequestCount, tags)
	Timing(APIRequestLatency, latency, tags)
}
