package config

import (
	"log"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/pkg/config"
	"github.com/spf13/viper"
)

func InitConfig(appConfig *structs.AppConfig) {
	config.InitEnv()
	staticConfig := appConfig.GetStaticConfig()
	cfg, ok := staticConfig.(*structs.Configs)
	if !ok {
		log.Fatal("Failed to cast static config to *Configs")
	}
	bindEnvVars()
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("Failed to unmarshal config from environment: %v", err)
	}
}

func bindEnvVars() {
	viper.BindEnv("app_name", "APP_NAME")
	viper.BindEnv("app_env", "APP_ENV")
	viper.BindEnv("auth_tokens", "AUTH_TOKENS")
	viper.BindEnv("collection_metric_enabled", "COLLECTION_METRIC_ENABLED")
	viper.BindEnv("collection_metric_publish", "COLLECTION_METRIC_PUBLISH")
	viper.BindEnv("kafka_broker", "KAFKA_BROKER")
	viper.BindEnv("kafka_group_id", "KAFKA_GROUP_ID")
	viper.BindEnv("kafka_topic", "KAFKA_TOPIC")
	viper.BindEnv("embedding_consumer_kafka_ids", "EMBEDDING_CONSUMER_KAFKA_IDS")
	viper.BindEnv("embedding_consumer_sequence_kafka_ids", "EMBEDDING_CONSUMER_SEQUENCE_KAFKA_IDS")
	viper.BindEnv("realtime_consumer_kafka_ids", "REALTIME_CONSUMER_KAFKA_IDS")
	viper.BindEnv("realtime_producer_kafka_id", "REALTIME_PRODUCER_KAFKA_ID")
	viper.BindEnv("realtime_delta_producer_kafka_id", "REALTIME_DELTA_PRODUCER_KAFKA_ID")
	viper.BindEnv("realtime_delta_consumer_kafka_id", "REALTIME_DELTA_CONSUMER_KAFKA_ID")
	viper.BindEnv("etcd_username", "ETCD_USERNAME")
	viper.BindEnv("etcd_password", "ETCD_PASSWORD")
	viper.BindEnv("etcd_server", "ETCD_SERVER")
	viper.BindEnv("etcd_watcher_enabled", "ETCD_WATCHER_ENABLED")
	viper.BindEnv("redis_addr", "REDIS_ADDR")
	viper.BindEnv("redis_password", "REDIS_PASSWORD")
	viper.BindEnv("redis_db", "REDIS_DB")
	viper.BindEnv("model_state_consumer", "MODEL_STATE_CONSUMER")
	viper.BindEnv("model_state_producer", "MODEL_STATE_PRODUCER")
	viper.BindEnv("port", "PORT")
	viper.BindEnv("staging_default_embedding_length", "STAGING_DEFAULT_EMBEDDING_LENGTH")
	viper.BindEnv("staging_default_model_name", "STAGING_DEFAULT_MODEL_NAME")
	viper.BindEnv("staging_default_variant", "STAGING_DEFAULT_VARIANT")
	viper.BindEnv("staging_default_entity", "STAGING_DEFAULT_ENTITY")
	viper.BindEnv("storage_aggregator_db_count", "STORAGE_AGGREGATOR_DB_COUNT")
	viper.BindEnv("storage_embedding_store_count", "STORAGE_EMBEDDING_STORE_COUNT")
}
