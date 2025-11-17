//go:build meesho

package infra

import (
	"errors"
	"time"

	gocql_v2 "github.com/Meesho/gocql"
	"github.com/spf13/viper"
)

func createSessionV2(cfg interface{}) (interface{}, error) {
	typedCfg, ok := cfg.(*gocql_v2.ClusterConfig)
	if !ok {
		return nil, errors.New("invalid gocql_v2 cluster config")
	}
	return typedCfg.CreateSession()
}

func isSessionClosedV2(session interface{}) bool {
	typedSession, ok := session.(*gocql_v2.Session)
	if !ok {
		return false
	}
	return typedSession.Closed()
}

// buildGocqlV2ClusterConfig creates and configures a cluster config using the gocql_v2 library
func buildGocqlV2ClusterConfig(hosts []string, envPrefix string, keyspace string) (*gocql_v2.ClusterConfig, error) {
	cfg := gocql_v2.NewCluster(hosts...)
	cfg.PoolConfig.HostSelectionPolicy = gocql_v2.TokenAwareHostPolicy(gocql_v2.RoundRobinHostPolicy())
	cfg.Consistency = gocql_v2.One

	// Set port
	if !viper.IsSet(envPrefix + portSuffix) {
		return nil, errors.New(envPrefix + portSuffix + " not set")
	}
	cfg.Port = viper.GetInt(envPrefix + portSuffix)
	cfg.Keyspace = keyspace

	// Set optional configurations
	if viper.IsSet(envPrefix + timeoutSuffix) {
		cfg.Timeout = time.Duration(viper.GetInt(envPrefix+timeoutSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + connectTimeoutSuffix) {
		cfg.ConnectTimeout = time.Duration(viper.GetInt(envPrefix+connectTimeoutSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + numConnsSuffix) {
		cfg.NumConns = viper.GetInt(envPrefix + numConnsSuffix)
	}
	if viper.IsSet(envPrefix + maxPreparedStmtsSuffix) {
		cfg.MaxPreparedStmts = viper.GetInt(envPrefix + maxPreparedStmtsSuffix)
	}
	if viper.IsSet(envPrefix + maxRoutingKeyInfoSuffix) {
		cfg.MaxRoutingKeyInfo = viper.GetInt(envPrefix + maxRoutingKeyInfoSuffix)
	}
	if viper.IsSet(envPrefix + pageSizeSuffix) {
		cfg.PageSize = viper.GetInt(envPrefix + pageSizeSuffix)
	}
	if viper.IsSet(envPrefix + maxWaitSchemaAgreementSuffix) {
		cfg.MaxWaitSchemaAgreement = time.Duration(viper.GetInt(envPrefix+maxWaitSchemaAgreementSuffix)) * time.Second
	}
	if viper.IsSet(envPrefix + reconnectIntervalSuffix) {
		cfg.ReconnectInterval = time.Duration(viper.GetInt(envPrefix+reconnectIntervalSuffix)) * time.Second
	}
	if viper.IsSet(envPrefix + writeCoalesceWaitTimeSuffix) {
		cfg.WriteCoalesceWaitTime = time.Duration(viper.GetInt(envPrefix+writeCoalesceWaitTimeSuffix)) * time.Microsecond
	}
	if viper.IsSet(envPrefix+username) && viper.IsSet(envPrefix+password) {
		cfg.Authenticator = gocql_v2.PasswordAuthenticator{
			Username: viper.GetString(envPrefix + username),
			Password: viper.GetString(envPrefix + password),
		}
	}

	return cfg, nil
}
