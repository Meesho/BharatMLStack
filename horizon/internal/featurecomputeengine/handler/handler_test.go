package handler

import (
	"context"
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetdeps"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetregistry"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetstate"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/datasetpartitions"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/materializations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock Repositories
// ---------------------------------------------------------------------------

type mockAssetRegistryRepo struct{ mock.Mock }

func (m *mockAssetRegistryRepo) Upsert(ctx context.Context, row assetregistry.AssetSpecRow) error {
	return m.Called(ctx, row).Error(0)
}
func (m *mockAssetRegistryRepo) BulkUpsert(ctx context.Context, rows []assetregistry.AssetSpecRow) error {
	return m.Called(ctx, rows).Error(0)
}
func (m *mockAssetRegistryRepo) Get(ctx context.Context, name string) (*assetregistry.AssetSpecRow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetregistry.AssetSpecRow), args.Error(1)
}
func (m *mockAssetRegistryRepo) GetByNotebook(ctx context.Context, nb string) ([]assetregistry.AssetSpecRow, error) {
	args := m.Called(ctx, nb)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetregistry.AssetSpecRow), args.Error(1)
}
func (m *mockAssetRegistryRepo) GetByEntity(ctx context.Context, e string) ([]assetregistry.AssetSpecRow, error) {
	args := m.Called(ctx, e)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetregistry.AssetSpecRow), args.Error(1)
}
func (m *mockAssetRegistryRepo) GetAll(ctx context.Context) ([]assetregistry.AssetSpecRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetregistry.AssetSpecRow), args.Error(1)
}
func (m *mockAssetRegistryRepo) Delete(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}
func (m *mockAssetRegistryRepo) GetByTriggerType(ctx context.Context, tt string) ([]assetregistry.AssetSpecRow, error) {
	args := m.Called(ctx, tt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetregistry.AssetSpecRow), args.Error(1)
}

type mockAssetDepsRepo struct{ mock.Mock }

func (m *mockAssetDepsRepo) ReplaceForAsset(ctx context.Context, name string, deps []assetdeps.AssetDependencyRow) error {
	return m.Called(ctx, name, deps).Error(0)
}
func (m *mockAssetDepsRepo) GetDependencies(ctx context.Context, name string) ([]assetdeps.AssetDependencyRow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetdeps.AssetDependencyRow), args.Error(1)
}
func (m *mockAssetDepsRepo) GetDependents(ctx context.Context, name string) ([]assetdeps.AssetDependencyRow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetdeps.AssetDependencyRow), args.Error(1)
}
func (m *mockAssetDepsRepo) GetAllEdges(ctx context.Context) ([]assetdeps.AssetDependencyRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetdeps.AssetDependencyRow), args.Error(1)
}
func (m *mockAssetDepsRepo) DeleteForAsset(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

type mockAssetStateRepo struct{ mock.Mock }

func (m *mockAssetStateRepo) BulkUpsert(ctx context.Context, rows []assetstate.AssetNecessityRow) error {
	return m.Called(ctx, rows).Error(0)
}
func (m *mockAssetStateRepo) Get(ctx context.Context, name string) (*assetstate.AssetNecessityRow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetstate.AssetNecessityRow), args.Error(1)
}
func (m *mockAssetStateRepo) GetAll(ctx context.Context) ([]assetstate.AssetNecessityRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetstate.AssetNecessityRow), args.Error(1)
}
func (m *mockAssetStateRepo) GetByNecessity(ctx context.Context, n string) ([]assetstate.AssetNecessityRow, error) {
	args := m.Called(ctx, n)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetstate.AssetNecessityRow), args.Error(1)
}
func (m *mockAssetStateRepo) SetServingOverride(ctx context.Context, name string, serving bool, reason string, by string) error {
	return m.Called(ctx, name, serving, reason, by).Error(0)
}
func (m *mockAssetStateRepo) ClearServingOverride(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

type mockMaterializationsRepo struct{ mock.Mock }

func (m *mockMaterializationsRepo) Create(ctx context.Context, row materializations.MaterializationRow) error {
	return m.Called(ctx, row).Error(0)
}
func (m *mockMaterializationsRepo) GetByComputeKey(ctx context.Context, asset, part, key string) (*materializations.MaterializationRow, error) {
	args := m.Called(ctx, asset, part, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*materializations.MaterializationRow), args.Error(1)
}
func (m *mockMaterializationsRepo) GetLatest(ctx context.Context, asset, part string) (*materializations.MaterializationRow, error) {
	args := m.Called(ctx, asset, part)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*materializations.MaterializationRow), args.Error(1)
}
func (m *mockMaterializationsRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *mockMaterializationsRepo) GetByAsset(ctx context.Context, asset string) ([]materializations.MaterializationRow, error) {
	args := m.Called(ctx, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]materializations.MaterializationRow), args.Error(1)
}
func (m *mockMaterializationsRepo) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

