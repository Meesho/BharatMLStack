package middlewares

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	CallerIdHeader  = "skye-caller-id"
	AuthTokenHeader = "skye-auth-token"
)

var (
	authTokens string
	initOnce   sync.Once
)

func Init() {
	initOnce.Do(func() {
		authTokens = structs.GetAppConfig().Configs.AuthTokens
	})
}

func ServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	startTime := time.Now()
	// check if headers are present
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "mandatory metadata are missing")
	}
	// check if caller-id header exists
	callerIdHeader, ok := md[CallerIdHeader]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s header is missing", CallerIdHeader)
	}
	// check if auth token header exists
	authHeader, ok := md[AuthTokenHeader]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s header is missing", AuthTokenHeader)
	}
	// check if token is valid
	if !isAuthorized(authHeader) {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid auth token")
	}

	// call handler
	h, err := handler(ctx, req)

	// if rpc succeeded or not
	rpcStatusCode := codes.OK
	if err != nil {
		rpcStatusCode = status.Code(err)
	}
	trackGenericMetrics(startTime, info, callerIdHeader[0], rpcStatusCode)
	return h, err
}

func isAuthorized(authHeaders []string) bool {
	if len(authTokens) == 0 {
		log.Panic().Msgf("AuthTokens not set")
	}
	permittedTokens := authTokens
	tokens := strings.Split(permittedTokens, ",")
	token := authHeaders[0]
	if slices.Contains(tokens, token) {
		return true
	}
	return false
}

func trackGenericMetrics(startTime time.Time, info *grpc.UnaryServerInfo, callerID string, statusCode codes.Code) {
	tags := []string{"method", info.FullMethod, "caller_id", callerID, "status", statusCode.String()}
	metric.Incr("skye_grpc_request", tags)
	metric.Timing("skye_grpc_request_latency", time.Since(startTime), tags)
}
