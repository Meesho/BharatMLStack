package pools

type Pool interface {
	Get() interface{}
	Put(obj interface{})
	RegisterPreDrefHook(hook func(obj interface{}))
}
