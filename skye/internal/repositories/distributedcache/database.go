package distributedcache

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
)

var (
	once sync.Once
)

type Database interface {
	MGet(keys map[string]repositories.CacheStruct, metricTags []string) (map[string][]byte, error)
	MSet(responseData map[string]repositories.CandidateResponseStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, byteResponseMap map[string][]byte, metricTags []string)
	MSetDotProduct(cacheKeys map[string]repositories.CacheStruct, foundcacheKeys map[string]repositories.CacheStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, metricTags []string)
}
