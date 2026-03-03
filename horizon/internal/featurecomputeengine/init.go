package featurecomputeengine

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	initOnce sync.Once
	AppEnv   string
)

// Init stores FCE configuration. Repository creation and route registration
// happen lazily in route.Init(), which must be called after infra and
// httpframework are initialized.
func Init(config configs.Configs) {
	initOnce.Do(func() {
		AppEnv = config.AppEnv
	})
}
