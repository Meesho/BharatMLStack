package inmemorycache

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
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
	sizeInMb   int
	inMemCache *freecache.Cache
}

type V1Builder struct {
	v1 *V1
}

// newV1Builder creates a new builder with required cache name
func newV1Builder(cacheName string) *V1Builder {
	if cacheName == "" {
		log.Panic().Msg("cache name cannot be empty")
	}
	return &V1Builder{
		v1: &V1{
			cacheName: cacheName,
		},
	}
}

// WithSizeInMB sets the cache size in megabytes
func (b *V1Builder) withSizeInMB(sizeMB int) *V1Builder {
	if sizeMB <= 0 {
		log.Panic().Msgf("cache size must be positive, got %d MB", sizeMB)
	}
	b.v1.sizeInMb = sizeMB
	return b
}

// SizeInMb returns the cache size in bytes
func (v *V1) SizeInMb() int {
	return v.sizeInMb
}

// Build creates the final CacheConfig
func (b *V1Builder) build() (*V1, error) {
	// Validate configuration
	if b.v1.cacheName == "" {
		return nil, fmt.Errorf("cache name is required")
	}
	if b.v1.sizeInMb <= 0 {
		return nil, fmt.Errorf("invalid cache size: %d MB", b.v1.sizeInMb)
	}
	b.v1.inMemCache = freecache.NewCache(b.v1.sizeInMb * 1024 * 1024)
	return b.v1, nil
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
		sizeInMb:   sizeInBytes / (1024 * 1024),
		inMemCache: freecache.NewCache(sizeInBytes),
	}

	if gcPercentage != -1 {
		debug.SetGCPercent(gcPercentage)
	}
	go v1InMemoryCache.publishMetric()
	return v1InMemoryCache
}

func newV1InMemoryCacheWithConf(cName string, sizeInMb int) InMemoryCache {
	v1InCacheBuilder := newV1Builder(cName)
	v1InMemoryCache, err := v1InCacheBuilder.withSizeInMB(sizeInMb).build()
	if err != nil {
		log.Panic().Err(err).Msgf("error building v1 in memory cache with conf")
	}
	gcPercentage := -1
	if !viper.IsSet(appGCPercentage) {
		log.Warn().Msgf("env::APP_GC_PERCENTAGE is not set")
	} else {
		gcPercentage = viper.GetInt(appGCPercentage)
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
