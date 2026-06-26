package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

func init() { gin.SetMode(gin.TestMode) }

// mockState is a minimal etcdstate.StateClient for server-level tests.
type mockState struct{ mock.Mock }

func (m *mockState) CreateStore(ctx context.Context, cfg model.StoreConfig) error {
	return m.Called(ctx, cfg).Error(0)
}
func (m *mockState) GetStore(ctx context.Context, t, s string) (*etcdstate.StoreState, error) {
	args := m.Called(ctx, t, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*etcdstate.StoreState), args.Error(1)
}
func (m *mockState) PublishVersion(ctx context.Context, t, s, v string, meta model.VersionMeta) error {
	return m.Called(ctx, t, s, v, meta).Error(0)
}
func (m *mockState) GetVersionMeta(ctx context.Context, t, s, v string) (*model.VersionMeta, error) {
	args := m.Called(ctx, t, s, v)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.VersionMeta), args.Error(1)
}
func (m *mockState) PromoteVersion(ctx context.Context, t, s, v string, a map[string][]string) error {
	return m.Called(ctx, t, s, v, a).Error(0)
}
func (m *mockState) RollbackStore(ctx context.Context, t, s string) (string, error) {
	args := m.Called(ctx, t, s)
	return args.String(0), args.Error(1)
}
func (m *mockState) RetireVersion(ctx context.Context, t, s, v string) error {
	return m.Called(ctx, t, s, v).Error(0)
}
func (m *mockState) GetTopology(ctx context.Context, t, s string) (*etcdstate.TopologyState, error) {
	args := m.Called(ctx, t, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*etcdstate.TopologyState), args.Error(1)
}
func (m *mockState) ListPods(ctx context.Context, t, s string) (map[string]model.PodData, error) {
	args := m.Called(ctx, t, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]model.PodData), args.Error(1)
}
func (m *mockState) ListVersions(ctx context.Context, t, s string) (map[string]*model.VersionMeta, error) {
	args := m.Called(ctx, t, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*model.VersionMeta), args.Error(1)
}
func (m *mockState) GetClientConfig(ctx context.Context, t, s string) (*model.ClientConfig, error) {
	args := m.Called(ctx, t, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientConfig), args.Error(1)
}
func (m *mockState) SetClientConfig(ctx context.Context, t, s string, cfg model.ClientConfig) error {
	return m.Called(ctx, t, s, cfg).Error(0)
}
func (m *mockState) PutDataflow(ctx context.Context, t, s string, cfg model.DataflowConfig) error {
	return m.Called(ctx, t, s, cfg).Error(0)
}
func (m *mockState) GetDataflow(ctx context.Context, t, s string) (*model.DataflowConfig, error) {
	args := m.Called(ctx, t, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DataflowConfig), args.Error(1)
}
func (m *mockState) Close() error                     { return m.Called().Error(0) }
func (m *mockState) Health(ctx context.Context) error { return m.Called(ctx).Error(0) }

func TestNew_ReturnsServer(t *testing.T) {
	s := New(":0", &mockState{})
	assert.NotNil(t, s)
	assert.NotNil(t, s.Handler())
}

func TestServer_HealthEndpoint(t *testing.T) {
	s := New(":0", &mockState{})
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_RouteRegistration(t *testing.T) {
	routes := []struct {
		method string
		path   string
		code   int // expected without a real handler (404 = not registered, 4xx+ = registered)
	}{
		{"POST", "/api/v1/tenants/t/stores", http.StatusBadRequest},           // OnboardStore — missing body
		{"GET", "/api/v1/tenants/t/stores/s", http.StatusInternalServerError}, // GetStore — mock returns error
		{"GET", "/api/v1/tenants/t/stores/s/topology", http.StatusInternalServerError},
	}

	ms := &mockState{}
	ms.On("GetStore", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
	ms.On("GetTopology", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
	s := New(":0", ms)

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route should be registered")
		})
	}
}

func TestServer_Run_GracefulShutdown(t *testing.T) {
	s := New(":0", &mockState{})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.Run(ctx)
	require.NoError(t, err)
}

func TestServer_Run_ListenError(t *testing.T) {
	// Hold a TCP listener on a port so the server cannot bind to it.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	s := New(l.Addr().String(), &mockState{})
	err = s.Run(context.Background())
	require.Error(t, err) // "address already in use"
}
