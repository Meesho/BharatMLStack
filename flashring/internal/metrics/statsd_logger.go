package metrics

import (
	"strconv"
	"time"
)

const (
	KEY_READ_LATENCY  = "flashringread_latency"
	KEY_WRITE_LATENCY = "flashringwrite_latency"
	KEY_RTHROUGHPUT   = "flashring_rthroughput"
	KEY_WTHROUGHPUT   = "flashring_wthroughput"
	KEY_HITRATE       = "flashring_hitrate"

	TAG_LATENCY_PERCENTILE = "latency_percentile"
	TAG_VALUE_P25          = "p25"
	TAG_VALUE_P50          = "p50"
	TAG_VALUE_P99          = "p99"

	TAG_SHARD_IDX = "shard_idx"
)

func RunStatsdLogger(metricsCollector *MetricsCollector) {

	// start a ticker to log the metrics every 30 seconds

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-metricsCollector.stopCh:
			return
		case <-ticker.C:
			currentMetrics = metricsCollector.GetMetrics()

			for idx, shard := range currentMetrics.ShardMetrics {

				shardIdx := strconv.Itoa(idx)

				Timing(KEY_READ_LATENCY, shard.RP99, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P99), NewTag(TAG_SHARD_IDX, shardIdx)))
				Timing(KEY_READ_LATENCY, shard.RP50, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P50), NewTag(TAG_SHARD_IDX, shardIdx)))
				Timing(KEY_READ_LATENCY, shard.RP25, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P25), NewTag(TAG_SHARD_IDX, shardIdx)))
				Timing(KEY_WRITE_LATENCY, shard.WP99, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P99), NewTag(TAG_SHARD_IDX, shardIdx)))
				Timing(KEY_WRITE_LATENCY, shard.WP50, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P50), NewTag(TAG_SHARD_IDX, shardIdx)))
				Timing(KEY_WRITE_LATENCY, shard.WP25, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P25), NewTag(TAG_SHARD_IDX, shardIdx)))
				Gauge(KEY_RTHROUGHPUT, shard.RThroughput, BuildTag(NewTag(TAG_SHARD_IDX, shardIdx)))
				Gauge(KEY_WTHROUGHPUT, shard.WThroughput, BuildTag(NewTag(TAG_SHARD_IDX, shardIdx)))
				Gauge(KEY_HITRATE, shard.HitRate, BuildTag(NewTag(TAG_SHARD_IDX, shardIdx)))
			}

		}

	}
}
