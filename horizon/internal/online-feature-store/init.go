package onlinefeaturestore

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	initOnlineFeatureStoreOnce       sync.Once
	ScyllaActiveConfIdsStr           string
	RedisFailoverActiveConfIdsStr    string
	OnlineFeatureStoreAppName        string
	DistributedCacheActiveConfIdsStr string
	InMemoryCacheActiveConfIdsStr    string
	AppEnv                           string
)

func Init(config configs.Configs) {
	initOnlineFeatureStoreOnce.Do(func() {
		ScyllaActiveConfIdsStr = config.ScyllaActiveConfIds
		RedisFailoverActiveConfIdsStr = config.RedisFailoverActiveConfIds
		OnlineFeatureStoreAppName = config.OnlineFeatureStoreAppName
		AppEnv = config.AppEnv
		DistributedCacheActiveConfIdsStr = config.DistributedCacheActiveConfIds
		InMemoryCacheActiveConfIdsStr = config.InMemoryCacheActiveConfIds
	})
}