type mockDatasetPartitionsRepo struct{ mock.Mock }

func (m *mockDatasetPartitionsRepo) Upsert(ctx context.Context, row datasetpartitions.DatasetPartitionRow) error {
	return m.Called(ctx, row).Error(0)
}
func (m *mockDatasetPartitionsRepo) Get(ctx context.Context, ds, pk string) (*datasetpartitions.DatasetPartitionRow, error) {
	args := m.Called(ctx, ds, pk)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*datasetpartitions.DatasetPartitionRow), args.Error(1)
}
func (m *mockDatasetPartitionsRepo) GetPartitions(ctx context.Context, ds string) ([]datasetpartitions.DatasetPartitionRow, error) {
	args := m.Called(ctx, ds)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]datasetpartitions.DatasetPartitionRow), args.Error(1)
}
func (m *mockDatasetPartitionsRepo) IsReady(ctx context.Context, ds, pk string) (bool, error) {
	args := m.Called(ctx, ds, pk)
	return args.Bool(0), args.Error(1)
}
func (m *mockDatasetPartitionsRepo) AreAllReady(ctx context.Context, inputs []datasetpartitions.DatasetPartitionInput) (bool, error) {
	args := m.Called(ctx, inputs)
	return args.Bool(0), args.Error(1)
}
func (m *mockDatasetPartitionsRepo) MarkReady(ctx context.Context, ds, pk string, dv int64) error {
	return m.Called(ctx, ds, pk, dv).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testConfig() Config {
	return Config{
		ArtifactBasePath: "/_artifacts",
		ServingBasePath:  "/serving",
		DefaultPartition: "ds",
	}
}

func newTestHandler(
	ar *mockAssetRegistryRepo,
	ad *mockAssetDepsRepo,
	as *mockAssetStateRepo,
	mat *mockMaterializationsRepo,
	dp *mockDatasetPartitionsRepo,
) *Handler {
	return New(testConfig(), ar, ad, as, mat, dp)
}

func makeLinearDAG() *LineageDAG {
	return &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: true, Notebook: "nb1", TriggerType: "schedule_3h", AssetVersion: "v1"},
			"B": {AssetName: "B", Serving: false, Notebook: "nb1", TriggerType: "upstream", AssetVersion: "v1"},
			"C": {AssetName: "C", Serving: false, Notebook: "nb1", TriggerType: "upstream", AssetVersion: "v1"},
		},
		Parents: map[string][]string{
			"B": {"A"},
			"C": {"B"},
		},
		Children: map[string][]string{
			"A": {"B"},
			"B": {"C"},
		},
		Edges: []DAGEdge{
			{From: "A", To: "B", InputType: "internal"},
			{From: "B", To: "C", InputType: "internal"},
		},
	}
}

func makeDiamondDAG() *LineageDAG {
	return &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: false, Notebook: "nb1", TriggerType: "schedule_3h", AssetVersion: "v1"},
			"B": {AssetName: "B", Serving: false, Notebook: "nb1", TriggerType: "upstream", AssetVersion: "v1"},
			"C": {AssetName: "C", Serving: false, Notebook: "nb1", TriggerType: "upstream", AssetVersion: "v1"},
			"D": {AssetName: "D", Serving: true, Notebook: "nb1", TriggerType: "upstream", AssetVersion: "v1"},
		},
		Parents: map[string][]string{
			"B": {"A"},
			"C": {"A"},
			"D": {"B", "C"},
		},
		Children: map[string][]string{
			"A": {"B", "C"},
			"B": {"D"},
			"C": {"D"},
		},
		Edges: []DAGEdge{
			{From: "A", To: "B", InputType: "internal"},
			{From: "A", To: "C", InputType: "internal"},
			{From: "B", To: "D", InputType: "internal"},
			{From: "C", To: "D", InputType: "internal"},
		},
	}
}

