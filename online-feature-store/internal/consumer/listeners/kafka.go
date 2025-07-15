package listeners

import (
	"context"
	"errors"
	"hash/fnv"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	handler                       *feature.PersistHandler
	kafkaConfigGenerator          kafkaConf.KafkaConfigGenerator
	consumers                     []*kafka.Consumer
	kafkaConfig                   *kafkaConf.KafkaConfig
	sigChan                       chan os.Signal
	workerChannels                []chan *persist.Query // Worker channels
	resultChannels                []chan bool           // Result channels
	maxWorkers                    int
	enableEntityIdBasedProcessing bool
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

		enableEntityIdBasedProcessing := false
		if viper.IsSet("KAFKA_CONSUMERS_FEATURE_CONSUMER_ENABLE_ENTITY_ID_BASED_PROCESSING") {
			enableEntityIdBasedProcessing = viper.GetBool("KAFKA_CONSUMERS_FEATURE_CONSUMER_ENABLE_ENTITY_ID_BASED_PROCESSING")
		}

		workerChannels := make([]chan *persist.Query, maxWorkers)
		resultChannels := make([]chan bool, maxWorkers)
		for i := 0; i < maxWorkers; i++ {
			workerChannels[i] = make(chan *persist.Query, 1)
			resultChannels[i] = make(chan bool, 1)
		}

		kafkaListener = &KafkaListener{
			handler:                       feature.InitPersistHandler(),
			kafkaConfigGenerator:          kafkaConfigGenerator,
			kafkaConfig:                   kafkaConfig,
			workerChannels:                workerChannels,
			resultChannels:                resultChannels,
			maxWorkers:                    maxWorkers,
			enableEntityIdBasedProcessing: enableEntityIdBasedProcessing,
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

	// Start worker goroutines
	k.startWorkers()
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

			messages := make([]*kafka.Message, 0, k.kafkaConfig.BatchSize)
			msgCount := 0
			flushTimer := time.NewTicker(30 * time.Second) // ⏳ Flush every 30 seconds (configurable)

			for run {
				select {
				case <-k.sigChan:
					log.Info().Msgf("Terminating Instance %v", c)

					if msgCount > 0 {
						log.Debug().Msgf("Processing remaining %d messages before shutdown", msgCount)
						k.processBatch(c, messages)
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
						log.Debug().Msgf("Processing %d messages due to timeout", msgCount)
						k.processBatch(c, messages)
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

						messages = append(messages, e)
						msgCount++

						if msgCount == k.kafkaConfig.BatchSize {
							log.Debug().Msgf("Processing batch of %d messages", msgCount)
							k.processBatch(c, messages)
							msgCount = 0
							messages = messages[:0]
						}

					case *kafka.AssignedPartitions:
						log.Debug().Msgf("Partitions assigned: %v", e.Partitions)
						// Commit the assignment
						if err := c.Assign(e.Partitions); err != nil {
							log.Error().Err(err).Msg("Failed to assign partitions")
						}

					case *kafka.RevokedPartitions:
						log.Debug().Msgf("Partitions revoked: %v", e.Partitions)

						// Process any remaining messages before partitions are revoked
						if msgCount > 0 {
							log.Debug().Msgf("Processing remaining %d messages before revocation", msgCount)
							k.processBatch(c, messages)
							msgCount = 0
							messages = messages[:0]
						}

						// Commit the revocation
						if err := c.Unassign(); err != nil {
							log.Error().Err(err).Msg("Failed to unassign partitions")
						}

					case kafka.Error:
						if e.IsFatal() {
							log.Error().Err(e).Msg("Fatal Kafka error. Shutting down consumer.")

							if msgCount > 0 {
								log.Info().Msgf("Processing remaining %d messages before fatal error", msgCount)
								k.processBatch(c, messages)
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

	partitionsMap := make(map[int32]kafka.TopicPartition)
	allSuccess := true

	for _, msg := range messages {
		partition, exists := partitionsMap[msg.TopicPartition.Partition]
		if !exists || msg.TopicPartition.Offset < partition.Offset {
			partitionsMap[msg.TopicPartition.Partition] = msg.TopicPartition
		}
		value := &persist.Query{}
		err := proto.Unmarshal(msg.Value, value)
		if err != nil {
			log.Error().Err(err).Msg("Failed to deserialize FeatureDataEvent")
			// Treat deserialization error as success, so don't change allSuccess
			continue
		}
		workerId := k.getWorkerId(value, k.enableEntityIdBasedProcessing)

		// Send message to worker
		k.workerChannels[workerId] <- value
		success := <-k.resultChannels[workerId]
		if !success {
			allSuccess = false
		}
	}

	// Check if all messages were processed successfully
	if !k.kafkaConfig.AutoCommitEnable {
		if allSuccess {
			if _, err := consumer.Commit(); err != nil {
				log.Error().Err(err).Msg("Failed to commit messages")
			}
		} else {
			topicPartitions := make([]kafka.TopicPartition, 0, len(messages))
			for _, value := range partitionsMap {
				topicPartitions = append(topicPartitions, value)
			}

			if _, err := consumer.SeekPartitions(topicPartitions); err != nil {
				log.Error().Msgf("%v : Failed to seek partitions", consumer)
			}
		}
	}
}

func (k *KafkaListener) process(value *persist.Query) bool {
	_, err := k.handler.Persist(context.Background(), value, enums.ConsistencyEventual)
	if err != nil {
		var invalidEventErr *feature.InvalidEventError
		if errors.As(err, &invalidEventErr) {
			log.Error().Err(err).Msg("Failed to persist features.")
			k.publishMetrics(value, value.EntityLabel, false, value)
			return true
		}
		k.publishMetrics(value, value.EntityLabel, false, value)
		return false
	} else {
		k.publishMetrics(value, value.EntityLabel, true, value)
		return true
	}
}

func (k *KafkaListener) startWorkers() {
	for i := 0; i < k.maxWorkers; i++ {
		workerID := i
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("Worker %d recovered from panic: %v", workerID, r)
					metric.Incr("worker_panic", []string{"worker_id:" + strconv.Itoa(workerID), "group:" + k.kafkaConfig.GroupID})
				}
			}()

			for msg := range k.workerChannels[workerID] {
				success := k.process(msg)
				k.resultChannels[workerID] <- success
			}
		}()
	}
}

func (k *KafkaListener) getWorkerId(value *persist.Query, enableEntityIdBasedProcessing bool) int {
	if !enableEntityIdBasedProcessing {
		return rand.Intn(k.maxWorkers)
	}
	if len(value.Data) > 1 {
		log.Error().Msg("For entityIdBasedProcessing there can't be >1 entity ids, using random index")
		return rand.Intn(k.maxWorkers)
	}

	var entityId strings.Builder
	for i, keyValue := range value.Data[0].KeyValues {
		if i > 0 {
			entityId.WriteString("_") // separator between key values
		}
		entityId.WriteString(keyValue)
	}
	entityIdStr := entityId.String()

	// Hash calculation using FNV-1a
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(entityIdStr))
	return int(hash.Sum32() % uint32(k.maxWorkers))
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
