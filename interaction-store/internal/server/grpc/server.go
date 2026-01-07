package grpc

import (
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/proto/timeseries"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/soheilhy/cmux"
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
func Init(config config.Configs) {
	once.Do(func() {
		// Create a gRPC server with the logger and recovery middleware
		grpcServer := grpc.NewServer(
			grpc.ChainUnaryInterceptor(ServerInterceptor, RecoveryInterceptor),
		)
		env := config.AppEnv
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
func (server *Server) Run(port int) error {
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

	timeseries.RegisterTimeSeriesServiceServer(server.GRPCServer, handler.InitInteractionHandler())
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
