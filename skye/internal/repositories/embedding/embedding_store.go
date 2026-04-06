package embedding

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/pkg/ds"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/scylla"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
)

type EmbeddingStore struct {
	Stores        map[string]StoreData
	configManager config.Manager
	sessionMap    map[int]*gocql.Session
}

type StoreData struct {
	Session   *gocql.Session
	TableName string
	Keyspace  string
}

const (
	envPrefix         = "STORAGE_EMBEDDING_STORE"
	searchColumns     = "search_embedding"
	embeddingColumns  = "embedding, search_embedding"
	dotProductColumns = "embedding"
	persistColumns    = "*"
)

// initEmbeddingStore initializes the embedding store and returns it.
func initEmbeddingStore() Store {
	if embeddingStore == nil {
		once.Do(func() {
			queryCache = ds.NewSyncMap[string, string]()
			sessionMap := InitSessions()
			stores, err := initializeStores(sessionMap)
			if err != nil {
				log.Panic().Msgf("Failed to initialize stores: %v", err)
			}
			embeddingStore = &EmbeddingStore{
				Stores:        stores,
				configManager: config.NewManager(config.DefaultVersion),
				sessionMap:    sessionMap,
			}
		})
	}
	return embeddingStore
}

// initializeStores sets up the store connections and returns a map of StoreData.
func initializeStores(sessionMap map[int]*gocql.Session) (map[string]StoreData, error) {
	stores := make(map[string]StoreData)
	configManager := config.NewManager(config.DefaultVersion)
	skyeConfig, err := configManager.GetSkyeConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting stores from etcd: %w", err)
	}
	etcdStores := skyeConfig.Storage.Stores
	for storeId, data := range etcdStores {
		store, err := createStoreData(data, sessionMap)
		if err != nil {
			log.Error().Msgf("Failed to create store data for storeId %s: %v", storeId, err)
			continue // Log error but continue with other stores
		}
		stores[storeId] = store
	}
	return stores, nil
}

