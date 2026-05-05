package delta_realtime

import (
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/realtime"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Consumer interface {
	Process(events []realtime.DeltaEvent, c *kafka.Consumer) error
	RefreshRateLimiters(key, value, eventType string) error
}
