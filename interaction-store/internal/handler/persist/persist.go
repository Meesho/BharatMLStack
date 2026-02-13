package persist

type PersistHandler[T any] interface {
	Persist(userId string, data T) error
}
