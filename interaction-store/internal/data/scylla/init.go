package scylla

import (
	"sync"

	"github.com/Meesho/interaction-store/internal/config"
)

var (
	appConfig config.Configs
	initOnce  sync.Once
)

func Init() {
	initOnce.Do(func() {
		appConfig = config.GetAppConfig().Configs
	})
}

func NewDatabase() Database {
	return InitScyllaDb()
}
