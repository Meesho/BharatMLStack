package feature

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config/enums"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/models"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/stores"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/rs/zerolog/log"
)

var (
	fpV2Once    sync.Once
	fpV2Handler *PersistHandler
)

type InvalidEventError struct {
	message string
}

func (e *InvalidEventError) Error() string {
	return e.message
}

func NewInvalidEventError(message string) *InvalidEventError {
	return &InvalidEventError{message: message}
}

type PersistHandler struct {
	persist.FeatureServiceServer
	config      config.Manager
	imcProvider provider.CacheProvider
	dcProvider  provider.CacheProvider
	dbProvider  provider.StoreProvider
}

func InitPersistHandler() *PersistHandler {
	if fpV2Handler == nil {
		fpV2Once.Do(func() {
			configManager := config.Instance(config.DefaultVersion)
			fpV2Handler = &PersistHandler{
				config:      configManager,
				imcProvider: provider.InMemoryCacheProviderImpl,
				dcProvider:  provider.DistributedCacheProviderImpl,
				dbProvider:  provider.StorageProviderImpl,
			}
		})
	}
	return fpV2Handler
}

func (h *PersistHandler) PersistFeatures(ctx context.Context, persistQuery *persist.Query) (*persist.Result, error) {
	_, err := h.Persist(ctx, persistQuery, enums.ConsistencyStrong)
	if err != nil {
		return nil, err
	}
	return &persist.Result{
		Message: "Data Persisted Successfully",
	}, nil
}

func (p *PersistHandler) Persist(ctx context.Context, persistQuery *persist.Query, consistency enums.Consistency) (*PersistData, error) {
	log.Debug().Msgf("Persist Request %s", persistQuery)
	startTime := time.Now()
	persistData := &PersistData{
		Query: persistQuery,
	}
	err := p.preProcessRequest(persistData)
	if err != nil {
		return nil, err
	}
	err = p.preparePersistData(persistData)
	if err != nil {
		return nil, err
	}
	err = p.PersistToDb(persistData)
	if err != nil {
		return nil, err
	}
	if persistData.IsAnyFGInMemCached && consistency == enums.ConsistencyStrong {
		err = p.RemoveFromInMemoryCache(persistData)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to remove from in-memory cache for key: %v", persistData.Query.Data)
		}
	}
	if persistData.IsAnyFGDistCached && consistency == enums.ConsistencyStrong {
		err = p.RemoveFromDistributedCache(persistData)
		if err != nil {
			return nil, err
		}
	}
	metric.Timing("orchestrator_persist_latency", time.Since(startTime), []string{"entity", persistQuery.EntityLabel})
	return persistData, nil
}

func (p *PersistHandler) preProcessRequest(persistData *PersistData) error {
	query := persistData.Query
	persistData.EntityLabel = query.EntityLabel
	err := p.preProcessEntity(persistData)
	if err != nil {
		return err
	}
	err = p.preProcessAndValidateFeatureSchema(persistData)
	if err != nil {
		return err
	}
	return nil

}

func (p *PersistHandler) preProcessEntity(persistData *PersistData) error {
	entityLabel := persistData.EntityLabel

	// Get entity and validate
	entity, err := p.config.GetEntity(entityLabel)
	if err != nil {
		return fmt.Errorf("invalid entity %s: %w", entityLabel, err)
	}

	// Pre-allocate maps based on entity's feature groups
	numEntityFGs := len(entity.FeatureGroups)
	persistData.AllFGIdToFgConf = make(map[int]FgConf, numEntityFGs)
	persistData.AllFGLabelToFGId = make(map[string]int, numEntityFGs)

	// Initialize sets with capacity
	persistData.IsAnyFGDistCached = false
	persistData.IsAnyFGInMemCached = false

	// Process all feature groups from entity
	for fgLabel, fg := range entity.FeatureGroups {

		dataType, err := types.ParseDataType(fg.DataType.String())
		if err != nil {
			return fmt.Errorf("invalid data type for feature group %s: %w", fgLabel, err)
		}

		fgConf := FgConf{
			DataType:    dataType,
			FeatureMeta: fg.Features[fg.ActiveVersion].FeatureMeta,
		}
		// Store all mappings regardless of request
		persistData.AllFGLabelToFGId[fgLabel] = fg.Id
		persistData.AllFGIdToFgConf[fg.Id] = fgConf
		// Track if any FG is distributed cached
		if fg.DistributedCacheEnabled && entity.DistributedCache.Enabled {
			persistData.IsAnyFGDistCached = true
		}

		// Track if any FG is in memory cached
		if fg.InMemoryCacheEnabled && entity.InMemoryCache.Enabled {
			persistData.IsAnyFGInMemCached = true
		}

	}

	return nil
}

