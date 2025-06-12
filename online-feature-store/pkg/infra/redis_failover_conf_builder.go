package infra

import (
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	storageRedisFailoverPrefix          = "STORAGE_REDIS_FAILOVER_"
	redisMasterNameEnvSuffix            = "_MASTER_NAME"
	redisFailoverRouteRandomlyEnvSuffix = "_ROUTE_RANDOM"
	redisFailoverSentinelAddresses      = "_SENTINEL_ADDRESSES"
)

// BuildRedisFailoverOptionsFromEnv constructs Redis Sentinel failover configuration options
// using environment variables with the specified prefix.
//
// This function leverages Viper to read environment variables, ensuring
// a configurable Redis failover setup. It extracts key parameters required
// for connecting to a Redis Sentinel-managed failover cluster.
//
// Mandatory environment variables:
//   - <envPrefix>_MASTER_NAME: Name of the Redis master node monitored by Sentinel
//   - <envPrefix>_SENTINEL_ADDRESSES: Comma-separated list of Sentinel node addresses (host:port)
//   - <envPrefix>_READ_TIMEOUT_IN_MS: Read timeout duration (milliseconds)
//   - <envPrefix>_WRITE_TIMEOUT_IN_MS: Write timeout duration (milliseconds)
//
// Optional environment variables:
//   - <envPrefix>_USERNAME: Redis username (if authentication is enabled)
//   - <envPrefix>_PASSWORD: Redis password
//   - <envPrefix>_DB: Redis database index
//   - <envPrefix>_MAX_RETRY: Max number of retry attempts
//   - <envPrefix>_MIN_RETRY_BACKOFF_IN_MS: Min retry backoff duration (milliseconds)
//   - <envPrefix>_MAX_RETRY_BACKOFF_IN_MS: Max retry backoff duration (milliseconds)
//   - <envPrefix>_DIAL_TIMEOUT_IN_MS: Dial timeout duration (milliseconds)
//   - <envPrefix>_POOL_SIZE: Maximum number of connections in the pool
//   - <envPrefix>_MIN_IDLE_CONN: Minimum number of idle connections
//   - <envPrefix>_MAX_IDLE_CONN: Maximum number of idle connections
//   - <envPrefix>_MAX_ACTIVE_CONN: Maximum number of active connections
//   - <envPrefix>_CONN_MAX_AGE_IN_MINUTES: Maximum age of a connection (minutes)
//   - <envPrefix>_POOL_TIMEOUT_IN_MS: Pool wait timeout duration (milliseconds)
//   - <envPrefix>_ROUTE_RANDOM: Enable routing queries randomly among Redis replicas
//   - <envPrefix>_POOL_FIFO: Use FIFO pool ordering (default is LIFO)
//   - <envPrefix>_CONN_MAX_IDLE_TIMEOUT_IN_MINUTES: Max idle timeout for connections (minutes)
//   - <envPrefix>_DISABLE_IDENTITY: Disable client identity tracking (boolean)
//
// Returns:
//   - A configured `redis.FailoverOptions` instance or an error if mandatory variables are missing.
func BuildRedisFailoverOptionsFromEnv(envPrefix string) (*redis.FailoverOptions, error) {
	log.Debug().Msgf("building redis failover cluster config from env, env prefix - %s", envPrefix)

	// Mandatory environment variables check
	if !viper.IsSet(envPrefix + redisMasterNameEnvSuffix) {
		return nil, errors.New(envPrefix + redisMasterNameEnvSuffix + " not set")
	}
	if !viper.IsSet(envPrefix + redisFailoverSentinelAddresses) {
		return nil, errors.New(envPrefix + redisFailoverSentinelAddresses + " not set")
	}
	if !viper.IsSet(envPrefix + redisReadTimeoutEnvSuffix) {
		return nil, errors.New(envPrefix + redisReadTimeoutEnvSuffix + " not set")
	}
	if !viper.IsSet(envPrefix + redisWriteTimeoutEnvSuffix) {
		return nil, errors.New(envPrefix + redisWriteTimeoutEnvSuffix + " not set")
	}

	masterName := viper.GetString(envPrefix + redisMasterNameEnvSuffix)
	addresses := strings.Split(viper.GetString(envPrefix+redisFailoverSentinelAddresses), ",")
	for i := range addresses {
		addresses[i] = strings.TrimSpace(addresses[i])
	}
	readTimeout := viper.GetInt(envPrefix + redisReadTimeoutEnvSuffix)
	writeTimeout := viper.GetInt(envPrefix + redisWriteTimeoutEnvSuffix)

	// Building the Redis failover options
	failoverOptions := redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: addresses,
		ReadTimeout:   time.Duration(readTimeout) * time.Millisecond,
		WriteTimeout:  time.Duration(writeTimeout) * time.Millisecond,
	}

	// Optional environment variables
	if viper.IsSet(envPrefix + redisUsernameEnvSuffix) {
		failoverOptions.Username = viper.GetString(envPrefix + redisUsernameEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisPasswordEnvSuffix) {
		failoverOptions.Password = viper.GetString(envPrefix + redisPasswordEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisDbEnvSuffix) {
		failoverOptions.DB = viper.GetInt(envPrefix + redisDbEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxRetryEnvSuffix) {
		failoverOptions.MaxRetries = viper.GetInt(envPrefix + redisMaxRetryEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMinRetryBackoffEnvSuffix) {
		failoverOptions.MinRetryBackoff = time.Duration(viper.GetInt(envPrefix+redisMinRetryBackoffEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisMaxRetryBackoffEnvSuffix) {
		failoverOptions.MaxRetryBackoff = time.Duration(viper.GetInt(envPrefix+redisMaxRetryBackoffEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisDialTimeoutEnvSuffix) {
		failoverOptions.DialTimeout = time.Duration(viper.GetInt(envPrefix+redisDialTimeoutEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisPoolSizeEnvSuffix) {
		failoverOptions.PoolSize = viper.GetInt(envPrefix + redisPoolSizeEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMinIdleEnvSuffix) {
		failoverOptions.MinIdleConns = viper.GetInt(envPrefix + redisMinIdleEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxIdleEnvSuffix) {
		failoverOptions.MaxIdleConns = viper.GetInt(envPrefix + redisMaxIdleEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxActiveEnvSuffix) {
		failoverOptions.MaxActiveConns = viper.GetInt(envPrefix + redisMaxActiveEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxConnAgeEnvSuffix) {
		failoverOptions.ConnMaxLifetime = time.Duration(viper.GetInt(envPrefix+redisMaxConnAgeEnvSuffix)) * time.Minute
	}
	if viper.IsSet(envPrefix + redisPoolTimeoutEnvSuffix) {
		failoverOptions.PoolTimeout = time.Duration(viper.GetInt(envPrefix+redisPoolTimeoutEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisFailoverRouteRandomlyEnvSuffix) {
		failoverOptions.RouteRandomly = viper.GetBool(envPrefix + redisFailoverRouteRandomlyEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisPoolFifoEnvSuffix) {
		failoverOptions.PoolFIFO = viper.GetBool(envPrefix + redisPoolFifoEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxConnIdleTimeoutEnvSuffix) {
		failoverOptions.ConnMaxIdleTime = time.Duration(viper.GetInt(envPrefix+redisMaxConnIdleTimeoutEnvSuffix)) * time.Minute
	}
	if viper.IsSet(envPrefix + redisDisableIdentityEnvSuffix) {
		failoverOptions.DisableIdentity = viper.GetBool(envPrefix + redisDisableIdentityEnvSuffix)
	}

	log.Info().Msgf("redis failover cluster options built from env, env prefix - %s, options - %+v", envPrefix, failoverOptions)
	return &failoverOptions, nil
}
