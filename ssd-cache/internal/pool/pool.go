package pool

type Pool[T any] interface {
	Get() (T, bool)
	Put(T)
	Count() int64
}
