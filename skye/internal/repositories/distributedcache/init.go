package distributedcache

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

func NewRepository(version int) Database {
	switch version {
	case DefaultVersion:
		return initRedisCache()
	default:
		return nil
	}
}
