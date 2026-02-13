package realtime

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

type Event struct {
	Timestamp int    `json:"timestamp"`
	Type      string `json:"type"`
	Entity    string `json:"entity"`
	Data      []Data `json:"data"`
}

type Data struct {
	Id    string  `json:"id"`
	Value []Value `json:"data"`
}

type Value struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type DeltaEvent struct {
	Entity         string
	Model          string
	Variant        string
	Version        int
	Id             string
	Payload        map[string]interface{}
	Vectors        []float32
	EventType      string
	TopicPartition kafka.TopicPartition
}
