package scylla

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/go-core/metric"
	"github.com/Meesho/go-core/scylla"
	"github.com/gocql/gocql"
)

var (
	queryCache sync.Map
	scyllaDb   Database
	once       sync.Once
)

const (
	envPrefix = "STORAGE_SCYLLA"
)

type Scylla struct {
	session  *gocql.Session
	keyspace string
}

func InitScyllaDb() Database {
	if scyllaDb == nil {
		once.Do(func() {
			clusterConfig, err := scylla.BuildClusterConfigFromEnv(envPrefix)
			if err != nil {
				log.Panic().Msgf("error initializing scylla db for env - %v and error %v", envPrefix, err)
			}
			session, err := clusterConfig.CreateSession()
			if err != nil {
				log.Panic().Msgf("error connecting scylla db - %v", err)
			}
			scyllaDb = &Scylla{
				session:  session,
				keyspace: appConfig.ScyllaKeyspace,
			}
		})
	}
	return scyllaDb
}

func (s *Scylla) RetrieveInteractions(tableName string, userId string, columns []string) (map[string]interface{}, error) {
	t1 := time.Now()
	metric.Incr("scylla_db_interactions_retrieve_count", []string{metric.TagAsString("table_name", tableName)})
	preparedQuery := createRetrieveInteractionsPreparedQuery(s.session, s.keyspace, tableName, columns)
	response := executeRetrieveInteractions(preparedQuery, userId)
	metric.Timing("scylla_db_interactions_retrieve_latency", time.Since(t1), []string{metric.TagAsString("table_name", tableName)})
	return response, nil
}

