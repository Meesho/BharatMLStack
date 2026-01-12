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

			rp99 := currentMetrics.AveragedMetrics.RP99.Milliseconds()
			rp50 := currentMetrics.AveragedMetrics.RP50.Milliseconds()
			rp25 := currentMetrics.AveragedMetrics.RP25.Milliseconds()
			wp99 := currentMetrics.AveragedMetrics.WP99.Milliseconds()
			wp50 := currentMetrics.AveragedMetrics.WP50.Milliseconds()
			wp25 := currentMetrics.AveragedMetrics.WP25.Milliseconds()

			rThroughput := currentMetrics.AveragedMetrics.RThroughput
			hitRate := currentMetrics.AveragedMetrics.HitRate

			log.Info().Msgf("RP99: %vms", rp99)
			log.Info().Msgf("RP50: %vms", rp50)
			log.Info().Msgf("RP25: %vms", rp25)
			log.Info().Msgf("WP99: %vms", wp99)
			log.Info().Msgf("WP50: %vms", wp50)
			log.Info().Msgf("WP25: %vms", wp25)
			log.Info().Msgf("RThroughput: %v", rThroughput)
			log.Info().Msgf("HitRate: %v", hitRate)

		}
	}
}
