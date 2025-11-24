package configs

type Configs struct {
	// App configuration
	AppName               string  `mapstructure:"app_name"`
	AppEnv                string  `mapstructure:"app_env"`
	AppGcPercentage       int     `mapstructure:"app_gc_percentage"`
	AppLogLevel           string  `mapstructure:"app_log_level"`
	AppMetricSamplingRate float64 `mapstructure:"app_metric_sampling_rate"`
	AppPort               int     `mapstructure:"app_port"`

	// MySQL configuration
	MysqlDbName            string `mapstructure:"mysql_db_name"`
	MysqlMasterHost        string `mapstructure:"mysql_master_host"`
	MysqlMasterMaxPoolSize string `mapstructure:"mysql_master_max_pool_size"`
	MysqlMasterMinPoolSize string `mapstructure:"mysql_master_min_pool_size"`
	MysqlMasterPassword    string `mapstructure:"mysql_master_password"`
	MysqlMasterPort        int    `mapstructure:"mysql_master_port"`
	MysqlMasterUsername    string `mapstructure:"mysql_master_username"`
	MysqlSlaveHost         string `mapstructure:"mysql_slave_host"`
	MysqlSlaveMaxPoolSize  string `mapstructure:"mysql_slave_max_pool_size"`
	MysqlSlaveMinPoolSize  string `mapstructure:"mysql_slave_min_pool_size"`
	MysqlSlavePassword     string `mapstructure:"mysql_slave_password"`
	MysqlSlavePort         int    `mapstructure:"mysql_slave_port"`
	MysqlSlaveUsername     string `mapstructure:"mysql_slave_username"`

	// Etcd configuration
	EtcdPassword       string `mapstructure:"etcd_password"`
	EtcdServer         string `mapstructure:"etcd_server"`
	EtcdUsername       string `mapstructure:"etcd_username"`
	EtcdWatcherEnabled bool   `mapstructure:"etcd_watcher_enabled"`

	// Ringmaster configuration
	RingmasterApiKey        string `mapstructure:"ringmaster_api_key"`
	RingmasterAuthorization string `mapstructure:"ringmaster_authorization"`
	RingmasterBaseUrl       string `mapstructure:"ringmaster_base_url"`
	RingmasterEnvironment   string `mapstructure:"ringmaster_environment"`
	RingmasterMiscSession   string `mapstructure:"ringmaster_misc_session"`

	// Slack configuration
	SlackCcTags       string `mapstructure:"slack_cc_tags"`
	SlackChannel      string `mapstructure:"slack_channel"`
	SlackInactiveDays int    `mapstructure:"slack_inactive_days"`
	SlackWebhookUrl   string `mapstructure:"slack_webhook_url"`

	// Vmselect configuration
	VmselectApiKey       string `mapstructure:"vmselect_api_key"`
	VmselectBaseUrl      string `mapstructure:"vmselect_base_url"`
	VmselectStartDaysAgo int    `mapstructure:"vmselect_start_days_ago"`

	// Horizon configuration
	HorizonAppName string `mapstructure:"horizon_app_name"`

	// Other configurations
	DefaultCpuThreshold string `mapstructure:"default_cpu_threshold"`
	DefaultGpuThreshold string `mapstructure:"default_gpu_threshold"`
	DefaultModelPath    string `mapstructure:"default_model_path"`

	FeatureGroupDataTypeMappingUrl string `mapstructure:"feature_group_data_type_mapping_url"`

	GcsModelBucket   string `mapstructure:"gcs_model_bucket"`
	GcsModelBasePath string `mapstructure:"gcs_model_base_path"`
	GcsEnabled       bool   `mapstructure:"gcs_enabled"`

	GrafanaBaseUrl string `mapstructure:"grafana_base_url"`

	HostUrlSuffix string `mapstructure:"host_url_suffix"`

	NumerixAppName       string `mapstructure:"numerix_app_name"`
	NumerixMonitoringUrl string `mapstructure:"numerix_monitoring_url"`

	MaxNumerixInactiveAge   int `mapstructure:"max_numerix_inactive_age"`
	MaxInferflowInactiveAge int `mapstructure:"max_inferflow_inactive_age"`
	MaxPredatorInactiveAge  int `mapstructure:"max_predator_inactive_age"`

	InferflowAppName string `mapstructure:"inferflow_app_name"`

	OnlineFeatureMappingUrl string `mapstructure:"online_feature_mapping_url"`

	PhoenixServerBaseUrl string `mapstructure:"phoenix_server_base_url"`

	PredatorMonitoringUrl string `mapstructure:"predator_monitoring_url"`
	ScheduledCronExpression string `mapstructure:"scheduled_cron_expression"`

	TestDeployableID    int `mapstructure:"test_deployable_id"`
	TestGpuDeployableID int `mapstructure:"test_gpu_deployable_id"`

	// Pricing Feature Retrieval Service configuration
	PricingFeatureRetrievalBatchSize           string `mapstructure:"pricing_feature_retrieval_batch_size"`
	PricingFeatureRetrievalDialTimeout         string `mapstructure:"pricing_feature_retrieval_dial_timeout_ms"`
	PricingFeatureRetrievalHost                string `mapstructure:"pricing_feature_retrieval_host"`
	PricingFeatureRetrievalIdleConnTimeout     string `mapstructure:"pricing_feature_retrieval_idle_conn_timeout_ms"`
	PricingFeatureRetrievalMaxIdleConns        string `mapstructure:"pricing_feature_retrieval_max_idle_conns"`
	PricingFeatureRetrievalMaxIdleConnsPerHost string `mapstructure:"pricing_feature_retrieval_max_idle_conns_per_host"`
	PricingFeatureRetrievalPort                string `mapstructure:"pricing_feature_retrieval_port"`
	PricingFeatureRetrievalGrpcPlainText       bool   `mapstructure:"pricing_feature_retrieval_grpc_plain_text"`
	PricingFeatureRetrievalTimeoutMs           string `mapstructure:"pricing_feature_retrieval_timeout_in_ms"`

	OnlineFeatureStoreAppName     string `mapstructure:"online_feature_store_app_name"`
	ScyllaActiveConfIds           string `mapstructure:"scylla_active_conf_ids"`
	RedisFailoverActiveConfIds    string `mapstructure:"redis_failover_active_conf_ids"`
	DistributedCacheActiveConfIds string `mapstructure:"distributed_cache_active_conf_ids"`
	InMemoryCacheActiveConfIds    string `mapstructure:"in_memory_cache_active_conf_ids"`

	IsMeeshoEnabled     bool `mapstructure:"is_meesho_enabled"`
	IsDummyModelEnabled bool `mapstructure:"is_dummy_model_enabled"`
}

type DynamicConfigs struct{}
