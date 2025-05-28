package gosdk

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

const (
	ResolverDefaultScheme = "dns"
)

type GRPCClient struct {
	Conn                *grpc.ClientConn
	DeadLine            int64
	externalServiceName string
}

func NewConnFromConfig(config *Config, externalServiceName string) *GRPCClient {
	conn, err := getGRPCConnections(*config)
	if err != nil {
		log.Panic().Msgf("error while GRPC connection initialization. %s", err)
	}
	conn.externalServiceName = externalServiceName
	return conn
}

func getGRPCConnections(config Config) (*GRPCClient, error) {
	resolver.SetDefaultScheme(ResolverDefaultScheme)
	var gConn *grpc.ClientConn
	var err error
	if config.PlainText {
		gConn, err = grpc.NewClient(config.Host+":"+config.Port,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		)
	} else {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		gConn, err = grpc.NewClient(config.Host+":"+config.Port,
			grpc.WithTransportCredentials(creds),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if err != nil {
		return nil, err
	}
	return &GRPCClient{Conn: gConn, DeadLine: int64(config.DeadLine)}, nil
}

// Invoke is a wrapper around grpc.ClientConn.Invoke and capable to generate metric for external grpc service
func (c *GRPCClient) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	startTime := time.Now()
	err := c.Conn.Invoke(ctx, method, args, reply, opts...)
	var code uint32 = 0
	if err != nil {
		if e, ok := status.FromError(err); ok {
			code = uint32(e.Code())
		}
	}
	latency := time.Now().Sub(startTime)
	latencyTags := BuildExternalGRPCServiceLatencyTags(c.externalServiceName, method, int(code))
	countTags := BuildExternalGRPCServiceCountTags(c.externalServiceName, method, int(code))
	metric.Timing(metric.ExternalApiRequestLatency, latency, latencyTags)
	metric.Incr(metric.ExternalApiRequestCount, countTags)
	return err
}

func BuildExternalGRPCServiceLatencyTags(service, method string, statusCode int) []string {
	return metric.BuildTag(
		metric.NewTag(metric.TagCommunicationProtocol, metric.TagValueCommunicationProtocolGrpc),
		metric.NewTag(metric.TagExternalService, service),
		metric.NewTag(metric.TagMethod, method),
		metric.NewTag(metric.TagGrpcStatusCode, strconv.Itoa(statusCode)),
	)
}

func BuildExternalGRPCServiceCountTags(service, method string, statusCode int) []string {
	return metric.BuildTag(
		metric.NewTag(metric.TagCommunicationProtocol, metric.TagValueCommunicationProtocolGrpc),
		metric.NewTag(metric.TagExternalService, service),
		metric.NewTag(metric.TagMethod, method),
		metric.NewTag(metric.TagGrpcStatusCode, strconv.Itoa(statusCode)),
	)
}
func (c *GRPCClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("NewStream is not implemented")
}
