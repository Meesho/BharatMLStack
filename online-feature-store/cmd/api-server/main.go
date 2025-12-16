package main

import (
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"

	featureConfig "github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/server/grpc"
	httpserver "github.com/Meesho/BharatMLStack/online-feature-store/internal/server/http"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/logger"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/p2p"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/rs/zerolog/log"
	"github.com/soheilhy/cmux"
	"github.com/spf13/viper"
	"google.golang.org/grpc/reflection"
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

	// HTTP API check
	enableHTTPStr := strings.ToLower(strings.TrimSpace(viper.GetString("ENABLE_HTTP_API")))
	enableHTTP := enableHTTPStr == "true" || enableHTTPStr == "1"

	if enableHTTP {
		log.Info().Msg("HTTP API mode enabled - starting HTTP and gRPC servers via cmux")

		// Initialize both servers
		grpc.Init()
		httpserver.Init()

		if !viper.IsSet("APP_PORT") {
			log.Panic().Msgf("Failed to start the application - APP_PORT is not set")
		}
		port := viper.GetInt("APP_PORT")

		listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			log.Panic().Msgf("Failed to start the application - Failed to listen: %v", err)
		}

		// cmux for both gRPC & HTTP on same port
		mux := cmux.New(listener)

		// cmux listener for HTTP connections
		httpListener := mux.Match(cmux.HTTP1Fast())
		// cmux listener for gRPC connections
		grpcListener := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"), cmux.Any())

		// register gRPC services
		grpcServer := grpc.Instance().GRPCServer
		retrieve.RegisterFeatureServiceServer(grpcServer, feature.InitV1())
		persist.RegisterFeatureServiceServer(grpcServer, feature.InitPersistHandler())
		p2p.RegisterP2PCacheServiceServer(grpcServer, feature.InitP2PCacheHandler())
		reflection.Register(grpcServer)

		// start HTTP server in a separate goroutine
		go func() {
			if err := http.Serve(httpListener, httpserver.Instance()); err != nil {
				log.Panic().Msgf("Failed to serve HTTP server: %v", err)
			}
		}()

		// start gRPC server in a separate goroutine
		go func() {
			if err := grpcServer.Serve(grpcListener); err != nil {
				log.Panic().Msgf("Failed to serve gRPC server: %v", err)
			}
		}()

		log.Info().Int("port", port).Msg("HTTP and gRPC servers started via cmux")
		if err := mux.Serve(); err != nil {
			log.Panic().Err(err).Msg("Error from running cmux server")
		}
	} else {
		log.Info().Msg("gRPC API mode enabled (default)")
		grpc.Init()
		err = grpc.Instance().Run()
		if err != nil {
			log.Panic().Err(err).Msg("Error from running online-feature-store api-server")
		}
		log.Info().Msgf("online-feature-store gRPC API server started.")
	}
}
