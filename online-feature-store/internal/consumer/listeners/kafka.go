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
	"github.com/spf13/viper"
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
	workerPool           chan struct{} // Semaphore for worker pool
}

func NewKafkaListener() *KafkaListener {
	once.Do(func() {
		kafkaConfigGenerator := kafkaConf.NewKafkaConfig()
		kafkaConfig, err := kafkaConfigGenerator.BuildConfigFromEnv(envPrefix)
		if err != nil {
			log.Panic().Err(err).Msg("Failed to build kafka config")
		}
		maxWorkers := 0
		if viper.IsSet("KAFKA_CONSUMERS_FEATURE_CONSUMER_MAX_WORKERS") {
			maxWorkers = viper.GetInt("KAFKA_CONSUMERS_FEATURE_CONSUMER_MAX_WORKERS")
		}
		if maxWorkers == 0 {
			panic("KAFKA_CONSUMERS_FEATURE_CONSUMER_MAX_WORKERS is not set")
		}

		log.Info().
			Int("max_workers", maxWorkers).
			Msg("Initializing worker pool")

		kafkaListener = &KafkaListener{
			handler:              feature.InitPersistHandler(),
			kafkaConfigGenerator: kafkaConfigGenerator,
			kafkaConfig:          kafkaConfig,
			workerPool:           make(chan struct{}, maxWorkers), // Initialize worker pool semaphore
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
			messages := make([][]byte, 0, k.kafkaConfig.BatchSize)
			msgCount := 0
			flushTimer := time.NewTicker(30 * time.Second) // ⏳ Flush every 30 seconds (configurable)

			for run {
				select {
				case <-k.sigChan:
					log.Info().Msgf("Terminating Instance %v", c)

					// Process remaining messages before shutdown
					if msgCount > 0 {
						log.Info().Msgf("Processing remaining %d messages before shutdown", msgCount)
						k.process(messages)
					}

					if err := c.Unsubscribe(); err != nil {
						log.Error().Msg("Error while UnSubscribing Topic")
					}
					if err := c.Close(); err != nil {
						log.Error().Msg("Error while Closing Consumer")
					}
					run = false

				case <-flushTimer.C: // ⏳ Flush on timeout even if batch size is not reached
					if msgCount > 0 {
						log.Info().Msgf("Processing %d messages due to timeout", msgCount)
						k.process(messages)
						msgCount = 0
						messages = messages[:0]
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

						messages = append(messages, e.Value)
						msgCount++

						// Process batch if it reaches batch size
						if msgCount == k.kafkaConfig.BatchSize {
							k.process(messages)
							msgCount = 0
							messages = messages[:0]
						}

					case kafka.Error:
						if e.IsFatal() {
							log.Error().Err(e).Msg("Fatal Kafka error. Shutting down consumer.")

							// Process remaining messages before shutting down due to fatal error
							if msgCount > 0 {
								log.Info().Msgf("Processing remaining %d messages before fatal error", msgCount)
								k.process(messages)
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

func (k *KafkaListener) process(event [][]byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Panic occurred in method %s: %v\n", r, debug.Stack())
		}
	}()
	// Process each message with worker pool
	for _, e := range event {
		e := e // Create new variable for goroutine
		// Acquire worker from pool (blocks if pool is full)
		k.workerPool <- struct{}{}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("Panic occurred in worker: %v\n", r)
					metric.Incr("feature_persist_panic_count", []string{})
				}
				<-k.workerPool // Release worker back to pool
			}()

			value := &persist.Query{}
			err := proto.Unmarshal(e, value)
			if err != nil {
				log.Error().Err(err).Msg("Failed to deserialize FeatureDataEvent")
				return
			}

			_, err = k.handler.Persist(context.Background(), value, enums.ConsistencyEventual)
			if err != nil {
				log.Error().Err(err).Msg("Failed to persist features.")
				metric.Incr("feature_persist", []string{"success", "false", "entity", value.EntityLabel})
			} else {
				metric.Incr("feature_persist", []string{"success", "true", "entity", value.EntityLabel})
			}
		}()
	}
}
