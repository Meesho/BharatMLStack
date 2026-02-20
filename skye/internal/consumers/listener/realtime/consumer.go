package realtime

type Consumer interface {
	Process(events []Event) error
}
