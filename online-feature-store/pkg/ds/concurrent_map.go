package ds

import "sync"

type ConcurrentMap[K comparable, V any] struct {
	rw  sync.RWMutex
	Map map[K]V
}

func NewConcurrentMap[K comparable, V any]() *ConcurrentMap[K, V] {
	return &ConcurrentMap[K, V]{
		Map: make(map[K]V),
	}
}

func (sm *ConcurrentMap[K, V]) Set(key K, value V) {
	sm.rw.Lock()
	defer sm.rw.Unlock()
	sm.Map[key] = value
}

func (sm *ConcurrentMap[K, V]) Get(key K) (V, bool) {
	sm.rw.RLock()
	defer sm.rw.RUnlock()
	value, ok := sm.Map[key]
	return value, ok
}
