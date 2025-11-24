package inmemorycache

import (
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
)

var InMemoryCacheInstance InMemoryCache

func InitInMemoryCache() {

	Init(1)
	InMemoryCacheInstance = Instance()
	logger.Info("In-Memory Cache instance created.")
}
