package middleware

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	MetadataMeeshoUserContext = "MEESHO-USER-CONTEXT"
)

func GRPCLogger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	startTime := time.Now()

	// Extract metadata from the context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	// Extract relevant information from the request
	userContext := md.Get(MetadataMeeshoUserContext)
	requestHeaders, _ := json.Marshal(md)

	// Call the gRPC handler to handle the request
	resp, err = handler(ctx, req)
	statusCode := codes.OK
	if err != nil {
		statusCode = status.Code(err)
	}
	// Extract relevant information from the response
	responseTime := time.Since(startTime)

	// Log the request and response information
	logMessage := strings.Join([]string{
		info.FullMethod,
		strconv.Itoa(int(statusCode)),
		responseTime.String(),
		string(requestHeaders),
	}, " | ")
	if err != nil {
		log.Error().Err(err).Msg(logMessage)
	} else {
		log.Info().Msg(logMessage)
	}
	telemetryMiddleware(info, responseTime, statusCode, userContext)
	return resp, err
}

func telemetryMiddleware(info *grpc.UnaryServerInfo, responseTime time.Duration,
	statusCode codes.Code, userContext []string) {
	metricTags := metric.BuildTag(
		metric.NewTag(metric.TagPath, info.FullMethod),
		metric.NewTag(metric.TagGrpcStatusCode, strconv.Itoa(int(statusCode))),
	)
	if len(userContext) != 0 {
		metricTags = append(metricTags, metric.TagAsString(metric.TagUserContext, userContext[0]))
	}
	metric.Timing(metric.ApiRequestLatency, responseTime, metricTags)
	metric.Incr(metric.ApiRequestCount, metricTags)
}
