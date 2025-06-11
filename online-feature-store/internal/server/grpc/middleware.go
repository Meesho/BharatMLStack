package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	AuthToken       = "AUTH_TOKEN"
	callerIdHeader  = "online-feature-store-caller-id"
	AuthTokenHeader = "online-feature-store-auth-token"
)

func ServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	startTime := time.Now()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "mandatory metadata are missing")
	}
	callerId, ok := md[callerIdHeader]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s header is missing", callerIdHeader)
	}
	authHeader, ok := md[AuthTokenHeader]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s header is missing", AuthTokenHeader)
	}
	if !isAuthorized(callerId, authHeader) {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid auth token")
	}

	h, err := handler(ctx, req)

	rpcStatusCode := codes.OK
	if err != nil {
		rpcStatusCode = status.Code(err)
	}
	trackGenericMetrics(startTime, info, callerId[0], rpcStatusCode)
	return h, err
}

func isAuthorized(callerId, authHeader []string) bool {
	registeredClients := config.Instance(config.DefaultVersion).GetAllRegisteredClients()
	token, ok := registeredClients[callerId[0]]
	if !ok {
		return false
	}
	return token == authHeader[0]
}

func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Recover from panic and create a gRPC error
			log.Error().Msgf("Panic occurred in method %s: %v\n%s", info.FullMethod, r, debug.Stack())
			err = status.Errorf(codes.Internal, "panic recovered: %v", r)
		}
	}()
	resp, err = handler(ctx, req)

	return resp, err
}

func trackGenericMetrics(startTime time.Time, info *grpc.UnaryServerInfo, callerId string, statusCode codes.Code) {
	metric.Incr("grpc_server_requests_total", []string{"method", info.FullMethod, "caller_id", callerId, "status", statusCode.String()})
	metric.Timing("grpc_server_request_latency", time.Since(startTime), []string{"method", info.FullMethod, "caller_id", callerId, "status", statusCode.String()})
}
