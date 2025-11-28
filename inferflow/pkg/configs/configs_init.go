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

	// Bind environment variables to config keys
	// This maps APP_NAME (env) -> app_name (config key)
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("Failed to unmarshal config from environment: %v", err)
	}

	log.Println("Configuration loaded from environment variables")
}
