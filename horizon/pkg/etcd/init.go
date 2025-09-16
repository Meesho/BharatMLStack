package etcd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	instances map[string]Etcd
)

// Init initializes the Etcd client, to be called from main.go
func Init(config interface{}) {
	appName := viper.GetString("APP_NAME")
	if instances == nil {
		instances = make(map[string]Etcd)
		if instances[appName] == nil {
			instances[appName] = newV1Etcd(config)
		}
	}
}

// InitFromAppName initializes the Etcd client, to be called from main.go
func InitFromAppName(config interface{}, appName string) {
	if instances == nil {
		instances = make(map[string]Etcd)
	}

	if instances[appName] == nil {
		instances[appName] = newV1EtcdFromAppName(config, appName)
	}
}

// InitV1 initializes the Etcd client with version 1
func InitV1(config interface{}) {
	Init(config)
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
