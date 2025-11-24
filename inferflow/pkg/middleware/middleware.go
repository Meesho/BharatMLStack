//go:build !meesho

package middleware

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	reqHeadersToLog *set.ThreadSafeSet
)

func InitGRPCMiddleware() {
	reqHeadersToLog = set.NewThreadSafeSet()
}

func WrappedGRPCMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	startTime := time.Now()

	// Extract metadata from the context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Retrieving metadata is failed")
	}

	if err := authorize(md); err != nil {
		return nil, err
	}
	// Extract relevant information from the request
	method := info.FullMethod
	requestHeaders, _ := json.Marshal(filterGRPCHeaders(md))

	// Call the gRPC handler to handle the request
	resp, err = handler(ctx, req)
	statusCode := codes.OK
	if err != nil {
		statusCode = status.Code(err)
	}
	// Extract relevant information from the response
	responseTime := time.Since(startTime)

	// Log the request and response information
	logVariables := []string{
		method,
		strconv.Itoa(int(statusCode)),
		responseTime.String(),
		string(requestHeaders),
	}
	if err != nil {
		logger.Error(strings.Join(logVariables, " | "), err)
	} else {
		logger.Info(strings.Join(logVariables, " | "))
	}
	telemetryMiddleware(info, responseTime, statusCode)
	return resp, err
}

func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic, stack trace, and the request
			log.Error().
				Str("service", info.FullMethod).
				Interface("request", req).
				Msgf("Recovered in recovery interceptor with err: %v, stack: %s", r, string(debug.Stack()))
			// Convert the panic to a gRPC error
			err = status.Errorf(codes.Internal, "Internal server error")
		}
	}()

	// Call the handler
	return handler(ctx, req)
}

func authorize(md metadata.MD) error {
	// Add authorization logic here if needed
	return nil
}

func filterGRPCHeaders(md metadata.MD) map[string][]string {

	filteredHeaders := make(map[string][]string)
	for k, v := range md {
		if reqHeadersToLog.Contains(k) {
			filteredHeaders[k] = v
		}
	}
	return filteredHeaders
}

func telemetryMiddleware(info *grpc.UnaryServerInfo, responseTime time.Duration, statusCode codes.Code) {
	tags := []string{"api:" + info.FullMethod, "status:" + strconv.Itoa(int(statusCode))}
	metrics.Timing("inferflow.router.api.request.latency", responseTime, tags)
	metrics.Count("inferflow.router.api.request.total", 1, tags)
}
