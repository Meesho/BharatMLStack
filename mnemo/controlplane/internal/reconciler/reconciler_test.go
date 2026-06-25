package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// fakeState is an in-memory StateReader for hermetic reconciler tests.
type fakeState struct {
	stores     []etcdstate.StoreRef
	dataflow   map[string]*model.DataflowConfig         // "tenant/store" -> cfg
	state      map[string]*etcdstate.StoreState         // "tenant/store" -> state
	versions   map[string]map[string]*model.VersionMeta // "tenant/store" -> vID -> meta
	pods       map[string]map[string]model.PodData      // "tenant/store" -> podID -> data
	promotes   []promoteCall
	promoteErr error
	retires    []string // vIDs passed to RetireVersion
}

type promoteCall struct {
	tenant, store, vID string
	assignment         map[string][]string
}

func key(t, s string) string { return t + "/" + s }

func (f *fakeState) ListStores(context.Context) ([]etcdstate.StoreRef, error) {
	return f.stores, nil
}
func (f *fakeState) GetStore(_ context.Context, t, s string) (*etcdstate.StoreState, error) {
	return f.state[key(t, s)], nil
}
func (f *fakeState) GetDataflow(_ context.Context, t, s string) (*model.DataflowConfig, error) {
	return f.dataflow[key(t, s)], nil
}
func (f *fakeState) ListVersions(_ context.Context, t, s string) (map[string]*model.VersionMeta, error) {
	return f.versions[key(t, s)], nil
}
func (f *fakeState) ListPods(_ context.Context, t, s string) (map[string]model.PodData, error) {
	return f.pods[key(t, s)], nil
}
func (f *fakeState) PromoteVersion(_ context.Context, t, s, vID string, a map[string][]string) error {
	if f.promoteErr != nil {
		return f.promoteErr
	}
	f.promotes = append(f.promotes, promoteCall{t, s, vID, a})
	return nil
}
func (f *fakeState) RetireVersion(_ context.Context, _, _, vID string) error {
	f.retires = append(f.retires, vID)
	return nil
}

// warmPods builds a pod map: one warm pod per shard in [0,shardCount) for version v.
func warmPods(shardCount int, v string) map[string]model.PodData {
	m := make(map[string]model.PodData)
	for i := 0; i < shardCount; i++ {
		podID := "store-shard-" + itoa(i) + "-1"
		m[podID] = model.PodData{PodIP: "10.0.0.1", Port: 9091 + i, WarmVersions: []string{v}}
	}
	return m
}

func itoa(i int) string { return string(rune('0' + i)) }

func baseFake(shardCount int, active string, autoPromote bool) *fakeState {
	return &fakeState{
		stores:   []etcdstate.StoreRef{{Tenant: "t", Store: "s"}},
		dataflow: map[string]*model.DataflowConfig{key("t", "s"): {AutoPromote: autoPromote}},
		state: map[string]*etcdstate.StoreState{
			key("t", "s"): {Config: model.StoreConfig{Tenant: "t", Store: "s", ShardCount: shardCount}, ActiveVersion: active},
		},
		versions: map[string]map[string]*model.VersionMeta{key("t", "s"): {}},
		pods:     map[string]map[string]model.PodData{},
	}
}

func TestReconcile_PromotesReadyAndFullyWarm(t *testing.T) {
	f := baseFake(2, "", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 2}
	f.pods[key("t", "s")] = warmPods(2, "20260101_001")

	New(f, 0).reconcileOnce(context.Background())

	require.Len(t, f.promotes, 1)
	assert.Equal(t, "20260101_001", f.promotes[0].vID)
	assert.Len(t, f.promotes[0].assignment["0"], 1)
	assert.Len(t, f.promotes[0].assignment["1"], 1)
}

func TestReconcile_SkipsWhenCoverageIncomplete(t *testing.T) {
	f := baseFake(2, "", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 2}
	// only shard 0 warm
	f.pods[key("t", "s")] = map[string]model.PodData{
		"store-shard-0-1": {PodIP: "10.0.0.1", Port: 9091, WarmVersions: []string{"20260101_001"}},
	}

	New(f, 0).reconcileOnce(context.Background())
	assert.Empty(t, f.promotes)
}

func TestReconcile_SkipsWhenNotOptedIn(t *testing.T) {
	f := baseFake(1, "", false) // autoPromote=false
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")

	New(f, 0).reconcileOnce(context.Background())
	assert.Empty(t, f.promotes)
}