func InitSessions() map[int]*gocql.Session {
	connectionMap := make(map[int]*gocql.Session)
	count := appConfig.StorageEmbeddingStoreCount
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

// createStoreData builds the StoreData from the given store configuration.
func createStoreData(data config.Data, sessionMap map[int]*gocql.Session) (StoreData, error) {
	if _, ok := sessionMap[data.ConfId]; !ok {
		return StoreData{}, fmt.Errorf("session not found for config id %d", data.ConfId)
	}
	return StoreData{
		Session:   sessionMap[data.ConfId],
		TableName: data.EmbeddingTable,
		Keyspace:  viper.GetString(fmt.Sprintf("%s_%d_KEYSPACE", envPrefix, data.ConfId)),
	}, nil
}

// BulkQuery retrieves the embeddings for the given candidate IDs.
func (e *EmbeddingStore) BulkQuery(storeId string, bulkQuery *BulkQuery, queryType string) error {
	startTime := time.Now()
	metric.Count("embedding_store_db_retrieve_count", int64(len(bulkQuery.CacheKeys)), []string{
		metric.TagAsString("store_id", storeId),
	})
	if queryType == "embedding" {
		preparedQuery := createPreparedQuery(e.Stores[storeId], embeddingColumns, queryType)
		err := bulkExecuteAsyncForEmbedding(bulkQuery, preparedQuery)
		if err != nil {
			return err
		}
	} else if queryType == "similar_candidate" {
		preparedQuery := createPreparedQuery(e.Stores[storeId], searchColumns, queryType)
		err := bulkExecuteAsync(bulkQuery, preparedQuery)
		if err != nil {
			return err
		}
	} else if queryType == "dot_product" {
		preparedQuery := createPreparedQuery(e.Stores[storeId], dotProductColumns, queryType)
		err := bulkExecuteAsync(bulkQuery, preparedQuery)
		if err != nil {
			return err
		}
	}
	metric.Timing("embedding_store_db_retrieve_latency", time.Since(startTime), []string{
		metric.TagAsString("store_id", storeId),
	})
	return nil
}

// BulkQuery retrieves the embeddings for the given candidate IDs.
func (e *EmbeddingStore) BulkQueryConsumer(storeId string, bulkQuery *BulkQuery) (map[string]map[string]interface{}, error) {
	t1 := time.Now()
	metric.Count("embedding_store_db_retrieve_count", int64(len(bulkQuery.CandidateIds)), []string{
		metric.TagAsString("store_id", storeId),
	})
	preparedQuery := createPreparedQuery(e.Stores[storeId], persistColumns, "consumer")
	embeddingStorePayloads := bulkExecuteAsyncForConsumer(bulkQuery, preparedQuery)
	metric.Timing("embedding_store_db_retrieve_latency", time.Since(t1), []string{
		metric.TagAsString("store_id", storeId),
	})
	return *embeddingStorePayloads, nil
}

// Persist stores the given payload in the embedding store with an optional TTL.
func (e *EmbeddingStore) Persist(storeId string, ttl int, payload Payload) error {
	startTime := time.Now()
	metric.Incr("embedding_store_db_persist_count", []string{metric.TagAsString("store_id", storeId)})
	columns, err := preparePersistColumns(payload)
	if err != nil {
		log.Error().Msgf("Error preparing columns for candidate %v: %v\n", payload.CandidateId, err)
		return err
	}
	preparedQuery, columnNames := createPersistPreparedQuery(e.Stores[storeId], columns, ttl)
	if err := executePersist(payload, columns, preparedQuery, columnNames); err != nil {
		log.Error().Msgf("Error persisting data for candidate %v: %v\n", payload.CandidateId, err)
		metric.Incr("embedding_store_db_persist_failure_count", []string{
			metric.TagAsString("store_id", storeId),
		})
		return err
	}
	metric.Timing("embedding_store_db_persist_latency", time.Since(startTime), []string{
		metric.TagAsString("store_id", storeId),
	})
	return nil
}

// preparePersistColumns prepares the column data for persisting.
func preparePersistColumns(payload Payload) (map[string]interface{}, error) {
	columns := map[string]interface{}{
		"id":               payload.CandidateId,
		"model_name":       payload.Model,
		"embedding":        payload.Embedding,
		"search_embedding": payload.SearchEmbedding,
		"version":          payload.Version,
	}

	for key, value := range payload.VariantsIndexMap {
		columns[key+"_to_be_indexed"] = value
	}

	return columns, nil
}

func bulkExecuteAsync(bulkQuery *BulkQuery, preparedQuery *gocql.Query) error {
	var wg sync.WaitGroup
	type embeddingResult struct {
		key       string
		embedding []float32
	}

	resultChan := make(chan embeddingResult, len(bulkQuery.CacheKeys))
	// Pre-extract candidateIds (single-threaded, fast)
	candidateIds := make(map[string]string, len(bulkQuery.CacheKeys))
	for key, cacheStruct := range bulkQuery.CacheKeys {
		candidateIds[key] = cacheStruct.CandidateId
	}

	for key := range bulkQuery.CacheKeys {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			// Avoid copying query by recreating from session if needed
			query := *preparedQuery
			(&query).Bind(bulkQuery.Model, bulkQuery.Version, candidateIds[key]).Consistency(gocql.One)
			var embedding []float32
			err := (&query).Scan(&embedding)
			if err != nil && err != gocql.ErrNotFound {
				metric.Incr("embedding_store_db_retrieve_failure", []string{"db", "scylla"})
				log.Error().Msgf("Error executing cql query for key %s: %v\n", key, err)
				return
			}
			resultChan <- embeddingResult{key: key, embedding: embedding}
		}(key)
	}

	wg.Wait()
	close(resultChan)

	for res := range resultChan {
		cacheStruct := bulkQuery.CacheKeys[res.key]
		cacheStruct.Embedding = res.embedding
		bulkQuery.CacheKeys[res.key] = cacheStruct
	}
	return nil
}

