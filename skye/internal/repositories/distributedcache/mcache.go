package distributedcache

import (
	"context"
	"encoding/binary"
	"math"

	// "crypto/sha1" // removed, using md5 for all hashes

	"math/rand"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/memcoil/v2/pkg/memcoil"
	memcoilProvider "github.com/Meesho/memcoil/v2/pkg/provider"
	"github.com/rs/zerolog/log"
	protosd "google.golang.org/protobuf/proto"
)

var (
	cacheDB Database
)

type Mcache struct {
	client memcoil.Client
}

func initMCache(mcacheId int) Database {
	if cacheDB == nil {
		once.Do(func() {
			clientProvider := memcoilProvider.GetInstance()
			converter := memcoil.NewBytesConverter()
			c := clientProvider.GetMemCoilClient(mcacheId, converter)
			pingResponse := c.Ping(context.Background())
			if err := pingResponse.Err(); err != nil {
				metric.Incr("distributed_cache_v2_mcache_failure", []string{})
				log.Panic().Msgf("MCache ping failed %v", err)
			}
			cacheDB = &Mcache{
				client: c,
			}
		})
	}
	return cacheDB
}

func (m *Mcache) MGet(keys map[string]repositories.CacheStruct, tags []string) (map[string][]byte, error) {
	startTime := time.Now()
	cacheKeyAndGenericResponse := make(map[string][]byte)
	keysSlice := make([]string, 0, len(keys))
	for k := range keys {
		keysSlice = append(keysSlice, k)
	}
	metric.Count("distributed_cache_v2_mget", int64(len(keysSlice)), tags)
	cmd := m.client.MGet(context.Background(), keysSlice...)
	vals, err := cmd.Result()
	if err != nil {
		metric.Incr("distributed_cache_v2_mget_failure", tags)
		log.Error().Msgf("Error fetching data from distributed cache for keys: %v, error: %v", keys, err)
		return cacheKeyAndGenericResponse, err
	}
	for i, val := range vals {
		if val != nil {
			cacheKeyAndGenericResponse[keysSlice[i]] = []byte(val.(string))
		}
	}
	metric.Timing("distributed_cache_v2_mget_latency", time.Since(startTime), tags)
	return cacheKeyAndGenericResponse, nil
}

func (m *Mcache) MSet(responseData map[string]repositories.CandidateResponseStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, byteResponseMap map[string][]byte, tags []string) {
	startTime := time.Now()
	finalTTL := getFinalTTLWithJitter(ttl)
	pipe := m.client.Pipeline()
	count := 0
	for key, value := range responseData {
		if _, ok := missingCacheKeys[key]; !ok {
			continue
		}
		metric.Incr("distributed_cache_v2_mset", tags)
		var toMarshal protosd.Message
		if value.EmbeddingResponse != nil {
			toMarshal = value.EmbeddingResponse
		} else if value.DotProductResponse != nil {
			toMarshal = value.DotProductResponse
		} else {
			toMarshal = value.Response
		}
		dataBytes, err := protosd.Marshal(toMarshal)
		if err != nil {
			metric.Incr("distributed_cache_v2_mset_failure", tags)
			log.Error().Msgf("Error during msgpack marshalling for key %s: %v", key, err)
			continue
		}
		byteResponseMap[key] = dataBytes
		pipe.Set(context.Background(), key, dataBytes, time.Second*time.Duration(finalTTL))
		count++
	}
	_, err := pipe.Exec(context.Background())
	if err != nil {
		metric.Count("distributed_cache_v2_mset_failure", int64(count), tags)
		log.Error().Msgf("Error while Persisting Data to dragonfly: %v", err)
		return
	}
	metric.Timing("distributed_cache_v2_mset_latency", time.Since(startTime), tags)
}

func (m *Mcache) MSetDotProduct(cacheKeys map[string]repositories.CacheStruct, foundcacheKeys map[string]repositories.CacheStruct, missingCacheKeys map[string]repositories.CacheStruct, ttl int, tags []string) {
	startTime := time.Now()
	finalTTL := getFinalTTLWithJitter(ttl)
	pipe := m.client.Pipeline()
	count := 0
	for key, cacheStruct := range cacheKeys {
		if _, ok := missingCacheKeys[key]; !ok {
			continue
		}
		metric.Incr("distributed_cache_v2_mset", tags)
		emb := cacheStruct.Embedding
		if len(emb) > 0 {
			buf := make([]byte, 4*len(emb))
			for i, v := range emb {
				binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
			}
			pipe.Set(context.Background(), key, buf, time.Second*time.Duration(finalTTL))
		}
		count++
	}
	for key, cacheStruct := range foundcacheKeys {
		if _, ok := missingCacheKeys[key]; !ok {
			continue
		}
		if _, ok := cacheKeys[key]; ok {
			continue
		}
		metric.Incr("distributed_cache_v2_mset", tags)
		emb := cacheStruct.Embedding
		if len(emb) > 0 {
			buf := make([]byte, 4*len(emb))
			for i, v := range emb {
				binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
			}
			pipe.Set(context.Background(), key, buf, time.Second*time.Duration(finalTTL))
		}
		count++
	}
	_, err := pipe.Exec(context.Background())
	if err != nil {
		metric.Count("distributed_cache_v2_mset_failure", int64(count), tags)
		log.Error().Msgf("Error while Persisting Data to dragonfly: %v", err)
		return
	}
	metric.Timing("distributed_cache_v2_mset_latency", time.Since(startTime), tags)
}

func getFinalTTLWithJitter(ttl int) int {
	jitterPercent := 10
	jitterRange := ttl * jitterPercent / 100
	jitter := rand.Intn(2*jitterRange+1) - jitterRange
	finalTTL := ttl + jitter

	if finalTTL < 1 {
		finalTTL = ttl
	}
	return finalTTL
}
