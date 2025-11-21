package infra

import (
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/spf13/viper"
)

func createSessionV1(cfg interface{}) (interface{}, error) {
	typedCfg, ok := cfg.(*gocql.ClusterConfig)
	if !ok {
		return nil, errors.New("invalid gocql cluster config")
	}
	return typedCfg.CreateSession()
}

func isSessionClosedV1(session interface{}) bool {
	typedSession, ok := session.(*gocql.Session)
	if !ok {
		return false
	}
	return typedSession.Closed()
}

// buildGocqlClusterConfig creates and configures a cluster config using the standard gocql/gocql library
func buildGocqlClusterConfig(hosts []string, envPrefix string, keyspace string) (*gocql.ClusterConfig, error) {
	cfg := gocql.NewCluster(hosts...)
	cfg.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	cfg.Consistency = gocql.One

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
		cfg.Authenticator = gocql.PasswordAuthenticator{
			Username: viper.GetString(envPrefix + username),
			Password: viper.GetString(envPrefix + password),
		}
	}

	return cfg, nil
}
