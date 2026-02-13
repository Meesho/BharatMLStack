package aggregator

var (
	SCYLLA = "SCYLLA"
)

func NewHandler(handlerType string) Handler {
	switch handlerType {
	case SCYLLA:
		return initScyllaAggregator()
	default:
		return nil
	}
}
