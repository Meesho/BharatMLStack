package feature

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/quantization"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/caches"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/stores"
	handler "github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/circuitbreaker"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/circuitbreaker"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/rs/zerolog/log"
)

var (
	frOnce    sync.Once
	frHandler *RetrieveHandler
)

const (
	featurePartsSeparator = "@"
	distributedCacheCBKey = "distributed_cache_retrieval"
)

type FGData struct {
	// 64-bit aligned fields (pointers and maps)
	data    *RetrieveData
	fgToDDB map[int]*blocks.DeserializedPSDB

	// 32-bit field
	keyIdx int

	// 1-bit field (will be aligned to byte)
	opClose bool
}

type RetrieveHandler struct {
	config                     config.Manager
	imcProvider                provider.CacheProvider
	dcProvider                 provider.CacheProvider
	dbProvider                 provider.StoreProvider
	p2pCacheProvider           provider.CacheProvider
	distributedCacheCBProvider *handler.DFCircuitBreakerHandler
}

func InitRetrieveHandler(configManager config.Manager) *RetrieveHandler {
	distributedCacheCBManager := circuitbreaker.GetManager("distributed_cache")
	if frHandler == nil {
		frOnce.Do(func() {
			frHandler = &RetrieveHandler{
				config:                     configManager,
				imcProvider:                provider.InMemoryCacheProviderImpl,
				dcProvider:                 provider.DistributedCacheProviderImpl,
				dbProvider:                 provider.StorageProviderImpl,
				p2pCacheProvider:           provider.P2PCacheProviderImpl,
				distributedCacheCBProvider: handler.NewDFCircuitBreakerHandler(distributedCacheCBManager),
			}
		})
	}
	return frHandler
}

func getKeyString(key *retrieve.Keys) string {
	return strings.Join(key.Cols, "|")
}

