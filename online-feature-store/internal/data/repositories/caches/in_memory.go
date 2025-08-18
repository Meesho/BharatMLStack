package caches

import (
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/coocood/freecache"
	"github.com/rs/zerolog/log"
)

var (
	metricUpdateInterval = 10 * time.Minute
	HitRate              = "in_memory_cache_hit_rate"
	ItemCount            = "in_memory_cache_item_count"
	EvacuateCount        = "in_memory_cache_evacuate_count"
	ExpiryCount          = "in_memory_cache_expiry_count"
)

type InMemoryCache struct {
	cache             *freecache.Cache
	inMemoryCacheName string
	config            config.Manager
}

func NewInMemoryCache(conn *infra.InMemoryCacheConnection) (Cache, error) {
	meta, err := conn.GetMeta()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)
	cache := &InMemoryCache{
		cache:             conn.Client,
		inMemoryCacheName: meta["name"].(string),
		config:            configManager,
	}
	go publishMetric(conn.Client, meta["name"].(string))
	return cache, nil
}

func (i *InMemoryCache) GetV2(entityLabel string, keys *retrieve.Keys) []byte {
	k := buildCacheKeyForRetrieve(keys, entityLabel)
	v, err := i.cache.Get([]byte(k))
	if err != nil {
		log.Debug().Msgf("in-memory cache miss for key: %s", k)
		return nil
	}
	log.Debug().Msgf("in-memory cache hit for key : %s", k)
	return v
}

func (i *InMemoryCache) MultiGetV2(entityLabel string, bulkKeys []*retrieve.Keys) ([][]byte, error) {
	panic("implement")
}

func publishMetric(cache *freecache.Cache, cacheName string) {
	ticker := time.NewTicker(metricUpdateInterval)
	cacheMetricTags := metric.BuildTag(metric.NewTag("cache_name", cacheName))
	defer func() {
		ticker.Stop()
		if r := recover(); r != nil {
			// log.Error().Msgf("Panic recovered: %v", r)
			metric.Count("online-feature-store.in-memory.panic.count", 1, nil)
		}
	}()
	for range ticker.C {
		metric.Gauge(HitRate, cache.HitRate(), cacheMetricTags)
		metric.Gauge(ItemCount, float64(cache.EntryCount()), cacheMetricTags)
		metric.Gauge(EvacuateCount, float64(cache.EvacuateCount()), cacheMetricTags)
		metric.Gauge(ExpiryCount, float64(cache.ExpiredCount()), cacheMetricTags)
	}

}

func (i *InMemoryCache) Delete(entityLabel string, key []string) error {
	k := buildCacheKeyForPersist(key, entityLabel)
	i.cache.Del([]byte(k))
	return nil
}

func (i *InMemoryCache) SetV2(entityLabel string, keys []string, data []byte) error {
	k := buildCacheKeyForPersist(keys, entityLabel)
	cacheConfig, err := i.config.GetInMemoryCacheConfForEntity(entityLabel)
	if err != nil {
		return err
	}
	ttlInSeconds := getFinalTTLWithJitter(cacheConfig)
	err = i.cache.Set([]byte(k), data, ttlInSeconds)
	log.Debug().Msgf("Set key: %s, data: %s, ttl: %d", k, data, ttlInSeconds)
	if err != nil {
		metric.Count("persist.failure", 1, []string{"cache_type", "in_memory", "entity", entityLabel})
		return err
	}
	return nil
}

func (i *InMemoryCache) MultiSetV2(entityLabel string, bulkKeys []*retrieve.Keys, values [][]byte) error {
	return fmt.Errorf("%w: MultiSetV2 for in-memory cache", ErrNotImplemented)
}
