package kafka

import (
	"fmt"
	"strconv"
	"sync"

	kafkaConf "github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

// ProducerMessage represents a single message to be produced.
type ProducerMessage struct {
	Key       *string
	Value     []byte
	Headers   map[string][]byte
	Partition *int
}

// producerEntry maps a kafkaId to a shared confluent producer and a target topic.
type producerEntry struct {
	producer *kafka.Producer
	topic    string
}

// produce sends messages to the entry's topic via the shared confluent producer.
func (pe *producerEntry) produce(msgs []ProducerMessage) error {
	for _, m := range msgs {
		kafkaMsg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &pe.topic,
				Partition: kafka.PartitionAny,
			},
			Value: m.Value,
		}
		if m.Key != nil {
			kafkaMsg.Key = []byte(*m.Key)
		}
		if m.Partition != nil {
			kafkaMsg.TopicPartition.Partition = int32(*m.Partition)
		}
		if len(m.Headers) > 0 {
			for k, v := range m.Headers {
				kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{Key: k, Value: v})
			}
		}
		if err := pe.producer.Produce(kafkaMsg, nil); err != nil {
			return fmt.Errorf("kafka produce error: %w", err)
		}
	}
	return nil
}

// --------------- Package-level producer registry ---------------
//
// Multiple kafkaIds that share the same broker+auth reuse one confluent
// *kafka.Producer; only the target topic differs.

var (
	entries  = make(map[int]*producerEntry)     // kafkaId → entry
	clusters = make(map[string]*kafka.Producer) // clusterKey → shared producer
	mu       sync.RWMutex
)

// clusterKey derives a dedup key from broker + auth fields.
func clusterKey(cfg *kafkaConf.ProducerConfig) string {
	return cfg.BootstrapURLs + "|" + cfg.SecurityProtocol + "|" + cfg.SaslMechanism + "|" + cfg.SaslUsername
}

// newConfluentProducer creates a confluent producer and starts a background
// goroutine to drain delivery reports.
func newConfluentProducer(cfg *kafkaConf.ProducerConfig) (*kafka.Producer, error) {
	configMap := kafka.ConfigMap{
		"bootstrap.servers": cfg.BootstrapURLs,
		"client.id":         cfg.ClientID,
	}
	if cfg.SecurityProtocol != "" {
		configMap["security.protocol"] = cfg.SecurityProtocol
	}
	if cfg.SaslMechanism != "" {
		configMap["sasl.mechanism"] = cfg.SaslMechanism
	}
	if cfg.SaslUsername != "" {
		configMap["sasl.username"] = cfg.SaslUsername
	}
	if cfg.SaslPassword != "" {
		configMap["sasl.password"] = cfg.SaslPassword
	}

	p, err := kafka.NewProducer(&configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	// Drain delivery reports in background so the producer doesn't block.
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Error().Err(ev.TopicPartition.Error).
						Str("topic", *ev.TopicPartition.Topic).
						Msg("kafka delivery failed")
				}
			}
		}
	}()

	return p, nil
}

// InitProducer builds a ProducerConfig from env prefix KAFKA_PRODUCER_<kafkaId>,
// reuses an existing confluent producer if one already exists for the same
// broker+auth cluster, and registers the kafkaId → topic mapping.
// It is idempotent — calling it again for an already-initialised kafkaId is a no-op.
func InitProducer(kafkaId int) {
	mu.RLock()
	_, exists := entries[kafkaId]
	mu.RUnlock()
	if exists {
		return
	}

	envPrefix := "KAFKA_PRODUCER_" + strconv.Itoa(kafkaId)
	cfg, err := kafkaConf.NewKafkaConfig().BuildProducerConfigFromEnv(envPrefix)
	if err != nil {
		log.Error().Err(err).Int("kafkaId", kafkaId).Msg("failed to build producer config")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Double-check after write lock.
	if _, exists := entries[kafkaId]; exists {
		return
	}

	key := clusterKey(cfg)
	p, ok := clusters[key]
	if !ok {
		p, err = newConfluentProducer(cfg)
		if err != nil {
			log.Error().Err(err).Int("kafkaId", kafkaId).Msg("failed to create kafka producer")
			return
		}
		clusters[key] = p
		log.Info().Int("kafkaId", kafkaId).Str("cluster", key).Msg("created new confluent producer for cluster")
	}

	entries[kafkaId] = &producerEntry{producer: p, topic: cfg.Topics}
	log.Info().Int("kafkaId", kafkaId).Str("topic", cfg.Topics).Msg("kafka producer registered")
}

// SendAndForget looks up the producer entry by kafkaId and produces messages
// to the entry's topic via the shared confluent producer.
func SendAndForget(kafkaId int, msgs []ProducerMessage) error {
	mu.RLock()
	entry, ok := entries[kafkaId]
	mu.RUnlock()
	if !ok {
		return fmt.Errorf("producer not initialised for kafkaId=%d", kafkaId)
	}
	return entry.produce(msgs)
}
