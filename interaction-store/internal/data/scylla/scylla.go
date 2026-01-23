package scylla

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/scylla"
	"github.com/cespare/xxhash/v2"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
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

func (s *Scylla) RetrieveMetadata(metadataTableName string, userId string, columns []string) (map[string]interface{}, error) {
	t1 := time.Now()
	metric.Incr("scylla_db_retrieve_metadata_count", []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	preparedQuery := createRetrieveMetadataPreparedQuery(s.session, s.keyspace, metadataTableName, columns)
	response := executeRetrieveMetadata(preparedQuery, userId)
	metric.Timing("scylla_db_retrieve_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return response, nil
}

func (s *Scylla) PersistMetadata(metadataTableName string, userId string, columns map[string]interface{}) error {
	t1 := time.Now()
	metric.Incr("scylla_db_persist_metadata_count", []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	preparedQuery, sortedColumns := createPersistMetadataPreparedQuery(s.session, s.keyspace, metadataTableName, columns)
	err := executePersistMetadata(preparedQuery, sortedColumns, userId, columns)
	if err != nil {
		log.Error().Msgf("error persisting metadata for user %v: %v", userId, err)
		return err
	}
	metric.Timing("scylla_db_persist_metadata_latency", time.Since(t1), []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	return nil
}

func (s *Scylla) UpdateMetadata(metadataTableName string, userId string, column string, value interface{}) error {
	t1 := time.Now()
	metric.Incr("scylla_db_update_metadata_count", []string{metric.TagAsString("metadata_table_name", metadataTableName)})
	preparedQuery := createUpdateMetadataPreparedQuery(s.session, s.keyspace, metadataTableName, column)
	err := executeUpdateMetadata(preparedQuery, value, userId)
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
	err := preparedQuery.Exec()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v\n", preparedQuery, err)
		return err
	}
	return nil
}

func executeUpdateInteractions(preparedQuery *gocql.Query, value interface{}, userId string) error {
	preparedQuery.Bind(value, userId).Consistency(gocql.Quorum)
	err := preparedQuery.Exec()
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
	if len(res) == 0 {
		return make(map[string]interface{})
	}
	return res[0]
}

func executePersistMetadata(preparedQuery *gocql.Query, sortedColumns []string, userId string, columns map[string]interface{}) error {
	var boundValues []interface{}
	for _, col := range sortedColumns {
		boundValues = append(boundValues, columns[col])
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

func executeUpdateMetadata(preparedQuery *gocql.Query, value interface{}, userId string) error {
	preparedQuery.Bind(value, userId).Consistency(gocql.Quorum)
	err := preparedQuery.Exec()
	if err != nil {
		log.Error().Msgf("error executing cql query %v: %v", preparedQuery, err)
		return err
	}
	return nil
}

// unorderedHashXXH64 computes a hash that's independent of column order
func unorderedHashXXH64(values []string) uint64 {
	var h uint64 = 0
	for _, v := range values {
		h += xxhash.Sum64String(v)
	}
	return h
}

// getRetrieveQueryCacheKey generates a hash-based cache key for retrieve queries
// The key is order-independent for columns, meaning [col1, col2] and [col2, col1] produce the same key
func getRetrieveQueryCacheKey(keyspace, tableName string, columns []string) string {
	h := xxhash.New()
	// Fixed parts (order matters)
	_, _ = h.WriteString(keyspace)
	_, _ = h.WriteString(tableName)

	// Unordered columns hash (order doesn't matter)
	columnsHash := unorderedHashXXH64(columns)
	buf := make([]byte, 8)
	putUint64(buf, columnsHash)
	_, _ = h.Write(buf)

	_, _ = h.WriteString("retrieve")

	return fmt.Sprintf("%x", h.Sum(nil))
}

// getPersistQueryCacheKey generates a hash-based cache key for persist queries
func getPersistQueryCacheKey(keyspace, tableName string, columns []string) string {
	h := xxhash.New()
	// Fixed parts
	_, _ = h.WriteString(keyspace)
	_, _ = h.WriteString(tableName)

	// Unordered columns hash
	columnsHash := unorderedHashXXH64(columns)
	buf := make([]byte, 8)
	putUint64(buf, columnsHash)
	_, _ = h.Write(buf)

	_, _ = h.WriteString("persist")

	return fmt.Sprintf("%x", h.Sum(nil))
}

// getUpdateQueryCacheKey generates a cache key for update queries
func getUpdateQueryCacheKey(keyspace, tableName, column string) string {
	h := xxhash.New()
	_, _ = h.WriteString(keyspace)
	_, _ = h.WriteString(tableName)
	_, _ = h.WriteString(column)
	_, _ = h.WriteString("update")

	return fmt.Sprintf("%x", h.Sum(nil))
}

// putUint64 writes uint64 to byte slice in little-endian format
func putUint64(b []byte, v uint64) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
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

func createPersistInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, userId string, columns map[string]interface{}) (*gocql.Query, []string) {
	columnNames := make([]string, 0, len(columns))
	for col := range columns {
		columnNames = append(columnNames, col)
	}
	sort.Strings(columnNames)

	cacheKey := getPersistQueryCacheKey(keyspace, tableName, columnNames)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string)), columnNames
	}

	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	columnsStr := strings.Join(columnNames, ", ") + ", user_id"
	placeholdersStr := strings.Join(placeholders, ", ") + ", ?"
	query := fmt.Sprintf(PersistQuery, keyspace, tableName, columnsStr, placeholdersStr)
	queryCache.Store(cacheKey, query)
	metric.Count("persist_query_cache_miss", 1, []string{"table", tableName})

	return session.Query(query), columnNames
}

func createUpdateInteractionsPreparedQuery(session *gocql.Session, keyspace string, tableName string, column string) *gocql.Query {
	cacheKey := getUpdateQueryCacheKey(keyspace, tableName, column)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string))
	}

	query := fmt.Sprintf(UpdateQuery, keyspace, tableName, column)
	queryCache.Store(cacheKey, query)

	return session.Query(query)
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

func createPersistMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string, columns map[string]interface{}) (*gocql.Query, []string) {
	columnNames := make([]string, 0, len(columns))
	for col := range columns {
		columnNames = append(columnNames, col)
	}
	sort.Strings(columnNames)

	cacheKey := getPersistQueryCacheKey(keyspace, metadataTableName, columnNames)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string)), columnNames
	}

	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	columnsStr := strings.Join(columnNames, ", ") + ", user_id"
	placeholdersStr := strings.Join(placeholders, ", ") + ", ?"
	query := fmt.Sprintf(PersistMetadataQuery, keyspace, metadataTableName, columnsStr, placeholdersStr)
	queryCache.Store(cacheKey, query)
	metric.Count("persist_metadata_query_cache_miss", 1, []string{"table", metadataTableName})

	return session.Query(query), columnNames
}

func createUpdateMetadataPreparedQuery(session *gocql.Session, keyspace string, metadataTableName string, column string) *gocql.Query {
	cacheKey := getUpdateQueryCacheKey(keyspace, metadataTableName, column)

	cachedQuery, found := queryCache.Load(cacheKey)
	if found {
		return session.Query(cachedQuery.(string))
	}

	query := fmt.Sprintf(UpdateMetadataQuery, keyspace, metadataTableName, column)
	queryCache.Store(cacheKey, query)

	return session.Query(query)
}
