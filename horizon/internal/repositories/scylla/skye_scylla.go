package scylla

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
)

type SkyeStore interface {
	CreateEmbeddingTable(tableName string, defaultTimeToLive int, variantsList []string) error
	AddEmbeddingColumn(tableName string, columnName string) error

	CreateAggregatorTable(tableName string, defaultTimeToLive int) error
	AddAggregatorColumn(tableName string, columnName string) error
}

// SkyeScylla implements SkyeStore interface
type SkyeScylla struct {
	keySpace string
	session  *gocql.Session
}

const (
	createEmbeddingTableQueryBase = `CREATE TABLE IF NOT EXISTS %s.%s (
		model_name text,
		version int,
		id text,
		embedding frozen<list<float>>,
		search_embedding frozen<list<float>>%s,
		PRIMARY KEY ((model_name, version, id))
	) WITH default_time_to_live = %d`

	createAggregatorTableQuery = `CREATE TABLE IF NOT EXISTS %s.%s (
		candidate_id text,
		PRIMARY KEY (candidate_id)
	) WITH default_time_to_live = %d`

	addBoolColumnQuery = `ALTER TABLE %s.%s ADD %s boolean`
	addTextColumnQuery = `ALTER TABLE %s.%s ADD %s text`

	checkTableExistsQuery = `SELECT table_name FROM system_schema.tables WHERE keyspace_name = ? AND table_name = ?`
)

func NewSkyeRepository(connection *infra.ScyllaClusterConnection) (SkyeStore, error) {
	meta, err := connection.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection meta: %w", err)
	}
	keySpace := meta["keyspace"].(string)

	session, err := connection.GetConn()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return &SkyeScylla{
		keySpace: keySpace,
		session:  session.(*gocql.Session),
	}, nil
}

// tableExists checks if a table exists in the keyspace
func (s *SkyeScylla) tableExists(tableName string) (bool, error) {
	var foundTableName string
	err := s.session.Query(checkTableExistsQuery, s.keySpace, tableName).Scan(&foundTableName)
	if err != nil {
		if err == gocql.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if table exists: %w", err)
	}
	return true, nil
}

func (s *SkyeScylla) CreateEmbeddingTable(tableName string, defaultTimeToLive int, variantsList []string) error {
	// Check if table already exists
	exists, err := s.tableExists(tableName)
	if err != nil {
		log.Error().Err(err).Str("keyspace", s.keySpace).Str("table", tableName).
			Msg("Failed to check if embedding table exists")
		return fmt.Errorf("failed to check if table exists: %w", err)
	}
	if exists {
		log.Error().Str("keyspace", s.keySpace).Str("table", tableName).
			Msg("Embedding table already exists, skipping creation")
		return nil
	}
	// Build variant column definitions
	var variantColumns strings.Builder
	for _, variant := range variantsList {
		// Convert variant name to snake_case and append _to_be_indexed
		variantColumnName := strings.ToLower(strings.ReplaceAll(variant, "-", "_")) + "_to_be_indexed"
		variantColumns.WriteString(fmt.Sprintf(",\n\t\t%s boolean", variantColumnName))
	}

	query := fmt.Sprintf(createEmbeddingTableQueryBase, s.keySpace, tableName, variantColumns.String(), defaultTimeToLive)

	err = s.session.Query(query).Exec()
	if err != nil {
		log.Error().Err(err).Str("keyspace", s.keySpace).Str("table", tableName).
			Msg("Failed to create embedding table")
		return fmt.Errorf("failed to create embedding table %s: %w", tableName, err)
	}

	log.Info().Str("keyspace", s.keySpace).Str("table", tableName).
		Msg("Successfully created embedding table")
	return nil
}

func (s *SkyeScylla) AddEmbeddingColumn(tableName string, columnName string) error {
	query := fmt.Sprintf(addBoolColumnQuery, s.keySpace, tableName, columnName)

	err := s.session.Query(query).Exec()
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Warn().Str("keyspace", s.keySpace).Str("table", tableName).
				Str("column", columnName).Msg("Column already exists")
			return nil
		}
		log.Error().Err(err).Str("keyspace", s.keySpace).Str("table", tableName).
			Str("column", columnName).Msg("Failed to add boolean column")
		return fmt.Errorf("failed to add column %s to table %s: %w", columnName, tableName, err)
	}

	log.Info().Str("keyspace", s.keySpace).Str("table", tableName).
		Str("column", columnName).Msg("Successfully added boolean column")
	return nil
}

func (s *SkyeScylla) CreateAggregatorTable(tableName string, defaultTimeToLive int) error {
	// Check if table already exists
	exists, err := s.tableExists(tableName)
	if err != nil {
		log.Error().Err(err).Str("keyspace", s.keySpace).Str("table", tableName).
			Msg("Failed to check if aggregator table exists")
		return fmt.Errorf("failed to check if table exists: %w", err)
	}
	if exists {
		log.Error().Str("keyspace", s.keySpace).Str("table", tableName).
			Msg("Aggregator table already exists, skipping creation")
		return nil
	}

	query := fmt.Sprintf(createAggregatorTableQuery, s.keySpace, tableName, defaultTimeToLive)

	err = s.session.Query(query).Exec()
	if err != nil {
		log.Error().Err(err).Str("keyspace", s.keySpace).Str("table", tableName).
			Msg("Failed to create aggregator table")
		return fmt.Errorf("failed to create aggregator table %s: %w", tableName, err)
	}

	log.Info().Str("keyspace", s.keySpace).Str("table", tableName).
		Msg("Successfully created aggregator table")
	return nil
}

func (s *SkyeScylla) AddAggregatorColumn(tableName string, columnName string) error {
	query := fmt.Sprintf(addTextColumnQuery, s.keySpace, tableName, columnName)

	err := s.session.Query(query).Exec()
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Warn().Str("keyspace", s.keySpace).Str("table", tableName).
				Str("column", columnName).Msg("Column already exists")
			return nil
		}
		log.Error().Err(err).Str("keyspace", s.keySpace).Str("table", tableName).
			Str("column", columnName).Msg("Failed to add text column")
		return fmt.Errorf("failed to add column %s to table %s: %w", columnName, tableName, err)
	}

	log.Info().Str("keyspace", s.keySpace).Str("table", tableName).
		Str("column", columnName).Msg("Successfully added text column")
	return nil
}
