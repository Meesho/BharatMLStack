package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/featurecomputeengine/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetdeps"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetregistry"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetstate"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/datasetpartitions"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/materializations"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock Repositories (mirrors handler_test.go pattern)
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
func (m *mockAssetStateRepo) SetServingOverride(ctx context.Context, name string, serving bool, reason, by string) error {
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

type testFixture struct {
	ctrl    *Controller
	arRepo  *mockAssetRegistryRepo
	adRepo  *mockAssetDepsRepo
	asRepo  *mockAssetStateRepo
	matRepo *mockMaterializationsRepo
	dpRepo  *mockDatasetPartitionsRepo
	router  *gin.Engine
}

func newTestFixture() *testFixture {
	gin.SetMode(gin.TestMode)

	arRepo := &mockAssetRegistryRepo{}
	adRepo := &mockAssetDepsRepo{}
	asRepo := &mockAssetStateRepo{}
	matRepo := &mockMaterializationsRepo{}
	dpRepo := &mockDatasetPartitionsRepo{}

	cfg := handler.Config{
		ArtifactBasePath: "/_artifacts",
		ServingBasePath:  "/serving",
		DefaultPartition: "ds",
	}

	h := handler.New(cfg, arRepo, adRepo, asRepo, matRepo, dpRepo)
	ctrl := New(h)

	r := gin.New()
	fce := r.Group("/api/v1/fce")
	{
		fce.POST("/assets/register", ctrl.RegisterAssets)
		fce.POST("/execution-plan", ctrl.GetExecutionPlan)
		fce.POST("/assets/ready", ctrl.ReportAssetReady)
		fce.POST("/assets/failed", ctrl.ReportAssetFailed)
		fce.GET("/necessity", ctrl.GetAllNecessity)
		fce.GET("/necessity/:asset_name", ctrl.GetNecessity)
		fce.POST("/serving/override", ctrl.SetServingOverride)
		fce.DELETE("/serving/override/:asset_name", ctrl.ClearServingOverride)
		fce.POST("/necessity/resolve", ctrl.ResolveNecessity)
		fce.GET("/lineage", ctrl.GetFullLineage)
		fce.GET("/lineage/:asset_name/upstream", ctrl.GetUpstream)
		fce.GET("/lineage/:asset_name/downstream", ctrl.GetDownstream)
		fce.POST("/plan", ctrl.ComputePlan)
		fce.POST("/plan/apply", ctrl.ApplyPlan)
		fce.POST("/datasets/ready", ctrl.MarkDatasetReady)
		fce.GET("/datasets/:dataset_name/partitions", ctrl.GetDatasetPartitions)
	}

	return &testFixture{
		ctrl:    ctrl,
		arRepo:  arRepo,
		adRepo:  adRepo,
		asRepo:  asRepo,
		matRepo: matRepo,
		dpRepo:  dpRepo,
		router:  r,
	}
}

func doRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseBody(w *httptest.ResponseRecorder) map[string]interface{} {
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRegisterAssets_BadRequest_EmptyBody(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/assets/register", map[string]interface{}{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterAssets_BadRequest_EmptyAssets(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/assets/register",
		map[string]interface{}{"assets": []interface{}{}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterAssets_Success(t *testing.T) {
	f := newTestFixture()
	f.arRepo.On("BulkUpsert", mock.Anything, mock.Anything).Return(nil)
	f.adRepo.On("ReplaceForAsset", mock.Anything, "fg.test", mock.Anything).Return(nil)
	// ResolveNecessity calls: GetAll specs, GetAllEdges, loadServingOverrides(GetAll state), BulkUpsert state
	f.arRepo.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "fg.test", Serving: true},
	}, nil)
	f.adRepo.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	f.asRepo.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{}, nil)
	f.asRepo.On("BulkUpsert", mock.Anything, mock.Anything).Return(nil)

	body := map[string]interface{}{
		"assets": []map[string]interface{}{
			{
				"name":       "fg.test",
				"version":    "v1",
				"entity":     "user",
				"entity_key": "user_id",
				"notebook":   "user_notebook",
				"partition":  "ds",
				"trigger":    "schedule",
				"serving":    true,
			},
		},
	}
	w := doRequest(f.router, "POST", "/api/v1/fce/assets/register", body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseBody(w)
	assert.Equal(t, float64(1), resp["registered"])
}

func TestGetExecutionPlan_BadRequest_MissingFields(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/execution-plan",
		map[string]interface{}{"notebook": "test"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportAssetReady_BadRequest(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/assets/ready",
		map[string]interface{}{"asset_name": "test"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportAssetFailed_BadRequest(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/assets/failed",
		map[string]interface{}{"asset_name": "test"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportAssetFailed_Success(t *testing.T) {
	f := newTestFixture()
	f.matRepo.On("GetLatest", mock.Anything, "fg.test", "2024-01-01").Return(
		&materializations.MaterializationRow{ID: 42}, nil)
	f.matRepo.On("UpdateStatus", mock.Anything, int64(42), "failed").Return(nil)

	w := doRequest(f.router, "POST", "/api/v1/fce/assets/failed",
		map[string]interface{}{
			"asset_name": "fg.test",
			"partition":  "2024-01-01",
			"error":      "spark OOM",
		})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAllNecessity_Success(t *testing.T) {
	f := newTestFixture()
	f.asRepo.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{
		{AssetName: "fg.a", Necessity: "active"},
		{AssetName: "fg.b", Necessity: "skipped"},
	}, nil)

	w := doRequest(f.router, "GET", "/api/v1/fce/necessity", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseBody(w)
	assert.Equal(t, "active", resp["fg.a"])
	assert.Equal(t, "skipped", resp["fg.b"])
}

func TestGetNecessity_Found(t *testing.T) {
	f := newTestFixture()
	f.asRepo.On("Get", mock.Anything, "fg.a").Return(
		&assetstate.AssetNecessityRow{AssetName: "fg.a", Necessity: "active"}, nil)

	w := doRequest(f.router, "GET", "/api/v1/fce/necessity/fg.a", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseBody(w)
	assert.Equal(t, "active", resp["necessity"])
}

func TestGetNecessity_NotFound(t *testing.T) {
	f := newTestFixture()
	f.asRepo.On("Get", mock.Anything, "fg.missing").Return(nil, gorm.ErrRecordNotFound)

	w := doRequest(f.router, "GET", "/api/v1/fce/necessity/fg.missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetServingOverride_BadRequest(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/serving/override",
		map[string]interface{}{"asset_name": "fg.a"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetServingOverride_Success(t *testing.T) {
	f := newTestFixture()
	f.asRepo.On("SetServingOverride", mock.Anything, "fg.a", true, "testing", "user@test.com").Return(nil)

	w := doRequest(f.router, "POST", "/api/v1/fce/serving/override",
		map[string]interface{}{
			"asset_name": "fg.a",
			"serving":    true,
			"reason":     "testing",
			"updated_by": "user@test.com",
		})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClearServingOverride_Success(t *testing.T) {
	f := newTestFixture()
	f.asRepo.On("ClearServingOverride", mock.Anything, "fg.a").Return(nil)

	w := doRequest(f.router, "DELETE", "/api/v1/fce/serving/override/fg.a", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetFullLineage_Success(t *testing.T) {
	f := newTestFixture()
	f.arRepo.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{
		{AssetName: "fg.a", Serving: true, EntityName: "user"},
	}, nil)
	f.adRepo.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)

	w := doRequest(f.router, "GET", "/api/v1/fce/lineage", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseBody(w)
	assert.NotNil(t, resp["nodes"])
}

func TestGetUpstream_NotFound(t *testing.T) {
	f := newTestFixture()
	f.arRepo.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{}, nil)
	f.adRepo.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)

	w := doRequest(f.router, "GET", "/api/v1/fce/lineage/fg.missing/upstream", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetDownstream_NotFound(t *testing.T) {
	f := newTestFixture()
	f.arRepo.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{}, nil)
	f.adRepo.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)

	w := doRequest(f.router, "GET", "/api/v1/fce/lineage/fg.missing/downstream", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestComputePlan_BadRequest_Empty(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/plan",
		map[string]interface{}{"assets": []interface{}{}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestApplyPlan_BadRequest_Empty(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/plan/apply",
		map[string]interface{}{"assets": []interface{}{}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkDatasetReady_BadRequest(t *testing.T) {
	f := newTestFixture()
	w := doRequest(f.router, "POST", "/api/v1/fce/datasets/ready",
		map[string]interface{}{"partition": "2024-01-01"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkDatasetReady_Success(t *testing.T) {
	f := newTestFixture()
	f.dpRepo.On("MarkReady", mock.Anything, "silver.orders", "2024-01-01", int64(5)).Return(nil)
	// EvaluateUpstreamTriggers: BuildDAG
	f.arRepo.On("GetAll", mock.Anything).Return([]assetregistry.AssetSpecRow{}, nil)
	f.adRepo.On("GetAllEdges", mock.Anything).Return([]assetdeps.AssetDependencyRow{}, nil)
	// loadNecessityMap
	f.asRepo.On("GetAll", mock.Anything).Return([]assetstate.AssetNecessityRow{}, nil)

	w := doRequest(f.router, "POST", "/api/v1/fce/datasets/ready",
		map[string]interface{}{
			"dataset_name":  "silver.orders",
			"partition":     "2024-01-01",
			"delta_version": 5,
		})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetDatasetPartitions_Success(t *testing.T) {
	f := newTestFixture()
	dv := int64(3)
	f.dpRepo.On("GetPartitions", mock.Anything, "silver.orders").Return(
		[]datasetpartitions.DatasetPartitionRow{
			{DatasetName: "silver.orders", PartitionKey: "2024-01-01", DeltaVersion: &dv, IsReady: true},
		}, nil)

	w := doRequest(f.router, "GET", "/api/v1/fce/datasets/silver.orders/partitions", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var parts []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parts)
	assert.Len(t, parts, 1)
	assert.Equal(t, "silver.orders", parts[0]["dataset_name"])
}

func TestInvalidJSON_Returns400(t *testing.T) {
	f := newTestFixture()
	req := httptest.NewRequest("POST", "/api/v1/fce/assets/register",
		bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
