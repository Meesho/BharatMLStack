package interactionstore

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const (
	ResolverDefaultScheme = "dns"

	// roundRobinServiceConfig is the default gRPC service config applied to every
	// interaction-store connection. It pins the load-balancing policy to round_robin.
	roundRobinServiceConfig = `{"loadBalancingPolicy":"round_robin"}`
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
	opts, err := dialOptions(config)
	if err != nil {
		return nil, err
	}
	gConn, err := grpc.NewClient(config.Host+":"+config.Port, opts...)
	if err != nil {
		return nil, err
	}
	return &GRPCClient{Conn: gConn, DeadLine: int64(config.DeadLine)}, nil
}

// dialOptions assembles the gRPC dial options for the connection: transport
// credentials (plaintext vs TLS) and the round-robin service config, plus an
// optional client keepalive that is appended only when configured (see
// keepaliveParamsFromConfig). When keepalive is not configured the option set is
// identical to the pre-keepalive behaviour, so existing callers are unaffected. It
// returns an error when the keepalive configuration is invalid.
func dialOptions(config Config) ([]grpc.DialOption, error) {
	var opts []grpc.DialOption
	if config.PlainText {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	opts = append(opts, grpc.WithDefaultServiceConfig(roundRobinServiceConfig))
	params, enabled, err := keepaliveParamsFromConfig(config)
	if err != nil {
		return nil, err
	}
	if enabled {
		opts = append(opts, grpc.WithKeepaliveParams(params))
	}
	return opts, nil
}

// keepaliveParamsFromConfig derives the client keepalive parameters from config.
// Keepalive is opt-in: it returns ok=false (and a nil error) when KeepaliveTimeMs <= 0,
// so no keepalive dial option is applied and behaviour is unchanged.
//
// When keepalive is enabled (KeepaliveTimeMs > 0), KeepaliveTimeoutMs must be positive.
// A non-positive timeout is rejected with an error rather than being converted into a
// zero or negative gRPC timer duration, which would make keepalive PINGs fail
// immediately and churn otherwise-healthy connections.
func keepaliveParamsFromConfig(config Config) (keepalive.ClientParameters, bool, error) {
	if config.KeepaliveTimeMs <= 0 {
		return keepalive.ClientParameters{}, false, nil
	}
	if config.KeepaliveTimeoutMs <= 0 {
		return keepalive.ClientParameters{}, false, fmt.Errorf(
			"interaction-store: KeepaliveTimeoutMs must be > 0 when keepalive is enabled "+
				"(KeepaliveTimeMs=%d ms), got %d ms", config.KeepaliveTimeMs, config.KeepaliveTimeoutMs)
	}
	return keepalive.ClientParameters{
		Time:                time.Duration(config.KeepaliveTimeMs) * time.Millisecond,
		Timeout:             time.Duration(config.KeepaliveTimeoutMs) * time.Millisecond,
		PermitWithoutStream: config.KeepalivePermitWithoutStream,
	}, true, nil
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