func TestReconcile_SkipsNonReadyStatus(t *testing.T) {
	f := baseFake(1, "", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")

	New(f, 0).reconcileOnce(context.Background())
	assert.Empty(t, f.promotes)
}

func TestReconcile_DoesNotRepromoteActive(t *testing.T) {
	// READY version equals the active one (e.g. status not yet rewritten) → skip.
	f := baseFake(1, "20260101_001", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")

	New(f, 0).reconcileOnce(context.Background())
	assert.Empty(t, f.promotes)
}

func TestReconcile_PicksNewestReadyAboveActive(t *testing.T) {
	f := baseFake(1, "20260101_001", true)
	v := f.versions[key("t", "s")]
	v["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	v["20260101_002"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	v["20260101_003"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	// both newer versions warm
	f.pods[key("t", "s")] = map[string]model.PodData{
		"store-shard-0-1": {PodIP: "10.0.0.1", Port: 9091, WarmVersions: []string{"20260101_002", "20260101_003"}},
	}

	New(f, 0).reconcileOnce(context.Background())
	require.Len(t, f.promotes, 1)
	assert.Equal(t, "20260101_003", f.promotes[0].vID) // newest wins
}

func TestReconcile_CASConflictIsBenign(t *testing.T) {
	f := baseFake(1, "", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")
	f.promoteErr = etcdstate.ErrCASConflict

	// must not panic / must complete cleanly
	New(f, 0).reconcileOnce(context.Background())
}

func TestReconcile_GCRetiresOldVersionsKeepingLast2(t *testing.T) {
	// active=003, rollback=002; keep default 2 → retire 001 (and any older).
	f := baseFake(1, "20260101_003", true)
	f.state[key("t", "s")].RollbackVersion = "20260101_002"
	v := f.versions[key("t", "s")]
	v["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1} // stale
	v["20260101_002"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	v["20260101_003"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}

	New(f, 0).reconcileOnce(context.Background())

	assert.Equal(t, []string{"20260101_001"}, f.retires)
	assert.Empty(t, f.promotes)
}

func TestReconcile_GCNeverRetiresActiveOrRollback(t *testing.T) {
	f := baseFake(1, "20260101_003", true)
	f.state[key("t", "s")].RollbackVersion = "20260101_001" // rollback is OLD (gap)
	v := f.versions[key("t", "s")]
	v["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1} // rollback
	v["20260101_002"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1} // between
	v["20260101_003"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1} // active

	New(f, 0).reconcileOnce(context.Background())

	// keep window = active(003) + next below (002); rollback(001) also kept → nothing retired
	assert.Empty(t, f.retires)
}

func TestReconcile_GCDisabledWhenNotManaged(t *testing.T) {
	f := baseFake(1, "20260101_003", false) // autoPromote=false, no KeepVersions
	v := f.versions[key("t", "s")]
	v["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	v["20260101_003"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}

	New(f, 0).reconcileOnce(context.Background())
	assert.Empty(t, f.retires)
}

func TestReconcile_GCSkipsBeforeFirstActive(t *testing.T) {
	f := baseFake(1, "", true) // no active version yet
	v := f.versions[key("t", "s")]
	v["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	// no pods warm → no promote; and no active → no GC
	New(f, 0).reconcileOnce(context.Background())
	assert.Empty(t, f.retires)
}

func TestVersionsToRetire(t *testing.T) {
	mk := func(s model.VersionStatus) *model.VersionMeta { return &model.VersionMeta{Status: s} }
	versions := map[string]*model.VersionMeta{
		"20260101_001": mk(model.StatusActive),
		"20260101_002": mk(model.StatusActive),
		"20260101_003": mk(model.StatusActive), // active
		"20260101_004": mk(model.StatusReady),  // in-flight (newer than active)
		"20260101_000": mk(model.StatusRetiring),
	}
	// active=003, rollback=002, keep=2 → keep {003,002,004(in-flight)}; 000 already retiring → skip; retire {001}
	got := versionsToRetire(versions, "20260101_003", "20260101_002", 2)
	assert.Equal(t, []string{"20260101_001"}, got)

	// keep=1 → keep {003, 004(in-flight), rollback 002}; retire {001}
	got = versionsToRetire(versions, "20260101_003", "20260101_002", 1)
	assert.ElementsMatch(t, []string{"20260101_001"}, got)
}

func TestEffectiveKeep(t *testing.T) {
	assert.Equal(t, 5, effectiveKeep(&model.DataflowConfig{KeepVersions: 5}))
	assert.Equal(t, defaultKeepVersions, effectiveKeep(&model.DataflowConfig{AutoPromote: true}))
	assert.Equal(t, 0, effectiveKeep(&model.DataflowConfig{}))
	assert.Equal(t, 3, effectiveKeep(&model.DataflowConfig{AutoPromote: true, KeepVersions: 3}))
}

func TestNewestPromotable(t *testing.T) {
	versions := map[string]*model.VersionMeta{
		"20260101_001": {Status: model.StatusActive},
		"20260101_002": {Status: model.StatusReady},
		"20260101_004": {Status: model.StatusReady},
		"20260101_003": {Status: model.StatusRetiring},
	}
	assert.Equal(t, "20260101_004", newestPromotable(versions, "20260101_001"))
	assert.Equal(t, "", newestPromotable(versions, "20260101_004")) // nothing newer
	assert.Equal(t, "", newestPromotable(map[string]*model.VersionMeta{}, ""))
}
