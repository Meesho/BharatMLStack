package stores

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/models"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/go-redsync/redsync/v4"
	redsyncgoredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

const (
	LockMaxAttempts   = 5
	LockBaseDelay     = 100 * time.Millisecond
	LockMaxDelay      = 2 * time.Second
	LockJitterDivisor = 4
	LockExpiry        = 5 * time.Second
	LockDriftFactor   = 0.01
)

type RedisStore struct {
	client        redis.UniversalClient
	configManager config.Manager
	ctx           context.Context
	rs            *redsync.Redsync
}

func NewRedisStore(connection *infra.RedisFailoverConnection) (*RedisStore, error) {
	client, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)
	pool := redsyncgoredis.NewPool(client.(redis.UniversalClient))
	rs := redsync.New(pool)

	return &RedisStore{
		client:        client.(redis.UniversalClient),
		ctx:           context.Background(),
		configManager: configManager,
		rs:            rs,
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

	// Lock all keys that will be modified to prevent race conditions
	uniqueKeys := DeduplicateKeys(keysToRead)
	locks, err := LockKeys(r.ctx, r.rs, uniqueKeys)
	if err != nil {
		return fmt.Errorf("failed to acquire locks for keys: %w", err)
	}
	defer UnlockKeys(locks)

	var existingValues []interface{}
	if !canReplace {
		existingValues, err = r.client.MGet(r.ctx, keysToRead...).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to read existing data from redis store for entity %s: %w", entityLabel, err)
		}
	}

	pipe := r.client.Pipeline()

	for i, key := range keysToRead {
		rowsForKey := keyToRowsMap[key]

		var finalCSDB *blocks.CacheStorageDataBlock
		var maxTtl uint64

		if canReplace {
			finalCSDB, maxTtl, err = r.createCSDBFromRows(rowsForKey)
			if err != nil {
				return fmt.Errorf("failed to create CSDB from rows for key %s: %w", key, err)
			}
		} else {
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

			finalCSDB, maxTtl, err = r.mergeRowsIntoCSDB(existingCSDB, rowsForKey)
			if err != nil {
				return fmt.Errorf("failed to merge rows for key %s: %w", key, err)
			}
		}

		serializedCSDB, err := finalCSDB.SerializeForDistributedCache()
		if err != nil {
			return fmt.Errorf("failed to serialize final CSDB for key %s: %w", key, err)
		}

		pipe.Set(r.ctx, key, serializedCSDB, time.Duration(maxTtl)*time.Second)
	}

	_, err2 := pipe.Exec(r.ctx)
	if err2 != nil {
		metric.Count("persist_query_failure", int64(len(rows)), []string{"entity_name", entityLabel, "db_type", "redis"})
		return fmt.Errorf("failed to execute batch persist: %w", err2)
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
			results[i] = nil
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

// LockKeys acquires locks for multiple keys in a consistent order to prevent deadlocks
func LockKeys(ctx context.Context, rs *redsync.Redsync, keys []string) ([]*redsync.Mutex, error) {
	sort.Strings(keys) // Prevent deadlocks by consistent ordering
	var locks []*redsync.Mutex

	for _, key := range keys {
		mutex, err := TryLockWithBackoff(ctx, rs, key, LockMaxAttempts)
		if err != nil {
			// Unlock all previously acquired locks
			for _, m := range locks {
				if ok, unlockErr := m.Unlock(); !ok || unlockErr != nil {
					log.Error().Err(unlockErr).Msg("Failed to release lock during cleanup")
				}
			}
			metric.Count("lock_acquire_failure", 1, []string{"key", key, "error", err.Error()})
			return nil, fmt.Errorf("failed to acquire lock for key %s: %w", key, err)
		}

		locks = append(locks, mutex)
	}
	return locks, nil
}

// UnlockKeys releases all acquired locks
func UnlockKeys(locks []*redsync.Mutex) {
	for _, l := range locks {
		if ok, err := l.Unlock(); !ok || err != nil {
			metric.Count("lock_release_failure", 1, []string{"key", l.Name(), "error", err.Error()})
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

func TryLockWithBackoff(ctx context.Context, rs *redsync.Redsync, key string, maxAttempts int) (*redsync.Mutex, error) {
	baseDelay := LockBaseDelay
	maxDelay := LockMaxDelay

	mutex := rs.NewMutex("lock:"+key,
		redsync.WithExpiry(LockExpiry),
		redsync.WithDriftFactor(LockDriftFactor),
	)

	var attempt int
	for {
		err := mutex.LockContext(ctx)
		if err == nil {
			return mutex, nil
		}

		attempt++
		if attempt >= maxAttempts {
			return nil, err
		}

		// Exponential backoff with jitter
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		if delay > maxDelay {
			delay = maxDelay
		}
		jitter := time.Duration(rand.Int63n(int64(delay / LockJitterDivisor)))
		delay += jitter

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
