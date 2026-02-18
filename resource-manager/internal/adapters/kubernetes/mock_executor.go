package kubernetes

import (
	"context"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

// MockExecutor is a placeholder integration until client-go is wired.
type MockExecutor struct{}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{}
}

func (m *MockExecutor) CreateDeployable(_ context.Context, _ string, _ models.DeployableSpec) error {
	return nil
}

func (m *MockExecutor) LoadModel(_ context.Context, _ string, _ models.ModelLoadSpec) error {
	return nil
}

func (m *MockExecutor) TriggerJob(_ context.Context, _ string, _ models.JobSpec) error {
	return nil
}

func (m *MockExecutor) RestartDeployable(_ context.Context, _ string, _ models.RestartSpec) error {
	return nil
}
