package indexer

var (
	QDRANT = "QDRANT"
)

func NewHandler(handlerType string) Handler {
	switch handlerType {
	case QDRANT:
		return initQdrantIndexerHandler()
	default:
		return nil
	}
}
