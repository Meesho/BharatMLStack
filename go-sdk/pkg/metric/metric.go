package metric

import (
	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"strconv"
	"sync"
	"time"
)

const (
	// Deprecated: Use ApiRequestCount instead
	HttpRequestCount = "http_request_count"
	// Deprecated: Use ApiRequestLatency instead
	HttpRequestLatency = "http_request_latency"
	// Deprecated: Use ExternalApiRequestCount instead
	ExternalHttpRequestCount = "external_http_request_count"
	// Deprecated: Use ExternalApiRequestLatency instead
	ExternalHttpRequestLatency = "external_http_request_latency"
	ApiRequestCount            = "api_request_count"
	ApiRequestLatency          = "api_request_latency"
	ExternalApiRequestCount    = "external_api_request_count"
	ExternalApiRequestLatency  = "external_api_request_latency"

	// Deprecated: Use DBCallLatency instead
	RedisLatency = "redis_latency"

	// Deprecated: Use DBCallCount instead
	RedisSuccessCount = "redis_success_count"

	// Deprecated: Use DBCallCount instead
	RedisFailureCount = "redis_failure_count"

	DBCallLatency = "db_call_latency"
	DBCallCount   = "db_call_count"
	MethodLatency = "method_latency"
	MethodCount   = "method_count"
	ServiceName   = "service_name"
)

var (
	// it is safe to use one client from multiple goroutines simultaneously
	statsDClient = getDefaultClient()
	// by default full sampling
	samplingRate    = 0.0
	telegrafAddress = "localhost:8125"
	appName         = ""
	initialized     = false
	once            sync.Once
)

// Init initializes the metrics client
func Init() {
	if initialized {
		log.Debug().Msgf("Metrics already initialized!")
		return
	}
	once.Do(func() {
		var err error
		samplingRate = viper.GetFloat64("APP_METRIC_SAMPLING_RATE")
		appName = viper.GetString("APP_NAME")
		globalTags := getGlobalTags()

		statsDClient, err = statsd.New(
			telegrafAddress,
			statsd.WithTags(globalTags),
		)

		if err != nil {
			log.Panic().AnErr("StatsD client initialization failed", err)
		}
		log.Info().Msgf("Metrics client initialized with telegraf address - %s, global tags - %v, and "+
			"sampling rate - %f", telegrafAddress, globalTags, samplingRate)
		initialized = true
	})
}

func getDefaultClient() *statsd.Client {
	client, _ := statsd.New("localhost:8125")
	return client
}

func getGlobalTags() []string {
	env := viper.GetString("APP_ENV")
	if len(env) == 0 {
		log.Warn().Msg("APP_ENV is not set")
	}
	service := viper.GetString("APP_NAME")
	if len(service) == 0 {
		log.Warn().Msg("APP_NAME is not set")
	}
	return []string{
		TagAsString(TagEnv, env),
		TagAsString(TagService, service),
	}
}

// Timing sends timing information
func Timing(name string, value time.Duration, tags []string) {
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Timing(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd timing", err)
	}
}

// TimingWithStart is a handy func when we want to measure latency of a function
// Can be used as 'defer metric.TimingWithStart("metric_name", time.Now(), []string{})' at the start of the function
func TimingWithStart(name string, startTime time.Time, tags []string) {
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Timing(name, time.Since(startTime), tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd timing", err)
	}
}

// Count Increases metric counter by value
func Count(name string, value int64, tags []string) {
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Count(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd count", err)
	}
}

// Incr Increases metric counter by 1
func Incr(name string, tags []string) {
	Count(name, 1, tags)
}

// Decr Decreases metric counter by 1
func Decr(name string, tags []string) {
	Count(name, -1, tags)
}

func Gauge(name string, value float64, tags []string) {
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Gauge(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd gauge", err)
	}
}

func BuildExternalHTTPServiceLatencyTags(service, path, method string, statusCode int) []string {
	return BuildTag(
		NewTag(TagCommunicationProtocol, TagValueCommunicationProtocolHttp),
		NewTag(TagExternalService, service),
		NewTag(TagPath, path),
		NewTag(TagMethod, method),
		NewTag(TagHttpStatusCode, strconv.Itoa(statusCode)),
	)
}

func BuildExternalHTTPServiceCountTags(service, path, method string, statusCode int) []string {
	return BuildTag(
		NewTag(TagCommunicationProtocol, TagValueCommunicationProtocolHttp),
		NewTag(TagExternalService, service),
		NewTag(TagPath, path),
		NewTag(TagMethod, method),
		NewTag(TagHttpStatusCode, strconv.Itoa(statusCode)),
	)
}

func BuildExternalGRPCServiceLatencyTags(service, method string, statusCode int) []string {
	return BuildTag(
		NewTag(TagCommunicationProtocol, TagValueCommunicationProtocolGrpc),
		NewTag(TagExternalService, service),
		NewTag(TagMethod, method),
		NewTag(TagGrpcStatusCode, strconv.Itoa(statusCode)),
	)
}

func BuildExternalGRPCServiceCountTags(service, method string, statusCode int) []string {
	return BuildTag(
		NewTag(TagCommunicationProtocol, TagValueCommunicationProtocolGrpc),
		NewTag(TagExternalService, service),
		NewTag(TagMethod, method),
		NewTag(TagGrpcStatusCode, strconv.Itoa(statusCode)),
	)
}
