package grpc

import (
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
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
}

var (
	server *Server
	once   sync.Once
)

// Init initializes the gRPC and HTTP server with the given middlewares if any
func Init() {
	once.Do(func() {
		// Create a gRPC server with the logger and recovery middleware
		grpcServer := grpc.NewServer(
			grpc.ChainUnaryInterceptor(ServerInterceptor, RecoveryInterceptor),
		)

		env := viper.GetString("APP_ENV")
		if env == "prod" || env == "production" {
			gin.SetMode(gin.ReleaseMode)
		}

		// Create a Gin router
		router := gin.New()

		// Create HTTP routes and handlers using Gin
		router.GET("/health/self", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "true"})
		})

		server = &Server{
			GRPCServer:  grpcServer,
			HTTPHandler: router,
		}

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
