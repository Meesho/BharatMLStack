package pools

import "sync"

// LeakyPool is a bounded object pool. When all objects are in use, Get creates
// new ones via createFunc. When returned objects exceed capacity, the excess is
// dropped (optionally via a pre-deref hook for cleanup like unmapping pages).
type LeakyPool[T any] struct {
	available   []T
	Meta        any
	createFunc  func() T
	preDrefHook func(obj T)
	capacity    int
	usage       int
	idx         int
	mu          sync.Mutex
}

type LeakyPoolConfig[T any] struct {
	Capacity   int
	Meta       any
	CreateFunc func() T
}

func NewLeakyPool[T any](config LeakyPoolConfig[T]) *LeakyPool[T] {
	return &LeakyPool[T]{
		available:  make([]T, config.Capacity),
		Meta:       config.Meta,
		capacity:   config.Capacity,
		createFunc: config.CreateFunc,
		usage:      0,
		idx:        -1,
	}
}

func (p *LeakyPool[T]) RegisterPreDrefHook(hook func(obj T)) {
	p.preDrefHook = hook
}

func (p *LeakyPool[T]) Get() T {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.usage++
	if p.idx == -1 {
		return p.createFunc()
	}
	o := p.available[p.idx]
	p.idx--
	return o
}

func (p *LeakyPool[T]) Put(obj T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.usage--
	p.idx++
	if p.idx == p.capacity {
		if p.preDrefHook != nil {
			p.preDrefHook(obj)
		}
		p.idx--
		return
	}
	p.available[p.idx] = obj
}
