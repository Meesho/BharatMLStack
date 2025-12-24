package internal

import (
	"runtime/debug"
	"sync/atomic"
	"time"

	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/coocood/freecache"
	"github.com/rs/zerolog/log"
)

type Freecache struct {
	cache *freecache.Cache
	stats *CacheStats
}

func NewFreecache(config WrapCacheConfig, logStats bool) (*Freecache, error) {

	cache := freecache.NewCache(int(config.FileSize))
	debug.SetGCPercent(20)

	fc := &Freecache{
		cache: cache,
		stats: &CacheStats{
			Hits:                   atomic.Uint64{},
			TotalGets:              atomic.Uint64{},
			TotalPuts:              atomic.Uint64{},
			ReWrites:               atomic.Uint64{},
			Expired:                atomic.Uint64{},
			ShardWiseActiveEntries: atomic.Uint64{},
			LatencyTracker:         filecache.NewLatencyTracker(),
		},
	}

	if logStats {
		go func() {
			sleepDuration := 10 * time.Second
			var prevTotalGets, prevTotalPuts uint64
			for {
				time.Sleep(sleepDuration)

				totalGets := fc.stats.TotalGets.Load()
				totalPuts := fc.stats.TotalPuts.Load()
				getsPerSec := float64(totalGets-prevTotalGets) / sleepDuration.Seconds()
				putsPerSec := float64(totalPuts-prevTotalPuts) / sleepDuration.Seconds()

				log.Info().Msgf("Shard %d HitRate: %v", 0, cache.HitRate())
				log.Info().Msgf("Shard %d Expired: %v", 0, cache.ExpiredCount())
				log.Info().Msgf("Shard %d Total: %v", 0, cache.EntryCount())
				log.Info().Msgf("Gets/sec: %v", getsPerSec)
				log.Info().Msgf("Puts/sec: %v", putsPerSec)

				getP25, getP50, getP99 := fc.stats.LatencyTracker.GetLatencyPercentiles()
				putP25, putP50, putP99 := fc.stats.LatencyTracker.PutLatencyPercentiles()

				log.Info().Msgf("Get Count: %v", totalGets)
				log.Info().Msgf("Put Count: %v", totalPuts)
				log.Info().Msgf("Get Latencies - P25: %v, P50: %v, P99: %v", getP25, getP50, getP99)
				log.Info().Msgf("Put Latencies - P25: %v, P50: %v, P99: %v", putP25, putP50, putP99)

				prevTotalGets = totalGets
				prevTotalPuts = totalPuts
			}
		}()
	}

	return fc, nil

}

func (c *Freecache) Put(key string, value []byte, exptimeInMinutes uint16) error {
	start := time.Now()
	defer func() {
		c.stats.LatencyTracker.RecordPut(time.Since(start))
	}()

	c.stats.TotalPuts.Add(1)
	c.cache.Set([]byte(key), value, int(exptimeInMinutes)*60)
	return nil
}

func (c *Freecache) Get(key string) ([]byte, bool, bool) {
	start := time.Now()
	defer func() {
		c.stats.LatencyTracker.RecordGet(time.Since(start))
	}()

	c.stats.TotalGets.Add(1)
	val, err := c.cache.Get([]byte(key))
	if err != nil {
		return nil, false, false
	}
	c.stats.Hits.Add(1)
	return val, true, false
}