func (h *RetrieveHandler) RetrieveFeatures(ctx context.Context, query *retrieve.Query) (*retrieve.Result, error) {
	log.Debug().Msgf("Retrieving features for query: %v", query)
	retrieveData := &RetrieveData{
		Query: query,
	}
	err := preProcessRequest(retrieveData, h.config)
	if err != nil {
		return nil, err
	}
	fgDataChan := make(chan *FGData)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for fgData := range fgDataChan {
			if fgData.opClose {
				break
			}
			h.fillMatrix(fgData.data, fgData.fgToDDB, fgData.keyIdx)
		}
	}()
	ReqInMemEmpty := retrieveData.ReqInMemCachedFGIds.IsEmpty()
	ReqDistEmpty := retrieveData.ReqDistCachedFGIds.IsEmpty()
	allKeys := retrieveData.UniqueKeys
	reqDistCachedFGIds := retrieveData.ReqDistCachedFGIds
	reqDbFGIds := retrieveData.ReqDbFGIds
	allDistFGIds := retrieveData.AllDistCachedFGIds

	isP2PEnabled := h.isP2PCacheEnabled(retrieveData.EntityLabel)

	if ReqInMemEmpty && ReqDistEmpty {
		_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds, fgDataChan)
		h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
		if err != nil {
			return nil, err
		}
		return retrieveData.Result, nil
	} else if ReqInMemEmpty && !ReqDistEmpty {
		missingDistKeys, err := h.retrieveFromDistributedCache(allKeys, retrieveData, reqDistCachedFGIds, fgDataChan)
		if err != nil {
			h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
			return nil, err
		}
		missingKeyExists := len(missingDistKeys) > 0
		reqDbFGIdsExists := !reqDbFGIds.IsEmpty()

		if !missingKeyExists && !reqDbFGIdsExists {
			// No Missing keys and no FGIds to fetch from DB
			// do nothing
		} else if !missingKeyExists && reqDbFGIdsExists {
			// No Missing keys and FGIds to fetch from DB
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds, fgDataChan)
		} else if missingKeyExists && !reqDbFGIdsExists {
			// Missing keys and no FGIds to fetch from DB
			_, err = h.retrieveFromDB(missingDistKeys, retrieveData, allDistFGIds, fgDataChan)
		} else if missingKeyExists && reqDbFGIdsExists {
			// Missing keys and FGIds to fetch from DB
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds.Union(allDistFGIds), fgDataChan)
		} else {
			log.Error().Msg("Invalid state missingKeyExists reqRemainingFGIdsExists")
			err = errors.New("invalid state missingKeyExists reqRemainingFGIdsExists")
		}
		h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
		if err != nil {
			return nil, err
		}
		if len(missingDistKeys) > 0 {
			go h.persistToDistributedCache(retrieveData.EntityLabel, retrieveData, allDistFGIds, missingDistKeys)
		}
		return retrieveData.Result, nil
	} else if !ReqInMemEmpty && ReqDistEmpty {
		missingInMemKeys, err := h.retrieveFromInMemoryCache(allKeys, retrieveData, retrieveData.ReqInMemCachedFGIds, fgDataChan, isP2PEnabled)
		if err != nil {
			h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
			return nil, err
		}
		missingKeyExists := len(missingInMemKeys) > 0
		reqDbFGIdsExists := !reqDbFGIds.IsEmpty()
		if !missingKeyExists && !reqDbFGIdsExists {
			// No Missing keys and no FGIds to fetch from DB
			// do nothing
		} else if !missingKeyExists && reqDbFGIdsExists {
			// No Missing keys and FGIds to fetch from DB
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds, fgDataChan)
		} else if missingKeyExists && !reqDbFGIdsExists {
			// Missing keys and no FGIds to fetch from DB, only fetch ReqInMemCachedFGIds
			_, err = h.retrieveFromDB(missingInMemKeys, retrieveData, retrieveData.ReqInMemCachedFGIds, fgDataChan)
		} else if missingKeyExists && reqDbFGIdsExists {
			// Missing keys and FGIds to fetch from DB, fetch ReqInMemCachedFGIds and ReqDbFGIds
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds.Union(retrieveData.ReqInMemCachedFGIds), fgDataChan)
		} else {
			log.Error().Msg("Invalid state missingKeyExists reqRemainingFGIdsExists")
			err = errors.New("invalid state missingKeyExists reqRemainingFGIdsExists")
		}
		h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
		if err != nil {
			return nil, err
		}
		if len(missingInMemKeys) > 0 {
			go h.persistToInMemoryCache(retrieveData.EntityLabel, retrieveData, retrieveData.ReqInMemCachedFGIds, missingInMemKeys, isP2PEnabled)
		}
		return retrieveData.Result, nil
	} else if !ReqInMemEmpty && !ReqDistEmpty {
		reqDbFGIdsExists := !reqDbFGIds.IsEmpty()
		missingInMemKeys, err := h.retrieveFromInMemoryCache(allKeys, retrieveData, retrieveData.ReqInMemCachedFGIds, fgDataChan, isP2PEnabled)
		if err != nil {
			h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
			return nil, err
		}
		missingInMemKeyExists := len(missingInMemKeys) > 0
		exclusiveDistFGIds := reqDistCachedFGIds.Difference(retrieveData.ReqInMemCachedFGIds)
		exclusiveInMemFGIds := retrieveData.ReqInMemCachedFGIds.Difference(reqDistCachedFGIds)
		exclusiveInMemExists := !exclusiveInMemFGIds.IsEmpty() && missingInMemKeyExists
		var missingDistKeys []*retrieve.Keys
		if !exclusiveDistFGIds.IsEmpty() {
			// some fgIds only present in distributed cache
			missingDistKeys, err = h.retrieveFromDistributedCache(allKeys, retrieveData, reqDistCachedFGIds, fgDataChan)
		} else if exclusiveDistFGIds.IsEmpty() && missingInMemKeyExists {
			// some fgIds only present in distributed cache and some keys are missing in in-memory cache
			missingDistKeys, err = h.retrieveFromDistributedCache(missingInMemKeys, retrieveData, reqDistCachedFGIds, fgDataChan)
		} else if exclusiveDistFGIds.IsEmpty() && !missingInMemKeyExists {
			// no fgIds only present in distributed cache and no keys are missing in in-memory cache, meaning we are done
			// do nothing
		} else {
			log.Error().Msg("Invalid state exclusiveDistFGIds missingInMemKeyExists")
			err = errors.New("invalid state exclusiveDistFGIds missingInMemKeyExists")
		}
		if err != nil {
			h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
			return nil, err
		}
		missingDistKeyExists := len(missingDistKeys) > 0
		if !missingDistKeyExists && !reqDbFGIdsExists && !exclusiveInMemExists {
			// do nothing
		} else if !missingDistKeyExists && !reqDbFGIdsExists && exclusiveInMemExists {
			// No missing dist keys, no DB FGs to fetch, but has exclusive in-mem FGs
			_, err = h.retrieveFromDB(missingInMemKeys, retrieveData, exclusiveInMemFGIds, fgDataChan)
		} else if !missingDistKeyExists && reqDbFGIdsExists && !exclusiveInMemExists {
			// No missing dist keys, has DB FGs to fetch, no exclusive in-mem FGs
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds, fgDataChan)
		} else if !missingDistKeyExists && reqDbFGIdsExists && exclusiveInMemExists {
			// No missing dist keys, has DB FGs to fetch, has exclusive in-mem FGs
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds.Union(exclusiveInMemFGIds), fgDataChan)
		} else if missingDistKeyExists && !reqDbFGIdsExists && !exclusiveInMemExists {
			// Has missing dist keys, no DB FGs to fetch, no exclusive in-mem FGs
			_, err = h.retrieveFromDB(missingDistKeys, retrieveData, allDistFGIds, fgDataChan)
		} else if missingDistKeyExists && !reqDbFGIdsExists && exclusiveInMemExists {
			// Has missing dist keys, no DB FGs to fetch, has exclusive in-mem FGs
			keysToFetch := append(missingInMemKeys, missingDistKeys...) // TODO: can do betterm this will add duplicates
			_, err = h.retrieveFromDB(keysToFetch, retrieveData, exclusiveInMemFGIds.Union(allDistFGIds), fgDataChan)
		} else if missingDistKeyExists && reqDbFGIdsExists && !exclusiveInMemExists {
			// Has missing dist keys, has DB FGs to fetch, no exclusive in-mem FGs
			_, err = h.retrieveFromDB(allKeys, retrieveData, reqDbFGIds.Union(allDistFGIds), fgDataChan)
		} else if missingDistKeyExists && reqDbFGIdsExists && exclusiveInMemExists {
			// Has missing dist keys, has DB FGs to fetch, has exclusive in-mem FGs
			_, err = h.retrieveFromDB(allKeys, retrieveData, exclusiveInMemFGIds.Union(allDistFGIds).Union(reqDbFGIds), fgDataChan)
		} else {
			log.Error().Msg("Invalid state missingDistKeyExists reqRemainingFGIdsExists exclusiveInMemExists")
			err = errors.New("invalid state missingDistKeyExists reqRemainingFGIdsExists exclusiveInMemExists")
		}
		h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
		if err != nil {
			return nil, err
		}
		if len(missingInMemKeys) > 0 {
			go h.persistToInMemoryCache(retrieveData.EntityLabel, retrieveData, retrieveData.ReqInMemCachedFGIds, missingInMemKeys, isP2PEnabled)
		}
		if len(missingDistKeys) > 0 {
			go h.persistToDistributedCache(retrieveData.EntityLabel, retrieveData, allDistFGIds, missingDistKeys)
		}
		return retrieveData.Result, nil
	} else {
		log.Error().Msg("Invalid state ReqInMemEmpty ReqDistEmpty")
		h.closeFeatureDataChannel(fgDataChan, retrieveData, &wg)
		return nil, errors.New("invalid state ReqInMemEmpty ReqDistEmpty")
	}
}

