package etcdstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// ── memKVOps: in-memory kvOps for fast domain-logic tests ────────────────────

type memKVOps struct {
	mu      sync.Mutex
	data    map[string]string
	revs    map[string]int64
	nextRev int64
}

func newMemKVOps() *memKVOps {
	return &memKVOps{
		data:    make(map[string]string),
		revs:    make(map[string]int64),
		nextRev: 1,
	}
}

func (m *memKVOps) get(_ context.Context, key string) (string, int64, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", 0, false, nil
	}
	return v, m.revs[key], true, nil
}

func (m *memKVOps) put(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextRev++
	m.data[key] = value
	m.revs[key] = m.nextRev
	return nil
}

func (m *memKVOps) getPrefix(_ context.Context, prefix string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]string)
	for k, v := range m.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result[k] = v
		}
	}
	return result, nil
}

func (m *memKVOps) atomicCreate(_ context.Context, guardKey string, pairs map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[guardKey]; exists {
		return ErrAlreadyExists
	}
	m.nextRev++
	for k, v := range pairs {
		m.data[k] = v
		m.revs[k] = m.nextRev
	}
	return nil
}

func (m *memKVOps) atomicSwap(_ context.Context, watchKey string, watchRev int64, updates map[string]string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.revs[watchKey] != watchRev {
		return false, nil
	}
	m.nextRev++
	for k, v := range updates {
		m.data[k] = v
		m.revs[k] = m.nextRev
	}
	return true, nil
}

// ── errKVOps: always returns errors ──────────────────────────────────────────

type errKVOps struct{ err error }

func (e *errKVOps) get(_ context.Context, _ string) (string, int64, bool, error) {
	return "", 0, false, e.err
}
func (e *errKVOps) put(_ context.Context, _, _ string) error { return e.err }
func (e *errKVOps) getPrefix(_ context.Context, _ string) (map[string]string, error) {
	return nil, e.err
}
func (e *errKVOps) atomicCreate(_ context.Context, _ string, _ map[string]string) error { return e.err }
func (e *errKVOps) atomicSwap(_ context.Context, _ string, _ int64, _ map[string]string) (bool, error) {
	return false, e.err
}

// ── casFailOps: atomicSwap always returns (false, nil) ───────────────────────

type casFailOps struct{ *memKVOps }

func (c *casFailOps) atomicSwap(_ context.Context, _ string, _ int64, _ map[string]string) (bool, error) {
	return false, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestStateClient(ops kvOps) *EtcdStateClient {
	return &EtcdStateClient{ops: ops}
}

func defaultCfg() model.StoreConfig {
	return model.StoreConfig{Tenant: "fs", Store: "features", EntityKey: "catalog_id", ShardCount: 3}
}

func defaultMeta() model.VersionMeta {
	return model.VersionMeta{Date: "20260603", Run: "001", ShardCount: 3, Status: model.StatusReady}
}

// ── CreateStore ───────────────────────────────────────────────────────────────

func TestCreateStore_Success(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
}

func TestCreateStore_AlreadyExists(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	err := sc.CreateStore(context.Background(), defaultCfg())
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestCreateStore_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("etcd down")})
	assert.Error(t, sc.CreateStore(context.Background(), defaultCfg()))
}

// ── GetStore ──────────────────────────────────────────────────────────────────

func TestGetStore_Success(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	cfg := defaultCfg()
	require.NoError(t, sc.CreateStore(context.Background(), cfg))

	state, err := sc.GetStore(context.Background(), cfg.Tenant, cfg.Store)
	require.NoError(t, err)
	assert.Equal(t, cfg.EntityKey, state.Config.EntityKey)
	assert.Equal(t, cfg.ShardCount, state.Config.ShardCount)
	assert.Equal(t, int64(1), state.TopologyVersion)
	assert.Empty(t, state.ActiveVersion)
}

