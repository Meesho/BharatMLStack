//go:build !meesho

package configs

import (
	"log"

	"github.com/Meesho/BharatMLStack/horizon/pkg/config"
	"github.com/spf13/viper"
)

// ConfigHolder interface for app config
type ConfigHolder interface {
	GetStaticConfig() interface{}
	GetDynamicConfig() interface{}
}

// InitConfig initializes configuration based on MEESHO_ENABLED flag
func InitConfig(configHolder ConfigHolder) {
	config.InitEnv()

	staticConfig := configHolder.GetStaticConfig()
	cfg, ok := staticConfig.(*Configs)
	if !ok {
		log.Fatal("Failed to cast static config to *Configs")
	}

	// Bind environment variables explicitly
	// Viper needs explicit binding to map MYSQL_MASTER_HOST -> mysql_master_host
	bindEnvVars()

	// Bind environment variables to config keys
	// This maps APP_NAME (env) -> app_name (config key)
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("Failed to unmarshal config from environment: %v", err)
	}

	log.Println("Configuration loaded from environment variables")
	//print the config
	log.Println("Config: ", cfg)
	log.Println("IsMeeshoEnabled: ", cfg.IsMeeshoEnabled)
	log.Println("OnlineFeatureStoreAppName: ", cfg.OnlineFeatureStoreAppName)
	log.Println("ScyllaActiveConfIds: ", cfg.ScyllaActiveConfIds)
	log.Println("RedisFailoverActiveConfIds: ", cfg.RedisFailoverActiveConfIds)
	log.Println("HorizonAppName: ", cfg.HorizonAppName)
	log.Println("FeatureGroupDataTypeMappingUrl: ", cfg.FeatureGroupDataTypeMappingUrl)
	log.Println("GcsModelBucket: ", cfg.GcsModelBucket)
	log.Println("GcsModelBasePath: ", cfg.GcsModelBasePath)
	log.Println("GrafanaBaseUrl: ", cfg.GrafanaBaseUrl)
	log.Println("HostUrlSuffix: ", cfg.HostUrlSuffix)
	log.Println("NumerixAppName: ", cfg.NumerixAppName)
	log.Println("NumerixMonitoringUrl: ", cfg.NumerixMonitoringUrl)
	log.Println("MaxNumerixInactiveAge: ", cfg.MaxNumerixInactiveAge)
	log.Println("MaxInferflowInactiveAge: ", cfg.MaxInferflowInactiveAge)
	log.Println("MaxPredatorInactiveAge: ", cfg.MaxPredatorInactiveAge)
	log.Println("InferflowAppName: ", cfg.InferflowAppName)
	log.Println("OnlineFeatureMappingUrl: ", cfg.OnlineFeatureMappingUrl)
	log.Println("PhoenixServerBaseUrl: ", cfg.PhoenixServerBaseUrl)
	log.Println("PredatorMonitoringUrl: ", cfg.PredatorMonitoringUrl)
	log.Println("ScheduledCronExpression: ", cfg.ScheduledCronExpression)
	log.Println("TestDeployableID: ", cfg.TestDeployableID)
	log.Println("TestGpuDeployableID: ", cfg.TestGpuDeployableID)
	log.Println("PricingFeatureRetrievalBatchSize: ", cfg.PricingFeatureRetrievalBatchSize)
	log.Println("PricingFeatureRetrievalDialTimeout: ", cfg.PricingFeatureRetrievalDialTimeout)
	log.Println("PricingFeatureRetrievalHost: ", cfg.PricingFeatureRetrievalHost)
	log.Println("PricingFeatureRetrievalIdleConnTimeout: ", cfg.PricingFeatureRetrievalIdleConnTimeout)
	log.Println("PricingFeatureRetrievalMaxIdleConns: ", cfg.PricingFeatureRetrievalMaxIdleConns)
	log.Println("PricingFeatureRetrievalMaxIdleConnsPerHost: ", cfg.PricingFeatureRetrievalMaxIdleConnsPerHost)
	log.Println("PricingFeatureRetrievalPort: ", cfg.PricingFeatureRetrievalPort)
	log.Println("PricingFeatureRetrievalGrpcPlainText: ", cfg.PricingFeatureRetrievalGrpcPlainText)
	log.Println("PricingFeatureRetrievalTimeoutMs: ", cfg.PricingFeatureRetrievalTimeoutMs)
	log.Println("PricingFeatureRetrievalTimeoutMs: ", cfg.PricingFeatureRetrievalTimeoutMs)
	log.Println("OnlineFeatureStoreAppName: ", cfg.OnlineFeatureStoreAppName)
	log.Println("ScyllaActiveConfIds: ", cfg.ScyllaActiveConfIds)
	log.Println("RedisFailoverActiveConfIds: ", cfg.RedisFailoverActiveConfIds)
	log.Println("MysqlDbName: ", cfg.MysqlDbName)
	log.Println("MysqlMasterHost: ", cfg.MysqlMasterHost)
	log.Println("MysqlMasterMaxPoolSize: ", cfg.MysqlMasterMaxPoolSize)
	log.Println("MysqlMasterMinPoolSize: ", cfg.MysqlMasterMinPoolSize)
	log.Println("MysqlMasterPassword: ", cfg.MysqlMasterPassword)
	log.Println("MysqlMasterPort: ", cfg.MysqlMasterPort)
	log.Println("MysqlMasterUsername: ", cfg.MysqlMasterUsername)
	log.Println("MysqlSlaveHost: ", cfg.MysqlSlaveHost)
	log.Println("MysqlSlaveMaxPoolSize: ", cfg.MysqlSlaveMaxPoolSize)
	log.Println("MysqlSlaveMinPoolSize: ", cfg.MysqlSlaveMinPoolSize)
	log.Println("MysqlSlavePassword: ", cfg.MysqlSlavePassword)
	log.Println("MysqlSlavePort: ", cfg.MysqlSlavePort)
	log.Println("MysqlSlaveUsername: ", cfg.MysqlSlaveUsername)
}

