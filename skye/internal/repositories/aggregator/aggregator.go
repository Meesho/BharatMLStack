package aggregator

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/scylla"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	queryCache   sync.Map
	aggregatorDb Database
	once         sync.Once
)

type Aggregator struct {
	Stores        map[string]Store
	configManager config.Manager
	sessionMap    map[int]*gocql.Session
}

type Store struct {
	Session   *gocql.Session
	TableName string
	Keyspace  string
}

const (
	envPrefix = "STORAGE_AGGREGATOR_DB"
)

// initAggregatorDb initializes the V1 database connection using configuration from Zookeeper.
// It returns a Database instance for further operations.
func initAggregatorDb() Database {
	if aggregatorDb == nil {
		once.Do(func() {
			stores := make(map[string]Store)
			configManager := config.NewManager(config.DefaultVersion)
			sessionMap := InitSessions()
			skyeConfig, err := configManager.GetSkyeConfig()
			etcdStores := skyeConfig.Storage.Stores
			if err != nil {
				log.Panic().Msgf("Error getting stores from etcd: %v", err)
			}
			for storeId, data := range etcdStores {
				store, err := initStore(data, sessionMap)
				if err != nil {
					log.Fatal().Msgf("Failed to initialize store %s: %v", storeId, err)
				}
				stores[storeId] = store
			}
			aggregatorDb = &Aggregator{
				Stores:        stores,
				configManager: configManager,
				sessionMap:    sessionMap,
			}
		})
	}
	return aggregatorDb
}

func InitSessions() map[int]*gocql.Session {
	connectionMap := make(map[int]*gocql.Session)
	count := appConfig.StorageAggregatorDbCount
	if count > 0 {
		for configId := 1; configId <= count; configId++ {
			configPrefix := fmt.Sprintf("%s_%d", envPrefix, configId)
			clusterConfig, err := scylla.BuildClusterConfigFromEnv(configPrefix)
			if err != nil {
				log.Panic().Msgf("error building scylla db cluster for configPrefix - %v with error %v"+
					"Error - ", configPrefix, err)
			}
			session, err := clusterConfig.CreateSession()
			if err != nil {
				log.Panic().Msgf("Error connecting scylla db.Error - %#v", err)
			}
			connectionMap[configId] = session
		}
	}
	return connectionMap
}

func initStore(data config.Data, sessionMap map[int]*gocql.Session) (Store, error) {
	if _, ok := sessionMap[data.ConfId]; !ok {
		return Store{}, fmt.Errorf("session not found for config id %d", data.ConfId)
	}
	return Store{
		Session:   sessionMap[data.ConfId],
		TableName: data.AggregatorTable,
		Keyspace:  viper.GetString(fmt.Sprintf("%s_%d_KEYSPACE", envPrefix, data.ConfId)),
	}, nil
}

// Query retrieves data from the database for a given candidateId.
func (a *Aggregator) Query(storeId string, query *Query) (map[string]interface{}, error) {
	t1 := time.Now()
	metric.Incr("aggregator_db_query_count", []string{metric.TagAsString("store_id", storeId)})
	preparedQuery := createRetrievePreparedQuery(a.Stores[storeId])
	aggregatorPayload := executeRetrieve(query, preparedQuery)
	metric.Timing("aggregator_db_retrieve_latency", time.Since(t1), []string{
		metric.TagAsString("store_id", storeId),
	})
	return aggregatorPayload, nil
}

// Persist persists data for a given candidateId.
func (a *Aggregator) Persist(storeId string, candidateId string, columns map[string]interface{}) error {
	t1 := time.Now()
	metric.Incr("aggregator_db_persist_count", []string{
		metric.TagAsString("store_id", storeId),
	})
	preparedQuery, sortedColumns := createPersistPreparedQuery(a.Stores[storeId], columns)
	err := executePersist(candidateId, columns, preparedQuery, sortedColumns)
	if err != nil {
		log.Error().Msgf("Error persisting data for candidate %v: %v\n", candidateId, err)
		return err
	}
	metric.Timing("aggregator_db_persist_latency", time.Since(t1), []string{
		metric.TagAsString("store_id", storeId),
	})
	return nil
}

func executeRetrieve(query *Query, preparedQuery *gocql.Query) map[string]interface{} {
	preparedQuery.Bind(query.CandidateId).Consistency(gocql.One)
	res, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("Error executing cql query %v: %v\n", query, err)
		return nil
	}

	if len(res) == 0 {
		return make(map[string]interface{})
	}
	return res[0]
}

func executePersist(candidateId string, columns map[string]interface{}, preparedQuery *gocql.Query, sortedColumns []string) error {
	var boundValues []interface{}
	for _, val := range sortedColumns {
		boundValues = append(boundValues, columns[val])
	}
	boundValues = append(boundValues, candidateId)
	preparedQuery.Bind(boundValues...)
	preparedQuery.Consistency(gocql.One)
	_, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("Error executing cql query %v: %v\n", candidateId, err)
		return err
	}
	return nil
}

func createRetrievePreparedQuery(store Store) *gocql.Query {
	cachedQuery, found := queryCache.Load(store.TableName + "_retrieve")
	var query string
	var preparedQuery *gocql.Query
	if !found {
		query = fmt.Sprintf(GenericRetrieveQuery, store.Keyspace, store.TableName, Id)
		queryCache.Store(store.TableName+"_retrieve", query)
		preparedQuery = store.Session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = store.Session.Query(query)
	}
	return preparedQuery
}

func createPersistPreparedQuery(store Store, columns map[string]interface{}) (*gocql.Query, []string) {
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
	columnsStr = columnsStr + ", " + Id
	cachedQuery, found := queryCache.Load(store.TableName + "_" + columnsStr + "_persist")
	if !found {
		placeholdersStr := strings.Join(placeholders, ", ")
		placeholdersStr = placeholdersStr + ", ?"
		query = fmt.Sprintf(GenericPersistQuery, store.Keyspace, store.TableName, columnsStr, placeholdersStr)
		queryCache.Store(store.TableName+"_"+columnsStr+"_persist", query)
		preparedQuery = store.Session.Query(query)
	} else {
		query = cachedQuery.(string)
		preparedQuery = store.Session.Query(query)
	}
	return preparedQuery, columnNames
}
