package infra

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	Scylla *ScyllaConnectors
)

type ScyllaClusterConnection struct {
	Session *gocql.Session
	Meta    map[string]interface{}
}

func (c *ScyllaClusterConnection) GetConn() (interface{}, error) {
	if c.Session == nil || c.Session.Closed() {
		return nil, errors.New("connection nil or closed")
	}
	return c.Session, nil
}

func (c *ScyllaClusterConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, errors.New("meta nil")
	}
	return c.Meta, nil
}

func (c *ScyllaClusterConnection) IsLive() bool {
	return !c.Session.Closed()
}

type ScyllaConnectors struct {
	ScyllaConnections map[int]ConnectionFacade
}

func (s *ScyllaConnectors) GetConnection(configId int) (ConnectionFacade, error) {
	conn, ok := s.ScyllaConnections[configId]
	if !ok {
		return nil, errors.New("connection not found")
	}
	return conn, nil
}

func initScyllaClusterConns() {
	activeConfIdsStr := viper.GetString(storageScyllaPrefix + activeConfIds)
	if activeConfIdsStr == "" {
		return
	}
	activeIds := strings.Split(activeConfIdsStr, ",")
	ScyllaConnections := make(map[int]ConnectionFacade, len(activeIds))
	for _, configIdStr := range activeIds {
		envPrefix := storageScyllaPrefix + configIdStr
		cfg, err := BuildClusterConfigFromEnv(envPrefix)
		if err != nil {
			log.Error().Err(err).Msg("Error building scylla cluster config")
			panic(err)
		}
		session, err := cfg.CreateSession()
		if err != nil {
			log.Error().Err(err).Msg("Error connecting scylla db")
			panic(err)
		}
		confId, err := strconv.Atoi(configIdStr)
		if err != nil {
			log.Error().Err(err).Msg("Error converting config id to int")
			panic(err)
		}
		if _, ok := ConfIdDBTypeMap[confId]; ok {
			log.Error().Err(err).Msg("Duplicate config id")
			panic("Duplicate config id")
		}
		ConfIdDBTypeMap[confId] = DBTypeScylla
		conn := &ScyllaClusterConnection{
			Session: session,
			Meta: map[string]interface{}{
				"configId": confId,
				"keyspace": cfg.Keyspace,
				"type":     DBTypeScylla,
			},
		}
		ScyllaConnections[confId] = conn
	}
	Scylla = &ScyllaConnectors{
		ScyllaConnections: ScyllaConnections,
	}
}
