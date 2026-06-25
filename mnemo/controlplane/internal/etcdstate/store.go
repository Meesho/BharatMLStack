package etcdstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// CreateStore writes the initial store configuration atomically.
// Returns ErrAlreadyExists if a store with the same tenant/name already exists.
func (c *EtcdStateClient) CreateStore(ctx context.Context, cfg model.StoreConfig) error {
	pairs := map[string]string{
		model.EntityKeyPath(cfg.Tenant, cfg.Store):       cfg.EntityKey,
		model.ShardCountPath(cfg.Tenant, cfg.Store):      strconv.Itoa(cfg.ShardCount),
		model.TopologyVersionPath(cfg.Tenant, cfg.Store): "1",
		model.ActiveVersionPath(cfg.Tenant, cfg.Store):   "",
		model.RollbackVersionPath(cfg.Tenant, cfg.Store): "",
	}
	return c.ops.atomicCreate(ctx, model.EntityKeyPath(cfg.Tenant, cfg.Store), pairs)
}

// GetStore reads the full state of a store.
// Returns ErrNotFound if the store has not been created.
func (c *EtcdStateClient) GetStore(ctx context.Context, tenant, store string) (*StoreState, error) {
	entityKey, _, found, err := c.ops.get(ctx, model.EntityKeyPath(tenant, store))
	if err != nil {
		return nil, fmt.Errorf("getting store %s/%s: %w", tenant, store, err)
	}
	if !found {
		return nil, ErrNotFound
	}

	shardCountStr, _, _, _ := c.ops.get(ctx, model.ShardCountPath(tenant, store))
	activeVersion, _, _, _ := c.ops.get(ctx, model.ActiveVersionPath(tenant, store))
	rollbackVersion, _, _, _ := c.ops.get(ctx, model.RollbackVersionPath(tenant, store))
	tvStr, _, _, _ := c.ops.get(ctx, model.TopologyVersionPath(tenant, store))

	shardCount, _ := strconv.Atoi(shardCountStr)
	tv, _ := strconv.ParseInt(tvStr, 10, 64)

	state := &StoreState{
		Config: model.StoreConfig{
			Tenant:     tenant,
			Store:      store,
			EntityKey:  entityKey,
			ShardCount: shardCount,
		},
		ActiveVersion:   activeVersion,
		RollbackVersion: rollbackVersion,
		TopologyVersion: tv,
	}

	// Best-effort: attach dataflow config if it exists.
	if df, err := c.GetDataflow(ctx, tenant, store); err == nil {
		state.Dataflow = df
	}

	// Best-effort: attach client config if it exists.
	if cc, err := c.GetClientConfig(ctx, tenant, store); err == nil {
		state.ClientConfig = cc
	}

	return state, nil
}

// PublishVersion creates a new version entry in etcd with the given metadata.
// Returns ErrNotFound if the store does not exist, ErrAlreadyExists if the version already exists.
func (c *EtcdStateClient) PublishVersion(ctx context.Context, tenant, store, vID string, meta model.VersionMeta) error {
	_, _, found, err := c.ops.get(ctx, model.EntityKeyPath(tenant, store))
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}

	_, _, vFound, err := c.ops.get(ctx, model.VersionPrefix(tenant, store, vID))
	if err != nil {
		return err
	}
	if vFound {
		return ErrAlreadyExists
	}

	b, _ := json.Marshal(meta) // model.VersionMeta is always JSON-serializable
	return c.ops.put(ctx, model.VersionPrefix(tenant, store, vID), string(b))
}

// GetVersionMeta reads the metadata for a specific version.
// Returns ErrNotFound if the version does not exist.
func (c *EtcdStateClient) GetVersionMeta(ctx context.Context, tenant, store, vID string) (*model.VersionMeta, error) {
	val, _, found, err := c.ops.get(ctx, model.VersionPrefix(tenant, store, vID))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	var meta model.VersionMeta
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		return nil, fmt.Errorf("parsing version meta for %s: %w", vID, err)
	}
	return &meta, nil
}

