package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"sync"
)

var (
	initialized = false
	once        sync.Once
)

func InitEnv() {
	if initialized {
		log.Debug().Msg("Env already initialized!")
		return
	}
	once.Do(func() {
		viper.AutomaticEnv()
		initialized = true
		log.Info().Msg("Env initialized!")
	})
}
