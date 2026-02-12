package main

import (
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
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/mq/consumer"
	"github.com/Meesho/BharatMLStack/skye/pkg/mq/producer"
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
	httpframework.Init()
	router.Init()
	consumer.Init()
	configManager := config.NewManager(config.DefaultVersion)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.QDRANT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.NGT).RefreshClients)
	var stringType string
	consumer.ConsumeWithManualAck(appConfig.Configs.ModelStateConsumer, adminConsumer.ProcessStatesConsumer, stringType, stringType)
	producer.Init()
	go qdrant.NewHandler(qdrant.DefaultVersion).PublishCollectionMetrics()
	server.InitServer(appConfig.Configs.Port)
}
