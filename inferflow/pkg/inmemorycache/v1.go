package inmemorycache

import (
	"runtime/debug"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/metric"
	"github.com/coocood/freecache"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	metricUpdateInterval = 1 * time.Minute
	infiniteExpiry       = -1
)

type V1 struct {
	cacheName  string
	inMemCache *freecache.Cache
}

func newV1InMemoryCache(cName string) InMemoryCache {
	gcPercentage := -1
	if !viper.IsSet(inMemoryCacheSizeInBytes) {
		log.Panic().Msgf("env::IN_MEMORY_CACHE_SIZE_IN_BYTES is not set !!")
	}
	sizeInBytes := viper.GetInt(inMemoryCacheSizeInBytes)

	if !viper.IsSet(appGCPercentage) {
		log.Warn().Msgf("env::APP_GC_PERCENTAGE is not set")
	} else {
		gcPercentage = viper.GetInt(appGCPercentage)
	}

	v1InMemoryCache := &V1{
		cacheName:  cName,
		inMemCache: freecache.NewCache(sizeInBytes),
	}
	if gcPercentage != -1 {
		debug.SetGCPercent(gcPercentage)
	}
	go v1InMemoryCache.publishMetric()
	return v1InMemoryCache
}

func (imc *V1) Get(key []byte) ([]byte, error) {
	return imc.inMemCache.Get(key)
}

func (imc *V1) Set(key, value []byte) error {
	return imc.inMemCache.Set(key, value, infiniteExpiry)
}

func (imc *V1) SetEx(key, value []byte, expiryInSec int) error {
	return imc.inMemCache.Set(key, value, expiryInSec)
}

func (imc *V1) Delete(key []byte) bool {
	return imc.inMemCache.Del(key)
}

// publishMetric publishes the in-memory-cache metrics every 1 min, configured by metricUpdateInterval
func (imc *V1) publishMetric() {
	ticker := time.NewTicker(metricUpdateInterval)
	cacheMetricTags := metric.BuildTag(metric.NewTag("cache_name", imc.cacheName))
	defer ticker.Stop()
	for range ticker.C {
		metric.Gauge(HitRate, imc.inMemCache.HitRate(), cacheMetricTags)
		metric.Gauge(ItemCount, float64(imc.inMemCache.EntryCount()), cacheMetricTags)
		metric.Gauge(EvacuateCount, float64(imc.inMemCache.EvacuateCount()), cacheMetricTags)
		metric.Gauge(ExpiryCount, float64(imc.inMemCache.ExpiredCount()), cacheMetricTags)
	}

}
