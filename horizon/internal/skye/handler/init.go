package handler

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	configOnce sync.Once
	config     Config
)

func NewConfigHandler(version int, appConfig configs.Configs) Config {
	switch version {
	case 1:
		return InitV1ConfigHandler(appConfig)
	default:
		return nil
	}
}
