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
	KEY_EXPIRED_ENTRIES = "flashring_expired_entries"
	KEY_REWRITES        = "flashring_rewrites"
	KEY_GETS            = "flashring_gets"
	KEY_PUTS            = "flashring_puts"
	KEY_HITS            = "flashring_hits"

	KEY_KEY_NOT_FOUND_COUNT = "flashring_key_not_found_count"
	KEY_KEY_EXPIRED_COUNT   = "flashring_key_expired_count"
	KEY_BAD_LENGTH_COUNT    = "flashring_bad_length_count"
	KEY_BAD_CR32_COUNT      = "flashring_bad_cr32_count"
	KEY_BAD_KEY_COUNT       = "flashring_bad_key_count"
	KEY_DELETED_KEY_COUNT   = "flashring_deleted_key_count"

	KEY_PUNCH_HOLE_COUNT = "flashring_punch_hole_count"
	KEY_PREAD_COUNT      = "flashring_pread_count"

	KEY_TRIM_HEAD_LATENCY = "flashring_wrap_file_trim_head_latency"
	KEY_PREAD_LATENCY     = "flashring_pread_latency"
	KEY_PWRITE_LATENCY    = "flashring_pwrite_latency"

	KEY_MEMTABLE_FLUSH_COUNT       = "flashring_memtable_flush_count"
	KEY_MEMTABLE_FLUSH_ERROR_COUNT = "flashring_memtable_flush_error_count"
	KEY_VALUE_TOO_LARGE_COUNT      = "flashring_value_too_large_count"

	LATENCY_WLOCK = "flashring_wlock_latency"

	KEY_RINGBUFFER_ACTIVE_ENTRIES = "flashring_ringbuffer_active_entries"
	KEY_MEMTABLE_HIT              = "flashring_memtable_hit"
	KEY_MEMTABLE_MISS             = "flashring_memtable_miss"
	KEY_DATA_LENGTH               = "flashring_data_length"
	KEY_IOURING_SIZE              = "flashring_iouring_size"
	KEY_REWRITE_SCORE             = "flashring_rewrite_score"
	KEY_REWRITE_DECISION          = "flashring_rewrite_decision"
	KEY_ACCESS_FREQ               = "flashring_access_freq"
	KEY_LAST_ACCESS               = "flashring_last_access"
)

// Rewrite predictor tag keys
const (
	TAG_SCORE_BUCKET = "score_bucket"
	TAG_DECISION     = "decision"
	TAG_RING_ZONE    = "ring_zone"
	TAG_FREQ_BAND    = "freq_band"
	TAG_RECENCY_BAND = "recency_band"
)

// Flashring tag keys
const (
	TAG_LATENCY_PERCENTILE = "latency_percentile"
	TAG_VALUE_P25          = "p25"
	TAG_VALUE_P50          = "p50"
	TAG_VALUE_P99          = "p99"
	TAG_SHARD_IDX          = "shard_idx"
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
	initialized     = false
	once            sync.Once

	// When false, all Timing/Count/Incr/Gauge calls are no-ops (zero allocations).
	// Controlled by FLASHRING_METRICS_ENABLED env var ("true"/"1" to enable).
	metricsEnabled = loadMetricsEnabled()

	// Per-group cardinality toggle for the rewrite-predictor telemetry
	// group (KEY_REWRITE_SCORE, KEY_REWRITE_DECISION, KEY_ACCESS_FREQ,
	// KEY_LAST_ACCESS). Defaults to true so enabling the master switch
	// preserves today's behaviour; consumers on tight cardinality
	// budgets opt out via FLASHRING_METRICS_PREDICTOR_ENABLED=false.
	// Read once at init — a restart is required to flip it, matching
	// the master switch's shape.
	predictorMetricsEnabled = loadPredictorMetricsEnabled()

	shardTags []string
)

func loadMetricsEnabled() bool {
	v := os.Getenv("FLASHRING_METRICS_ENABLED")
	if v == "" {
		return false
	}
	return strings.EqualFold(v, "true") || v == "1"
}

func loadPredictorMetricsEnabled() bool {
	v := os.Getenv("FLASHRING_METRICS_PREDICTOR_ENABLED")
	if v == "" {
		return true
	}
	return !(strings.EqualFold(v, "false") || v == "0")
}

// PredictorMetricsEnabled reports whether the rewrite-predictor metric
// group is enabled. The master switch (metricsEnabled) still gates
// every emission, so this returning true only means "emit if the
// master switch is on too".
func PredictorMetricsEnabled() bool {
	return predictorMetricsEnabled
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
	err := statsDClient.Gauge(name, value, tags, samplingRate)
	if err != nil {
		log.Warn().AnErr("Error occurred while doing statsd gauge", err)
	}
}

// Enabled returns whether flashring metrics are enabled.
func Enabled() bool {
	return metricsEnabled
}

// GetShardTag returns the precomputed tag slice for shardIdx. Returns
// a one-element sub-slice of the shared shardTags array — do not
// append to or mutate the result. When shardIdx is out of range (which
// includes the "BuildShardTags was never called" case that internal
// unit tests hit — they exercise emitters like MemtableManager.Flush
// without spinning up a WrapCache), returns nil so callers that
// unconditionally pass GetShardTag(...) into a metric call don't panic.
// The subsequent Incr/Gauge/etc. no-ops via metricsEnabled anyway.
func GetShardTag(shardIdx uint32) []string {
	if int(shardIdx) >= len(shardTags) {
		return nil
	}
	// Capacity is capped at one element so an append by the caller copies
	// instead of overwriting the next shard's tag in the shared array.
	return shardTags[shardIdx : shardIdx+1 : shardIdx+1]
}

// BuildShardTags precomputes the per-shard tag strings and stores them
// in package state for GetShardTag to slice into. Must be called once
// at startup before any shard emits metrics; calling again replaces
// the table without synchronization, so do not call concurrently with
// GetShardTag.
func BuildShardTags(shardCount int) {
	tags := make([]string, 0, shardCount)
	for i := 0; i < shardCount; i++ {
		tags = append(tags, BuildTag(NewTag(TAG_SHARD_IDX, strconv.Itoa(int(i))))...)
	}
	shardTags = tags
}
