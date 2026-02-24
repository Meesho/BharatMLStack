package scylla

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/scylla"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
)

var (
	queryCache sync.Map
	scyllaDb   Database
	once       sync.Once
)

const (
	envPrefix   = "STORAGE_SCYLLA"
	cacheKeySep = "_"
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
	preparedQuery := createRetrieveInteractionsPreparedQuery(s.session, s.keyspace, tableName, columns)
	response := executeRetrieveInteractions(preparedQuery, userId, columns)
	metric.Timing("scylla_db_interactions_retrieve_latency", time.Since(t1), []string{metric.TagAsString("table_name", tableName)})
	return response, nil
}

func (s *Scylla) UpdateInteractions(tableName string, userId string, columns map[string]interface{}) error {
	t1 := time.Now()
	preparedQuery, sortedColumns := createUpdateInteractionsPreparedQuery(s.session, s.keyspace, tableName, userId, columns)
	err := executeUpdateInteractions(preparedQuery, sortedColumns, userId, columns)
	if err != nil {
		log.Error().Msgf("error updating data for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_interactions_update_latency", time.Since(t1), []string{metric.TagAsString("table_name", tableName)})
	return nil
}

func (s *Scylla) RetrieveMetadata(metadataTableName string, userId string, columns []string) (map[string]interface{}, error) {
	t1 := time.Now()
	preparedQuery := createRetrieveMetadataPreparedQuery(s.session, s.keyspace, metadataTableName, columns)
	response := executeRetrieveMetadata(preparedQuery, userId, columns)
	metric.Timing("scylla_db_retrieve_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return response, nil
}

func (s *Scylla) UpdateMetadata(metadataTableName string, userId string, columns map[string]interface{}) error {
	t1 := time.Now()
	preparedQuery, sortedColumns := createUpdateMetadataPreparedQuery(s.session, s.keyspace, metadataTableName, columns)
	err := executeUpdateMetadata(preparedQuery, sortedColumns, userId, columns)
	if err != nil {
		log.Error().Msgf("error updating metadata for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_update_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return nil
}

func executeRetrieveInteractions(preparedQuery *gocql.Query, userId string, columns []string) map[string]interface{} {
	preparedQuery.Bind(userId).Consistency(gocql.One)
	iter := preparedQuery.Iter()

	scanDest := make([]*[]byte, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range columns {
		scanDest[i] = new([]byte)
		scanArgs[i] = scanDest[i]
	}

	if !iter.Scan(scanArgs...) {
		if err := iter.Close(); err != nil {
			log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		}
		return make(map[string]interface{})
	}

	result := make(map[string]interface{}, len(columns))
	for i, col := range columns {
		result[col] = *scanDest[i]
	}

	if err := iter.Close(); err != nil {
		log.Error().Msgf("error closing iterator for query %v: %v", preparedQuery, err)
	}
	return result
}

func executeUpdateInteractions(preparedQuery *gocql.Query, sortedColumns []string, userId string, columns map[string]interface{}) error {
	var boundValues []interface{}
	for _, val := range sortedColumns {
		boundValues = append(boundValues, columns[val])
	}
	boundValues = append(boundValues, userId)
	preparedQuery.Bind(boundValues...).Consistency(gocql.Quorum)
	err := preparedQuery.Exec()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v\n", preparedQuery, err)
		return err
	}
	return nil
}

// executeRetrieveMetadata scans metadata table columns as int (schema: week_0..week_23 are int).
func executeRetrieveMetadata(preparedQuery *gocql.Query, userId string, columns []string) map[string]interface{} {
	preparedQuery.Bind(userId).Consistency(gocql.One)
	iter := preparedQuery.Iter()

	scanDest := make([]*int, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range columns {
		scanDest[i] = new(int)
		scanArgs[i] = scanDest[i]
	}

	if !iter.Scan(scanArgs...) {
		if err := iter.Close(); err != nil {
			log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		}
		return make(map[string]interface{})
	}

	result := make(map[string]interface{}, len(columns))
	for i, col := range columns {
		result[col] = *scanDest[i]
	}

	if err := iter.Close(); err != nil {
		log.Error().Msgf("error closing iterator for query %v: %v", preparedQuery, err)
	}
	return result
}

func executeUpdateMetadata(preparedQuery *gocql.Query, sortedColumns []string, userId string, columns map[string]interface{}) error {
	var boundValues []interface{}
	for _, val := range sortedColumns {
		boundValues = append(boundValues, columns[val])
	}
	boundValues = append(boundValues, userId)
	preparedQuery.Bind(boundValues...).Consistency(gocql.Quorum)
	err := preparedQuery.Exec()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		return err
	}
	return nil
}

// Callers pass columns already sorted, so no hash is needed.
func getRetrieveQueryCacheKey(keyspace, tableName string, columns []string) string {
	var b strings.Builder
	// keyspace + tableName exact; ~10 chars per column name (heuristic); +16 for "_" separators and "retrieve" suffix.
	b.Grow(len(keyspace) + len(tableName) + len(columns)*10 + 16)
	b.WriteString(keyspace)
	b.WriteString(cacheKeySep)
	b.WriteString(tableName)
	b.WriteString(cacheKeySep)
	b.WriteString(strings.Join(columns, ","))
	b.WriteString(cacheKeySep)
	b.WriteString("retrieve")
	return b.String()
}

func getUpdateQueryCacheKey(keyspace, tableName string, columns []string) string {
	var b strings.Builder
	// Grow hint: keyspace + tableName exact; ~10 chars per column name (heuristic); +12 for "_" separators and "update" suffix.
	b.Grow(len(keyspace) + len(tableName) + len(columns)*10 + 12)
	b.WriteString(keyspace)
	b.WriteString(cacheKeySep)
	b.WriteString(tableName)
	b.WriteString(cacheKeySep)
	b.WriteString(strings.Join(columns, ","))
	b.WriteString(cacheKeySep)
	b.WriteString("update")
	return b.String()
}

func createRetrieveInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, columns []string) *gocql.Query {
	cacheKey := getRetrieveQueryCacheKey(keyspace, tableName, columns)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string))
	}

	columnsStr := strings.Join(columns, ", ")
	query := fmt.Sprintf(RetrieveQuery, columnsStr, keyspace, tableName)
	queryCache.Store(cacheKey, query)
	metric.Count("retrieve_query_cache_miss", 1, []string{"table", tableName})

	return session.Query(query)
}

func createUpdateInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, userId string, columns map[string]interface{}) (*gocql.Query, []string) {
	columnNames := make([]string, 0, len(columns))
	for col := range columns {
		columnNames = append(columnNames, col)
	}
	sort.Strings(columnNames)

	cacheKey := getUpdateQueryCacheKey(keyspace, tableName, columnNames)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string)), columnNames
	}

	assignments := make([]string, len(columnNames))
	for i, col := range columnNames {
		assignments[i] = col + " = ?"
	}

	assignmentsStr := strings.Join(assignments, ", ")
	query := fmt.Sprintf(UpdateQuery, keyspace, tableName, assignmentsStr)
	queryCache.Store(cacheKey, query)

	return session.Query(query), columnNames
}

func createRetrieveMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string, columns []string) *gocql.Query {
	cacheKey := getRetrieveQueryCacheKey(keyspace, metadataTableName, columns)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string))
	}

	columnsStr := strings.Join(columns, ", ")
	query := fmt.Sprintf(RetrieveMetadataQuery, columnsStr, keyspace, metadataTableName)
	queryCache.Store(cacheKey, query)
	metric.Count("retrieve_metadata_query_cache_miss", 1, []string{"table", metadataTableName})

	return session.Query(query)
}

func createUpdateMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string, columns map[string]interface{}) (*gocql.Query, []string) {
	columnNames := make([]string, 0, len(columns))
	for col := range columns {
		columnNames = append(columnNames, col)
	}
	sort.Strings(columnNames)

	cacheKey := getUpdateQueryCacheKey(keyspace, metadataTableName, columnNames)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string)), columnNames
	}

	assignments := make([]string, len(columnNames))
	for i, col := range columnNames {
		assignments[i] = col + " = ?"
	}

	assignmentsStr := strings.Join(assignments, ", ")
	query := fmt.Sprintf(UpdateMetadataQuery, keyspace, metadataTableName, assignmentsStr)
	queryCache.Store(cacheKey, query)

	return session.Query(query), columnNames
}
