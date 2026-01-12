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

	// Per-shard observation metrics
	RecordRP99(shardIdx int, value time.Duration)
	RecordRP50(shardIdx int, value time.Duration)
	RecordRP25(shardIdx int, value time.Duration)
	RecordWP99(shardIdx int, value time.Duration)
	RecordWP50(shardIdx int, value time.Duration)
	RecordWP25(shardIdx int, value time.Duration)
	RecordRThroughput(shardIdx int, value float64)
	RecordWThroughput(shardIdx int, value float64)
	RecordHitRate(shardIdx int, value float64)
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
	RP99          time.Duration
	RP50          time.Duration
	RP25          time.Duration
	WP99          time.Duration
	WP50          time.Duration
	WP25          time.Duration
	RThroughput   float64
	WThroughput   float64
	HitRate       float64
	ActiveEntries float64
}

// Define your parameter structure
type RunMetrics struct {
	// Per-shard observation parameters
	ShardMetrics []ShardMetrics

	// Averaged metrics over all shards
	AveragedMetrics ShardMetrics
}

// ShardMetricValue represents a metric value for a specific shard
type ShardMetricValue struct {
	ShardIdx int
	Value    float64
}

// ShardDurationValue represents a duration metric value for a specific shard
type ShardDurationValue struct {
	ShardIdx int
	Value    time.Duration
}

// MetricChannels holds separate channels for each metric type (per-shard)
type MetricChannels struct {
	RP99          chan ShardDurationValue
	RP50          chan ShardDurationValue
	RP25          chan ShardDurationValue
	WP99          chan ShardDurationValue
	WP50          chan ShardDurationValue
	WP25          chan ShardDurationValue
	RThroughput   chan ShardMetricValue
	WThroughput   chan ShardMetricValue
	HitRate       chan ShardMetricValue
	ActiveEntries chan ShardMetricValue
}

// MetricsCollector collects and averages all metrics (per-shard)
type MetricsCollector struct {
	Config          MetricsCollectorConfig
	channels        MetricChannels                     //channels for each metric type (per-shard)
	averagedMetrics map[string]*MetricAverager         // metricName -> averager
	instantMetrics  map[int]map[string]*MetricAverager // shardIdx -> metricName -> averager
	stopCh          chan struct{}                      //channel to stop the collector when running from console
	wg              sync.WaitGroup
	mu              sync.RWMutex
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
		go RunCSVLoggerWaitForShutdown()
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
			RP99:          make(chan ShardDurationValue, bufferSize),
			RP50:          make(chan ShardDurationValue, bufferSize),
			RP25:          make(chan ShardDurationValue, bufferSize),
			WP99:          make(chan ShardDurationValue, bufferSize),
			WP50:          make(chan ShardDurationValue, bufferSize),
			WP25:          make(chan ShardDurationValue, bufferSize),
			RThroughput:   make(chan ShardMetricValue, bufferSize),
			WThroughput:   make(chan ShardMetricValue, bufferSize),
			HitRate:       make(chan ShardMetricValue, bufferSize),
			ActiveEntries: make(chan ShardMetricValue, bufferSize),
		},
		averagedMetrics: make(map[string]*MetricAverager),
		instantMetrics:  make(map[int]map[string]*MetricAverager),
		stopCh:          make(chan struct{}),
	}

	// Initialize averagedMetrics with MetricAverager instances
	metricNames := []string{"RP99", "RP50", "RP25", "WP99", "WP50", "WP25", "RThroughput", "WThroughput", "HitRate", "ActiveEntries"}
	for _, name := range metricNames {
		mc.averagedMetrics[name] = &MetricAverager{}
	}

	// Initialize instantMetrics for each shard with MetricAverager instances
	shards := config.Metadata["shards"].(int)
	for shardIdx := 0; shardIdx < shards; shardIdx++ {
		mc.instantMetrics[shardIdx] = make(map[string]*MetricAverager)
		for _, name := range metricNames {
			mc.instantMetrics[shardIdx][name] = &MetricAverager{}
		}
	}

	return mc
}

// Start begins collecting metrics from all channels
func (mc *MetricsCollector) Start() {
	// Start a goroutine for each metric channel
	mc.wg.Add(10)

	go mc.collectShardDuration(mc.channels.RP99, "RP99")
	go mc.collectShardDuration(mc.channels.RP50, "RP50")
	go mc.collectShardDuration(mc.channels.RP25, "RP25")
	go mc.collectShardDuration(mc.channels.WP99, "WP99")
	go mc.collectShardDuration(mc.channels.WP50, "WP50")
	go mc.collectShardDuration(mc.channels.WP25, "WP25")
	go mc.collectShardMetric(mc.channels.RThroughput, "RThroughput")
	go mc.collectShardMetric(mc.channels.WThroughput, "WThroughput")
	go mc.collectShardMetric(mc.channels.HitRate, "HitRate")
	go mc.collectShardMetric(mc.channels.ActiveEntries, "ActiveEntries")
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
			instants := mc.instantMetrics[sv.ShardIdx]
			instants[name].Add(sv.Value)
			mc.averagedMetrics[name].Add(sv.Value)

		}
	}
}

