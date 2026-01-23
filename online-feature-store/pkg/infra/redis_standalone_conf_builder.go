package infra

import (
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	storageRedisStandalonePrefix     = "STORAGE_REDIS_STANDALONE_"
	redisNetworkEnvSuffix            = "_NETWORK"
	redisAddrEnvSuffix               = "_ADDR"
	redisUsernameEnvSuffix           = "_USERNAME"
	redisPasswordEnvSuffix           = "_PASSWORD"
	redisDbEnvSuffix                 = "_DB"
	redisMaxRetryEnvSuffix           = "_MAX_RETRY"
	redisMinRetryBackoffEnvSuffix    = "_MIN_RETRY_BACKOFF_IN_MS"
	redisMaxRetryBackoffEnvSuffix    = "_MAX_RETRY_BACKOFF_IN_MS"
	redisDialTimeoutEnvSuffix        = "_DIAL_TIMEOUT_IN_MS"
	redisReadTimeoutEnvSuffix        = "_READ_TIMEOUT_IN_MS"
	redisWriteTimeoutEnvSuffix       = "_WRITE_TIMEOUT_IN_MS"
	redisPoolFifoEnvSuffix           = "_POOL_FIFO"
	redisPoolSizeEnvSuffix           = "_POOL_SIZE"
	redisMinIdleEnvSuffix            = "_MIN_IDLE_CONN"
	redisMaxIdleEnvSuffix            = "_MAX_IDLE_CONN"
	redisMaxActiveEnvSuffix          = "_MAX_ACTIVE_CONN"
	redisMaxConnAgeEnvSuffix         = "_CONN_MAX_AGE_IN_MINUTES"
	redisPoolTimeoutEnvSuffix        = "_POOL_TIMEOUT_IN_MS"
	redisMaxConnIdleTimeoutEnvSuffix = "_CONN_MAX_IDLE_TIMEOUT_IN_MINUTES"
	redisDisableIdentityEnvSuffix    = "_DISABLE_IDENTITY"
)

// BuildRedisOptionsFromEnv constructs Redis standalone configuration options
// using environment variables with the specified prefix.
//
// This function leverages Viper to read environment variables, ensuring
// a configurable Redis setup. It extracts the key parameters required
// for connecting to a standalone Redis instance.
//
// Mandatory environment variables:
//   - <envPrefix>_ADDR: Redis server address (host:port)
//   - <envPrefix>_DB: Redis database index
//   - <envPrefix>_READ_TIMEOUT_IN_MS: Read timeout duration (milliseconds)
//   - <envPrefix>_WRITE_TIMEOUT_IN_MS: Write timeout duration (milliseconds)
//
// Optional environment variables:
//   - <envPrefix>_NETWORK: Network type (tcp, unix, etc.)
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
//   - A configured `redis.Options` instance or an error if mandatory variables are missing.

func BuildRedisOptionsFromEnv(envPrefix string) (*redis.Options, error) {

	log.Debug().Msgf("building redis standalone config from env, env prefix - %s", envPrefix)

	// Mandatory environment variables check
	if !viper.IsSet(envPrefix + redisAddrEnvSuffix) {
		return nil, errors.New(envPrefix + redisAddrEnvSuffix + " not set")
	}
	if !viper.IsSet(envPrefix + redisDbEnvSuffix) {
		return nil, errors.New(envPrefix + redisDbEnvSuffix + " not set")
	}
	if !viper.IsSet(envPrefix + redisReadTimeoutEnvSuffix) {
		return nil, errors.New(envPrefix + redisReadTimeoutEnvSuffix + " not set")
	}
	if !viper.IsSet(envPrefix + redisWriteTimeoutEnvSuffix) {
		return nil, errors.New(envPrefix + redisWriteTimeoutEnvSuffix + " not set")
	}
	addr := viper.GetString(envPrefix + redisAddrEnvSuffix)
	db := viper.GetInt(envPrefix + redisDbEnvSuffix)
	readTimeout := viper.GetInt(envPrefix + redisReadTimeoutEnvSuffix)
	writeTimeout := viper.GetInt(envPrefix + redisWriteTimeoutEnvSuffix)

	// Building the Redis options
	redisOptions := redis.Options{
		Addr:         addr,
		DB:           db,
		ReadTimeout:  time.Duration(readTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(writeTimeout) * time.Millisecond,
	}

	// Optional environment variables
	if viper.IsSet(envPrefix + redisNetworkEnvSuffix) {
		redisOptions.Network = viper.GetString(envPrefix + redisNetworkEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisUsernameEnvSuffix) {
		redisOptions.Username = viper.GetString(envPrefix + redisUsernameEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisPasswordEnvSuffix) {
		redisOptions.Password = viper.GetString(envPrefix + redisPasswordEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxRetryEnvSuffix) {
		redisOptions.MaxRetries = viper.GetInt(envPrefix + redisMaxRetryEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMinRetryBackoffEnvSuffix) {
		redisOptions.MinRetryBackoff = time.Duration(viper.GetInt(envPrefix+redisMinRetryBackoffEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisMaxRetryBackoffEnvSuffix) {
		redisOptions.MaxRetryBackoff = time.Duration(viper.GetInt(envPrefix+redisMaxRetryBackoffEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisDialTimeoutEnvSuffix) {
		redisOptions.DialTimeout = time.Duration(viper.GetInt(envPrefix+redisDialTimeoutEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisPoolFifoEnvSuffix) {
		redisOptions.PoolFIFO = viper.GetBool(envPrefix + redisPoolFifoEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisPoolSizeEnvSuffix) {
		redisOptions.PoolSize = viper.GetInt(envPrefix + redisPoolSizeEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMinIdleEnvSuffix) {
		redisOptions.MinIdleConns = viper.GetInt(envPrefix + redisMinIdleEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxIdleEnvSuffix) {
		redisOptions.MaxIdleConns = viper.GetInt(envPrefix + redisMaxIdleEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxActiveEnvSuffix) {
		redisOptions.MaxActiveConns = viper.GetInt(envPrefix + redisMaxActiveEnvSuffix)
	}
	if viper.IsSet(envPrefix + redisMaxConnAgeEnvSuffix) {
		redisOptions.ConnMaxLifetime = time.Duration(viper.GetInt(envPrefix+redisMaxConnAgeEnvSuffix)) * time.Minute
	}
	if viper.IsSet(envPrefix + redisPoolTimeoutEnvSuffix) {
		redisOptions.PoolTimeout = time.Duration(viper.GetInt(envPrefix+redisPoolTimeoutEnvSuffix)) * time.Millisecond
	}
	if viper.IsSet(envPrefix + redisMaxConnIdleTimeoutEnvSuffix) {
		redisOptions.ConnMaxIdleTime = time.Duration(viper.GetInt(envPrefix+redisMaxConnIdleTimeoutEnvSuffix)) * time.Minute
	}
	if viper.IsSet(envPrefix + redisDisableIdentityEnvSuffix) {
		redisOptions.DisableIndentity = viper.GetBool(envPrefix + redisDisableIdentityEnvSuffix)
	}
	log.Info().Msgf("redis options built from env, env prefix - %s, options - %+v", envPrefix, redisOptions)
	return &redisOptions, nil
}
