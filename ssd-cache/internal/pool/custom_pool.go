package pool

import (
	"sync"
)

type CustomPool struct {
	pool       sync.Pool
	totalInUse int64 // Total number of Get operations
	newFunc    func() interface{}
}

// NewCustomPool creates a new custom pool with the given factory function
func NewCustomPool(newFunc func() interface{}) *CustomPool {
	cp := &CustomPool{
		newFunc: newFunc,
	}
	cp.pool = sync.Pool{
		New: func() interface{} {
			return cp.newFunc()
		},
	}
	return cp
}

// Get retrieves an object from the pool
func (cp *CustomPool) Get() interface{} {
	obj := cp.pool.Get()
	cp.totalInUse++
	return obj
}

// Put returns an object to the pool
func (cp *CustomPool) Put(obj interface{}) {
	cp.totalInUse--
	cp.pool.Put(obj)
}

// Count returns the current number of objects in the pool (approximate)
func (cp *CustomPool) Count() int64 {
	return cp.totalInUse
}
