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

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
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

// assertBadShardCountErr checks the contract for a store whose shardCount can't
// be used: an error naming the store (never a zero ShardCount, and never
// ErrNotFound, because the store itself exists).
func assertBadShardCountErr(t *testing.T, state *StoreState, err error, wantInMsg ...string) {
	t.Helper()
	require.Error(t, err)
	assert.Nil(t, state)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "fs/features")
	for _, s := range wantInMsg {
		assert.Contains(t, err.Error(), s)
	}
}

// A corrupt shardCount written out-of-band (e.g. a manual etcdctl put) used to
// parse silently as 0, which let the reconciler promote an empty assignment.
func TestGetStore_CorruptShardCountInEtcd_ReturnsError(t *testing.T) {
	clientURL, teardown := startEmbeddedEtcd(t)
	defer teardown()
	sc, err := NewEtcdStateClient([]string{clientURL})
	require.NoError(t, err)
	defer sc.Close()
	ctx := context.Background()
	cfg := defaultCfg()
	require.NoError(t, sc.CreateStore(ctx, cfg))
	_, err = sc.client.Put(ctx, model.ShardCountPath(cfg.Tenant, cfg.Store), "three")
	require.NoError(t, err)

	state, err := sc.GetStore(ctx, cfg.Tenant, cfg.Store)

	assertBadShardCountErr(t, state, err, `invalid shardCount "three"`)
}

// CreateStore always writes shardCount in the same transaction as entityKey, so
// a store without it was damaged out-of-band. It is an error, not "not found".
func TestGetStore_MissingShardCountInEtcd_ReturnsError(t *testing.T) {
	clientURL, teardown := startEmbeddedEtcd(t)
	defer teardown()
	sc, err := NewEtcdStateClient([]string{clientURL})
	require.NoError(t, err)
	defer sc.Close()
	ctx := context.Background()
	cfg := defaultCfg()
	require.NoError(t, sc.CreateStore(ctx, cfg))
	_, err = sc.client.Delete(ctx, model.ShardCountPath(cfg.Tenant, cfg.Store))
	require.NoError(t, err)

	state, err := sc.GetStore(ctx, cfg.Tenant, cfg.Store)

	assertBadShardCountErr(t, state, err, "no shardCount")
}

// Unparseable values used to become 0 and non-positive ones were passed through;
// either way every shard looked covered, so an empty assignment could be
// promoted. The API enforces min=1, so only bad etcd data produces these.
func TestGetStore_InvalidShardCount_ReturnsError(t *testing.T) {
	tests := []struct {
		value   string
		wantMsg string
	}{
		{"", "invalid syntax"},
		{"abc", "invalid syntax"},
		{"3.0", "invalid syntax"},
		{" 3", "invalid syntax"},
		{"3\n", "invalid syntax"},
		{"0x3", "invalid syntax"},
		{"99999999999999999999", "out of range"},
		{"0", "must be >= 1"},
		{"-0", "must be >= 1"},
		{"-1", "must be >= 1"},
		{"-64", "must be >= 1"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q", tc.value), func(t *testing.T) {
			mem := newMemKVOps()
			sc := newTestStateClient(mem)
			require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
			require.NoError(t, mem.put(context.Background(), model.ShardCountPath("fs", "features"), tc.value))

			state, err := sc.GetStore(context.Background(), "fs", "features")

			assertBadShardCountErr(t, state, err, fmt.Sprintf("invalid shardCount %q", tc.value), tc.wantMsg)
		})
	}
}

func TestGetStore_AcceptsShardCountFromOne(t *testing.T) {
	for _, n := range []int{1, 64} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			sc := newTestStateClient(newMemKVOps())
			cfg := defaultCfg()
			cfg.ShardCount = n
			require.NoError(t, sc.CreateStore(context.Background(), cfg))

			state, err := sc.GetStore(context.Background(), cfg.Tenant, cfg.Store)

			require.NoError(t, err)
			assert.Equal(t, n, state.Config.ShardCount)
		})
	}
}

