package caches

import (
	"context"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	redisCache Cache
)

type RedisCache struct {
	conn     redis.UniversalClient
	DBType   infra.DBType
	configId int
	config   config.Manager
}

func NewRedisCacheFromRedisClusterConnection(conn *infra.RedisClusterConnection) (Cache, error) {
	meta, err := conn.GetMeta()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)
	redisCache = &RedisCache{
		conn:     conn.Client,
		DBType:   meta["type"].(infra.DBType),
		configId: meta["configId"].(int),
		config:   configManager,
	}
	return redisCache, nil
}

func NewRedisCacheFromRedisStandaloneConnection(conn *infra.RedisStandaloneConnection) (Cache, error) {
	meta, err := conn.GetMeta()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)
	redisCache = &RedisCache{
		conn:     conn.Client,
		DBType:   meta["type"].(infra.DBType),
		configId: meta["configId"].(int),
		config:   configManager,
	}
	return redisCache, nil
}

func NewRedisCacheFromRedisFailoverConnection(conn *infra.RedisFailoverConnection) (Cache, error) {
	meta, err := conn.GetMeta()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)
	redisCache = &RedisCache{
		conn:     conn.Client,
		DBType:   meta["type"].(infra.DBType),
		configId: meta["configId"].(int),
		config:   configManager,
	}
	return redisCache, nil
}

func (r *RedisCache) GetV2(entityLabel string, keys *retrieve.Keys) []byte {
	panic("implement me")
}

func (r *RedisCache) MultiGetV2(entityLabel string, bulkKeys []*retrieve.Keys) ([][]byte, error) {
	t1 := time.Now()
	cacheKeys := make([]string, len(bulkKeys))
	cacheData := make([][]byte, len(bulkKeys))
	for i, keys := range bulkKeys {
		k := buildCacheKeyForRetrieve(keys, entityLabel)
		cacheKeys[i] = k
	}
	v, err := r.conn.MGet(context.Background(), cacheKeys...).Result()

	if err != nil {
		metric.Count("feature.retrieve.cache.error", 1, []string{"entity_name", entityLabel, "cache_type", "distributed"})
		// log.Error().Err(err).Msgf("distributed cache get error for keys : %s", cacheKeys)
		return nil, fmt.Errorf("failed to retrieve data from Redis: %s", err)
	}
	if v == nil {
		metric.Count("feature.retrieve.cache.miss", 1, []string{"entity_name", entityLabel, "cache_type", "distributed"})
		log.Debug().Msgf("distributed cache miss for keys: %s", cacheKeys)
		return nil, nil
	}
	idx := 0
	for idx < len(bulkKeys) {
		if v[idx] == nil || v[idx] == "" {
			metric.Count("feature.retrieve.cache.miss", 1, []string{"entity_name", entityLabel, "cache_type", "distributed"})
			log.Debug().Msgf("distributed cache miss cacheKey : %s", cacheKeys[idx])
			idx++
			continue
		}
		metric.Count("feature.retrieve.cache.hit", 1, []string{"entity_name", entityLabel, "cache_type", "distributed"})
		log.Debug().Msgf("distributed cache hit cacheKey : %s", cacheKeys[idx])
		cacheData[idx] = []byte(v[idx].(string))
		idx++
	}
	metric.Timing("feature.retrieve.cache.latency", time.Since(t1), []string{"entity_name", entityLabel, "cache_type", "distributed"})
	return cacheData, nil

}

func (r *RedisCache) Delete(entityLabel string, key []string) error {
	k := buildCacheKeyForPersist(key, entityLabel)
	err := r.conn.Del(context.Background(), k).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key: %s", err)
	}
	return nil
}

func (r *RedisCache) SetV2(entityLabel string, keys []string, data []byte) error {
	k := buildCacheKeyForPersist(keys, entityLabel)
	cacheConfig, err := r.config.GetDistributedCacheConfForEntity(entityLabel)
	if err != nil {
		return err
	}
	ttlInSeconds := getFinalTTLWithJitter(cacheConfig)
	err = r.conn.Set(context.Background(), k, data, time.Duration(ttlInSeconds)*time.Second).Err()
	log.Debug().Msgf("Set key: %s, data: %s, ttl: %d", k, data, ttlInSeconds)
	if err != nil {
		metric.Count("persist.failure", 1, []string{"cache_type", "distributed", "entity", entityLabel})
		return err
	}
	return nil
}

func (r *RedisCache) MultiSetV2(entityLabel string, keys []*retrieve.Keys, values [][]byte) error {
	if len(keys) == 0 || len(values) == 0 {
		return fmt.Errorf("%w: keys or values length is zero", ErrInvalidInput)
	}
	if len(keys) != len(values) {
		return fmt.Errorf("%w: keys and values length mismatch", ErrInvalidInput)
	}
	cacheConfig, err := r.config.GetDistributedCacheConfForEntity(entityLabel)
	if err != nil {
		return err
	}
	ttlInSeconds := getFinalTTLWithJitter(cacheConfig)
	pipe := r.conn.Pipeline()

	for i, key := range keys {
		k := buildCacheKeyForPersist(key.Cols, entityLabel)
		pipe.Set(context.Background(), k, values[i], time.Duration(ttlInSeconds)*time.Second)
		log.Debug().Msgf("Set key: %s, data: %s, ttl: %d", k, values[i], ttlInSeconds)
	}

	_, err2 := pipe.Exec(context.Background())
	if err2 != nil {
		metric.Count("persist.failure", 1, []string{"cache_type", "distributed", "entity", entityLabel})
		return err
	}
	return nil
}