package infra

import (
	"errors"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Mandatory config keys
// <SCYLLA_1_CONTACT_POINTS> =
// <SCYLLA_1_PORT> =
// <SCYLLA_1_KEYSPACE> =
const (
	storageScyllaPrefix          = "SCYLLA_"
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

// BuildClusterConfigFromEnv constructs a ScyllaDB cluster configuration
// using environment variables with the specified prefix.
//
// The function leverages Viper to read environment variables, ensuring
// a flexible and configurable setup. It extracts key parameters required
// for configuring a gocql cluster.
//
// Mandatory environment variables:
//   - <envPrefix>_CONTACT_POINTS: Comma-separated list of Scylla nodes
//   - <envPrefix>_PORT: Scylla port
//   - <envPrefix>_KEYSPACE: Keyspace to connect to
//
// Optional environment variables:
//   - <envPrefix>_TIMEOUT_IN_MS: Request timeout (milliseconds)
//   - <envPrefix>_CONNECT_TIMEOUT_IN_MS: Connection timeout (milliseconds)
//   - <envPrefix>_NUM_CONNS: Number of connections per host
//   - <envPrefix>_MAX_PREPARED_STMTS: Max prepared statements in cache
//   - <envPrefix>_MAX_ROUTING_KEY_INFO: Max routing key info in cache
//   - <envPrefix>_PAGE_SIZE: Number of rows per page in queries
//   - <envPrefix>_MAX_WAIT_SCHEMA_AGREEMENT_IN_S: Max wait time for schema agreement (seconds)
//   - <envPrefix>_RECONNECT_INTERVAL_IN_S: Reconnection interval (seconds)
//   - <envPrefix>_WRITE_COALESCE_WAIT_TIME_IN_US: Write coalescing wait time (microseconds)
//
// Returns:
//   - A configured `gocql.ClusterConfig` instance or an error if mandatory variables are missing.
func BuildClusterConfigFromEnv(envPrefix string) (*gocql.ClusterConfig, error) {

	log.Debug().Msgf("building scylla cluster config from env, env prefix - %s", envPrefix)
	if !viper.IsSet(envPrefix + contactPointsSuffix) {
		return nil, errors.New(envPrefix + contactPointsSuffix + " not set")
	}
	contactPoints := viper.GetString(envPrefix + contactPointsSuffix)
	hosts := strings.Split(contactPoints, ",")

	cfg := gocql.NewCluster(hosts...)

	if !viper.IsSet(envPrefix + portSuffix) {
		return nil, errors.New(envPrefix + portSuffix + " not set")
	}
	cfg.Port = viper.GetInt(envPrefix + portSuffix)

	if !viper.IsSet(envPrefix + keyspaceSuffix) {
		return nil, errors.New(envPrefix + keyspaceSuffix + " not set")
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
