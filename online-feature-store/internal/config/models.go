package config

import (
	"github.com/Meesho/orion/internal/config/enums"
)

type _version int
type _fgId int
type _featureLabel string
type _entityLabel string
type _pkId string
type _columnLabel string
type _defaultInBytes []byte

type FeatureRegistry struct {
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
	P2PCache         Cache                   `json:"p2p-cache"`
}

type Cache struct {
	Enabled          bool `json:"enabled"`
	TtlInSeconds     int  `json:"ttl-in-seconds"`
	JitterPercentage int  `json:"jitter-percentage"`
	ConfId           int  `json:"conf-id"`
}

type Key struct {
	Sequence    int    `json:"sequence"`
	EntityLabel string `json:"entity-label"`
	ColumnLabel string `json:"column-label"`
}

type FeatureGroup struct {
	Id                      int                      `json:"id"`
	ActiveVersion           string                   `json:"active-version"`
	Columns                 map[string]Column        `json:"columns"`
	Features                map[string]FeatureSchema `json:"features"`
	StoreId                 string                   `json:"store-id"`
	DataType                enums.DataType           `json:"data-type"`
	TtlInSeconds            uint64                   `json:"ttl-in-seconds"`
	JobId                   string                   `json:"job-id"`
	InMemoryCacheEnabled    bool                     `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool                     `json:"distributed-cache-enabled"`
	LayoutVersion           int                      `json:"layout-version"`
}

type Column struct {
	Label              string `json:"label"`
	CurrentSizeInBytes int    `json:"current-size-in-bytes"`
}

type FeatureSchema struct {
	Labels      string                 `json:"labels"`
	FeatureMeta map[string]FeatureMeta `json:"feature-meta"`
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

// TODO: Change this to JSON
type Store struct {
	DbType               string `json:"db-type"`
	ConfId               int    `json:"conf-id"`
	Table                string `json:"table"`
	MaxColumnSizeInBytes int    `json:"max-column-size-in-bytes"`
	MaxRowSizeInBytes    int    `json:"max-row-size-in-bytes"`
}

type NormalizedFGConfig struct {
	Versions        map[_version][]string
	DefaultsInBytes map[_version]map[string][]byte
	Sequences       map[_version]map[string]int
	StringLengths   map[_version][]uint16
	VectorLengths   map[_version][]uint16
	NumofFeatures   map[_version]int
	Columns         []string
	ActiveVersion   _version
}
type NormalizedEntity struct {
	PrimaryKeyColumnNames []string
	ColumnToPKIdMap       map[string]string
	FGs                   map[int]NormalizedFGConfig
}
type NormalizedEntities struct {
	Entities map[string]NormalizedEntity
}
