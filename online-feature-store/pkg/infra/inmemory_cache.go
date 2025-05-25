package infra

import (
	"github.com/coocood/freecache"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

var (
	InMemoryCache                *InMemoryCacheConnectors
	InMemoryCacheLoadedConfigIds []int
)

type InMemoryCacheConnection struct {
	Client *freecache.Cache
	Meta   map[string]interface{}
}

func (c *InMemoryCacheConnection) GetConn() (interface{}, error) {
	if c.Client == nil {
		return nil, nil
	}
	return c.Client, nil
}

func (c *InMemoryCacheConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, nil
	}
	return c.Meta, nil
}

func (c *InMemoryCacheConnection) IsLive() bool {
	meta := c.Meta
	if meta == nil {
		return false
	}
	enabled := meta["enabled"].(bool)
	return enabled
}

type InMemoryCacheConnectors struct {
	InMemoryCacheConnections map[int]ConnectionFacade
}

func (s *InMemoryCacheConnectors) GetConnection(configId int) (ConnectionFacade, error) {
	conn, ok := s.InMemoryCacheConnections[configId]
	if !ok {
		return nil, nil
	}
	return conn, nil
}

func initInMemoryCacheConns() {
	activeConfIdsStr := viper.GetString(inMemoryCachePrefix + activeConfIds)
	if activeConfIdsStr == "" {
		return
	}
	activeIds := strings.Split(activeConfIdsStr, ",")
	InMemoryCacheConnections := make(map[int]ConnectionFacade, len(activeIds))
	for _, configIdStr := range activeIds {
		envPrefix := inMemoryCachePrefix + configIdStr
		conf, err := BuildInMemoryCacheConfFromEnv(envPrefix)
		if err != nil {
			log.Error().Err(err).Msg("Error building in memory cache conf")
			panic(err)
		}
		if !conf.Enabled {
			continue
		}
		configId, err := strconv.Atoi(configIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing in memory cache config id")
			panic(err)
		}
		if _, ok := ConfIdDBTypeMap[configId]; ok {
			log.Error().Msg("Invalid in memory cache config id")
			panic("Invalid in memory cache config id")
		}
		ConfIdDBTypeMap[configId] = DBTypeInMemory
		InMemoryCacheLoadedConfigIds = append(InMemoryCacheLoadedConfigIds, configId)
		client := freecache.NewCache(conf.SizeInBytes)
		meta := map[string]interface{}{
			"enabled": conf.Enabled,
			"name":    conf.Name,
		}
		InMemoryCacheConnections[configId] = &InMemoryCacheConnection{
			Client: client,
			Meta:   meta,
		}
	}
	InMemoryCache = &InMemoryCacheConnectors{
		InMemoryCacheConnections: InMemoryCacheConnections,
	}
}
