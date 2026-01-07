//go:build !meesho

package config

import (
	"log"

	"github.com/Meesho/BharatMLStack/interaction-store/pkg/config"
	"github.com/spf13/viper"
)

type ConfigHolder interface {
	GetStaticConfig() interface{}
	GetDynamicConfig() interface{}
}

func InitConfig(configHolder ConfigHolder) {
	config.InitEnv()
	staticConfig := configHolder.GetStaticConfig()
	cfg, ok := staticConfig.(*Configs)
	if !ok {
		log.Fatal("Failed to cast static config to *Configs")
	}
	bindEnvVars()
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("Failed to unmarshal config from environment: %v", err)
	}
}

func bindEnvVars() {
	// App configuration
	viper.BindEnv("app_name", "APP_NAME")
	viper.BindEnv("app_env", "APP_ENV")
	viper.BindEnv("app_log_level", "APP_LOG_LEVEL")
	viper.BindEnv("app_metric_sampling_rate", "APP_METRIC_SAMPLING_RATE")
	viper.BindEnv("app_port", "APP_PORT")

	// MQ configuration
	viper.BindEnv("mq_api_auth_token", "MQ_API_AUTH_TOKEN")
	viper.BindEnv("mq_batch_size", "MQ_BATCH_SIZE")
	viper.BindEnv("mq_cache_refresh_initial", "MQ_CACHE_REFRESH_INITIAL")
	viper.BindEnv("mq_cache_refresh_schedule", "MQ_CACHE_REFRESH_SCHEDULE")
	viper.BindEnv("mq_consumer_enabled", "MQ_CONSUMER_ENABLED")
	viper.BindEnv("mq_consumer_version", "MQ_CONSUMER_VERSION")
	viper.BindEnv("mq_enabled", "MQ_ENABLED")
	viper.BindEnv("mq_env", "MQ_ENV")
	viper.BindEnv("mq_flush_interval_millis", "MQ_FLUSH_INTERVAL_MILLIS")
	viper.BindEnv("mq_keep_alive_duration", "MQ_KEEP_ALIVE_DURATION")
	viper.BindEnv("mq_max_idle_connections", "MQ_MAX_IDLE_CONNECTIONS")
	viper.BindEnv("mq_open_circuit_duration", "MQ_OPEN_CIRCUIT_DURATION")
	viper.BindEnv("mq_producer_enabled", "MQ_PRODUCER_ENABLED")
	viper.BindEnv("mq_queue_size_log_delay_seconds", "MQ_QUEUE_SIZE_LOG_DELAY_SECONDS")
	viper.BindEnv("mq_server_api_url", "MQ_SERVER_API_URL")
	viper.BindEnv("mq_server_connection_timeouts", "MQ_SERVER_CONNECTION_TIMEOUTS")

	// Profiling configuration
	viper.BindEnv("profiling_enabled", "PROFILING_ENABLED")
	viper.BindEnv("profiling_port", "PROFILING_PORT")

	// Click consumer configuration
	viper.BindEnv("click_consumer_mq_id", "CLICK_CONSUMER_MQ_ID")
	viper.BindEnv("order_consumer_mq_id", "ORDER_CONSUMER_MQ_ID")

	// Storage configuration
	viper.BindEnv("storage_scylla_contact_points", "STORAGE_SCYLLA_CONTACT_POINTS")
	viper.BindEnv("storage_scylla_keyspace", "STORAGE_SCYLLA_KEYSPACE")
	viper.BindEnv("storage_scylla_num_conns", "STORAGE_SCYLLA_NUM_CONNS")
	viper.BindEnv("storage_scylla_port", "STORAGE_SCYLLA_PORT")
	viper.BindEnv("storage_scylla_timeout_ms", "STORAGE_SCYLLA_TIMEOUT_MS")
}