func (h *RetrieveHandler) isP2PCacheEnabled(entityLabel string) bool {
	if h.p2pCacheProvider == nil {
		return false
	}
	// If cache is setup, then enable based on percentage
	return rand.Intn(100) < h.config.GetP2PEnabledPercentage()
}

func (h *RetrieveHandler) retrieveFromInMemoryCache(keys []*retrieve.Keys, retrieveData *RetrieveData, fgIds ds.Set[int], fgDataChan chan *FGData, isP2PEnabled bool) ([]*retrieve.Keys, error) {
	// TODO: Remove once fully scaled
	if isP2PEnabled {
		return h.retrieveFromP2PCache(keys, retrieveData, fgIds, fgDataChan)
	}
	entityLabel := retrieveData.EntityLabel
	cache, err := h.imcProvider.GetCache(entityLabel)
	if err != nil {
		return nil, err
	}
	missingDataKeys := make([]*retrieve.Keys, 0)
	for _, key := range keys {
		keyIdx := retrieveData.ReqKeyToIdx[getKeyString(key)]
		serData := cache.GetV2(entityLabel, key)
		if len(serData) == 0 {
			metric.Count("feature.retrieve.cache.miss", 1, []string{"entity_name", entityLabel, "cache_type", "in_memory"})
			missingDataKeys = append(missingDataKeys, key)
			continue
		}
		csdb, _ := blocks.CreateCSDBForInMemory(serData)
		fgIdToDDB, _ := csdb.GetDeserializedPSDBForFGIds(retrieveData.ReqInMemCachedFGIds)
		if fgIdToDDB != nil {
			metric.Count("feature.retrieve.cache.hit", 1, []string{"entity_name", entityLabel, "cache_type", "in_memory"})
			fgDataChan <- &FGData{
				data:    retrieveData,
				fgToDDB: fgIdToDDB,
				keyIdx:  keyIdx,
				opClose: false,
			}
		} else {
			metric.Count("feature.retrieve.cache.miss", 1, []string{"entity_name", entityLabel, "cache_type", "in_memory"})
			missingDataKeys = append(missingDataKeys, key)
		}
	}
	return missingDataKeys, nil
}

func (h *RetrieveHandler) retrieveFromP2PCache(keys []*retrieve.Keys, retrieveData *RetrieveData, fgIds ds.Set[int], fgDataChan chan *FGData) ([]*retrieve.Keys, error) {
	entityLabel := retrieveData.EntityLabel
	metric.Count("feature.retrieve.cache.requests.total", 1, []string{"entity_name", entityLabel, "cache_type", "p2p"})

	startTime := time.Now()
	log.Debug().Msgf("Retrieving features from P2P cache for keys %v", keys)

	cache, err := h.p2pCacheProvider.GetCache(entityLabel)
	if err != nil {
		return nil, err
	}
	metric.Count("feature.retrieve.cache.total", int64(len(keys)), []string{"entity_name", entityLabel, "cache_type", "p2p"})

	cacheData, err := cache.MultiGetV2(entityLabel, keys)
	log.Debug().Msgf("Retrieved features from P2P cache for keys %v, cacheData: %v", keys, cacheData)
	if err != nil {
		log.Error().Err(err).Msgf("Error while retrieving features from P2P cache for keys %v", keys)
		return nil, err
	}
	missingDataKeys := make([]*retrieve.Keys, 0)
	for i, key := range keys {
		keyIdx := retrieveData.ReqKeyToIdx[getKeyString(key)]
		serData := cacheData[i]
		if len(serData) == 0 {
			missingDataKeys = append(missingDataKeys, key)
			continue
		}
		csdb, _ := blocks.CreateCSDBForInMemory(serData)
		fgIdToDDB, _ := csdb.GetDeserializedPSDBForFGIds(retrieveData.ReqInMemCachedFGIds)
		if fgIdToDDB != nil {
			fgDataChan <- &FGData{
				data:    retrieveData,
				fgToDDB: fgIdToDDB,
				keyIdx:  keyIdx,
				opClose: false,
			}
		} else {
			missingDataKeys = append(missingDataKeys, key)
		}
	}
	metric.Count("feature.retrieve.cache.hit", int64(len(keys)-len(missingDataKeys)), []string{"entity_name", entityLabel, "cache_type", "p2p"})
	metric.Count("feature.retrieve.cache.miss", int64(len(missingDataKeys)), []string{"entity_name", entityLabel, "cache_type", "p2p"})
	log.Debug().Msgf("Missing data keys from P2P cache: %v", missingDataKeys)
	metric.Timing("feature.retrieve.cache.latency", time.Since(startTime), []string{"entity_name", entityLabel, "cache_type", "p2p"})
	return missingDataKeys, nil
}

