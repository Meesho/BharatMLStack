package inmemorycache

import "sync"

const (
	inMemoryCacheSizeInBytes = "IN_MEMORY_CACHE_SIZE_IN_BYTES"
	appGCPercentage          = "APP_GC_PERCENTAGE"
)

var (
	once sync.Once
)

const (
	HitRate       = "in_memory_cache_hit_rate"
	ItemCount     = "in_memory_cache_item_count"
	EvacuateCount = "in_memory_cache_evacuate_count"
	ExpiryCount   = "in_memory_cache_expiry_count"
)

type InMemoryCache interface {
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	SetEx(key, value []byte, expiryInSec int) error
	Delete(key []byte) bool
}
