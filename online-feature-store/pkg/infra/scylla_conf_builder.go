package infra

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Mandatory config keys
// <STORAGE_SCYLLA_1_CONTACT_POINTS> =
// <STORAGE_SCYLLA_1_PORT> =
// <STORAGE_SCYLLA_1_KEYSPACE> =
// <STORAGE_SCYLLA_1_MAJOR_VERSION> = Scylla major version (e.g., 5, 6)
const (
	storageScyllaPrefix          = "STORAGE_SCYLLA_"
	contactPointsSuffix          = "_CONTACT_POINTS"
	portSuffix                   = "_PORT"
	keyspaceSuffix               = "_KEYSPACE"
	isMeeshoVersionSuffix        = "_IS_MEESHO_VERSION"
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
	Version  string      // Major version number (e.g., 5, 6)
	Keyspace string
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
//   - <envPrefix>_MAJOR_VERSION: Scylla major version (e.g., 5, 6)
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
	if !viper.IsSet(envPrefix + isMeeshoVersionSuffix) {
		return nil, errors.New(envPrefix + isMeeshoVersionSuffix + " not set")
	}
	isMeeshoVersion := viper.GetString(envPrefix + isMeeshoVersionSuffix)
	if isMeeshoVersion != "true" && isMeeshoVersion != "false" {
		return nil, errors.New(envPrefix + isMeeshoVersionSuffix + " must be true or false")
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
	var err error
	if isMeeshoVersion == "true" {
		// Use gocql_v2 for Meesho version
		cfg, err = buildGocqlV2ClusterConfig(hosts, envPrefix, keyspace)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Using gocql_v2 library for Scylla version: %d (major: %d)", isMeeshoVersion, isMeeshoVersion)
	} else {
		// Use standard gocql for non-Meesho version
		cfg, err = buildGocqlClusterConfig(hosts, envPrefix, keyspace)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Using standard gocql library for Scylla version: %d (major: %d)", isMeeshoVersion, isMeeshoVersion)
	}

	return &ScyllaClusterConfig{
		Config:   cfg,
		Version:  isMeeshoVersion,
		Keyspace: keyspace,
	}, nil
}
