package metrics

import (
	"sync"
	"time"
)

// Global variable to hold runtime data
var currentMetrics RunMetrics
var metricsCollector *MetricsCollector

// MetricsRecorder is an interface for recording metrics from the cache
// Implement this interface to receive per-shard metrics from the cache layer
type MetricsRecorder interface {
	RecordGets(shardIdx int, value int64)
	RecordPuts(shardIdx int, value int64)
	RecordHits(shardIdx int, value int64)
	RecordActiveEntries(shardIdx int, value int64)
	RecordExpiredEntries(shardIdx int, value int64)
	RecordRewrites(shardIdx int, value int64)

	// Per-shard observation metrics
	RecordRP99(shardIdx int, value time.Duration)
	RecordRP50(shardIdx int, value time.Duration)
	RecordRP25(shardIdx int, value time.Duration)
	RecordWP99(shardIdx int, value time.Duration)
	RecordWP50(shardIdx int, value time.Duration)
	RecordWP25(shardIdx int, value time.Duration)
}

type MetricsCollectorConfig struct {
	StatsEnabled bool //Stats enabled - global flag

	CsvLogging     bool //Log to CSV enabled
	ConsoleLogging bool //Log to console enabled
	StatsdLogging  bool //Log to Statsd enabled

	InstantMetrics  bool //Metrics at every instant
	AveragedMetrics bool //Metrics averaged over a period of time

	// Metadata for external systems to use
	// must include shards, keys_per_shard, read_workers, write_workers, plan
	Metadata map[string]any
}

// ShardMetrics holds observation metrics for a single shard
type ShardMetrics struct {
	Gets           int64
	Puts           int64
	Hits           int64
	ActiveEntries  int64
	ExpiredEntries int64
	Rewrites       int64
	RP99           time.Duration
	RP50           time.Duration
	RP25           time.Duration
	WP99           time.Duration
	WP50           time.Duration
	WP25           time.Duration
}

// Define your parameter structure
type RunMetrics struct {
	// Per-shard observation parameters
	ShardMetrics []ShardMetrics
}

// ShardMetricValue represents a metric value for a specific shard
type ShardMetricValue struct {
	ShardIdx int
	value    int64
}

// MetricChannels holds separate channels for each metric type (per-shard)
type MetricChannels struct {
	Gets           chan ShardMetricValue
	Puts           chan ShardMetricValue
	Hits           chan ShardMetricValue
	ActiveEntries  chan ShardMetricValue
	ExpiredEntries chan ShardMetricValue
	Rewrites       chan ShardMetricValue
	RP99           chan ShardMetricValue
	RP50           chan ShardMetricValue
	RP25           chan ShardMetricValue
	WP99           chan ShardMetricValue
	WP50           chan ShardMetricValue
	WP25           chan ShardMetricValue
}

// MetricsCollector collects and averages all metrics (per-shard)
type MetricsCollector struct {
	Config         MetricsCollectorConfig
	channels       MetricChannels           //channels for each metric type (per-shard)
	instantMetrics map[int]map[string]int64 // shardIdx -> metricName -> value
	stopCh         chan struct{}            //channel to stop the collector when running from console
	wg             sync.WaitGroup
	mu             sync.RWMutex
}

// InitMetricsCollector creates and starts the metrics collector, returning it
// so it can be passed to other components (e.g., cache config)
func InitMetricsCollector(config MetricsCollectorConfig) *MetricsCollector {
	Init()
	metricsCollector = NewMetricsCollector(config, 100)

	shouldLog := config.StatsEnabled && (config.CsvLogging || config.ConsoleLogging || config.StatsdLogging)

	if shouldLog {
		metricsCollector.Start()
	}

	if config.CsvLogging {
		csvLogger := CsvLogger{metricsCollector: metricsCollector}
		go csvLogger.RunCSVLoggerWaitForShutdown()
	}

	if config.StatsdLogging {
		go RunStatsdLogger(metricsCollector)
	}

	if config.ConsoleLogging {
		go RunConsoleLogger(metricsCollector)
	}

	return metricsCollector
}

