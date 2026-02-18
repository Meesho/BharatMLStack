package application

import (
	"context"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/ports"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

type ShadowService struct {
	store ports.ShadowStateStore
}

func NewShadowService(store ports.ShadowStateStore) *ShadowService {
	return &ShadowService{store: store}
}

func (s *ShadowService) List(ctx context.Context, env string, filter models.ShadowFilter) ([]models.ShadowDeployable, error) {
	return s.store.List(ctx, env, filter)
}

func (s *ShadowService) Procure(ctx context.Context, env, name, runID, plan string) (models.ShadowDeployable, bool, error) {
	return s.store.Procure(ctx, env, name, runID, plan)
}

func (s *ShadowService) Release(ctx context.Context, env, name, runID string) (models.ShadowDeployable, bool, error) {
	return s.store.Release(ctx, env, name, runID)
}

func (s *ShadowService) ChangeMinPodCount(ctx context.Context, env, name string, action rmtypes.Action, count int) (models.ShadowDeployable, error) {
	return s.store.ChangeMinPodCount(ctx, env, name, action, count)
}
