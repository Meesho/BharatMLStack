package provider

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/caches"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	distributedCacheConfIds   = "DISTRIBUTED_CACHE_CONF_IDS"
	distributedCacheWatchPath = "/entities"
)

var (
	DistributedCacheProviderImpl *DistributedCacheProvider
)

type DistributedCacheProvider struct {
	entityCacheMap map[string]int
	caches         map[int]*CacheMetadata
	configManager  config.Manager
}

type CacheMetadata struct {
	CacheType infra.DBType
	hash      string
	Cache     caches.Cache
}

func (d *DistributedCacheProvider) GetCache(entityLabel string) (caches.Cache, error) {
	if cacheId, exists := d.entityCacheMap[entityLabel]; exists {
		if cache, exists := d.caches[cacheId]; exists {
			return cache.Cache, nil
		}
	}
	return nil, fmt.Errorf("cache not found for entity %s", entityLabel)
}

func (d *DistributedCacheProvider) updateCacheMapping() error {
	err := createEntityCacheMap(d.configManager, d.caches, d.entityCacheMap)
	if err != nil {
		log.Error().Err(err).Msg("Error updating cache mapping")
		return err
	}
	return nil
}

func InitializeDistributedCacheProvider(configManager config.Manager, etcD etcd.Etcd) error {
	confIdCacheMap := make(map[int]*CacheMetadata)
	err := loadCaches(confIdCacheMap)
	if err != nil {
		return err
	}
	entityCacheMap := make(map[string]int)
	err = createEntityCacheMap(configManager, confIdCacheMap, entityCacheMap)
	if err != nil {
		return err
	}
	dcp := &DistributedCacheProvider{
		entityCacheMap: entityCacheMap,
		caches:         confIdCacheMap,
		configManager:  configManager,
	}
	DistributedCacheProviderImpl = dcp

	err = etcD.RegisterWatchPathCallback(distributedCacheWatchPath, dcp.updateCacheMapping)
	if err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for dist cache")
	}
	return nil
}

func loadCaches(confIdCacheMap map[int]*CacheMetadata) error {
	cacheConfIdStr := viper.GetString(distributedCacheConfIds)
	cacheConfIds := strings.Split(cacheConfIdStr, ",")
	for _, confIdStr := range cacheConfIds {
		configId, err := strconv.Atoi(confIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing cache config id")
			return err
		}
		if _, ok := infra.ConfIdDBTypeMap[configId]; !ok {
			log.Error().Msg("Invalid cache config id")
			return fmt.Errorf("invalid cache config id %d", configId)
		}
		dbType := infra.ConfIdDBTypeMap[configId]
		switch dbType {
		case infra.DBTypeRedisCluster:
			err2 := loadRedisCluster(configId, confIdCacheMap)
			if err2 != nil {
				return err2
			}
		case infra.DBTypeRedisStandalone:
			err2 := loadRedisStandalone(configId, confIdCacheMap)
			if err2 != nil {
				return err2
			}
		case infra.DBTypeRedisFailover:
			err2 := loadRedisFailover(configId, confIdCacheMap)
			if err2 != nil {
				return err2
			}
		default:
			log.Error().Msg("Invalid cache type")
			return fmt.Errorf("invalid cache type %s", dbType)
		}
	}
	return nil
}

func createEntityCacheMap(configManager config.Manager, confIdCacheMap map[int]*CacheMetadata, entityCacheMap map[string]int) error {
	for _, entity := range configManager.GetAllEntities() {
		dc, err := configManager.GetDistributedCacheConfForEntity(entity.Label)
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
			log.Error().Msgf("Distributed Cache not found for entity %s", entity.Label)
			return fmt.Errorf("distributed cache not found for entity %s", entity.Label)
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

func loadRedisFailover(configId int, confIdCacheMap map[int]*CacheMetadata) error {
	conFacade, err := infra.RedisFailover.GetConnection(configId)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis failover connection ")
		return err
	}
	conn := conFacade.(*infra.RedisFailoverConnection)
	cache, err2 := caches.NewRedisCacheFromRedisFailoverConnection(conn)
	if err2 != nil {
		log.Error().Err(err2).Msg("Error creating redis failover cache")
		return err2
	}
	confIdCacheMap[configId] = &CacheMetadata{
		CacheType: infra.DBTypeRedisFailover,
		hash:      "",
		Cache:     cache,
	}
	return nil
}

func loadRedisStandalone(configId int, confIdCacheMap map[int]*CacheMetadata) error {
	conFacade, err := infra.RedisStandalone.GetConnection(configId)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis standalone connection")
		return err
	}
	conn := conFacade.(*infra.RedisStandaloneConnection)
	cache, err2 := caches.NewRedisCacheFromRedisStandaloneConnection(conn)
	if err2 != nil {
		log.Error().Err(err2).Msg("Error creating redis standalone cache")
		return err2
	}
	confIdCacheMap[configId] = &CacheMetadata{
		CacheType: infra.DBTypeRedisStandalone,
		hash:      "",
		Cache:     cache,
	}
	return nil
}

func loadRedisCluster(configId int, confIdCacheMap map[int]*CacheMetadata) error {
	conFacade, err := infra.RedisCluster.GetConnection(configId)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis cluster connection")
		return err
	}
	conn := conFacade.(*infra.RedisClusterConnection)
	cache, err2 := caches.NewRedisCacheFromRedisClusterConnection(conn)
	if err2 != nil {
		log.Error().Err(err2).Msg("Error creating redis cluster cache")
		return err2
	}
	confIdCacheMap[configId] = &CacheMetadata{
		CacheType: infra.DBTypeRedisCluster,
		hash:      "",
		Cache:     cache,
	}
	return nil
}