// NewMetricsCollector creates a new metrics collector with channels
func NewMetricsCollector(config MetricsCollectorConfig, bufferSize int) *MetricsCollector {
	mc := &MetricsCollector{
		Config: config,
		channels: MetricChannels{
			Gets:           make(chan ShardMetricValue, bufferSize),
			Puts:           make(chan ShardMetricValue, bufferSize),
			Hits:           make(chan ShardMetricValue, bufferSize),
			ActiveEntries:  make(chan ShardMetricValue, bufferSize),
			ExpiredEntries: make(chan ShardMetricValue, bufferSize),
			Rewrites:       make(chan ShardMetricValue, bufferSize),
			RP99:           make(chan ShardMetricValue, bufferSize),
			RP50:           make(chan ShardMetricValue, bufferSize),
			RP25:           make(chan ShardMetricValue, bufferSize),
			WP99:           make(chan ShardMetricValue, bufferSize),
			WP50:           make(chan ShardMetricValue, bufferSize),
			WP25:           make(chan ShardMetricValue, bufferSize),
		},

		instantMetrics: make(map[int]map[string]int64),
		stopCh:         make(chan struct{}),
	}

	// Initialize averagedMetrics with MetricAverager instances
	metricNames := []string{"RP99", "RP50", "RP25", "WP99", "WP50", "WP25", "Gets", "Puts", "Hits", "ActiveEntries", "ExpiredEntries", "Rewrites"}

	// Initialize instantMetrics for each shard with MetricAverager instances
	shards := config.Metadata["shards"].(int)
	for shardIdx := 0; shardIdx < shards; shardIdx++ {
		mc.instantMetrics[shardIdx] = make(map[string]int64)
		for _, name := range metricNames {
			mc.instantMetrics[shardIdx][name] = 0
		}
	}

	return mc
}

// Start begins collecting metrics from all channels
func (mc *MetricsCollector) Start() {
	// Start a goroutine for each metric channel
	mc.wg.Add(12)

	go mc.collectShardMetric(mc.channels.RP99, "RP99")
	go mc.collectShardMetric(mc.channels.RP50, "RP50")
	go mc.collectShardMetric(mc.channels.RP25, "RP25")
	go mc.collectShardMetric(mc.channels.WP99, "WP99")
	go mc.collectShardMetric(mc.channels.WP50, "WP50")
	go mc.collectShardMetric(mc.channels.WP25, "WP25")

	go mc.collectShardMetric(mc.channels.ActiveEntries, "ActiveEntries")
	go mc.collectShardMetric(mc.channels.ExpiredEntries, "ExpiredEntries")
	go mc.collectShardMetric(mc.channels.Rewrites, "Rewrites")
	go mc.collectShardMetric(mc.channels.Gets, "Gets")
	go mc.collectShardMetric(mc.channels.Puts, "Puts")
	go mc.collectShardMetric(mc.channels.Hits, "Hits")
}

func (mc *MetricsCollector) collectShardMetric(ch chan ShardMetricValue, name string) {
	defer mc.wg.Done()
	for {
		select {
		case <-mc.stopCh:
			return
		case sv, ok := <-ch:
			if !ok {
				return
			}

			mc.instantMetrics[sv.ShardIdx][name] = sv.value

		}
	}
}

// RecordRP99 sends a value to the RP99 channel for a specific shard
func (mc *MetricsCollector) RecordRP99(shardIdx int, value time.Duration) {
	select {
	case mc.channels.RP99 <- ShardMetricValue{ShardIdx: shardIdx, value: int64(value)}:
	default: // Don't block if channel is full
	}
}

// RecordRP50 sends a value to the RP50 channel for a specific shard
func (mc *MetricsCollector) RecordRP50(shardIdx int, value time.Duration) {
	select {
	case mc.channels.RP50 <- ShardMetricValue{ShardIdx: shardIdx, value: int64(value)}:
	default:
	}
}