func makeCyclicDAG() *LineageDAG {
	return &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A"},
			"B": {AssetName: "B"},
			"C": {AssetName: "C"},
		},
		Parents: map[string][]string{
			"A": {"C"},
			"B": {"A"},
			"C": {"B"},
		},
		Children: map[string][]string{
			"A": {"B"},
			"B": {"C"},
			"C": {"A"},
		},
		Edges: []DAGEdge{
			{From: "A", To: "B", InputType: "internal"},
			{From: "B", To: "C", InputType: "internal"},
			{From: "C", To: "A", InputType: "internal"},
		},
	}
}

func makeExternalDepDAG() *LineageDAG {
	return &LineageDAG{
		Nodes: map[string]AssetNode{
			"B": {AssetName: "B", Serving: true, TriggerType: "upstream", AssetVersion: "v1"},
		},
		Parents: map[string][]string{
			"B": {"ext:kafka"},
		},
		Children: map[string][]string{
			"ext:kafka": {"B"},
		},
		Edges: []DAGEdge{
			{From: "ext:kafka", To: "B", InputType: "external"},
		},
	}
}

// ---------------------------------------------------------------------------
// DAG Builder Tests
// ---------------------------------------------------------------------------

func TestBuildDAG_LinearChain(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	mat := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, mat, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "A", AssetVersion: "v1", EntityName: "user", EntityKey: "uid", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "B", AssetVersion: "v1", EntityName: "user", EntityKey: "uid", Notebook: "nb1", TriggerType: "upstream", Serving: false},
		{AssetName: "C", AssetVersion: "v1", EntityName: "user", EntityKey: "uid", Notebook: "nb1", TriggerType: "upstream", Serving: false},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "B", InputName: "A", InputType: "internal"},
		{AssetName: "C", InputName: "B", InputType: "internal"},
	}, nil)

	ctx := context.Background()
	dag, err := h.BuildDAG(ctx)

	assert.NoError(t, err)
	assert.Len(t, dag.Nodes, 3)
	assert.Equal(t, []string{"A"}, dag.Parents["B"])
	assert.Equal(t, []string{"B"}, dag.Parents["C"])
	assert.Equal(t, []string{"B"}, dag.Children["A"])
	assert.Equal(t, []string{"C"}, dag.Children["B"])
}

func TestTopologicalSort_LinearChain(t *testing.T) {
	dag := makeLinearDAG()
	sorted, err := dag.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, sorted)
}

func TestTopologicalSort_Diamond(t *testing.T) {
	dag := makeDiamondDAG()
	sorted, err := dag.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, "A", sorted[0], "A must be first")
	assert.Equal(t, "D", sorted[3], "D must be last")
}

func TestTopologicalSort_CycleDetected(t *testing.T) {
	dag := makeCyclicDAG()
	_, err := dag.TopologicalSort()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestUpstreamOf_Diamond(t *testing.T) {
	dag := makeDiamondDAG()
	upstream := dag.UpstreamOf("D")

	assert.ElementsMatch(t, []string{"A", "B", "C"}, upstream)
}

func TestDownstreamOf_Diamond(t *testing.T) {
	dag := makeDiamondDAG()
	downstream := dag.DownstreamOf("A")

	assert.ElementsMatch(t, []string{"B", "C", "D"}, downstream)
}

func TestValidate_ExternalDep(t *testing.T) {
	dag := makeExternalDepDAG()
	issues := dag.Validate()

	assert.Empty(t, issues, "external dependencies should not generate warnings")
}

func TestValidate_MissingInternalDep(t *testing.T) {
	dag := &LineageDAG{
		Nodes: map[string]AssetNode{
			"B": {AssetName: "B"},
		},
		Parents: map[string][]string{
			"B": {"A"},
		},
		Children: map[string][]string{
			"A": {"B"},
		},
		Edges: []DAGEdge{
			{From: "A", To: "B", InputType: "internal"},
		},
	}
	issues := dag.Validate()

	assert.Len(t, issues, 1)
	assert.Contains(t, issues[0], "unregistered internal asset")
}

// ---------------------------------------------------------------------------
// Necessity Resolver Tests
// ---------------------------------------------------------------------------

func TestResolveNecessity_AllActive(t *testing.T) {
	dag := &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: true},
			"B": {AssetName: "B", Serving: true},
			"C": {AssetName: "C", Serving: true},
		},
		Parents:  map[string][]string{},
		Children: map[string][]string{},
	}

	result := resolveNecessityFromDAG(dag, nil)

	assert.Equal(t, NecessityActive, result["A"])
	assert.Equal(t, NecessityActive, result["B"])
	assert.Equal(t, NecessityActive, result["C"])
}

