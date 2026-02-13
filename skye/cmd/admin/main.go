package main

import (
	"strconv"

	adminConsumer "github.com/Meesho/BharatMLStack/skye/internal/admin/consumer"
	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/qdrant"
	"github.com/Meesho/BharatMLStack/skye/internal/admin/router"
	"github.com/Meesho/BharatMLStack/skye/internal/bootstrap"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/internal/server"
	"github.com/Meesho/BharatMLStack/skye/pkg/etcd"
	"github.com/Meesho/BharatMLStack/skye/pkg/httpframework"
	skafka "github.com/Meesho/BharatMLStack/skye/pkg/kafka"
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

const (
	EntityWatchPath = "/entity"
)

func main() {
	bootstrap.InitAdmin()
	appConfig := structs.GetAppConfig()
	logger.Init()
	metric.Init()
	etcd.InitFromAppName(&config.Skye{}, appConfig.Configs.AppName, appConfig.Configs)
	// Initialise Kafka producer for model state transitions.
	skafka.InitProducer(appConfig.Configs.ModelStateProducer)

	httpframework.Init()
	router.Init()
	configManager := config.NewManager(config.DefaultVersion)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.QDRANT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.NGT).RefreshClients)

	// Model state consumer — processes each record individually (commit per record)
	if appConfig.Configs.ModelStateConsumer == 0 {
		log.Error().Msg("ModelStateConsumer is not set")
	} else {
		skafka.StartConsumers(strconv.Itoa(appConfig.Configs.ModelStateConsumer), "model-state", func(msgs []*kafka.Message, c *kafka.Consumer) error {
			records := skafka.MessagesToRecordStrings(msgs)
			for _, record := range records {
				if err := adminConsumer.ProcessStatesConsumer(record); err != nil {
					return err
				}
			}
			return nil
		})
	}

	go qdrant.NewHandler(qdrant.DefaultVersion).PublishCollectionMetrics()
	server.InitServer(appConfig.Configs.Port)
}
