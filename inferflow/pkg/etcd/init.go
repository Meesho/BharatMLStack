package etcd

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
)

var (
	instance       Etcd
	DefaultVersion = 1
)

// Init initializes the Etcd client, to be called from main.go
func Init(version int, config interface{}, configs *configs.AppConfigs) {
	once.Do(func() {
		switch version {
		case DefaultVersion:
			instance = newV1Etcd(config, configs)
		default:
			logger.Panic(fmt.Sprintf("invalid version %d", version), nil)
		}
	})
}

// Instance returns the Etcd client instance. Ensure that Init is called before calling this function
func Instance() Etcd {
	if instance == nil {
		logger.Panic("etcd client not initialized, call Init first", nil)
	}
	return instance
}
