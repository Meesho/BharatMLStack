package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

const dataflowURL = "/api/v1/tenants/fs/stores/features/dataflow"

func sampleDataflow() model.DataflowConfig {
	return model.DataflowConfig{
		SourcePath:   "gs://bucket/src",
		GcsOutRoot:   "gs://bucket/out",
		NumShards:    4,
		TargetSstMB:  256,
		JobID:        "job-1",
		AutoPromote:  true,
		KeepVersions: 3,
	}
}

func decodeJSONObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var got map[string]any
	require.NoError(t, json.Unmarshal(body, &got))
	return got
}

// ── PutDataflow ───────────────────────────────────────────────────────────────

func TestPutDataflow_Success(t *testing.T) {
	cfg := sampleDataflow()
	m := &mockStateClient{}
	m.On("PutDataflow", mock.Anything, "fs", "features", cfg).Return(nil)

	body, _ := json.Marshal(cfg)
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("PUT", dataflowURL, bytes.NewReader(body)))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, map[string]any{
		"tenant": "fs",
		"store":  "features",
		"status": "dataflow config saved",
	}, decodeJSONObject(t, w.Body.Bytes()))
	m.AssertExpectations(t)
}

func TestPutDataflow_InvalidBody(t *testing.T) {
	m := &mockStateClient{}

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("PUT", dataflowURL, bytes.NewBufferString(`{"numShards":`)))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, decodeJSONObject(t, w.Body.Bytes()), "error")
	m.AssertNotCalled(t, "PutDataflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPutDataflow_StateErrors(t *testing.T) {
	tests := []struct {
		name     string
		stateErr error
		wantCode int
		wantErr  string
	}{
		{"store not found maps to 404", etcdstate.ErrNotFound, http.StatusNotFound, "store not found"},
		{"wrapped not found maps to 404", fmt.Errorf("lookup: %w", etcdstate.ErrNotFound), http.StatusNotFound, "store not found"},
		{"backend error maps to 500", errors.New("etcd down"), http.StatusInternalServerError, "etcd down"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &mockStateClient{}
			m.On("PutDataflow", mock.Anything, "fs", "features", mock.Anything).Return(tc.stateErr)

			body, _ := json.Marshal(sampleDataflow())
			w := httptest.NewRecorder()
			newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("PUT", dataflowURL, bytes.NewReader(body)))

			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, map[string]any{"error": tc.wantErr}, decodeJSONObject(t, w.Body.Bytes()))
		})
	}
}

// ── GetDataflow ───────────────────────────────────────────────────────────────

func TestGetDataflow_Success(t *testing.T) {
	cfg := sampleDataflow()
	m := &mockStateClient{}
	m.On("GetDataflow", mock.Anything, "fs", "features").Return(&cfg, nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", dataflowURL, nil))

	require.Equal(t, http.StatusOK, w.Code)
	var got model.DataflowConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, cfg, got)
}

func TestGetDataflow_StateErrors(t *testing.T) {
	tests := []struct {
		name     string
		stateErr error
		wantCode int
		wantErr  string
	}{
		{"missing config maps to 404", etcdstate.ErrNotFound, http.StatusNotFound, "dataflow config not found"},
		{"backend error maps to 500", errors.New("timeout"), http.StatusInternalServerError, "timeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &mockStateClient{}
			m.On("GetDataflow", mock.Anything, "fs", "features").Return(nil, tc.stateErr)

			w := httptest.NewRecorder()
			newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", dataflowURL, nil))

			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, map[string]any{"error": tc.wantErr}, decodeJSONObject(t, w.Body.Bytes()))
		})
	}
}