func (h *RetrieveHandler) retrieveFromDistributedCache(keys []*retrieve.Keys, retrieveData *RetrieveData, fgIds ds.Set[int], fgDataChan chan *FGData) ([]*retrieve.Keys, error) {
	log.Debug().Msgf("Retrieving features from distributed cache for keys %v and fgIds %v", keys, fgIds)
	entityLabel := retrieveData.EntityLabel
	cache, err := h.dcProvider.GetCache(entityLabel)
	if err != nil {
		return nil, err
	}
	var cacheData [][]byte
	if h.distributedCacheCBProvider.IsCBEnabled(distributedCacheCBKey) && !h.distributedCacheCBProvider.IsCallAllowed(distributedCacheCBKey) {
		log.Debug().Msgf("Circuit breaker is open, returning negative cache for all fgIds")
		metric.Gauge("feature.retrieve.cb.open", 1, []string{"entity_name", entityLabel, "cb_key", distributedCacheCBKey})
		for _, key := range keys {
			keyIdx := retrieveData.ReqKeyToIdx[getKeyString(key)]
			fgIdToDDB := make(map[int]*blocks.DeserializedPSDB)
			fgIds.KeyIterator(func(fgId int) bool {
				fgIdToDDB[fgId] = blocks.NegativeCacheDeserializePSDB()
				return true
			})
			fgDataChan <- &FGData{
				data:    retrieveData,
				fgToDDB: fgIdToDDB,
				keyIdx:  keyIdx,
				opClose: false,
			}
		}
		return nil, nil
	}
	metric.Gauge("feature.retrieve.cb.open", 0, []string{"entity_name", entityLabel, "cb_key", distributedCacheCBKey})
	cacheData, err = cache.MultiGetV2(entityLabel, keys)
	log.Debug().Msgf("Retrieved features from distributed cache for keys %v and fgIds %v, cacheData: %v", keys, fgIds, cacheData)
	if err != nil {
		if h.distributedCacheCBProvider.IsCBEnabled(distributedCacheCBKey) {
			h.distributedCacheCBProvider.RecordFailure(distributedCacheCBKey)
		}
		log.Error().Err(err).Msgf("Error while retrieving features from distributed cache for keys %v and fgIds %v", keys, fgIds)
		return nil, err
	}
	if h.distributedCacheCBProvider.IsCBEnabled(distributedCacheCBKey) {
		h.distributedCacheCBProvider.RecordSuccess(distributedCacheCBKey)
	}
	missingDataKeys := make([]*retrieve.Keys, 0)
	for i, key := range keys {
		keyIdx := retrieveData.ReqKeyToIdx[getKeyString(key)]
		serData := cacheData[i]
		// todo: combine else if and else block
		if len(serData) == 0 {
			missingDataKeys = append(missingDataKeys, key)
			continue
		}
		csdb, _ := blocks.CreateCSDBForDistributedCache(serData)
		fgIdToDDB, _ := csdb.GetDeserializedPSDBForFGIds(fgIds)
		if fgIdToDDB != nil {
			fgDataChan <- &FGData{
				data:    retrieveData,
				fgToDDB: fgIdToDDB,
				keyIdx:  keyIdx,
				opClose: false,
			}
		} else {
			missingDataKeys = append(missingDataKeys, key)
		}
	}
	log.Debug().Msgf("Missing data keys from distributed cache: %v", missingDataKeys)
	return missingDataKeys, nil
}

func (h *RetrieveHandler) retrieveFromDB(keys []*retrieve.Keys, retrieveData *RetrieveData, fgIds ds.Set[int], fgDataChan chan *FGData) ([]*retrieve.Keys, error) {
	log.Debug().Msgf("Retrieving features from DB for keys %v and fgIds %v", keys, fgIds)
	missingDataKeys := make([]*retrieve.Keys, 0)
	keySchema := retrieveData.Query.KeysSchema
	allFGIdToStoreId := retrieveData.AllFGIdToStoreId
	entityLabel := retrieveData.EntityLabel

	storeIdToFGIds := make(map[string][]int)
	fgIds.KeyIterator(func(fgId int) bool {
		storeId := allFGIdToStoreId[fgId]
		if _, ok := storeIdToFGIds[storeId]; !ok {
			storeIdToFGIds[storeId] = make([]int, 0)
		}
		storeIdToFGIds[storeId] = append(storeIdToFGIds[storeId], fgId)
		return true
	})

	pkMaps := make([]map[string]string, len(keys))
	for i, ids := range keys {
		pkMap := make(map[string]string)
		for j, key := range keySchema {
			pkMap[key] = ids.Cols[j]
		}
		pkMaps[i] = pkMap
	}

	var wg sync.WaitGroup

	for storeId, storeFgIds := range storeIdToFGIds {
		log.Debug().Msgf("Retrieving features from DB for store: %s and fgIds: %v", storeId, storeFgIds)
		store, err := h.dbProvider.GetStore(storeId)
		if err != nil {
			return nil, err
		}

		if store.Type() == stores.StoreTypeRedis {
			// Use BatchRetrieveV2 for Redis
			results, err := store.BatchRetrieveV2(entityLabel, pkMaps, storeFgIds)
			if err != nil {
				return nil, err
			}

			for i, result := range results {
				if len(result) == 0 {
					missingDataKeys = append(missingDataKeys, keys[i])
					continue
				}
				keyIdx := retrieveData.ReqKeyToIdx[getKeyString(keys[i])]
				fgDataChan <- &FGData{
					data:    retrieveData,
					fgToDDB: result,
					keyIdx:  keyIdx,
					opClose: false,
				}
			}
		} else {
			wg.Add(len(keys))
			for i, pkMap := range pkMaps {
				keyIdx := retrieveData.ReqKeyToIdx[getKeyString(keys[i])]
				go func(pkMap map[string]string, keys *retrieve.Keys, keyIdx int) {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Msgf("Recovered from panic: %v", r)
						}
						wg.Done()
					}()
					fgIdToDDB, err := store.RetrieveV2(entityLabel, pkMap, storeFgIds)
					if err != nil {
						return
					}
					if fgIdToDDB == nil {
						missingDataKeys = append(missingDataKeys, keys)
					} else {
						fgDataChan <- &FGData{
							data:    retrieveData,
							fgToDDB: fgIdToDDB,
							keyIdx:  keyIdx,
							opClose: false,
						}
					}
				}(pkMap, keys[i], keyIdx)
			}
		}
	}
	wg.Wait()
	return missingDataKeys, nil
}

