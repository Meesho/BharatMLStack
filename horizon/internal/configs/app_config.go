package configs

type Configs struct {
	// App configuration
	AppName               string  `mapstructure:"app_name"`
	AppEnv                string  `mapstructure:"app_env"`
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

	GcsModelBucket    string `mapstructure:"gcs_model_bucket"`
	GcsModelBasePath  string `mapstructure:"gcs_model_base_path"`
	GcsConfigBasePath string `mapstructure:"gcs_config_base_path"`
	GcsConfigBucket   string `mapstructure:"gcs_config_bucket"`

	GrafanaBaseUrl string `mapstructure:"grafana_base_url"`

	HostUrlSuffix string `mapstructure:"host_url_suffix"`

	NumerixAppName       string `mapstructure:"numerix_app_name"`
	NumerixMonitoringUrl string `mapstructure:"numerix_monitoring_url"`

	BulkDeletePredatorEnabled                  bool `mapstructure:"bulk_delete_predator_enabled"`
	BulkDeleteInferflowEnabled                 bool `mapstructure:"bulk_delete_inferflow_enabled"`
	BulkDeleteNumerixEnabled                   bool `mapstructure:"bulk_delete_numerix_enabled"`
	BulkDeletePredatorMaxInactiveDays          int  `mapstructure:"bulk_delete_predator_max_inactive_days"`
	BulkDeleteInferflowMaxInactiveDays         int  `mapstructure:"bulk_delete_inferflow_max_inactive_days"`
	BulkDeleteNumerixMaxInactiveDays           int  `mapstructure:"bulk_delete_numerix_max_inactive_days"`
	BulkDeletePredatorRequestSubmissionEnabled bool `mapstructure:"bulk_delete_predator_request_submission_enabled"`

	InferflowAppName string `mapstructure:"inferflow_app_name"`

	SkyeAppName string `mapstructure:"skye_app_name"`

	PhoenixServerBaseUrl string `mapstructure:"phoenix_server_base_url"`

	PredatorMonitoringUrl   string `mapstructure:"predator_monitoring_url"`
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

	ArgoCDAPI   string `mapstructure:"argocd_api"`
	ArgoCDToken string `mapstructure:"argocd_token"`

	WorkingEnv string `mapstructure:"working_env"`

	GitHubAppID          int64  `mapstructure:"github_app_id"`
	GitHubInstallationID int64  `mapstructure:"github_installation_id"`
	GitHubPrivateKey     string `mapstructure:"github_private_key"`
	GitHubOwner          string `mapstructure:"github_owner"`
	LocalModelPath       string `mapstructure:"local_model_path"`

	ServiceConfigSource string `mapstructure:"service_config_source"`
	ServiceConfigRepo   string `mapstructure:"service_config_repo"`
	ServiceConfigPath   string `mapstructure:"service_config_path"`
	GitHubCommitAuthor  string `mapstructure:"github_commit_author"`
	GitHubCommitEmail   string `mapstructure:"github_commit_email"`

	GitHubHelmChartRepo      string `mapstructure:"github_helm_chart_repo"`
	GitHubInfraHelmChartRepo string `mapstructure:"github_infra_helm_chart_repo"`
	GitHubArgoRepo           string `mapstructure:"github_argo_repo"`

	VictoriaMetricsServerAddress string `mapstructure:"victoriametrics_server_address"`

	GitHubBranchPrd    string `mapstructure:"github_branch_prd"`
	GitHubBranchGcpPrd string `mapstructure:"github_branch_gcp_prd"`
	GitHubBranchInt    string `mapstructure:"github_branch_int"`
	GitHubBranchGcpInt string `mapstructure:"github_branch_gcp_int"`
	GitHubBranchDev    string `mapstructure:"github_branch_dev"`
	GitHubBranchGcpDev string `mapstructure:"github_branch_gcp_dev"`
	GitHubBranchGcpStg string `mapstructure:"github_branch_gcp_stg"`
	GitHubBranchFtr    string `mapstructure:"github_branch_ftr"`
	GitHubBranchGcpFtr string `mapstructure:"github_branch_gcp_ftr"`

	IsMeeshoEnabled bool `mapstructure:"is_meesho_enabled"`

	RepositoryName string `mapstructure:"repository_name"`
	BranchName     string `mapstructure:"branch_name"`

	// DNS API configuration (for Meesho builds only)
	DNSAPIBaseURL string `mapstructure:"dns_api_base_url"`
	DNSAPIKey     string `mapstructure:"dns_api_key"`

	ScyllaActiveConfigIds string `mapstructure:"scylla_active_config_ids"`
	Scylla1ContactPoints  string `mapstructure:"scylla_1_contact_points"`
	Scylla1Port           int    `mapstructure:"scylla_1_port"`
	Scylla1Keyspace       string `mapstructure:"scylla_1_keyspace"`
	Scylla1Username       string `mapstructure:"scylla_1_username"`
	Scylla1Password       string `mapstructure:"scylla_1_password"`

	PrismBaseUrl                    string `mapstructure:"prism_base_url"`
	PrismAppUserID                  string `mapstructure:"prism_app_user_id"`
	AirflowBaseUrl                  string `mapstructure:"airflow_base_url"`
	AirflowUsername                 string `mapstructure:"airflow_username"`
	AirflowPassword                 string `mapstructure:"airflow_password"`
	InitialIngestionPrismJobID      int    `mapstructure:"initial_ingestion_prism_job_id"`
	InitialIngestionPrismStepID     int    `mapstructure:"initial_ingestion_prism_step_id"`
	InitialIngestionAirflowDAGID    string `mapstructure:"initial_ingestion_airflow_dag_id"`
	VariantScaleUpPrismJobID        int    `mapstructure:"variant_scaleup_prism_job_id"`
	VariantScaleUpPrismStepID       int    `mapstructure:"variant_scaleup_prism_step_id"`
	VariantScaleUpAirflowDAGID      string `mapstructure:"variant_scaleup_airflow_dag_id"`
	VariantOnboardingCronExpression string `mapstructure:"variant_onboarding_cron_expression"`
	VariantScaleUpCronExpression    string `mapstructure:"variant_scaleup_cron_expression"`
	MQIdTopicsMapping               string `mapstructure:"mq_id_topics_mapping"`
	VariantsList                    string `mapstructure:"variants_list"`

	SkyeScyllaActiveConfigIds    string `mapstructure:"skye_scylla_active_config_ids"`
	Scylla2ContactPoints         string `mapstructure:"scylla_2_contact_points"`
	Scylla2Port                  int    `mapstructure:"scylla_2_port"`
	Scylla2Keyspace              string `mapstructure:"scylla_2_keyspace"`
	Scylla2Username              string `mapstructure:"scylla_2_username"`
	Scylla2Password              string `mapstructure:"scylla_2_password"`
	SkyeNumberOfPartitions       int    `mapstructure:"skye_number_of_partitions"`
	SkyeFailureProducerMqId      int    `mapstructure:"skye_failure_producer_mq_id"`
	HorizonToSkyeScyllaConfIdMap string `mapstructure:"horizon_to_skye_scylla_conf_id_map"`

	SkyeHost             string `mapstructure:"skye_host"`
	SkyePort             string `mapstructure:"skye_port"`
	SkyeAuthToken        string `mapstructure:"skye_auth_token"`
	SkyeDeadlineExceedMS int    `mapstructure:"skye_deadline_exceed_ms"`
}

type DynamicConfigs struct{}
