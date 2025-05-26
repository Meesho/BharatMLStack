package stores

import (
	"fmt"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/models"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"strings"
	"time"
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

	colPKMap, pkCols, err := r.configManager.GetPKMapAndPKColumnsForEntity(entityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error while getting PK and PK columns for entity: %s", entityLabel)
		return err
	}
	pipe := r.client.Pipeline()

	for _, row := range rows {
		// Create key from pkMap
		key := buildCacheKeyForPersist(row.PkMap, colPKMap, pkCols, entityLabel)

		csdb := blocks.NewCacheStorageDataBlock(1)

		var maxTtlAcrossFgs = uint64(0)
		// Process each PSDB in the row
		for fgId, psdb := range row.FgIdToPsDb {
			// Serialize the PSDB
			serializedData, err := psdb.Serialize()
			if err != nil {
				return fmt.Errorf("failed to serialize PSDB for fgId %d: %w", fgId, err)
			}
			// Deserialize into DDB
			ddb, err := blocks.DeserializePSDBWithoutDecompression(serializedData)
			if err != nil {
				return fmt.Errorf("failed to deserialize PSDB for fgId %d: %w", fgId, err)
			}
			// Add to CSDB's FGIdToDDB mapping
			err = csdb.AddFGIdToDDB(fgId, ddb)
			if err != nil {
				return fmt.Errorf("failed to add FGId to DDB mapping for fgId %d: %w", fgId, err)
			}
			maxTtlAcrossFgs = max(maxTtlAcrossFgs, ddb.ExpiryAt-uint64(time.Now().Unix()))
		}

		// Serialize CSDB for distributed cache
		serializedCSDB, err := csdb.SerializeForDistributedCache()
		if err != nil {
			return fmt.Errorf("failed to serialize CSDB for key %s: %w", key, err)
		}

		// Add SET command to pipeline
		pipe.Set(r.ctx, key, serializedCSDB, time.Duration(maxTtlAcrossFgs)*time.Second)
	}

	// Execute the pipeline
	_, err2 := pipe.Exec(r.ctx)
	if err2 != nil {
		return fmt.Errorf("failed to execute batch persist: %w", err2)
	}

	return nil
}

func (r *RedisStore) BatchRetrieveV2(entityLabel string, pkMaps []map[string]string, fgIds []int) ([]map[int]*blocks.DeserializedPSDB, error) {
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

		// Filter requested FGIds
		fgIdToDDB, err := csdb.GetDeserializedPSDBForFGIds(ds.NewOrderedSetFromSlice(fgIds))
		if err != nil {
			return nil, err
		}
		results[i] = fgIdToDDB
	}

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
