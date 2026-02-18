package embedding

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/pkg/ds"
)

var (
	queryCache     *ds.SyncMap[string, string]
	embeddingStore Store
	once           sync.Once
	DefaultVersion = 1
	appConfig      structs.Configs
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		appConfig = structs.GetAppConfig().Configs
	})
}

func NewRepository(version int) Store {
	switch version {
	case DefaultVersion:
		return initEmbeddingStore()
	default:
		return nil
	}
}

func SetInstance(provider Store) {
	embeddingStore = provider
	once.Do(func() {}) // Marking the sync once as done
}