// bindEnvVars explicitly binds environment variables to Viper keys
// This is necessary because Viper's AutomaticEnv() doesn't automatically
// map uppercase env vars (MYSQL_MASTER_HOST) to lowercase keys (mysql_master_host)
func bindEnvVars() {
	// App configuration
	viper.BindEnv("app_name", "APP_NAME")
	viper.BindEnv("app_env", "APP_ENV")
	viper.BindEnv("app_gc_percentage", "APP_GC_PERCENTAGE")
	viper.BindEnv("app_log_level", "APP_LOG_LEVEL")
	viper.BindEnv("app_metric_sampling_rate", "APP_METRIC_SAMPLING_RATE")
	viper.BindEnv("app_port", "APP_PORT")

	// MySQL configuration
	viper.BindEnv("mysql_db_name", "MYSQL_DB_NAME")
	viper.BindEnv("mysql_master_host", "MYSQL_MASTER_HOST")
	viper.BindEnv("mysql_master_max_pool_size", "MYSQL_MASTER_MAX_POOL_SIZE")
	viper.BindEnv("mysql_master_min_pool_size", "MYSQL_MASTER_MIN_POOL_SIZE")
	viper.BindEnv("mysql_master_password", "MYSQL_MASTER_PASSWORD")
	viper.BindEnv("mysql_master_port", "MYSQL_MASTER_PORT")
	viper.BindEnv("mysql_master_username", "MYSQL_MASTER_USERNAME")
	viper.BindEnv("mysql_slave_host", "MYSQL_SLAVE_HOST")
	viper.BindEnv("mysql_slave_max_pool_size", "MYSQL_SLAVE_MAX_POOL_SIZE")
	viper.BindEnv("mysql_slave_min_pool_size", "MYSQL_SLAVE_MIN_POOL_SIZE")
	viper.BindEnv("mysql_slave_password", "MYSQL_SLAVE_PASSWORD")
	viper.BindEnv("mysql_slave_port", "MYSQL_SLAVE_PORT")
	viper.BindEnv("mysql_slave_username", "MYSQL_SLAVE_USERNAME")

	// Etcd configuration
	viper.BindEnv("etcd_password", "ETCD_PASSWORD")
	viper.BindEnv("etcd_server", "ETCD_SERVER")
	viper.BindEnv("etcd_username", "ETCD_USERNAME")
	viper.BindEnv("etcd_watcher_enabled", "ETCD_WATCHER_ENABLED")

	// Ringmaster configuration
	viper.BindEnv("ringmaster_api_key", "RINGMASTER_API_KEY")
	viper.BindEnv("ringmaster_authorization", "RINGMASTER_AUTHORIZATION")
	viper.BindEnv("ringmaster_base_url", "RINGMASTER_BASE_URL")
	viper.BindEnv("ringmaster_environment", "RINGMASTER_ENVIRONMENT")
	viper.BindEnv("ringmaster_misc_session", "RINGMASTER_MISC_SESSION")

	// Slack configuration
	viper.BindEnv("slack_cc_tags", "SLACK_CC_TAGS")
	viper.BindEnv("slack_channel", "SLACK_CHANNEL")
	viper.BindEnv("slack_inactive_days", "SLACK_INACTIVE_DAYS")
	viper.BindEnv("slack_webhook_url", "SLACK_WEBHOOK_URL")

	// Vmselect configuration
	viper.BindEnv("vmselect_api_key", "VMSELECT_API_KEY")
	viper.BindEnv("vmselect_base_url", "VMSELECT_BASE_URL")
	viper.BindEnv("vmselect_start_days_ago", "VMSELECT_START_DAYS_AGO")

	// Horizon configuration
	viper.BindEnv("horizon_app_name", "HORIZON_APP_NAME")

	// Other configurations
	viper.BindEnv("default_cpu_threshold", "DEFAULT_CPU_THRESHOLD")
	viper.BindEnv("default_gpu_threshold", "DEFAULT_GPU_THRESHOLD")
	viper.BindEnv("default_model_path", "DEFAULT_MODEL_PATH")
	viper.BindEnv("feature_group_data_type_mapping_url", "FEATURE_GROUP_DATA_TYPE_MAPPING_URL")
	viper.BindEnv("gcs_model_bucket", "GCS_MODEL_BUCKET")
	viper.BindEnv("gcs_model_base_path", "GCS_MODEL_BASE_PATH")
	viper.BindEnv("grafana_base_url", "GRAFANA_BASE_URL")
	viper.BindEnv("host_url_suffix", "HOST_URL_SUFFIX")
	viper.BindEnv("numerix_app_name", "NUMERIX_APP_NAME")
	viper.BindEnv("numerix_monitoring_url", "NUMERIX_MONITORING_URL")
	viper.BindEnv("max_numerix_inactive_age", "MAX_NUMERIX_INACTIVE_AGE")
	viper.BindEnv("max_inferflow_inactive_age", "MAX_INFERFLOW_INACTIVE_AGE")
	viper.BindEnv("max_predator_inactive_age", "MAX_PREDATOR_INACTIVE_AGE")
	viper.BindEnv("inferflow_app_name", "INFERFLOW_APP_NAME")
	viper.BindEnv("online_feature_mapping_url", "ONLINE_FEATURE_MAPPING_URL")
	viper.BindEnv("phoenix_server_base_url", "PHOENIX_SERVER_BASE_URL")
	viper.BindEnv("predator_monitoring_url", "PREDATOR_MONITORING_URL")
	viper.BindEnv("scheduled_cron_expression", "SCHEDULED_CRON_EXPRESSION")
	viper.BindEnv("test_deployable_id", "TEST_DEPLOYABLE_ID")
	viper.BindEnv("test_gpu_deployable_id", "TEST_GPU_DEPLOYABLE_ID")

	// Pricing Feature Retrieval Service configuration
	viper.BindEnv("pricing_feature_retrieval_batch_size", "PRICING_FEATURE_RETRIEVAL_BATCH_SIZE")
	viper.BindEnv("pricing_feature_retrieval_dial_timeout_ms", "PRICING_FEATURE_RETRIEVAL_DIAL_TIMEOUT_MS")
	viper.BindEnv("pricing_feature_retrieval_host", "PRICING_FEATURE_RETRIEVAL_HOST")
	viper.BindEnv("pricing_feature_retrieval_idle_conn_timeout_ms", "PRICING_FEATURE_RETRIEVAL_IDLE_CONN_TIMEOUT_MS")
	viper.BindEnv("pricing_feature_retrieval_max_idle_conns", "PRICING_FEATURE_RETRIEVAL_MAX_IDLE_CONNS")
	viper.BindEnv("pricing_feature_retrieval_max_idle_conns_per_host", "PRICING_FEATURE_RETRIEVAL_MAX_IDLE_CONNS_PER_HOST")
	viper.BindEnv("pricing_feature_retrieval_port", "PRICING_FEATURE_RETRIEVAL_PORT")
	viper.BindEnv("pricing_feature_retrieval_grpc_plain_text", "PRICING_FEATURE_RETRIEVAL_GRPC_PLAIN_TEXT")
	viper.BindEnv("pricing_feature_retrieval_timeout_in_ms", "PRICING_FEATURE_RETRIEVAL_TIMEOUT_IN_MS")

	// Online Feature Store configuration
	viper.BindEnv("online_feature_store_app_name", "ONLINE_FEATURE_STORE_APP_NAME")
	viper.BindEnv("scylla_active_conf_ids", "SCYLLA_ACTIVE_CONFIG_IDS")
	viper.BindEnv("redis_failover_active_conf_ids", "REDIS_FAILOVER_ACTIVE_CONFIG_IDS")

	// Meesho flag
	viper.BindEnv("is_meesho_enabled", "MEESHO_ENABLED")
}

