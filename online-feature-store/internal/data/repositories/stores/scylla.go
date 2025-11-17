package stores

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/models"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
)

const (
	genericScyllaRetrieveQuery = "SELECT %s FROM %s.%s WHERE %s"
	genericScyllaPersistQuery  = "INSERT INTO %s.%s (%s) VALUES (%s)"
)

type ScyllaStore struct {
	keySpace      string
	table         string
	configManager config.Manager
	session       interface{} // Will hold either gocql or gocql_v2 session
	queryCache    *ds.SyncMap[string, string]
	sessionType   string // "gocql" or "gocql_v2"
}

// QueryInterface defines the common interface for both gocql and gocql_v2 queries
type QueryInterface interface {
	Iter() interface{}
	Bind(values ...interface{}) interface{}
	Consistency(consistency interface{}) interface{}
	Exec() error
}

// SessionInterface defines the common interface for both gocql and gocql_v2 sessions
type SessionInterface interface {
	Query(query string) interface{}
}

func NewScyllaStore(table string, connection *infra.ScyllaClusterConnection) (Store, error) {
	meta, err := connection.GetMeta()
	if err != nil {
		return nil, err
	}
	keySpace := meta["keyspace"].(string)
	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	// Determine session type and validate
	var sessionType string
	switch connection.IsMeeshoVersion {
	case "false":
		sessionType = "gocql"
	case "true":
		sessionType = "gocql_v2"
	default:
		return nil, fmt.Errorf("unsupported session type")
	}

	configManager := config.Instance(config.DefaultVersion)
	return &ScyllaStore{
		table:         table,
		keySpace:      keySpace,
		session:       session,
		sessionType:   sessionType,
		configManager: configManager,
		queryCache:    ds.NewSyncMap[string, string](),
	}, nil
}

func (s *ScyllaStore) RetrieveV2(entityLabel string, pkMap map[string]string, fgIds []int) (map[int]*blocks.DeserializedPSDB, error) {
	t1 := time.Now()
	metric.Incr("db_retrieve_count", []string{"entity_label", entityLabel, "db_type", "scylla"})
	colPKMap, pkCols, err := s.configManager.GetPKMapAndPKColumnsForEntity(entityLabel)
	if err != nil {
		// log.Error().Err(err).Msgf("Error while getting PK and PK columns for entity: %s", entityLabel)
		return nil, err
	}
	if len(pkMap) == 0 || len(colPKMap) != len(pkMap) || len(fgIds) == 0 {
		// log.Error().Msgf("Error while getting PK and PK columns for entity: %s", entityLabel)
		return nil, fmt.Errorf("error while getting PK and PK columns for entity: %s", entityLabel)
	}
	fgCols := make([]string, 0)
	for _, fgId := range fgIds {
		cols, err := s.configManager.GetColumnsForEntityAndFG(entityLabel, fgId)
		if err != nil {
			// log.Error().Err(err).Msgf("Error while getting columns for entity: %s and fgId: %d", entityLabel, fgId)
			fgCols = fgCols[:0]
			break
		}
		fgCols = append(fgCols, cols...)
	}

	if len(fgCols) == 0 {
		// log.Error().Msgf("Error while getting columns for entity: %s", entityLabel)
		return nil, fmt.Errorf("error while getting columns for entity: %s", entityLabel)
	}
	query := s.getRetrievePreparedStatement(s.keySpace, s.table, fgCols, pkCols, s.session)
	query = prepareRetrieveQueryV2(pkMap, colPKMap, pkCols, query, s.sessionType)
	log.Debug().Msgf("DB retrieve query : %s", query)
	fgIdToDDB := make(map[int]*blocks.DeserializedPSDB, len(fgIds))

	// Execute query based on session type
	var rowData []map[string]interface{}
	switch s.sessionType {
	case "gocql":
		rowData, err = retrieveV1(query)
	case "gocql_v2":
		rowData, err = retrieveV2(query)
	default:
		return nil, fmt.Errorf("unknown session type: %s", s.sessionType)
	}

	log.Debug().Msgf("DB retrieve query result : %v", rowData)
	if err != nil {
		metric.Count("retrieve.failure", 1, []string{"db_type", "scylla", "entity", entityLabel})
		// log.Error().Err(err).Msgf("Scylla error | in execute query for entityLabel: %s, keys: %v", entityLabel, pkMap)
		return nil, err
	}
	for _, fgId := range fgIds {
		if len(rowData) == 0 {
			fgIdToDDB[fgId] = &blocks.DeserializedPSDB{
				NegativeCache: true,
			}
			continue
		}
		fgData := make([]byte, 0)
		cols, err := s.configManager.GetColumnsForEntityAndFG(entityLabel, fgId)
		if err != nil {
			// log.Error().Err(err).Msgf("Error while getting FG columns for entity: %s and fgId: %d", entityLabel, fgId)
			return nil, err
		}
		for _, col := range cols {
			if values, ok := rowData[0][col]; ok {
				fgData = append(fgData, values.([]byte)...)
			}
		}
		ddb, err2 := blocks.DeserializePSDB(fgData)
		if err2 != nil {
			// log.Error().Err(err2).Msgf("Error while deserializing PSDB for entity: %s and fgId: %d", entityLabel, fgId)
			return nil, err2
		}
		fgIdToDDB[fgId] = ddb
	}
	metric.Timing("db_retrieve_latency", time.Since(t1), []string{"entity_label", entityLabel, "db_type", "scylla"})
	return fgIdToDDB, nil
}

