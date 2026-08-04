package etcdstate

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/placement"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
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

// UpdateAssignment refreshes the VersionMeta.Assignment for an active version
// from current pod registrations. This keeps the etcd-stored assignment in sync
// with reality after pod rescheduling (new IPs). Uses CAS on topologyVersion
// to avoid stomping a concurrent promote/rollback.
// Returns (changed, error) — changed is true when the assignment actually differed.
func (c *EtcdStateClient) UpdateAssignment(ctx context.Context, tenant, store, vID string, assignment map[string][]string) (bool, error) {
	meta, err := c.GetVersionMeta(ctx, tenant, store, vID)
	if err != nil {
		return false, err
	}

	// Check if the assignment actually changed.
	if assignmentsEqual(meta.Assignment, assignment) {
		return false, nil
	}

	tvVal, tvRev, found, err := c.ops.get(ctx, model.TopologyVersionPath(tenant, store))
	if err != nil {
		return false, err
	}
	if !found {
		return false, ErrNotFound
	}
	tvInt, _ := strconv.ParseInt(tvVal, 10, 64)

	meta.Assignment = assignment
	b, _ := json.Marshal(meta)

	newTV := strconv.FormatInt(tvInt+1, 10)
	updates := map[string]string{
		model.TopologyVersionPath(tenant, store): newTV,
		model.VersionPrefix(tenant, store, vID):  string(b),
	}

	succeeded, err := c.ops.atomicSwap(ctx, model.TopologyVersionPath(tenant, store), tvRev, updates)
	if err != nil {
		return false, err
	}
	if !succeeded {
		return false, ErrCASConflict
	}
	return true, nil
}

// assignmentsEqual returns true if two assignment maps have the same content.
func assignmentsEqual(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || len(av) != len(bv) {
			return false
		}
		// Sort copies to compare without mutating originals.
		as := make([]string, len(av))
		bs := make([]string, len(bv))
		copy(as, av)
		copy(bs, bv)
		sort.Strings(as)
		sort.Strings(bs)
		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}
	}
	return true
}

// GetTopology returns the active version, its shard→pod assignment, and a full
// snapshot of every kept version's data-plane status (warm/loading/rolling-out
// pod counts) derived from current pod registrations. The assignment always
// reflects which pods are actually alive and warm right now — not the static
// snapshot captured at promote time.
func (c *EtcdStateClient) GetTopology(ctx context.Context, tenant, store string) (*TopologyState, error) {
	activeVer, _, found, err := c.ops.get(ctx, model.ActiveVersionPath(tenant, store))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	rollbackVer, _, _, _ := c.ops.get(ctx, model.RollbackVersionPath(tenant, store))
	tvVal, _, _, _ := c.ops.get(ctx, model.TopologyVersionPath(tenant, store))
	tvInt, _ := strconv.ParseInt(tvVal, 10, 64)

	topo := &TopologyState{
		ActiveVersion:   activeVer,
		RollbackVersion: rollbackVer,
		TopologyVersion: tvInt,
		Assignment:      map[string][]string{},
	}
	if activeVer == "" {
		return topo, nil
	}

	// Fetch all version metadata and pod registrations.
	versions, _ := c.ListVersions(ctx, tenant, store)
	pods, err := c.ListPods(ctx, tenant, store)

	var shardCount int
	if meta, ok := versions[activeVer]; ok && meta != nil {
		shardCount = meta.ShardCount
	}

	if err != nil {
		// Fall back to static assignment if pod listing fails.
		if meta, ok := versions[activeVer]; ok && meta != nil && meta.Assignment != nil {
			topo.Assignment = meta.Assignment
		}
		return topo, nil
	}

	// Derive live assignment for the active version.
	if shardCount > 0 {
		topo.Assignment = placement.DeriveAssignment(shardCount, pods, activeVer)
	}

	// Build per-pod state snapshot.
	topo.Pods = make(map[string]PodState, len(pods))
	for podID, pd := range pods {
		topo.Pods[podID] = PodState{
			PodIP:          pd.PodIP,
			ServingVersion: pd.ServingVersion,
			LoadingVersion: pd.LoadingVersion,
			RolloutVersion: pd.RolloutVersion,
			RolloutPct:     pd.RolloutPct,
			WarmVersions:   pd.WarmVersions,
		}
	}

	// Build per-version info by scanning pod registrations.
	versionInfoMap := make(map[string]*VersionInfo)
	for vID, meta := range versions {
		if meta == nil || meta.Status == model.StatusRetiring {
			continue
		}
		vi := &VersionInfo{
			VersionID: vID,
			Status:    string(meta.Status),
		}
		versionInfoMap[vID] = vi
	}

	for _, pd := range pods {
		for _, wv := range pd.WarmVersions {
			if vi, ok := versionInfoMap[wv]; ok {
				vi.WarmPods++
			}
		}
		if pd.LoadingVersion != "" {
			if vi, ok := versionInfoMap[pd.LoadingVersion]; ok {
				vi.LoadingPods++
			}
		}
		if pd.RolloutVersion != "" {
			if vi, ok := versionInfoMap[pd.RolloutVersion]; ok {
				vi.RollingOutPods++
			}
		}
	}

	// Attach active version assignment to its VersionInfo.
	if vi, ok := versionInfoMap[activeVer]; ok {
		vi.Assignment = topo.Assignment
	}

	// Collect into sorted slice (newest first).
	topo.Versions = make([]VersionInfo, 0, len(versionInfoMap))
	for _, vi := range versionInfoMap {
		topo.Versions = append(topo.Versions, *vi)
	}
	sort.Slice(topo.Versions, func(i, j int) bool {
		return topo.Versions[i].VersionID > topo.Versions[j].VersionID
	})

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
