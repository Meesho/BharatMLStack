//go:build !meesho

package consumer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ProcessClickEvents(record []*kafka.Message, c *kafka.Consumer) error {
	// clickConsumer := click.NewConsumer(click.DefaultVersion)
	// var events []model.ClickEvent

	// for _, r := range record {
	// 	var event model.ClickEvent
	// 	err := json.Unmarshal([]byte(r.Value), &event)
	// 	if err != nil {
	// 		log.Error().Msgf("error in json deserialization: %s", err)
	// 		continue
	// 	}
	// 	metric.Incr("click_consumer_event", []string{"type", "click"})
	// 	events = append(events, event)
	// }

	// err := clickConsumer.Process(events)
	// if err != nil {
	// 	log.Error().Msgf("error in processing click event %v", err)
	// 	return err
	// }

	// err = consumer.Commit(c)
	// if err != nil {
	// 	return err
	// }
	return nil
}
