package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Meesho/go-core/config"
	"github.com/Meesho/go-core/grpc"
	"github.com/Meesho/go-core/metric"
	pb "github.com/Meesho/skye-eigenix/internal/client"
	"github.com/Meesho/skye-eigenix/internal/hnswlib"
	"github.com/Meesho/skye-eigenix/internal/qdrant"
	"github.com/Meesho/skye-eigenix/internal/serving"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	// Load environment variables from .env file
	config.InitEnv()
	metric.Init()
	os.Setenv("APP_NAME", "eigenix")
	os.Setenv("APP_PORT", "8080")
	initHNSWLib()
	initQdrant()
	grpc.Init()
	// Register gRPC service
	srv := serving.Init()
	pb.RegisterEigenixServiceServer(grpc.Instance().GRPCServer, srv)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Start gRPC server in a goroutine so it doesn't block
	// Recover from any panics in this goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("gRPC server goroutine panicked")
			}
		}()
		// Errors from Run() during shutdown are expected
		_ = grpc.Instance().Run()
	}()
	log.Info().Msg("Server started successfully")

	// Wait for termination signal
	sig := <-sigChan
	log.Info().Msgf("Received signal: %v. Shutting down gracefully...", sig)
	log.Info().Msg("Initiating shutdown...")
}

func initHNSWLib() {
	log.Info().Msg("Initializing HNSW library...")
	db := hnswlib.Init()

	// Load existing indices on startup
	log.Info().Msg("Loading existing indices...")
	db.LoadAllIndices()

	// Ensure indices are saved on exit, even if there's a panic
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("Recovered from panic during shutdown")
		}
		log.Info().Msg("Saving all indices before exit...")
		db.SaveAllIndices()
		log.Info().Msg("All indices saved successfully")
	}()
}

func initQdrant() {
	log.Info().Msg("Initializing Qdrant...")
	db := qdrant.Init()
	db.LoadCentroids(viper.GetString("QDRANT_CENTROID_FILE_PATH"))
}