// RecordRP25 sends a value to the RP25 channel for a specific shard
func (mc *MetricsCollector) RecordRP25(shardIdx int, value time.Duration) {
	select {
	case mc.channels.RP25 <- ShardMetricValue{ShardIdx: shardIdx, value: int64(value)}:
	default:
	}
}

// RecordWP99 sends a value to the WP99 channel for a specific shard
func (mc *MetricsCollector) RecordWP99(shardIdx int, value time.Duration) {
	select {
	case mc.channels.WP99 <- ShardMetricValue{ShardIdx: shardIdx, value: int64(value)}:
	default:
	}
}

// RecordWP50 sends a value to the WP50 channel for a specific shard
func (mc *MetricsCollector) RecordWP50(shardIdx int, value time.Duration) {
	select {
	case mc.channels.WP50 <- ShardMetricValue{ShardIdx: shardIdx, value: int64(value)}:
	default:
	}
}

// RecordWP25 sends a value to the WP25 channel for a specific shard
func (mc *MetricsCollector) RecordWP25(shardIdx int, value time.Duration) {
	select {
	case mc.channels.WP25 <- ShardMetricValue{ShardIdx: shardIdx, value: int64(value)}:
	default:
	}
}

// RecordGets sends a value to the Gets channel for a specific shard
func (mc *MetricsCollector) RecordGets(shardIdx int, value int64) {
	select {
	case mc.channels.Gets <- ShardMetricValue{ShardIdx: shardIdx, value: value}:
	default:
	}
}

// RecordPuts sends a value to the Puts channel for a specific shard
func (mc *MetricsCollector) RecordPuts(shardIdx int, value int64) {
	select {
	case mc.channels.Puts <- ShardMetricValue{ShardIdx: shardIdx, value: value}:
	default:
	}
}

// RecordHits sends a value to the Hits channel for a specific shard
func (mc *MetricsCollector) RecordHits(shardIdx int, value int64) {
	select {
	case mc.channels.Hits <- ShardMetricValue{ShardIdx: shardIdx, value: value}:
	default:
	}
}

// RecordActiveEntries sends a value to the ActiveEntries channel for a specific shard
func (mc *MetricsCollector) RecordActiveEntries(shardIdx int, value int64) {
	select {
	case mc.channels.ActiveEntries <- ShardMetricValue{ShardIdx: shardIdx, value: value}:
	default:
	}
}

// RecordExpiredEntries sends a value to the ExpiredEntries channel for a specific shard
func (mc *MetricsCollector) RecordExpiredEntries(shardIdx int, value int64) {
	select {
	case mc.channels.ExpiredEntries <- ShardMetricValue{ShardIdx: shardIdx, value: value}:
	default:
	}
}

// RecordRewrites sends a value to the Rewrites channel for a specific shard
func (mc *MetricsCollector) RecordRewrites(shardIdx int, value int64) {
	select {
	case mc.channels.Rewrites <- ShardMetricValue{ShardIdx: shardIdx, value: value}:
	default:
	}
}

func (mc *MetricsCollector) GetMetrics() RunMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	shards := mc.Config.Metadata["shards"].(int)

	// Build per-shard metrics
	shardMetrics := make([]ShardMetrics, shards)
	for shardIdx := 0; shardIdx < shards; shardIdx++ {
		if instants, exists := mc.instantMetrics[shardIdx]; exists {
			shardMetrics[shardIdx] = ShardMetrics{
				RP99:           time.Duration(instants["RP99"]),
				RP50:           time.Duration(instants["RP50"]),
				RP25:           time.Duration(instants["RP25"]),
				WP99:           time.Duration(instants["WP99"]),
				WP50:           time.Duration(instants["WP50"]),
				WP25:           time.Duration(instants["WP25"]),
				Gets:           instants["Gets"],
				Puts:           instants["Puts"],
				Hits:           instants["Hits"],
				ActiveEntries:  instants["ActiveEntries"],
				ExpiredEntries: instants["ExpiredEntries"],
				Rewrites:       instants["Rewrites"],
			}
		}
	}

	return RunMetrics{
		ShardMetrics: shardMetrics,
	}
}

// Stop stops all collector goroutines
func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
	mc.wg.Wait()
}
