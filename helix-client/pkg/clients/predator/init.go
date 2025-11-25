package predator

import (
	"sync"

	"github.com/rs/zerolog/log"
)

const (
	Version1 = 1
)

var (
	registry = make(map[int]Client)
	onceMap  = make(map[int]*sync.Once)
)

// It panics if the client is already initialised
func InitClient(version int, conf *Config) Client {
	// Ensure a `sync.Once` instance exists for the given version
	if _, exists := onceMap[version]; !exists {
		onceMap[version] = &sync.Once{}
	}

	onceMap[version].Do(func() {
		if registry[version] != nil {
			log.Panic().Msgf("Client for version %d already initialised", version)
		}
		registry[version] = NewClientV1(conf)
	})
	return registry[version]
}

func GetInstance(version int) Client {
	if registry[version] == nil {
		log.Panic().Msgf("Client for version %d not initialised", version)
	}
	return registry[version]
}
