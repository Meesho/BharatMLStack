//go:build !meesho

package configs

import (
	"log"

	"github.com/spf13/viper"
)

func InitConfig(appConfigs *AppConfigs) {
	InitEnv()

	staticConfig := appConfigs.GetStaticConfig()
	cfg, ok := staticConfig.(*Configs)
	if !ok {
		log.Fatal("Failed to cast static config to *Configs")
	}

	// Manually bind environment variables to mapstructure keys
	// This ensures proper mapping from env vars to struct fields
	bindEnvVars()

	// Bind environment variables to config keys
	// This maps environment variables to struct fields using mapstructure tags
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("Failed to unmarshal config from environment: %v", err)
	}

	log.Println("Configuration loaded from environment variables")
}

func bindEnvVars() {
	// Application config
	viper.BindEnv("app_env", "APP_ENV")
	viper.BindEnv("app_log_level", "APP_LOG_LEVEL")
	viper.BindEnv("app_name", "APP_NAME")
	viper.BindEnv("app_port", "APP_PORT")
	viper.BindEnv("app_gc_percentage", "APP_GC_PERCENTAGE")

	// In-memory cache config
	viper.BindEnv("in-memory-cache_size-in-bytes", "IN_MEMORY_CACHE_SIZE_IN_BYTES")

	// DAG topology config
	viper.BindEnv("dagTopologyCache_cacheSize", "DAG_TOPOLOGY_CACHE_SIZE")
	viper.BindEnv("dagTopologyCache_ttlSec", "DAG_TOPOLOGY_CACHE_TTL_SEC")

	// ETCD config
	viper.BindEnv("etcd_watcherEnabled", "ETCD_WATCHER_ENABLED")
	viper.BindEnv("etcd_server", "ETCD_SERVER")
	viper.BindEnv("etcd_username", "ETCD_USERNAME")
	viper.BindEnv("etcd_password", "ETCD_PASSWORD")

	// Numerix client config
	viper.BindEnv("numerixClientV1_host", "NUMERIX_CLIENT_V1_HOST")
	viper.BindEnv("numerixClientV1_port", "NUMERIX_CLIENT_V1_PORT")
	viper.BindEnv("numerixClientV1_deadlineMs", "NUMERIX_CLIENT_V1_DEADLINE_MS")
	viper.BindEnv("numerixClientV1_plainText", "NUMERIX_CLIENT_V1_PLAINTEXT")
	viper.BindEnv("numerixClientV1_authToken", "NUMERIX_CLIENT_V1_AUTHTOKEN")
	viper.BindEnv("numerixClientV1_batchSize", "NUMERIX_CLIENT_V1_BATCHSIZE")

	// ONFS config
	viper.BindEnv("externalServiceOnFs_fsHost", "EXTERNAL_SERVICE_ONFS_FS_HOST")
	viper.BindEnv("externalServiceOnFs_fsPort", "EXTERNAL_SERVICE_ONFS_FS_PORT")
	viper.BindEnv("externalServiceOnFs_fsGrpcPlainText", "EXTERNAL_SERVICE_ONFS_FS_GRPC_PLAIN_TEXT")
	viper.BindEnv("externalServiceOnFs_fsCallerId", "EXTERNAL_SERVICE_ONFS_FS_CALLER_ID")
	viper.BindEnv("externalServiceOnFs_fsCallerToken", "EXTERNAL_SERVICE_ONFS_FS_CALLER_TOKEN")
	viper.BindEnv("externalServiceOnFs_fsdeadLine", "EXTERNAL_SERVICE_ONFS_FS_DEAD_LINE")
	viper.BindEnv("externalServiceOnFs_fsBatchSize", "EXTERNAL_SERVICE_ONFS_FS_BATCH_SIZE")

	// Predator config
	viper.BindEnv("externalServicePredator_Port", "EXTERNAL_SERVICE_PREDATOR_PORT")
	viper.BindEnv("externalServicePredator_GrpcPlainText", "EXTERNAL_SERVICE_PREDATOR_GRPC_PLAIN_TEXT")
	viper.BindEnv("externalServicePredator_CallerId", "EXTERNAL_SERVICE_PREDATOR_CALLER_ID")
	viper.BindEnv("externalServicePredator_CallerToken", "EXTERNAL_SERVICE_PREDATOR_CALLER_TOKEN")
	viper.BindEnv("externalServicePredator_Deadline", "EXTERNAL_SERVICE_PREDATOR_DEADLINE")

	// Metrics / Telegraf config
	viper.BindEnv("metrics_sampling_rate", "METRIC_SAMPLING_RATE")
	viper.BindEnv("telegraf_host", "TELEGRAF_HOST")
	viper.BindEnv("telegraf_port", "TELEGRAF_PORT")

	// Kafka inference logging config
	viper.BindEnv("kafka_bootstrapServers", "KAFKA_BOOTSTRAP_SERVERS")
	viper.BindEnv("kafka_v2LogTopic", "KAFKA_V2_LOG_TOPIC")
}
