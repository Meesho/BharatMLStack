// Package etcdstate manages all etcd reads, writes, and CAS operations
// for the OnyxDB control plane.
package etcdstate

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// Sentinel errors returned by StateClient operations.
var (
	ErrNotFound      = fmt.Errorf("not found")
	ErrAlreadyExists = fmt.Errorf("already exists")
	ErrCASConflict   = fmt.Errorf("CAS conflict: topology version changed concurrently")
	ErrNoRollback    = fmt.Errorf("no rollback version available")
)

// StoreState holds the full persisted state for a store.
type StoreState struct {
	Config          model.StoreConfig     `json:"config"`
	ActiveVersion   string                `json:"activeVersion"`
	RollbackVersion string                `json:"rollbackVersion"`
	TopologyVersion int64                 `json:"topologyVersion"`
	Dataflow        *model.DataflowConfig `json:"dataflow,omitempty"`
	ClientConfig    *model.ClientConfig   `json:"clientConfig,omitempty"`
}

// VersionInfo describes the status and pod-level state for a single version
// in the topology response. Derived at read time from pod registrations.
type VersionInfo struct {
	VersionID string `json:"versionId"`
	Status    string `json:"status"`
	// WarmPods is the number of pods that have this version warm.
	WarmPods int `json:"warmPods"`
	// LoadingPods is the number of pods currently downloading this version.
	LoadingPods int `json:"loadingPods"`
	// RollingOutPods is the number of pods currently rolling out this version.
	RollingOutPods int `json:"rollingOutPods"`
	// Assignment is the shard→pod address map for this version.
	// Only populated for the active version.
	Assignment map[string][]string `json:"assignment,omitempty"`
}

// TopologyState holds the active version, its shard→pod assignment, and
// the status of all kept versions across the data plane.
type TopologyState struct {
	ActiveVersion   string `json:"activeVersion"`
	RollbackVersion string `json:"rollbackVersion,omitempty"`
	TopologyVersion int64  `json:"topologyVersion"`
	// Assignment for the active version (maintained for backward compatibility).
	Assignment map[string][]string `json:"assignment"`
	// Versions lists every non-retired version with its per-pod rollout state.
	Versions []VersionInfo `json:"versions,omitempty"`
	// Pods lists every registered pod's current state.
	Pods map[string]PodState `json:"pods,omitempty"`
}

// PodState is a snapshot of a single pod's data-plane state, derived from
// its etcd registration. Included in the topology response for observability.
type PodState struct {
	PodIP          string `json:"podIP"`
	ServingVersion string `json:"servingVersion"`
	LoadingVersion string `json:"loadingVersion,omitempty"`
	RolloutVersion string `json:"rolloutVersion,omitempty"`
	RolloutPct     int    `json:"rolloutPct,omitempty"`
	WarmVersions   []string `json:"warmVersions"`
}

// StateClient is the interface all control plane handlers depend on.
type StateClient interface {
	CreateStore(ctx context.Context, cfg model.StoreConfig) error
	GetStore(ctx context.Context, tenant, store string) (*StoreState, error)

	PublishVersion(ctx context.Context, tenant, store, vID string, meta model.VersionMeta) error
	GetVersionMeta(ctx context.Context, tenant, store, vID string) (*model.VersionMeta, error)
	PromoteVersion(ctx context.Context, tenant, store, vID string, assignment map[string][]string) error
	RollbackStore(ctx context.Context, tenant, store string) (string, error)
	RetireVersion(ctx context.Context, tenant, store, vID string) error

	PutDataflow(ctx context.Context, tenant, store string, cfg model.DataflowConfig) error
	GetDataflow(ctx context.Context, tenant, store string) (*model.DataflowConfig, error)

	GetClientConfig(ctx context.Context, tenant, store string) (*model.ClientConfig, error)
	SetClientConfig(ctx context.Context, tenant, store string, cfg model.ClientConfig) error

	GetTopology(ctx context.Context, tenant, store string) (*TopologyState, error)
	ListPods(ctx context.Context, tenant, store string) (map[string]model.PodData, error)
	ListVersions(ctx context.Context, tenant, store string) (map[string]*model.VersionMeta, error)

	// Health does a cheap bounded read to confirm etcd is reachable. Used by the
	// readiness probe so a pod that can't reach etcd is taken out of the endpoints.
	Health(ctx context.Context) error

	Close() error
}

