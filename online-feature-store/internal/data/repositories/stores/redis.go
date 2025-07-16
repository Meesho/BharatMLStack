package stores

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/models"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

const (
	LockKeyPrefix     = "l:"
	LockMaxRetries    = 3
	LockBackoffBaseMs = 100
)

type RedisStore struct {
	client        redis.UniversalClient
	configManager config.Manager
	ctx           context.Context
}

func NewRedisStore(connection *infra.RedisFailoverConnection) (*RedisStore, error) {
	client, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)

	return &RedisStore{
		client:        client.(redis.UniversalClient),
		ctx:           context.Background(),
		configManager: configManager,
	}, nil
}

func (r *RedisStore) PersistV2(storeId string, entityLabel string, pkMap map[string]string, fgIdToPsDb map[int]*blocks.PermStorageDataBlock) error {
	return fmt.Errorf("%w: PersistV2 for Redis store", ErrNotImplemented)
}

func (r *RedisStore) RetrieveV2(entityLabel string, pkMap map[string]string, fgIds []int) (map[int]*blocks.DeserializedPSDB, error) {
	return nil, fmt.Errorf("%w: RetrieveV2 for Redis store", ErrNotImplemented)
}

func (r *RedisStore) BatchPersistV2(storeId string, entityLabel string, rows []models.Row) error {
	metric.Count("db_persist_count", int64(len(rows)), []string{"entity_label", entityLabel, "db_type", "redis"})
	colPKMap, pkCols, err := r.configManager.GetPKMapAndPKColumnsForEntity(entityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error while getting PK and PK columns for entity: %s", entityLabel)
		return err
	}

	allFGIds, err := r.configManager.GetAllFGIdsForEntity(entityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error while getting all FGIDs for entity: %s", entityLabel)
		return err
	}

	batchFGIds := make(map[int]bool)
	for _, row := range rows {
		for fgId := range row.FgIdToPsDb {
			batchFGIds[fgId] = true
		}
	}

	canReplace := len(batchFGIds) == len(allFGIds)
	if canReplace {
		for fgId := range allFGIds {
			if !batchFGIds[fgId] {
				canReplace = false
				break
			}
		}
	}

	keyToRowsMap := make(map[string][]models.Row)
	keysToRead := make([]string, 0)

	for _, row := range rows {
		key := buildCacheKeyForPersist(row.PkMap, colPKMap, pkCols, entityLabel)
		if _, exists := keyToRowsMap[key]; !exists {
			keysToRead = append(keysToRead, key)
			keyToRowsMap[key] = make([]models.Row, 0)
		}
		keyToRowsMap[key] = append(keyToRowsMap[key], row)
	}

	if canReplace {
		return r.batchPersistReplace(entityLabel, keysToRead, keyToRowsMap)
	} else {
		return r.batchPersistMerge(entityLabel, keysToRead, keyToRowsMap)
	}
}

func (r *RedisStore) batchPersistReplace(entityLabel string, keysToRead []string, keyToRowsMap map[string][]models.Row) error {
	pipe := r.client.Pipeline()

	for _, key := range keysToRead {
		rowsForKey := keyToRowsMap[key]

		finalCSDB, maxTtl, err := r.createCSDBFromRows(rowsForKey)
		if err != nil {
			return fmt.Errorf("failed to create CSDB from rows for key %s: %w", key, err)
		}

		serializedCSDB, err := finalCSDB.SerializeForDistributedCache()
		if err != nil {
			return fmt.Errorf("failed to serialize final CSDB for key %s: %w", key, err)
		}

		pipe.Set(r.ctx, key, serializedCSDB, time.Duration(maxTtl)*time.Second)
	}

	_, err := pipe.Exec(r.ctx)
	if err != nil {
		metric.Count("persist_query_failure", int64(len(keysToRead)), []string{"entity_name", entityLabel, "db_type", "redis"})
		return fmt.Errorf("failed to execute batch persist replace: %w", err)
	}

	return nil
}

