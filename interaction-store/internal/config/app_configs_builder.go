//go:build !meesho

package config

import (
	"log"

	"github.com/Meesho/BharatMLStack/interaction-store/pkg/config"
	"github.com/spf13/viper"
)

type ConfigHolder interface {
	GetStaticConfig() interface{}
	GetDynamicConfig() interface{}
}

func InitConfig(configHolder ConfigHolder) {
	config.InitEnv()
	staticConfig := configHolder.GetStaticConfig()
	cfg, ok := staticConfig.(*Configs)
	if !ok {
		log.Fatal("Failed to cast static config to *Configs")
	}
	bindEnvVars()
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("Failed to unmarshal config from environment: %v", err)
	}
}

func bindEnvVars() {
	// App configuration
	viper.BindEnv("app_name", "APP_NAME")
	viper.BindEnv("app_env", "APP_ENV")
	viper.BindEnv("app_log_level", "APP_LOG_LEVEL")
	viper.BindEnv("app_metric_sampling_rate", "APP_METRIC_SAMPLING_RATE")
	viper.BindEnv("app_port", "APP_PORT")

	// Profiling configuration
	viper.BindEnv("profiling_enabled", "PROFILING_ENABLED")
	viper.BindEnv("profiling_port", "PROFILING_PORT")

	// Storage configuration
	viper.BindEnv("storage_scylla_contact_points", "STORAGE_SCYLLA_CONTACT_POINTS")
	viper.BindEnv("storage_scylla_keyspace", "STORAGE_SCYLLA_KEYSPACE")
	viper.BindEnv("storage_scylla_num_conns", "STORAGE_SCYLLA_NUM_CONNS")
	viper.BindEnv("storage_scylla_port", "STORAGE_SCYLLA_PORT")
	viper.BindEnv("storage_scylla_timeout_ms", "STORAGE_SCYLLA_TIMEOUT_MS")
}
