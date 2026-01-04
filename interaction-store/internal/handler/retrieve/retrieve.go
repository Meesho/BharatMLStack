package retrieve

type RetrieveHandler interface {
	Retrieve(userId string, startTimestampMs int64, endTimestampMs int64, limit int32) (data any, err error)
}
