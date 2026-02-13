package inmemorycache

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/Meesho/BharatMLStack/skye/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	protosd "google.golang.org/protobuf/proto"
)

var (
	inMemoryDatabase Database
	once             sync.Once
)

type FreeCache struct {
	cache inmemorycache.InMemoryCache
}

func initFreeCache() Database {
	if inMemoryDatabase == nil {
		once.Do(func() {
			inmemorycache.Init(1)
			inMemoryDatabase = &FreeCache{
				cache: inmemorycache.Instance(),
			}
		})
	}
	return inMemoryDatabase
}

func (f *FreeCache) MGet(keys map[string]repositories.CacheStruct, metricTags []string) map[string][]byte {
	startTime := time.Now()
	responseMap := make(map[string][]byte)
	for key := range keys {
		metric.Incr("in_memory_cache_mget", metricTags)
		byteResponse, err := f.cache.Get([]byte(key))
		if err == nil {
			responseMap[key] = byteResponse
		}
	}
	metric.Timing("in_memory_cache_mget_latency", time.Since(startTime), metricTags)
	return responseMap
}

func (f *FreeCache) MSet(responseData map[string]repositories.CandidateResponseStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, byteResponseMap map[string][]byte, metricTags []string) {
	startTime := time.Now()
	for key, value := range responseData {
		if _, ok := missingCacheKeys[key]; !ok {
			continue
		}
		metric.Incr("in_memory_cache_mset", metricTags)
		var toMarshal protosd.Message
		if value.EmbeddingResponse != nil {
			toMarshal = value.EmbeddingResponse
		} else if value.DotProductResponse != nil {
			toMarshal = value.DotProductResponse
		} else {
			toMarshal = value.Response
		}
		if byteResponseMap[key] != nil {
			f.cache.SetEx([]byte(key), byteResponseMap[key], ttl)
		} else {
			if valueInBytes, err := protosd.Marshal(toMarshal); err == nil {
				f.cache.SetEx([]byte(key), valueInBytes, ttl)
			}
		}
	}
	metric.Timing("in_memory_cache_mset_latency", time.Since(startTime), metricTags)
}

func (f *FreeCache) MSetDotProduct(cacheKeys map[string]repositories.CacheStruct, foundcacheKeys map[string]repositories.CacheStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, metricTags []string) {
	startTime := time.Now()
	for key, cacheStruct := range cacheKeys {
		if _, ok := missingCacheKeys[key]; !ok {
			continue
		}
		metric.Incr("in_memory_cache_mset", metricTags)
		emb := cacheStruct.Embedding
		if len(emb) > 0 {
			buf := make([]byte, 4*len(emb))
			for i, v := range emb {
				binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
			}
			f.cache.SetEx([]byte(key), buf, ttl)
		}
	}
	for key, cacheStruct := range foundcacheKeys {
		if _, ok := missingCacheKeys[key]; !ok {
			continue
		}
		if _, ok := cacheKeys[key]; ok {
			continue
		}
		metric.Incr("in_memory_cache_mset", metricTags)
		emb := cacheStruct.Embedding
		if len(emb) > 0 {
			buf := make([]byte, 4*len(emb))
			for i, v := range emb {
				binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
			}
			f.cache.SetEx([]byte(key), buf, ttl)
		}
	}
	metric.Timing("in_memory_cache_mset_latency", time.Since(startTime), metricTags)
}
