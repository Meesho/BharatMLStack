package delta_realtime

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
)

var (
	DefaultVersion = 1
	appConfig      structs.Configs
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		appConfig = structs.GetAppConfig().Configs
	})
}

func NewConsumer(version int) Consumer {
	switch version {
	case DefaultVersion:
		return newRealTimeDeltaConsumer()
	default:
		return nil
	}
}
