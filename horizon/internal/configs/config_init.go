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

const (
	MeeshoEnabled = "MEESHO_ENABLED"
)

// InitConfig initializes configuration based on MEESHO_ENABLED flag
func InitConfig(configHolder ConfigHolder) {
	viper.AutomaticEnv()
	isMeeshoEnabled := viper.GetBool(MeeshoEnabled)

	if isMeeshoEnabled {
		log.Println("Initializing using Meesho cacConfig")
		initUsingCacConfig(configHolder)
	} else {
		log.Println("Initializing using environment variables")
		initUsingEnvConfig(configHolder)
	}
}

func initUsingEnvConfig(configHolder ConfigHolder) {
	config.InitEnv()

	staticConfig := configHolder.GetStaticConfig()
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
