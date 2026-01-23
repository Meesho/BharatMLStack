package scylla

import (
	"errors"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	contactPointsSuffix          = "_CONTACT_POINTS"
	portSuffix                   = "_PORT"
	keyspaceSuffix               = "_KEYSPACE"
	timeoutSuffix                = "_TIMEOUT_IN_MS"
	connectTimeoutSuffix         = "_CONNECT_TIMEOUT_IN_MS"
	numConnsSuffix               = "_NUM_CONNS"
	maxPreparedStmtsSuffix       = "_MAX_PREPARED_STATEMENTS"
	maxRoutingKeyInfoSuffix      = "_MAX_ROUTING_KEY_INFO"
	pageSizeSuffix               = "_PAGE_SIZE"
	maxWaitSchemaAgreementSuffix = "_MAX_WAIT_SCHEMA_AGREEMENT"
	reconnectIntervalSuffix      = "_RECONNECT_INTERVAL"
	writeCoalesceWaitTimeSuffix  = "_WRITE_COALESCE_WAIT_TIME"
	username                     = "_USERNAME"
	password                     = "_PASSWORD"
)

// BuildClusterConfigFromEnv builds a scylla cluster config from the environment variables
// It uses viper to read environment variables
// Env names are env prefix + scylla config name
// In case of type duration, env name is env prefix + scylla config name + "_IN_MS".
// This is done to avoid unit confusion
//
// Mandatory environment variables:
//   - <envPrefix>_CONTACT_POINTS
//   - <envPrefix>_PORT
//   - <envPrefix>_KEYSPACE
//
// Optional environment variables:
//   - <envPrefix>_TIMEOUT_IN_MS (default: 600ms)
//   - <envPrefix>_CONNECT_TIMEOUT_IN_MS (default: 600ms)
//   - <envPrefix>_NUM_CONNS (default: 1)
//   - <envPrefix>_MAX_PREPARED_STATEMENTS (default: 1000)
//   - <envPrefix>_MAX_ROUTING_KEY_INFO (default: 1000)
//   - <envPrefix>_PAGE_SIZE (default: 5000)
//   - <envPrefix>_MAX_WAIT_SCHEMA_AGREEMENT (default: 60s)
//   - <envPrefix>_RECONNECT_INTERVAL (default: 60s)
//   - <envPrefix>_WRITE_COALESCE_WAIT_TIME (default: 200us)
func BuildClusterConfigFromEnv(envPrefix string) (*gocql.ClusterConfig, error) {
	log.Debug().Msgf("building scylla cluster config from env, env prefix - %s", envPrefix)
	if !viper.IsSet(envPrefix + contactPointsSuffix) {
		return nil, errors.New(envPrefix + "_CONTACT_POINTS not set")
	}
	contactPoints := viper.GetString(envPrefix + contactPointsSuffix)
	hosts := strings.Split(contactPoints, ",")

	cfg := gocql.NewCluster(hosts...)

	if !viper.IsSet(envPrefix + portSuffix) {
		return nil, errors.New(envPrefix + "_PORT not set")
	}
	cfg.Port = viper.GetInt(envPrefix + portSuffix)

	if !viper.IsSet(envPrefix + keyspaceSuffix) {
		return nil, errors.New(envPrefix + "_KEYSPACE not set")
	}
	cfg.Keyspace = viper.GetString(envPrefix + keyspaceSuffix)

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
