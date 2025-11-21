package ds

import "sync"

type SyncMap[K comparable, V any] struct {
	rw  sync.RWMutex
	Map map[K]V
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		Map: make(map[K]V),
	}
}

func (sm *SyncMap[K, V]) Set(key K, value V) {
	sm.rw.Lock()
	defer sm.rw.Unlock()
	sm.Map[key] = value
}

func (sm *SyncMap[K, V]) Get(key K) (V, bool) {
	sm.rw.RLock()
	defer sm.rw.RUnlock()
	value, ok := sm.Map[key]
	return value, ok
}

func (sm *SyncMap[K, V]) DeleteIf(cond func(K, V) bool) {
	sm.rw.RLock()
	var keys []K
	for k, v := range sm.Map {
		if cond(k, v) {
			keys = append(keys, k)
		}
	}
	sm.rw.RUnlock()

	if len(keys) == 0 {
		return
	}
	sm.rw.Lock()
	for _, k := range keys {
		delete(sm.Map, k)
	}
	sm.rw.Unlock()
}