func TestGetStore_NotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	_, err := sc.GetStore(context.Background(), "t", "s")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetStore_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.GetStore(context.Background(), "t", "s")
	assert.Error(t, err)
}

// ── PublishVersion ────────────────────────────────────────────────────────────

func TestPublishVersion_Success(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))

	meta := defaultMeta()
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", meta))
}

func TestPublishVersion_StoreNotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	err := sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPublishVersion_AlreadyExists(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	err := sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta())
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestPublishVersion_BackendError_OnStoreCheck(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("io error")})
	err := sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta())
	assert.Error(t, err)
}

// ── GetVersionMeta ────────────────────────────────────────────────────────────

func TestGetVersionMeta_Success(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	meta, err := sc.GetVersionMeta(context.Background(), "fs", "features", "v1")
	require.NoError(t, err)
	assert.Equal(t, model.StatusReady, meta.Status)
	assert.Equal(t, "20260603", meta.Date)
}

func TestGetVersionMeta_NotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	_, err := sc.GetVersionMeta(context.Background(), "fs", "features", "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetVersionMeta_CorruptJSON(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	// Manually inject corrupt JSON
	_ = mem.put(context.Background(), model.VersionPrefix("fs", "features", "bad"), "not-json")
	_, err := sc.GetVersionMeta(context.Background(), "fs", "features", "bad")
	assert.Error(t, err)
}

func TestGetVersionMeta_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.GetVersionMeta(context.Background(), "fs", "features", "v1")
	assert.Error(t, err)
}

// ── PromoteVersion ────────────────────────────────────────────────────────────

func TestPromoteVersion_Success(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	cfg := defaultCfg()
	require.NoError(t, sc.CreateStore(context.Background(), cfg))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	assignment := map[string][]string{"0": {"10.0.0.1:9091"}}
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v1", assignment))

	state, err := sc.GetStore(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "v1", state.ActiveVersion)
	assert.Equal(t, int64(2), state.TopologyVersion)
}

func TestPromoteVersion_StoreNotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	err := sc.PromoteVersion(context.Background(), "fs", "features", "v1", nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPromoteVersion_VersionNotFound(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	// Don't publish any version
	err := sc.PromoteVersion(context.Background(), "fs", "features", "v1", nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPromoteVersion_CASConflict(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	// Use a casFailOps wrapper so atomicSwap always returns false
	sc2 := newTestStateClient(&casFailOps{mem})
	err := sc2.PromoteVersion(context.Background(), "fs", "features", "v1", nil)
	assert.ErrorIs(t, err, ErrCASConflict)
}

func TestPromoteVersion_BackendErrorOnTopologyGet(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("io error")})
	err := sc.PromoteVersion(context.Background(), "fs", "features", "v1", nil)
	assert.Error(t, err)
}

// ── RollbackStore ─────────────────────────────────────────────────────────────

func TestRollbackStore_Success(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v1", nil))

	// Promote a second version so rollback returns to v1
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v2",
		model.VersionMeta{Date: "20260604", Run: "001", ShardCount: 3, Status: model.StatusReady}))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v2", nil))

	newActive, err := sc.RollbackStore(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "v1", newActive)
}

func TestRollbackStore_NoRollbackVersion(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))

	_, err := sc.RollbackStore(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrNoRollback)
}

func TestRollbackStore_StoreNotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	_, err := sc.RollbackStore(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrNoRollback) // rollbackVersion key doesn't exist → not found
}

func TestRollbackStore_CASConflict(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v1", nil))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v2",
		model.VersionMeta{Status: model.StatusReady}))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v2", nil))

	sc2 := newTestStateClient(&casFailOps{mem})
	_, err := sc2.RollbackStore(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrCASConflict)
}

func TestRollbackStore_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.RollbackStore(context.Background(), "fs", "features")
	assert.Error(t, err)
}

// ── RetireVersion ─────────────────────────────────────────────────────────────

