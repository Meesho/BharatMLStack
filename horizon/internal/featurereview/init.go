package featurereview

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var initOnce sync.Once

func Init(config configs.Configs) {
	initOnce.Do(func() {
		// Configuration is loaded lazily via viper in route.Init()
	})
}
