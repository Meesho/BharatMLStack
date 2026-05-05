package pools

// Pool is a generic object pool that reuses pre-allocated objects.
type Pool[T any] interface {
	Get() T
	Put(obj T)
}