// A failed read of the shardCount key must surface as an error rather than
// being treated as a missing (zero) value.
func TestGetStore_ShardCountReadError_ReturnsError(t *testing.T) {
	mem := newMemKVOps()
	require.NoError(t, newTestStateClient(mem).CreateStore(context.Background(), defaultCfg()))
	sc := newTestStateClient(&errOnKeyOps{memKVOps: mem, failKey: model.ShardCountPath("fs", "features")})

	state, err := sc.GetStore(context.Background(), "fs", "features")

	assertBadShardCountErr(t, state, err, "injected get error")
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
	// GetTopology is resilient to corrupt version metadata: ListVersions
	// silently skips unparseable entries, so GetTopology returns successfully
	// with an empty assignment (shardCount=0) rather than propagating an error.
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	_ = mem.put(context.Background(), model.ActiveVersionPath("fs", "features"), "v_bad")
	_ = mem.put(context.Background(), model.VersionPrefix("fs", "features", "v_bad"), "not-json")

	topo, err := sc.GetTopology(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "v_bad", topo.ActiveVersion)
	assert.Empty(t, topo.Assignment)
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

// ── errOnPrefixOps: fails getPrefix() for a specific prefix ───────────────────

type errOnPrefixOps struct {
	*memKVOps
	failPrefix string
}

func (e *errOnPrefixOps) getPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	if prefix == e.failPrefix {
		return nil, errors.New("injected getPrefix error")
	}
	return e.memKVOps.getPrefix(ctx, prefix)
}

func putJSON(t *testing.T, mem *memKVOps, key string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, mem.put(context.Background(), key, string(b)))
}

// ── GetStore attaches optional configs ───────────────────────────────────────

func TestGetStore_AttachesDataflowAndClientConfig(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	df := model.DataflowConfig{SourcePath: "gs://src", NumShards: 3, AutoPromote: true}
	cc := model.ClientConfig{RequestTimeoutMs: 75, MaxConnsPerPod: 6}
	require.NoError(t, sc.PutDataflow(context.Background(), "fs", "features", df))
	require.NoError(t, sc.SetClientConfig(context.Background(), "fs", "features", cc))

	state, err := sc.GetStore(context.Background(), "fs", "features")
	require.NoError(t, err)
	require.NotNil(t, state.Dataflow)
	require.NotNil(t, state.ClientConfig)
	assert.Equal(t, df, *state.Dataflow)
	assert.Equal(t, cc, *state.ClientConfig)
}

func TestGetStore_OmitsCorruptOptionalConfigs(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, mem.put(context.Background(), model.DataflowPath("fs", "features"), "not-json"))
	require.NoError(t, mem.put(context.Background(), model.ClientConfigPath("fs", "features"), "not-json"))

	state, err := sc.GetStore(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, "catalog_id", state.Config.EntityKey)
	assert.Nil(t, state.Dataflow)
	assert.Nil(t, state.ClientConfig)
}

// ── UpdateAssignment ─────────────────────────────────────────────────────────

// storeWithActiveVersion creates the default store with v1 published and
// promoted using the given assignment, returning the backing memKVOps.
func storeWithActiveVersion(t *testing.T, assignment map[string][]string) *memKVOps {
	t.Helper()
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	require.NoError(t, sc.PromoteVersion(context.Background(), "fs", "features", "v1", assignment))
	return mem
}

func TestUpdateAssignment_WritesChangedAssignment(t *testing.T) {
	mem := storeWithActiveVersion(t, map[string][]string{"0": {"10.0.0.1:9091"}})
	sc := newTestStateClient(mem)
	fresh := map[string][]string{"0": {"10.0.0.9:9091"}, "1": {"10.0.0.2:9091"}}

	changed, err := sc.UpdateAssignment(context.Background(), "fs", "features", "v1", fresh)
	require.NoError(t, err)
	assert.True(t, changed)

	meta, err := sc.GetVersionMeta(context.Background(), "fs", "features", "v1")
	require.NoError(t, err)
	assert.Equal(t, fresh, meta.Assignment)
	assert.Equal(t, model.StatusActive, meta.Status)
	tv, _, _, _ := mem.get(context.Background(), model.TopologyVersionPath("fs", "features"))
	assert.Equal(t, "3", tv) // create=1, promote=2, refresh=3
}

