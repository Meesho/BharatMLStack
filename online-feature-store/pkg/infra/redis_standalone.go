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
	RedisStandalone *RedisStandaloneConnectors
)

type RedisStandaloneConnection struct {
	Client redis.UniversalClient
	Meta   map[string]interface{}
}

func (c *RedisStandaloneConnection) GetConn() (interface{}, error) {
	if c.Client == nil || (c.Client).(*redis.Client).Conn() == nil {
		return nil, errors.New("connection nil")
	}
	return c.Client, nil
}

func (c *RedisStandaloneConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, errors.New("meta nil")
	}
	return c.Meta, nil
}

func (c *RedisStandaloneConnection) IsLive() bool {
	if err := c.Client.Ping(context.Background()).Err(); err != nil {
		return false
	}
	return true
}

type RedisStandaloneConnectors struct {
	RedisStandaloneConnections map[int]ConnectionFacade
}

func (s *RedisStandaloneConnectors) GetConnection(configId int) (ConnectionFacade, error) {
	conn, ok := s.RedisStandaloneConnections[configId]
	if !ok {
		return nil, errors.New("connection not found")
	}
	return conn, nil
}

func initRedisStandaloneConns() {
	activeConfIdsStr := viper.GetString(storageRedisStandalonePrefix + activeConfIds)
	if activeConfIdsStr == "" {
		return
	}
	activeIds := strings.Split(activeConfIdsStr, ",")
	RedisStandaloneConnections := make(map[int]ConnectionFacade, len(activeIds))
	for _, configIdStr := range activeIds {
		envPrefix := storageRedisStandalonePrefix + configIdStr
		redisOptions, err := BuildRedisOptionsFromEnv(envPrefix)
		if err != nil {
			log.Error().Err(err).Msg("Error building redis standalone config")
			panic(err)
		}
		configId, err := strconv.Atoi(configIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error converting configId to int")
			panic(err)
		}
		client := redis.NewClient(redisOptions)
		if _, ok := ConfIdDBTypeMap[configId]; ok {
			log.Error().Err(err).Msg("Duplicate config id")
			panic("Duplicate config id")
		}
		ConfIdDBTypeMap[configId] = DBTypeRedisStandalone
		RedisStandaloneConnections[configId] = &RedisStandaloneConnection{
			Client: client,
			Meta: map[string]interface{}{
				"configId": configId,
				"type":     DBTypeRedisStandalone,
			},
		}
	}
	RedisStandalone = &RedisStandaloneConnectors{
		RedisStandaloneConnections: RedisStandaloneConnections,
	}
}
