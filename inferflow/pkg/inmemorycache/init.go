package inmemorycache

import (
	"errors"
	"github.com/rs/zerolog/log"
	"sync"
)

var (
	namedInstances map[string]InMemoryCache
	instance       InMemoryCache
	cacheOnce      sync.Once
)

const inMemoryCacheV1Name = "in_memory_cache_v1"

// Init initializes the in-memory-cache, to be called from main.go
func Init(version int) {
	once.Do(func() {
		switch version {
		case 1:
			instance = newV1InMemoryCache(inMemoryCacheV1Name)
		default:
			log.Panic().Msgf("invalid version %d", version)
		}
	})
}

func InitMultiInMemoryCache(cacheNames []string) {
	// Note: This function does not use once.Do since multiple caches can be initialized

	if namedInstances == nil {
		namedInstances = make(map[string]InMemoryCache)
	}

	for _, cacheName := range cacheNames {
		if _, exist := namedInstances[cacheName]; exist {
			log.Panic().Msgf("Inmemory with Cache Name - %v already exist, Please check initialize of all inmemory cache",
				cacheName)
		}
		// Initialize each cache individually
		// Use a sync.Once to ensure that each specific cache is only initialized once
		cacheOnce.Do(func() {
			namedInstances[cacheName] = newV1InMemoryCache(cacheName)
		})
	}
}

// InitV1 initializes the in-memory-cache with version 1
func InitV1() {
	Init(1)
}

// Instance returns the in-memory-cache instance. Ensure that Init
// is called before calling this function
func Instance() InMemoryCache {
	if instance == nil {
		log.Panic().Msg("in-memory-cache not initialized, call Init first")
	}
	return instance
}

func InstanceByName(cacheName string) (InMemoryCache, error) {
	if _, exist := namedInstances[cacheName]; !exist {
		return nil, errors.New("in-memory-cache not initialized, call Init first")
	}
	return namedInstances[cacheName], nil
}

// SetMockInstance sets the mock instance of in-memory-cache
// This would be handy in places where we are directly using in-memory-cache as inmemorycache.Instance()
func SetMockInstance(mock InMemoryCache) {
	instance = mock
}
