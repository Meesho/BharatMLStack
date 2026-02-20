package main

import (
	"github.com/Meesho/BharatMLStack/resource-manager/internal/app"
	rmconfig "github.com/Meesho/BharatMLStack/resource-manager/internal/config"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
	pkgetcd "github.com/Meesho/BharatMLStack/resource-manager/pkg/etcd"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/logger"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/metric"
	"github.com/rs/zerolog/log"
)

const shadowDeployablesWatchPath = "/shadow-deployables"

func main() {
	config.InitEnv()
	logger.Init()
	metric.Init()
	if !config.Instance().UseMockAdapters {
		pkgetcd.Init(pkgetcd.DefaultVersion, &rmconfig.EtcdRegistry{})
		rmconfig.InitEtcDBridge()
		configManager := rmconfig.Instance(rmconfig.DefaultVersion)
		if err := configManager.RefreshShadowDeployables(); err != nil {
			log.Error().Err(err).Msg("Error refreshing shadow deployables cache")
		}
		if err := pkgetcd.Instance().RegisterWatchPathCallback(shadowDeployablesWatchPath, configManager.RefreshShadowDeployables); err != nil {
			log.Error().Err(err).Msg("Error registering watch path callback for shadow deployables")
		}
	}

	handler, err := app.BuildHandler()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build api handler")
	}
	server := app.NewServer(config.Instance().AppPort, handler)
	if err := server.Run(); err != nil {
		log.Fatal().Err(err).Msg("api-server exited with error")
	}
}
