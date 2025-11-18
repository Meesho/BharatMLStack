package config

import (
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
)

const (
	EnvDelimiter    string = "__"
	ConfigDelimiter string = "."

	// this key is used in EKS, don't change / delete
	TelegrafUdpHost string = "TELEGRAF_UDP_HOST"
	TelegrafUdpPort string = "TELEGRAF_UDP_PORT"
)

var (
	// Can be concurrently accessed if not being watched
	AppConfig *koanf.Koanf
)

// env delimiter is "__" (double underscore) for nested config
func InitConfig() {
	AppConfig = koanf.New(ConfigDelimiter)

	// load default configs
	AppConfig.Load(confmap.Provider(map[string]interface{}{
		TelegrafUdpHost: "localhost",
		TelegrafUdpPort: "8125",
	}, ConfigDelimiter), nil)

	// load env variable configs which will override default config values
	err := AppConfig.Load(env.Provider("", EnvDelimiter, nil), nil)
	if err != nil {
		logger.Panic("Error occurred while loading environment variables!", err)
	}
}
