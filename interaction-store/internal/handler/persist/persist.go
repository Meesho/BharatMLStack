package persist

type PersistHandler interface {
	Persist(userId string, data any) error
}
