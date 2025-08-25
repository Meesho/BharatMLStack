package infra

import (
	"errors"
	"strconv"
	"strings"

	gocql_v2 "github.com/Meesho/gocql"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	Scylla *ScyllaConnectors
)

type ScyllaClusterConnection struct {
	Session interface{} // Will hold either gocql or gocql_v2 session
	Meta    map[string]interface{}
}

func (c *ScyllaClusterConnection) GetConn() (interface{}, error) {
	if c.Session == nil {
		return nil, errors.New("connection nil")
	}

	// Check if session is closed based on its type
	switch session := c.Session.(type) {
	case *gocql.Session:
		if session.Closed() {
			return nil, errors.New("gocql session closed")
		}
	case *gocql_v2.Session:
		if session.Closed() {
			return nil, errors.New("gocql_v2 session closed")
		}
	default:
		return nil, errors.New("unknown session type")
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
	if c.Session == nil {
		return false
	}

	// Check if session is live based on its type
	switch session := c.Session.(type) {
	case *gocql.Session:
		return !session.Closed()
	case *gocql_v2.Session:
		return !session.Closed()
	default:
		return false
	}
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

		// Create session based on the config type
		var session interface{}
		switch typedCfg := cfg.Config.(type) {
		case *gocql.ClusterConfig:
			// Create standard gocql session
			gocqlSession, err := typedCfg.CreateSession()
			if err != nil {
				log.Error().Err(err).Msg("Error connecting to gocql scylla db")
				panic(err)
			}
			session = gocqlSession
			log.Debug().Msgf("Created gocql session for config %s", configIdStr)

		case *gocql_v2.ClusterConfig:
			// Create gocql_v2 session
			gocqlV2Session, err := typedCfg.CreateSession()
			if err != nil {
				log.Error().Err(err).Msg("Error connecting to gocql_v2 scylla db")
				panic(err)
			}
			session = gocqlV2Session
			log.Debug().Msgf("Created gocql_v2 session for config %s", configIdStr)

		default:
			log.Error().Msg("Unsupported cluster config type")
			panic("Unsupported cluster config type")
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

		// Use the original DBTypeScylla for all Scylla configurations
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
