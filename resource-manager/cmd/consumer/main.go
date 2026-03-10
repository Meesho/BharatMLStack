package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/app"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/logger"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/metric"
	"github.com/rs/zerolog/log"
)

func main() {
	config.InitEnv()
	logger.Init()
	metric.Init()

	service, err := app.BuildConsumerService()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build consumer service")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := service.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("consumer exited with error")
	}
	<-ctx.Done()
	log.Info().Msg("consumer stopped")
	os.Exit(0)
}
