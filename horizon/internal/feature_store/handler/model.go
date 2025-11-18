package handler

type Conf struct {
	EntityConf             map[string]EntityConf   `json:"entities"`
	Storage                Storage                 `json:"storage"`
	InMemory               map[string]InMemoryConf `json:"in-memory"`
	McacheEnabled          ExperimentControl       `json:"mcache-enabled"` //for all clusters
	ErrorLoggingPercentage string                  `json:"error-logging-percentage"`
}

type InMemoryConf struct {
	FeatureGroupConf map[string]string `json:"feature-groups"`
}

type EntityConf struct {
	Label                           string                      `json:"label"`
	Keys                            map[int]Keys                `json:"keys"`
	FeatureGroupConf                map[string]FeatureGroupConf `json:"feature-groups"`
	InMemoryEnabled                 string                      `json:"in-memory-enabled"`
	DistributedCacheEnabled         string                      `json:"distributed-cache-enabled"`
	RedisCacheTTL                   string                      `json:"redis-cache-ttl"`
	DistributeCacheInfraVersion     string                      `json:"distributed-cache-infra-version"`
	DistributedCacheWritePercentage string                      `json:"distributed-cache-write-percentage"`
}

type Keys struct {
	Sequence    string `json:"sequence"`
	EntityLabel string `json:"entity-label"`
	ColumnLabel string `json:"column-label"`
}

type FeatureGroupConf struct {
	FeatureColumnConf map[string]FeatureColumnConf `json:"columns"`
	StoreId           string                       `json:"storeId"`
	Label             string                       `json:"label"`
	DataType          string                       `json:"data-type"`
	Ttl               string                       `json:"ttl"`
	JobId             string                       `json:"jobId"`
	MaxSizeByte       string                       `json:"max-size-byte"`
}

type FeatureColumnConf struct {
	ColumnLabel     string                 `json:"column-label"`
	ActiveConfig    string                 `json:"active-config"`
	CurrentSizeByte string                 `json:"current-size-byte"`
	FeatureConf     map[string]FeatureConf `json:"config"`
}

type FeatureConf struct {
	FeatureLabels        string `json:"feature-labels"`
	FeatureDefaultValues string `json:"feature-default-values"`
}

type Storage struct {
	RateLimiter             RateLimiter       `json:"rate-limiter"`
	RedisCacheTTL           string            `json:"redis-cache-ttl"`
	Stores                  map[string]string `json:"stores"`
	DefaultScyllaPercentage string            `json:"default-scylla-percentage"`
}

type RateLimiter struct {
	PermitsPerSecond string `json:"permits-per-second"`
	WarmupPeriod     string `json:"warmup-period"`
}

type ExperimentControl struct {
	Enabled           string `json:"enabled"`
	PercentageEnabled string `json:"percentage-enabled"`
	Deadline          string `json:"deadline"`
	JitterPercentage  string `json:"jitter-percentage"`
}

type FsConfigResponse struct {
	Data []byte `json:"data"`
}
