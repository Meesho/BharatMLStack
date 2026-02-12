package registry

import (
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
)

type CreateStoreRequest struct {
	ConfId          int    `json:"conf_id"`
	Db              string `json:"db"`
	EmbeddingsTable string `json:"embeddings_table"`
	AggregatorTable string `json:"aggregator_table"`
}

type CreateFrequencyRequest struct {
	Frequency string `json:"frequency"`
}

type RegisterEntityRequest struct {
	Entity  string `json:"entity"`
	StoreId string `json:"store_id"`
}

type RegisterModelRequest struct {
	Entity                string                 `json:"entity"`
	Model                 string                 `json:"model"`
	EmbeddingStoreEnabled bool                   `json:"embedding_store_enabled"`
	EmbeddingStoreTtl     int                    `json:"embedding_store_ttl"`
	ModelConfig           map[string]interface{} `json:"model_config"`
	ModelType             string                 `json:"model_type"`
	TrainingDataPath      string                 `json:"training_data_path"`
	Metadata              config.Metadata        `json:"metadata"`
	JobFrequency          string                 `json:"job_frequency"`
	NumberOfPartitions    int                    `json:"number_of_partitions"`
	TopicName             string                 `json:"topic_name"`
}

type RegisterVariantRequest struct {
	Entity                     string                `json:"entity"`
	Model                      string                `json:"model"`
	Variant                    string                `json:"variant"`
	VectorDbConfig             config.VectorDbConfig `json:"vector_db_config"`
	VectorDbType               string                `json:"vector_db_type"`
	Filter                     []config.Criteria     `json:"filter"`
	Type                       enums.Type            `json:"type"`
	InMemoryCachingEnabled     bool                  `json:"in_memory_caching_enabled"`
	InMemoryCacheTTLSeconds    int                   `json:"in_memory_cache_ttl_seconds"`
	DistributedCachingEnabled  bool                  `json:"distributed_caching_enabled"`
	DistributedCacheTTLSeconds int                   `json:"distributed_cache_ttl_seconds"`
	RtPartition                int                   `json:"rt_partition"`
	RateLimiters               config.RateLimiter    `json:"rate_limiters"`
}

type PromoteVariantWriteRequest struct {
	Entity       string `json:"entity"`
	Model        string `json:"model"`
	Variant      string `json:"variant"`
	VectorDbHost string `json:"vector_db_host"`
}

type PromoteVariantReadRequest struct {
	Entity  string `json:"entity"`
	Model   string `json:"model"`
	Variant string `json:"variant"`
}
