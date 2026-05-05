package numerix

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	initnumerixOnce      sync.Once
	NumerixAppName       string
	AppEnv               string
	NumerixMonitoringUrl string
)

func Init(config configs.Configs) {
	initnumerixOnce.Do(func() {
		NumerixAppName = config.NumerixAppName
		AppEnv = config.AppEnv
		NumerixMonitoringUrl = config.NumerixMonitoringUrl
	})
}
