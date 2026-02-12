package config

import "github.com/Meesho/BharatMLStack/skye/internal/config/enums"

type Skye struct {
	Entity                              map[string]Models
	Storage                             Storage
	DefaultInMemoryCachingTTLSeconds    int
	DefaultDistributedCachingTTLSeconds int
}

type RateLimiter struct {
	RateLimit  int
	BurstLimit int
}

type Models struct {
	StoreId string
	Models  map[string]Model
}

type Model struct {
	EmbeddingStoreEnabled bool
	EmbeddingStoreTtl     int
	EmbeddingStoreVersion int
	TopicName             string
	Metadata              Metadata
	ModelConfig           ModelConfig
	TrainingDataPath      string
	Variants              map[string]Variant
	ModelType             enums.ModelType
	PartitionStates       map[string]int
	NumberOfPartitions    int
	JobFrequency          string
}

type ModelConfig struct {
	DistanceFunction string `json:"distance_function"`
	VectorDimension  uint64 `json:"vector_dimension"`
}

type Variant struct {
	ReqEmbLoggingPercentage             int
	RTPartition                         int
	RateLimiter                         RateLimiter
	Filter                              map[string][]Criteria
	Enabled                             bool
	VariantState                        enums.VariantState
	VectorDbType                        enums.VectorDbType
	VectorDbConfig                      VectorDbConfig
	RtDeltaProcessing                   bool
	VectorDbReadVersion                 int
	VectorDbWriteVersion                int
	EmbeddingStoreReadVersion           int
	EmbeddingStoreWriteVersion          int
	Onboarded                           bool
	PartialHitEnabled                   bool
	BackupConfig                        BackupConfig
	DefaultResponsePercentage           int
	InMemoryCachingEnabled              bool
	InMemoryCacheTTLSeconds             int
	DistributedCachingEnabled           bool
	DistributedCacheTTLSeconds          int
	Type                                enums.Type
	PartialHitDisabled                  bool
	TestConfig                          TestConfig
	EmbeddingRetrievalInMemoryConfig    Config
	EmbeddingRetrievalDistributedConfig Config
	DotProductInMemoryConfig            Config
	DotProductDistributedConfig         Config
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
	FieldSchema   string `json:"field_schema"`
	DefaultValue  string `json:"default_value"`
	LookupEnabled bool   `json:"lookup_enabled"`
	IsPrincipal   bool   `json:"is_principal"`
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
