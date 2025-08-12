package storage

import (
	"github.com/coocood/freecache"
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
