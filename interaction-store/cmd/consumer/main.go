package main

import (
	http2 "net/http"

	kafka "github.com/Meesho/BharatMLStack/interaction-store/internal"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	clickConsumer "github.com/Meesho/BharatMLStack/interaction-store/internal/consumer/click"
	orderConsumer "github.com/Meesho/BharatMLStack/interaction-store/internal/consumer/order"
	repository "github.com/Meesho/BharatMLStack/interaction-store/internal/data"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/server/grpc"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/server/http"
	muxserver "github.com/Meesho/BharatMLStack/interaction-store/internal/server/mux"
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
	clickConsumer.Init(appConfig.Configs)
	orderConsumer.Init(appConfig.Configs)
	kafka.Listen(appConfig.Configs)
	log.Info().Msgf("Interaction-store timeseries consumer started.")

	// Initiate HTTP, GRPC and Mux server
	http.Init(appConfig.Configs)
	grpc.Init(appConfig.Configs)
	mux, err := muxserver.Init(appConfig.Configs)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to initialize mux server")
	}
	mux.RegisterServices()
	log.Info().Msgf("Starting interaction-store http server on port :%d", appConfig.Configs.AppPort)
	if err := mux.Run(); err != nil {
		log.Panic().Err(err).Msg("Error running interaction-store mux server")
	}
}
