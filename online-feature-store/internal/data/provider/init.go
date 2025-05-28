package provider

import (
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/spf13/viper"
)

var (
	mut sync.Mutex
)

const (
	inMemoryActiveConfIds = "IN_MEM_CACHE_ACTIVE_CONFIG_IDS"
	p2PActiveConfIds      = "P2P_CACHE_ACTIVE_CONFIG_IDS"
)

func InitProvider(configManager config.Manager, etcD etcd.Etcd) {
	mut.Lock()
	defer mut.Unlock()
	if StorageProviderImpl == nil {
		err := InitStorageProvider(configManager, etcD)
		if err != nil {
			panic(err)
		}
	}

	if DistributedCacheProviderImpl == nil && viper.IsSet(distributedCacheConfIds) {
		err := InitializeDistributedCacheProvider(configManager, etcD)
		if err != nil {
			panic(err)
		}
	}

	if InMemoryCacheProviderImpl == nil && viper.IsSet(inMemoryActiveConfIds) {
		err := InitializeInMemoryCacheProvider(configManager, etcD)
		if err != nil {
			panic(err)
		}
	}

	if P2PCacheProviderImpl == nil && viper.IsSet(p2PActiveConfIds) {
		err := InitializeP2PCacheProvider(configManager, etcD)
		if err != nil {
			panic(err)
		}
	}
}
