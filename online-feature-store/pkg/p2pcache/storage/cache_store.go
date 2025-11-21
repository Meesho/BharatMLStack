package storage

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/coocood/freecache"
)

var (
	HitRate       = "p2p_cache_hit_rate"
	ItemCount     = "p2p_cache_item_count"
	EvacuateCount = "p2p_cache_evacuate_count"
	ExpiryCount   = "p2p_cache_expiry_count"
)

type CacheStore struct {
	ownPartitionCache *freecache.Cache
	globalCache       *freecache.Cache
}

func NewCacheStore(ownPartitionSizeInBytes int, globalSizeInBytes int) *CacheStore {
	return &CacheStore{
		ownPartitionCache: freecache.NewCache(ownPartitionSizeInBytes),
		globalCache:       freecache.NewCache(globalSizeInBytes),
	}
}

func (c *CacheStore) Get(key string) ([]byte, error) {
	value, err := c.ownPartitionCache.Get([]byte(key))
	if err == nil {
		return value, nil
	}
	value, err = c.globalCache.Get([]byte(key))
	if err == nil {
		return value, nil
	}
	return nil, err
}

func (c *CacheStore) MultiSetIntoGlobalCache(kvMap map[string][]byte, ttlInSeconds int) error {
	for key, value := range kvMap {
		_ = c.globalCache.Set([]byte(key), value, ttlInSeconds)
	}
	return nil
}

func (c *CacheStore) MultiSetIntoOwnPartitionCache(kvMap map[string][]byte, ttlInSeconds int) error {
	for key, value := range kvMap {
		_ = c.ownPartitionCache.Set([]byte(key), value, ttlInSeconds)
	}
	return nil
}

func (c *CacheStore) SetIntoOwnPartitionCache(key string, value []byte, ttlInSeconds int) error {
	return c.ownPartitionCache.Set([]byte(key), value, ttlInSeconds)
}

func (c *CacheStore) SetIntoGlobalCache(key string, value []byte, ttlInSeconds int) error {
	return c.globalCache.Set([]byte(key), value, ttlInSeconds)
}

func (c *CacheStore) MultiDelete(keys []string) error {
	for _, key := range keys {
		c.ownPartitionCache.Del([]byte(key))
		c.globalCache.Del([]byte(key))
	}
	return nil
}

func (c *CacheStore) PublishMetrics(cacheName string) {
	go publishMetric(c.ownPartitionCache, cacheName, "own_partition")
	go publishMetric(c.globalCache, cacheName, "global")
}

func publishMetric(cache *freecache.Cache, cacheName string, cacheType string) {
	cacheMetricTags := metric.BuildTag(metric.NewTag("cache_name", cacheName), metric.NewTag("cache_type", cacheType))
	metric.Gauge(HitRate, cache.HitRate(), cacheMetricTags)
	metric.Gauge(ItemCount, float64(cache.EntryCount()), cacheMetricTags)
	metric.Gauge(EvacuateCount, float64(cache.EvacuateCount()), cacheMetricTags)
	metric.Gauge(ExpiryCount, float64(cache.ExpiredCount()), cacheMetricTags)
}
