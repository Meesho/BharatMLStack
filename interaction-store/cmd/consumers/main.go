package main

import (
	http2 "net/http"

	kafka "github.com/Meesho/BharatMLStack/interaction-store/internal"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	click "github.com/Meesho/BharatMLStack/interaction-store/internal/consumer/click"
	order "github.com/Meesho/BharatMLStack/interaction-store/internal/consumer/order"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/scylla"
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
	scylla.Init(appConfig.Configs)
	logger.Init()
	metric.Init()
	profiling.Init()
	click.Init(appConfig.Configs)
	order.Init(appConfig.Configs)
	kafka.Init(appConfig.Configs)
	http.Init(appConfig.Configs)
	// Initialize and run mux server
	mux, err := muxserver.Init(appConfig.Configs)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to initialize mux server")
	}
	mux.RegisterServices()
	if err := mux.Run(); err != nil {
		log.Panic().Err(err).Msg("Error running interaction-store mux server")
	}
	log.Info().Msgf("interaction-store timeseries consumer started.")
}
