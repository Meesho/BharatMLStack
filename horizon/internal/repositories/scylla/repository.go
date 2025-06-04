package scylla

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
)

const (
	addColumnQuery   = "ALTER TABLE %s.%s ADD %s blob"
	createTableQuery = "CREATE TABLE IF NOT EXISTS %s.%s (%s PRIMARY KEY (%s)) WITH default_time_to_live = %v;"
	tableExistsQuery = "SELECT table_name FROM system_schema.tables WHERE keyspace_name='%s' AND table_name='%s'"
)

type Scylla struct {
	keySpace string
	session  *gocql.Session
}

func NewRepository(connection *infra.ScyllaClusterConnection) (Store, error) {
	meta, err := connection.GetMeta()
	if err != nil {
		return nil, err
	}
	keySpace := meta["keyspace"].(string)
	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	return &Scylla{
		keySpace: keySpace,
		session:  session.(*gocql.Session),
	}, nil
}

// CreateTable implements the Store interface method to create a table
func (s *Scylla) CreateTable(tableName string, pkColumns []string, defaultTimeToLive int) error {
	// Build the column definitions
	var columnDefs []string

	// Add primary key columns as text type
	for _, pkCol := range pkColumns {
		columnDefs = append(columnDefs, pkCol+" text")
	}

	// Join all column definitions
	allColumns := strings.Join(columnDefs, ", ")
	allColumns = allColumns + ", "
	// Format the primary key part
	var pkDef string
	if len(pkColumns) == 1 {
		pkDef = pkColumns[0]
	} else {
		pkDef = "(" + strings.Join(pkColumns, ", ") + ")"
	}

	// Use default TTL if not provided
	if defaultTimeToLive == 0 {
		defaultTimeToLive = 0
	}

	// Create the query
	query := fmt.Sprintf(createTableQuery, s.keySpace, tableName, allColumns, pkDef, defaultTimeToLive)
	// Check if the table already exists
	tableExistsQuery := fmt.Sprintf(tableExistsQuery, s.keySpace, tableName)
	var tableNameResult string
	err := s.session.Query(tableExistsQuery).Scan(&tableNameResult)
	if err == nil && tableNameResult == tableName {
		return fmt.Errorf("table %s already exists in keyspace %s", tableName, s.keySpace)
	}

	// Execute the query
	err = s.session.Query(query).Exec()
	if err != nil {
		log.Error().Msgf("Error Creating Table for %s , %s with Error %v", s.keySpace, tableName, err)
		return err
	}
	return nil
}

// AddColumn implements the Store interface method to add a column to the table
func (s *Scylla) AddColumn(tableName string, column string) error {
	query := fmt.Sprintf(addColumnQuery, s.keySpace, tableName, column)
	return s.session.Query(query).Exec()
}