func TestUpdateAssignment_NoopWhenUnchanged(t *testing.T) {
	mem := storeWithActiveVersion(t, map[string][]string{"0": {"10.0.0.1:9091", "10.0.0.2:9091"}})
	sc := newTestStateClient(mem)

	// Same pods in a different order is not a change.
	changed, err := sc.UpdateAssignment(context.Background(), "fs", "features", "v1",
		map[string][]string{"0": {"10.0.0.2:9091", "10.0.0.1:9091"}})
	require.NoError(t, err)
	assert.False(t, changed)

	tv, _, _, _ := mem.get(context.Background(), model.TopologyVersionPath("fs", "features"))
	assert.Equal(t, "2", tv) // topology version not bumped
}

func TestUpdateAssignment_VersionNotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	changed, err := sc.UpdateAssignment(context.Background(), "fs", "features", "v1", map[string][]string{"0": {"a"}})
	assert.ErrorIs(t, err, ErrNotFound)
	assert.False(t, changed)
}

func TestUpdateAssignment_TopologyKeyMissing(t *testing.T) {
	mem := newMemKVOps()
	putJSON(t, mem, model.VersionPrefix("fs", "features", "v1"), defaultMeta())
	sc := newTestStateClient(mem)

	changed, err := sc.UpdateAssignment(context.Background(), "fs", "features", "v1", map[string][]string{"0": {"a"}})
	assert.ErrorIs(t, err, ErrNotFound)
	assert.False(t, changed)
}

func TestUpdateAssignment_BackendFailures(t *testing.T) {
	newAssignment := map[string][]string{"0": {"10.0.0.9:9091"}}
	tests := []struct {
		name    string
		wrap    func(*memKVOps) kvOps
		wantErr error
		wantMsg string
	}{
		{
			name: "topology version get error",
			wrap: func(m *memKVOps) kvOps {
				return &errOnKeyOps{memKVOps: m, failKey: model.TopologyVersionPath("fs", "features")}
			},
			wantMsg: "injected get error",
		},
		{
			name:    "atomic swap error",
			wrap:    func(m *memKVOps) kvOps { return &swapErrorOps{m} },
			wantMsg: "injected swap error",
		},
		{
			name:    "CAS conflict",
			wrap:    func(m *memKVOps) kvOps { return &casFailOps{m} },
			wantErr: ErrCASConflict,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mem := storeWithActiveVersion(t, map[string][]string{"0": {"10.0.0.1:9091"}})
			sc := newTestStateClient(tc.wrap(mem))

			changed, err := sc.UpdateAssignment(context.Background(), "fs", "features", "v1", newAssignment)
			require.Error(t, err)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.EqualError(t, err, tc.wantMsg)
			}
			assert.False(t, changed)

			meta, err := newTestStateClient(mem).GetVersionMeta(context.Background(), "fs", "features", "v1")
			require.NoError(t, err)
			assert.Equal(t, map[string][]string{"0": {"10.0.0.1:9091"}}, meta.Assignment) // not overwritten
		})
	}
}

// ── assignmentsEqual ─────────────────────────────────────────────────────────

func TestAssignmentsEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b map[string][]string
		want bool
	}{
		{"both nil", nil, nil, true},
		{"nil and empty", nil, map[string][]string{}, true},
		{"identical", map[string][]string{"0": {"a", "b"}}, map[string][]string{"0": {"a", "b"}}, true},
		{"same pods different order", map[string][]string{"0": {"b", "a"}}, map[string][]string{"0": {"a", "b"}}, true},
		{"different shard count", map[string][]string{"0": {"a"}}, map[string][]string{"0": {"a"}, "1": {"b"}}, false},
		{"shard missing on other side", map[string][]string{"0": {"a"}}, map[string][]string{"1": {"a"}}, false},
		{"different pod count", map[string][]string{"0": {"a"}}, map[string][]string{"0": {"a", "b"}}, false},
		{"different pod address", map[string][]string{"0": {"a", "b"}}, map[string][]string{"0": {"a", "c"}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, assignmentsEqual(tc.a, tc.b))
		})
	}
}

