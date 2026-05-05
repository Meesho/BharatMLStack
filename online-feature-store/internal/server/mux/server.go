package mux

import (
	"net"
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/server/grpc"
	httpserver "github.com/Meesho/BharatMLStack/online-feature-store/internal/server/http"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/p2p"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/rs/zerolog/log"
	"github.com/soheilhy/cmux"
	"github.com/spf13/viper"
	"google.golang.org/grpc/reflection"
)

// Server handles multiplexing HTTP and gRPC on the same port
type Server struct {
	mux cmux.CMux
}

// Init initializes the mux server (both gRPC and HTTP should be initialized before calling this)
func Init() (*Server, error) {
	if !viper.IsSet("APP_PORT") {
		log.Panic().Msgf("Failed to start the application - APP_PORT is not set")
	}
	port := viper.GetInt("APP_PORT")

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	// Create a cmux multiplexer that will multiplex 2 protocols on same port
	mux := cmux.New(listener)

	return &Server{
		mux: mux,
	}, nil
}

func (s *Server) RegisterServices() {
	grpcServer := grpc.Instance().GRPCServer
	retrieve.RegisterFeatureServiceServer(grpcServer, feature.InitV1())
	persist.RegisterFeatureServiceServer(grpcServer, feature.InitPersistHandler())
	p2p.RegisterP2PCacheServiceServer(grpcServer, feature.InitP2PCacheHandler())
	reflection.Register(grpcServer)
}

func (s *Server) Run() error {
	httpListener := s.mux.Match(cmux.HTTP1Fast())
	grpcListener := s.mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"), cmux.Any())

	go func() {
		if err := http.Serve(httpListener, httpserver.Instance()); err != nil {
			log.Panic().Msgf("Failed to serve HTTP server: %v", err)
		}
	}()

	grpcServer := grpc.Instance().GRPCServer
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Panic().Msgf("Failed to serve gRPC server: %v", err)
		}
	}()

	log.Info().Int("port", viper.GetInt("APP_PORT")).Msg("HTTP and gRPC servers started via cmux")
	return s.mux.Serve()
}
