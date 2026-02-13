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

			rThroughput := int(float64(getsTotal-prevGetsTotal) / float64(30))
			wThroughput := int(float64(putsTotal-prevPutsTotal) / float64(30))
			hitRate := float64(hitsTotal-prevHitsTotal) / float64(getsTotal-prevGetsTotal)
			activeEntries := float64(activeEntriesTotal-prevActiveEntriesTotal) / float64(30)
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

			keyNotFoundTotal := int64(0)
			keyExpiredTotal := int64(0)
			badDataTotal := int64(0)
			badLengthTotal := int64(0)
			badCR32Total := int64(0)
			badKeyTotal := int64(0)
			deletedKeyTotal := int64(0)
			writeTotal := int64(0)
			punchHoleTotal := int64(0)

			for _, shard := range currentMetrics.ShardIndexMetrics {
				keyNotFoundTotal += shard.KeyNotFoundCount
				keyExpiredTotal += shard.KeyExpiredCount
				badDataTotal += shard.BadDataCount
				badLengthTotal += shard.BadLengthCount
				badCR32Total += shard.BadCR32Count
				badKeyTotal += shard.BadKeyCount
				deletedKeyTotal += shard.DeletedKeyCount
				writeTotal += shard.WriteCount
				punchHoleTotal += shard.PunchHoleCount
			}

			log.Info().Msgf("KeyNotFoundTotal: %v", keyNotFoundTotal)
			log.Info().Msgf("KeyExpiredTotal: %v", keyExpiredTotal)
			log.Info().Msgf("BadDataTotal: %v", badDataTotal)
			log.Info().Msgf("BadLengthTotal: %v", badLengthTotal)
			log.Info().Msgf("BadCR32Total: %v", badCR32Total)
			log.Info().Msgf("BadKeyTotal: %v", badKeyTotal)
			log.Info().Msgf("DeletedKeyTotal: %v", deletedKeyTotal)
			log.Info().Msgf("WriteTotal: %v", writeTotal)
			log.Info().Msgf("PunchHoleTotal: %v", punchHoleTotal)

			// Debug: Log cumulative totals to understand the issue
			log.Info().Msgf("DEBUG - GetsTotal: %v, HitsTotal: %v, PutsTotal: %v, ActiveEntriesTotal: %v", getsTotal, hitsTotal, putsTotal, activeEntriesTotal)

			// Debug: Log per-shard ActiveEntries to check distribution (first 5 shards)
			if len(currentMetrics.ShardMetrics) >= 5 {
				log.Info().Msgf("DEBUG PER-SHARD ActiveEntries - shard0: %d, shard1: %d, shard2: %d, shard3: %d, shard4: %d",
					currentMetrics.ShardMetrics[0].ActiveEntries,
					currentMetrics.ShardMetrics[1].ActiveEntries,
					currentMetrics.ShardMetrics[2].ActiveEntries,
					currentMetrics.ShardMetrics[3].ActiveEntries,
					currentMetrics.ShardMetrics[4].ActiveEntries)
			}

			// Update prev values for next iteration
			prevGetsTotal = getsTotal
			prevPutsTotal = putsTotal
			prevHitsTotal = hitsTotal
			prevExpiredTotal = expiredTotal
			prevReWritesTotal = reWritesTotal
			prevActiveEntriesTotal = activeEntriesTotal
		}
	}
}