// bindEnvVars explicitly binds environment variables to Viper keys
// This is necessary because Viper's AutomaticEnv() doesn't automatically
// map uppercase env vars (MYSQL_MASTER_HOST) to lowercase keys (mysql_master_host)
func bindEnvVars() {
	// App configuration
	viper.BindEnv("app_name", "APP_NAME")
	viper.BindEnv("app_env", "APP_ENV")
	viper.BindEnv("app_log_level", "APP_LOG_LEVEL")
	viper.BindEnv("app_metric_sampling_rate", "APP_METRIC_SAMPLING_RATE")
	viper.BindEnv("app_port", "APP_PORT")

	// MySQL configuration
	viper.BindEnv("mysql_db_name", "MYSQL_DB_NAME")
	viper.BindEnv("mysql_master_host", "MYSQL_MASTER_HOST")
	viper.BindEnv("mysql_master_max_pool_size", "MYSQL_MASTER_MAX_POOL_SIZE")
	viper.BindEnv("mysql_master_min_pool_size", "MYSQL_MASTER_MIN_POOL_SIZE")
	viper.BindEnv("mysql_master_password", "MYSQL_MASTER_PASSWORD")
	viper.BindEnv("mysql_master_port", "MYSQL_MASTER_PORT")
	viper.BindEnv("mysql_master_username", "MYSQL_MASTER_USERNAME")
	viper.BindEnv("mysql_slave_host", "MYSQL_SLAVE_HOST")
	viper.BindEnv("mysql_slave_max_pool_size", "MYSQL_SLAVE_MAX_POOL_SIZE")
	viper.BindEnv("mysql_slave_min_pool_size", "MYSQL_SLAVE_MIN_POOL_SIZE")
	viper.BindEnv("mysql_slave_password", "MYSQL_SLAVE_PASSWORD")
	viper.BindEnv("mysql_slave_port", "MYSQL_SLAVE_PORT")
	viper.BindEnv("mysql_slave_username", "MYSQL_SLAVE_USERNAME")

	// Etcd configuration
	viper.BindEnv("etcd_password", "ETCD_PASSWORD")
	viper.BindEnv("etcd_server", "ETCD_SERVER")
	viper.BindEnv("etcd_username", "ETCD_USERNAME")
	viper.BindEnv("etcd_watcher_enabled", "ETCD_WATCHER_ENABLED")

	// Ringmaster configuration
	viper.BindEnv("ringmaster_api_key", "RINGMASTER_API_KEY")
	viper.BindEnv("ringmaster_authorization", "RINGMASTER_AUTHORIZATION")
	viper.BindEnv("ringmaster_base_url", "RINGMASTER_BASE_URL")
	viper.BindEnv("ringmaster_environment", "RINGMASTER_ENVIRONMENT")
	viper.BindEnv("ringmaster_misc_session", "RINGMASTER_MISC_SESSION")

	// Slack configuration
	viper.BindEnv("slack_cc_tags", "SLACK_CC_TAGS")
	viper.BindEnv("slack_channel", "SLACK_CHANNEL")
	viper.BindEnv("slack_inactive_days", "SLACK_INACTIVE_DAYS")
	viper.BindEnv("slack_webhook_url", "SLACK_WEBHOOK_URL")

	// Vmselect configuration
	viper.BindEnv("vmselect_api_key", "VMSELECT_API_KEY")
	viper.BindEnv("vmselect_base_url", "VMSELECT_BASE_URL")
	viper.BindEnv("vmselect_start_days_ago", "VMSELECT_START_DAYS_AGO")

	// Horizon configuration
	viper.BindEnv("horizon_app_name", "HORIZON_APP_NAME")

	// Other configurations
	viper.BindEnv("default_cpu_threshold", "DEFAULT_CPU_THRESHOLD")
	viper.BindEnv("default_gpu_threshold", "DEFAULT_GPU_THRESHOLD")
	viper.BindEnv("default_model_path", "DEFAULT_MODEL_PATH")

	viper.BindEnv("gcs_model_bucket", "GCS_MODEL_BUCKET")
	viper.BindEnv("gcs_model_base_path", "GCS_MODEL_BASE_PATH")

	viper.BindEnv("grafana_base_url", "GRAFANA_BASE_URL")

	viper.BindEnv("host_url_suffix", "HOST_URL_SUFFIX")

	viper.BindEnv("numerix_app_name", "NUMERIX_APP_NAME")
	viper.BindEnv("numerix_monitoring_url", "NUMERIX_MONITORING_URL")

	viper.BindEnv("max_numerix_inactive_age", "MAX_NUMERIX_INACTIVE_AGE")
	viper.BindEnv("max_inferflow_inactive_age", "MAX_INFERFLOW_INACTIVE_AGE")
	viper.BindEnv("max_predator_inactive_age", "MAX_PREDATOR_INACTIVE_AGE")

	viper.BindEnv("inferflow_app_name", "INFERFLOW_APP_NAME")

	viper.BindEnv("phoenix_server_base_url", "PHOENIX_SERVER_BASE_URL")

	viper.BindEnv("scheduled_cron_expression", "SCHEDULED_CRON_EXPRESSION")

	viper.BindEnv("test_deployable_id", "TEST_DEPLOYABLE_ID")
	viper.BindEnv("test_gpu_deployable_id", "TEST_GPU_DEPLOYABLE_ID")

	// Online Feature Store configuration
	viper.BindEnv("online_feature_store_app_name", "ONLINE_FEATURE_STORE_APP_NAME")
	viper.BindEnv("scylla_active_conf_ids", "SCYLLA_ACTIVE_CONFIG_IDS")
	viper.BindEnv("redis_failover_active_conf_ids", "REDIS_FAILOVER_ACTIVE_CONFIG_IDS")
	viper.BindEnv("distributed_cache_active_conf_ids", "DISTRIBUTED_CACHE_ACTIVE_CONFIG_IDS")
	viper.BindEnv("in_memory_cache_active_conf_ids", "IN_MEMORY_CACHE_ACTIVE_CONFIG_IDS")

	// Meesho flag
	viper.BindEnv("is_meesho_enabled", "MEESHO_ENABLED")
	viper.BindEnv("is_dummy_model_enabled", "IS_DUMMY_MODEL_ENABLED")
}
