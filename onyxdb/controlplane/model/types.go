package model

import "fmt"

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
	// LoadCfg holds per-load read-server options. The data loader re-reads it
	// from etcd right before every IPC load, so a change applies to the next
	// load with no restart or redeploy.
	LoadCfg *LoadConfig `json:"loadCfg,omitempty"`
	// SlowStartCfg turns on client-side slow start for pods that start serving
	// from an empty block cache (docs/design/onyxdb-pod-slow-start.md). The
	// data loader reads it when such a pod goes live and publishes it in the
	// pod's registration. Nil means off: clients give a new pod a full share.
	SlowStartCfg *SlowStartConfig `json:"slowStartCfg,omitempty"`
}

// SlowStartConfig sizes the client-side ramp of a pod that has just started
// serving from an empty block cache.
type SlowStartConfig struct {
	// WindowSec is how long clients take to ramp such a pod from 1% of a full
	// share to a full share. 0 means off. Must stay below the data-plane
	// chart's minReadySeconds, or the old pod goes away mid-ramp.
	WindowSec int `json:"windowSec,omitempty"`
}

// LoadConfig holds per-load read-server options
// (docs/design/onyxdb-warm-ahead-prewarm.md).
type LoadConfig struct {
	// PreloadIndexFilter makes the read server read every SST's index and
	// filter block once, single-threaded, right after opening a version that
	// is about to serve (the active version, or a newer one). Rollback
	// pre-warm loads never preload. Nil means the read server's own
	// --preload-index-filter default (off).
	PreloadIndexFilter *bool `json:"preloadIndexFilter,omitempty"`
	// DeferAtCap makes the data loader park a newer version while an older one
	// is resident on the read server and awaiting promotion, instead of
	// loading a third version beside it (docs/design/onyxdb-version-memory-guard.md
	// §4). Nil means on; false is the kill switch.
	DeferAtCap *bool `json:"deferAtCap,omitempty"`
	// AdmitOverCap makes the read server admit a load past its resident-version
	// cap without checking that it fits in memory — the behaviour before the
	// version memory guard. Nil or false means the check runs. An escape hatch
	// for an estimate that refuses loads which would fit; setting it brings the
	// OOM risk back.
	AdmitOverCap *bool `json:"admitOverCap,omitempty"`
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
	// Warm configures warm-ahead pre-warming of each slice before it moves
	// (docs/design/onyxdb-warm-ahead-prewarm.md). Nil or Enabled=false keeps today's
	// timed ramp exactly. Read servers that predate warm-ahead refuse the warm
	// IPC commands, and the data loader then falls back to the timed ramp.
	Warm *WarmConfig `json:"warm,omitempty"`
}

// WarmConfig tunes the warm-ahead gated ramp. Zero values mean "use the
// default" (see the dataloader's resolveWarm), except Enabled, which is off
// unless set.
type WarmConfig struct {
	// Enabled turns on warm-ahead and the gated ramp. Default false.
	Enabled bool `json:"enabled"`
	// LookaheadPct caps how much of the keyspace one warmed slice covers;
	// configured steps wider than this are split into sub-steps. Default 5.
	LookaheadPct int `json:"lookaheadPct,omitempty"`
	// MaxKeysPerSec is the per-pod warm budget (every warm miss is a disk
	// read). Default 15000.
	MaxKeysPerSec int `json:"maxKeysPerSec,omitempty"`
	// TargetHitRatio is the warm-read block-cache hit ratio at which a slice
	// counts as warm and moves. Default 0.95.
	TargetHitRatio float64 `json:"targetHitRatio,omitempty"`
	// MaxDwellSec bounds the wait for TargetHitRatio; the slice moves anyway
	// when it runs out. Default 180.
	MaxDwellSec int `json:"maxDwellSec,omitempty"`
	// BrakeP99Factor holds a step while the read server's batch p99 is above
	// this multiple of its pre-ramp baseline. Default 3.
	BrakeP99Factor float64 `json:"brakeP99Factor,omitempty"`
	// BrakeMissFactor holds a step while the serving block-read rate is above
	// this multiple of its pre-ramp baseline. Default 2.
	BrakeMissFactor float64 `json:"brakeMissFactor,omitempty"`
	// MaxHoldSec bounds one brake hold; past it the step proceeds (logged),
	// so a pod whose traffic simply grew cannot stall its ramp forever.
	// Default 600.
	MaxHoldSec int `json:"maxHoldSec,omitempty"`
	// PauseEngineConcurrency and PauseP99Ms override the read server's
	// --warm-pause-engine-concurrency / --warm-pause-p99-ms for this ramp:
	// the warmer pauses while serving is above either. Sent with every warm
	// window; zero means the read server's flag default (32 / 25 ms).
	PauseEngineConcurrency float64 `json:"pauseEngineConcurrency,omitempty"`
	PauseP99Ms             float64 `json:"pauseP99Ms,omitempty"`
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
	// KeepAliveTimeoutMs is accepted and passed to the Go SDK's pool, but the
	// SDK does not apply it to the socket today. Only KeepAliveIntervalMs is set
	// (via SetKeepAlivePeriod). A dead read server is still detected: the next
	// request on the connection fails at its RequestTimeoutMs deadline, and idle
	// connections are closed after IdleTimeoutMs. Default: 5000 (no effect).
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
	// ServingSince is when this pod started serving from an empty block cache
	// (unix milliseconds): its first activation after a read-server start or a
	// repair. Clients ramp the pod's share of its shard up from this instant
	// over SlowStartSec. 0 means no ramp: clients give the pod a full share.
	ServingSince int64 `json:"servingSince,omitempty"`
	// SlowStartSec is the ramp window the loader read from the store's
	// SlowStartCfg when the pod went live. 0 means no ramp.
	SlowStartSec int `json:"slowStartSec,omitempty"`
}

// DefaultReadServerPort is the read server's TCP port when a registration
// omits Port (one read server per pod IP).
const DefaultReadServerPort = 9091

// Addr is the address clients reach this pod's read server on, the entry
// placement puts in the shard assignment: PodIP and Port, or
// DefaultReadServerPort when Port is unset.
func (p PodData) Addr() string {
	port := p.Port
	if port == 0 {
		port = DefaultReadServerPort
	}
	return fmt.Sprintf("%s:%d", p.PodIP, port)
}
