package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// ── PublishVersion ────────────────────────────────────────────────────────────

func TestPublishVersion_Success(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 3}}, nil)
	m.On("PublishVersion", mock.Anything, "fs", "features", "v1", mock.AnythingOfType("model.VersionMeta")).Return(nil)

	body, _ := json.Marshal(PublishVersionRequest{Date: "20260603", Run: "001"})
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/publish", bytes.NewReader(body)))

	assert.Equal(t, http.StatusCreated, w.Code)
	m.AssertExpectations(t)
}

func TestPublishVersion_BadBody(t *testing.T) {
	w := httptest.NewRecorder()
	newRouter(New(&mockStateClient{})).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/publish", bytes.NewBufferString(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPublishVersion_StoreNotFound(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(nil, etcdstate.ErrNotFound)

	body, _ := json.Marshal(PublishVersionRequest{Date: "20260603", Run: "001"})
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/publish", bytes.NewReader(body)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPublishVersion_GetStoreError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(nil, errors.New("timeout"))

	body, _ := json.Marshal(PublishVersionRequest{Date: "20260603", Run: "001"})
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/publish", bytes.NewReader(body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPublishVersion_AlreadyExists(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 3}}, nil)
	m.On("PublishVersion", mock.Anything, "fs", "features", "v1", mock.Anything).Return(etcdstate.ErrAlreadyExists)

	body, _ := json.Marshal(PublishVersionRequest{Date: "20260603", Run: "001"})
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/publish", bytes.NewReader(body)))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPublishVersion_PublishError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 3}}, nil)
	m.On("PublishVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("etcd error"))

	body, _ := json.Marshal(PublishVersionRequest{Date: "20260603", Run: "001"})
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/publish", bytes.NewReader(body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── PromoteVersion ────────────────────────────────────────────────────────────

func TestPromoteVersion_Success(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 2}}, nil)
	m.On("ListPods", mock.Anything, "fs", "features").Return(
		map[string]model.PodData{
			"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
			"fs-features-shard-1-0": {PodIP: "10.0.0.2", WarmVersions: []string{"v1"}},
		}, nil)
	m.On("PromoteVersion", mock.Anything, "fs", "features", "v1", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestPromoteVersion_StoreNotFound(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(nil, etcdstate.ErrNotFound)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPromoteVersion_GetStoreError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(nil, errors.New("timeout"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPromoteVersion_ListPodsError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 1}}, nil)
	m.On("ListPods", mock.Anything, "fs", "features").Return(nil, errors.New("timeout"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPromoteVersion_CoverageIncomplete(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 2}}, nil)
	// Only shard 0 is warm, shard 1 is missing
	m.On("ListPods", mock.Anything, "fs", "features").Return(
		map[string]model.PodData{
			"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		}, nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPromoteVersion_CASConflict(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 1}}, nil)
	m.On("ListPods", mock.Anything, "fs", "features").Return(
		map[string]model.PodData{
			"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		}, nil)
	m.On("PromoteVersion", mock.Anything, "fs", "features", "v1", mock.Anything).Return(etcdstate.ErrCASConflict)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPromoteVersion_VersionNotFound(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 1}}, nil)
	m.On("ListPods", mock.Anything, "fs", "features").Return(
		map[string]model.PodData{
			"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		}, nil)
	m.On("PromoteVersion", mock.Anything, "fs", "features", "v1", mock.Anything).Return(etcdstate.ErrNotFound)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPromoteVersion_PromoteError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetStore", mock.Anything, "fs", "features").Return(
		&etcdstate.StoreState{Config: model.StoreConfig{ShardCount: 1}}, nil)
	m.On("ListPods", mock.Anything, "fs", "features").Return(
		map[string]model.PodData{
			"fs-features-shard-0-0": {PodIP: "10.0.0.1", WarmVersions: []string{"v1"}},
		}, nil)
	m.On("PromoteVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("etcd error"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/promote", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Rollback ──────────────────────────────────────────────────────────────────

func TestRollback_Success(t *testing.T) {
	m := &mockStateClient{}
	m.On("RollbackStore", mock.Anything, "fs", "features").Return("v1", nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/rollback", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRollback_NoRollbackVersion(t *testing.T) {
	m := &mockStateClient{}
	m.On("RollbackStore", mock.Anything, "fs", "features").Return("", etcdstate.ErrNoRollback)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/rollback", nil))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRollback_CASConflict(t *testing.T) {
	m := &mockStateClient{}
	m.On("RollbackStore", mock.Anything, "fs", "features").Return("", etcdstate.ErrCASConflict)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/rollback", nil))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRollback_InternalError(t *testing.T) {
	m := &mockStateClient{}
	m.On("RollbackStore", mock.Anything, "fs", "features").Return("", errors.New("timeout"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/rollback", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── RetireVersion ─────────────────────────────────────────────────────────────

func TestRetireVersion_Success(t *testing.T) {
	m := &mockStateClient{}
	m.On("RetireVersion", mock.Anything, "fs", "features", "v1").Return(nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/retire", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRetireVersion_NotFound(t *testing.T) {
	m := &mockStateClient{}
	m.On("RetireVersion", mock.Anything, "fs", "features", "v1").Return(etcdstate.ErrNotFound)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/retire", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRetireVersion_InternalError(t *testing.T) {
	m := &mockStateClient{}
	m.On("RetireVersion", mock.Anything, "fs", "features", "v1").Return(errors.New("timeout"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest(
		"POST", "/api/v1/tenants/fs/stores/features/versions/v1/retire", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
