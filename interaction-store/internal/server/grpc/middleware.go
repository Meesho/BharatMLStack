package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	callerIdHeader = "interaction-store-caller-id"
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

	h, err := handler(ctx, req)

	rpcStatusCode := codes.OK
	if err != nil {
		rpcStatusCode = status.Code(err)
	}
	trackGenericMetrics(startTime, info, callerId[0], rpcStatusCode)
	return h, err
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
