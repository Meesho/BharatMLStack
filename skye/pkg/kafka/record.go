package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ConsumerRecord holds a single consumed message with typed key and value.
type ConsumerRecord[K any, V any] struct {
	Key            K
	Value          V
	TopicPartition kafka.TopicPartition
}

// MessagesToRecordBytes converts raw Kafka messages to []ConsumerRecord[string, []byte].
// Use for consumers that need byte-valued records (e.g. embedding).
func MessagesToRecordBytes(msgs []*kafka.Message) []ConsumerRecord[string, []byte] {
	out := make([]ConsumerRecord[string, []byte], 0, len(msgs))
	for _, m := range msgs {
		key := ""
		if m.Key != nil {
			key = string(m.Key)
		}
		val := m.Value
		if val == nil {
			val = []byte{}
		}
		out = append(out, ConsumerRecord[string, []byte]{
			Key:            key,
			Value:          val,
			TopicPartition: m.TopicPartition,
		})
	}
	return out
}

// MessagesToRecordStrings converts raw Kafka messages to []ConsumerRecord[string, string].
// Use for consumers that need string-valued records (e.g. realtime, delta).
func MessagesToRecordStrings(msgs []*kafka.Message) []ConsumerRecord[string, string] {
	out := make([]ConsumerRecord[string, string], 0, len(msgs))
	for _, m := range msgs {
		key := ""
		if m.Key != nil {
			key = string(m.Key)
		}
		val := ""
		if m.Value != nil {
			val = string(m.Value)
		}
		out = append(out, ConsumerRecord[string, string]{
			Key:            key,
			Value:          val,
			TopicPartition: m.TopicPartition,
		})
	}
	return out
}
