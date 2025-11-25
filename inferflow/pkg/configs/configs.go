//go:build !meesho

package configs

type Configs struct {
	ApplicationEnv      string `mapstructure:"app_env"`
	ApplicationLogLevel string `mapstructure:"app_log_level"`
	ApplicationName     string `mapstructure:"app_name"`
	ApplicationPort     int    `mapstructure:"app_port"`
	AppGcPercentage     int    `mapstructure:"app_gc_percentage"`

	//in-memory-cache-config
	InMemoryCacheSizeInBytes string `mapstructure:"in-memory-cache_size-in-bytes"`

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
