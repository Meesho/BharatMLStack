package main

import (
	http2 "net/http"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/scylla"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/server/grpc"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/logger"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/profiling"
	"github.com/rs/zerolog/log"
)

type AppConfig struct {
	Configs        config.Configs
	DynamicConfigs config.DynamicConfigs
}

func (cfg *AppConfig) GetStaticConfig() interface{} {
	return &cfg.Configs
}

func (cfg *AppConfig) GetDynamicConfig() interface{} {
	return &cfg.DynamicConfigs
}

var (
	appConfig AppConfig
)

func main() {
	config.InitConfig(&appConfig)
	go func() {
		http2.ListenAndServe(":8080", nil)
	}()
	scylla.Init(appConfig.Configs)
	logger.Init()
	metric.Init()
	profiling.Init()
	grpc.Init(appConfig.Configs)
	err := grpc.Instance().Run(appConfig.Configs.AppPort)
	if err != nil {
		log.Panic().Err(err).Msg("Error from running interaction-store api-server")
	}
	log.Info().Msgf("interaction-store gRPC API server started.")
}
