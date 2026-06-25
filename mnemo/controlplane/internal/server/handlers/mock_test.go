package handlers

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// mockStateClient is a testify mock that implements etcdstate.StateClient.
type mockStateClient struct {
	mock.Mock
}

func (m *mockStateClient) CreateStore(ctx context.Context, cfg model.StoreConfig) error {
	return m.Called(ctx, cfg).Error(0)
}

func (m *mockStateClient) GetStore(ctx context.Context, tenant, store string) (*etcdstate.StoreState, error) {
	args := m.Called(ctx, tenant, store)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*etcdstate.StoreState), args.Error(1)
}

func (m *mockStateClient) PublishVersion(ctx context.Context, tenant, store, vID string, meta model.VersionMeta) error {
	return m.Called(ctx, tenant, store, vID, meta).Error(0)
}

func (m *mockStateClient) GetVersionMeta(ctx context.Context, tenant, store, vID string) (*model.VersionMeta, error) {
	args := m.Called(ctx, tenant, store, vID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.VersionMeta), args.Error(1)
}

func (m *mockStateClient) PromoteVersion(ctx context.Context, tenant, store, vID string, assignment map[string][]string) error {
	return m.Called(ctx, tenant, store, vID, assignment).Error(0)
}

func (m *mockStateClient) RollbackStore(ctx context.Context, tenant, store string) (string, error) {
	args := m.Called(ctx, tenant, store)
	return args.String(0), args.Error(1)
}

func (m *mockStateClient) RetireVersion(ctx context.Context, tenant, store, vID string) error {
	return m.Called(ctx, tenant, store, vID).Error(0)
}

func (m *mockStateClient) GetTopology(ctx context.Context, tenant, store string) (*etcdstate.TopologyState, error) {
	args := m.Called(ctx, tenant, store)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*etcdstate.TopologyState), args.Error(1)
}

func (m *mockStateClient) ListPods(ctx context.Context, tenant, store string) (map[string]model.PodData, error) {
	args := m.Called(ctx, tenant, store)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]model.PodData), args.Error(1)
}

func (m *mockStateClient) GetClientConfig(ctx context.Context, tenant, store string) (*model.ClientConfig, error) {
	args := m.Called(ctx, tenant, store)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientConfig), args.Error(1)
}

func (m *mockStateClient) SetClientConfig(ctx context.Context, tenant, store string, cfg model.ClientConfig) error {
	return m.Called(ctx, tenant, store, cfg).Error(0)
}

func (m *mockStateClient) PutDataflow(ctx context.Context, tenant, store string, cfg model.DataflowConfig) error {
	return m.Called(ctx, tenant, store, cfg).Error(0)
}

func (m *mockStateClient) GetDataflow(ctx context.Context, tenant, store string) (*model.DataflowConfig, error) {
	args := m.Called(ctx, tenant, store)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DataflowConfig), args.Error(1)
}

func (m *mockStateClient) Close() error {
	return m.Called().Error(0)
}

func (m *mockStateClient) Health(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
