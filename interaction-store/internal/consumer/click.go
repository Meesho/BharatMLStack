//go:build !meesho

package consumer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ProcessClickEvents(record []*kafka.Message, c *kafka.Consumer) error {
	return nil
}
