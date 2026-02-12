package etcd

import "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd/enums"

type Skye struct {
	Entity                              map[string]Models
	Storage                             Storage
	DefaultInMemoryCachingTTLSeconds    int
	DefaultDistributedCachingTTLSeconds int
}

type RateLimiter struct {
	RateLimit  int `json:"rate_limit"`
	BurstLimit int `json:"burst_limit"`
}

type Models struct {
	StoreId string
	Models  map[string]Model
	Filters map[string]Criteria
}

type Model struct {
	EmbeddingStoreEnabled bool               `json:"embedding_store_enabled"`
	EmbeddingStoreTtl     int                `json:"embedding_store_ttl"`
	EmbeddingStoreVersion int                `json:"embedding_store_version"`
	MqId                  int                `json:"mq_id"`
	TopicName             string             `json:"topic_name"`
	Metadata              Metadata           `json:"metadata"`
	ModelConfig           ModelConfig        `json:"model_config"`
	TrainingDataPath      string             `json:"training_data_path"`
	Variants              map[string]Variant `json:"variants"`
	ModelType             enums.ModelType    `json:"model_type"`
	PartitionStates       map[string]int64   `json:"partition_states"`
	NumberOfPartitions    int                `json:"number_of_partitions"`
	FailureProducerMqId   int                `json:"failure_producer_mq_id"`
	JobFrequency          string             `json:"job_frequency"`
}

type ModelConfig struct {
	DistanceFunction string `json:"distance_function"`
	VectorDimension  uint64 `json:"vector_dimension"`
}

type Variant struct {
	ReqEmbLoggingPercentage             int                   `json:"req_emb_logging_percentage"`
	RTPartition                         int                   `json:"rt_partition"`
	RateLimiter                         RateLimiter           `json:"rate_limiter"`
	Filter                              map[string][]Criteria `json:"filter"`
	Enabled                             bool                  `json:"enabled"`
	VariantState                        enums.VariantState    `json:"variant_state"`
	VectorDbType                        enums.VectorDbType    `json:"vector_db_type"`
	VectorDbConfig                      VectorDbConfig        `json:"vector_db_config"`
	RtDeltaProcessing                   bool                  `json:"rt_delta_processing"`
	VectorDbReadVersion                 int                   `json:"vector_db_read_version"`
	VectorDbWriteVersion                int                   `json:"vector_db_write_version"`
	EmbeddingStoreReadVersion           int                   `json:"embedding_store_read_version"`
	EmbeddingStoreWriteVersion          int                   `json:"embedding_store_write_version"`
	Onboarded                           bool                  `json:"onboarded"`
	PartialHitEnabled                   bool                  `json:"partial_hit_enabled"`
	BackupConfig                        BackupConfig          `json:"backup_config"`
	DefaultResponsePercentage           int                   `json:"default_response_percentage"`
	InMemoryCachingEnabled              bool                  `json:"in_memory_caching_enabled"`
	InMemoryCacheTTLSeconds             int                   `json:"in_memory_cache_ttl_seconds"`
	DistributedCachingEnabled           bool                  `json:"distributed_caching_enabled"`
	DistributedCacheTTLSeconds          int                   `json:"distributed_cache_ttl_seconds"`
	Type                                enums.Type            `json:"type"`
	PartialHitDisabled                  bool                  `json:"partial_hit_disabled"`
	TestConfig                          TestConfig            `json:"test_config"`
	EmbeddingRetrievalInMemoryConfig    Config                `json:"embedding_retrieval_in_memory_config"`
	EmbeddingRetrievalDistributedConfig Config                `json:"embedding_retrieval_distributed_config"`
	DotProductInMemoryConfig            Config                `json:"dot_product_in_memory_config"`
	DotProductDistributedConfig         Config                `json:"dot_product_distributed_config"`
}

type Config struct {
	Enabled bool
	TTL     int
}

type BackupConfig struct {
	Enabled          bool
	RoutePercentage  int
	DualWriteEnabled bool
	Host             string
	Version          int
}

type TestConfig struct {
	VectorDbType enums.VectorDbType
	Percentage   int
	Entity       string
	Model        string
	Variant      string
	Version      int
}

type Metadata struct {
	Entity  string `json:"entity"`
	KeyType string `json:"key-type"`
	Details struct {
		CatalogId struct {
			FeatureGroup string `json:"feature-group"`
			Feature      string `json:"feature"`
		} `json:"catalog_id"`
	} `json:"details"`
}

type VectorDbConfig struct {
	ReadHost    string             `json:"read_host"`
	WriteHost   string             `json:"write_host"`
	Port        string             `json:"port"`
	Http2Config Http2Config        `json:"http2config"`
	Http1Config Http1Config        `json:"http1config"`
	Params      map[string]string  `json:"params"`
	Payload     map[string]Payload `json:"payload"`
}

type Http2Config struct {
	Deadline       int    `json:"deadline"`
	WriteDeadline  int    `json:"write_deadline"`
	KeepAliveTime  string `json:"keep_alive_time"`
	ThreadPoolSize string `json:"thread_pool_size"`
	IsPlainText    bool   `json:"is_plain_text"`
}

type Http1Config struct {
	TimeoutInMs               int `json:"timeout_in_ms"`
	DialTimeoutInMs           int `json:"dial_timeout_in_ms"`
	MaxIdleConnections        int `json:"max_idle_connections"`
	MaxIdleConnectionsPerHost int `json:"max_idle_connections_per_host"`
	IdleConnTimeoutInMs       int `json:"idle_conn_timeout_in_ms"`
	KeepAliveTimeoutInMs      int `json:"keep_alive_timeout_in_ms"`
}

type Payload struct {
	FieldSchema  string `json:"field_schema"`
	DefaultValue string `json:"default_value"`
}

type Storage struct {
	Stores      map[string]Data
	Frequencies string
}

type Data struct {
	ConfId          int    `json:"conf_id"`
	EmbeddingTable  string `json:"embeddings_table"`
	AggregatorTable string `json:"aggregator_table"`
	Db              string `json:"db"`
}

type Criteria struct {
	ColumnName   string `json:"column_name"`
	FilterValue  string `json:"filter_value"`
	DefaultValue string `json:"default_value"`
}
