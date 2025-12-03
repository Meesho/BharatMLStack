package set

import (
	"sync"

	"github.com/emirpasic/gods/sets/hashset"
)

// implement more methods exposed by hashset.Set if required
type ThreadSafeSet struct {
	set     *hashset.Set
	rwMutex sync.RWMutex
}

func NewThreadSafeSet(items ...interface{}) *ThreadSafeSet {
	hashSet := hashset.New(items...)
	return &ThreadSafeSet{set: hashSet, rwMutex: sync.RWMutex{}}
}

func (t *ThreadSafeSet) Contains(items ...interface{}) bool {
	// multiple goroutine reads allowed
	t.rwMutex.RLock()
	defer t.rwMutex.RUnlock()
	return t.set.Contains(items...)
}

func (t *ThreadSafeSet) Add(items ...interface{}) {
	// read-write lock
	t.rwMutex.Lock()
	defer t.rwMutex.Unlock()
	t.set.Add(items...)
}

func (t *ThreadSafeSet) Remove(items ...interface{}) {
	// read-write lock
	t.rwMutex.Lock()
	defer t.rwMutex.Unlock()
	t.set.Remove(items...)
}

func (t *ThreadSafeSet) Clear() {
	// read-write lock
	t.rwMutex.Lock()
	defer t.rwMutex.Unlock()
	t.set.Clear()
}
