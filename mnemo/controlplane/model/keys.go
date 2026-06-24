// Package schema defines the etcd key layout and shared data types for mNemo.
//
// All store keys live under /config/mnemo/... and all pod registration keys
// live under /config/mnemo-cluster-manager/... . Use the type-safe helpers
// below so key typos are caught at compile time.
package model

import "fmt"

// AppPrefix is the top-level etcd prefix for all mNemo store state.
const AppPrefix = "/config/mnemo"

// ClusterManagerPrefix is the top-level prefix for ephemeral pod registrations.
const ClusterManagerPrefix = "/config/mnemo-cluster-manager"

// VersionStatus represents the lifecycle state of a version.
type VersionStatus string

const (
	StatusAllocated VersionStatus = "ALLOCATED"
	StatusIngesting VersionStatus = "INGESTING"
	StatusReady     VersionStatus = "READY"
	StatusActive    VersionStatus = "ACTIVE"
	StatusRetiring  VersionStatus = "RETIRING"
)

// ── Store-level keys ───────────────────────────────────────────────────────────

// StorePrefix returns the base etcd path for a tenant's store.
//
//	/config/mnemo/tenants/{tenant}/stores/{store}
func StorePrefix(tenant, store string) string {
	return fmt.Sprintf("%s/tenants/%s/stores/%s", AppPrefix, tenant, store)
}

// EntityKeyPath returns the key that holds the entity key schema string.
func EntityKeyPath(tenant, store string) string {
	return StorePrefix(tenant, store) + "/entityKey"
}

// ShardCountPath returns the key that holds the shard count S.
func ShardCountPath(tenant, store string) string {
	return StorePrefix(tenant, store) + "/shardCount"
}

// ActiveVersionPath returns the key that holds the currently active version ID.
func ActiveVersionPath(tenant, store string) string {
	return StorePrefix(tenant, store) + "/activeVersion"
}

// RollbackVersionPath returns the key that holds the rollback (previous active) version ID.
func RollbackVersionPath(tenant, store string) string {
	return StorePrefix(tenant, store) + "/rollbackVersion"
}

// TopologyVersionPath returns the monotonic CAS-guarded topology counter key.
func TopologyVersionPath(tenant, store string) string {
	return StorePrefix(tenant, store) + "/topologyVersion"
}

// ── Version-level keys ─────────────────────────────────────────────────────────

// VersionPrefix returns the base etcd path for a specific version's metadata.
//
//	/config/mnemo/tenants/{tenant}/stores/{store}/versions/{versionID}
func VersionPrefix(tenant, store, versionID string) string {
	return fmt.Sprintf("%s/versions/%s", StorePrefix(tenant, store), versionID)
}

// VersionsWatchPrefix returns the prefix to watch for all version changes on a store.
func VersionsWatchPrefix(tenant, store string) string {
	return StorePrefix(tenant, store) + "/versions/"
}

// ── Dataflow config key ────────────────────────────────────────────────────

// DataflowPath returns the etcd key for a store's dataflow pipeline config.
//
//	/config/mnemo/tenants/{tenant}/stores/{store}/dataflow
func DataflowPath(tenant, store string) string {
	return StorePrefix(tenant, store) + "/dataflow"
}

// ── Pod registration keys (ephemeral, lease-bound) ────────────────────────────

// PodDataPath returns the ephemeral etcd key for a single pod's registration.
//
//	/config/mnemo-cluster-manager/{tenant}/{store}/{podID}
func PodDataPath(tenant, store, podID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", ClusterManagerPrefix, tenant, store, podID)
}

// PodWatchPrefix returns the prefix to watch all pod registrations for a store.
func PodWatchPrefix(tenant, store string) string {
	return fmt.Sprintf("%s/%s/%s/", ClusterManagerPrefix, tenant, store)
}
