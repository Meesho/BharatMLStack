package caches

import (
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
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
