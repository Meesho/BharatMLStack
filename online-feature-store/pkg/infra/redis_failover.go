package infra

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	RedisFailover *RedisFailoverConnectors
)

type RedisFailoverConnection struct {
	Client redis.UniversalClient
	Meta   map[string]interface{}
}

func (c *RedisFailoverConnection) GetConn() (interface{}, error) {
	if c.Client == nil {
		return nil, errors.New("connection nil")
	}
	return c.Client, nil
}

func (c *RedisFailoverConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, errors.New("meta nil")
	}
	return c.Meta, nil
}

func (c *RedisFailoverConnection) IsLive() bool {
	if err := c.Client.Ping(context.Background()).Err(); err != nil {
		return false
	}
	return true
}

type RedisFailoverConnectors struct {
	RedisFailoverConnections map[int]ConnectionFacade
}

func (s *RedisFailoverConnectors) GetConnection(configId int) (ConnectionFacade, error) {
	conn, ok := s.RedisFailoverConnections[configId]
	if !ok {
		return nil, errors.New("connection not found")
	}
	return conn, nil
}

func initRedisFailoverConns() {
	activeConfIdsStr := viper.GetString(storageRedisFailoverPrefix + activeConfIds)
	if activeConfIdsStr == "" {
		return
	}
	activeIds := strings.Split(activeConfIdsStr, ",")
	RedisFailoverConnections := make(map[int]ConnectionFacade, len(activeIds))
	for _, configIdStr := range activeIds {
		envPrefix := storageRedisFailoverPrefix + configIdStr
		cfg, err := BuildRedisFailoverOptionsFromEnv(envPrefix)
		if err != nil {
			log.Error().Err(err).Msg("Error building redis failover options")
			panic(err)
		}
		failoverClient := redis.NewFailoverClusterClient(cfg)
		configId, err := strconv.Atoi(configIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error converting configId")
			panic(err)
		}
		if _, ok := ConfIdDBTypeMap[configId]; ok {
			log.Error().Err(err).Msg("Duplicate config id")
			panic("Duplicate config id")
		}
		ConfIdDBTypeMap[configId] = DBTypeRedisFailover
		RedisFailoverConnections[configId] = &RedisFailoverConnection{
			Client: failoverClient,
			Meta: map[string]interface{}{
				"configId": configId,
				"type":     DBTypeRedisFailover,
			},
		}
	}
	RedisFailover = &RedisFailoverConnectors{
		RedisFailoverConnections: RedisFailoverConnections,
	}
}
