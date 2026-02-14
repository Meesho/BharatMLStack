package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/rs/zerolog/log"
)

var (
	// It is safe to use one Client from multiple goroutines simultaneously
	statsDClient *statsd.Client = getDefaultClient()

	// by default full sampling
	samplingRate float64 = 0.0
)

func InitMetrics(configs *configs.AppConfigs) {
	var err error
	samplingRate, err = strconv.ParseFloat(configs.Configs.MetricsSamplingRate, 64)
	if err != nil {
		logger.Panic("Error parsing metrics sampling rate", err)
	}
	telegrafAddress := getTelegrafAddress(configs)
	globalTags := getGlobalTags(configs)

	statsDClient, err = statsd.New(
		telegrafAddress,
		statsd.WithTags(globalTags),
	)
	if err != nil {
		// In local/dev environments Telegraf may not be running; log and continue
		// with the default no-op-safe client instead of crashing the service.
		logger.Error("StatsD client initialization failed, metrics will be unavailable", err)
		statsDClient = getDefaultClient()
		return
	}
	//go initJMXServer()
	logger.Info(fmt.Sprintf("Metrics client initialized with telegraf address - %s, global tags - %v, and sampling rate - %f",
		telegrafAddress, globalTags, samplingRate))
}

func getDefaultClient() *statsd.Client {
	client, err := statsd.New("localhost:8125")
	if err != nil {
		// Return a no-op client so callers never hit nil-pointer panics.
		client, _ = statsd.New("localhost:8125", statsd.WithoutTelemetry())
	}
	return client
}

func getGlobalTags(configs *configs.AppConfigs) []string {
	return []string{
		"env:" + configs.Configs.ApplicationEnv,
		"service:" + configs.Configs.ApplicationName,
	}
}

func getTelegrafAddress(configs *configs.AppConfigs) string {
	host := configs.Configs.Telegraf_Host
	port := configs.Configs.Telegraf_Port
	return host + ":" + port
}

func Timing(name string, value time.Duration, tags []string) {
	err := statsDClient.Timing(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd timing", err)
	}
}

func Count(name string, value int64, tags []string) {
	err := statsDClient.Count(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd count", err)
	}
}

func Gauge(name string, value float64, tags []string) {
	err := statsDClient.Gauge(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd gauge", err)
	}
}
