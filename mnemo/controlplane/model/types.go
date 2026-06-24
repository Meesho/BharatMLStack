package model

// StoreConfig is the configuration for a tenant's store, persisted in etcd.
type StoreConfig struct {
	Tenant     string `json:"tenant"`
	Store      string `json:"store"`
	EntityKey  string `json:"entityKey"`  // e.g. "catalog_id|geohash"
	ShardCount int    `json:"shardCount"` // S
}

// VersionMeta is the metadata for a single version, stored at the version key.
type VersionMeta struct {
	Date       string              `json:"date"`       // e.g. "20260528"
	Run        string              `json:"run"`        // e.g. "001"
	ShardCount int                 `json:"shardCount"` // S (snapshot at publish time)
	Status     VersionStatus       `json:"status"`
	Assignment map[string][]string `json:"assignment"` // shard ID → []pod addresses
}

// DataflowConfig holds pipeline parameters for a store's SST producer job.
// Persisted in etcd at /config/mnemo/tenants/{tenant}/stores/{store}/dataflow.
type DataflowConfig struct {
	SourcePath  string         `json:"sourcePath"`           // GCS path to source parquet files
	GcsOutRoot  string         `json:"gcsOutRoot"`           // GCS root for SST output
	NumShards   int            `json:"numShards"`            // number of shards
	TargetSstMB int            `json:"targetSstMB"`          // target SST file size in MB
	MetadataURL string         `json:"metadataUrl"`          // Horizon metadata API URL
	JobID       string         `json:"jobId"`                // Horizon job ID
	NumOfFiles  int            `json:"numOfFiles,omitempty"` // 0 = all files in the partition
	RocksDBCfg  map[string]any `json:"rocksdbCfg,omitempty"` // compression, bloom_bits_per_key, block_size_kb
	QcCfg       map[string]any `json:"qcCfg,omitempty"`      // max_shard_skew, min_total_rows
}

// PodData is the ephemeral registration for a single pod, lease-bound in etcd.
type PodData struct {
	NodeIP         string   `json:"nodeIP"`
	PodIP          string   `json:"podIP"`
	ServingVersion string   `json:"servingVersion"`
	WarmVersions   []string `json:"warmVersions"`
	LoadingVersion string   `json:"loadingVersion,omitempty"`
}