func (r *RedisStore) batchPersistMerge(entityLabel string, keysToRead []string, keyToRowsMap map[string][]models.Row) error {
	// Lock all keys that will be modified to prevent race conditions
	uniqueKeys := DeduplicateKeys(keysToRead)
	acquiredLocks, err := LockKeys(r.ctx, r.client, uniqueKeys, entityLabel)
	defer UnlockKeys(r.ctx, r.client, acquiredLocks, entityLabel)

	existingValues, err := r.client.MGet(r.ctx, keysToRead...).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to read existing data from redis store for entity %s: %w", entityLabel, err)
	}

	pipe := r.client.Pipeline()

	for i, key := range keysToRead {
		rowsForKey := keyToRowsMap[key]

		var existingCSDB *blocks.CacheStorageDataBlock
		if existingValues[i] != nil {
			existingData := []byte(existingValues[i].(string))
			existingCSDB, err = blocks.CreateCSDBForDistributedCache(existingData)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to parse existing CSDB for key %s, creating new", key)
				existingCSDB = blocks.NewCacheStorageDataBlock(1)
			}
		} else {
			existingCSDB = blocks.NewCacheStorageDataBlock(1)
		}

		finalCSDB, maxTtl, err := r.mergeRowsIntoCSDB(existingCSDB, rowsForKey)
		if err != nil {
			return fmt.Errorf("failed to merge rows for key %s: %w", key, err)
		}

		serializedCSDB, err := finalCSDB.SerializeForDistributedCache()
		if err != nil {
			return fmt.Errorf("failed to serialize final CSDB for key %s: %w", key, err)
		}

		pipe.Set(r.ctx, key, serializedCSDB, time.Duration(maxTtl)*time.Second)
	}

	_, err = pipe.Exec(r.ctx)
	if err != nil {
		metric.Count("persist_query_failure", int64(len(keysToRead)), []string{"entity_name", entityLabel, "db_type", "redis"})
		return fmt.Errorf("failed to execute batch persist merge: %w", err)
	}

	return nil
}

func (r *RedisStore) createCSDBFromRows(rows []models.Row) (*blocks.CacheStorageDataBlock, uint64, error) {
	csdb := blocks.NewCacheStorageDataBlock(1)
	maxTtlAcrossFgs := uint64(0)
	currentTime := uint64(time.Now().Unix())

	for _, row := range rows {
		for fgId, psdb := range row.FgIdToPsDb {
			serializedData, err := psdb.Serialize()
			if err != nil {
				return nil, 0, fmt.Errorf("failed to serialize PSDB for fgId %d: %w", fgId, err)
			}

			ddb, err := blocks.DeserializePSDBWithoutDecompression(serializedData)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to deserialize PSDB for fgId %d: %w", fgId, err)
			}

			if !ddb.Expired && ddb.ExpiryAt > currentTime {
				err = csdb.AddFGIdToDDB(fgId, ddb)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to add FGId to CSDB for fgId %d: %w", fgId, err)
				}
				maxTtlAcrossFgs = max(maxTtlAcrossFgs, ddb.ExpiryAt-currentTime)
			}
		}
	}

	return csdb, maxTtlAcrossFgs, nil
}

func (r *RedisStore) mergeRowsIntoCSDB(existingCSDB *blocks.CacheStorageDataBlock, rows []models.Row) (*blocks.CacheStorageDataBlock, uint64, error) {
	var existingFGIdtoCSDBMap map[int]*blocks.DeserializedPSDB
	if len(existingCSDB.GetSerializedData()) > 0 {
		var err error
		existingFGIdtoCSDBMap, err = existingCSDB.GetDeserializedPSDBForAllFGIds()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to deserialize existing FGs, proceeding with new data only")
			existingFGIdtoCSDBMap = make(map[int]*blocks.DeserializedPSDB)
		}
	} else {
		existingFGIdtoCSDBMap = make(map[int]*blocks.DeserializedPSDB)
	}

	mergedCSDB := blocks.NewCacheStorageDataBlock(1)

	maxTtlAcrossFgs := uint64(0)
	currentTime := uint64(time.Now().Unix())

	for fgId, ddb := range existingFGIdtoCSDBMap {
		if !ddb.Expired && ddb.ExpiryAt > currentTime {
			err := mergedCSDB.AddFGIdToDDB(fgId, ddb.Copy())
			if err != nil {
				return nil, 0, fmt.Errorf("failed to add existing fg id %d to ddb: %w", fgId, err)
			}
			maxTtlAcrossFgs = max(maxTtlAcrossFgs, ddb.ExpiryAt-currentTime)
		}
	}

	for _, row := range rows {
		for fgId, psdb := range row.FgIdToPsDb {
			serializedData, err := psdb.Serialize()
			if err != nil {
				return nil, 0, fmt.Errorf("failed to serialize PSDB for fgId %d: %w", fgId, err)
			}

			ddb, err := blocks.DeserializePSDBWithoutDecompression(serializedData)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to deserialize PSDB for fgId %d: %w", fgId, err)
			}

			if !ddb.Expired && ddb.ExpiryAt > currentTime {
				err = mergedCSDB.AddFGIdToDDB(fgId, ddb)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to add FGId to merged CSDB for fgId %d: %w", fgId, err)
				}
				maxTtlAcrossFgs = max(maxTtlAcrossFgs, ddb.ExpiryAt-currentTime)
			}
		}
	}

	return mergedCSDB, maxTtlAcrossFgs, nil
}

