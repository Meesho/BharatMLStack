package config

import "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"

type FeatureRegistry struct {
	Source   map[string]string `json:"source"`
	Entities map[string]Entity `json:"entities"`
	Storage  Storage           `json:"storage"`
	Security Security          `json:"security"`
}

type Security struct {
	Writer map[string]Property `json:"writer"`
	Reader map[string]Property `json:"reader"`
}

type Property struct {
	Token string `json:"token"`
}

type Entity struct {
	Label            string                  `json:"label"`
	Keys             map[string]Key          `json:"keys"`
	FeatureGroups    map[string]FeatureGroup `json:"feature-groups"`
	DistributedCache Cache                   `json:"distributed-cache"`
	InMemoryCache    Cache                   `json:"in-memory-cache"`
}

type Cache struct {
	Enabled          string `json:"enabled"`
	TtlInSeconds     int    `json:"ttl-in-seconds"`
	JitterPercentage int    `json:"jitter-percentage"`
	ConfId           int    `json:"conf-id"`
}

type Key struct {
	Sequence    int    `json:"sequence"`
	EntityLabel string `json:"entity-label"`
	ColumnLabel string `json:"column-label"`
}

type FeatureGroup struct {
	Id                      int                `json:"id"`
	ActiveVersion           string             `json:"active-version"`
	Columns                 map[string]Column  `json:"columns"`
	Features                map[string]Feature `json:"features"`
	StoreId                 string             `json:"store-id"`
	DataType                enums.DataType     `json:"data-type"`
	TtlInSeconds            uint64             `json:"ttl-in-seconds"`
	JobId                   string             `json:"job-id"`
	InMemoryCacheEnabled    bool               `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool               `json:"distributed-cache-enabled"`
	LayoutVersion           int                `json:"layout-version"`
}

type Column struct {
	Label              string `json:"label"`
	CurrentSizeInBytes int    `json:"current-size-in-bytes"`
}

type Feature struct {
	Labels        string                 `json:"labels"`
	DefaultValues string                 `json:"default-values"`
	FeatureMeta   map[string]FeatureMeta `json:"feature-meta"`
}

type FeatureMeta struct {
	Sequence             int    `json:"sequence"`
	DefaultValuesInBytes []byte `json:"default-value"`
	StringLength         uint16 `json:"string-length"`
	VectorLength         uint16 `json:"vector-length"`
}

type Storage struct {
	Stores map[string]Store `json:"stores"`
}

type Store struct {
	DbType               string   `json:"db-type"`
	ConfId               int      `json:"conf-id"`
	Table                string   `json:"table"`
	MaxColumnSizeInBytes int      `json:"max-column-size-in-bytes"`
	MaxRowSizeInBytes    int      `json:"max-row-size-in-bytes"`
	PrimaryKeys          []string `json:"primary-keys"`
	TableTtl             int      `json:"table-ttl"`
}
