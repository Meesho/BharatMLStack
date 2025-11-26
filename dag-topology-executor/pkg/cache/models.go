package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

type RistrettoConfig struct {
	Ttl  int64 `koanf:"ttlSec"`    // Expiration time in seconds
	Size int64 `koanf:"cacheSize"` // Maximum number of items to be cached
}

type Cache struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

func NewCache(size int64, ttl int64) *Cache {
	cache, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(10 * size),
		MaxCost:     size,
		BufferItems: 64,
	})
	t := time.Duration(ttl) * time.Second
	return &Cache{cache: cache, ttl: t}
}

func (c *Cache) SetWithTTL(key string, value interface{}) {

	// Schedule a function to delete the key from the cache after the TTL has expired
	time.AfterFunc(c.ttl, func() {
		c.cache.Del(key)
	})

	c.Set(key, value)
}

func (c *Cache) Set(key string, value interface{}) {
	c.cache.Set(key, value, 1)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	value, found := c.cache.Get(key)
	return value, found
}
