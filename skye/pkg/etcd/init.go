package etcd

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/rs/zerolog/log"
)

var (
	instances           map[string]Etcd
	appName             string
	etcdServers         string
	username            string
	password            string
	watcherEnabled      bool
	initOnce            sync.Once
	initOnceFromAppName sync.Once
)

// Init initializes the Etcd client, to be called from main.go
func Init(config interface{}, appConfig structs.Configs) {
	initOnce.Do(func() {
		appName = appConfig.AppName
		etcdServers = appConfig.EtcdServer
		username = appConfig.EtcdUsername
		password = appConfig.EtcdPassword
		watcherEnabled = appConfig.EtcdWatcherEnabled
	})
	if instances == nil {
		instances = make(map[string]Etcd)
		if instances[appName] == nil {
			instances[appName] = newV1Etcd(config)
		}
	}
}

// InitFromAppName initializes the Etcd client, to be called from main.go
func InitFromAppName(config interface{}, AppName string, appConfig structs.Configs) {
	initOnceFromAppName.Do(func() {
		appName = AppName
		etcdServers = appConfig.EtcdServer
		username = appConfig.EtcdUsername
		password = appConfig.EtcdPassword
		watcherEnabled = appConfig.EtcdWatcherEnabled

	})
	if instances == nil {
		instances = make(map[string]Etcd)
		if instances[appName] == nil {
			instances[appName] = newV1EtcdFromAppName(config, appName)
		}
	}
}

// InitV1 initializes the Etcd client with version 1
func InitV1(config interface{}, appConfig structs.Configs) {
	Init(config, appConfig)
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
