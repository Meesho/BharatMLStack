package infra

import (
	"errors"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	storageRedisClusterPrefix  = "STORAGE_REDIS_CLUSTER_"
	redisClusterAddrsEnvSuffix = "_ADDRESSES"
)

// BuildRedisClusterOptionsFromEnv constructs Redis Cluster configuration options
// using environment variables with the specified prefix.
//
// This function leverages Viper to read environment variables, ensuring
// a configurable Redis Cluster setup. It extracts key parameters required
// for connecting to a Redis Cluster.
//
// Mandatory environment variables:
//   - <envPrefix>_ADDRESSES: Comma-separated list of Redis cluster node addresses (host:port)
//   - <envPrefix>_READ_TIMEOUT_IN_MS: Read timeout duration (milliseconds)
//   - <envPrefix>_WRITE_TIMEOUT_IN_MS: Write timeout duration (milliseconds)
//
// Optional environment variables:
//   - <envPrefix>_USERNAME: Redis username (if authentication is enabled)
//   - <envPrefix>_PASSWORD: Redis password
//   - <envPrefix>_MAX_RETRY: Max number of retry attempts
//   - <envPrefix>_MIN_RETRY_BACKOFF_IN_MS: Min retry backoff duration (milliseconds)
//   - <envPrefix>_MAX_RETRY_BACKOFF_IN_MS: Max retry backoff duration (milliseconds)
//   - <envPrefix>_DIAL_TIMEOUT_IN_MS: Dial timeout duration (milliseconds)
//   - <envPrefix>_POOL_FIFO: Use FIFO pool ordering (default is LIFO)
//   - <envPrefix>_POOL_SIZE: Maximum number of connections in the pool
//   - <envPrefix>_MIN_IDLE_CONN: Minimum number of idle connections
//   - <envPrefix>_MAX_IDLE_CONN: Maximum number of idle connections
//   - <envPrefix>_MAX_ACTIVE_CONN: Maximum number of active connections
//   - <envPrefix>_CONN_MAX_AGE_IN_MINUTES: Maximum age of a connection (minutes)
//   - <envPrefix>_POOL_TIMEOUT_IN_MS: Pool wait timeout duration (milliseconds)
//   - <envPrefix>_CONN_MAX_IDLE_TIMEOUT_IN_MINUTES: Max idle timeout for connections (minutes)
//   - <envPrefix>_DISABLE_IDENTITY: Disable client identity tracking (boolean)
//
// Returns:
//   - A configured `redis.ClusterOptions` instance or an error if mandatory variables are missing.
func BuildRedisClusterOptionsFromEnv(envPrefix string) (*redis.ClusterOptions, error) {
	log.Debug().Msgf("building redis cluster options from env, env prefix - %s", envPrefix)

	// Mandatory environment variables check
	if !viper.IsSet(envPrefix + redisClusterAddrsEnvSuffix) {
		return nil, errors.New(envPrefix + redisClusterAddrsEnvSuffix + " not set")
	}
	addresses := strings.Split(viper.GetString(envPrefix+redisClusterAddrsEnvSuffix), ",")
	for i := range addresses {
		addresses[i] = strings.TrimSpace(addresses[i])
	}

	if !viper.IsSet(envPrefix + redisReadTimeoutEnvSuffix) {
		return nil, errors.New(envPrefix + redisReadTimeoutEnvSuffix + " not set")
	}
	readTimeout := viper.GetInt(envPrefix + redisReadTimeoutEnvSuffix)

	if !viper.IsSet(envPrefix + redisWriteTimeoutEnvSuffix) {
		return nil, errors.New(envPrefix + redisWriteTimeoutEnvSuffix + " not set")
	}
	writeTimeout := viper.GetInt(envPrefix + redisWriteTimeoutEnvSuffix)

	// Building the ClusterOptions struct
	clusterOptions := redis.ClusterOptions{
		Addrs:        addresses,
		ReadTimeout:  time.Duration(readTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(writeTimeout) * time.Millisecond,
	}

	// Optional environment variables
	if viper.IsSet(envPrefix + redisUsernameEnvSuffix) {
		clusterOptions.Username = viper.GetString(envPrefix + redisUsernameEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisPasswordEnvSuffix) {
		clusterOptions.Password = viper.GetString(envPrefix + redisPasswordEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxRetryEnvSuffix) {
		clusterOptions.MaxRetries = viper.GetInt(envPrefix + redisMaxRetryEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMinRetryBackoffEnvSuffix) {
		clusterOptions.MinRetryBackoff = time.Duration(viper.GetInt(envPrefix+redisMinRetryBackoffEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisMaxRetryBackoffEnvSuffix) {
		clusterOptions.MaxRetryBackoff = time.Duration(viper.GetInt(envPrefix+redisMaxRetryBackoffEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisDialTimeoutEnvSuffix) {
		clusterOptions.DialTimeout = time.Duration(viper.GetInt(envPrefix+redisDialTimeoutEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisPoolFifoEnvSuffix) {
		clusterOptions.PoolFIFO = viper.GetBool(envPrefix + redisPoolFifoEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisPoolSizeEnvSuffix) {
		clusterOptions.PoolSize = viper.GetInt(envPrefix + redisPoolSizeEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMinIdleEnvSuffix) {
		clusterOptions.MinIdleConns = viper.GetInt(envPrefix + redisMinIdleEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxIdleEnvSuffix) {
		clusterOptions.MaxIdleConns = viper.GetInt(envPrefix + redisMaxIdleEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxActiveEnvSuffix) {
		clusterOptions.MaxActiveConns = viper.GetInt(envPrefix + redisMaxActiveEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxConnAgeEnvSuffix) {
		clusterOptions.ConnMaxLifetime = time.Duration(viper.GetInt(envPrefix+redisMaxConnAgeEnvSuffix)) * time.Minute
	}
	if viper.IsSet(envPrefix + redisPoolTimeoutEnvSuffix) {
		clusterOptions.PoolTimeout = time.Duration(viper.GetInt(envPrefix+redisPoolTimeoutEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisMaxConnIdleTimeoutEnvSuffix) {
		clusterOptions.ConnMaxIdleTime = time.Duration(viper.GetInt(envPrefix+redisMaxConnIdleTimeoutEnvSuffix)) * time.Minute
	}
	if viper.IsSet(envPrefix + redisDisableIdentityEnvSuffix) {
		clusterOptions.DisableIndentity = viper.GetBool(envPrefix + redisDisableIdentityEnvSuffix)
	}

	log.Info().Msgf("redis cluster options built from env, env prefix - %s, options - %+v", envPrefix, clusterOptions)

	return &clusterOptions, nil
}
