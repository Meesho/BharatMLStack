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
	// AutoPromote opts this store into reconciler-driven auto-promotion: once a
	// READY version newer than the active one has every shard warm, the control
	// plane promotes it automatically (no manual promote call). Default false.
	AutoPromote bool `json:"autoPromote,omitempty"`
	// KeepVersions bounds on-disk retention: the reconciler retires versions
	// older than the newest N (the active version + its rollback chain), freeing
	// their SSTs from pod disk. 0 = use the default (2) for auto-promote stores,
	// and disabled for stores not managed by the reconciler. The active and
	// rollback versions are always kept regardless of N.
	KeepVersions int `json:"keepVersions,omitempty"`
	// RolloutCfg configures gradual version rollout with traffic-percentage ramp.
	// When set, the dataloader ramps traffic from old → new version instead of
	// doing an atomic flip.
	RolloutCfg *RolloutConfig `json:"rolloutCfg,omitempty"`
}

// RolloutConfig controls the gradual traffic ramp during version promotion.
type RolloutConfig struct {
	// Steps is the ordered list of traffic percentages to ramp through.
	// Each value is 0–100. The last value should be 100.
	// Default: [10, 25, 50, 75, 100]
	Steps []int `json:"steps,omitempty"`
	// StepIntervalSec is the dwell time at each percentage step in seconds.
	// Default: 60.
	StepIntervalSec int `json:"stepIntervalSec,omitempty"`
}

// ClientConfig holds SDK connection-pool and transport settings for a store.
// Persisted in etcd at /config/mnemo/tenants/{tenant}/stores/{store}/clientConfig.
// The SDK fetches this once on init and applies the values; changes require a
// client restart (or a future watch). Zero values mean "use SDK default".
type ClientConfig struct {
	// ConnectTimeoutMs is the TCP dial timeout in milliseconds. Default: 5000.
	ConnectTimeoutMs int `json:"connectTimeoutMs,omitempty"`
	// RequestTimeoutMs is the per-request deadline in milliseconds. Default: 100.
	RequestTimeoutMs int `json:"requestTimeoutMs,omitempty"`
	// KeepAliveIntervalMs is the TCP keepalive probe interval. Default: 15000.
	KeepAliveIntervalMs int `json:"keepAliveIntervalMs,omitempty"`
	// KeepAliveTimeoutMs is the keepalive timeout before the connection is
	// considered dead and closed. Default: 5000.
	KeepAliveTimeoutMs int `json:"keepAliveTimeoutMs,omitempty"`
	// IdleTimeoutMs evicts connections that have been idle longer than this.
	// Default: 60000.
	IdleTimeoutMs int `json:"idleTimeoutMs,omitempty"`
	// IdleCheckIntervalMs is the sweep interval for idle eviction. Default: 10000.
	IdleCheckIntervalMs int `json:"idleCheckIntervalMs,omitempty"`
	// MinConnsPerPod is the warm floor: pre-dialed connections kept per pod even
	// when idle. Default: 1.
	MinConnsPerPod int `json:"minConnsPerPod,omitempty"`
	// MaxConnsPerPod is the pool ceiling per pod. Default: 4.
	MaxConnsPerPod int `json:"maxConnsPerPod,omitempty"`
	// DNSRefreshIntervalMs is the DNS re-resolve cadence for K8s headless
	// Services. Ignored in assignment-aware mode. Default: 30000.
	DNSRefreshIntervalMs int `json:"dnsRefreshIntervalMs,omitempty"`
	// WarmUpOnTopologyChange pre-dials MinConnsPerPod connections to newly
	// discovered pods when the assignment changes. Default: true.
	WarmUpOnTopologyChange *bool `json:"warmUpOnTopologyChange,omitempty"`
}

// PodData is the ephemeral registration for a single pod, lease-bound in etcd.
type PodData struct {
	NodeIP string `json:"nodeIP"`
	PodIP  string `json:"podIP"`
	// Port is the read server's TCP port. 0 means "unset" — placement falls back
	// to the default 9091 (one-readserver-per-IP / K8s pod model). Set explicitly
	// when multiple read servers are co-located on one host (host networking).
	Port           int      `json:"port,omitempty"`
	ServingVersion string   `json:"servingVersion"`
	WarmVersions   []string `json:"warmVersions"`
	LoadingVersion string   `json:"loadingVersion,omitempty"`
	// RolloutVersion is the version currently being rolled out on this pod.
	// Empty when no rollout is active.
	RolloutVersion string `json:"rolloutVersion,omitempty"`
	// RolloutPct is the current traffic percentage routed to RolloutVersion (0–100).
	// 0 when no rollout is active.
	RolloutPct int `json:"rolloutPct,omitempty"`
}
