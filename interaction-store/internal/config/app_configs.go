package config

type Configs struct {
	// App configuration
	AppName               string  `mapstructure:"app_name"`
	AppEnv                string  `mapstructure:"app_env"`
	AppLogLevel           string  `mapstructure:"app_log_level"`
	AppMetricSamplingRate float64 `mapstructure:"app_metric_sampling_rate"`
	AppPort               int     `mapstructure:"app_port"`

	// MQ configuration
	MqApiAuthToken             string `mapstructure:"mq_api_auth_token"`
	MqBatchSize                int    `mapstructure:"mq_batch_size"`
	MqCacheRefreshInitial      int    `mapstructure:"mq_cache_refresh_initial"`
	MqCacheRefreshSchedule     int    `mapstructure:"mq_cache_refresh_schedule"`
	MqConsumerEnabled          bool   `mapstructure:"mq_consumer_enabled"`
	MqConsumerVersion          int    `mapstructure:"mq_consumer_version"`
	MqEnabled                  bool   `mapstructure:"mq_enabled"`
	MqEnv                      string `mapstructure:"mq_env"`
	MqFlushIntervalMillis      int    `mapstructure:"mq_flush_interval_millis"`
	MqKeepAliveDuration        int    `mapstructure:"mq_keep_alive_duration"`
	MqMaxIdleConnections       int    `mapstructure:"mq_max_idle_connections"`
	MqOpenCircuitDuration      int    `mapstructure:"mq_open_circuit_duration"`
	MqProducerEnabled          bool   `mapstructure:"mq_producer_enabled"`
	MqQueueSizeLogDelaySeconds int    `mapstructure:"mq_queue_size_log_delay_seconds"`
	MqServerApiUrl             string `mapstructure:"mq_server_api_url"`
	MqServerConnectionTimeouts int    `mapstructure:"mq_server_connection_timeouts"`

	// Profiling configuration
	ProfilingEnabled bool `mapstructure:"profiling_enabled"`
	ProfilingPort    int  `mapstructure:"profiling_port"`

	// Click consumer configuration
	ClickConsumerMqId int `mapstructure:"click_consumer_mq_id"`
	OrderConsumerMqId int `mapstructure:"order_consumer_mq_id"`

	// Storage configuration
	ScyllaContactPoints string `mapstructure:"storage_scylla_contact_points"`
	ScyllaKeyspace      string `mapstructure:"storage_scylla_keyspace"`
	ScyllaNumConns      int    `mapstructure:"storage_scylla_num_conns"`
	ScyllaPort          int    `mapstructure:"storage_scylla_port"`
	ScyllaTimeoutMs     int    `mapstructure:"storage_scylla_timeout_ms"`
}

type DynamicConfigs struct {
}
