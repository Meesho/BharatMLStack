package vector

import (
	"crypto/tls"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
)

const (
	ResolverDefaultScheme = "dns"
)

type Config struct {
	Host                string
	Port                string
	DeadLine            int
	LoadBalancingPolicy string
	KeepAliveTimeMs     int
	KeepAliveTimeoutMs  int
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

func getGRPCConnections(config Config) (*GRPCClient, error) {
	resolver.SetDefaultScheme(ResolverDefaultScheme)
	var gConn *grpc.ClientConn
	var err error
	if config.LoadBalancingPolicy == "" {
		log.Warn().Msgf("Load balancing policy is not set for %s. Setting it to round robin", config.Host)
		config.LoadBalancingPolicy = "round_robin"
	}
	if config.PlainText {
		gConn, err = grpc.NewClient(config.Host+":"+config.Port,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"`+config.LoadBalancingPolicy+`"}`),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:    time.Duration(config.KeepAliveTimeMs) * time.Millisecond,
				Timeout: time.Duration(config.KeepAliveTimeoutMs) * time.Millisecond,
			}),
		)
	} else {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		gConn, err = grpc.Dial(config.Host+":"+config.Port,
			grpc.WithTransportCredentials(creds),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"`+config.LoadBalancingPolicy+`"}`),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:    time.Duration(config.KeepAliveTimeMs) * time.Millisecond,
				Timeout: time.Duration(config.KeepAliveTimeoutMs) * time.Millisecond,
			}),
		)
	}
	if err != nil {
		return nil, err
	}
	return &GRPCClient{Conn: gConn, DeadLine: int64(config.DeadLine)}, nil
}
