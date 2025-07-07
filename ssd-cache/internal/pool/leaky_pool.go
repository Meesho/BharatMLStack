package pool

type LeakyPool struct {
	availabilityList []interface{}
	createFunc       func() interface{}
	preDrefHook      func(obj interface{})
	capacity         int
	usage            int
	idx              int
}

func NewLeakyPool(capacity int, createFunc func() interface{}) *LeakyPool {
	return &LeakyPool{
		availabilityList: make([]interface{}, capacity),
		capacity:         capacity,
		createFunc:       createFunc,
		usage:            0,
		idx:              -1,
		preDrefHook:      nil,
	}
}

func (p *LeakyPool) RegisterPreDrefHook(hook func(obj interface{})) {
	p.preDrefHook = hook
}

func (p *LeakyPool) Get() (interface{}, bool) {
	p.usage++
	if p.idx == -1 && p.usage > p.capacity {
		return p.createFunc(), true
	} else if p.idx == -1 {
		return p.createFunc(), false
	}
	o := p.availabilityList[p.idx]
	p.idx--
	return o, false
}

func (p *LeakyPool) Put(obj interface{}) {
	p.usage--
	p.idx++
	if p.idx == p.capacity {
		if p.preDrefHook != nil {
			p.preDrefHook(obj)
		}
		p.idx--
		return
	}
	p.availabilityList[p.idx] = obj
}
