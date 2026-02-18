package indexer

type EventType string

const (
	Upsert        EventType = "UPSERT"
	Delete        EventType = "DELETE"
	UpsertPayload EventType = "UPSERT_PAYLOAD"
)

type Event struct {
	Data map[EventType][]Data
}

type Data struct {
	Entity  string
	Model   string
	Variant string
	Version int
	Id      string
	Payload map[string]interface{}
	Vectors []float32
}
