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

const clientConfigURL = "/api/v1/tenants/fs/stores/features/clientConfig"

func sampleClientConfig() model.ClientConfig {
	warm := false
	return model.ClientConfig{
		ConnectTimeoutMs:       2000,
		RequestTimeoutMs:       50,
		MinConnsPerPod:         2,
		MaxConnsPerPod:         8,
		WarmUpOnTopologyChange: &warm,
	}
}

// ── PutClientConfig ───────────────────────────────────────────────────────────

func TestPutClientConfig_Success(t *testing.T) {
	cfg := sampleClientConfig()
	m := &mockStateClient{}
	m.On("SetClientConfig", mock.Anything, "fs", "features", cfg).Return(nil)

	body, _ := json.Marshal(cfg)
	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("PUT", clientConfigURL, bytes.NewReader(body)))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, map[string]any{
		"tenant": "fs",
		"store":  "features",
		"status": "client config saved",
	}, decodeJSONObject(t, w.Body.Bytes()))
	m.AssertExpectations(t)
}

func TestPutClientConfig_InvalidBody(t *testing.T) {
	m := &mockStateClient{}

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("PUT", clientConfigURL, bytes.NewBufferString(`{"maxConnsPerPod":"many"}`)))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, decodeJSONObject(t, w.Body.Bytes()), "error")
	m.AssertNotCalled(t, "SetClientConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPutClientConfig_StateErrors(t *testing.T) {
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
			m.On("SetClientConfig", mock.Anything, "fs", "features", mock.Anything).Return(tc.stateErr)

			body, _ := json.Marshal(sampleClientConfig())
			w := httptest.NewRecorder()
			newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("PUT", clientConfigURL, bytes.NewReader(body)))

			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, map[string]any{"error": tc.wantErr}, decodeJSONObject(t, w.Body.Bytes()))
		})
	}
}

// ── GetClientConfig ───────────────────────────────────────────────────────────

func TestGetClientConfig_Success(t *testing.T) {
	cfg := sampleClientConfig()
	m := &mockStateClient{}
	m.On("GetClientConfig", mock.Anything, "fs", "features").Return(&cfg, nil)

	w := httptest.NewRecorder()
	newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", clientConfigURL, nil))

	require.Equal(t, http.StatusOK, w.Code)
	var got model.ClientConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, cfg, got)
}

func TestGetClientConfig_StateErrors(t *testing.T) {
	tests := []struct {
		name     string
		stateErr error
		wantCode int
		wantErr  string
	}{
		{"missing config maps to 404", etcdstate.ErrNotFound, http.StatusNotFound, "client config not found"},
		{"backend error maps to 500", errors.New("timeout"), http.StatusInternalServerError, "timeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &mockStateClient{}
			m.On("GetClientConfig", mock.Anything, "fs", "features").Return(nil, tc.stateErr)

			w := httptest.NewRecorder()
			newRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", clientConfigURL, nil))

			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, map[string]any{"error": tc.wantErr}, decodeJSONObject(t, w.Body.Bytes()))
		})
	}
}
