package main

import (
	"net/http"
	_ "net/http/pprof"

	featureConfig "github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/server/grpc"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/logger"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
)

const configManagerVersion = 1
const normalizedEntitiesWatchPath = "/entities"
const registeredClientsWatchPath = "/security/reader"

func main() {
	config.InitEnv()
	go func() {
		http.ListenAndServe(":8080", nil)
	}()
	logger.Init()
	etcd.Init(configManagerVersion, &featureConfig.FeatureRegistry{})
	metric.Init()
	infra.InitDBConnectors()
	grpc.Init()
	system.Init()
	featureConfig.InitEtcDBridge()
	configManager := featureConfig.Instance(featureConfig.DefaultVersion)
	configManager.GetNormalizedEntities()
	err := etcd.Instance().RegisterWatchPathCallback(normalizedEntitiesWatchPath, configManager.GetNormalizedEntities)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for in-memory cache")
	}
	configManager.RegisterClients()
	err = etcd.Instance().RegisterWatchPathCallback(registeredClientsWatchPath, configManager.RegisterClients)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for registered clients")
	}
	provider.InitProvider(configManager, etcd.Instance())
	err = grpc.Instance().Run()
	if err != nil {
		log.Panic().Err(err).Msg("Error from running online-feature-store api-server")
	}
	log.Info().Msgf("online-feature-store api server started.")
}
