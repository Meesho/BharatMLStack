package provider

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/caches"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/rs/zerolog/log"
)

var (
	InMemoryCacheProviderImpl *InMemoryCacheProvider
)

const (
	inMemoryCacheCacheWatchPath = "/entities"
)

type InMemoryCacheProvider struct {
	caches         map[int]caches.Cache
	entityCacheMap map[string]int
	configManager  config.Manager
}

func (i *InMemoryCacheProvider) GetCache(entityLabel string) (caches.Cache, error) {
	if cacheId, exists := i.entityCacheMap[entityLabel]; exists {
		if cache, exists := i.caches[cacheId]; exists {
			return cache, nil
		}
	}
	return nil, fmt.Errorf("cache not found for entity %s", entityLabel)
}

func (i *InMemoryCacheProvider) updateCacheMapping() error {
	err := createEntityIMCacheMap(i.configManager, i.caches, i.entityCacheMap)
	if err != nil {
		log.Error().Err(err).Msg("Error updating cache mapping")
		return err
	}
	return nil
}

func InitializeInMemoryCacheProvider(configManager config.Manager, etcD etcd.Etcd) error {
	cacheMap := make(map[int]caches.Cache)
	err := loadIMCache(cacheMap)
	if err != nil {
		return err
	}
	entityCacheMap := make(map[string]int)
	err = createEntityIMCacheMap(configManager, cacheMap, entityCacheMap)
	if err != nil {
		return err
	}
	imcp := &InMemoryCacheProvider{
		caches:         cacheMap,
		entityCacheMap: entityCacheMap,
		configManager:  configManager,
	}
	InMemoryCacheProviderImpl = imcp
	err = etcD.RegisterWatchPathCallback(inMemoryCacheCacheWatchPath, imcp.updateCacheMapping)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for in-memory cache")
	}
	return nil
}

func createEntityIMCacheMap(configManager config.Manager, confIdCacheMap map[int]caches.Cache, entityCacheMap map[string]int) error {
	for _, entity := range configManager.GetAllEntities() {
		dc, err := configManager.GetInMemoryCacheConfForEntity(entity.Label)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting distributed cache conf for entity %s", entity.Label)
			return err
		}
		confId, entityCacheMapped := entityCacheMap[entity.Label]
		if dc == nil && entityCacheMapped {
			delete(entityCacheMap, entity.Label)
			continue
		}
		if confIdCacheMap[dc.ConfId] == nil {
			log.Error().Msgf("In Memory Cache not found for entity %s", entity.Label)
			return fmt.Errorf("in memory cache not found for entity %s", entity.Label)
		}

		//case 1: entity not mapped to any cache
		if !entityCacheMapped && dc != nil {
			entityCacheMap[entity.Label] = dc.ConfId
		} else if entityCacheMapped && confId != dc.ConfId {
			entityCacheMap[entity.Label] = dc.ConfId
		} else {
			continue
		}
	}
	return nil
}

func loadIMCache(cacheMap map[int]caches.Cache) error {
	for _, configId := range infra.InMemoryCacheLoadedConfigIds {
		connFacade, err := infra.InMemoryCache.GetConnection(configId)
		if err != nil {
			log.Error().Err(err).Msg("Error getting in memory connection")
			return err
		}
		conn := connFacade.(*infra.InMemoryCacheConnection)
		cache, err := caches.NewInMemoryCache(conn)
		if err != nil {
			log.Error().Err(err).Msg("Error creating in memory cache")
			return err
		}
		cacheMap[configId] = cache
	}
	return nil
}
