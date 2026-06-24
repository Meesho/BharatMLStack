package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
)

func TestGetTopology_Success(t *testing.T) {
	m := &mockStateClient{}
	topo := &etcdstate.TopologyState{
		ActiveVersion:   "v1",
		TopologyVersion: 2,
		Assignment:      map[string][]string{"0": {"10.0.0.1:9091"}},
	}
	m.On("GetTopology", mock.Anything, "fs", "features").Return(topo, nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/features/topology", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestGetTopology_NotFound(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetTopology", mock.Anything, "fs", "features").Return(nil, etcdstate.ErrNotFound)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/features/topology", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetTopology_InternalError(t *testing.T) {
	m := &mockStateClient{}
	m.On("GetTopology", mock.Anything, "fs", "features").Return(nil, errors.New("timeout"))

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/tenants/fs/stores/features/topology", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
