package configs

type Configs struct {
	ApplicationEnv      string `mapstructure:"app_env"`
	ApplicationLogLevel string `mapstructure:"app_log_level"`
	ApplicationName     string `mapstructure:"app_name"`
	ApplicationPort     int    `mapstructure:"app_port"`
	AppGcPercentage     int    `mapstructure:"app_gc_percentage"`

	//in-memory-cache-config
	InMemoryCacheSizeInBytes string `mapstructure:"in-memory-cache_size-in-bytes"`

	//zookeeper-config
	ZookeeperServer string `mapstructure:"zookeeper_server"`

	//dag-topology-executor-config
	DagTopologyCacheTTL  int64 `mapstructure:"dagTopologyCache_ttlSec"`
	DagTopologyCacheSize int64 `mapstructure:"dagTopologyCache_cacheSize"`

	//telegraf-config
	MetricsSamplingRate string `mapstructure:"metrics_sampling_rate"`
	Telegraf_Host       string `mapstructure:"telegraf_host"`
	Telegraf_Port       string `mapstructure:"telegraf_port"`

	//predator-config
	ExternalServicePredator string `mapstructure:"externalServicePredator"`

	//onFs
	ExternalServiceOnFs_Host          string `mapstructure:"externalServiceOnFs_fsHost"`
	ExternalServiceOnFs_Port          string `mapstructure:"externalServiceOnFs_fsPort"`
	ExternalServiceOnFs_GrpcPlainText bool   `mapstructure:"externalServiceOnFs_fsGrpcPlainText"`
	ExternalServiceOnFs_CallerID      string `mapstructure:"externalServiceOnFs_fsCallerId"`
	ExternalServiceOnFs_CallerToken   string `mapstructure:"externalServiceOnFs_fsCallerToken"`
	ExternalServiceOnFs_DeadLine      int    `mapstructure:"externalServiceOnFs_fsdeadLine"`
	ExternalServiceOnFs_BatchSize     int    `mapstructure:"externalServiceOnFs_fsBatchSize"`

	//prism-client-config
	ExternalServicePrism_Host      string `mapstructure:"externalServicePrism_prismHost"`
	ExternalServicePrism_EventMqId string `mapstructure:"externalServicePrism_prismEventMqId"`
	ExternalServicePrism_Port      string `mapstructure:"externalServicePrism_prismPort"`
	ExternalServicePrism_Timeout   string `mapstructure:"externalServicePrism_timeout"`
	ExternalServicePrism_Username  string `mapstructure:"externalServicePrism_username"`
	ExternalServicePrism_Password  string `mapstructure:"externalServicePrism_password"`

	PricingFeatureRetrievalService_DialTimeout         string `mapstructure:"pricingFeatureRetrievalService_dialTimeoutInMs"`
	PricingFeatureRetrievalService_Host                string `mapstructure:"pricingFeatureRetrievalService_host"`
	PricingFeatureRetrievalService_IdleConnTimeout     string `mapstructure:"pricingFeatureRetrievalService_idleConnTimeoutInMs"`
	PricingFeatureRetrievalService_MaxIdleConns        string `mapstructure:"pricingFeatureRetrievalService_maxIdleConns"`
	PricingFeatureRetrievalService_MaxIdleConnsPerHost string `mapstructure:"pricingFeatureRetrievalService_maxIdleConnsPerHost"`
	PricingFeatureRetrievalService_Port                string `mapstructure:"pricingFeatureRetrievalService_port"`
	PricingFeatureRetrievalService_GrpcPlainText       bool   `mapstructure:"pricingFeatureRetrievalService_grpcPlainText"`
	PricingFeatureRetrievalService_Timeout             string `mapstructure:"pricingFeatureRetrievalService_timeoutInMs"`

	KV_Host          string `mapstructure:"kv_host"`
	KV_Port          string `mapstructure:"kv_port"`
	KV_DeadLineMs    int    `mapstructure:"kv_deadlineMs"`
	KV_GrpcPlainText bool   `mapstructure:"kv_grpcPlainText"`

	ETCD_WATCHER_ENABLED bool   `mapstructure:"etcd_watcherEnabled"`
	ETCD_SERVER          string `mapstructure:"etcd_server"`

	NumerixClientV1_Host          string `mapstructure:"numerixClientV1_host"`
	NumerixClientV1_Port          int    `mapstructure:"numerixClientV1_port"`
	NumerixClientV1_Deadline      int    `mapstructure:"numerixClientV1_deadlineMs"`
	NumerixClientV1_GrpcPlainText bool   `mapstructure:"numerixClientV1_plainText"`
	NumerixClientV1_AuthToken     string `mapstructure:"numerixClientV1_authToken"`
	NumerixClientV1_BatchSize     int    `mapstructure:"numerixClientV1_batchSize"`
}

type DynamicConfigs struct {
}

type AppConfigs struct {
	Configs        Configs
	DynamicConfigs DynamicConfigs
}

func (a *AppConfigs) GetStaticConfig() interface{} {
	return &a.Configs
}

func (a *AppConfigs) GetDynamicConfig() interface{} {
	return &a.Configs
}
