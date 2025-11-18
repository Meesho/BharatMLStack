package inmemorycache

import (
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/go-core/inmemorycache"
)

var InMemoryCacheInstance inmemorycache.InMemoryCache

func InitInMemoryCache() {

	inmemorycache.Init(1)
	InMemoryCacheInstance = inmemorycache.Instance()
	logger.Info("In-Memory Cache instance created.")
}
