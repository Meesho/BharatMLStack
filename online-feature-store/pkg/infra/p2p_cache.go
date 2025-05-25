package infra

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	P2PCache                *P2PCacheConnectors
	P2PCacheLoadedConfigIds []int
)

type P2PCacheConnection struct {
	Client *p2pcache.P2PCache
	Meta   map[string]interface{}
}

func (c *P2PCacheConnection) GetConn() (interface{}, error) {
	if c.Client == nil {
		return nil, nil
	}
	return c.Client, nil
}

func (c *P2PCacheConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, nil
	}
	return c.Meta, nil
}

func (c *P2PCacheConnection) IsLive() bool {
	meta := c.Meta
	if meta == nil {
		return false
	}
	enabled := meta["enabled"].(bool)
	return enabled
}

type P2PCacheConnectors struct {
	P2PCacheConnections map[int]ConnectionFacade
}

func (s *P2PCacheConnectors) GetConnection(configId int) (ConnectionFacade, error) {
	conn, ok := s.P2PCacheConnections[configId]
	if !ok {
		return nil, nil
	}
	return conn, nil
}

func initP2PCacheConns() {
	activeConfIdsStr := viper.GetString(p2PCachePrefix + activeConfIds)
	if activeConfIdsStr == "" {
		return
	}
	activeConfIds := strings.Split(activeConfIdsStr, ",")
	p2PCacheConnections := make(map[int]ConnectionFacade, len(activeConfIds))
	for _, confIdStr := range activeConfIds {
		conf, err := BuildP2PCacheConfFromEnv(p2PCachePrefix + confIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error building p2p cache conf")
			panic(err)
		}
		if !conf.Enabled {
			continue
		}
		confId, err := strconv.Atoi(confIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing p2p cache config id")
			panic(err)
		}
		if _, ok := ConfIdDBTypeMap[confId]; ok {
			log.Error().Msgf("Invalid p2p cache config id %d", confId)
			panic(fmt.Sprintf("Invalid p2p cache config id %d", confId))
		}
		ConfIdDBTypeMap[confId] = DBTypeP2P
		P2PCacheLoadedConfigIds = append(P2PCacheLoadedConfigIds, confId)
		client := p2pcache.NewP2PCache(conf.Name, conf.OwnPartitionSizeInBytes, conf.GlobalSizeInBytes)
		meta := map[string]interface{}{
			"enabled": conf.Enabled,
			"name":    conf.Name,
		}
		p2PCacheConnections[confId] = &P2PCacheConnection{
			Client: client,
			Meta:   meta,
		}
	}
	P2PCache = &P2PCacheConnectors{
		P2PCacheConnections: p2PCacheConnections,
	}
}
