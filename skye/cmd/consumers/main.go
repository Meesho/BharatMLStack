package main

import (
	"strconv"

	"github.com/Meesho/BharatMLStack/skye/internal/bootstrap"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/api"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/delta_realtime"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/internal/server"
	"github.com/Meesho/BharatMLStack/skye/pkg/etcd"
	"github.com/Meesho/BharatMLStack/skye/pkg/httpframework"
	skafka "github.com/Meesho/BharatMLStack/skye/pkg/kafka"
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/profiling"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

const (
	EntityWatchPath         = "/entity"
	ConsumerConfigWatchPath = "/consumer-config"
)

func main() {
	bootstrap.InitConsumers()
	appConfig := structs.GetAppConfig()
	logger.Init()
	metric.Init()
	etcd.InitFromAppName(&config.Skye{}, appConfig.Configs.AppName, appConfig.Configs)
	profiling.Init()

	// Initialise Kafka producers (static IDs known at startup).
	// Failure producers (per-model dynamic IDs) are initialised lazily in embedding handler.
	skafka.InitProducer(appConfig.Configs.RealtimeProducerKafkaId)
	skafka.InitProducer(appConfig.Configs.RealTimeDeltaProducerKafkaId)

	configManager := config.NewManager(config.DefaultVersion)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, delta_realtime.NewConsumer(config.DefaultVersion).RefreshRateLimiters)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.QDRANT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.NGT).RefreshClients)

	// Embedding batch consumers
	skafka.StartConsumers(appConfig.Configs.EmbeddingConsumerKafkaIds, "embedding", func(msgs []*kafka.Message, c *kafka.Consumer) error {
		records := skafka.MessagesToRecordBytes(msgs)
		return listener.ProcessEmbeddingEvents(records, c)
	})

	// Embedding sequence consumers
	skafka.StartConsumers(appConfig.Configs.EmbeddingConsumerSequenceKafkaIds, "embedding-sequence", func(msgs []*kafka.Message, c *kafka.Consumer) error {
		records := skafka.MessagesToRecordBytes(msgs)
		return listener.ProcessEmbeddingEventsInSequence(records, c)
	})

	// Realtime consumers
	skafka.StartConsumers(appConfig.Configs.RealtimeConsumerKafkaIds, "realtime", func(msgs []*kafka.Message, c *kafka.Consumer) error {
		records := skafka.MessagesToRecordStrings(msgs)
		return listener.ConsumeRealtimeEvents(records, c)
	})

	// Realtime delta consumer (single ID)
	if appConfig.Configs.RealTimeDeltaConsumerKafkaId == 0 {
		log.Error().Msg("RealTimeDeltaConsumerKafkaId is not set")
	} else {
		skafka.StartConsumers(strconv.Itoa(appConfig.Configs.RealTimeDeltaConsumerKafkaId), "realtime-delta", func(msgs []*kafka.Message, c *kafka.Consumer) error {
			records := skafka.MessagesToRecordStrings(msgs)
			return listener.ConsumeRealTimeDeltaEvents(records, c)
		})
	}

	httpframework.Init()
	api.Init()
	server.InitServer(appConfig.Configs.Port)
}
