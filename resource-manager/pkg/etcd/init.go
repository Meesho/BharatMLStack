package etcd

import (
	"github.com/rs/zerolog/log"
)

var (
	instance       Etcd
	DefaultVersion = 1
)

// Init initializes the Etcd client, to be called from main.go
func Init(version int, config interface{}) {
	once.Do(func() {
		switch version {
		case DefaultVersion:
			instance = newV1Etcd(config)
			bootstrapIfEnabled(instance)
		default:
			log.Panic().Msgf("invalid version %d", version)
		}
	})
}

// InitV1 initializes the Etcd client with version 1
func InitV1(config interface{}) {
	Init(1, config)
}

// Instance returns the Etcd client instance. Ensure that Init is called before calling this function
func Instance() Etcd {
	if instance == nil {
		log.Panic().Msg("etcd client not initialized, call Init first")
	}
	return instance
}

// SetMockInstance sets the mock instance of Etcd client
// This would be handy in places where we are directly using Etcd as etcd.Instance()
func SetMockInstance(mock Etcd) {
	instance = mock
}
