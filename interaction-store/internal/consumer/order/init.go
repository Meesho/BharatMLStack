package order

import (
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
)

var (
	DefaultVersion = 1
	appConfig      config.Configs
	initOnce       sync.Once
)

func Init(config config.Configs) {
	initOnce.Do(func() {
		appConfig = config
	})
}

func NewConsumer(version int) Consumer {
	switch version {
	case DefaultVersion:
		return newOrderConsumer()
	default:
		return nil
	}
}