func TestRetireVersion_Success(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	require.NoError(t, sc.RetireVersion(context.Background(), "fs", "features", "v1"))

	meta, err := sc.GetVersionMeta(context.Background(), "fs", "features", "v1")
	require.NoError(t, err)
	assert.Equal(t, model.StatusRetiring, meta.Status)
}

func TestRetireVersion_NotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	err := sc.RetireVersion(context.Background(), "fs", "features", "v1")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── GetTopology ───────────────────────────────────────────────────────────────

func TestGetTopology_NoActiveVersion(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))

	topo, err := sc.GetTopology(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Empty(t, topo.ActiveVersion)
	assert.Empty(t, topo.Assignment)
}

func TestGetTopology_WithActiveVersion(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	// Register a live pod so DeriveAssignment finds it.
	podData := model.PodData{PodIP: "10.0.0.1", WarmVersions: []string{"v1"}}
	b, _ := json.Marshal(podData)
	_ = mem.put(context.Background(), model.PodDataPath("fs", "features", "fs-features-shard-0-0"), string(b))

	assignment := map[string][]string{"0": {"10.0.0.1:9091"}}
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v1", assignment))

	topo, err := sc.GetTopology(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "v1", topo.ActiveVersion)
	// Assignment is now derived live from pod registrations.
	assert.Equal(t, []string{"10.0.0.1:9091"}, topo.Assignment["0"])
	assert.Equal(t, int64(2), topo.TopologyVersion)
}

func TestGetTopology_StoreNotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	_, err := sc.GetTopology(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetTopology_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.GetTopology(context.Background(), "fs", "features")
	assert.Error(t, err)
}

// ── ListPods ──────────────────────────────────────────────────────────────────

func TestListPods_Empty(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	pods, err := sc.ListPods(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Empty(t, pods)
}

func TestListPods_WithPods(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)

	data := model.PodData{PodIP: "10.0.0.1", WarmVersions: []string{"v1"}}
	b, _ := json.Marshal(data)
	_ = mem.put(context.Background(), model.PodDataPath("fs", "features", "shard-0-pod-0"), string(b))

	pods, err := sc.ListPods(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Len(t, pods, 1)
	assert.Equal(t, "10.0.0.1", pods["shard-0-pod-0"].PodIP)
}

func TestListPods_CorruptEntrySkipped(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)

	_ = mem.put(context.Background(), model.PodDataPath("fs", "features", "bad-pod"), "not-json")
	valid := model.PodData{PodIP: "10.0.0.2"}
	b, _ := json.Marshal(valid)
	_ = mem.put(context.Background(), model.PodDataPath("fs", "features", "good-pod"), string(b))

	pods, err := sc.ListPods(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Len(t, pods, 1) // corrupt entry skipped
	assert.Equal(t, "10.0.0.2", pods["good-pod"].PodIP)
}

func TestListPods_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.ListPods(context.Background(), "fs", "features")
	assert.Error(t, err)
}

func TestListPods_KeyEqualsPrefix(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	// Put a key that is exactly the prefix — TrimPrefix produces "", which should be skipped.
	prefix := model.PodWatchPrefix("fs", "features")
	_ = mem.put(context.Background(), prefix, "spurious-value")

	pods, err := sc.ListPods(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Empty(t, pods) // zero-length podID skipped
}

// ── RollbackStore topology key not found edge case ────────────────────────────

func TestRollbackStore_TopologyKeyMissing(t *testing.T) {
	mem := newMemKVOps()
	// Manually set rollback version but leave topology key missing
	_ = mem.put(context.Background(), model.RollbackVersionPath("fs", "features"), "v1")

	sc := newTestStateClient(mem)
	_, err := sc.RollbackStore(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── errOnKeyOps: fails get() for a specific key ───────────────────────────────

type errOnKeyOps struct {
	*memKVOps
	failKey string
}

func (e *errOnKeyOps) get(ctx context.Context, key string) (string, int64, bool, error) {
	if key == e.failKey {
		return "", 0, false, errors.New("injected get error")
	}
	return e.memKVOps.get(ctx, key)
}

// ── swapErrorOps: atomicSwap returns a real error (not just CAS false) ────────

type swapErrorOps struct{ *memKVOps }

func (s *swapErrorOps) atomicSwap(_ context.Context, _ string, _ int64, _ map[string]string) (bool, error) {
	return false, errors.New("injected swap error")
}

// ── PublishVersion second-get error ──────────────────────────────────────────

func TestPublishVersion_VersionExistenceCheckError(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))

	// Wrap: version-existence get fails
	sc2 := newTestStateClient(&errOnKeyOps{memKVOps: mem, failKey: model.VersionPrefix("fs", "features", "v1")})
	err := sc2.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta())
	assert.Error(t, err)
}