func bulkExecuteAsyncForEmbedding(bulkQuery *BulkQuery, preparedQuery *gocql.Query) error {
	var wg sync.WaitGroup
	type embeddingResult struct {
		key             string
		embedding       []float32
		searchEmbedding []float32
	}

	resultChan := make(chan embeddingResult, len(bulkQuery.CacheKeys))

	candidateIds := make(map[string]string, len(bulkQuery.CacheKeys))
	for key, cacheStruct := range bulkQuery.CacheKeys {
		candidateIds[key] = cacheStruct.CandidateId
	}

	for key := range bulkQuery.CacheKeys {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			query := *preparedQuery
			(&query).Bind(bulkQuery.Model, bulkQuery.Version, candidateIds[key]).Consistency(gocql.One)
			var embedding []float32
			var searchEmbedding []float32
			err := (&query).Scan(&embedding, &searchEmbedding)
			if err != nil && err != gocql.ErrNotFound {
				metric.Incr("embedding_store_db_retrieve_failure", []string{"db", "scylla"})
				log.Error().Msgf("Error executing cql query for key %s: %v\n", key, err)
				return
			}
			resultChan <- embeddingResult{key: key, embedding: embedding, searchEmbedding: searchEmbedding}
		}(key)
	}

	wg.Wait()
	close(resultChan)

	for res := range resultChan {
		cacheStruct := bulkQuery.CacheKeys[res.key]
		cacheStruct.Embedding = res.embedding
		cacheStruct.SearchEmbedding = res.searchEmbedding
		bulkQuery.CacheKeys[res.key] = cacheStruct
	}
	return nil
}

func bulkExecuteAsyncForConsumer(bulkQuery *BulkQuery, preparedQuery *gocql.Query) *map[string]map[string]interface{} {
	var wg sync.WaitGroup
	var mu sync.Mutex
	payloads := make(map[string]map[string]interface{})
	for _, candidateId := range bulkQuery.CandidateIds {
		wg.Add(1)
		go func(candidateId string) {
			defer wg.Done()
			query := *preparedQuery
			query.Bind(bulkQuery.Model, bulkQuery.Version, candidateId).Consistency(gocql.One)
			res, err := query.Iter().SliceMap()
			if err != nil {
				metric.Incr("embedding_store_db_retrieve_failure", []string{"db", "scylla"})
				log.Error().Msgf("Error executing cql query %v: %v\n", query, err)
				return
			}
			mu.Lock()
			if len(res) != 0 {
				payloads[candidateId] = res[0]
			}
			mu.Unlock()
		}(candidateId)
	}

	wg.Wait()
	return &payloads
}

func executePersist(payload Payload, columns map[string]interface{}, preparedQuery *gocql.Query, columnNames []string) error {
	var bindValues []interface{}
	for _, column := range columnNames {
		bindValues = append(bindValues, columns[column])
	}
	preparedQuery.Bind(bindValues...)
	preparedQuery.Consistency(gocql.One)
	_, err := preparedQuery.Iter().SliceMap()
	if err != nil {
		log.Error().Msgf("Error executing cql query %v: %v\n", payload.CandidateId, err)
		return err
	}
	return nil
}

func createPreparedQuery(store StoreData, column string, queryType string) *gocql.Query {
	cachedQuery, found := queryCache.Get(store.TableName + "_retrieve_" + queryType)
	var query string
	var preparedQuery *gocql.Query
	if !found {
		query = fmt.Sprintf(GenericRetrieveQuery, column, store.Keyspace, store.TableName, ModelName, Version, Id)
		queryCache.Set(store.TableName+"_retrieve_"+queryType, query)
		preparedQuery = store.Session.Query(query)
	} else {
		preparedQuery = store.Session.Query(cachedQuery)
	}
	return preparedQuery
}

func createPersistPreparedQuery(store StoreData, columns map[string]interface{}, ttl int) (*gocql.Query, []string) {
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
	cachedQuery, found := queryCache.Get(store.TableName + "_" + columnsStr + "_persist")
	if !found {
		placeholdersStr := strings.Join(placeholders, ", ")
		query = fmt.Sprintf(GenericPersistQuery, store.Keyspace, store.TableName, columnsStr, placeholdersStr, ttl)
		queryCache.Set(store.TableName+"_"+columnsStr+"_persist", query)
		preparedQuery = store.Session.Query(query)
	} else {
		query = cachedQuery
		preparedQuery = store.Session.Query(query)
	}
	return preparedQuery, columnNames
}
