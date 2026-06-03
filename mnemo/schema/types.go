package schema

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

// PodData is the ephemeral registration for a single pod, lease-bound in etcd.
type PodData struct {
	NodeIP         string   `json:"nodeIP"`
	PodIP          string   `json:"podIP"`
	ServingVersion string   `json:"servingVersion"`
	WarmVersions   []string `json:"warmVersions"`
	LoadingVersion string   `json:"loadingVersion,omitempty"`
}
