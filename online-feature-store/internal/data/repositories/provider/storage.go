package provider

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/stores"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"time"
)

var (
	StorageProviderImpl *StorageProvider
)

const (
	scylla             = "scylla"
	redisCluster       = "redis_cluster"
	redisStandalone    = "redis_standalone"
	redisFailover      = "redis_failover"
	refreshInterval    = 60 * time.Second
	refreshIntervalEnv = "STORE_REFRESH_INTERVAL"
	storeWatchPath     = "/storage/stores"
)

type StorageProvider struct {
	StoreIdRegistry map[string]*StorageMetadata
	configManager   config.Manager
}

type StorageMetadata struct {
	DbType string
	hash   string
	store  stores.Store
}

func (sp *StorageProvider) GetStore(storeId string) (stores.Store, error) {
	storeMeta, exists := sp.StoreIdRegistry[storeId]
	if !exists {
		return nil, fmt.Errorf("store with ID %s not found", storeId)
	}
	return storeMeta.store, nil
}

func (sp *StorageProvider) UpdateStore() error {
	err := loadStores(sp.StoreIdRegistry, sp.configManager)
	if err != nil {
		return err
	}
	return nil
}

func InitStorageProvider(configManager config.Manager, etcD etcd.Etcd) error {
	StoreIdRegistry := make(map[string]*StorageMetadata)
	err := loadStores(StoreIdRegistry, configManager)
	if err != nil {
		return err
	}
	StorageProviderImpl = &StorageProvider{
		StoreIdRegistry: StoreIdRegistry,
		configManager:   configManager,
	}
	err = etcD.RegisterWatchPathCallback(storeWatchPath, StorageProviderImpl.UpdateStore)
	if err != nil {
		return err
	}
	return nil
}

func loadStores(storeIdRegistry map[string]*StorageMetadata, configManager config.Manager) error {
	storeConfigs, err := configManager.GetStores()
	if err != nil {
		log.Panic().Msgf("Error getting store configs: %v", err)
	}
	for storeId, sConfig := range *storeConfigs {
		hash, err := getHash(sConfig)
		if err != nil {
			return err
		}
		if meta, ok := storeIdRegistry[storeId]; ok && meta.hash == hash {
			log.Debug().Msgf("Store %s already loaded, no change detected... skipping reload", storeId)
			continue
		}
		switch sConfig.DbType {
		case scylla:
			{
				err := loadScylla(storeIdRegistry, sConfig, storeId, hash)
				if err != nil {
					return err
				}
			}
		case redisFailover:
			{
				err := loadStorageRedisFailover(storeIdRegistry, sConfig, storeId, hash)
				if err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("store type %s not supported", sConfig.DbType)
		}

	}
	return nil

}

func loadScylla(StoreIdRegistry map[string]*StorageMetadata, sConfig config.Store, storeId string, hash string) error {
	connFacade, err := infra.Scylla.GetConnection(sConfig.ConfId)
	if err != nil {
		return err
	}
	conn := connFacade.(*infra.ScyllaClusterConnection)
	repository, err2 := stores.NewScyllaStore(sConfig.Table, conn)
	if err2 != nil {
		return err2
	}
	StoreIdRegistry[storeId] = &StorageMetadata{
		DbType: sConfig.DbType,
		hash:   hash,
		store:  repository,
	}
	return nil
}

func loadStorageRedisFailover(storeIdRegistry map[string]*StorageMetadata, sConfig config.Store, storeId string, hash string) error {
	connFacade, err := infra.RedisFailover.GetConnection(sConfig.ConfId)
	if err != nil {
		return err
	}

	conn := connFacade.(*infra.RedisFailoverConnection)
	repository, err2 := stores.NewRedisStore(conn)

	if err2 != nil {
		return err2
	}

	storeIdRegistry[storeId] = &StorageMetadata{
		DbType: sConfig.DbType,
		hash:   hash,
		store:  repository,
	}
	return nil
}

func getHash(store config.Store) (string, error) {
	jsonBytes, err := json.Marshal(store)
	if err != nil {
		log.Error().Msgf("Error marshalling store config: %v", err)
		return "", err
	}
	hash := sha1.Sum(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

func periodicRefresh(storeIdRegistry map[string]*StorageMetadata, configManager config.Manager) {
	interval := refreshInterval
	if viper.IsSet(refreshIntervalEnv) {
		interval = time.Duration(viper.GetInt(refreshIntervalEnv)) * time.Second
	}
	ticker := time.NewTicker(interval)
	defer func() {
		ticker.Stop()
		if r := recover(); r != nil {
			log.Error().Msgf("Panic recovered: %v", r)
			metric.Count("orion.store-refresh.panic.count", 1, nil)
		}
	}()
	for range ticker.C {
		err := loadStores(storeIdRegistry, configManager)
		if err != nil {
			log.Error().Msgf("Error refreshing stores: %v", err)
		}
	}

}