func TestAssignmentsEqual_DoesNotReorderInputs(t *testing.T) {
	a := map[string][]string{"0": {"c", "a", "b"}}
	b := map[string][]string{"0": {"b", "c", "a"}}

	require.True(t, assignmentsEqual(a, b))
	assert.Equal(t, []string{"c", "a", "b"}, a["0"])
	assert.Equal(t, []string{"b", "c", "a"}, b["0"])
}

// ── GetTopology per-version / per-pod state ─────────────────────────────────

func TestGetTopology_ReportsPerVersionAndPerPodState(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	cfg := defaultCfg()
	cfg.ShardCount = 2
	require.NoError(t, sc.CreateStore(context.Background(), cfg))
	two := func(s model.VersionStatus) model.VersionMeta { return model.VersionMeta{ShardCount: 2, Status: s} }
	putJSON(t, mem, model.VersionPrefix("fs", "features", "20260101_001"), two(model.StatusRetiring))
	putJSON(t, mem, model.VersionPrefix("fs", "features", "20260101_002"), two(model.StatusActive))
	putJSON(t, mem, model.VersionPrefix("fs", "features", "20260101_003"), two(model.StatusReady))
	require.NoError(t, mem.put(context.Background(), model.ActiveVersionPath("fs", "features"), "20260101_002"))
	require.NoError(t, mem.put(context.Background(), model.RollbackVersionPath("fs", "features"), "20260101_001"))

	pods := map[string]model.PodData{
		// Shard 0: serving 002, still holds retiring 001, loading 003.
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", ServingVersion: "20260101_002",
			WarmVersions: []string{"20260101_001", "20260101_002"}, LoadingVersion: "20260101_003"},
		// Shard 1: ramping 002 → 003 at 40%, loading an unknown version.
		"fs-features-shard-1-0": {PodIP: "10.0.0.2", Port: 9200, ServingVersion: "20260101_002",
			WarmVersions: []string{"20260101_002", "20260101_003"}, LoadingVersion: "20260101_009",
			RolloutVersion: "20260101_003", RolloutPct: 40},
		// Rolling out a version the store does not know about.
		"fs-features-shard-1-1": {PodIP: "10.0.0.3", RolloutVersion: "20260101_009", RolloutPct: 10},
	}
	for id, pd := range pods {
		putJSON(t, mem, model.PodDataPath("fs", "features", id), pd)
	}

	topo, err := sc.GetTopology(context.Background(), "fs", "features")
	require.NoError(t, err)

	wantAssignment := map[string][]string{"0": {"10.0.0.1:9091"}, "1": {"10.0.0.2:9200"}}
	assert.Equal(t, "20260101_002", topo.ActiveVersion)
	assert.Equal(t, "20260101_001", topo.RollbackVersion)
	assert.Equal(t, wantAssignment, topo.Assignment)
	// Retiring 001 is excluded; the rest are sorted newest first.
	assert.Equal(t, []VersionInfo{
		{VersionID: "20260101_003", Status: "READY", WarmPods: 1, LoadingPods: 1, RollingOutPods: 1},
		{VersionID: "20260101_002", Status: "ACTIVE", WarmPods: 2, Assignment: wantAssignment},
	}, topo.Versions)
	assert.Equal(t, map[string]PodState{
		"fs-features-shard-0-0": {PodIP: "10.0.0.1", ServingVersion: "20260101_002",
			LoadingVersion: "20260101_003", WarmVersions: []string{"20260101_001", "20260101_002"}},
		"fs-features-shard-1-0": {PodIP: "10.0.0.2", ServingVersion: "20260101_002",
			LoadingVersion: "20260101_009", RolloutVersion: "20260101_003", RolloutPct: 40,
			WarmVersions: []string{"20260101_002", "20260101_003"}},
		"fs-features-shard-1-1": {PodIP: "10.0.0.3", RolloutVersion: "20260101_009", RolloutPct: 10},
	}, topo.Pods)
}