func (s *Scylla) PersistInteractions(tableName string, userId string, columns map[string]interface{}) error {
	t1 := time.Now()
	metric.Incr("scylla_db_interactions_persist_count", []string{metric.TagAsString("table_name", tableName)})
	preparedQuery, sortedColumns := createPersistInteractionsPreparedQuery(s.session, s.keyspace, tableName, userId, columns)
	err := executePersistInteractions(preparedQuery, sortedColumns, userId, columns)
	if err != nil {
		log.Error().Msgf("error persisting data for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_interactions_persist_latency", time.Since(t1), []string{metric.TagAsString("table_name", tableName)})
	return nil
}

func (s *Scylla) UpdateInteractions(tableName string, userId string, column string, value interface{}) error {
	t1 := time.Now()
	metric.Incr("scylla_db_interactions_update_count", []string{metric.TagAsString("table_name", tableName)})
	preparedQuery := createUpdateInteractionsPreparedQuery(s.session, s.keyspace, tableName, column)
	err := executeUpdateInteractions(preparedQuery, value, userId)
	if err != nil {
		log.Error().Msgf("error updating data for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_interactions_update_latency", time.Since(t1), []string{metric.TagAsString("table_name", tableName)})
	return nil
}

func (s *Scylla) RetrieveMetadata(metadataTableName string, userId string) (map[string]interface{}, error) {
	t1 := time.Now()
	metric.Incr("scylla_db_retrieve_metadata_count", []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	preparedQuery := createRetrieveMetadataPreparedQuery(s.session, s.keyspace, metadataTableName)
	response := executeRetrieveMetadata(preparedQuery, userId)
	metric.Timing("scylla_db_retrieve_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return response, nil
}

func (s *Scylla) PersistMetadata(metadataTableName string, userId string, weeklyEventCount interface{}) error {
	t1 := time.Now()
	metric.Incr("scylla_db_persist_metadata_count", []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	preparedQuery := createPersistMetadataPreparedQuery(s.session, s.keyspace, metadataTableName)
	err := executePersistMetadata(preparedQuery, weeklyEventCount, userId)
	if err != nil {
		log.Error().Msgf("error persisting metadata for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_persist_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return nil
}

func (s *Scylla) UpdateMetadata(metadataTableName string, userId string, weeklyEventCount interface{}) error {
	t1 := time.Now()
	metric.Incr("scylla_db_update_metadata_count", []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	preparedQuery := createUpdateMetadataPreparedQuery(s.session, s.keyspace, metadataTableName)
	err := executeUpdateMetadata(preparedQuery, userId, weeklyEventCount)
	if err != nil {
		log.Error().Msgf("error updating metadata for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_update_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return nil
}

func executeRetrieveInteractions(preparedQuery *gocql.Query, userId string) map[string]interface{} {
	preparedQuery.Bind(userId).Consistency(gocql.One)
	res, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		return nil
	}
	if len(res) == 0 {
		return make(map[string]interface{})
	}
	return res[0]
}

func executePersistInteractions(preparedQuery *gocql.Query, sortedColumns []string, userId string, columns map[string]interface{}) error {
	var boundValues []interface{}
	for _, val := range sortedColumns {
		boundValues = append(boundValues, columns[val])
	}
	boundValues = append(boundValues, userId)
	preparedQuery.Bind(boundValues...)
	preparedQuery.Consistency(gocql.Quorum)
	_, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v\n", preparedQuery, err)
		return err
	}
	return nil
}

func executeUpdateInteractions(preparedQuery *gocql.Query, value interface{}, userId string) error {
	preparedQuery.Bind(value, userId).Consistency(gocql.Quorum)
	_, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v\n", preparedQuery, err)
		return err
	}
	return nil
}

func executeRetrieveMetadata(preparedQuery *gocql.Query, userId string) map[string]interface{} {
	preparedQuery.Bind(userId).Consistency(gocql.One)
	res, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		return nil
	}
	return res[0]
}

func executePersistMetadata(preparedQuery *gocql.Query, weeklyEventCount interface{}, userId string) error {
	preparedQuery.Bind(weeklyEventCount, userId).Consistency(gocql.Quorum)
	_, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		return err
	}
	return nil
}

func executeUpdateMetadata(preparedQuery *gocql.Query, userId string, weeklyEventCount interface{}) error {
	preparedQuery.Bind(weeklyEventCount, userId).Consistency(gocql.Quorum)
	_, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		return err
	}
	return nil
}

func createRetrieveInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, columns []string) *gocql.Query {
	var query string
	var preparedQuery *gocql.Query
	columnsStr := strings.Join(columns, ", ")
	cachedQuery, found := queryCache.Load(tableName + "_" + columnsStr + "_retrieve")
	if !found {
		query = fmt.Sprintf(RetrieveQuery, columnsStr, keyspace, tableName)
		queryCache.Store(tableName+"_"+columnsStr+"_retrieve", query)
		preparedQuery = session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = session.Query(query)
	}
	return preparedQuery
}

func createPersistInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, userId string, columns map[string]interface{}) (*gocql.Query, []string) {
	var query string
	var preparedQuery *gocql.Query
	columnNames := make([]string, 0, len(columns))
	placeholders := make([]string, 0, len(columns))
	for col := range columns {
		columnNames = append(columnNames, col)
		placeholders = append(placeholders, "?")
	}
	sort.Strings(columnNames)
	columnsStr := strings.Join(columnNames, ", ")
	columnsStr = columnsStr + ", user_id"
	cachedQuery, found := queryCache.Load(tableName + "_" + columnsStr + "_persist")
	if !found {
		placeholdersStr := strings.Join(placeholders, ", ")
		placeholdersStr = placeholdersStr + ", ?"
		query = fmt.Sprintf(PersistQuery, keyspace, tableName, columnsStr, placeholdersStr)
		queryCache.Store(tableName+"_"+columnsStr+"_persist", query)
		preparedQuery = session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = session.Query(query)
	}
	return preparedQuery, columnNames
}

func createUpdateInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, column string) *gocql.Query {
	var query string
	var preparedQuery *gocql.Query
	cachedQuery, found := queryCache.Load(tableName + "_" + column + "_update")
	if !found {
		query = fmt.Sprintf(UpdateQuery, keyspace, tableName, column)
		queryCache.Store(tableName+"_"+column+"_update", query)
		preparedQuery = session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = session.Query(query)
	}
	return preparedQuery
}

func createRetrieveMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string) *gocql.Query {
	var query string
	var preparedQuery *gocql.Query
	cachedQuery, found := queryCache.Load(metadataTableName + "_retrieve")
	if !found {
		query = fmt.Sprintf(RetrieveMetadataQuery, keyspace, metadataTableName)
		queryCache.Store(metadataTableName+"_retrieve", query)
		preparedQuery = session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = session.Query(query)
	}
	return preparedQuery
}

func createPersistMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string) *gocql.Query {
	var query string
	var preparedQuery *gocql.Query
	cachedQuery, found := queryCache.Load(metadataTableName + "_persist_metadata")
	if !found {
		query = fmt.Sprintf(PersistMetadataQuery, keyspace, metadataTableName)
		queryCache.Store(metadataTableName+"_persist_metadata", query)
		preparedQuery = session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = session.Query(query)
	}
	return preparedQuery
}

func createUpdateMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string) *gocql.Query {
	var query string
	var preparedQuery *gocql.Query
	cachedQuery, found := queryCache.Load(metadataTableName + "_update")
	if !found {
		query = fmt.Sprintf(UpdateMetadataQuery, keyspace, metadataTableName)
		queryCache.Store(metadataTableName+"_update", query)
		preparedQuery = session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = session.Query(query)
	}
	return preparedQuery
}