func TestResolveNecessity_OneActiveTwoTransient(t *testing.T) {
	dag := &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: false},
			"B": {AssetName: "B", Serving: false},
			"C": {AssetName: "C", Serving: true},
		},
		Parents: map[string][]string{
			"C": {"B"},
			"B": {"A"},
		},
		Children: map[string][]string{
			"A": {"B"},
			"B": {"C"},
		},
		Edges: []DAGEdge{
			{From: "A", To: "B", InputType: "internal"},
			{From: "B", To: "C", InputType: "internal"},
		},
	}

	result := resolveNecessityFromDAG(dag, nil)

	assert.Equal(t, NecessityTransient, result["A"])
	assert.Equal(t, NecessityTransient, result["B"])
	assert.Equal(t, NecessityActive, result["C"])
}

func TestResolveNecessity_AllSkipped(t *testing.T) {
	dag := &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: false},
			"B": {AssetName: "B", Serving: false},
		},
		Parents:  map[string][]string{},
		Children: map[string][]string{},
	}

	result := resolveNecessityFromDAG(dag, nil)

	assert.Equal(t, NecessitySkipped, result["A"])
	assert.Equal(t, NecessitySkipped, result["B"])
}

func TestResolveNecessity_ServingOverride(t *testing.T) {
	dag := &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: true},
		},
		Parents:  map[string][]string{},
		Children: map[string][]string{},
	}

	overrideFalse := false
	overrides := map[string]*bool{
		"A": &overrideFalse,
	}

	result := resolveNecessityFromDAG(dag, overrides)

	assert.Equal(t, NecessitySkipped, result["A"], "override should override spec serving=true")
}

func TestResolveNecessity_Chain(t *testing.T) {
	dag := &LineageDAG{
		Nodes: map[string]AssetNode{
			"A": {AssetName: "A", Serving: true},
			"B": {AssetName: "B", Serving: false},
			"C": {AssetName: "C", Serving: false},
		},
		Parents: map[string][]string{
			"A": {"B"},
			"B": {"C"},
		},
		Children: map[string][]string{
			"C": {"B"},
			"B": {"A"},
		},
		Edges: []DAGEdge{
			{From: "B", To: "A", InputType: "internal"},
			{From: "C", To: "B", InputType: "internal"},
		},
	}

	result := resolveNecessityFromDAG(dag, nil)

	assert.Equal(t, NecessityActive, result["A"])
	assert.Equal(t, NecessityTransient, result["B"])
	assert.Equal(t, NecessityTransient, result["C"])
}

