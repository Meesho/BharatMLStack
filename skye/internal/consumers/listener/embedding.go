package listener

import (
	"encoding/json"

	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/embedding"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

func ProcessEmbeddingEvents(record []kafka.Message, c *kafka.Consumer) error {
	embeddingConsumer := embedding.NewConsumer(embedding.DefaultVersion)
	var events []embedding.Event

	for _, r := range record {
		var event embedding.Event
		err := json.Unmarshal(r.Value, &event)
		if err != nil {
			log.Error().Msgf("Error in JSON deserialization: %s", err)
			continue
		}

		metric.Incr("embedding_consumer_event", []string{"type", "embedding",
			"entity_label", event.Entity,
			"model_name", event.Model,
			"environment", event.Environment})
		events = append(events, event)
	}

	err := embeddingConsumer.Process(events)
	if err != nil {
		log.Error().Msgf("Error in processing Embedding Event %v", err)
		return err
	}

	err = c.Commit(c)
	if err != nil {
		return err
	}
	return nil
}

func ProcessEmbeddingEventsInSequence(record []kafka.Message, c *kafka.Consumer) error {
	embeddingConsumer := embedding.NewConsumer(embedding.DefaultVersion)
	var events []embedding.Event

	for _, r := range record {
		var event embedding.Event
		err := json.Unmarshal(r.Value, &event)
		if err != nil {
			log.Error().Msgf("Error in JSON deserialization: %s", err)
			continue
		}
		events = append(events, event)
	}
	err := embeddingConsumer.ProcessInSequence(events)
	if err != nil {
		log.Error().Msgf("Error in processing Embedding Event %v", err)
		return err
	}
	err = c.Commit(c)
	if err != nil {
		return err
	}
	return nil
}
