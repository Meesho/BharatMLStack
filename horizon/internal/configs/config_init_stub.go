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

// InitConfig initializes configuration for open-source builds
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
	viper.BindEnv("gcs_enabled", "GCS_ENABLED")

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

	viper.BindEnv("argocd_api", "ARGOCD_API")
	viper.BindEnv("argocd_token", "ARGOCD_TOKEN")

	viper.BindEnv("working_env", "WORKING_ENV")

	viper.BindEnv("github_app_id", "GITHUB_APP_ID")
	viper.BindEnv("github_installation_id", "GITHUB_INSTALLATION_ID")
	viper.BindEnv("github_private_key_path", "GITHUB_PRIVATE_KEY_PATH")
	viper.BindEnv("github_owner", "GITHUB_OWNER")
	viper.BindEnv("github_commit_author", "GITHUB_COMMIT_AUTHOR")
	viper.BindEnv("github_commit_email", "GITHUB_COMMIT_EMAIL")

	viper.BindEnv("github_helm_chart_repo", "GITHUB_HELM_CHART_REPO")
	viper.BindEnv("github_infra_helm_chart_repo", "GITHUB_INFRA_HELM_CHART_REPO")
	viper.BindEnv("github_argo_repo", "GITHUB_ARGO_REPO")

	viper.BindEnv("victoriametrics_server_address", "VICTORIAMETRICS_SERVER_ADDRESS")

	viper.BindEnv("github_branch_prd", "GITHUB_BRANCH_PRD")
	viper.BindEnv("github_branch_gcp_prd", "GITHUB_BRANCH_GCP_PRD")
	viper.BindEnv("github_branch_int", "GITHUB_BRANCH_INT")
	viper.BindEnv("github_branch_gcp_int", "GITHUB_BRANCH_GCP_INT")
	viper.BindEnv("github_branch_dev", "GITHUB_BRANCH_DEV")
	viper.BindEnv("github_branch_gcp_dev", "GITHUB_BRANCH_GCP_DEV")
	viper.BindEnv("github_branch_gcp_stg", "GITHUB_BRANCH_GCP_STG")
	viper.BindEnv("github_branch_ftr", "GITHUB_BRANCH_FTR")
	viper.BindEnv("github_branch_gcp_ftr", "GITHUB_BRANCH_GCP_FTR")

	viper.BindEnv("repository_name", "REPOSITORY_NAME")
	viper.BindEnv("branch_name", "BRANCH_NAME")

	viper.BindEnv("service_config_source", "SERVICE_CONFIG_SOURCE")
	viper.BindEnv("service_config_repo", "SERVICE_CONFIG_REPO")
	viper.BindEnv("service_config_path", "SERVICE_CONFIG_PATH")

	// Meesho flags
	viper.BindEnv("is_meesho_enabled", "MEESHO_ENABLED")
	viper.BindEnv("is_dummy_model_enabled", "IS_DUMMY_MODEL_ENABLED")

	// DNS API configuration
	viper.BindEnv("dns_api_base_url", "DNS_API_BASE_URL")
	viper.BindEnv("dns_api_key", "DNS_API_KEY")

	viper.BindEnv("scylla_active_config_ids", "SCYLLA_ACTIVE_CONFIG_IDS")
	viper.BindEnv("scylla_1_contact_points", "SCYLLA_1_CONTACT_POINTS")
	viper.BindEnv("scylla_1_port", "SCYLLA_1_PORT")
	viper.BindEnv("scylla_1_keyspace", "SCYLLA_1_KEYSPACE")
	viper.BindEnv("scylla_1_username", "SCYLLA_1_USERNAME")
	viper.BindEnv("scylla_1_password", "SCYLLA_1_PASSWORD")
}
