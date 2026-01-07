package mux

import (
	"net"
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/server/grpc"
	httpserver "github.com/Meesho/BharatMLStack/interaction-store/internal/server/http"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/proto/timeseries"
	"github.com/rs/zerolog/log"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc/reflection"
)

// Server handles multiplexing HTTP and gRPC on the same port
type Server struct {
	mux cmux.CMux
}

// Init initializes the mux server (both gRPC and HTTP should be initialized before calling this)
func Init(config config.Configs) (*Server, error) {
	port := config.AppPort

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
	timeseries.RegisterTimeSeriesServiceServer(grpcServer, handler.InitInteractionHandler())
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

	return s.mux.Serve()
}
