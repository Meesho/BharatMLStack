package config

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
	AppName             string `mapstructure:"app_name"`
	AppEnv              string `mapstructure:"app_env"`
	Port                int    `mapstructure:"port"`
	ClickConsumerMqId   int    `mapstructure:"click_consumer_mq_id"`
	OrderConsumerMqId   int    `mapstructure:"order_consumer_mq_id"`
	ScyllaContactPoints string `mapstructure:"storage_scylla_contact_points"`
	ScyllaKeyspace      string `mapstructure:"storage_scylla_keyspace"`
	ScyllaNumConns      int    `mapstructure:"storage_scylla_num_conns"`
	ScyllaPort          int    `mapstructure:"storage_scylla_port"`
	ScyllaTimeoutMs     int    `mapstructure:"storage_scylla_timeout_ms"`
}

type DynamicConfigs struct {
}