// kvOps is the narrow internal interface over etcd KV operations.
// Keeping it minimal allows straightforward in-memory test doubles.
type kvOps interface {
	get(ctx context.Context, key string) (value string, modRev int64, found bool, err error)
	put(ctx context.Context, key, value string) error
	getPrefix(ctx context.Context, prefix string) (map[string]string, error)
	// atomicCreate writes pairs only if guardKey does not yet exist.
	// Returns ErrAlreadyExists if the guard key already has a value.
	atomicCreate(ctx context.Context, guardKey string, pairs map[string]string) error
	// atomicSwap writes updates atomically only when watchKey's etcd modRevision == watchRev.
	// Returns (true, nil) on success, (false, nil) on CAS failure.
	atomicSwap(ctx context.Context, watchKey string, watchRev int64, updates map[string]string) (bool, error)
}

// etcdKVOps is the live etcd implementation of kvOps.
type etcdKVOps struct {
	kv clientv3.KV
}

func (e *etcdKVOps) get(ctx context.Context, key string) (string, int64, bool, error) {
	resp, err := e.kv.Get(ctx, key)
	if err != nil {
		return "", 0, false, err
	}
	if len(resp.Kvs) == 0 {
		return "", 0, false, nil
	}
	kv := resp.Kvs[0]
	return string(kv.Value), kv.ModRevision, true, nil
}

func (e *etcdKVOps) put(ctx context.Context, key, value string) error {
	_, err := e.kv.Put(ctx, key, value)
	return err
}

func (e *etcdKVOps) getPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	resp, err := e.kv.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = string(kv.Value)
	}
	return result, nil
}

func (e *etcdKVOps) atomicCreate(ctx context.Context, guardKey string, pairs map[string]string) error {
	ops := make([]clientv3.Op, 0, len(pairs))
	for k, v := range pairs {
		ops = append(ops, clientv3.OpPut(k, v))
	}
	resp, err := e.kv.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(guardKey), "=", 0)).
		Then(ops...).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrAlreadyExists
	}
	return nil
}

func (e *etcdKVOps) atomicSwap(ctx context.Context, watchKey string, watchRev int64, updates map[string]string) (bool, error) {
	ops := make([]clientv3.Op, 0, len(updates))
	for k, v := range updates {
		ops = append(ops, clientv3.OpPut(k, v))
	}
	resp, err := e.kv.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(watchKey), "=", watchRev)).
		Then(ops...).
		Commit()
	if err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

// EtcdStateClient is the real etcd-backed implementation of StateClient.
type EtcdStateClient struct {
	ops    kvOps
	client *clientv3.Client
}

// NewEtcdStateClient dials the given etcd endpoints and returns a StateClient.
func NewEtcdStateClient(endpoints []string) (*EtcdStateClient, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to etcd at %v: %w", endpoints, err)
	}
	return &EtcdStateClient{
		ops:    &etcdKVOps{kv: cli.KV},
		client: cli,
	}, nil
}

// Health does a cheap bounded read on a probe key to confirm etcd is reachable.
// A "not found" result is healthy (etcd answered); only a transport/timeout error
// means unreachable.
func (c *EtcdStateClient) Health(ctx context.Context) error {
	_, _, _, err := c.ops.get(ctx, model.AppPrefix+"/_health")
	return err
}

// Close releases the underlying etcd connection.
func (c *EtcdStateClient) Close() error {
	return c.client.Close()
}
