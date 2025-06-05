package config

import (
	"sync"
)

var (
	registry = make(map[int]Manager)
	once     sync.Once
)

const (
	DefaultVersion = 1
)

func Init() {
	once.Do(func() {
		registry[DefaultVersion] = NewEtcdConfig()
	})
}

func Instance(version int) Manager {
	switch version {
	case DefaultVersion:
		return registry[DefaultVersion]
	default:
		return nil
	}
}