func TestResolveNecessity_Integration(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	mat := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, mat, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", Serving: true, AssetVersion: "v1"},
		{AssetName: "Y", Serving: false, AssetVersion: "v1"},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "X", InputName: "Y", InputType: "internal"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{}, nil)
	as.On("BulkUpsert", mock.Anything, mock.Anything).Return(nil)

	result, err := h.ResolveNecessity(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, NecessityActive, result["X"])
	assert.Equal(t, NecessityTransient, result["Y"])
	as.AssertCalled(t, "BulkUpsert", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// Compute Key Tests
// ---------------------------------------------------------------------------

func TestGenerateComputeKey_Deterministic(t *testing.T) {
	input := ComputeKeyInput{
		AssetVersion:   "abc123",
		InputVersions:  map[string]string{"orders": "10", "users": "5"},
		Params:         map[string]string{"mode": "full"},
		EnvFingerprint: "envhash",
	}

	key1 := GenerateComputeKey(input)
	key2 := GenerateComputeKey(input)

	assert.Equal(t, key1, key2)
	assert.Len(t, key1, 16)
}

func TestGenerateComputeKey_DifferentVersion(t *testing.T) {
	base := ComputeKeyInput{
		AssetVersion:   "v1",
		InputVersions:  map[string]string{"A": "1"},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}
	modified := ComputeKeyInput{
		AssetVersion:   "v2",
		InputVersions:  map[string]string{"A": "1"},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}

	assert.NotEqual(t, GenerateComputeKey(base), GenerateComputeKey(modified))
}

func TestGenerateComputeKey_DifferentInputVersion(t *testing.T) {
	base := ComputeKeyInput{
		AssetVersion:   "v1",
		InputVersions:  map[string]string{"A": "1"},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}
	modified := ComputeKeyInput{
		AssetVersion:   "v1",
		InputVersions:  map[string]string{"A": "2"},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}

	assert.NotEqual(t, GenerateComputeKey(base), GenerateComputeKey(modified))
}

func TestGenerateComputeKey_WindowedSameVersions(t *testing.T) {
	input1 := ComputeKeyInput{
		AssetVersion: "v1",
		InputVersions: map[string]string{
			"orders@2024-01-01": "1",
			"orders@2024-01-02": "1",
			"orders@2024-01-03": "1",
		},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}
	input2 := ComputeKeyInput{
		AssetVersion: "v1",
		InputVersions: map[string]string{
			"orders@2024-01-01": "1",
			"orders@2024-01-02": "1",
			"orders@2024-01-03": "1",
		},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}

	assert.Equal(t, GenerateComputeKey(input1), GenerateComputeKey(input2))
}

func TestGenerateComputeKey_WindowedOneChanges(t *testing.T) {
	input1 := ComputeKeyInput{
		AssetVersion: "v1",
		InputVersions: map[string]string{
			"orders@2024-01-01": "1",
			"orders@2024-01-02": "1",
			"orders@2024-01-03": "1",
		},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}
	input2 := ComputeKeyInput{
		AssetVersion: "v1",
		InputVersions: map[string]string{
			"orders@2024-01-01": "1",
			"orders@2024-01-02": "2",
			"orders@2024-01-03": "1",
		},
		Params:         map[string]string{},
		EnvFingerprint: "",
	}

	assert.NotEqual(t, GenerateComputeKey(input1), GenerateComputeKey(input2))
}

func TestResolveComputeKey_CacheMiss(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("Get", mock.Anything, "X").Return(&assetregistry.AssetSpecRow{
		AssetName: "X", AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, "X").Return([]assetdeps.AssetDependencyRow{
		{AssetName: "X", InputName: "src1", InputType: "external"},
	}, nil)
	dv := int64(10)
	dp.On("Get", mock.Anything, "src1", "2024-01-15").Return(&datasetpartitions.DatasetPartitionRow{
		DatasetName: "src1", PartitionKey: "2024-01-15", DeltaVersion: &dv,
	}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, "X", "2024-01-15", mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)

	key, hit, err := h.ResolveComputeKey(context.Background(), "X", "2024-01-15")

	assert.NoError(t, err)
	assert.False(t, hit)
	assert.Len(t, key, 16)
}

func TestResolveComputeKey_CacheHit(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("Get", mock.Anything, "X").Return(&assetregistry.AssetSpecRow{
		AssetName: "X", AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, "X").Return([]assetdeps.AssetDependencyRow{}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, "X", "2024-01-15", mock.Anything).
		Return(&materializations.MaterializationRow{
			AssetName: "X", PartitionKey: "2024-01-15", Status: "succeeded",
			ArtifactPath: "/_artifacts/X/2024-01-15/abc/",
		}, nil)

	key, hit, err := h.ResolveComputeKey(context.Background(), "X", "2024-01-15")

	assert.NoError(t, err)
	assert.True(t, hit)
	assert.Len(t, key, 16)
}

// ---------------------------------------------------------------------------
// Trigger Evaluator Tests
// ---------------------------------------------------------------------------

func TestEvaluateUpstreamTriggers_BasicTrigger(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "A", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "B", AssetVersion: "v1", Notebook: "nb1", TriggerType: "upstream", Serving: true},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "B", InputName: "A", InputType: "internal"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "A", Necessity: "active"},
		{AssetName: "B", Necessity: "active"},
	}, nil)
	dp.On("IsReady", mock.Anything, "A", "2024-01-15").Return(true, nil)

	ar.On("Get", mock.Anything, "B").Return(&assetregistry.AssetSpecRow{
		AssetName: "B", AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, "B").Return([]assetdeps.AssetDependencyRow{
		{AssetName: "B", InputName: "A", InputType: "internal"},
	}, nil)
	dv := int64(1)
	dp.On("Get", mock.Anything, "A", "2024-01-15").Return(&datasetpartitions.DatasetPartitionRow{
		DatasetName: "A", PartitionKey: "2024-01-15", DeltaVersion: &dv,
	}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, "B", "2024-01-15", mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)

	actions, err := h.EvaluateUpstreamTriggers(context.Background(), "A", "2024-01-15")

	assert.NoError(t, err)
	assert.Len(t, actions, 1)
	assert.Equal(t, "B", actions[0].AssetName)
	assert.Equal(t, "upstream", actions[0].TriggerType)
}

