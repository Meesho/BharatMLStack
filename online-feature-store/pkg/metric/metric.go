package metric

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	ExternalApiRequestCount   = "external_api_request_count"
	ExternalApiRequestLatency = "external_api_request_latency"
)

//TODO: implement additional methods needed for Observability

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

func Gauge(name string, value float64, tags []string) {
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Gauge(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd gauge", err)
	}
}

// CountBatch sends multiple count metrics efficiently for different tag combinations
func CountBatch(name string, value int64, baseTags []string, tagKey string, tagValues []string) {
	for _, tagValue := range tagValues {
		tags := append(baseTags, tagKey+":"+tagValue)
		Count(name, value, tags)
	}
}