// ── PromoteVersion second-get and atomicSwap error ───────────────────────────

func TestPromoteVersion_ActiveVersionGetError(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	sc2 := newTestStateClient(&errOnKeyOps{memKVOps: mem, failKey: model.ActiveVersionPath("fs", "features")})
	err := sc2.PromoteVersion(context.Background(), "fs", "features", "v1", nil)
	assert.Error(t, err)
}

func TestPromoteVersion_AtomicSwapError(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))

	sc2 := newTestStateClient(&swapErrorOps{mem})
	err := sc2.PromoteVersion(context.Background(), "fs", "features", "v1", nil)
	assert.Error(t, err)
}

// ── RollbackStore second-get and atomicSwap error ────────────────────────────

func TestRollbackStore_TopologyVersionGetError(t *testing.T) {
	mem := newMemKVOps()
	// Manually set rollback version so the first get succeeds
	_ = mem.put(context.Background(), model.RollbackVersionPath("fs", "features"), "v1")

	sc2 := newTestStateClient(&errOnKeyOps{memKVOps: mem, failKey: model.TopologyVersionPath("fs", "features")})
	_, err := sc2.RollbackStore(context.Background(), "fs", "features")
	assert.Error(t, err)
}

func TestRollbackStore_AtomicSwapError(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v1", nil))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v2",
		model.VersionMeta{Status: model.StatusReady}))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v2", nil))

	sc2 := newTestStateClient(&swapErrorOps{mem})
	_, err := sc2.RollbackStore(context.Background(), "fs", "features")
	assert.Error(t, err)
}

// ── GetTopology missing-meta and nil-assignment paths ────────────────────────

func TestGetTopology_ActiveVersionSetButMetaMissing(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	// Force-set activeVersion to a version ID that has no metadata
	_ = mem.put(context.Background(), model.ActiveVersionPath("fs", "features"), "ghost_v")

	topo, err := sc.GetTopology(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "ghost_v", topo.ActiveVersion)
	assert.Empty(t, topo.Assignment)
}

func TestGetTopology_VersionMetaCorrupt(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	_ = mem.put(context.Background(), model.ActiveVersionPath("fs", "features"), "v_bad")
	_ = mem.put(context.Background(), model.VersionPrefix("fs", "features", "v_bad"), "not-json")

	_, err := sc.GetTopology(context.Background(), "fs", "features")
	require.Error(t, err)
}

func TestGetTopology_NilAssignment(t *testing.T) {
	// Published (not promoted) version has nil Assignment — topology derives
	// from live pods (none registered → every shard has an empty pod list).
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	// Force-set activeVersion without going through PromoteVersion
	_ = mem.put(context.Background(), model.ActiveVersionPath("fs", "features"), "v1")

	topo, err := sc.GetTopology(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "v1", topo.ActiveVersion)
	// No pods registered → all shards have empty lists.
	for i := 0; i < 3; i++ {
		assert.Empty(t, topo.Assignment[fmt.Sprintf("%d", i)])
	}
}