func TestEvaluateUpstreamTriggers_PartialReadiness(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "A", AssetVersion: "v1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "B", AssetVersion: "v1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "C", AssetVersion: "v1", Notebook: "nb1", TriggerType: "upstream", Serving: true},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "C", InputName: "A", InputType: "internal"},
		{AssetName: "C", InputName: "B", InputType: "internal"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "C", Necessity: "active"},
	}, nil)

	dp.On("IsReady", mock.Anything, "A", "2024-01-15").Return(true, nil)
	dp.On("IsReady", mock.Anything, "B", "2024-01-15").Return(false, nil)

	actions, err := h.EvaluateUpstreamTriggers(context.Background(), "A", "2024-01-15")

	assert.NoError(t, err)
	assert.Empty(t, actions, "C should not trigger because B is not ready")
}

func TestEvaluateUpstreamTriggers_SkippedDownstream(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "A", AssetVersion: "v1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "B", AssetVersion: "v1", Notebook: "nb1", TriggerType: "upstream", Serving: false},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "B", InputName: "A", InputType: "internal"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "A", Necessity: "active"},
		{AssetName: "B", Necessity: "skipped"},
	}, nil)

	actions, err := h.EvaluateUpstreamTriggers(context.Background(), "A", "2024-01-15")

	assert.NoError(t, err)
	assert.Empty(t, actions, "B is skipped, should not trigger")
}

func TestEvaluateUpstreamTriggers_CacheHit(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "A", AssetVersion: "v1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "B", AssetVersion: "v1", Notebook: "nb1", TriggerType: "upstream", Serving: true},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "B", InputName: "A", InputType: "internal"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "A", Necessity: "active"},
		{AssetName: "B", Necessity: "active"},
	}, nil)
	dp.On("IsReady", mock.Anything, "A", "2024-01-15").Return(true, nil)

	ar.On("Get", mock.Anything, "B").Return(&assetregistry.AssetSpecRow{
		AssetName: "B", AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, "B").Return([]assetdeps.AssetDependencyRow{
		{AssetName: "B", InputName: "A", InputType: "internal"},
	}, nil)
	dv := int64(1)
	dp.On("Get", mock.Anything, "A", "2024-01-15").Return(&datasetpartitions.DatasetPartitionRow{
		DatasetName: "A", PartitionKey: "2024-01-15", DeltaVersion: &dv,
	}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, "B", "2024-01-15", mock.Anything).
		Return(&materializations.MaterializationRow{
			AssetName: "B", Status: "succeeded",
		}, nil)
	dp.On("MarkReady", mock.Anything, "B", "2024-01-15", int64(0)).Return(nil)

	actions, err := h.EvaluateUpstreamTriggers(context.Background(), "A", "2024-01-15")

	assert.NoError(t, err)
	assert.Empty(t, actions, "B has cache hit, should not trigger execution")
	dp.AssertCalled(t, "MarkReady", mock.Anything, "B", "2024-01-15", int64(0))
}

