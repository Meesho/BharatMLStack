package listeners

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config/enums"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"google.golang.org/protobuf/proto"

	kafkaConf "github.com/Meesho/BharatMLStack/online-feature-store/internal/consumer/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

const (
	envPrefix            = "KAFKA_CONSUMERS_FEATURE_CONSUMER"
	bootstrapServers     = "bootstrap.servers"
	groupID              = "group.id"
	autoOffsetReset      = "auto.offset.reset"
	reBalanceEnable      = "go.application.rebalance.enable"
	enableAutoCommit     = "enable.auto.commit"
	autoCommitIntervalMs = "auto.commit.interval.ms"
	saslUsername         = "sasl.username"
	saslPassword         = "sasl.password"
	saslMechanism        = "sasl.mechanisms"
	securityProtocol     = "security.protocol"
	clientId             = "client.id"
)

var (
	once          sync.Once
	kafkaListener *KafkaListener
)

type KafkaListener struct {
	handler              *feature.PersistHandler
	kafkaConfigGenerator kafkaConf.KafkaConfigGenerator
	consumers            []*kafka.Consumer
	kafkaConfig          *kafkaConf.KafkaConfig
	sigChan              chan os.Signal
}

func NewKafkaListener() *KafkaListener {
	once.Do(func() {
		kafkaConfigGenerator := kafkaConf.NewKafkaConfig()
		kafkaConfig, err := kafkaConfigGenerator.BuildConfigFromEnv(envPrefix)
		if err != nil {
			log.Panic().Err(err).Msg("Failed to build kafka config")
		}

		kafkaListener = &KafkaListener{
			handler:              feature.InitPersistHandler(),
			kafkaConfigGenerator: kafkaConfigGenerator,
			kafkaConfig:          kafkaConfig,
		}
	})
	return kafkaListener
}

func (k *KafkaListener) Init() {
	for i := 0; i < k.kafkaConfig.Concurrency; i++ {
		indexString := strconv.Itoa(i)
		consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
			bootstrapServers:     k.kafkaConfig.BootstrapURLs,
			groupID:              k.kafkaConfig.GroupID,
			autoOffsetReset:      k.kafkaConfig.AutoOffsetReset,
			reBalanceEnable:      k.kafkaConfig.ReBalanceEnable,
			enableAutoCommit:     k.kafkaConfig.AutoCommitEnable,
			autoCommitIntervalMs: k.kafkaConfig.AutoCommitIntervalInMs,
			saslUsername:         k.kafkaConfig.SaslUsername,
			saslPassword:         k.kafkaConfig.SaslPassword,
			securityProtocol:     k.kafkaConfig.SecurityProtocol,
			saslMechanism:        k.kafkaConfig.SaslMechanism,
			clientId:             k.kafkaConfig.ClientID + "-" + indexString,
		})
		if err != nil {
			log.Panic().Err(err).Msg("Failed to create Kafka consumer.")
		}
		err = consumer.SubscribeTopics([]string{k.kafkaConfig.Topic}, nil)
		if err != nil {
			log.Panic().Err(err).Msgf("Failed to subscribe to topic %s", k.kafkaConfig.Topic)
		}
		k.consumers = append(k.consumers, consumer)
	}
	k.sigChan = make(chan os.Signal, 1)
	signal.Notify(k.sigChan, syscall.SIGINT, syscall.SIGTERM)
}

