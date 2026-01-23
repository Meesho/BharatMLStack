package main

import (
	http2 "net/http"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	repository "github.com/Meesho/BharatMLStack/interaction-store/internal/data"
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
	logger.Init()
	metric.Init()
	profiling.Init()
	repository.Init(appConfig.Configs)
	grpc.Init(appConfig.Configs)
	log.Info().Msgf("Starting interaction-store gRPC server on port :%d", appConfig.Configs.AppPort)
	err := grpc.Instance().Run(appConfig.Configs.AppPort)
	if err != nil {
		log.Panic().Err(err).Msg("Error running interaction-store api-server")
	}
}
