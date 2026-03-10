package application

import (
	"context"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type MockCallbackDispatcher struct{}

func NewMockCallbackDispatcher() *MockCallbackDispatcher {
	return &MockCallbackDispatcher{}
}

func (m *MockCallbackDispatcher) Dispatch(_ context.Context, _ models.WatchIntent, _ error) error {
	return nil
}
