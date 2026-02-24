package kafka

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kafkaConf "github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

const (
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

// splitAndTrimTopics splits a comma-separated topic list and trims spaces (e.g. "a, b" -> ["a", "b"]).
func splitAndTrimTopics(topicsStr string) []string {
	parts := strings.Split(topicsStr, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// BatchHandler processes a batch of raw Kafka messages.
// Return nil on success (processBatch will commit); return error to trigger seek-back.
type BatchHandler func(msgs []*kafka.Message, c *kafka.Consumer) error

type KafkaListener struct {
	consumers    []*kafka.Consumer
	kafkaConfig  *kafkaConf.KafkaConfig
	sigChan      chan os.Signal // Signal channel
	batchHandler BatchHandler
}

// StartConsumers splits a comma-separated list of kafka IDs, builds a KafkaConfig
// per ID from env prefix KAFKA_<id>, and starts a KafkaListener with the given handler.
func StartConsumers(kafkaIds string, consumerName string, handler BatchHandler) {
	for _, kafkaId := range strings.Split(kafkaIds, ",") {
		kafkaId = strings.TrimSpace(kafkaId)
		if kafkaId == "" {
			continue
		}
		envPrefix := "KAFKA_" + kafkaId
		cfg, err := kafkaConf.NewKafkaConfig().BuildConfigFromEnv(envPrefix)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to build kafka config for %s (kafkaId=%s)", consumerName, kafkaId)
			continue
		}
		log.Info().Str("topic", cfg.Topics).Str("bootstrap", cfg.BootstrapURLs).Str("group", cfg.GroupID).
			Msgf("Starting %s consumer kafkaId=%s (subscribe to topic)", consumerName, kafkaId)
		kl := NewKafkaListener(cfg, handler)
		kl.Init()
		kl.Consume()
		log.Info().Msgf("Started %s consumer for kafkaId=%s", consumerName, kafkaId)
	}
}

func NewKafkaListener(cfg *kafkaConf.KafkaConfig, batchHandler BatchHandler) *KafkaListener {
	return &KafkaListener{
		kafkaConfig:  cfg,
		batchHandler: batchHandler,
	}
}

func (k *KafkaListener) Init() {
	for i := 0; i < k.kafkaConfig.Concurrency; i++ {
		indexString := strconv.Itoa(i)
		configMap := &kafka.ConfigMap{
			bootstrapServers:     k.kafkaConfig.BootstrapURLs,
			groupID:              k.kafkaConfig.GroupID,
			autoOffsetReset:      k.kafkaConfig.AutoOffsetReset,
			reBalanceEnable:      k.kafkaConfig.ReBalanceEnable,
			enableAutoCommit:     k.kafkaConfig.AutoCommitEnable,
			autoCommitIntervalMs: k.kafkaConfig.AutoCommitIntervalInMs,
			clientId:             k.kafkaConfig.ClientID + "-" + indexString,
		}
		if k.kafkaConfig.SecurityProtocol != "" {
			(*configMap)[securityProtocol] = k.kafkaConfig.SecurityProtocol
		}
		if k.kafkaConfig.SaslMechanism != "" {
			(*configMap)[saslMechanism] = k.kafkaConfig.SaslMechanism
		}
		if k.kafkaConfig.SaslUsername != "" {
			(*configMap)[saslUsername] = k.kafkaConfig.SaslUsername
		}
		if k.kafkaConfig.SaslPassword != "" {
			(*configMap)[saslPassword] = k.kafkaConfig.SaslPassword
		}
		consumer, err := kafka.NewConsumer(configMap)
		if err != nil {
			log.Panic().Err(err).Msg("Failed to create Kafka consumer.")
		}
		topics := splitAndTrimTopics(k.kafkaConfig.Topics)
		if len(topics) == 0 {
			topics = []string{strings.TrimSpace(k.kafkaConfig.Topics)}
		}
		err = consumer.SubscribeTopics(topics, nil)
		if err != nil {
			log.Panic().Err(err).Msgf("Failed to subscribe to topics %v", topics)
		}
		k.consumers = append(k.consumers, consumer)
	}
	k.sigChan = make(chan os.Signal, 1)
	signal.Notify(k.sigChan, syscall.SIGINT, syscall.SIGTERM)
}

func (k *KafkaListener) Consume() {
	for i, c := range k.consumers {
		consumer := c
		log.Info().Msgf("Starting Consumption for FeatureDataEvent %v", i)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("%v : Recovered from panic: %v", consumer, r)
					partitions, _ := consumer.Assignment()
					_, err := consumer.SeekPartitions(partitions)
					if err != nil {
						log.Error().Msgf("%v : Failed to seek partitions", consumer)
					}
					metric.Incr("consumer_panic", []string{"group:" + k.kafkaConfig.GroupID, "client:" + k.kafkaConfig.ClientID})
				}
			}()
			run := true

			messages := make([]*kafka.Message, 0, k.kafkaConfig.BatchSize)
			msgCount := 0
			flushTimer := time.NewTicker(30 * time.Second) // ⏳ Flush every 30 seconds (configurable)

			for run {
				select {
				case <-k.sigChan:
					log.Info().Msgf("Terminating Instance %v", consumer)

					if msgCount > 0 {
						log.Debug().Msgf("Processing remaining %d messages before shutdown", msgCount)
						k.processBatch(consumer, messages)
					}

					if err := consumer.Unsubscribe(); err != nil {
						log.Error().Msg("Error while UnSubscribing Topic")
					}
					if err := consumer.Close(); err != nil {
						log.Error().Msg("Error while Closing Consumer")
					}
					run = false

				case <-flushTimer.C:
					if msgCount > 0 {
						log.Info().Int("msgCount", msgCount).Msg("Flushing batch due to timeout")
						k.processBatch(consumer, messages)
						msgCount = 0
						messages = messages[:0]
					}

				default:
					ev := consumer.Poll(k.kafkaConfig.PollTimeout)
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
						log.Info().Str("topic", *e.TopicPartition.Topic).Int32("partition", e.TopicPartition.Partition).Msg("Kafka message received")

						messages = append(messages, e)
						msgCount++

						if msgCount == k.kafkaConfig.BatchSize {
							log.Info().Int("msgCount", msgCount).Msg("Processing batch (batch full)")
							k.processBatch(consumer, messages)
							msgCount = 0
							messages = messages[:0]
						}

					case kafka.Error:
						if e.IsFatal() {
							log.Error().Err(e).Msg("Fatal Kafka error. Shutting down consumer.")

							if msgCount > 0 {
								log.Info().Msgf("Processing remaining %d messages before fatal error", msgCount)
								k.processBatch(consumer, messages)
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

func (k *KafkaListener) processBatch(consumer *kafka.Consumer, messages []*kafka.Message) {
	if len(messages) == 0 {
		return
	}
	err := k.batchHandler(messages, consumer)
	if err != nil {
		log.Error().Err(err).Msg("Batch processing failed, seeking back")
		partitionsMap := make(map[kafka.TopicPartition]kafka.TopicPartition)
		for _, m := range messages {
			partitionsMap[m.TopicPartition] = m.TopicPartition
		}
		topicPartitions := make([]kafka.TopicPartition, 0, len(partitionsMap))
		for _, tp := range partitionsMap {
			topicPartitions = append(topicPartitions, tp)
		}
		if _, seekErr := consumer.SeekPartitions(topicPartitions); seekErr != nil {
			log.Error().Err(seekErr).Msg("Failed to seek partitions")
		}
		return
	}
	if !k.kafkaConfig.AutoCommitEnable {
		if _, err := consumer.Commit(); err != nil {
			log.Error().Err(err).Msg("Failed to commit")
		}
	}
}
