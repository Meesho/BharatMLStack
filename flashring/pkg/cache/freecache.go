package cache

import (
	"runtime/debug"
	"time"

	"github.com/coocood/freecache"
)

// Freecache adapts coocood/freecache to the Cache interface as a
// comparison backend for the benchmark harness in cmd/flashringtest.
// It is not the production NVMe path — for that, use WrapCache.
type Freecache struct {
	cache *freecache.Cache
}

// NewFreecache lowers the process-wide GC target percentage to 20%
// (from the default 100%) to keep freecache's large off-heap regions
// from triggering excessive heap growth during benchmarks. This is a
// global Go runtime change and persists for the lifetime of the
// process.
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