func preProcessRequest(retrieveData *RetrieveData, configManager config.Manager) error {
	query := retrieveData.Query
	retrieveData.Result = &retrieve.Result{
		KeysSchema: query.KeysSchema,
	}
	retrieveData.EntityLabel = query.EntityLabel
	err := preProcessFGs(retrieveData, configManager)
	if err != nil {
		return err
	}
	err = preProcessEntity(retrieveData, configManager)
	if err != nil {
		return err
	}
	err = preProcessForKeys(retrieveData, configManager)
	if err != nil {
		return err
	}
	return nil

}

func preProcessEntity(retrieveData *RetrieveData, configManager config.Manager) error {
	entityLabel := retrieveData.EntityLabel

	// Get entity and validate
	entity, err := configManager.GetEntity(entityLabel)
	if err != nil {
		return fmt.Errorf("invalid entity %s: %w", entityLabel, err)
	}

	// Pre-allocate maps based on entity's feature groups
	numEntityFGs := len(entity.FeatureGroups)
	retrieveData.AllFGIdToDataType = make(map[int]types.DataType, numEntityFGs)
	retrieveData.AllFGIdToStoreId = make(map[int]string, numEntityFGs)
	retrieveData.AllFGLabelToFGId = make(map[string]int, numEntityFGs)
	retrieveData.AllFGIdToFGLabel = make(map[int]string, numEntityFGs)

	// Initialize sets with capacity
	retrieveData.ReqInMemCachedFGIds = ds.NewOrderedSetWithCapacity[int](numEntityFGs)
	retrieveData.ReqDistCachedFGIds = ds.NewOrderedSetWithCapacity[int](numEntityFGs)
	retrieveData.ReqDbFGIds = ds.NewOrderedSetWithCapacity[int](numEntityFGs)
	retrieveData.AllDistCachedFGIds = ds.NewOrderedSetWithCapacity[int](numEntityFGs)

	// Process all feature groups from entity
	for fgLabel, fg := range entity.FeatureGroups {
		// Store all mappings regardless of request
		retrieveData.AllFGLabelToFGId[fgLabel] = fg.Id // TODO: immutable map creation?
		retrieveData.AllFGIdToFGLabel[fg.Id] = fgLabel // TODO: immutable map creation?
		retrieveData.AllFGIdToStoreId[fg.Id] = fg.StoreId

		// Parse and store data type
		dataType, err := types.ParseDataType(fg.DataType.String())
		if err != nil {
			return fmt.Errorf("invalid data type for feature group %s: %w", fgLabel, err)
		}
		retrieveData.AllFGIdToDataType[fg.Id] = dataType

		// Track distributed cache capability for all FGs
		if fg.DistributedCacheEnabled && entity.DistributedCache.Enabled {
			retrieveData.AllDistCachedFGIds.Add(fg.Id)
		}

		// Only add to request sets if this FG is in the request
		if retrieveData.ReqFGIds.Has(fg.Id) {
			isRequestDbFgId := true
			if fg.InMemoryCacheEnabled && entity.InMemoryCache.Enabled {
				retrieveData.ReqInMemCachedFGIds.Add(fg.Id)
				isRequestDbFgId = false
			}
			if fg.DistributedCacheEnabled && entity.DistributedCache.Enabled {
				retrieveData.ReqDistCachedFGIds.Add(fg.Id)
				isRequestDbFgId = false
			}
			if isRequestDbFgId {
				retrieveData.ReqDbFGIds.Add(fg.Id)
			}
		}
	}

	return nil
}

func preProcessForKeys(retrieveData *RetrieveData, configManager config.Manager) error {
	query := retrieveData.Query
	rows := make([]*retrieve.Row, len(query.Keys))
	reqKeyToIdx := make(map[string]int, len(query.Keys))
	uniquekeys := make([]*retrieve.Keys, 0)
	keyToOriginalIndices := make(map[string][]int)
	// Create a map of feature group properties for quick lookup
	fgProps := make(map[string]struct {
		id       int
		dataType types.DataType
	})

	// Initialize feature group properties
	for _, fg := range retrieveData.Result.FeatureSchemas {
		fgId := retrieveData.AllFGLabelToFGId[fg.FeatureGroupLabel]
		dataType := retrieveData.AllFGIdToDataType[fgId]

		// TODO: Have to check this usage?
		// Get feature group for vector length check
		// fgConfig, err := configManager.GetFeatureGroup(retrieveData.EntityLabel, fg.FeatureGroupLabel)
		// if err != nil {
		// 	return fmt.Errorf("failed to get active schema: %w", err)
		// }

		fgProps[fg.FeatureGroupLabel] = struct {
			id       int
			dataType types.DataType
		}{
			id:       fgId,
			dataType: dataType,
		}
	}

	// Process each key
	for i, key := range query.Keys {
		keyStr := getKeyString(key)

		if _, exists := keyToOriginalIndices[keyStr]; !exists {
			uniquekeys = append(uniquekeys, key)
			reqKeyToIdx[keyStr] = i
			keyToOriginalIndices[keyStr] = make([]int, 0)
		}

		keyToOriginalIndices[keyStr] = append(keyToOriginalIndices[keyStr], i)
		// Create row with pre-allocated columns
		dt := make([][]byte, retrieveData.ReqColumnCount)
		rows[i] = &retrieve.Row{
			Keys:    key.Cols,
			Columns: dt,
		}

		// Process each feature group for this key
		for _, fg := range retrieveData.Result.FeatureSchemas {
			fgProp := fgProps[fg.FeatureGroupLabel]

			// Skip byte allocation for string types
			if fgProp.dataType == types.DataTypeString || fgProp.dataType == types.DataTypeStringVector {
				continue
			}

			// Get active schema version
			fgConfig, err := configManager.GetFeatureGroup(retrieveData.EntityLabel, fg.FeatureGroupLabel)
			if err != nil {
				return fmt.Errorf("failed to get active schema: %w", err)
			}
			version, err := strconv.Atoi(fgConfig.ActiveVersion)
			if err != nil {
				return fmt.Errorf("invalid version number: %w", err)
			}

			// Get default values for each feature
			for _, f := range fg.Features {
				defaultValue, err := configManager.GetDefaultValueByte(retrieveData.EntityLabel, fgProp.id, version, f.Label)
				if err != nil {
					return fmt.Errorf("failed to get default value for feature %s: %w", f.Label, err)
				}
				rows[i].Columns[f.ColumnIdx] = defaultValue
			}
		}
	}
	retrieveData.UniqueKeys = uniquekeys
	retrieveData.KeyToOriginalIndices = keyToOriginalIndices
	retrieveData.ReqKeyToIdx = reqKeyToIdx
	retrieveData.Result.Rows = rows
	return nil
}

