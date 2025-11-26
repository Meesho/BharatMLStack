package cache

import (
	"github.com/Meesho/BharatMLStack/dag-topology-executor/pkg/logger"
)

func InitRistrettoCache(cacheSize, cacheTTL int64) *Cache {
	cache := NewCache(cacheSize, cacheTTL)
	logger.Info("Ristretto cache initialized")
	return cache
}
