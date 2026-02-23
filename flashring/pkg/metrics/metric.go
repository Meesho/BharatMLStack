package metrics

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Flashring metric keys
const (
	KEY_GET_LATENCY     = "flashring_get_latency"
	KEY_PUT_LATENCY     = "flashring_put_latency"
	KEY_RTHROUGHPUT     = "flashring_rthroughput"
	KEY_WTHROUGHPUT     = "flashring_wthroughput"
	KEY_HITRATE         = "flashring_hitrate"
	KEY_ACTIVE_ENTRIES  = "flashring_active_entries"
	KEY_EXPIRED_ENTRIES = "flashring_expired_entries"
	KEY_REWRITES        = "flashring_rewrites"
	KEY_GETS            = "flashring_gets"
	KEY_PUTS            = "flashring_puts"
	KEY_HITS            = "flashring_hits"

	KEY_KEY_NOT_FOUND_COUNT = "flashring_key_not_found_count"
	KEY_KEY_EXPIRED_COUNT   = "flashring_key_expired_count"
	KEY_BAD_DATA_COUNT      = "flashring_bad_data_count"
	KEY_BAD_LENGTH_COUNT    = "flashring_bad_length_count"
	KEY_BAD_CR32_COUNT      = "flashring_bad_cr32_count"
	KEY_BAD_KEY_COUNT       = "flashring_bad_key_count"
	KEY_DELETED_KEY_COUNT   = "flashring_deleted_key_count"

	KEY_WRITE_COUNT      = "flashring_write_count"
	KEY_PUNCH_HOLE_COUNT = "flashring_punch_hole_count"
	KEY_PREAD_COUNT      = "flashring_pread_count"

	KEY_TRIM_HEAD_LATENCY = "flashring_wrap_file_trim_head_latency"
	KEY_PREAD_LATENCY     = "flashring_pread_latency"
	KEY_PWRITE_LATENCY    = "flashring_pwrite_latency"

	KEY_MEMTABLE_FLUSH_COUNT = "flashring_memtable_flush_count"

	LATENCY_RLOCK = "flashring_rlock_latency"
	LATENCY_WLOCK = "flashring_wlock_latency"

	KEY_RINGBUFFER_ACTIVE_ENTRIES = "flashring_ringbuffer_active_entries"
	KEY_MEMTABLE_ENTRY_COUNT      = "flashring_memtable_entry_count"
	KEY_MEMTABLE_HIT              = "flashring_memtable_hit"
	KEY_MEMTABLE_MISS             = "flashring_memtable_miss"
	KEY_DATA_LENGTH               = "flashring_data_length"
	KEY_IOURING_SIZE              = "flashring_iouring_size"
)

// Flashring tag keys
const (
	TAG_LATENCY_PERCENTILE = "latency_percentile"
	TAG_VALUE_P25          = "p25"
	TAG_VALUE_P50          = "p50"
	TAG_VALUE_P99          = "p99"
	TAG_SHARD_IDX          = "shard_idx"
	TAG_MEMTABLE_ID        = "memtable_id"
)

// Application-level metric keys
const (
	ApiRequestCount           = "api_request_count"
	ApiRequestLatency         = "api_request_latency"
	ExternalApiRequestCount   = "external_api_request_count"
	ExternalApiRequestLatency = "external_api_request_latency"
	DBCallLatency             = "db_call_latency"
	DBCallCount               = "db_call_count"
	MethodLatency             = "method_latency"
	MethodCount               = "method_count"
)

var (
	statsDClient    = getDefaultClient()
	samplingRate    = 0.1
	telegrafAddress = "localhost:8125"
	appName         = ""
	initialized     = false
	once            sync.Once

	// When false, all Timing/Count/Incr/Gauge calls are no-ops (zero allocations).
	// Controlled by FLASHRING_METRICS_ENABLED env var ("true"/"1" to enable).
	// Defaults to true for backward compatibility.
	metricsEnabled = loadMetricsEnabled()
)

func loadMetricsEnabled() bool {
	v := os.Getenv("FLASHRING_METRICS_ENABLED")
	if v == "" {
		return false
	}
	return strings.EqualFold(v, "true") || v == "1"
}

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
			"sampling rate - %f, flashring metrics enabled - %v", telegrafAddress, globalTags, samplingRate, metricsEnabled)
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

// Timing sends timing information. No-op when metrics are disabled.
func Timing(name string, value time.Duration, tags []string) {
	if !metricsEnabled {
		return
	}
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Timing(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd timing", err)
	}
}

// Count increases metric counter by value. No-op when metrics are disabled.
func Count(name string, value int64, tags []string) {
	if !metricsEnabled {
		return
	}
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Count(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd count", err)
	}
}

// Incr increases metric counter by 1. No-op when metrics are disabled.
func Incr(name string, tags []string) {
	if !metricsEnabled {
		return
	}
	Count(name, 1, tags)
}

// Gauge sets a gauge value. No-op when metrics are disabled.
func Gauge(name string, value float64, tags []string) {
	if !metricsEnabled {
		return
	}
	tags = append(tags, TagAsString(TagService, appName))
	err := statsDClient.Gauge(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd gauge", err)
	}
}

// Enabled returns whether flashring metrics are enabled.
// Call sites should check this before allocating tags to avoid heap allocations.
func Enabled() bool {
	return metricsEnabled
}

func GetShardTag(shardIdx uint32) []string {
	return BuildTag(NewTag(TAG_SHARD_IDX, strconv.Itoa(int(shardIdx))))
}

func GetMemtableTag(memtableId uint32) []string {
	return BuildTag(NewTag(TAG_MEMTABLE_ID, strconv.Itoa(int(memtableId))))
}
