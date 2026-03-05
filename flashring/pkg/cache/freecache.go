package cache

import (
	"runtime/debug"
	"time"

	"github.com/coocood/freecache"
)

type Freecache struct {
	cache *freecache.Cache
}

func NewFreecache(sizeBytes int) (*Freecache, error) {
	cache := freecache.NewCache(sizeBytes)
	debug.SetGCPercent(20)
	return &Freecache{cache: cache}, nil
}

func (c *Freecache) Put(key string, value []byte, ttl time.Duration) error {
	c.cache.Set([]byte(key), value, int(ttl.Seconds()))
	return nil
}

func (c *Freecache) Get(key string) ([]byte, bool, bool) {
	val, err := c.cache.Get([]byte(key))
	if err != nil {
		return nil, false, false
	}
	return val, true, false
}

func (c *Freecache) Close() error {
	return nil
}
