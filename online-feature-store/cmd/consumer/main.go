package main

import (
	http2 "net/http"
	_ "net/http/pprof"

	featureConfig "github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/consumer/listeners"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/server/http"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/logger"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const configManagerVersion = 1
const normalizedEntitiesWatchPath = "/entities"

func main() {
	config.InitEnv()
	go func() {
		http2.ListenAndServe(":8080", nil)
	}()
	logger.Init()
	etcd.Init(configManagerVersion, &featureConfig.FeatureRegistry{})
	metric.Init()
	infra.InitDBConnectors()
	http.Init()
	system.Init()
	featureConfig.InitEtcDBridge()
	configManager := featureConfig.Instance(featureConfig.DefaultVersion)
	configManager.GetNormalizedEntities()
	err := etcd.Instance().RegisterWatchPathCallback(normalizedEntitiesWatchPath, configManager.GetNormalizedEntities)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for in-memory cache")
	}
	provider.InitProvider(configManager, etcd.Instance())
	kafkaListener := listeners.NewKafkaListener()
	kafkaListener.Init()
	kafkaListener.Consume()
	err = http.Instance().Run(":" + viper.GetString("APP_PORT"))
	if err != nil {
		log.Panic().Err(err).Msg("Error from running online-feature-store consumer")
	}
}