func TestGetTopology_PodListFailureFallsBackToStoredAssignment(t *testing.T) {
	stored := map[string][]string{"0": {"10.0.0.1:9091"}}
	tests := []struct {
		name           string
		activeMeta     *model.VersionMeta // nil = no metadata for the active version
		wantAssignment map[string][]string
	}{
		{"stored assignment is used", &model.VersionMeta{ShardCount: 3, Status: model.StatusActive, Assignment: stored}, stored},
		{"meta without assignment yields empty", &model.VersionMeta{ShardCount: 3, Status: model.StatusActive}, map[string][]string{}},
		{"missing meta yields empty", nil, map[string][]string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mem := newMemKVOps()
			require.NoError(t, newTestStateClient(mem).CreateStore(context.Background(), defaultCfg()))
			require.NoError(t, mem.put(context.Background(), model.ActiveVersionPath("fs", "features"), "v1"))
			if tc.activeMeta != nil {
				putJSON(t, mem, model.VersionPrefix("fs", "features", "v1"), *tc.activeMeta)
			}
			sc := newTestStateClient(&errOnPrefixOps{memKVOps: mem, failPrefix: model.PodWatchPrefix("fs", "features")})

			topo, err := sc.GetTopology(context.Background(), "fs", "features")
			require.NoError(t, err)
			assert.Equal(t, "v1", topo.ActiveVersion)
			assert.Equal(t, tc.wantAssignment, topo.Assignment)
			assert.Nil(t, topo.Pods)
			assert.Nil(t, topo.Versions)
		})
	}
}

// ── PutDataflow / GetDataflow ────────────────────────────────────────────────

func TestPutDataflow_RoundTripsThroughGetDataflow(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	cfg := model.DataflowConfig{SourcePath: "gs://src", GcsOutRoot: "gs://out", NumShards: 3, KeepVersions: 4}

	require.NoError(t, sc.PutDataflow(context.Background(), "fs", "features", cfg))

	got, err := sc.GetDataflow(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, cfg, *got)
}

func TestPutDataflow_StoreNotFound(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)

	err := sc.PutDataflow(context.Background(), "fs", "features", model.DataflowConfig{NumShards: 1})
	assert.ErrorIs(t, err, ErrNotFound)
	_, _, found, _ := mem.get(context.Background(), model.DataflowPath("fs", "features"))
	assert.False(t, found) // nothing written for an unknown store
}

func TestPutDataflow_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("etcd down")})
	err := sc.PutDataflow(context.Background(), "fs", "features", model.DataflowConfig{})
	assert.EqualError(t, err, "etcd down")
}

func TestGetDataflow_NotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	_, err := sc.GetDataflow(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetDataflow_CorruptJSON(t *testing.T) {
	mem := newMemKVOps()
	require.NoError(t, mem.put(context.Background(), model.DataflowPath("fs", "features"), "not-json"))
	_, err := newTestStateClient(mem).GetDataflow(context.Background(), "fs", "features")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing dataflow config for fs/features")
}

func TestGetDataflow_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.GetDataflow(context.Background(), "fs", "features")
	assert.EqualError(t, err, "timeout")
}

// ── SetClientConfig / GetClientConfig ────────────────────────────────────────

func TestSetClientConfig_RoundTripsThroughGetClientConfig(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	warm := true
	cfg := model.ClientConfig{ConnectTimeoutMs: 1000, MaxConnsPerPod: 2, WarmUpOnTopologyChange: &warm}

	require.NoError(t, sc.SetClientConfig(context.Background(), "fs", "features", cfg))

	got, err := sc.GetClientConfig(context.Background(), "fs", "features")
	require.NoError(t, err)
	assert.Equal(t, cfg, *got)
}

