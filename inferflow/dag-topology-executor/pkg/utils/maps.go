package utils

import "sync"

type ConcurrentMap struct {
	m     map[string]int
	mutex sync.Mutex
}

func NewConcurrentMap() *ConcurrentMap {
	return &ConcurrentMap{m: make(map[string]int)}
}

func (c *ConcurrentMap) Get(key string) (int, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	value, ok := c.m[key]
	return value, ok
}

func (c *ConcurrentMap) Set(key string, value int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.m[key] = value
}

func (c *ConcurrentMap) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.m, key)
}

func (c *ConcurrentMap) Size() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return len(c.m)
}

func (c *ConcurrentMap) UpdateAndGet(key string) (int, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if value, ok := c.m[key]; ok {
		value--
		c.m[key] = value
	}
	value, ok := c.m[key]
	return value, ok
}