func preProcessFGs(retrieveData *RetrieveData, configManager config.Manager) error {
	query := retrieveData.Query
	entityLabel := retrieveData.EntityLabel

	// Validate entity exists first
	_, err := configManager.GetEntity(entityLabel)
	if err != nil {
		return fmt.Errorf("invalid entity %s: %w", entityLabel, err)
	}
	retrieveData.ReqIdxToFgIdToDdb = make(map[int]map[int]*blocks.DeserializedPSDB)
	numFGs := len(query.FeatureGroups)
	featureSchema := make([]*retrieve.FeatureSchema, numFGs)
	fgIdToFeatureLabels := make(map[int]ds.Set[string], numFGs)
	fgIdToFeatureLabelWithQuantType := make(map[int]map[string]types.DataType, numFGs)

	// Initialize reqFGIds with capacity
	reqFGIds := ds.NewOrderedSetWithCapacity[int](numFGs)

	var columnIdx int32 = 0

	for i, fg := range query.FeatureGroups {
		// Get active schema first to validate features
		activeSchema, err := configManager.GetActiveFeatureSchema(entityLabel, fg.Label)
		if err != nil {
			return fmt.Errorf("failed to get active schema for feature group %s: %w", fg.Label, err)
		}

		// Get feature group properties
		fgProp, err := configManager.GetFeatureGroup(entityLabel, fg.Label)
		if err != nil {
			return fmt.Errorf("failed to get feature group %s: %w", fg.Label, err)
		}

		// Get feature group data type
		dataType, err := types.ParseDataType(fgProp.DataType.String())
		if err != nil {
			return fmt.Errorf("invalid data type for feature group %s: %w", fg.Label, err)
		}

		// Create feature set with known capacity
		featureSet := ds.NewOrderedSetWithCapacity[string](len(fg.FeatureLabels))
		featureLabelWithQuantType := make(map[string]types.DataType, len(fg.FeatureLabels))
		features := make([]*retrieve.Feature, 0, len(fg.FeatureLabels))

		// Validate and process features
		for _, featureLabel := range fg.FeatureLabels {
			// Parse feature label to handle quantization
			baseFeatureLabel, quantType, err := ParseFeatureLabel(featureLabel, dataType)
			if err != nil {
				return fmt.Errorf("failed to parse feature label %s: %w", featureLabel, err)
			}
			if _, ok := activeSchema.FeatureMeta[baseFeatureLabel]; !ok {
				return fmt.Errorf("feature %s not found in active schema of feature group %s",
					baseFeatureLabel, fg.Label)
			}

			// If quantization is requested, validate it's supported
			if quantType != types.DataTypeUnknown {
				if !quantization.IsQuantizationCompatible(dataType, quantType) {
					return fmt.Errorf("unsupported quantization from %v to %v for feature %s in feature group %s",
						dataType, quantType, baseFeatureLabel, fg.Label)
				}
				featureLabelWithQuantType[baseFeatureLabel] = quantType
			}

			// Add base feature label with metadata (columnIdx)
			featureSet.AddWithMeta(baseFeatureLabel, int(columnIdx))

			features = append(features, &retrieve.Feature{
				Label:     baseFeatureLabel, // Use base feature label for schema
				ColumnIdx: columnIdx,
			})
			columnIdx++
		}

		featureSchema[i] = &retrieve.FeatureSchema{
			FeatureGroupLabel: fg.Label,
			Features:          features,
		}
		// Store feature set in map and add FG ID to required set
		fgIdToFeatureLabels[fgProp.Id] = featureSet
		fgIdToFeatureLabelWithQuantType[fgProp.Id] = featureLabelWithQuantType
		reqFGIds.Add(fgProp.Id)
	}

	// Update retrieveData all at once
	retrieveData.Result.FeatureSchemas = featureSchema
	retrieveData.Result.EntityLabel = entityLabel
	retrieveData.ReqFGIdToFeatureLabels = fgIdToFeatureLabels
	retrieveData.ReqFGIdToFeatureLabelWithQuantType = fgIdToFeatureLabelWithQuantType
	retrieveData.ReqColumnCount = int(columnIdx)
	retrieveData.ReqFGIds = reqFGIds

	return nil
}

