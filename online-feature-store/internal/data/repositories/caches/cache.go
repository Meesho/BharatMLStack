package caches

import (
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"math/rand"
)

const (
	CacheTypeDistributed = "distributed"
	CacheTypeInMem       = "in_memory"
)

type Cache interface {
	GetV2(entityLabel string, keys *retrieve.Keys) []byte
	MultiGetV2(entityLabel string, bulkKeys []*retrieve.Keys) ([][]byte, error)
	Delete(entityLabel string, key []string) error
	SetV2(entityLabel string, keys []string, data []byte) error
	MultiSetV2(entityLabel string, bulkKeys []*retrieve.Keys, values [][]byte) error
}

// builds cache key
func buildCacheKeyForRetrieve(keys *retrieve.Keys, entityLabel string) string {
	return entityLabel + ":" + strings.Join(keys.Cols, "|")
}

func buildCacheKeyForPersist(keys []string, entityLabel string) string {
	return entityLabel + ":" + strings.Join(keys, "|")
}

func getFinalTTLWithJitter(cacheConfig *config.Cache) int{
	ttlInSeconds := cacheConfig.TtlInSeconds
	jitterPercentage := cacheConfig.JitterPercentage
	jitterRange := ttlInSeconds * jitterPercentage / 100
	jitter := rand.Intn(2*jitterRange+1) - jitterRange // Random value between -jitterRange and +jitterRange
	finalTTL := ttlInSeconds + jitter

	// Ensure TTL is positive
	if finalTTL < 1 {
		finalTTL = ttlInSeconds
	}
	return finalTTL
}
