package scylla

import (
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
)

var (
	appConfig config.Configs
	initOnce  sync.Once
)

func Init(config config.Configs) {
	initOnce.Do(func() {
		appConfig = config
	})
}

func NewDatabase() Database {
	return InitScyllaDb()
}