func (s *ScyllaStore) PersistV2(storeId string, entityLabel string, pkMap map[string]string, fgIdToPsDb map[int]*blocks.PermStorageDataBlock) error {
	metric.Incr("db_persist_count", []string{"entity_label", entityLabel, "db_type", "scylla"})
	colPKMap, pkCols, err := s.configManager.GetPKMapAndPKColumnsForEntity(entityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error while getting PK and PK columns for entity: %s", entityLabel)
		return err
	}
	if len(pkMap) == 0 || len(colPKMap) != len(pkMap) || len(fgIdToPsDb) == 0 {
		log.Error().Msgf("Error while getting PK and PK columns for entity: %s", entityLabel)
		return fmt.Errorf("error while getting PK and PK columns for entity: %s", entityLabel)
	}
	fgCols := make([]string, 0)
	columnToPSDBMap := make(map[string][]byte)
	maxColumnSize, err := s.configManager.GetMaxColumnSize(storeId)
	if err != nil {
		log.Error().Err(err).Msgf("Error while getting max column size for entity: %s", entityLabel)
		return err
	}
	for fgId, psDbBlock := range fgIdToPsDb {
		cols, err := s.configManager.GetColumnsForEntityAndFG(entityLabel, fgId)
		if err != nil {
			log.Error().Err(err).Msgf("Error while getting columns for entity: %s and fgId: %d", entityLabel, fgId)
			fgCols = fgCols[:0]
			break
		}
		fgCols = append(fgCols, cols...)
		s.serializePSDbData(entityLabel, fgId, cols, maxColumnSize, psDbBlock, columnToPSDBMap)
	}

	if len(fgCols) == 0 {
		log.Error().Msgf("Error while getting columns for entity: %s", entityLabel)
		return fmt.Errorf("error while getting columns for entity: %s", entityLabel)
	}
	columns := append(pkCols, fgCols...)
	ps := s.getPersistPreparedStatement(s.keySpace, s.table, columns, s.session)
	query := preparePersistQueryV2(pkMap, pkCols, fgCols, colPKMap, columnToPSDBMap, ps, s.sessionType)
	log.Debug().Msgf("Persist Query : %v", query)

	// Execute query based on session type
	switch s.sessionType {
	case "gocql":
		err = persistV1(query)
	case "gocql_v2":
		err = persistV2(query)
	default:
		return fmt.Errorf("unknown session type: %s", s.sessionType)
	}

	if err != nil {
		log.Error().Msgf(" Error while executing persist query %v with error %v", query, err)
		metric.Count("persist_query_failure", 1, []string{"entity_name", entityLabel, "db_type", "scylla"})
	}
	return nil
}

func (s *ScyllaStore) getPersistPreparedStatement(keyspace, table string, columns []string, session interface{}) interface{} {
	key := getPersistPreparedStatementKey(keyspace, table, columns)
	var query string
	query, _ = s.queryCache.Get(key)
	if query == "" {
		placeholders := make([]string, len(columns))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		query = buildPersistQueryTemplate(keyspace, table, columns, placeholders)
		s.queryCache.Set(key, query)
	}

	// Create query based on session type
	switch s.sessionType {
	case "gocql":
		query, err := getQueryV1(session, query)
		if err != nil {
			return nil
		}
		return query
	case "gocql_v2":
		query, err := getQueryV2(session, query)
		if err != nil {
			return nil
		}
		return query
	}
	return nil
}

