package config

import "sync"

var (
	registry = make(map[int]Manager)
	once     sync.Once
)

const (
	DefaultVersion = 1
)

func InitEtcDBridge() {
	once.Do(func() {
		registry[DefaultVersion] = NewEtcdConfig()
	})
}

func Instance(version int) Manager {
	switch version {
	case DefaultVersion:
		return registry[DefaultVersion]
	default:
		return registry[DefaultVersion]
	}
}
