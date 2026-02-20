package internal

import (
	"runtime/debug"

	"github.com/coocood/freecache"
)

type Freecache struct {
	cache *freecache.Cache
}

func NewFreecache(config WrapCacheConfig, logStats bool) (*Freecache, error) {

	cache := freecache.NewCache(int(config.FileSize))
	debug.SetGCPercent(20)

	fc := &Freecache{
		cache: cache,
	}

	return fc, nil

}

func (c *Freecache) Put(key string, value []byte, exptimeInMinutes uint16) error {

	c.cache.Set([]byte(key), value, int(exptimeInMinutes)*60)
	return nil
}

func (c *Freecache) Get(key string) ([]byte, bool, bool) {

	val, err := c.cache.Get([]byte(key))
	if err != nil {
		return nil, false, false
	}

	return val, true, false
}
