package infra

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	Scylla *ScyllaConnectors
)

type ScyllaClusterConnection struct {
	Session       interface{} // Will hold either gocql or gocql_v2 session
	IsMeesho      bool
	ScyllaVersion int
	Meta          map[string]interface{}
}

func (c *ScyllaClusterConnection) GetConn() (interface{}, error) {
	if c.Session == nil {
		return nil, errors.New("connection nil")
	}

	switch {
	case c.ScyllaVersion == 5, c.ScyllaVersion == 6 && !c.IsMeesho:
		if isSessionClosedV1(c.Session) {
			return nil, errors.New("gocql session closed")
		}
	case c.ScyllaVersion == 6 && c.IsMeesho:
		if isSessionClosedV2(c.Session) {
			return nil, errors.New("gocql_v2 session closed")
		}
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
	switch {
	case c.ScyllaVersion == 5, c.ScyllaVersion == 6 && !c.IsMeesho:
		return !isSessionClosedV1(c.Session)
	case c.ScyllaVersion == 6 && c.IsMeesho:
		return !isSessionClosedV2(c.Session)
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
		switch {
		case cfg.Version == 5, cfg.Version == 6 && !cfg.IsMeesho:
			session, err = createSessionV1(cfg.Config)
			if err != nil {
				log.Error().Err(err).Msg("Error connecting to gocql scylla db")
				panic(err)
			}
			log.Debug().Msgf("Created gocql session for config %s", configIdStr)

		case cfg.Version == 6 && cfg.IsMeesho:
			session, err = createSessionV2(cfg.Config)
			if err != nil {
				log.Error().Err(err).Msg("Error connecting to gocql_v2 scylla db")
				panic(err)
			}
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
			Session:       session,
			ScyllaVersion: cfg.Version,
			IsMeesho:      cfg.IsMeesho,
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
