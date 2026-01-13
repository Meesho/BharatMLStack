package metrics

import (
	"time"

	"github.com/rs/zerolog/log"
)

func RunConsoleLogger(metricsCollector *MetricsCollector) {

	// start a ticker to log the metrics every 30 seconds

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	shards := metricsCollector.Config.Metadata["shards"].(int)

	prevGetsTotal := uint64(0)
	prevPutsTotal := uint64(0)
	prevHitsTotal := uint64(0)
	prevExpiredTotal := uint64(0)
	prevReWritesTotal := uint64(0)
	prevActiveEntriesTotal := uint64(0)

	for {
		select {
		case <-metricsCollector.stopCh:
			return
		case <-ticker.C:
			currentMetrics = metricsCollector.GetMetrics()

			getsTotal := uint64(0)
			putsTotal := uint64(0)
			hitsTotal := uint64(0)
			expiredTotal := uint64(0)
			reWritesTotal := uint64(0)
			activeEntriesTotal := uint64(0)

			rp99 := time.Duration(0)
			rp50 := time.Duration(0)
			rp25 := time.Duration(0)
			wp99 := time.Duration(0)
			wp50 := time.Duration(0)
			wp25 := time.Duration(0)

			for _, shard := range currentMetrics.ShardMetrics {
				getsTotal += uint64(shard.Gets)
				putsTotal += uint64(shard.Puts)
				hitsTotal += uint64(shard.Hits)
				expiredTotal += uint64(shard.ExpiredEntries)
				reWritesTotal += uint64(shard.Rewrites)
				activeEntriesTotal += uint64(shard.ActiveEntries)

				rp99 += shard.RP99
				rp50 += shard.RP50
				rp25 += shard.RP25
				wp99 += shard.WP99
				wp50 += shard.WP50
				wp25 += shard.WP25
			}

			rp99 = rp99 / time.Duration(shards)
			rp50 = rp50 / time.Duration(shards)
			rp25 = rp25 / time.Duration(shards)
			wp99 = wp99 / time.Duration(shards)
			wp50 = wp50 / time.Duration(shards)
			wp25 = wp25 / time.Duration(shards)

			rThroughput := float64(getsTotal-prevGetsTotal) / float64(30)
			wThroughput := float64(putsTotal-prevPutsTotal) / float64(30)
			hitRate := float64(hitsTotal-prevHitsTotal) / float64(getsTotal-prevGetsTotal)
			activeEntries := float64(activeEntriesTotal - prevActiveEntriesTotal)
			expiredEntries := float64(expiredTotal - prevExpiredTotal)
			reWrites := float64(reWritesTotal - prevReWritesTotal)

			log.Info().Msgf("RP99: %v", rp99)
			log.Info().Msgf("RP50: %v", rp50)
			log.Info().Msgf("RP25: %v", rp25)
			log.Info().Msgf("WP99: %v", wp99)
			log.Info().Msgf("WP50: %v", wp50)
			log.Info().Msgf("WP25: %v", wp25)
			log.Info().Msgf("RThroughput: %v/s", rThroughput)
			log.Info().Msgf("WThroughput: %v/s", wThroughput)
			log.Info().Msgf("HitRate: %v", hitRate)
			log.Info().Msgf("ActiveEntries: %v", activeEntries)
			log.Info().Msgf("ExpiredEntries: %v", expiredEntries)
			log.Info().Msgf("ReWrites: %v", reWrites)
		}
	}
}