func (s *ScyllaStore) getRetrievePreparedStatement(keyspace, table string, fgColumns []string, idColumns []string, session interface{}) interface{} {
	key := getRetrievePreparedStatementKey(keyspace, table, fgColumns, idColumns)
	var query string
	query, _ = s.queryCache.Get(key)
	if query == "" {
		query = buildRetrieveQueryTemplate(keyspace, table, fgColumns, idColumns)
		s.queryCache.Set(key, query)
	}

	// Create query based on session type
	switch s.sessionType {
	case "gocql":
		query, err := getQueryV1(session, query)
		if err != nil {
			return nil
		}
		return query
	case "gocql_v2":
		query, err := getQueryV2(session, query)
		if err != nil {
			return nil
		}
		return query
	}
	return nil
}

func buildRetrieveQueryTemplate(keyspace, table string, retrieveColumns, idColumns []string) string {
	preparedIdColumns := ""
	for i, idColumn := range idColumns {
		if i == len(idColumns)-1 {
			// For the last element, do not append "AND"
			preparedIdColumns += idColumn + " = ?"
		} else {
			preparedIdColumns += idColumn + " = ? AND "
		}
	}
	preparedRetrieveColumns := strings.Join(retrieveColumns, ",")
	query := fmt.Sprintf(genericScyllaRetrieveQuery, preparedRetrieveColumns, keyspace, table, preparedIdColumns)
	return query
}

func buildPersistQueryTemplate(keyspace, table string, preparedColumns []string, placeholders []string) string {
	preparedPersistColumns := strings.Join(preparedColumns, ",")
	preparedPlaceholders := strings.Join(placeholders, ",")
	query := fmt.Sprintf(genericScyllaPersistQuery, keyspace, table, preparedPersistColumns, preparedPlaceholders)
	return query
}

func getRetrievePreparedStatementKey(keyspace, table string, retrieveColumns, idColumns []string) string {
	return keyspace + table + strings.Join(retrieveColumns, "") + strings.Join(idColumns, "") + "retrieve"
}

func getPersistPreparedStatementKey(keyspace, table string, persistColumns []string) string {
	return keyspace + table + strings.Join(persistColumns, "") + "persist"
}

func prepareRetrieveQueryV2(pkMap map[string]string, colPKMap map[string]string, pkCols []string, ps interface{}, sessionType string) interface{} {
	var bindKeys []interface{}
	for _, pkCol := range pkCols {
		bindKeys = append(bindKeys, pkMap[colPKMap[pkCol]])
	}

	// Bind and set consistency based on session type
	switch sessionType {
	case "gocql":
		return bindV1(ps, bindKeys)
	case "gocql_v2":
		return bindV2(ps, bindKeys)
	}
	return nil
}

func (s *ScyllaStore) serializePSDbData(entityLabel string, fgId int, columns []string, maxColumnSize int,
	psDbBlock *blocks.PermStorageDataBlock, columnToPSDBMap map[string][]byte) error {
	serializedFeatures, err := psDbBlock.Serialize()
	if err != nil {
		log.Error().Err(err).Msgf("Store error | serialization failed for fgId: %v", fgId)
		metric.Count("serialize_psdb_data_failure", 1, []string{"entity_name", entityLabel, "fg_id", strconv.Itoa(fgId)})
		return err
	}
	for startIdx, col := 0, 0; col < len(columns); col++ {
		endIdx := startIdx + maxColumnSize
		if endIdx > len(serializedFeatures) {
			endIdx = len(serializedFeatures)
		}
		columnToPSDBMap[columns[col]] = serializedFeatures[startIdx:endIdx]
		startIdx = endIdx
	}
	return nil
}

func preparePersistQueryV2(pkMap map[string]string, pkCols []string, columns []string, colPKMap map[string]string, fgColsToPsdb map[string][]byte, ps interface{}, sessionType string) interface{} {
	var bindValues []interface{}
	for _, pkCol := range pkCols {
		bindValues = append(bindValues, pkMap[colPKMap[pkCol]])
	}
	for _, column := range columns {
		bindValues = append(bindValues, fgColsToPsdb[column])
	}

	// Bind and set consistency based on session type
	switch sessionType {
	case "gocql":
		return bindV1(ps, bindValues)
	case "gocql_v2":
		return bindV2(ps, bindValues)
	}
	return nil
}

func (s *ScyllaStore) BatchPersistV2(storeId string, entityLabel string, rows []models.Row) error {
	return fmt.Errorf("%w: BatchPersistV2 for Scylla store", ErrNotImplemented)
}

func (s *ScyllaStore) BatchRetrieveV2(entityLabel string, pkMaps []map[string]string, fgIds []int) ([]map[int]*blocks.DeserializedPSDB, error) {
	return nil, fmt.Errorf("%w: BatchRetrieveV2 for Scylla store", ErrNotImplemented)
}

func (s *ScyllaStore) Type() string {
	return StoreTypeScylla
}
