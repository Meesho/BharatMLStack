package zerogcoverhead

import (
	"runtime/debug"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metric"
	"github.com/coocood/freecache"
	"github.com/rs/zerolog/log"
)

const (
	metricUpdateInterval = 1 * time.Minute
	infiniteExpiry       = -1
)

type ZeroGCOverhead struct {
	cacheName  string
	inMemCache *freecache.Cache
}

type Conf struct {
	CacheName           string // mandatory
	InMemorySizeInBytes int    // mandatory
	GCPercentage        int    // optional
}

// NewZeroGCOverheadCache creates a new in-memory cache from freecache with zero GC overhead based
// on conf provided. This is expected to call during the application startup
// It publishes the in-memory cache metrics every 1 min
func NewZeroGCOverheadCache(conf Conf) *ZeroGCOverhead {
	if len(conf.CacheName) == 0 {
		log.Panic().Msgf("cache name is empty, conf - %#v", conf)
	}
	if conf.InMemorySizeInBytes == 0 {
		log.Panic().Msgf("invalid in memory cache size, conf - %#v", conf)
	}
	if conf.GCPercentage > 0 {
		log.Warn().Msgf("GC percentage is set to %d, conf - %#v", conf.GCPercentage, conf)
		debug.SetGCPercent(conf.GCPercentage)
	}
	cache := &ZeroGCOverhead{
		cacheName:  conf.CacheName,
		inMemCache: freecache.NewCache(conf.InMemorySizeInBytes),
	}
	go cache.publishMetric()
	return cache
}

func (imc *ZeroGCOverhead) Get(key []byte) ([]byte, error) {
	return imc.inMemCache.Get(key)
}

func (imc *ZeroGCOverhead) Set(key, value []byte) error {
	return imc.inMemCache.Set(key, value, infiniteExpiry)
}

func (imc *ZeroGCOverhead) SetEx(key, value []byte, expiryInSec int) error {
	return imc.inMemCache.Set(key, value, expiryInSec)
}

func (imc *ZeroGCOverhead) Delete(key []byte) bool {
	return imc.inMemCache.Del(key)
}

// publishMetric publishes the in-memory-cache metrics every 1 min, configured by metricUpdateInterval
func (imc *ZeroGCOverhead) publishMetric() {
	ticker := time.NewTicker(metricUpdateInterval)
	cacheMetricTags := metric.BuildTag(metric.NewTag("cache_name", imc.cacheName))
	defer ticker.Stop()
	for range ticker.C {
		metric.Gauge(inmemorycache.HitRate, imc.inMemCache.HitRate(), cacheMetricTags)
		metric.Gauge(inmemorycache.ItemCount, float64(imc.inMemCache.EntryCount()), cacheMetricTags)
		metric.Gauge(inmemorycache.EvacuateCount, float64(imc.inMemCache.EvacuateCount()), cacheMetricTags)
		metric.Gauge(inmemorycache.ExpiryCount, float64(imc.inMemCache.ExpiredCount()), cacheMetricTags)
	}

}