// PromoteVersion atomically sets the active version using CAS on the topology version key.
// The assignment map is stored in the version metadata.
// Returns ErrCASConflict when another promote raced this one.
func (c *EtcdStateClient) PromoteVersion(ctx context.Context, tenant, store, vID string, assignment map[string][]string) error {
	tvVal, tvRev, found, err := c.ops.get(ctx, model.TopologyVersionPath(tenant, store))
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	tvInt, _ := strconv.ParseInt(tvVal, 10, 64)

	currentActive, _, _, err := c.ops.get(ctx, model.ActiveVersionPath(tenant, store))
	if err != nil {
		return err
	}

	meta, err := c.GetVersionMeta(ctx, tenant, store, vID)
	if err != nil {
		return err
	}
	meta.Status = model.StatusActive
	meta.Assignment = assignment

	b, _ := json.Marshal(meta) // model.VersionMeta is always JSON-serializable

	newTV := strconv.FormatInt(tvInt+1, 10)
	updates := map[string]string{
		model.TopologyVersionPath(tenant, store): newTV,
		model.ActiveVersionPath(tenant, store):   vID,
		model.RollbackVersionPath(tenant, store): currentActive,
		model.VersionPrefix(tenant, store, vID):  string(b),
	}

	succeeded, err := c.ops.atomicSwap(ctx, model.TopologyVersionPath(tenant, store), tvRev, updates)
	if err != nil {
		return err
	}
	if !succeeded {
		return ErrCASConflict
	}
	return nil
}

// RollbackStore atomically swaps the active version back to the rollback version.
// Returns ErrNoRollback when there is no rollback version set.
// Returns ErrCASConflict on concurrent topology modification.
func (c *EtcdStateClient) RollbackStore(ctx context.Context, tenant, store string) (string, error) {
	rollbackVer, _, found, err := c.ops.get(ctx, model.RollbackVersionPath(tenant, store))
	if err != nil {
		return "", err
	}
	if !found || rollbackVer == "" {
		return "", ErrNoRollback
	}

	tvVal, tvRev, tvFound, err := c.ops.get(ctx, model.TopologyVersionPath(tenant, store))
	if err != nil {
		return "", err
	}
	if !tvFound {
		return "", ErrNotFound
	}
	tvInt, _ := strconv.ParseInt(tvVal, 10, 64)

	currentActive, _, _, _ := c.ops.get(ctx, model.ActiveVersionPath(tenant, store))

	newTV := strconv.FormatInt(tvInt+1, 10)
	updates := map[string]string{
		model.TopologyVersionPath(tenant, store): newTV,
		model.ActiveVersionPath(tenant, store):   rollbackVer,
		model.RollbackVersionPath(tenant, store): currentActive,
	}

	succeeded, err := c.ops.atomicSwap(ctx, model.TopologyVersionPath(tenant, store), tvRev, updates)
	if err != nil {
		return "", err
	}
	if !succeeded {
		return "", ErrCASConflict
	}
	return rollbackVer, nil
}

// RetireVersion marks a version as RETIRING so data loaders will drop its files.
func (c *EtcdStateClient) RetireVersion(ctx context.Context, tenant, store, vID string) error {
	meta, err := c.GetVersionMeta(ctx, tenant, store, vID)
	if err != nil {
		return err
	}
	meta.Status = model.StatusRetiring
	b, _ := json.Marshal(meta) // model.VersionMeta is always JSON-serializable
	return c.ops.put(ctx, model.VersionPrefix(tenant, store, vID), string(b))
}

// GetTopology returns the active version and its shard→pod assignment.
func (c *EtcdStateClient) GetTopology(ctx context.Context, tenant, store string) (*TopologyState, error) {
	activeVer, _, found, err := c.ops.get(ctx, model.ActiveVersionPath(tenant, store))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	tvVal, _, _, _ := c.ops.get(ctx, model.TopologyVersionPath(tenant, store))
	tvInt, _ := strconv.ParseInt(tvVal, 10, 64)

	topo := &TopologyState{
		ActiveVersion:   activeVer,
		TopologyVersion: tvInt,
		Assignment:      map[string][]string{},
	}
	if activeVer == "" {
		return topo, nil
	}

	meta, err := c.GetVersionMeta(ctx, tenant, store, activeVer)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return topo, nil
		}
		return nil, err
	}
	if meta.Assignment != nil {
		topo.Assignment = meta.Assignment
	}
	return topo, nil
}