func (h *RetrieveHandler) fillMatrix(data *RetrieveData, fgToDDB map[int]*blocks.DeserializedPSDB, keyIdx int) {
	keyStr := getKeyString(data.Query.Keys[keyIdx])

	for fgId, ddb := range fgToDDB {
		if _, ok := data.ReqIdxToFgIdToDdb[keyIdx]; !ok {
			data.ReqIdxToFgIdToDdb[keyIdx] = make(map[int]*blocks.DeserializedPSDB)
		}
		data.ReqIdxToFgIdToDdb[keyIdx][fgId] = ddb
		if _, exists := data.ReqFGIdToFeatureLabels[fgId]; !exists {
			continue
		}
		if ddb.Expired {
			metric.Count("online.feature.store.retrieve.validity", 1, []string{"feature_group", data.AllFGIdToFGLabel[fgId], "entity", data.EntityLabel, "validity", "expired"})
		} else if ddb.NegativeCache {
			metric.Count("online.feature.store.retrieve.validity", 1, []string{"feature_group", data.AllFGIdToFGLabel[fgId], "entity", data.EntityLabel, "validity", "negative_cache"})
		} else {
			metric.Count("online.feature.store.retrieve.validity", 1, []string{"feature_group", data.AllFGIdToFGLabel[fgId], "entity", data.EntityLabel, "validity", "valid"})
		}
		data.ReqFGIdToFeatureLabels[fgId].FastIterator(func(featureLabel string, meta interface{}) {
			featureLabelWithQuantType := data.ReqFGIdToFeatureLabelWithQuantType[fgId]
			var version int
			var err error

			if ddb.NegativeCache || ddb.Expired {
				version, err = h.config.GetActiveVersion(data.EntityLabel, fgId)
				if err != nil {
					log.Error().Err(err).Msgf("Error while getting active version for feature %s", featureLabel)
					return
				}
			} else {
				version = int(ddb.FeatureSchemaVersion)
			}
			seq, err := h.config.GetSequenceNo(data.EntityLabel, fgId, int(version), featureLabel)
			if err != nil {
				log.Error().Err(err).Msgf("Error while getting sequence no for feature %s", featureLabel)
				return
			}
			// Handle missing features (seq = -1)
			if seq == -1 {
				log.Warn().Msgf("Feature %s not found in version %d, switching to active version for default value", featureLabel, version)

				// Get active version
				version, err = h.config.GetActiveVersion(data.EntityLabel, fgId)
				if err != nil {
					log.Error().Err(err).Msgf("Error while getting active version for missing feature %s", featureLabel)
					return
				}
				// Try to get sequence from active version
				activeSeq, err := h.config.GetSequenceNo(data.EntityLabel, fgId, version, featureLabel)
				if err != nil {
					log.Error().Err(err).Msgf("Error while getting sequence no from active version for feature %s", featureLabel)
					return
				}
				// if feature is not present in active version just return
				if activeSeq == -1 {
					log.Warn().Msgf("Feature %s is not available in active version", featureLabel)
					return
				}
				// Feature exists in active version
				seq = activeSeq
				ddb.NegativeCache = true
				metric.Count("online.feature.store.retrieve.validity", 1, []string{"feature_group", data.AllFGIdToFGLabel[fgId], "entity", data.EntityLabel})
			}

			stringLengths, err := h.config.GetStringLengths(data.EntityLabel, fgId, int(version))
			if err != nil {
				log.Error().Err(err).Msgf("Error while getting string lengths for feature %s", featureLabel)
				return
			}
			vectorLengths, err := h.config.GetVectorLengths(data.EntityLabel, fgId, int(version))
			if err != nil {
				log.Error().Err(err).Msgf("Error while getting vector lengths for feature %s", featureLabel)
				return
			}
			numOfFeatures, err := h.config.GetNumOfFeatures(data.EntityLabel, fgId, int(version))
			if err != nil {
				log.Error().Err(err).Msgf("Error while getting number of features for feature %s", featureLabel)
				return
			}

			var fdata []byte
			if ddb.NegativeCache || ddb.Expired {
				fdata, err = h.config.GetDefaultValueByte(data.EntityLabel, fgId, int(version), featureLabel)
				if err != nil {
					log.Error().Err(err).Msgf("Error while getting default value for feature %s", featureLabel)
					return
				}
			} else {
				// Get feature in original datatype
				fdata, err = GetFeature(ddb.DataType, ddb, seq, numOfFeatures, stringLengths, vectorLengths)
				if err != nil {
					log.Error().Err(err).Msgf("Error while getting feature for sequence no %d from ddb [feature: %s]", seq, featureLabel)
					return
				}
			}
			// Apply quantization if requested
			if quantType, ok := featureLabelWithQuantType[featureLabel]; ok {
				quantFunc, err := quantization.GetQuantizationFunction(ddb.DataType, quantType)
				if err != nil {
					log.Error().Err(err).Msgf("Error getting quantization function from %v to %v", ddb.DataType, quantType)
					return
				}
				fdata = quantFunc(fdata)
			}

			colIdx := meta.(int)
			for _, idx := range data.KeyToOriginalIndices[keyStr] {
				data.Result.Rows[idx].Columns[colIdx] = fdata
			}
		})
	}
}

func (h *RetrieveHandler) closeFeatureDataChannel(fgDataChan chan *FGData, retrieveData *RetrieveData, wg *sync.WaitGroup) {
	fgDataChan <- &FGData{
		data:    retrieveData,
		fgToDDB: nil,
		keyIdx:  -1,
		opClose: true,
	}
	wg.Wait()
	close(fgDataChan)
}

