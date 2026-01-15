package retrieve

type RetrieveHandler[T any] interface {
	Retrieve(userId string, startTimestampMs int64, endTimestampMs int64, limit int32) (data T, err error)
}
