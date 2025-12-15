package grpc

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
	httpserver "github.com/Meesho/BharatMLStack/online-feature-store/internal/server/http"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/p2p"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/soheilhy/cmux"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	GRPCServer  *grpc.Server
	HTTPHandler *gin.Engine
	enableHTTP  bool
}

var (
	server *Server
	once   sync.Once
)

// Init initializes the gRPC and HTTP server with the given middlewares if any
func Init() {
	once.Do(func() {

		env := viper.GetString("APP_ENV")
		if env == "prod" || env == "production" {
			gin.SetMode(gin.ReleaseMode)
		}

		// HTTP - Use string comparison (more reliable for env vars)
		enableHTTPStr := strings.ToLower(strings.TrimSpace(viper.GetString("ENABLE_HTTP_API")))
		enableHTTP := enableHTTPStr == "true" || enableHTTPStr == "1"

		// Create a Gin router
		router := gin.New()
		router.Use(gin.Recovery())
		router.Use(gin.Logger())

		// Create HTTP routes and handlers using Gin
		router.GET("/health/self", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "true"})
		})

		if enableHTTP {
			log.Info().Msg("HTTP API mode enabled - initializing HTTP server")
			httpserver.Init()
			log.Info().Msg("HTTP API routes registered")
		} else {
			log.Info().Msg("gRPC API mode enabled (default)")
			log.Warn().
				Str("ENABLE_HTTP_API_env", os.Getenv("ENABLE_HTTP_API")).
				Str("enableHTTPStr", enableHTTPStr).
				Msg("HTTP API mode DISABLED - using gRPC mode")
			grpcServer := grpc.NewServer(
				grpc.ChainUnaryInterceptor(ServerInterceptor, RecoveryInterceptor),
			)
			server = &Server{
				GRPCServer:  grpcServer,
				HTTPHandler: router,
				enableHTTP:  false,
			}
			return
		}
		server = &Server{
			GRPCServer:  nil,
			HTTPHandler: httpserver.Instance(),
			enableHTTP:  true,
		}
		log.Info().Msg("HTTP API server initialized successfully")
	})
}

// Run starts the cmux multiplexer (Mux) to handle incoming connections and route them to the appropriate servers
func (server *Server) Run() error {
	if !viper.IsSet("APP_PORT") {
		log.Panic().Msgf("Failed to start the application - APP_PORT is not set")
	}
	port := viper.GetInt("APP_PORT")

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Panic().Msgf("Failed to start the application - Failed to listen: %v", err)
	}

	if server.enableHTTP {
		return http.Serve(listener, server.HTTPHandler)
	}

	// Create a cmux multiplexer that will multiplex 2 protocols on same port
	mux := cmux.New(listener)

	// Create a cmux listener for HTTP connections
	httpListener := mux.Match(cmux.HTTP1Fast())
	// Create a cmux listener for gRPC connections
	grpcListener := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"), cmux.Any())

	retrieve.RegisterFeatureServiceServer(server.GRPCServer, feature.InitV1())
	persist.RegisterFeatureServiceServer(server.GRPCServer, feature.InitPersistHandler())
	p2p.RegisterP2PCacheServiceServer(server.GRPCServer, feature.InitP2PCacheHandler())
	reflection.Register(server.GRPCServer)
	// Start listeners for each protocol
	// Start the gRPC server in a separate goroutine
	go func() {
		if err := server.GRPCServer.Serve(grpcListener); err != nil {
			log.Panic().Msgf("Failed to serve gRPC server: %v", err)
		}
	}()
	// Start the HTTP server in a separate goroutine
	go func() {
		if err := http.Serve(httpListener, server.HTTPHandler); err != nil {
			log.Panic().Msgf("Failed to serve HTTP server: %v", err)
		}
	}()

	return mux.Serve()
}

// Instance returns the grpc instance
func Instance() *Server {
	if server == nil {
		log.Panic().Msg("Server not initialized, call Init first")
	}
	return server
}
