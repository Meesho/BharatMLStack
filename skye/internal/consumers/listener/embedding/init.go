package embedding

var (
	DefaultVersion = 1
)

func NewConsumer(version int) Consumer {
	switch version {
	case DefaultVersion:
		return newEmbeddingConsumer()
	default:
		return nil
	}
}
