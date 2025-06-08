package onfs

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	Version1 = 1
)

var (
	registry = make(map[int]Client)
	mut      sync.Mutex
)

// InitClient initialises the client for the given version
func InitClient(version int, conf *Config, timing func(name string, value time.Duration, tags []string), count func(name string, value int64, tags []string)) Client {
	mut.Lock()
	defer mut.Unlock()
	if registry[version] != nil {
		log.Panic().Msgf("Client for version %d already initialised", version)
	}
	switch version {
	case Version1:
		registry[version] = NewClientV1(conf, timing, count)
	}
	return registry[version]
}

// GetInstance returns the client for the given version
func GetInstance(version int) Client {
	if registry[version] == nil {
		log.Panic().Msgf("Client for version %d not initialised", version)
	}
	return registry[version]
}
