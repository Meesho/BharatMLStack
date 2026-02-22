package config

import (
	"sync"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/etcd"
	"github.com/rs/zerolog/log"
)

type Manager interface {
	RefreshShadowDeployables() error
	ListShadowDeployables(filter models.ShadowFilter) []models.ShadowDeployable
}

type Etcd struct {
	instance etcd.Etcd

	mu     sync.RWMutex
	shadow map[string][]models.ShadowDeployable
}

func NewEtcdConfig() Manager {
	return &Etcd{
		instance: etcd.Instance(),
		shadow:   make(map[string][]models.ShadowDeployable),
	}
}

func (e *Etcd) GetEtcdInstance() *EtcdRegistry {
	instance, ok := e.instance.GetConfigInstance().(*EtcdRegistry)
	if !ok {
		log.Panic().Msg("invalid etcd instance")
	}
	return instance
}

func (e *Etcd) RefreshShadowDeployables() error {
	instance := e.GetEtcdInstance()

	tmp := make(map[string][]models.ShadowDeployable)
	total := 0
	for env, byName := range instance.ShadowDeployables {
		items := make([]models.ShadowDeployable, 0, len(byName))
		for _, item := range byName {
			items = append(items, item)
		}
		tmp[env] = items
		total += len(items)
	}

	e.mu.Lock()
	e.shadow = tmp
	e.mu.Unlock()
	log.Info().
		Int("env_count", len(tmp)).
		Int("deployable_count", total).
		Msg("shadow deployables cache refreshed from watcher/config bridge")
	return nil
}

func (e *Etcd) ListShadowDeployables(filter models.ShadowFilter) []models.ShadowDeployable {
	e.mu.RLock()
	defer e.mu.RUnlock()

	items := e.shadow[filter.Env]
	out := make([]models.ShadowDeployable, 0, len(items))
	for _, item := range items {
		inUse := item.State == rmtypes.ShadowStateProcured
		if filter.InUse != nil && *filter.InUse != inUse {
			continue
		}
		if filter.NodePool != "" && filter.NodePool != item.NodePool {
			continue
		}
		out = append(out, item)
	}
	return out
}
