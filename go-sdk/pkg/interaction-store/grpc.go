package interactionstore

import (
	"context"
	"crypto/tls"
	"errors"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const (
	ResolverDefaultScheme = "dns"
)

// GRPCClient wraps a gRPC client connection with metrics support
type GRPCClient struct {
	Conn                *grpc.ClientConn
	DeadLine            int64
	externalServiceName string
	timing              func(name string, value time.Duration, tags []string)
	count               func(name string, value int64, tags []string)
}

// NewConnFromConfig creates a new gRPC connection from the provided configuration
func NewConnFromConfig(config *Config, externalServiceName string, timing func(name string, value time.Duration, tags []string), count func(name string, value int64, tags []string)) *GRPCClient {
	conn, err := getGRPCConnections(*config)
	if err != nil {
		log.Panic().Msgf("error while GRPC connection initialization. %s", err)
	}
	conn.externalServiceName = externalServiceName
	conn.timing = timing
	conn.count = count
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

// Invoke is a wrapper around grpc.ClientConn.Invoke with metrics support
func (c *GRPCClient) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	startTime := time.Now()
	err := c.Conn.Invoke(ctx, method, args, reply, opts...)
	var code uint32 = 0
	if err != nil {
		if e, ok := status.FromError(err); ok {
			code = uint32(e.Code())
		}
	}
	latency := time.Since(startTime)
	latencyTags := BuildExternalGRPCServiceLatencyTags(c.externalServiceName, method, int(code))
	countTags := BuildExternalGRPCServiceCountTags(c.externalServiceName, method, int(code))
	if c.timing != nil {
		c.timing("interaction-store.grpc.invoke.latency", latency, latencyTags)
	}
	if c.count != nil {
		c.count("interaction-store.grpc.invoke.count", 1, countTags)
	}
	return err
}

// BuildExternalGRPCServiceLatencyTags builds tags for latency metrics
func BuildExternalGRPCServiceLatencyTags(service, method string, statusCode int) []string {
	return []string{
		"communication_protocol:grpc",
		"external_service:" + service,
		"method:" + method,
		"grpc_status_code:" + strconv.Itoa(statusCode),
	}
}

// BuildExternalGRPCServiceCountTags builds tags for count metrics
func BuildExternalGRPCServiceCountTags(service, method string, statusCode int) []string {
	return []string{
		"communication_protocol:grpc",
		"external_service:" + service,
		"method:" + method,
		"grpc_status_code:" + strconv.Itoa(statusCode),
	}
}

// NewStream is not implemented for this client
func (c *GRPCClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("NewStream is not implemented")
}