func (h *RetrieveHandler) persistToCache(entityLabel string, retrieveData *RetrieveData, fgIds ds.Set[int], missingKeys []*retrieve.Keys, isDistributed bool, isP2PEnabled bool) {
	var cache caches.Cache
	var err error

	if isP2PEnabled {
		cache, err = h.p2pCacheProvider.GetCache(entityLabel)
	} else if isDistributed {
		cache, err = h.dcProvider.GetCache(entityLabel)
	} else {
		cache, err = h.imcProvider.GetCache(entityLabel)
	}

	if err != nil {
		log.Error().Err(err).Msgf("Error while getting cache provider for entity %s", entityLabel)
		return
	}

	var cacheKeys []*retrieve.Keys
	var cacheData [][]byte
	// Process each missing key
	for _, key := range missingKeys {
		keyStr := getKeyString(key)
		keyIdx := retrieveData.ReqKeyToIdx[keyStr]
		fgIdToDdb := retrieveData.ReqIdxToFgIdToDdb[keyIdx]

		// Create a new CSDB for this key
		csdb := blocks.NewCacheStorageDataBlock(blocks.CSDBLayoutVersion1)

		for fgId, ddb := range fgIdToDdb {
			if fgIds.Has(fgId) {
				csdb.FGIdToDDB[fgId] = ddb.Copy()
			}
		}
		// Serialize and store in cache
		var serializedData []byte
		if isDistributed {
			serializedData, err = csdb.SerializeForDistributedCache()
		} else {
			serializedData, err = csdb.SerializeForInMemory()
			if err == nil {
				// Store the serialized data in the cache
				err = cache.SetV2(entityLabel, key.Cols, serializedData)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to set cache for key %s", keyStr)
				}
				continue
			}
		}
		if err != nil {
			//log.Error().Err(err).Msgf("Failed to serialize csdb for key %s", keyStr)
			continue
		}

		cacheKeys = append(cacheKeys, key)
		cacheData = append(cacheData, serializedData)
	}
	if isDistributed {
		_ = cache.MultiSetV2(entityLabel, cacheKeys, cacheData)
	}
}

func (h *RetrieveHandler) persistToInMemoryCache(entityLabel string, retrieveData *RetrieveData, fgIds ds.Set[int], missingInMemKeys []*retrieve.Keys, isP2PEnabled bool) {
	log.Debug().Msgf("Persisting to in memory cache for entity %s, fgIds %v, missingInMemKeys %v", entityLabel, fgIds, missingInMemKeys)
	h.persistToCache(entityLabel, retrieveData, fgIds, missingInMemKeys, false, isP2PEnabled)
}

func (h *RetrieveHandler) persistToDistributedCache(entityLabel string, retrieveData *RetrieveData, fgIds ds.Set[int], missingDistKeys []*retrieve.Keys) {
	log.Debug().Msgf("Persisting to distributed cache for entity %s, fgIds %v, missingDistKeys %v", entityLabel, fgIds, missingDistKeys)
	h.persistToCache(entityLabel, retrieveData, fgIds, missingDistKeys, true, false)
}

// ... existing code ...

func GetFeature(dataType types.DataType, ddb *blocks.DeserializedPSDB, seq, numOfFeatures int, stringLengths []uint16, vectorLengths []uint16) ([]byte, error) {
	switch dataType {
	case types.DataTypeBool:
		data, err := ddb.GetBoolScalarFeature(seq)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeInt8, types.DataTypeInt16, types.DataTypeInt32, types.DataTypeInt64:
		data, err := ddb.GetNumericScalarFeature(seq)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeUint8, types.DataTypeUint16, types.DataTypeUint32, types.DataTypeUint64:
		data, err := ddb.GetNumericScalarFeature(seq)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeFP16, types.DataTypeFP32, types.DataTypeFP64, types.DataTypeFP8E4M3, types.DataTypeFP8E5M2:
		data, err := ddb.GetNumericScalarFeature(seq)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeString:
		data, err := ddb.GetStringScalarFeature(seq, numOfFeatures)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeBoolVector:
		data, err := ddb.GetBoolVectorFeature(seq, vectorLengths)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeStringVector:
		data, err := ddb.GetStringVectorFeature(seq, numOfFeatures, vectorLengths)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeInt8Vector, types.DataTypeInt16Vector, types.DataTypeInt32Vector, types.DataTypeInt64Vector:
		data, err := ddb.GetNumericVectorFeature(seq, vectorLengths)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeUint8Vector, types.DataTypeUint16Vector, types.DataTypeUint32Vector, types.DataTypeUint64Vector:
		data, err := ddb.GetNumericVectorFeature(seq, vectorLengths)
		if err != nil {
			return nil, err
		}
		return data, nil

	case types.DataTypeFP16Vector, types.DataTypeFP32Vector, types.DataTypeFP64Vector, types.DataTypeFP8E4M3Vector, types.DataTypeFP8E5M2Vector:
		data, err := ddb.GetNumericVectorFeature(seq, vectorLengths)
		if err != nil {
			return nil, err
		}
		return data, nil

	default:
		return nil, fmt.Errorf("unsupported data type: %v", dataType)
	}
}

// ParseFeatureLabel parses a feature label and returns the base label and quantization type
func ParseFeatureLabel(featureLabel string, originalDataType types.DataType) (string, types.DataType, error) {
	parts := strings.Split(featureLabel, featurePartsSeparator)
	if len(parts) != 2 || parts[1] == "" {
		return featureLabel, types.DataTypeUnknown, nil
	}

	quantType, err := types.ParseDataType(parts[1])
	if err != nil {
		log.Error().Err(err).Msgf("Invalid quantization type %s for feature %s", parts[1], featureLabel)
		return featureLabel, types.DataTypeUnknown, fmt.Errorf("invalid quantization type %s for feature %s: %w", parts[1], featureLabel, err)
	}

	// Check if source type is compatible with quantization
	if !quantization.IsQuantizationCompatible(originalDataType, quantType) {
		log.Error().Msgf("Incompatible quantization request from %v to %v for feature %s", originalDataType, quantType, featureLabel)
		return featureLabel, types.DataTypeUnknown, fmt.Errorf("incompatible quantization request from %v to %v for feature %s", originalDataType, quantType, featureLabel)
	}

	return parts[0], quantType, nil
}