func (p *PersistHandler) preProcessAndValidateFeatureSchema(persistData *PersistData) error {
	query := persistData.Query
	entityLabel := persistData.EntityLabel

	// Validate entity exists first
	entityConf, err := p.config.GetEntity(entityLabel)
	if err != nil || entityConf == nil {
		return NewInvalidEventError(fmt.Sprintf("invalid entity %s: %v", entityLabel, err))
	}
	persistData.StoreIdToMaxColumnSize = make(map[string]int)

	for _, fg := range query.FeatureGroupSchema {
		fgConf, ok := entityConf.FeatureGroups[fg.Label]
		if !ok {
			return NewInvalidEventError(fmt.Sprintf("feature group %s not found for entity : %s", fg.Label, entityLabel))
		}
		storeConf, err := p.config.GetStore(fgConf.StoreId)
		if err != nil {
			return fmt.Errorf("failed to get store for feature group %s: %w", fg.Label, err)
		}
		persistData.StoreIdToMaxColumnSize[fgConf.StoreId] = storeConf.MaxColumnSizeInBytes
	}

	return nil
}

func (p *PersistHandler) preparePersistData(persistData *PersistData) error {
	persistData.StoreIdToRows = make(map[string][]Row)
	for rowIndex, data := range persistData.Query.Data {
		pkMap := make(map[string]string)
		for index := range data.GetKeyValues() {
			pkMap[persistData.Query.KeysSchema[index]] = data.GetKeyValues()[index]
		}
		for fgIndex, fgSchema := range persistData.Query.FeatureGroupSchema {
			fgId := persistData.AllFGLabelToFGId[fgSchema.GetLabel()]
			fgConf, err := p.config.GetFeatureGroup(persistData.EntityLabel, fgSchema.GetLabel())
			if err != nil {
				return fmt.Errorf("failed to get feature group %s: %w", fgSchema.GetLabel(), err)
			}
			featureData, err := system.ParseFeatureValue(fgSchema.GetFeatureLabels(), data.GetFeatureValues()[fgIndex], persistData.AllFGIdToFgConf[fgId].DataType, persistData.AllFGIdToFgConf[fgId].FeatureMeta)
			if err != nil {
				return NewInvalidEventError(fmt.Sprintf("failed to parse feature value for entity %s and feature group %s: %v", persistData.EntityLabel, fgSchema.GetLabel(), err))
			}
			activeVersion, err := p.config.GetActiveVersion(persistData.EntityLabel, fgId)
			if err != nil {
				return fmt.Errorf("failed to get active version for feature group %s: %w", fgSchema.GetLabel(), err)
			}
			psDbBlock := p.BuildPSDBBlock(persistData.EntityLabel, persistData.AllFGIdToFgConf[fgId].DataType, featureData, fgConf, uint32(activeVersion))
			if persistData.StoreIdToRows[fgConf.StoreId] == nil {
				persistData.StoreIdToRows[fgConf.StoreId] = make([]Row, len(persistData.Query.Data))
			}
			if persistData.StoreIdToRows[fgConf.StoreId][rowIndex].FgIdToPsDb == nil {
				persistData.StoreIdToRows[fgConf.StoreId][rowIndex] = Row{
					PkMap: pkMap,
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						fgId: psDbBlock,
					},
				}
			} else {
				persistData.StoreIdToRows[fgConf.StoreId][rowIndex].FgIdToPsDb[fgId] = psDbBlock
			}
		}
	}
	return nil
}

