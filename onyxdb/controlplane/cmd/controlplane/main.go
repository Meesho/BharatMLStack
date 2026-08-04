package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/reconciler"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/server"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("control plane exited with error")
	}
}

func run() error {
	addr := envOrDefault("ONYXDB_CP_ADDR", ":8080")
	rawEndpoints := envOrDefault("ONYXDB_ETCD_ENDPOINTS", "localhost:2379")
	endpoints := strings.Split(rawEndpoints, ",")

	state, err := etcdstate.NewEtcdStateClient(endpoints)
	if err != nil {
		return err
	}
	defer state.Close()

	srv := server.New(addr, state)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Auto-promote reconciler (ADR-0009, first slice). Disabled by setting the
	// interval to 0; otherwise it ticks and promotes ready, fully-warm versions
	// for stores with dataflow.autoPromote=true.
	if interval := autoPromoteInterval(); interval > 0 {
		go reconciler.New(state, interval).Run(ctx)
	} else {
		log.Info().Msg("auto-promote reconciler disabled (ONYXDB_AUTO_PROMOTE_INTERVAL=0)")
	}

	log.Info().Str("addr", addr).Strs("etcd", endpoints).Msg("OnyxDB control plane starting")
	return srv.Run(ctx)
}

// autoPromoteInterval returns the reconciler tick interval. Defaults to 5s; set
// ONYXDB_AUTO_PROMOTE_INTERVAL to a Go duration ("10s", "1m") to override, or "0"
// to disable the reconciler entirely.
func autoPromoteInterval() time.Duration {
	raw := envOrDefault("ONYXDB_AUTO_PROMOTE_INTERVAL", "5s")
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Warn().Str("value", raw).Msg("invalid ONYXDB_AUTO_PROMOTE_INTERVAL, using 5s")
		return 5 * time.Second
	}
	return d
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
