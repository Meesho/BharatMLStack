package reconciler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// fakeState is an in-memory StateReader for hermetic reconciler tests.
type fakeState struct {
	stores                  []etcdstate.StoreRef
	dataflow                map[string]*model.DataflowConfig         // "tenant/store" -> cfg
	state                   map[string]*etcdstate.StoreState         // "tenant/store" -> state
	versions                map[string]map[string]*model.VersionMeta // "tenant/store" -> vID -> meta
	pods                    map[string]map[string]model.PodData      // "tenant/store" -> podID -> data
	promotes                []promoteCall
	promoteErr              error
	assignmentUpdates       []promoteCall
	updateAssignmentChanged bool
	updateAssignmentErr     error
	retires                 []string // vIDs successfully passed to RetireVersion

	// Error injection for the read paths and per-version retire failures.
	listStoresErr   error
	getDataflowErr  error
	getStoreErr     error
	listVersionsErr error
	listPodsErr     error
	retireErr       map[string]error // vID -> error

	calls []string // StateReader method names, in call order
}

type promoteCall struct {
	tenant, store, vID string
	assignment         map[string][]string
}

func key(t, s string) string { return t + "/" + s }

func (f *fakeState) ListStores(context.Context) ([]etcdstate.StoreRef, error) {
	f.calls = append(f.calls, "ListStores")
	if f.listStoresErr != nil {
		return nil, f.listStoresErr
	}
	return f.stores, nil
}
func (f *fakeState) GetStore(_ context.Context, t, s string) (*etcdstate.StoreState, error) {
	f.calls = append(f.calls, "GetStore")
	if f.getStoreErr != nil {
		return nil, f.getStoreErr
	}
	return f.state[key(t, s)], nil
}
func (f *fakeState) GetDataflow(_ context.Context, t, s string) (*model.DataflowConfig, error) {
	f.calls = append(f.calls, "GetDataflow")
	if f.getDataflowErr != nil {
		return nil, f.getDataflowErr
	}
	return f.dataflow[key(t, s)], nil
}
func (f *fakeState) ListVersions(_ context.Context, t, s string) (map[string]*model.VersionMeta, error) {
	f.calls = append(f.calls, "ListVersions")
	if f.listVersionsErr != nil {
		return nil, f.listVersionsErr
	}
	return f.versions[key(t, s)], nil
}
func (f *fakeState) ListPods(_ context.Context, t, s string) (map[string]model.PodData, error) {
	f.calls = append(f.calls, "ListPods")
	if f.listPodsErr != nil {
		return nil, f.listPodsErr
	}
	return f.pods[key(t, s)], nil
}
func (f *fakeState) PromoteVersion(_ context.Context, t, s, vID string, a map[string][]string) error {
	f.calls = append(f.calls, "PromoteVersion")
	if f.promoteErr != nil {
		return f.promoteErr
	}
	f.promotes = append(f.promotes, promoteCall{t, s, vID, a})
	return nil
}
func (f *fakeState) UpdateAssignment(_ context.Context, t, s, vID string, a map[string][]string) (bool, error) {
	f.calls = append(f.calls, "UpdateAssignment")
	if f.updateAssignmentErr != nil {
		return false, f.updateAssignmentErr
	}
	f.assignmentUpdates = append(f.assignmentUpdates, promoteCall{t, s, vID, a})
	return f.updateAssignmentChanged, nil
}
func (f *fakeState) RetireVersion(_ context.Context, _, _, vID string) error {
	f.calls = append(f.calls, "RetireVersion")
	if err := f.retireErr[vID]; err != nil {
		return err
	}
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

// v2 is warm on every shard; v3 was published before v2 was promoted and is
// warm nowhere, because every data loader parks it while v2 is resident and
// unpromoted (docs/design/onyxdb-version-memory-guard.md §4). Promote v2: waiting for
// v3 to be covered never ends.
func TestReconcile_PromotesOlderCoveredVersionWhenNewestIsUncovered(t *testing.T) {
	f := baseFake(2, "20260101_001", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 2}
	f.versions[key("t", "s")]["20260101_002"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 2}
	f.versions[key("t", "s")]["20260101_003"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 2}
	f.pods[key("t", "s")] = warmPods(2, "20260101_002")

	New(f, 0).reconcileOnce(context.Background())

	require.Len(t, f.promotes, 1)
	assert.Equal(t, "20260101_002", f.promotes[0].vID)
	assert.Len(t, f.promotes[0].assignment["0"], 1)
	assert.Len(t, f.promotes[0].assignment["1"], 1)
}

// Both READY versions are covered: the newest wins, as it always has.
func TestReconcile_NewestCoveredVersionWins(t *testing.T) {
	f := baseFake(1, "20260101_001", true)
	f.versions[key("t", "s")]["20260101_002"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.versions[key("t", "s")]["20260101_003"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = map[string]model.PodData{
		"store-shard-0-1": {PodIP: "10.0.0.1", Port: 9091, WarmVersions: []string{"20260101_002", "20260101_003"}},
	}

	New(f, 0).reconcileOnce(context.Background())

	require.Len(t, f.promotes, 1)
	assert.Equal(t, "20260101_003", f.promotes[0].vID)
}

// No READY version is covered: nothing is promoted.
func TestReconcile_NoCoveredVersionPromotesNothing(t *testing.T) {
	f := baseFake(2, "20260101_001", true)
	f.versions[key("t", "s")]["20260101_002"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 2}
	f.versions[key("t", "s")]["20260101_003"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 2}
	f.pods[key("t", "s")] = map[string]model.PodData{
		"store-shard-0-1": {PodIP: "10.0.0.1", Port: 9091, WarmVersions: []string{"20260101_002", "20260101_003"}},
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

func TestReconcile_RefreshesStaleAssignment(t *testing.T) {
	// Active version exists, no candidate to promote, but pods have new IPs.
	f := baseFake(2, "20260101_001", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 2}
	f.pods[key("t", "s")] = warmPods(2, "20260101_001")
	f.updateAssignmentChanged = true

	New(f, 0).reconcileOnce(context.Background())

	assert.Empty(t, f.promotes) // no new version to promote
	require.Len(t, f.assignmentUpdates, 1)
	assert.Equal(t, "20260101_001", f.assignmentUpdates[0].vID)
	assert.Len(t, f.assignmentUpdates[0].assignment["0"], 1)
	assert.Len(t, f.assignmentUpdates[0].assignment["1"], 1)
}

func TestReconcile_SkipsAssignmentRefreshWhenNoActive(t *testing.T) {
	f := baseFake(1, "", true) // no active version
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	// no pods → no promote, no active → no assignment refresh

	New(f, 0).reconcileOnce(context.Background())

	assert.Empty(t, f.assignmentUpdates)
}

func TestReconcile_AssignmentRefreshCASConflictIsBenign(t *testing.T) {
	f := baseFake(1, "20260101_001", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")
	f.updateAssignmentErr = etcdstate.ErrCASConflict

	// must not panic
	New(f, 0).reconcileOnce(context.Background())
}

func TestReconcile_SkipsAssignmentRefreshAfterPromote(t *testing.T) {
	// When a new version is promoted, the reconciler returns early and does
	// not redundantly refresh the assignment (promote already wrote it).
	f := baseFake(1, "", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")

	New(f, 0).reconcileOnce(context.Background())

	require.Len(t, f.promotes, 1)
	assert.Empty(t, f.assignmentUpdates) // no redundant refresh
}

func TestPromotable(t *testing.T) {
	versions := map[string]*model.VersionMeta{
		"20260101_001": {Status: model.StatusActive},
		"20260101_002": {Status: model.StatusReady},
		"20260101_004": {Status: model.StatusReady},
		"20260101_003": {Status: model.StatusRetiring},
		"20260101_005": nil,
	}
	assert.Equal(t, []string{"20260101_004", "20260101_002"}, promotable(versions, "20260101_001"))
	assert.Empty(t, promotable(versions, "20260101_004")) // nothing newer
	assert.Empty(t, promotable(map[string]*model.VersionMeta{}, ""))
}

// ── Run ───────────────────────────────────────────────────────────────────────

// tickSignalState signals the first ListStores call, i.e. the first tick.
type tickSignalState struct {
	*fakeState
	ticked chan struct{}
	once   sync.Once
}

func (s *tickSignalState) ListStores(ctx context.Context) ([]etcdstate.StoreRef, error) {
	s.once.Do(func() { close(s.ticked) })
	return s.fakeState.ListStores(ctx)
}

func TestRun_ReconcilesOnTickUntilCancelled(t *testing.T) {
	f := baseFake(1, "", true)
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")
	st := &tickSignalState{fakeState: f, ticked: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		New(st, time.Millisecond).Run(ctx)
		close(done)
	}()

	select {
	case <-st.ticked:
	case <-time.After(5 * time.Second):
		t.Fatal("reconciler did not tick")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	// The first tick ran a full reconcile pass before Run observed cancellation.
	require.NotEmpty(t, f.promotes)
	assert.Equal(t, "20260101_001", f.promotes[0].vID)
}

func TestRun_ReturnsWithoutReconcilingWhenAlreadyCancelled(t *testing.T) {
	f := baseFake(1, "", true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	New(f, time.Hour).Run(ctx) // must return instead of blocking on the ticker

	assert.Empty(t, f.calls)
}

// ── reconcileOnce / reconcileStore failure handling ──────────────────────────

// promotableFake is a store that, absent failures, would promote 20260101_004,
// refresh the active assignment and retire 20260101_001.
func promotableFake() *fakeState {
	f := baseFake(1, "20260101_003", true)
	f.state[key("t", "s")].RollbackVersion = "20260101_002"
	v := f.versions[key("t", "s")]
	v["20260101_001"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	v["20260101_002"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	v["20260101_003"] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	v["20260101_004"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_004")
	return f
}

func TestReconcile_ListStoresErrorEndsPass(t *testing.T) {
	f := promotableFake()
	f.listStoresErr = errors.New("etcd down")

	New(f, 0).reconcileOnce(context.Background())

	assert.Equal(t, []string{"ListStores"}, f.calls)
}

func TestReconcile_NoStoresIsNoop(t *testing.T) {
	f := promotableFake()
	f.stores = nil

	New(f, 0).reconcileOnce(context.Background())

	assert.Equal(t, []string{"ListStores"}, f.calls)
}

func TestReconcile_SkipsStoreWithoutUsableDataflow(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*fakeState)
	}{
		{"dataflow read error", func(f *fakeState) { f.getDataflowErr = errors.New("timeout") }},
		{"no dataflow config", func(f *fakeState) { delete(f.dataflow, key("t", "s")) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := promotableFake()
			tc.setup(f)

			New(f, 0).reconcileOnce(context.Background())

			assert.Equal(t, []string{"ListStores", "GetDataflow"}, f.calls)
		})
	}
}

func TestReconcile_ReadErrorAbandonsStore(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*fakeState)
		wantCalls []string
	}{
		{
			name:      "GetStore error",
			setup:     func(f *fakeState) { f.getStoreErr = errors.New("timeout") },
			wantCalls: []string{"ListStores", "GetDataflow", "GetStore"},
		},
		{
			name:      "ListVersions error",
			setup:     func(f *fakeState) { f.listVersionsErr = errors.New("timeout") },
			wantCalls: []string{"ListStores", "GetDataflow", "GetStore", "ListVersions"},
		},
		{
			name:      "ListPods error",
			setup:     func(f *fakeState) { f.listPodsErr = errors.New("timeout") },
			wantCalls: []string{"ListStores", "GetDataflow", "GetStore", "ListVersions", "ListPods"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := promotableFake()
			tc.setup(f)

			New(f, 0).reconcileOnce(context.Background())

			assert.Equal(t, tc.wantCalls, f.calls)
			assert.Empty(t, f.promotes)
			assert.Empty(t, f.assignmentUpdates)
			assert.Empty(t, f.retires)
		})
	}
}

func TestReconcile_PromoteFailureSkipsRefreshAndGC(t *testing.T) {
	f := promotableFake()
	f.promoteErr = errors.New("etcd down")

	New(f, 0).reconcileOnce(context.Background())

	assert.Equal(t, []string{"ListStores", "GetDataflow", "GetStore", "ListVersions", "ListPods", "PromoteVersion"}, f.calls)
	assert.Empty(t, f.promotes)
	assert.Empty(t, f.retires)
}

func TestReconcile_AssignmentRefreshFailureStillRunsGC(t *testing.T) {
	f := promotableFake()
	f.versions[key("t", "s")]["20260101_004"].Status = model.StatusIngesting // nothing to promote
	f.updateAssignmentErr = errors.New("etcd down")

	New(f, 0).reconcileOnce(context.Background())

	assert.Empty(t, f.promotes)
	assert.Empty(t, f.assignmentUpdates)
	assert.Equal(t, []string{"20260101_001"}, f.retires)
}

func TestReconcile_GCContinuesPastRetireFailure(t *testing.T) {
	f := baseFake(1, "20260101_004", true)
	f.state[key("t", "s")].RollbackVersion = "20260101_003"
	v := f.versions[key("t", "s")]
	for _, id := range []string{"20260101_001", "20260101_002", "20260101_003", "20260101_004"} {
		v[id] = &model.VersionMeta{Status: model.StatusActive, ShardCount: 1}
	}
	f.retireErr = map[string]error{"20260101_002": errors.New("etcd down")}

	New(f, 0).reconcileOnce(context.Background())

	// 002 (newest outside the window) fails first; 001 is still retired.
	assert.Equal(t, []string{"20260101_001"}, f.retires)
	assert.Equal(t, []string{"RetireVersion", "RetireVersion"}, f.calls[len(f.calls)-2:])
}

func TestReconcile_EvaluatesEveryStore(t *testing.T) {
	f := baseFake(1, "", true)
	f.stores = []etcdstate.StoreRef{{Tenant: "t", Store: "manual"}, {Tenant: "t", Store: "s"}}
	f.dataflow[key("t", "manual")] = &model.DataflowConfig{AutoPromote: false}
	f.versions[key("t", "s")]["20260101_001"] = &model.VersionMeta{Status: model.StatusReady, ShardCount: 1}
	f.pods[key("t", "s")] = warmPods(1, "20260101_001")

	New(f, 0).reconcileOnce(context.Background())

	require.Len(t, f.promotes, 1)
	assert.Equal(t, promoteCall{"t", "s", "20260101_001", map[string][]string{"0": {"10.0.0.1:9091"}}}, f.promotes[0])
}

// ── invalid shardCount, through the real etcd state client ───────────────────

// startEmbeddedEtcd starts an in-process etcd on free loopback ports and
// returns its client URL. It is stopped when the test ends.
func startEmbeddedEtcd(t *testing.T) string {
	t.Helper()
	cu, pu := freeLoopbackURL(t), freeLoopbackURL(t)

	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	cfg.LogLevel = "error"
	cfg.ListenClientUrls = []url.URL{*cu}
	cfg.AdvertiseClientUrls = []url.URL{*cu}
	cfg.ListenPeerUrls = []url.URL{*pu}
	cfg.AdvertisePeerUrls = []url.URL{*pu}
	cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, pu.String())

	e, err := embed.StartEtcd(cfg)
	require.NoError(t, err)
	t.Cleanup(e.Close)

	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		t.Fatal("embedded etcd did not start in time")
	}
	return cu.String()
}

func freeLoopbackURL(t *testing.T) *url.URL {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return &url.URL{Scheme: "http", Host: addr}
}

// etcdFixture is a real EtcdStateClient plus a raw client for out-of-band
// writes (pod registrations, and corrupting keys the way a manual etcdctl
// edit would).
type etcdFixture struct {
	sc  *etcdstate.EtcdStateClient
	raw *clientv3.Client
}

func newEtcdFixture(t *testing.T) *etcdFixture {
	t.Helper()
	etcdURL := startEmbeddedEtcd(t)
	sc, err := etcdstate.NewEtcdStateClient([]string{etcdURL})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sc.Close() })
	raw, err := clientv3.New(clientv3.Config{Endpoints: []string{etcdURL}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	return &etcdFixture{sc: sc, raw: raw}
}

// seedPromotableStore creates the 1-shard auto-promote store t/<store> with
// READY version v already warm on its only pod, so a reconcile pass promotes it.
func (f *etcdFixture) seedPromotableStore(t *testing.T, store, v string) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, f.sc.CreateStore(ctx, model.StoreConfig{Tenant: "t", Store: store, EntityKey: "id", ShardCount: 1}))
	require.NoError(t, f.sc.PutDataflow(ctx, "t", store, model.DataflowConfig{AutoPromote: true}))
	require.NoError(t, f.sc.PublishVersion(ctx, "t", store, v, model.VersionMeta{ShardCount: 1, Status: model.StatusReady}))
	pod, err := json.Marshal(model.PodData{PodIP: "10.0.0.1", Port: 9091, WarmVersions: []string{v}})
	require.NoError(t, err)
	f.put(t, model.PodDataPath("t", store, store+"-shard-0-0"), string(pod))
}

func (f *etcdFixture) put(t *testing.T, key, val string) {
	t.Helper()
	_, err := f.raw.Put(context.Background(), key, val)
	require.NoError(t, err)
}

func (f *etcdFixture) activeVersion(t *testing.T, store string) string {
	t.Helper()
	resp, err := f.raw.Get(context.Background(), model.ActiveVersionPath("t", store))
	require.NoError(t, err)
	require.Len(t, resp.Kvs, 1)
	return string(resp.Kvs[0].Value)
}

// captureLogs redirects the global zerolog logger into a buffer for the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = prev })
	return &buf
}

// getStoreFailures maps store -> error for every "GetStore failed" log line.
func getStoreFailures(t *testing.T, logs *bytes.Buffer) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		var e struct {
			Level, Store, Error, Message string
		}
		require.NoError(t, json.Unmarshal(line, &e), string(line))
		if e.Message == "reconciler: GetStore failed" {
			assert.Equal(t, "error", e.Level)
			out[e.Store] = e.Error
		}
	}
	return out
}

// With zero shards nothing is "uncovered", so the reconciler used to promote a
// store whose shardCount was unusable with an EMPTY assignment. The store must
// instead be skipped (via the GetStore-failed path) while other stores in the
// same pass are still promoted. Only bad etcd data gets here: the API enforces
// shardCount >= 1.
func TestReconcile_NeverPromotesStoreWithInvalidShardCountInEtcd(t *testing.T) {
	f := newEtcdFixture(t)
	const v = "20260101_001"
	bad := map[string]string{"zero": "0", "negative": "-1", "corrupt": "abc", "empty": ""}
	for store, val := range bad {
		f.seedPromotableStore(t, store, v)
		f.put(t, model.ShardCountPath("t", store), val)
	}
	f.seedPromotableStore(t, "good", v)
	logs := captureLogs(t)

	New(f.sc, 0).reconcileOnce(context.Background())

	failures := getStoreFailures(t, logs)
	for store, val := range bad {
		assert.Empty(t, f.activeVersion(t, store), "store %s must not be promoted", store)
		meta, err := f.sc.GetVersionMeta(context.Background(), "t", store, v)
		require.NoError(t, err)
		assert.Equal(t, model.StatusReady, meta.Status, store)
		assert.Nil(t, meta.Assignment, store)
		assert.Contains(t, failures[store], fmt.Sprintf("store t/%s has invalid shardCount %q", store, val))
	}
	assert.Equal(t, v, f.activeVersion(t, "good"), "a valid store in the same pass is still promoted")
	assert.NotContains(t, failures, "good")
}

// Once a store is active, an unusable shardCount used to make the assignment
// refresh overwrite the active version's assignment with an empty map. It must
// be left as it was.
func TestReconcile_InvalidShardCountInEtcdKeepsActiveAssignment(t *testing.T) {
	f := newEtcdFixture(t)
	const v = "20260101_001"
	f.seedPromotableStore(t, "s", v)
	New(f.sc, 0).reconcileOnce(context.Background())
	require.Equal(t, v, f.activeVersion(t, "s"))
	want := map[string][]string{"0": {"10.0.0.1:9091"}}
	before, err := f.sc.GetVersionMeta(context.Background(), "t", "s", v)
	require.NoError(t, err)
	require.Equal(t, want, before.Assignment)

	f.put(t, model.ShardCountPath("t", "s"), "0")
	New(f.sc, 0).reconcileOnce(context.Background())

	after, err := f.sc.GetVersionMeta(context.Background(), "t", "s", v)
	require.NoError(t, err)
	assert.Equal(t, want, after.Assignment)
	assert.Equal(t, v, f.activeVersion(t, "s"))
}

// ── gcStore ──────────────────────────────────────────────────────────────────

func TestGCStore_DisabledForNonPositiveKeep(t *testing.T) {
	for _, keep := range []int{0, -1} {
		f := promotableFake()
		st := f.state[key("t", "s")]

		New(f, 0).gcStore(context.Background(), etcdstate.StoreRef{Tenant: "t", Store: "s"}, st, f.versions[key("t", "s")], keep)

		assert.Empty(t, f.calls, "keep=%d", keep)
	}
}

// ── pure helpers ─────────────────────────────────────────────────────────────

func TestVersionsToRetire_EdgeCases(t *testing.T) {
	active := &model.VersionMeta{Status: model.StatusActive}
	tests := []struct {
		name             string
		versions         map[string]*model.VersionMeta
		active, rollback string
		keep             int
		want             []string
	}{
		{
			name:   "no versions",
			active: "20260101_003", keep: 2,
			want: nil,
		},
		{
			name: "empty version ID is never retired",
			versions: map[string]*model.VersionMeta{
				"": active, "20260101_001": active, "20260101_002": active, "20260101_003": active,
			},
			active: "20260101_003", rollback: "20260101_002", keep: 1,
			want: []string{"20260101_001"},
		},
		{
			name: "nil metadata outside the window is retired",
			versions: map[string]*model.VersionMeta{
				"20260101_001": nil, "20260101_002": active, "20260101_003": active,
			},
			active: "20260101_003", rollback: "20260101_002", keep: 2,
			want: []string{"20260101_001"},
		},
		{
			name: "keep 0 still retains active rollback and in-flight",
			versions: map[string]*model.VersionMeta{
				"20260101_001": active, "20260101_002": active, "20260101_003": active,
				"20260101_004": {Status: model.StatusIngesting},
			},
			active: "20260101_003", rollback: "20260101_001", keep: 0,
			want: []string{"20260101_002"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, versionsToRetire(tc.versions, tc.active, tc.rollback, tc.keep))
		})
	}
}

func TestUncoveredShards(t *testing.T) {
	tests := []struct {
		name       string
		shardCount int
		assignment map[string][]string
		want       []string
	}{
		{"zero shards", 0, nil, nil},
		{"every shard covered", 2, map[string][]string{"0": {"a"}, "1": {"b"}}, nil},
		{"empty and absent shards are missing", 4, map[string][]string{"0": {"a"}, "1": {}, "2": {"c"}}, []string{"1", "3"}},
		{"nil assignment", 2, nil, []string{"0", "1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, uncoveredShards(tc.shardCount, tc.assignment))
		})
	}
}
