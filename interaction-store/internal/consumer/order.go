//go:build meesho

package consumer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ProcessOrderEvents(record []*kafka.Message, c *kafka.Consumer) error {
	// orderConsumer := order.NewConsumer(order.DefaultVersion)
	// var events []model.OrderPlacedEvent

	// for _, r := range record {
	// 	var event model.OrderPlacedEvent
	// 	err := json.Unmarshal([]byte(r.Value), &event)
	// 	if err != nil {
	// 		log.Error().Msgf("error in json deserialization: %s", err)
	// 		continue
	// 	}
	// 	metric.Incr("order_consumer_event", []string{"type", "order"})
	// 	events = append(events, event)
	// }

	// err := orderConsumer.Process(events)
	// if err != nil {
	// 	log.Error().Msgf("error in processing order event %v", err)
	// 	return err
	// }

	// err = consumer.Commit(c)
	// if err != nil {
	// 	return err
	// }
	return nil
}
