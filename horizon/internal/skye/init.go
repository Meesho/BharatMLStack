package skye

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	initSkyeOnce          sync.Once
	SkyeAppName           string
	AppEnv                string
	ScyllaActiveConfigIds string
)

func Init(config configs.Configs) {
	initSkyeOnce.Do(func() {
		SkyeAppName = config.SkyeAppName
		AppEnv = config.AppEnv
		ScyllaActiveConfigIds = config.ScyllaActiveConfigIds
	})
}
