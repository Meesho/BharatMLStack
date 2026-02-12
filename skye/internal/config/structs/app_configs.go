package structs

var (
	appConfig AppConfig
)

type AppConfig struct {
	Configs        Configs
	DynamicConfigs DynamicConfigs
}

func (cfg *AppConfig) GetStaticConfig() interface{} {
	return &cfg.Configs
}

func (cfg *AppConfig) GetDynamicConfig() interface{} {
	return &cfg.DynamicConfigs
}
func GetAppConfig() *AppConfig {
	return &appConfig
}

type Configs struct {
	AppName                        string `mapstructure:"app_name"`
	AppEnv                         string `mapstructure:"app_env"`
	AuthTokens                     string `mapstructure:"auth_tokens"`
	CollectionMetricEnabled        bool   `mapstructure:"collection_metric_enabled"`
	CollectionMetricPublish        int    `mapstructure:"collection_metric_publish"`
	EmbeddingConsumerMqIds         string `mapstructure:"embedding_consumer_mq_ids"`
	EmbeddingConsumerSequenceMqIds string `mapstructure:"embedding_consumer_sequence_mq_ids"`
	RealtimeConsumerMqIds          string `mapstructure:"realtime_consumer_mq_ids"`
	RealtimeProducerMqId           int    `mapstructure:"realtime_producer_mq_id"`
	RealTimeDeltaProducerMqId      int    `mapstructure:"realtime_delta_producer_mq_id"`
	RealTimeDeltaConsumerMqId      int    `mapstructure:"realtime_delta_consumer_mq_id"`
	EtcdUsername                   string `mapstructure:"etcd_username"`
	EtcdPassword                   string `mapstructure:"etcd_password"`
	EtcdServer                     string `mapstructure:"etcd_server"`
	EtcdWatcherEnabled             bool   `mapstructure:"etcd_watcher_enabled"`
	McacheId                       int    `mapstructure:"mcache_id"`
	ModelStateConsumer             int    `mapstructure:"model_state_consumer"`
	ModelStateProducer             int    `mapstructure:"model_state_producer"`
	Port                           int    `mapstructure:"port"`
	StagingDefaultEmbeddingLength  int    `mapstructure:"staging_default_embedding_length"`
	StagingDefaultModelName        string `mapstructure:"staging_default_model_name"`
	StagingDefaultVariant          string `mapstructure:"staging_default_variant"`
	StagingDefaultEntity           string `mapstructure:"staging_default_entity"`
	StorageAggregatorDbCount       int    `mapstructure:"storage_aggregator_db_count"`
	StorageEmbeddingStoreCount     int    `mapstructure:"storage_embedding_store_count"`
}

type DynamicConfigs struct {
}
