package grpcclient

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const (
	ResolverDefaultScheme = "dns"
)

type Config struct {
	Host                string
	Port                string
	DeadLine            int
	LoadBalancingPolicy string
	PlainText           bool
}

type GRPCClient struct {
	Conn      *grpc.ClientConn
	DeadLine  int64
	envPrefix string
}

func NewConnFromConfig(config *Config, envPrefix string) *GRPCClient {
	conn, err := getGRPCConnections(*config)
	if err != nil {
		log.Panic().Msgf("error while GRPC connection initialization from config - %#v. error - %s", config, err)
	}
	conn.envPrefix = envPrefix
	return conn
}

func NewConn(envPrefix string) *GRPCClient {
	config := newConfig(envPrefix)
	conn, err := getGRPCConnections(*config)
	if err != nil {
		log.Panic().Msgf("error while %s GRPC connection initialization. %s", envPrefix, err)
	}
	conn.envPrefix = envPrefix
	return conn
}

func newConfig(envPrefix string) *Config {
	config := &Config{
		DeadLine:  200_000,
		PlainText: false,
	}
	if !viper.IsSet(envPrefix + "_HOST") {
		log.Panic().Msg(envPrefix + "_HOST not set")
	}
	if !viper.IsSet(envPrefix + "_PORT") {
		log.Panic().Msg(envPrefix + "_PORT not set")
	}
	if !viper.IsSet(envPrefix + "_TIMEOUT_IN_MS") {
		log.Panic().Msg(envPrefix + "_TIMEOUT_IN_MS not set")
	}
	if viper.IsSet(envPrefix + "_GRPC_PLAIN_TEXT") {
		config.PlainText = viper.GetBool(envPrefix + "_GRPC_PLAIN_TEXT")
	}
	if viper.IsSet(envPrefix + "_LOAD_BALANCING_POLICY") {
		config.LoadBalancingPolicy = viper.GetString(envPrefix + "_LOAD_BALANCING_POLICY")
	} else {
		config.LoadBalancingPolicy = "round_robin"
	}
	config.Host = viper.GetString(envPrefix + "_HOST")
	config.Port = viper.GetString(envPrefix + "_PORT")
	config.DeadLine = viper.GetInt(envPrefix + "_TIMEOUT_IN_MS")
	return config
}

func getGRPCConnections(config Config) (*GRPCClient, error) {
	resolver.SetDefaultScheme(ResolverDefaultScheme)
	var gConn *grpc.ClientConn
	var err error
	if config.Host == "" {
		return nil, errors.New("host is not set")
	}
	if config.Port == "" {
		return nil, errors.New("port is not set")
	}
	if config.DeadLine == 0 || int64(config.DeadLine) < 0 {
		return nil, errors.New("deadline is not set or is negative")
	}

	if config.LoadBalancingPolicy == "" {
		log.Warn().Msgf("Load balancing policy is not set for %s. Setting it to round robin", config.Host)
		config.LoadBalancingPolicy = "round_robin"
	}
	if config.PlainText {
		gConn, err = grpc.NewClient(config.Host+":"+config.Port,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"`+config.LoadBalancingPolicy+`"}`),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
	} else {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		gConn, err = grpc.NewClient(config.Host+":"+config.Port,
			grpc.WithTransportCredentials(creds),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"`+config.LoadBalancingPolicy+`"}`),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
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
	latency := time.Since(startTime)
	latencyTags := metric.BuildExternalGRPCServiceLatencyTags(c.envPrefix, method, int(code))
	countTags := metric.BuildExternalGRPCServiceCountTags(c.envPrefix, method, int(code))
	metric.Timing(metric.ExternalApiRequestLatency, latency, latencyTags)
	metric.Incr(metric.ExternalApiRequestCount, countTags)
	return err
}

func (c *GRPCClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return c.Conn.NewStream(ctx, desc, method, opts...)
}
