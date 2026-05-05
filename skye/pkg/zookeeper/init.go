package zookeeper

import (
	"github.com/rs/zerolog/log"
)

var (
	instance ZK
)

// Init initializes the zookeeper client, to be called from main.go
func Init(version int, config interface{}) {
	once.Do(func() {
		switch version {
		case 1:
			instance = newV1ZK(config)
		case 2:
			// Deprecated: Version 2 is in experimental phase. Please use version 1 instead.
			instance = newV2ZK(config)
		default:
			log.Panic().Msgf("invalid version %d", version)
		}
	})
}

// InitV1 initializes the zookeeper client with version 1
func InitV1(config interface{}) {
	Init(1, config)
}

// Deprecated: InitV2 is part of the experimental V2 implementation. Please use InitV1 instead.
func InitV2(config interface{}) {
	Init(2, config)
}

// Instance returns the zookeeper client instance. Ensure that Init is called before calling this function
func Instance() ZK {
	if instance == nil {
		log.Panic().Msg("zookeeper client not initialized, call Init first")
	}
	return instance
}

// SetMockInstance sets the mock instance of zookeeper client
// This would be handy in places where we are directly using zookeeper as zookeeper.Instance()
func SetMockInstance(mock ZK) {
	instance = mock
}