func (k *KafkaListener) Consume() {
	for i, c := range k.consumers {
		log.Info().Msgf("Starting Consumption for FeatureDataEvent %v", i)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("%v : Recovered from panic: %v", c, r)
					partitions, _ := c.Assignment()
					_, err := c.SeekPartitions(partitions)
					if err != nil {
						log.Error().Msgf("%v : Failed to seek partitions", c)
					}
					metric.Incr("consumer_panic", []string{"group:" + k.kafkaConfig.GroupID, "client:" + k.kafkaConfig.ClientID})
				}
			}()
			run := true

			// Partition-wise message storage
			partitionMessages := make(map[int32][]*kafka.Message)
			partitionCounts := make(map[int32]int)
			flushTimer := time.NewTicker(30 * time.Second) // ⏳ Flush every 30 seconds (configurable)

			for run {
				select {
				case <-k.sigChan:
					log.Info().Msgf("Terminating Instance %v", c)

					// Process remaining messages from all partitions before shutdown
					for partition, messages := range partitionMessages {
						if len(messages) > 0 {
							log.Info().Msgf("Processing remaining %d messages from partition %d before shutdown", len(messages), partition)
							k.process(c, messages)
						}
					}

					if err := c.Unsubscribe(); err != nil {
						log.Error().Msg("Error while UnSubscribing Topic")
					}
					if err := c.Close(); err != nil {
						log.Error().Msg("Error while Closing Consumer")
					}
					run = false

				case <-flushTimer.C: // ⏳ Flush on timeout even if batch size is not reached
					for partition, messages := range partitionMessages {
						if len(messages) > 0 {
							log.Info().Msgf("Processing %d messages from partition %d due to timeout", len(messages), partition)
							k.process(c, messages)
							partitionMessages[partition] = partitionMessages[partition][:0] // Clear the slice
							partitionCounts[partition] = 0
						}
					}

				default:
					ev := c.Poll(k.kafkaConfig.PollTimeout)
					if ev == nil {
						continue
					}
					switch e := ev.(type) {
					case *kafka.Message:
						metric.Incr("events_consumed", []string{
							"topic:" + *e.TopicPartition.Topic,
							"group:" + k.kafkaConfig.GroupID,
							"client:" + k.kafkaConfig.ClientID,
						})

						partition := e.TopicPartition.Partition
						if _, exists := partitionMessages[partition]; !exists {
							partitionMessages[partition] = make([]*kafka.Message, 0, k.kafkaConfig.BatchSize)
							partitionCounts[partition] = 0
						}
						// Add message to partition-specific list
						partitionMessages[partition] = append(partitionMessages[partition], e)
						partitionCounts[partition]++

						// Process batch if this partition reaches batch size
						if partitionCounts[partition] == k.kafkaConfig.BatchSize {
							log.Info().Msgf("Processing batch of %d messages from partition %d", partitionCounts[partition], partition)
							k.process(c, partitionMessages[partition])
							partitionMessages[partition] = partitionMessages[partition][:0] // Clear the slice
							partitionCounts[partition] = 0
						}

					case kafka.Error:
						if e.IsFatal() {
							log.Error().Err(e).Msg("Fatal Kafka error. Shutting down consumer.")

							// Process remaining messages from all partitions before shutting down due to fatal error
							for partition, messages := range partitionMessages {
								if len(messages) > 0 {
									log.Info().Msgf("Processing remaining %d messages from partition %d before fatal error", len(messages), partition)
									k.process(c, messages)
								}
							}

							run = false
						} else {
							log.Error().Err(e).Msg("Non-fatal Kafka error encountered.")
						}

					default:
						log.Debug().Msgf("Ignored event: %#v", e)
					}
				}
			}
		}()
	}
}

func (k *KafkaListener) process(consumer *kafka.Consumer, messages []*kafka.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Panic occurred in method %s: %v\n", r, debug.Stack())
		}
	}()
	startOffset := messages[0].TopicPartition.Offset
	topic := messages[0].TopicPartition.Topic
	partition := messages[0].TopicPartition.Partition
	isFailed := false
	// Process each message
	for _, msg := range messages {
		value := &persist.Query{}
		err := proto.Unmarshal(msg.Value, value)
		if err != nil {
			log.Error().Err(err).Msg("Failed to deserialize FeatureDataEvent")
			continue
		}
		_, err = k.handler.Persist(context.Background(), value, enums.ConsistencyEventual)
		if err != nil {
			if _, ok := err.(*feature.InvalidEventError); !ok {
				isFailed = true
			}
			log.Error().Err(err).Msg("Failed to persist features.")
			k.publishMetrics(value, value.EntityLabel, false, value)
		} else {
			k.publishMetrics(value, value.EntityLabel, true, value)
		}
	}

	if !k.kafkaConfig.AutoCommitEnable {
		if !isFailed {
			if _, err := consumer.Commit(); err != nil {
				log.Error().Err(err).Msg("Failed to commit messages")
			}
		} else {
			// Seek back to the start of the failed batch
			seekPartitions := []kafka.TopicPartition{
				{
					Topic:     topic,
					Partition: partition,
					Offset:    kafka.Offset(startOffset),
				},
			}
			if _, err := consumer.SeekPartitions(seekPartitions); err != nil {
				log.Error().Msgf("%v : Failed to seek partitions", consumer)
			}
		}
	}
}

func (k *KafkaListener) publishMetrics(persistQuery *persist.Query, entityLabel string, success bool, value *persist.Query) {
	dataSize := int64(len(persistQuery.Data))
	successStr := "false"
	if success {
		successStr = "true"
	}
	baseTags := []string{"success:" + successStr, "entity:" + entityLabel}

	// Publish metric for each feature group
	for _, fg := range value.FeatureGroupSchema {
		metric.Count("feature_persist", dataSize, append(baseTags, "feature_group:"+fg.Label))
	}
}