func (p *PersistHandler) PersistToDb(persistData *PersistData) error {
	var wg sync.WaitGroup
	const batchSize = 100

	for storeId, allRows := range persistData.StoreIdToRows {
		// Restructure rows by grouping them based on primary keys
		restructuredRows := p.restructureRowsByPrimaryKeys(allRows)

		store, err := p.dbProvider.GetStore(storeId)
		if err != nil {
			return fmt.Errorf("failed to get store for storeId %s: %w", storeId, err)
		}

		if store.Type() == stores.StoreTypeRedis {
			if err := p.processBatchesForRedis(store, storeId, persistData.EntityLabel, restructuredRows, batchSize); err != nil {
				return err
			}
		} else {
			if err := p.processRowsForScylla(&wg, store, storeId, persistData.EntityLabel, restructuredRows); err != nil {
				return err
			}
		}
		delete(persistData.StoreIdToRows, storeId)
	}
	wg.Wait()
	return nil
}

// restructureRowsByPrimaryKeys groups rows by their primary keys and merges feature groups
// For example, if we have:
// row1: catalog_id:123 -> {fg1->psdb1}
// row2: catalog_id:123 -> {fg2->psdb2}
// It becomes: catalog_id:123 -> {fg1->psdb1, fg2->psdb2}
func (p *PersistHandler) restructureRowsByPrimaryKeys(rows []Row) []Row {
	if len(rows) == 0 {
		return rows
	}

	pkToRow := make(map[string]*Row)

	for _, row := range rows {
		keyStr := p.createPrimaryKeyStr(row.PkMap)
		if existingRow, exists := pkToRow[keyStr]; exists {
			// Merge feature groups from current row into existing row
			for fgId, psdb := range row.FgIdToPsDb {
				existingRow.FgIdToPsDb[fgId] = psdb
			}
			log.Debug().Msgf("Merged feature groups for primary key: %s", keyStr)
		} else {
			// Create a new row with copied primary key map and feature groups
			newRow := Row{
				PkMap:      make(map[string]string, len(row.PkMap)),
				FgIdToPsDb: make(map[int]*blocks.PermStorageDataBlock, len(row.FgIdToPsDb)),
			}

			// Copy primary key map
			for k, v := range row.PkMap {
				newRow.PkMap[k] = v
			}

			// Copy feature groups
			for fgId, psdb := range row.FgIdToPsDb {
				newRow.FgIdToPsDb[fgId] = psdb
			}

			pkToRow[keyStr] = &newRow
			log.Debug().Msgf("Created new row for primary key: %s", keyStr)
		}
	}

	// Convert map back to slice
	restructuredRows := make([]Row, 0, len(pkToRow))
	for _, row := range pkToRow {
		restructuredRows = append(restructuredRows, *row)
	}

	log.Debug().Msgf("Restructuring complete: %d rows -> %d rows", len(rows), len(restructuredRows))
	return restructuredRows
}

func (p *PersistHandler) createPrimaryKeyStr(pkMap map[string]string) string {
	keys := make([]string, 0, len(pkMap))
	for key := range pkMap {
		keys = append(keys, pkMap[key])
	}
	return strings.Join(keys, "|")
}
func (p *PersistHandler) processBatchesForRedis(store stores.Store, storeId, entityLabel string, rows []Row, batchSize int) error {
	var wg sync.WaitGroup
	errChan := make(chan error, (len(rows)+batchSize-1)/batchSize)

	// Process batches concurrently
	for i := 0; i < len(rows); i += batchSize {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("Panic recovered: %v", r)
					metric.Count("online.feature.store.store.panic.count", 1, []string{"storeId:" + storeId, "method:batch_persist"})
					errChan <- fmt.Errorf("panic recovered in batch %d: %v", start, r)
				}
			}()

			end := min(start+batchSize, len(rows))
			batchRows := rows[start:end]

			if err := store.BatchPersistV2(storeId, entityLabel, toModelsRows(batchRows)); err != nil {
				errChan <- fmt.Errorf("batch persist failed for store %s (batch %d-%d): %w",
					storeId, start, end, err)
			}
			cleanupPSDBs(batchRows)
		}(i)
	}

	// Start a goroutine to close errChan when all workers are done
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return fmt.Errorf("persist failed for redis store %s: %w", storeId, err)
		}
	}

	return nil
}

