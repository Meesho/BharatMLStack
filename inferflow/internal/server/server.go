package server

import (
	"fmt"
	"net"
	"net/http"
	"os"

	pb "github.com/Meesho/BharatMLStack/inferflow/client/grpc"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/inferflow"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/middleware"
	"github.com/cockroachdb/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func InitServer(configs *configs.AppConfigs) {

	address := fmt.Sprintf("%d", configs.Configs.ApplicationPort)
	listener, err := net.Listen("tcp", ":"+address)
	if err != nil {
		logger.Panic("Failed to start inferflow application!", err)
	}

	// create the cmux object that will multiplex 2 protocols on same port
	mux := cmux.New(listener)
	// match gRPC requests, otherwise regular HTTP requests
	grpcListener := mux.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpListener := mux.Match(cmux.Any())

	// GRPC Server :
	// initializing GRPC Middleware
	middleware.InitGRPCMiddleware()
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware.RecoveryInterceptor, middleware.WrappedGRPCMiddleware),
	)
	reflection.Register(grpcServer)
	pb.RegisterInferflowServer(grpcServer, &inferflow.Inferflow{})

	// HTTP Server :
	h := http.NewServeMux()
	h.HandleFunc("/health/self", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "true")
	})
	httpServer := &http.Server{
		Handler: h,
	}

	// Collect on this channel,the exits of each protocol's .Serve() call
	eps := make(chan error, 2)
	// Start listeners for each protocol
	go func() { eps <- grpcServer.Serve(grpcListener) }()
	go func() { eps <- httpServer.Serve(httpListener) }()

	logger.Info(fmt.Sprintf("inferflow started at port on %s", address))
	handleErrors(mux, eps)
}

func handleErrors(mux cmux.CMux, eps chan error) {

	// code handles exit errors of the muxes
	err := mux.Serve()
	var failed bool
	if err != nil {
		logger.Error("cmux serve error: %v", err)
		failed = true
	}
	var i int
	for err := range eps {
		if err != nil {
			logger.Error("protocol serve error: %v", err)
			failed = true
		}
		i++
		if i == cap(eps) {
			close(eps)
			break
		}
	}
	if failed {
		os.Exit(1)
	}
}