func (mc *MetricsCollector) collectShardDuration(ch chan ShardDurationValue, name string) {
	defer mc.wg.Done()
	for {
		select {
		case <-mc.stopCh:
			return
		case sv, ok := <-ch:
			if !ok {
				return
			}
			instants := mc.instantMetrics[sv.ShardIdx]
			instants[name].AddDuration(sv.Value)
			mc.averagedMetrics[name].AddDuration(sv.Value)
		}
	}
}

// RecordRP99 sends a value to the RP99 channel for a specific shard
func (mc *MetricsCollector) RecordRP99(shardIdx int, value time.Duration) {
	select {
	case mc.channels.RP99 <- ShardDurationValue{ShardIdx: shardIdx, Value: value}:
	default: // Don't block if channel is full
	}
}

// RecordRP50 sends a value to the RP50 channel for a specific shard
func (mc *MetricsCollector) RecordRP50(shardIdx int, value time.Duration) {
	select {
	case mc.channels.RP50 <- ShardDurationValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordRP25 sends a value to the RP25 channel for a specific shard
func (mc *MetricsCollector) RecordRP25(shardIdx int, value time.Duration) {
	select {
	case mc.channels.RP25 <- ShardDurationValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordWP99 sends a value to the WP99 channel for a specific shard
func (mc *MetricsCollector) RecordWP99(shardIdx int, value time.Duration) {
	select {
	case mc.channels.WP99 <- ShardDurationValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordWP50 sends a value to the WP50 channel for a specific shard
func (mc *MetricsCollector) RecordWP50(shardIdx int, value time.Duration) {
	select {
	case mc.channels.WP50 <- ShardDurationValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordWP25 sends a value to the WP25 channel for a specific shard
func (mc *MetricsCollector) RecordWP25(shardIdx int, value time.Duration) {
	select {
	case mc.channels.WP25 <- ShardDurationValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordRThroughput sends a value to the RThroughput channel for a specific shard
func (mc *MetricsCollector) RecordRThroughput(shardIdx int, value float64) {
	select {
	case mc.channels.RThroughput <- ShardMetricValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordWThroughput sends a value to the WThroughput channel for a specific shard
func (mc *MetricsCollector) RecordWThroughput(shardIdx int, value float64) {
	select {
	case mc.channels.WThroughput <- ShardMetricValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordHitRate sends a value to the HitRate channel for a specific shard
func (mc *MetricsCollector) RecordHitRate(shardIdx int, value float64) {
	select {
	case mc.channels.HitRate <- ShardMetricValue{ShardIdx: shardIdx, Value: value}:
	default:
	}
}

// RecordActiveEntries sends a value to the ActiveEntries channel for a specific shard
func (mc *MetricsCollector) RecordActiveEntries(shardIdx int, value float64) {
	select {
	case mc.channels.ActiveEntries <- ShardMetricValue{ShardIdx: shardIdx, Value: value}:
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
				RP99:          time.Duration(instants["RP99"].Latest()),
				RP50:          time.Duration(instants["RP50"].Latest()),
				RP25:          time.Duration(instants["RP25"].Latest()),
				WP99:          time.Duration(instants["WP99"].Latest()),
				WP50:          time.Duration(instants["WP50"].Latest()),
				WP25:          time.Duration(instants["WP25"].Latest()),
				RThroughput:   instants["RThroughput"].Latest(),
				WThroughput:   instants["WThroughput"].Latest(),
				HitRate:       instants["HitRate"].Latest(),
				ActiveEntries: instants["ActiveEntries"].Latest(),
			}
		}
	}

	averagedMetrics := ShardMetrics{
		RP99:          time.Duration(mc.averagedMetrics["RP99"].Average()),
		RP50:          time.Duration(mc.averagedMetrics["RP50"].Average()),
		RP25:          time.Duration(mc.averagedMetrics["RP25"].Average()),
		WP99:          time.Duration(mc.averagedMetrics["WP99"].Average()),
		WP50:          time.Duration(mc.averagedMetrics["WP50"].Average()),
		WP25:          time.Duration(mc.averagedMetrics["WP25"].Average()),
		RThroughput:   mc.averagedMetrics["RThroughput"].Average(),
		WThroughput:   mc.averagedMetrics["WThroughput"].Average(),
		HitRate:       mc.averagedMetrics["HitRate"].Average(),
		ActiveEntries: mc.averagedMetrics["ActiveEntries"].Average(),
	}

	return RunMetrics{
		ShardMetrics:    shardMetrics,
		AveragedMetrics: averagedMetrics,
	}
}

// ResetAverages resets all averagers to start fresh
func (mc *MetricsCollector) ResetAverages() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, shardInstant := range mc.instantMetrics {
		for _, instantMetric := range shardInstant {
			instantMetric.Reset() // Reset the instant metric
		}
	}
	for _, averagedMetric := range mc.averagedMetrics {
		averagedMetric.Reset() // Reset the averaged metric
	}
}

// Stop stops all collector goroutines
func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
	mc.wg.Wait()
}