func TestSetClientConfig_StoreNotFound(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)

	err := sc.SetClientConfig(context.Background(), "fs", "features", model.ClientConfig{MaxConnsPerPod: 1})
	assert.ErrorIs(t, err, ErrNotFound)
	_, _, found, _ := mem.get(context.Background(), model.ClientConfigPath("fs", "features"))
	assert.False(t, found)
}

func TestSetClientConfig_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("etcd down")})
	err := sc.SetClientConfig(context.Background(), "fs", "features", model.ClientConfig{})
	assert.EqualError(t, err, "etcd down")
}

func TestGetClientConfig_NotFound(t *testing.T) {
	sc := newTestStateClient(newMemKVOps())
	_, err := sc.GetClientConfig(context.Background(), "fs", "features")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetClientConfig_CorruptJSON(t *testing.T) {
	mem := newMemKVOps()
	require.NoError(t, mem.put(context.Background(), model.ClientConfigPath("fs", "features"), "not-json"))
	_, err := newTestStateClient(mem).GetClientConfig(context.Background(), "fs", "features")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing client config for fs/features")
}

func TestGetClientConfig_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	_, err := sc.GetClientConfig(context.Background(), "fs", "features")
	assert.EqualError(t, err, "timeout")
}

// ── ListStores ───────────────────────────────────────────────────────────────

func TestListStores_ReturnsEveryStoreAcrossTenants(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	for _, cfg := range []model.StoreConfig{
		{Tenant: "fs", Store: "features", EntityKey: "catalog_id", ShardCount: 3},
		{Tenant: "fs", Store: "embeddings", EntityKey: "user_id", ShardCount: 1},
		{Tenant: "recsys", Store: "catalog", EntityKey: "sku", ShardCount: 2},
	} {
		require.NoError(t, sc.CreateStore(context.Background(), cfg))
	}

	refs, err := sc.ListStores(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []StoreRef{
		{Tenant: "fs", Store: "features"},
		{Tenant: "fs", Store: "embeddings"},
		{Tenant: "recsys", Store: "catalog"},
	}, refs)
}

func TestListStores_SkipsMalformedEntityKeyPaths(t *testing.T) {
	mem := newMemKVOps()
	prefix := model.AppPrefix + "/tenants/"
	for _, k := range []string{
		prefix + "fs/entityKey",                // no "/stores/" segment
		prefix + "/stores/orphan/entityKey",    // empty tenant
		prefix + "fs/stores//entityKey",        // empty store
		prefix + "fs/stores/features/dataflow", // not an entityKey marker
	} {
		require.NoError(t, mem.put(context.Background(), k, "x"))
	}

	refs, err := newTestStateClient(mem).ListStores(context.Background())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestListStores_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	refs, err := sc.ListStores(context.Background())
	assert.EqualError(t, err, "timeout")
	assert.Nil(t, refs)
}

// ── ListVersions ─────────────────────────────────────────────────────────────

func TestListVersions_ReturnsParsedVersionsSkippingBadEntries(t *testing.T) {
	mem := newMemKVOps()
	sc := newTestStateClient(mem)
	require.NoError(t, sc.CreateStore(context.Background(), defaultCfg()))
	require.NoError(t, sc.PublishVersion(context.Background(), "fs", "features", "v1", defaultMeta()))
	// A key equal to the prefix (empty version ID) and a corrupt entry are skipped.
	require.NoError(t, mem.put(context.Background(), model.VersionsWatchPrefix("fs", "features"), "spurious"))
	require.NoError(t, mem.put(context.Background(), model.VersionPrefix("fs", "features", "bad"), "not-json"))

	versions, err := sc.ListVersions(context.Background(), "fs", "features")
	require.NoError(t, err)
	want := defaultMeta()
	assert.Equal(t, map[string]*model.VersionMeta{"v1": &want}, versions)
}

func TestListVersions_BackendError(t *testing.T) {
	sc := newTestStateClient(&errKVOps{err: errors.New("timeout")})
	versions, err := sc.ListVersions(context.Background(), "fs", "features")
	assert.EqualError(t, err, "timeout")
	assert.Nil(t, versions)
}
