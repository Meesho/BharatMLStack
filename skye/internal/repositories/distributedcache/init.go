package distributedcache

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

func NewRepository(version int) Database {
	switch version {
	case DefaultVersion:
		return initRedisCache()
	default:
		return nil
	}
}
