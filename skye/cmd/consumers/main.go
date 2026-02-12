package main

import (
	"strconv"
	"strings"

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
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/mq/consumer"
	"github.com/Meesho/BharatMLStack/skye/pkg/mq/producer"
	"github.com/Meesho/BharatMLStack/skye/pkg/profiling"
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
	consumer.Init()
	producer.Init()
	var stringType string
	var byteType []byte
	configManager := config.NewManager(config.DefaultVersion)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, delta_realtime.NewConsumer(config.DefaultVersion).RefreshRateLimiters)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.QDRANT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.NGT).RefreshClients)
	for _, kafkaId := range strings.Split(appConfig.Configs.EmbeddingConsumerKafkaIds, ",") {
		kafkaIdInt, err := strconv.Atoi(kafkaId)
		if err != nil {
			log.Error().Msgf("Error converting Embedding kafkaId to int: %s", err)
			continue
		}
		consumer.ConsumeBatchWithManualAck(kafkaIdInt, listener.ProcessEmbeddingEvents, stringType, byteType)
	}
	for _, kafkaId := range strings.Split(appConfig.Configs.EmbeddingConsumerSequenceKafkaIds, ",") {
		kafkaIdInt, err := strconv.Atoi(kafkaId)
		if err != nil {
			log.Error().Msgf("Error converting Embedding Sequence kafkaId to int: %s", err)
			continue
		}
		consumer.ConsumeBatchWithManualAck(kafkaIdInt, listener.ProcessEmbeddingEventsInSequence, stringType, byteType)
	}
	for _, kafkaId := range strings.Split(appConfig.Configs.RealtimeConsumerKafkaIds, ",") {
		kafkaIdInt, err := strconv.Atoi(kafkaId)
		if err != nil {
			log.Error().Msgf("Error converting Realtime kafkaId to int: %s", err)
			continue
		}
		consumer.ConsumeBatchWithManualAck(kafkaIdInt, listener.ConsumeRealtimeEvents, stringType, stringType)
	}
	if appConfig.Configs.RealTimeDeltaConsumerKafkaId == 0 {
		log.Error().Msg("RealTimeDeltaConsumerKafkaId is not set")
	}
	if appConfig.Configs.RealTimeDeltaConsumerKafkaId != 0 {
		consumer.ConsumeBatchWithManualAck(appConfig.Configs.RealTimeDeltaConsumerKafkaId, listener.ConsumeRealTimeDeltaEvents, stringType, stringType)
	}
	httpframework.Init()
	api.Init()
	server.InitServer(appConfig.Configs.Port)
}
