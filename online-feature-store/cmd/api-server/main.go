package main

import (
	"net/http"
	_ "net/http/pprof"

	featureConfig "github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/server/grpc"
	httpserver "github.com/Meesho/BharatMLStack/online-feature-store/internal/server/http"
	muxserver "github.com/Meesho/BharatMLStack/online-feature-store/internal/server/mux"
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
const registeredClientsWatchPath = "/security/reader"
const cbWatchPath = "/circuitbreaker"

func main() {
	config.InitEnv()
	go func() {
		http.ListenAndServe(":8080", nil)
	}()
	logger.Init()
	etcd.Init(configManagerVersion, &featureConfig.FeatureRegistry{})
	metric.Init()
	infra.InitDBConnectors()
	system.Init()
	featureConfig.InitEtcDBridge()
	configManager := featureConfig.Instance(featureConfig.DefaultVersion)
	configManager.UpdateCBConfigs()
	err := etcd.Instance().RegisterWatchPathCallback(cbWatchPath, configManager.UpdateCBConfigs)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for circuit breaker")
	}
	configManager.GetNormalizedEntities()
	err = etcd.Instance().RegisterWatchPathCallback(normalizedEntitiesWatchPath, configManager.GetNormalizedEntities)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for in-memory cache")
	}
	configManager.RegisterClients()
	err = etcd.Instance().RegisterWatchPathCallback(registeredClientsWatchPath, configManager.RegisterClients)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for registered clients")
	}
	provider.InitProvider(configManager, etcd.Instance())

	// Check if HTTP API should be enabled
	enableHTTP := viper.GetBool("ENABLE_HTTP_API")
	grpc.Init()
	if enableHTTP {
		log.Info().Msg("HTTP API mode enabled - starting HTTP and gRPC servers via cmux")

		// Initialize HTTP server along with gRPC server
		httpserver.Init()

		// Initialize and run mux server
		mux, err := muxserver.Init()
		if err != nil {
			log.Panic().Err(err).Msg("Failed to initialize mux server")
		}

		mux.RegisterServices()

		if err := mux.Run(); err != nil {
			log.Panic().Err(err).Msg("Error from running mux server")
		}
	} else {
		log.Info().Msg("gRPC API mode enabled (default)")
		err = grpc.Instance().Run()
		if err != nil {
			log.Panic().Err(err).Msg("Error from running online-feature-store api-server")
		}
		log.Info().Msgf("online-feature-store gRPC API server started.")
	}
}
