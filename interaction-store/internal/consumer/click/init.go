package click

import (
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
)

var (
	DefaultVersion = 1
	initOnce       sync.Once
)

func Init(_ config.Configs) {
	initOnce.Do(func() {
		// Configuration initialized
	})
}

func NewConsumer(version int) Consumer {
	switch version {
	case DefaultVersion:
		return newClickConsumer()
	default:
		return nil
	}
}
