package infra

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	gocql_v2 "github.com/Meesho/gocql"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Mandatory config keys
// <STORAGE_SCYLLA_1_CONTACT_POINTS> =
// <STORAGE_SCYLLA_1_PORT> =
// <STORAGE_SCYLLA_1_KEYSPACE> =
// <STORAGE_SCYLLA_1_VERSION> = Scylla version (e.g., 5.0, 6.0)
const (
	storageScyllaPrefix          = "STORAGE_SCYLLA_"
	contactPointsSuffix          = "_CONTACT_POINTS"
	portSuffix                   = "_PORT"
	keyspaceSuffix               = "_KEYSPACE"
	versionSuffix                = "_VERSION"
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

// ScyllaClusterConfig wraps the cluster config with type information
type ScyllaClusterConfig struct {
	Config   interface{} // Will hold either gocql or gocql_v2 config
	Version  string
	Keyspace string
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

// parseVersion parses the version string and returns the major version number
func parseVersion(versionStr string) (int, error) {
	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 {
		return 0, errors.New("invalid version format")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.New("invalid major version number")
	}
	return major, nil
}

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
//   - <envPrefix>_VERSION: Scylla version (e.g., 5.0, 6.0)
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
//   - A configured `ScyllaClusterConfig` instance or an error if mandatory variables are missing.
func BuildClusterConfigFromEnv(envPrefix string) (*ScyllaClusterConfig, error) {

	log.Debug().Msgf("building scylla cluster config from env, env prefix - %s", envPrefix)

	// Check for version first - this determines which gocql library to use
	if !viper.IsSet(envPrefix + versionSuffix) {
		return nil, errors.New(envPrefix + versionSuffix + " not set")
	}
	versionStr := viper.GetString(envPrefix + versionSuffix)
	majorVersion, err := parseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %v", err)
	}

	if !viper.IsSet(envPrefix + contactPointsSuffix) {
		return nil, errors.New(envPrefix + contactPointsSuffix + " not set")
	}
	contactPoints := viper.GetString(envPrefix + contactPointsSuffix)
	hosts := strings.Split(contactPoints, ",")

	// Get the keyspace first for validation
	if !viper.IsSet(envPrefix + keyspaceSuffix) {
		return nil, errors.New(envPrefix + keyspaceSuffix + " not set")
	}
	keyspace := viper.GetString(envPrefix + keyspaceSuffix)

	// Use the appropriate gocql library based on version
	// Version >= 6 uses gocql_v2, else uses standard gocql
	var cfg interface{}
	if majorVersion >= 6 {
		// Use gocql_v2 for Scylla 6.0+
		cfg, err = buildGocqlV2ClusterConfig(hosts, envPrefix, keyspace)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Using gocql_v2 library for Scylla version: %s (major: %d)", versionStr, majorVersion)
	} else {
		// Use standard gocql for Scylla < 6.0
		cfg, err = buildGocqlClusterConfig(hosts, envPrefix, keyspace)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Using standard gocql library for Scylla version: %s (major: %d)", versionStr, majorVersion)
	}

	return &ScyllaClusterConfig{
		Config:   cfg,
		Version:  versionStr,
		Keyspace: keyspace,
	}, nil
}
