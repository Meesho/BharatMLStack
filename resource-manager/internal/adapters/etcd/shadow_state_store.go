package etcd

import (
	"context"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmerrors "github.com/Meesho/BharatMLStack/resource-manager/internal/errors"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

type MemoryShadowStateStore struct {
	mu    sync.RWMutex
	state map[string]map[string]models.ShadowDeployable
}

func NewMemoryShadowStateStore(seed map[string][]models.ShadowDeployable) *MemoryShadowStateStore {
	store := &MemoryShadowStateStore{
		state: make(map[string]map[string]models.ShadowDeployable),
	}
	for env, items := range seed {
		if _, ok := store.state[env]; !ok {
			store.state[env] = make(map[string]models.ShadowDeployable)
		}
		for _, item := range items {
			store.state[env][item.Name] = item
		}
	}
	return store
}

func (s *MemoryShadowStateStore) List(_ context.Context, env string, filter models.ShadowFilter) ([]models.ShadowDeployable, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return nil, rmerrors.ErrUnsupportedEnv
	}
	env = string(rmtypes.NormalizePoolEnv(env))
	s.mu.RLock()
	defer s.mu.RUnlock()

	byName := s.state[env]
	result := make([]models.ShadowDeployable, 0, len(byName))
	for _, item := range byName {
		inUse := item.State == rmtypes.ShadowStateProcured
		if filter.InUse != nil && *filter.InUse != inUse {
			continue
		}
		if filter.NodePool != "" && filter.NodePool != item.NodePool {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *MemoryShadowStateStore) Procure(_ context.Context, env, name, runID, plan string) (models.ShadowDeployable, bool, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return models.ShadowDeployable{}, false, rmerrors.ErrUnsupportedEnv
	}
	env = string(rmtypes.NormalizePoolEnv(env))
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.state[env][name]
	if !ok {
		return models.ShadowDeployable{}, false, rmerrors.ErrNotFound
	}
	if item.State != rmtypes.ShadowStateFree {
		return item, true, nil
	}

	item.State = rmtypes.ShadowStateProcured
	item.Owner = &models.Owner{
		WorkflowRunID: runID,
		WorkflowPlan:  plan,
		ProcuredAt:    time.Now().UTC(),
	}
	item.Version++
	item.LastUpdatedAt = time.Now().UTC()

	s.state[env][name] = item
	return item, false, nil
}

func (s *MemoryShadowStateStore) Release(_ context.Context, env, name, runID string) (models.ShadowDeployable, bool, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return models.ShadowDeployable{}, false, rmerrors.ErrUnsupportedEnv
	}
	env = string(rmtypes.NormalizePoolEnv(env))
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.state[env][name]
	if !ok {
		return models.ShadowDeployable{}, false, rmerrors.ErrNotFound
	}
	if item.State != rmtypes.ShadowStateProcured {
		return item, true, nil
	}
	if item.Owner == nil || item.Owner.WorkflowRunID != runID {
		return item, false, rmerrors.ErrInvalidOwner
	}

	item.State = rmtypes.ShadowStateFree
	item.Owner = nil
	item.MinPodCount = 0
	item.Version++
	item.LastUpdatedAt = time.Now().UTC()

	s.state[env][name] = item
	return item, false, nil
}

func (s *MemoryShadowStateStore) ChangeMinPodCount(_ context.Context, env, name string, action rmtypes.Action, count int) (models.ShadowDeployable, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return models.ShadowDeployable{}, rmerrors.ErrUnsupportedEnv
	}
	env = string(rmtypes.NormalizePoolEnv(env))
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.state[env][name]
	if !ok {
		return models.ShadowDeployable{}, rmerrors.ErrNotFound
	}

	switch action {
	case rmtypes.ActionIncrease:
		item.MinPodCount += count
	case rmtypes.ActionDecrease:
		item.MinPodCount -= count
		if item.MinPodCount < 0 {
			item.MinPodCount = 0
		}
	case rmtypes.ActionResetTo0:
		item.MinPodCount = 0
	default:
		return models.ShadowDeployable{}, rmerrors.ErrInvalidAction
	}

	item.Version++
	item.LastUpdatedAt = time.Now().UTC()
	s.state[env][name] = item
	return item, nil
}
