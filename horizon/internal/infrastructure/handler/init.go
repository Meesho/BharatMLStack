package handler

import (
	"sync"
)

var (
	infraHandler InfrastructureHandler
	initOnce     sync.Once
)

func InitInfrastructureHandler() InfrastructureHandler {
	initOnce.Do(func() {
		infraHandler = NewInfrastructureHandler()
	})
	return infraHandler
}
