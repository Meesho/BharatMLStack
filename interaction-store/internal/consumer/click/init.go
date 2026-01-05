package click

import (
	"sync"

	"github.com/Meesho/interaction-store/internal/config"
)

var (
	DefaultVersion = 1
	appConfig      config.Configs
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		appConfig = config.GetAppConfig().Configs
	})
}

func NewConsumer(version int) Consumer {
	switch version {
	case DefaultVersion:
		return newClickConsumer()
	default:
		return nil
	}
}
