package similar_candidate

import (
	"encoding/binary"
	"hash/fnv"
	"strings"
	"unsafe"

	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
)

const (
	SimilarCandidate  = "sc"
	CacheKeySeparator = ":"
	CacheVersion      = "V2"
)

func GetCacheKeysForEmbeddings(request SkyeStructRequest) map[string]repositories.CacheStruct {
	cacheKeys := make(map[string]repositories.CacheStruct, len(request.Embeddings))
	for index, embedding := range request.Embeddings {
		key := buildDetailedCacheKey(SimilarCandidate, request.Entity, request.ModelName, request.Variant, getHashForEmbedding(embedding), getHash(request.Filters[index]), getHash(request.Attributes))
		if _, ok := cacheKeys[key]; ok {
			cacheStruct := cacheKeys[key]
			cacheStruct.Index = append(cacheStruct.Index, index)
			cacheKeys[key] = cacheStruct
		} else {
			cacheKeys[key] = repositories.CacheStruct{
				Index:       []int{index},
				Embedding:   embedding,
				CandidateId: "",
				Filters:     request.Filters[index],
			}
		}
	}
	return cacheKeys
}

func GetCacheKeysForCandidateIds(request SkyeStructRequest) map[string]repositories.CacheStruct {
	cacheKeys := make(map[string]repositories.CacheStruct, len(request.CandidateIds))
	for index, id := range request.CandidateIds {
		key := buildDetailedCacheKey(SimilarCandidate, request.Entity, request.ModelName, request.Variant, id, getHash(request.Filters[index]), getHash(request.Attributes))
		if _, ok := cacheKeys[key]; ok {
			cacheStruct := cacheKeys[key]
			cacheStruct.Index = append(cacheStruct.Index, index)
			cacheKeys[key] = cacheStruct
		} else {
			cacheKeys[key] = repositories.CacheStruct{
				Index:       []int{index},
				Embedding:   nil,
				CandidateId: id,
				Filters:     request.Filters[index],
			}
		}
	}
	return cacheKeys
}

// Optimized: Use FNV hash instead of MD5, unsafe conversion for performance
func getHashForEmbedding(embedding []float32) string {
	if len(embedding) == 0 {
		return "e"
	}

	// Use FNV hash which is faster than MD5
	hasher := fnv.New64a()

	// Convert float32 slice to bytes efficiently using unsafe
	// This is safe because we're only reading the data
	if len(embedding) > 0 {
		// Safe unsafe conversion: convert each float32 to bytes
		for i := 0; i < len(embedding); i++ {
			floatBytes := (*[4]byte)(unsafe.Pointer(&embedding[i]))
			hasher.Write(floatBytes[:])
		}
	}

	hash := hasher.Sum64()

	// Convert to hex string efficiently
	const hexChars = "0123456789abcdef"
	result := make([]byte, 16) // 64 bits = 16 hex chars
	for i := 0; i < 8; i++ {
		b := byte(hash >> (8 * (7 - i)))
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}

// Optimized: Remove reflection, use type switches for common cases
func getHash(any interface{}) string {
	if any == nil {
		return "e"
	}

	// Fast path for common types to avoid reflection
	switch v := any.(type) {
	case []*pb.Filter:
		if len(v) == 0 {
			return "e"
		}
		hasher := fnv.New64a()
		for _, filter := range v {
			if filter != nil {
				hasher.Write([]byte(filter.Field))
				binary.Write(hasher, binary.LittleEndian, int32(filter.Op))
				for _, value := range filter.Values {
					hasher.Write([]byte(value))
				}
			}
		}
		hash := hasher.Sum64()
		return hashToHexString(hash)

	case []string:
		if len(v) == 0 {
			return "e"
		}
		hasher := fnv.New64a()
		for _, s := range v {
			hasher.Write([]byte(s))
		}
		hash := hasher.Sum64()
		return hashToHexString(hash)

	case string:
		if v == "" {
			return "e"
		}
		hasher := fnv.New64a()
		hasher.Write([]byte(v))
		hash := hasher.Sum64()
		return hashToHexString(hash)

	default:
		// Fallback for unknown types - simple hash
		hasher := fnv.New64a()
		hasher.Write([]byte("unknown"))
		hash := hasher.Sum64()
		return hashToHexString(hash)
	}
}

// Helper function to convert hash to hex string efficiently
func hashToHexString(hash uint64) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, 16) // 64 bits = 16 hex chars
	for i := 0; i < 8; i++ {
		b := byte(hash >> (8 * (7 - i)))
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}

func buildDetailedCacheKey(prefix, entity, model, variant, id, filter, attribute string) string {
	var b strings.Builder
	// Pre-allocate capacity to avoid reallocations
	totalLen := len(prefix) + len(entity) + len(model) + len(variant) + len(id) +
		len(filter) + len(attribute) + len(CacheVersion) + 35 // 35 for separators
	b.Grow(totalLen)

	b.WriteString(prefix)
	b.WriteString(CacheKeySeparator)
	b.WriteString(entity)
	b.WriteString(CacheKeySeparator)
	b.WriteString(model)
	b.WriteString(CacheKeySeparator)
	b.WriteString(variant)
	b.WriteString(CacheKeySeparator)
	b.WriteString(id)
	b.WriteString(CacheKeySeparator)
	b.WriteString(filter)
	b.WriteString(CacheKeySeparator)
	b.WriteString(attribute)
	b.WriteString(CacheKeySeparator)
	b.WriteString(CacheVersion)
	return b.String()
}