// ---------------------------------------------------------------------------
// Execution Plan Tests
// ---------------------------------------------------------------------------

func TestGetExecutionPlan_FilterByTrigger(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "Y", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_6h", Serving: true},
		{AssetName: "Z", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	ar.On("GetByNotebook", mock.Anything, "nb1").Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
		{AssetName: "Y", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_6h", Serving: true},
		{AssetName: "Z", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "X", Necessity: "active"},
		{AssetName: "Y", Necessity: "active"},
		{AssetName: "Z", Necessity: "active"},
	}, nil)

	ar.On("Get", mock.Anything, mock.Anything).Return(&assetregistry.AssetSpecRow{
		AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)

	plan, err := h.GetExecutionPlan(context.Background(), GetExecutionPlanRequest{
		Notebook: "nb1", TriggerType: "schedule_3h", Partition: "2024-01-15",
	})

	assert.NoError(t, err)
	assert.Len(t, plan.Assets, 2, "only schedule_3h assets should be included")
	for _, a := range plan.Assets {
		assert.Contains(t, []string{"X", "Z"}, a.AssetName)
	}
}

func TestGetExecutionPlan_SkippedAsset(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h"},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	ar.On("GetByNotebook", mock.Anything, "nb1").Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "X", Necessity: "skipped"},
	}, nil)

	plan, err := h.GetExecutionPlan(context.Background(), GetExecutionPlanRequest{
		Notebook: "nb1", TriggerType: "schedule_3h", Partition: "2024-01-15",
	})

	assert.NoError(t, err)
	assert.Len(t, plan.Assets, 1)
	assert.Equal(t, ActionSkipUnnecessary, plan.Assets[0].Action)
}

func TestGetExecutionPlan_CachedAsset(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	ar.On("GetByNotebook", mock.Anything, "nb1").Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "X", Necessity: "active"},
	}, nil)

	ar.On("Get", mock.Anything, "X").Return(&assetregistry.AssetSpecRow{
		AssetName: "X", AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, "X").Return([]assetdeps.AssetDependencyRow{}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, "X", "2024-01-15", mock.Anything).
		Return(&materializations.MaterializationRow{
			AssetName: "X", Status: "succeeded",
			ArtifactPath: "/_artifacts/X/2024-01-15/abc/",
		}, nil)

	plan, err := h.GetExecutionPlan(context.Background(), GetExecutionPlanRequest{
		Notebook: "nb1", TriggerType: "schedule_3h", Partition: "2024-01-15",
	})

	assert.NoError(t, err)
	assert.Len(t, plan.Assets, 1)
	assert.Equal(t, ActionSkipCached, plan.Assets[0].Action)
	assert.Equal(t, "/_artifacts/X/2024-01-15/abc/", plan.Assets[0].ExistingArtifact)
}

func TestGetExecutionPlan_Execute(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	ar.On("GetByNotebook", mock.Anything, "nb1").Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", Notebook: "nb1", TriggerType: "schedule_3h", Serving: true},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "X", Necessity: "active"},
	}, nil)

	ar.On("Get", mock.Anything, "X").Return(&assetregistry.AssetSpecRow{
		AssetName: "X", AssetVersion: "v1",
	}, nil)
	ad.On("GetDependencies", mock.Anything, "X").Return([]assetdeps.AssetDependencyRow{
		{AssetName: "X", InputName: "src1", InputType: "external"},
	}, nil)
	dv := int64(42)
	dp.On("Get", mock.Anything, "src1", "2024-01-15").Return(&datasetpartitions.DatasetPartitionRow{
		DatasetName: "src1", PartitionKey: "2024-01-15", DeltaVersion: &dv,
	}, nil)
	matRepo.On("GetByComputeKey", mock.Anything, "X", "2024-01-15", mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)

	plan, err := h.GetExecutionPlan(context.Background(), GetExecutionPlanRequest{
		Notebook: "nb1", TriggerType: "schedule_3h", Partition: "2024-01-15",
	})

	assert.NoError(t, err)
	assert.Len(t, plan.Assets, 1)
	assert.Equal(t, ActionExecute, plan.Assets[0].Action)
	assert.Contains(t, plan.Assets[0].ArtifactPath, "/_artifacts/X/2024-01-15/")
	assert.Equal(t, "42", plan.Assets[0].InputBindings["src1"])
	assert.Equal(t, "42", plan.SharedInputs["src1"])
}

