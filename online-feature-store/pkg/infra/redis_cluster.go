package infra

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	RedisCluster *RedisClusterConnectors
)

type RedisClusterConnection struct {
	Client redis.UniversalClient
	Meta   map[string]interface{}
}

func (c *RedisClusterConnection) GetConn() (interface{}, error) {
	if c.Client == nil || (c.Client).(*redis.Client).Conn() == nil {
		return nil, errors.New("connection nil")
	}
	return c.Client, nil
}

func (c *RedisClusterConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, errors.New("meta nil")
	}
	return c.Meta, nil
}

func (c *RedisClusterConnection) IsLive() bool {
	if err := c.Client.Ping(context.Background()).Err(); err != nil {
		return false
	}
	return true
}

type RedisClusterConnectors struct {
	RedisClusterConnections map[int]ConnectionFacade
}

func (s *RedisClusterConnectors) GetConnection(configId int) (ConnectionFacade, error) {
	conn, ok := s.RedisClusterConnections[configId]
	if !ok {
		return nil, errors.New("connection not found")
	}
	return conn, nil
}

func initRedisClusterConns() {
	activeConfIdsStr := viper.GetString(storageRedisClusterPrefix + activeConfIds)
	if activeConfIdsStr == "" {
		return
	}
	activeIds := strings.Split(activeConfIdsStr, ",")
	RedisClusterConnections := make(map[int]ConnectionFacade, len(activeIds))
	for _, configIdStr := range activeIds {
		envPrefix := storageRedisClusterPrefix + configIdStr
		cfg, err := BuildRedisClusterOptionsFromEnv(envPrefix)
		if err != nil {
			log.Error().Err(err).Msg("Error building redis cluster config")
			panic(err)
		}
		clusterClient := redis.NewClusterClient(cfg)
		configId, err := strconv.Atoi(configIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error converting configId to int")
			panic(err)
		}
		if _, ok := ConfIdDBTypeMap[configId]; ok {
			log.Error().Err(err).Msg("Duplicate config id")
			panic("Duplicate config id")
		}
		ConfIdDBTypeMap[configId] = DBTypeRedisCluster
		RedisClusterConnections[configId] = &RedisClusterConnection{
			Client: clusterClient,
			Meta: map[string]interface{}{
				"configId": configId,
				"type":     DBTypeRedisCluster,
			},
		}
	}
	RedisCluster = &RedisClusterConnectors{
		RedisClusterConnections: RedisClusterConnections,
	}
}
