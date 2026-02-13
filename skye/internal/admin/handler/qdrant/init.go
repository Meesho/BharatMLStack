package qdrant

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
)

var (
	DefaultVersion = 1
	appConfig      structs.Configs
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		appConfig = structs.GetAppConfig().Configs
	})
}

func NewHandler(version int) Db {
	switch version {
	case DefaultVersion:
		return initQdrantHandler()
	default:
		return nil
	}
}
