package pools

type LeakyPool struct {
	availabilityList []interface{}
	Meta             interface{}
	createFunc       func() interface{}
	preDrefHook      func(obj interface{})
	capacity         int
	usage            int
	idx              int
	stats            *Stats
}

type Stats struct {
	Usage    int
	Capacity int
}

type LeakyPoolConfig struct {
	Capacity   int
	Meta       interface{}
	CreateFunc func() interface{}
}

func NewLeakyPool(config LeakyPoolConfig) *LeakyPool {
	return &LeakyPool{
		availabilityList: make([]interface{}, config.Capacity),
		Meta:             config.Meta,
		capacity:         config.Capacity,
		createFunc:       config.CreateFunc,
		usage:            0,
		idx:              -1,
		preDrefHook:      nil,
		stats:            &Stats{Usage: 0, Capacity: config.Capacity},
	}
}

func (p *LeakyPool) RegisterPreDrefHook(hook func(obj interface{})) {
	p.preDrefHook = hook
}

func (p *LeakyPool) Get() interface{} {
	p.usage++
	if p.idx == -1 && p.usage > p.capacity {
		return p.createFunc()
	} else if p.idx == -1 {
		return p.createFunc()
	}
	o := p.availabilityList[p.idx]
	p.idx--
	return o
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
