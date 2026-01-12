package metrics

import (
	"time"

	"github.com/rs/zerolog/log"
)

func RunConsoleLogger(metricsCollector *MetricsCollector) {

	// start a ticker to log the metrics every 30 seconds

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-metricsCollector.stopCh:
			return
		case <-ticker.C:
			currentMetrics = metricsCollector.GetMetrics()

			rp99 := currentMetrics.AveragedMetrics.RP99
			rp50 := currentMetrics.AveragedMetrics.RP50
			rp25 := currentMetrics.AveragedMetrics.RP25
			wp99 := currentMetrics.AveragedMetrics.WP99
			wp50 := currentMetrics.AveragedMetrics.WP50
			wp25 := currentMetrics.AveragedMetrics.WP25

			rThroughput := currentMetrics.AveragedMetrics.RThroughput
			hitRate := currentMetrics.AveragedMetrics.HitRate
			wThroughput := currentMetrics.AveragedMetrics.WThroughput
			activeEntries := currentMetrics.AveragedMetrics.ActiveEntries

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
		}
	}
}
