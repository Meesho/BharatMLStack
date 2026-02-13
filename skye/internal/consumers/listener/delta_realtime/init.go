package delta_realtime

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
)

var (
	DefaultVersion = 1
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		_ = structs.GetAppConfig().Configs
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