func (r *RedisStore) BatchRetrieveV2(entityLabel string, pkMaps []map[string]string, fgIds []int) ([]map[int]*blocks.DeserializedPSDB, error) {
	t1 := time.Now()
	metric.Count("db_retrieve_count", int64(len(pkMaps)), []string{"entity_label", entityLabel, "db_type", "redis"})
	results := make([]map[int]*blocks.DeserializedPSDB, len(pkMaps))

	colPKMap, pkCols, err := r.configManager.GetPKMapAndPKColumnsForEntity(entityLabel)
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(pkMaps))
	for i, pkMap := range pkMaps {
		keys[i] = buildCacheKeyForPersist(pkMap, colPKMap, pkCols, entityLabel)
	}

	values, err := r.client.MGet(r.ctx, keys...).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	for i, value := range values {
		if value == nil {
			// No data found for this key, create negative cache entries for all fgIds
			results[i] = make(map[int]*blocks.DeserializedPSDB)
			for _, fgId := range fgIds {
				results[i][fgId] = &blocks.DeserializedPSDB{
					NegativeCache: true,
				}
			}
			continue
		}

		data := []byte(value.(string))
		csdb, err := blocks.CreateCSDBForDistributedCache(data)
		if err != nil {
			return nil, err
		}

		fgIdToDDB, err := csdb.GetDeserializedPSDBForFGIds(ds.NewOrderedSetFromSlice(fgIds))
		if err != nil {
			return nil, err
		}
		results[i] = fgIdToDDB
	}
	metric.Timing("db_retrieve_latency", time.Since(t1), []string{"entity_label", entityLabel, "db_type", "redis"})
	return results, nil
}

func (r *RedisStore) Type() string {
	return StoreTypeRedis
}

func buildCacheKeyForPersist(pkMap map[string]string, colPKMap map[string]string, pkCols []string, entityLabel string) string {
	keys := make([]string, 0, len(pkMap))

	for _, pkCol := range pkCols {
		keys = append(keys, pkMap[colPKMap[pkCol]])
	}

	return entityLabel + ":" + strings.Join(keys, "|")
}

// LockKeys acquires locks for multiple keys using SETNX with retries
func LockKeys(ctx context.Context, client redis.UniversalClient, keys []string, entityLabel string) ([]string, error) {
	sort.Strings(keys) // Prevent deadlocks by consistent ordering
	var acquiredLocks []string

	// Round current timestamp to 5 seconds
	currentTime := time.Now().Unix()
	roundedTime := (currentTime / 5) * 5

	for _, key := range keys {
		lockKey := fmt.Sprintf("%s%s:%d", LockKeyPrefix, key, roundedTime)

		// Retry logic for lock acquisition
		var success bool
		var err error

		for attempt := 0; attempt < LockMaxRetries; attempt++ {
			success, err = client.SetNX(ctx, lockKey, "1", 0).Result()
			if err == nil && success {
				break
			}

			if attempt < LockMaxRetries-1 {
				// Wait before retry (exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms)
				backoffTime := time.Duration(LockBackoffBaseMs*(1<<attempt)) * time.Millisecond
				select {
				case <-time.After(backoffTime):
				case <-ctx.Done():
					return acquiredLocks, ctx.Err()
				}
			}
		}

		if err != nil || !success {
			metric.Count("fs_persist_lock_acquire_failure", 1, []string{"entity_label", entityLabel})
			continue
		}
		acquiredLocks = append(acquiredLocks, lockKey)
	}

	return acquiredLocks, nil
}

// UnlockKeys releases all acquired locks
func UnlockKeys(ctx context.Context, client redis.UniversalClient, lockKeys []string, entityLabel string) {
	for _, lockKey := range lockKeys {
		if err := client.Del(ctx, lockKey).Err(); err != nil {
			metric.Count("fs_persist_lock_release_failure", 1, []string{"entity_label", entityLabel})
			log.Error().Err(err).Msg("Failed to release lock")
		}
	}
}

// DeduplicateKeys removes duplicate keys from the slice while preserving order
func DeduplicateKeys(keys []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, key := range keys {
		if !seen[key] {
			seen[key] = true
			result = append(result, key)
		}
	}
	return result
}
