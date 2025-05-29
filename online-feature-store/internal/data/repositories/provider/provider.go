package provider

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/caches"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/stores"
)

type StoreProvider interface {
	GetStore(storeId string) (stores.Store, error)
	UpdateStore() error
}

type CacheProvider interface {
	GetCache(entityLabel string) (caches.Cache, error)
	updateCacheMapping() error
}