// ---------------------------------------------------------------------------
// ReportAssetReady Tests
// ---------------------------------------------------------------------------

func TestReportAssetReady(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	dp.On("MarkReady", mock.Anything, "X", "2024-01-15", int64(5)).Return(nil)
	matRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", TriggerType: "schedule_3h"},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{}, nil)

	actions, err := h.ReportAssetReady(context.Background(), ReportAssetReadyRequest{
		AssetName:    "X",
		Partition:    "2024-01-15",
		ComputeKey:   "abc123",
		ArtifactPath: "/_artifacts/X/2024-01-15/abc123/",
		DeltaVersion: 5,
	})

	assert.NoError(t, err)
	assert.Empty(t, actions)
	dp.AssertCalled(t, "MarkReady", mock.Anything, "X", "2024-01-15", int64(5))
	matRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// Plan Stage Tests
// ---------------------------------------------------------------------------

func TestComputePlan_AddedAsset(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{}, nil)

	result, err := h.ComputePlan(context.Background(), ProposePlanRequest{
		Assets: []AssetSpecPayload{
			{Name: "new_asset", Version: "v1", Entity: "user", EntityKey: "uid",
				Notebook: "nb1", Partition: "ds", Trigger: "schedule_3h", Serving: true},
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.Changes, 1)
	assert.Equal(t, "added", result.Changes[0].ChangeType)
	assert.Equal(t, "new_asset", result.Changes[0].AssetName)
}

func TestComputePlan_ModifiedAsset(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1", EntityName: "user", Notebook: "nb1"},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{
		{AssetName: "X", InputName: "src1"},
	}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "X", Necessity: "active"},
	}, nil)

	result, err := h.ComputePlan(context.Background(), ProposePlanRequest{
		Assets: []AssetSpecPayload{
			{Name: "X", Version: "v2", Entity: "user", EntityKey: "uid",
				Notebook: "nb1", Partition: "ds", Trigger: "schedule_3h", Serving: true,
				Inputs: []InputPayload{
					{Name: "src1", Partition: "ds", Type: "external"},
					{Name: "src2", Partition: "ds", Type: "external"},
				}},
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.Changes, 1)
	assert.Equal(t, "modified", result.Changes[0].ChangeType)
	assert.True(t, result.Changes[0].CodeChanged)
	assert.Contains(t, result.Changes[0].InputsAdded, "src2")
}

func TestComputePlan_RemovedAsset(t *testing.T) {
	ar := &mockAssetRegistryRepo{}
	ad := &mockAssetDepsRepo{}
	as := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dp := &mockDatasetPartitionsRepo{}
	h := newTestHandler(ar, ad, as, matRepo, dp)

	ar.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "X", AssetVersion: "v1"},
		{AssetName: "Y", AssetVersion: "v1"},
	}, nil)
	ad.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	as.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "X", Necessity: "active"},
		{AssetName: "Y", Necessity: "active"},
	}, nil)

	result, err := h.ComputePlan(context.Background(), ProposePlanRequest{
		Assets: []AssetSpecPayload{
			{Name: "X", Version: "v1", Entity: "user", EntityKey: "uid",
				Notebook: "nb1", Partition: "ds", Trigger: "schedule_3h"},
		},
	})

	assert.NoError(t, err)

	var removedFound bool
	for _, ch := range result.Changes {
		if ch.AssetName == "Y" && ch.ChangeType == "removed" {
			removedFound = true
		}
	}
	assert.True(t, removedFound, "Y should be marked as removed")
}
