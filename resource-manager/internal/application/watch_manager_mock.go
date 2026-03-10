package application

import (
	"context"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type MockWatchManager struct{}

func NewMockWatchManager() *MockWatchManager {
	return &MockWatchManager{}
}

func (m *MockWatchManager) Watch(ctx context.Context, _ models.WatchIntent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return nil
	}
}
