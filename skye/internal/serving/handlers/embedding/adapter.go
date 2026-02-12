package embedding

import (
	"bytes"
	"encoding/binary"

	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/rs/zerolog/log"
	protosd "google.golang.org/protobuf/proto"
)

func parseEmbeddingResponse(responseMap map[string]repositories.CandidateResponseStruct, cacheKeys map[string]repositories.CacheStruct) {
	for k, v := range cacheKeys {
		Embedding := convertFloat32ListToFloat64(v.Embedding)
		SearchEmbedding := convertFloat32ListToFloat64(v.SearchEmbedding)
		responseMap[k] = repositories.CandidateResponseStruct{
			Index: v.Index,
			EmbeddingResponse: &pb.CandidateEmbedding{
				Embedding:       Embedding,
				SearchEmbedding: SearchEmbedding,
				Id:              v.CandidateId,
			},
		}
	}
}

func convertFloat32ListToFloat64(src []float32) []float64 {
	dst := make([]float64, 0, len(src))
	for _, v := range src {
		dst = append(dst, float64(v))
	}
	return dst
}

func processCacheResponse(cacheKeys map[string]repositories.CacheStruct, cacheResp map[string][]byte, respMap map[string]repositories.CandidateResponseStruct, commonMetricTags []string, cacheType string) map[string]repositories.CacheStruct {
	missingCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))
	for k := range cacheKeys {
		if raw, ok := cacheResp[k]; ok {
			resp := &pb.CandidateEmbedding{}
			if err := protosd.Unmarshal(raw, resp); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal candidate response bytes")
				continue
			}
			respMap[k] = repositories.CandidateResponseStruct{
				Index:             cacheKeys[k].Index,
				EmbeddingResponse: resp,
			}
			delete(cacheKeys, k)
			if cacheType == "in_memory" {
				metric.Incr("in_memory_embedding_cache_hit", commonMetricTags)
			} else {
				metric.Incr("distributed_embedding_cache_hit", commonMetricTags)
			}
		} else {
			missingCacheKeys[k] = cacheKeys[k]
			if cacheType == "in_memory" {
				metric.Incr("in_memory_embedding_cache_miss", commonMetricTags)
			} else {
				metric.Incr("distributed_embedding_cache_miss", commonMetricTags)
			}
		}
	}
	return missingCacheKeys
}

func processCacheResponseDotProduct(cacheKeys map[string]repositories.CacheStruct, RequestEmbedding []float64, cacheResp map[string][]byte, respMap map[string]repositories.CandidateResponseStruct, commonMetricTags []string, cacheType string, foundcacheKeys map[string]repositories.CacheStruct) map[string]repositories.CacheStruct {
	missingCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))
	for k := range cacheKeys {
		if raw, ok := cacheResp[k]; ok {
			buf := bytes.NewReader(raw)
			var embedding []float32
			for buf.Len() > 0 {
				var f float32
				if err := binary.Read(buf, binary.LittleEndian, &f); err != nil {
					log.Error().Err(err).Msg("failed to read float32 from raw bytes")
					break
				}
				embedding = append(embedding, f)
			}
			score := CalculateDotProduct(RequestEmbedding, embedding)
			respMap[k] = repositories.CandidateResponseStruct{
				Index: cacheKeys[k].Index,
				DotProductResponse: &pb.CandidateEmbeddingScore{
					Score:       score,
					CandidateId: cacheKeys[k].CandidateId,
				},
			}
			foundcacheKeys[k] = repositories.CacheStruct{
				Index:       cacheKeys[k].Index,
				CandidateId: cacheKeys[k].CandidateId,
				Embedding:   embedding,
			}
			delete(cacheKeys, k)
			if cacheType == "in_memory" {
				metric.Incr("in_memory_embedding_cache_hit", commonMetricTags)
			} else {
				metric.Incr("distributed_embedding_cache_hit", commonMetricTags)
			}
		} else {
			missingCacheKeys[k] = cacheKeys[k]
			if cacheType == "in_memory" {
				metric.Incr("in_memory_embedding_cache_miss", commonMetricTags)
			} else {
				metric.Incr("distributed_embedding_cache_miss", commonMetricTags)
			}
		}
	}
	return missingCacheKeys
}
