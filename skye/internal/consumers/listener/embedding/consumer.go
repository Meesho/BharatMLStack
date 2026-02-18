package embedding

type Consumer interface {
	Process(event []Event) error
	ProcessInSequence(events []Event) error
}
