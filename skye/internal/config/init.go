package config

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
)

var (
	manager        Manager
	once           sync.Once
	DefaultVersion = 1
	appName        string
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		appName = structs.GetAppConfig().Configs.AppName
	})
}

func NewManager(version int) Manager {
	switch version {
	case DefaultVersion:
		return initSkyeManager()
	default:
		return nil
	}
}

func SetInstance(provider Manager) {
	manager = provider
	once.Do(func() {})
}
