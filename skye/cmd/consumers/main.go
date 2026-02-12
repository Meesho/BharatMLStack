package main

import (
	"github.com/Meesho/BharatMLStack/skye/internal/bootstrap"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/api"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/delta_realtime"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/internal/server"
	"github.com/Meesho/BharatMLStack/skye/pkg/etcd"
	"github.com/Meesho/BharatMLStack/skye/pkg/httpframework"
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/profiling"
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
	configManager := config.NewManager(config.DefaultVersion)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, delta_realtime.NewConsumer(config.DefaultVersion).RefreshRateLimiters)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.QDRANT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.NGT).RefreshClients)
	httpframework.Init()
	api.Init()
	server.InitServer(appConfig.Configs.Port)
}
