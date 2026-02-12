package listener

import (
	"encoding/json"

	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/delta_realtime"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/realtime"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

func ConsumeRealTimeDeltaEvents(records []kafka.Message, c *kafka.Consumer) error {
	deltaRtConsumer := delta_realtime.NewConsumer(delta_realtime.DefaultVersion)
	events := make([]realtime.DeltaEvent, 0)
	for _, r := range records {
		log.Info().Msgf("Received message: %s and %s", r.Key, r.Value)
		var event realtime.DeltaEvent
		err := json.Unmarshal([]byte(r.Value), &event)
		if err != nil {
			log.Error().Msgf("Error in Unmarshalling %s", err)
			continue
		}
		event.TopicPartition = r.TopicPartition
		metric.Incr("realtime_delta_consumer_event", []string{"type", event.EventType, "entity", event.Entity, "model", event.Model, "variant", event.Variant})
		events = append(events, event)
	}
	err := deltaRtConsumer.Process(events, c)
	if err != nil {
		log.Error().Msgf("Error in processing RealTimeDelta Event %v", err)
		return err
	}
	return nil
}
