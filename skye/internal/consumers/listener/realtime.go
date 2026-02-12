package listener

import (
	"encoding/json"

	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/realtime"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

func ConsumeRealtimeEvents(records []kafka.Message, c *kafka.Consumer) error {
	rtConsumer := realtime.NewConsumer(realtime.DefaultVersion)
	events := make([]realtime.Event, 0)
	for _, r := range records {
		log.Info().Msgf("Received message: %s and %s", r.Key, r.Value)
		var event realtime.Event
		err := json.Unmarshal([]byte(r.Value), &event)
		if err != nil {
			log.Error().Msgf("Error in Unmarshalling %s", err)
			continue
		}
		metric.Incr("realtime_consumer_event", []string{"type", event.Type})
		events = append(events, event)
	}
	err := rtConsumer.Process(events)
	if err != nil {
		log.Error().Msgf("Error in processing Realtime Event %v", err)
		return err
	}

	err = c.Commit(c)
	if err != nil {
		return err
	}
	return nil
}
