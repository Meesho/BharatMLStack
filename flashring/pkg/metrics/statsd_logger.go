package metrics

import (
	"strconv"
	"time"
)

const (
	KEY_READ_LATENCY         = "flashringread_latency"
	KEY_READ_LATENCY_STATSD  = "flashringread_latency_statsd"
	KEY_WRITE_LATENCY        = "flashringwrite_latency"
	KEY_WRITE_LATENCY_STATSD = "flashringwrite_latency_statsd"
	KEY_RTHROUGHPUT          = "flashring_rthroughput"
	KEY_WTHROUGHPUT          = "flashring_wthroughput"
	KEY_HITRATE              = "flashring_hitrate"
	KEY_ACTIVE_ENTRIES       = "flashring_active_entries"
	KEY_EXPIRED_ENTRIES      = "flashring_expired_entries"
	KEY_REWRITES             = "flashring_rewrites"
	KEY_GETS                 = "flashring_gets"
	KEY_PUTS                 = "flashring_puts"
	KEY_HITS                 = "flashring_hits"

	KEY_KEY_NOT_FOUND_COUNT = "flashring_key_not_found_count"
	KEY_KEY_EXPIRED_COUNT   = "flashring_key_expired_count"
	KEY_BAD_DATA_COUNT      = "flashring_bad_data_count"
	KEY_BAD_LENGTH_COUNT    = "flashring_bad_length_count"
	KEY_BAD_CR32_COUNT      = "flashring_bad_cr32_count"
	KEY_BAD_KEY_COUNT       = "flashring_bad_key_count"
	KEY_DELETED_KEY_COUNT   = "flashring_deleted_key_count"

	TAG_LATENCY_PERCENTILE = "latency_percentile"
	TAG_VALUE_P25          = "p25"
	TAG_VALUE_P50          = "p50"
	TAG_VALUE_P99          = "p99"
	TAG_SHARD_IDX          = "shard_idx"

	KEY_WRITE_COUNT      = "flashring_write_count"
	KEY_PUNCH_HOLE_COUNT = "flashring_punch_hole_count"
)

func RunStatsdLogger(metricsCollector *MetricsCollector) {

	// start a ticker to log the metrics every 30 seconds

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	//prev values per shard
	prevActiveEntries := make(map[int]int64)
	prevExpiredEntries := make(map[int]int64)
	prevRewrites := make(map[int]int64)
	prevGets := make(map[int]int64)
	prevPuts := make(map[int]int64)
	prevHits := make(map[int]int64)

	prevKeyNotFoundCount := make(map[int]int64)
	prevKeyExpiredCount := make(map[int]int64)
	prevBadDataCount := make(map[int]int64)
	prevBadLengthCount := make(map[int]int64)
	prevBadCR32Count := make(map[int]int64)
	prevBadKeyCount := make(map[int]int64)
	prevDeletedKeyCount := make(map[int]int64)

	prevWriteCount := make(map[int]int64)
	prevPunchHoleCount := make(map[int]int64)

	for {
		select {
		case <-metricsCollector.stopCh:
			return
		case <-ticker.C:
			currentMetrics = metricsCollector.GetMetrics()

			for idx, shard := range currentMetrics.ShardMetrics {

				shardIdx := strconv.Itoa(idx)
				shardBuildTag := NewTag(TAG_SHARD_IDX, shardIdx)

				Count(KEY_ACTIVE_ENTRIES, shard.ActiveEntries-prevActiveEntries[idx], BuildTag(shardBuildTag))
				Count(KEY_EXPIRED_ENTRIES, shard.ExpiredEntries-prevExpiredEntries[idx], BuildTag(shardBuildTag))
				Count(KEY_REWRITES, shard.Rewrites-prevRewrites[idx], BuildTag(shardBuildTag))
				Count(KEY_GETS, shard.Gets-prevGets[idx], BuildTag(shardBuildTag))
				Count(KEY_PUTS, shard.Puts-prevPuts[idx], BuildTag(shardBuildTag))
				Count(KEY_HITS, shard.Hits-prevHits[idx], BuildTag(shardBuildTag))

				Timing(KEY_READ_LATENCY, shard.RP99, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P99), shardBuildTag))
				Timing(KEY_READ_LATENCY, shard.RP50, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P50), shardBuildTag))
				Timing(KEY_READ_LATENCY, shard.RP25, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P25), shardBuildTag))
				Timing(KEY_WRITE_LATENCY, shard.WP99, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P99), shardBuildTag))
				Timing(KEY_WRITE_LATENCY, shard.WP50, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P50), shardBuildTag))
				Timing(KEY_WRITE_LATENCY, shard.WP25, BuildTag(NewTag(TAG_LATENCY_PERCENTILE, TAG_VALUE_P25), shardBuildTag))

				prevActiveEntries[idx] = shard.ActiveEntries
				prevExpiredEntries[idx] = shard.ExpiredEntries
				prevRewrites[idx] = shard.Rewrites
				prevGets[idx] = shard.Gets
				prevPuts[idx] = shard.Puts
				prevHits[idx] = shard.Hits

			}

			for idx, shard := range currentMetrics.ShardIndexMetrics {
				shardIdx := strconv.Itoa(idx)
				shardBuildTag := NewTag(TAG_SHARD_IDX, shardIdx)

				Count(KEY_KEY_NOT_FOUND_COUNT, shard.KeyNotFoundCount-prevKeyNotFoundCount[idx], BuildTag(shardBuildTag))
				Count(KEY_KEY_EXPIRED_COUNT, shard.KeyExpiredCount-prevKeyExpiredCount[idx], BuildTag(shardBuildTag))
				Count(KEY_BAD_DATA_COUNT, shard.BadDataCount-prevBadDataCount[idx], BuildTag(shardBuildTag))
				Count(KEY_BAD_LENGTH_COUNT, shard.BadLengthCount-prevBadLengthCount[idx], BuildTag(shardBuildTag))
				Count(KEY_BAD_CR32_COUNT, shard.BadCR32Count-prevBadCR32Count[idx], BuildTag(shardBuildTag))
				Count(KEY_BAD_KEY_COUNT, shard.BadKeyCount-prevBadKeyCount[idx], BuildTag(shardBuildTag))
				Count(KEY_DELETED_KEY_COUNT, shard.DeletedKeyCount-prevDeletedKeyCount[idx], BuildTag(shardBuildTag))
				Count(KEY_WRITE_COUNT, shard.WriteCount-prevWriteCount[idx], BuildTag(shardBuildTag))
				Count(KEY_PUNCH_HOLE_COUNT, shard.PunchHoleCount-prevPunchHoleCount[idx], BuildTag(shardBuildTag))

				prevKeyNotFoundCount[idx] = shard.KeyNotFoundCount
				prevKeyExpiredCount[idx] = shard.KeyExpiredCount
				prevBadDataCount[idx] = shard.BadDataCount
				prevBadLengthCount[idx] = shard.BadLengthCount
				prevBadCR32Count[idx] = shard.BadCR32Count
				prevBadKeyCount[idx] = shard.BadKeyCount
				prevDeletedKeyCount[idx] = shard.DeletedKeyCount
				prevWriteCount[idx] = shard.WriteCount
				prevPunchHoleCount[idx] = shard.PunchHoleCount
			}

		}
	}
}