// PutDataflow writes or updates the dataflow config for a store.
// Returns ErrNotFound if the store does not exist.
func (c *EtcdStateClient) PutDataflow(ctx context.Context, tenant, store string, cfg model.DataflowConfig) error {
	_, _, found, err := c.ops.get(ctx, model.EntityKeyPath(tenant, store))
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	b, _ := json.Marshal(cfg)
	return c.ops.put(ctx, model.DataflowPath(tenant, store), string(b))
}

// GetDataflow reads the dataflow config for a store.
// Returns ErrNotFound if the store or its dataflow config does not exist.
func (c *EtcdStateClient) GetDataflow(ctx context.Context, tenant, store string) (*model.DataflowConfig, error) {
	val, _, found, err := c.ops.get(ctx, model.DataflowPath(tenant, store))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	var cfg model.DataflowConfig
	if err := json.Unmarshal([]byte(val), &cfg); err != nil {
		return nil, fmt.Errorf("parsing dataflow config for %s/%s: %w", tenant, store, err)
	}
	return &cfg, nil
}

// GetClientConfig reads the client config for a store.
// Returns ErrNotFound if the store or its client config does not exist.
func (c *EtcdStateClient) GetClientConfig(ctx context.Context, tenant, store string) (*model.ClientConfig, error) {
	val, _, found, err := c.ops.get(ctx, model.ClientConfigPath(tenant, store))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	var cfg model.ClientConfig
	if err := json.Unmarshal([]byte(val), &cfg); err != nil {
		return nil, fmt.Errorf("parsing client config for %s/%s: %w", tenant, store, err)
	}
	return &cfg, nil
}

// SetClientConfig writes the client config for a store.
// Returns ErrNotFound if the store does not exist.
func (c *EtcdStateClient) SetClientConfig(ctx context.Context, tenant, store string, cfg model.ClientConfig) error {
	_, _, found, err := c.ops.get(ctx, model.EntityKeyPath(tenant, store))
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	b, _ := json.Marshal(cfg)
	return c.ops.put(ctx, model.ClientConfigPath(tenant, store), string(b))
}

// ListPods returns all currently registered pods for a store, keyed by pod ID.
func (c *EtcdStateClient) ListPods(ctx context.Context, tenant, store string) (map[string]model.PodData, error) {
	prefix := model.PodWatchPrefix(tenant, store)
	raw, err := c.ops.getPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	pods := make(map[string]model.PodData, len(raw))
	for key, val := range raw {
		podID := strings.TrimPrefix(key, prefix)
		if podID == "" {
			continue
		}
		var data model.PodData
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			continue // skip corrupt registrations
		}
		pods[podID] = data
	}
	return pods, nil
}

// StoreRef identifies a store by tenant + name (the keys needed to address it).
type StoreRef struct {
	Tenant string
	Store  string
}

// ListStores enumerates every store across all tenants. It scans the tenants
// prefix for the per-store "/entityKey" marker (exactly one per store) and parses
// the tenant/store out of each key path.
func (c *EtcdStateClient) ListStores(ctx context.Context) ([]StoreRef, error) {
	prefix := model.AppPrefix + "/tenants/"
	raw, err := c.ops.getPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	var refs []StoreRef
	for key := range raw {
		// key: /config/mnemo/tenants/{tenant}/stores/{store}/entityKey
		if !strings.HasSuffix(key, "/entityKey") {
			continue
		}
		mid := strings.TrimSuffix(key, "/entityKey")
		mid = strings.TrimPrefix(mid, prefix) // {tenant}/stores/{store}
		tenant, store, ok := strings.Cut(mid, "/stores/")
		if !ok || tenant == "" || store == "" {
			continue
		}
		refs = append(refs, StoreRef{Tenant: tenant, Store: store})
	}
	return refs, nil
}

// ListVersions returns every version's metadata for a store, keyed by version ID.
func (c *EtcdStateClient) ListVersions(ctx context.Context, tenant, store string) (map[string]*model.VersionMeta, error) {
	prefix := model.VersionsWatchPrefix(tenant, store)
	raw, err := c.ops.getPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	versions := make(map[string]*model.VersionMeta, len(raw))
	for key, val := range raw {
		vID := strings.TrimPrefix(key, prefix)
		if vID == "" {
			continue
		}
		var meta model.VersionMeta
		if err := json.Unmarshal([]byte(val), &meta); err != nil {
			continue // skip corrupt version entries
		}
		versions[vID] = &meta
	}
	return versions, nil
}
