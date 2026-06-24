package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

func init() { gin.SetMode(gin.TestMode) }

func newRouter(h *Handlers) *gin.Engine {
	r := gin.New()
	r.POST("/api/v1/tenants/:tenant/stores", h.OnboardStore)
	r.GET("/api/v1/tenants/:tenant/stores/:store", h.GetStore)
	r.POST("/api/v1/tenants/:tenant/stores/:store/versions/:vId/publish", h.PublishVersion)
	r.POST("/api/v1/tenants/:tenant/stores/:store/versions/:vId/promote", h.PromoteVersion)
	r.POST("/api/v1/tenants/:tenant/stores/:store/rollback", h.Rollback)
	r.POST("/api/v1/tenants/:tenant/stores/:store/versions/:vId/retire", h.RetireVersion)
	r.GET("/api/v1/tenants/:tenant/stores/:store/topology", h.GetTopology)
	return r
}

// ── OnboardStore ──────────────────────────────────────────────────────────────

func TestOnboardStore_Success(t *testing.T) {
	m := &mockStateClient{}
	m.On("CreateStore", mock.Anything, model.StoreConfig{
		Tenant: "fs", Store: "features", EntityKey: "catalog_id", ShardCount: 10,
	}).Return(nil)

	body, _ := json.Marshal(CreateStoreRequest{Name: "features", EntityKey: "catalog_id", ShardCount: 10})
	w := httptest.NewRecorder()
	r := newRouter(New(m))
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/tenants/fs/stores", bytes.NewReader(body)))

	assert.Equal(t, http.StatusCreated, w.Code)
	m.AssertExpectations(t)
}

func TestOnboardStore_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRouter(New(&mockStateClient{}))
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/tenants/fs/stores", bytes.NewBufferString(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnboardStore_AlreadyExists(t *testing.T) {
	m := &mockStateClient{}
	m.On("CreateStore", mock.Anything, mock.Anything).Return(etcdstate.ErrAlreadyExists)

	body, _ := json.Marshal(CreateStoreRequest{Name: "f", EntityKey: "k", ShardCount: 1})
	w := httptest.NewRecorder()
	r := newRouter(New(m))
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/tenants/fs/stores", bytes.NewReader(body)))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestOnboardStore_InternalError(t *testing.T) {
	m := &mockStateClient{}
	m.On("CreateStore", mock.Anything, mock.Anything).Return(errors.New("etcd down"))

	body, _ := json.Marshal(CreateStoreRequest{Name: "f", EntityKey: "k", ShardCount: 1})
	w := httptest.NewRecorder()
	r := newRouter(New(m))
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/tenants/fs/stores", bytes.NewReader(body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetStore ──────────────────────────────────────────────────────────────────

func TestGetStore_Success(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(&etcdstate.StoreState{
		Config: model.StoreConfig{Tenant: "fs", Store: "features", ShardCount: 10},
	}, nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/features", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestGetStore_NotFound(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "missing").Return(nil, etcdstate.ErrNotFound)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/missing", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetStore_InternalError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(nil, errors.New("timeout"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/features", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetStore_ResponseBody(t *testing.T) {
	m := &mockStateClient{}
	expected := &etcdstate.StoreState{
		Config:          model.StoreConfig{Tenant: "fs", Store: "features", EntityKey: "cat", ShardCount: 5},
		ActiveVersion:   "v1",
		TopologyVersion: 2,
	}
	m.On("GetStore", mock.Anything, "fs", "features").Return(expected, nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/features", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var got etcdstate.StoreState
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "v1", got.ActiveVersion)
	assert.Equal(t, int64(2), got.TopologyVersion)
}
