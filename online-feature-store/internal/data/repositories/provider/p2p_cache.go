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
	P2PCacheProviderImpl *P2PCacheProvider
)

const (
	p2PCacheCacheWatchPath = "/entities"
)

type P2PCacheProvider struct {
	caches         map[int]caches.Cache
	entityCacheMap map[string]int
	configManager  config.Manager
}

func (d *P2PCacheProvider) GetCache(entityLabel string) (caches.Cache, error) {
	if cacheId, exists := d.entityCacheMap[entityLabel]; exists {
		if cache, exists := d.caches[cacheId]; exists {
			return cache, nil
		}
	}
	return nil, fmt.Errorf("cache not found for entity %s", entityLabel)
}

func (d *P2PCacheProvider) updateCacheMapping() error {
	err := createEntityP2PCacheMap(d.configManager, d.caches, d.entityCacheMap)
	if err != nil {
		log.Error().Err(err).Msg("Error updating cache mapping")
		return err
	}
	return nil
}

func InitializeP2PCacheProvider(configManager config.Manager, etcD etcd.Etcd) error {
	cacheMap := make(map[int]caches.Cache)
	err := loadP2PCache(cacheMap)
	if err != nil {
		return err
	}
	entityCacheMap := make(map[string]int)
	err = createEntityP2PCacheMap(configManager, cacheMap, entityCacheMap)
	if err != nil {
		return err
	}
	p2pcp := &P2PCacheProvider{
		caches:         cacheMap,
		entityCacheMap: entityCacheMap,
		configManager:  configManager,
	}
	P2PCacheProviderImpl = p2pcp
	err = etcD.RegisterWatchPathCallback(p2PCacheCacheWatchPath, p2pcp.updateCacheMapping)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for p2p cache")
	}
	return nil
}

func createEntityP2PCacheMap(configManager config.Manager, confIdCacheMap map[int]caches.Cache, entityCacheMap map[string]int) error {
	for _, entity := range configManager.GetAllEntities() {
		dc, err := configManager.GetP2PCacheConfForEntity(entity.Label)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting p2p cache conf for entity %s", entity.Label)
			return err
		}
		confId, entityCacheMapped := entityCacheMap[entity.Label]
		if dc == nil {
			if entityCacheMapped {
				delete(entityCacheMap, entity.Label)
			}
			continue
		}
		if confIdCacheMap[dc.ConfId] == nil {
			log.Error().Msgf("P2P Cache not found for entity %s", entity.Label)
			return fmt.Errorf("P2P cache not found for entity %s", entity.Label)
		}

		if !entityCacheMapped || (entityCacheMapped && confId != dc.ConfId) {
			entityCacheMap[entity.Label] = dc.ConfId
		}
	}
	return nil
}

func loadP2PCache(cacheMap map[int]caches.Cache) error {
	for _, configId := range infra.P2PCacheLoadedConfigIds {
		connFacade, err := infra.P2PCache.GetConnection(configId)
		if err != nil {
			log.Error().Err(err).Msg("Error getting p2p cache connection")
			return err
		}
		conn := connFacade.(*infra.P2PCacheConnection)
		cache, err := caches.NewP2PCache(conn)
		if err != nil {
			log.Error().Err(err).Msg("Error creating p2p cache")
			return err
		}
		cacheMap[configId] = cache
	}
	return nil
}
