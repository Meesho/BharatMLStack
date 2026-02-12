package ds

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyncMap(t *testing.T) {
	sm := NewSyncMap[string, int]()
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.Map)
	assert.Len(t, sm.Map, 0)
}

func TestSyncMap_SetAndGet(t *testing.T) {
	sm := NewSyncMap[string, int]()

	sm.Set("a", 1)
	sm.Set("b", 2)

	val, ok := sm.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = sm.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
}

func TestSyncMap_GetMissing(t *testing.T) {
	sm := NewSyncMap[string, string]()

	val, ok := sm.Get("missing")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestSyncMap_Overwrite(t *testing.T) {
	sm := NewSyncMap[string, int]()
	sm.Set("key", 10)
	sm.Set("key", 20)

	val, ok := sm.Get("key")
	assert.True(t, ok)
	assert.Equal(t, 20, val)
}

func TestSyncMap_ConcurrentAccess(t *testing.T) {
	sm := NewSyncMap[int, int]()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sm.Set(i, i*2)
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		val, ok := sm.Get(i)
		assert.True(t, ok)
		assert.Equal(t, i*2, val)
	}
}
