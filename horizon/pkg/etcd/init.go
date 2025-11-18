package etcd

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/rs/zerolog/log"
)

var (
	instances         map[string]Etcd
	envEtcdServer     string
	envEtcdUsername   string
	envEtcdPassword   string
	envWatcherEnabled bool
	initEtcdOnce      sync.Once
)

// Init initializes the Etcd client, to be called from main.go
func InitFromAppName(config interface{}, appName string, appConfig configs.Configs) {
	initEtcdOnce.Do(func() {
		envEtcdServer = appConfig.EtcdServer
		envEtcdUsername = appConfig.EtcdUsername
		envEtcdPassword = appConfig.EtcdPassword
		envWatcherEnabled = appConfig.EtcdWatcherEnabled
	})
	if instances == nil {
		instances = make(map[string]Etcd)
	}

	if instances[appName] == nil {
		instances[appName] = newV1EtcdFromAppName(config, appName)
	}
}

func InitMPEtcdFromRegistry(config interface{}, appConfig configs.Configs) {
	initEtcdOnce.Do(func() {
		envEtcdServer = appConfig.EtcdServer
		envEtcdUsername = appConfig.EtcdUsername
		envEtcdPassword = appConfig.EtcdPassword
		envWatcherEnabled = appConfig.EtcdWatcherEnabled
	})

	if instances == nil {
		instances = make(map[string]Etcd)
	}

	appName := "inferflow"
	if instances[appName] == nil {
		instances[appName] = newV1EtcdFromCustomPath(config, "/config", appName)
	}
}

// Instance returns the Etcd client instance. Ensure that Init is called before calling this function
func Instance() map[string]Etcd {
	if instances == nil {
		log.Panic().Msg("etcd client not initialized, call Init first")
	}
	return instances
}

// SetMockInstance sets the mock instance of Etcd client
// This would be handy in places where we are directly using Etcd as etcd.Instance()
func SetMockInstance(mock map[string]Etcd) {
	instances = mock
}
