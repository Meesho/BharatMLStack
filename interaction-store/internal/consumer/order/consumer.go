package order

type Consumer interface {
	Process(event []OrderPlacedEvent) error
}
