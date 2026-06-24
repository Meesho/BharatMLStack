package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/server"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("control plane exited with error")
	}
}

func run() error {
	addr := envOrDefault("MNEMO_CP_ADDR", ":8080")
	rawEndpoints := envOrDefault("MNEMO_ETCD_ENDPOINTS", "localhost:2379")
	endpoints := strings.Split(rawEndpoints, ",")

	state, err := etcdstate.NewEtcdStateClient(endpoints)
	if err != nil {
		return err
	}
	defer state.Close()

	srv := server.New(addr, state)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info().Str("addr", addr).Strs("etcd", endpoints).Msg("mNemo control plane starting")
	return srv.Run(ctx)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