func (p *PersistHandler) processRowsForScylla(wg *sync.WaitGroup, store stores.Store, storeId, entityLabel string, rows []Row) error {
	errChan := make(chan error, len(rows))
	wg.Add(len(rows))

	for _, row := range rows {
		go func(r Row) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("Panic recovered: %v", r)
					metric.Count("online.feature.store.store.panic.count", 1, []string{"storeId:" + storeId, "method:persist"})
					errChan <- fmt.Errorf("panic recovered: %v", r)
				}
			}()

			if err := store.PersistV2(storeId, entityLabel, r.PkMap, r.FgIdToPsDb); err != nil {
				errChan <- err
				return
			}
			cleanupPSDBs([]Row{r})
		}(row)
	}

	// Start a goroutine to close errChan when all workers are done
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return fmt.Errorf("persist failed for scylla store %s: %w", storeId, err)
		}
	}

	return nil
}

func (p *PersistHandler) RemoveFromInMemoryCache(persistData *PersistData) error {
	cache, err := p.imcProvider.GetCache(persistData.EntityLabel)
	if err != nil {
		return err
	}
	for _, data := range persistData.Query.Data {
		keys := data.GetKeyValues()
		err := cache.Delete(persistData.EntityLabel, keys)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to remove from in-memory cache for key: %v", keys)
		}
	}
	return nil
}

func (p *PersistHandler) RemoveFromDistributedCache(persistData *PersistData) error {
	cache, err := p.dcProvider.GetCache(persistData.EntityLabel)
	if err != nil {
		return err
	}
	for _, data := range persistData.Query.Data {
		keys := data.GetKeyValues()
		err := cache.Delete(persistData.EntityLabel, keys)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to remove from distributed cache for key: %v", keys)
		}
	}
	return nil
}

func (p *PersistHandler) BuildPSDBBlock(entityLabel string, dataType types.DataType, featureData interface{}, fgConf *config.FeatureGroup, activeVersion uint32) *blocks.PermStorageDataBlock {
	psDbPool := blocks.GetPSDBPool()
	builder := psDbPool.Get().Builder.
		SetID(uint(fgConf.LayoutVersion)).
		SetDataType(dataType).
		SetCompressionB(compression.TypeZSTD).
		SetTTL(fgConf.TtlInSeconds).
		SetVersion(activeVersion)
	numOfFeatures, err := p.config.GetNumOfFeatures(entityLabel, fgConf.Id, int(activeVersion))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get number of features for feature group %v", fgConf.Id)
	}
	stringLengths, err := p.config.GetStringLengths(entityLabel, fgConf.Id, int(activeVersion))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get string lengths for feature group %v", fgConf.Id)
	}
	vectorLengths, err := p.config.GetVectorLengths(entityLabel, fgConf.Id, int(activeVersion))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get vector lengths for feature group %v", fgConf.Id)
	}
	switch dataType.String() {
	case "DataTypeString":
		psdb, err := builder.
			SetStringValue(stringLengths).
			SetScalarValues(featureData, numOfFeatures).
			Build()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to build PSDB block for feature group %v", fgConf.Id)
		}
		return psdb
	case "DataTypeStringVector":
		psdb, err := builder.
			SetStringValue(stringLengths).
			SetVectorValues(featureData, numOfFeatures, vectorLengths).
			Build()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to build PSDB block for feature group %v", fgConf.Id)
		}
		return psdb
	default:
		if dataType.IsVector() {
			psdb, err := builder.
				SetVectorValues(featureData, numOfFeatures, vectorLengths).
				Build()
			if err != nil {
				log.Error().Err(err).Msgf("Failed to build PSDB block for feature group %v", fgConf.Id)
			}
			return psdb
		}
		psdb, err := builder.
			SetScalarValues(featureData, numOfFeatures).
			Build()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to build PSDB block for feature group %v", fgConf.Id)
		}
		return psdb
	}
}

// TODO: refactor model.go to avoid this
func toModelsRows(rows []Row) []models.Row {
	modelRows := make([]models.Row, len(rows))
	for i, row := range rows {
		modelRows[i] = models.Row{
			PkMap:      row.PkMap,
			FgIdToPsDb: row.FgIdToPsDb,
		}
	}
	return modelRows
}

func cleanupPSDBs(rows []Row) {
	psdbPool := blocks.GetPSDBPool()
	for _, row := range rows {
		for _, psdb := range row.FgIdToPsDb {
			psdb.Clear()
			psdbPool.Put(psdb)
		}
	}
}
