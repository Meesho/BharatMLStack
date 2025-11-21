package ds

import (
	"time"
)

type SyncMapWithTtl[K comparable, V any] struct {
	Map SyncMap[K, valueWithTtl[V]]
}

type valueWithTtl[V any] struct {
	value   V
	expires time.Time
}

const (
	sweepInterval = 5 * time.Minute
)

func NewSyncMapWithTtl[K comparable, V any]() *SyncMapWithTtl[K, V] {
	sm := &SyncMapWithTtl[K, V]{
		Map: *NewSyncMap[K, valueWithTtl[V]](),
	}
	go sm.startJanitor()
	return sm
}

func (sm *SyncMapWithTtl[K, V]) Set(key K, value V, ttl time.Duration) {
	sm.Map.Set(key, valueWithTtl[V]{value: value, expires: time.Now().Add(ttl)})
}

func (sm *SyncMapWithTtl[K, V]) Get(key K) (V, bool) {
	value, ok := sm.Map.Get(key)
	if !ok {
		return value.value, false
	}
	return value.value, true
}

// startJanitor periodically deletes expired cache entries
func (sm *SyncMapWithTtl[K, V]) startJanitor() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for now := range ticker.C {
		sm.Map.DeleteIf(func(k K, e valueWithTtl[V]) bool {
			return now.After(e.expires)
		})
	}
}
