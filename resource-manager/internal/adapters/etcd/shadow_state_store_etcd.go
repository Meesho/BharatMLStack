package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmerrors "github.com/Meesho/BharatMLStack/resource-manager/internal/errors"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdShadowStateStore struct {
	client *clientv3.Client
}

func NewEtcdShadowStateStore(client *clientv3.Client) *EtcdShadowStateStore {
	return &EtcdShadowStateStore{client: client}
}

func (s *EtcdShadowStateStore) List(ctx context.Context, env string, filter models.ShadowFilter) ([]models.ShadowDeployable, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return nil, rmerrors.ErrUnsupportedEnv
	}
	prefix := shadowEnvPath(env) + "/"
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	result := make([]models.ShadowDeployable, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var item models.ShadowDeployable
		if err := json.Unmarshal(kv.Value, &item); err != nil {
			return nil, fmt.Errorf("invalid shadow deployable at %s: %w", string(kv.Key), err)
		}
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

func (s *EtcdShadowStateStore) Procure(ctx context.Context, env, name, runID, plan string) (models.ShadowDeployable, bool, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return models.ShadowDeployable{}, false, rmerrors.ErrUnsupportedEnv
	}
	key := shadowKey(env, name)
	log.Debug().Str("env", env).Str("name", name).Str("run_id", runID).Str("key", key).Msg("procure requested")

	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return models.ShadowDeployable{}, false, err
	}
	if len(resp.Kvs) == 0 {
		return models.ShadowDeployable{}, false, rmerrors.ErrNotFound
	}

	kv := resp.Kvs[0]
	var current models.ShadowDeployable
	if err := json.Unmarshal(kv.Value, &current); err != nil {
		return models.ShadowDeployable{}, false, err
	}
	log.Debug().
		Str("env", env).
		Str("name", name).
		Str("state_before", string(current.State)).
		Int64("version_before", current.Version).
		Int64("mod_revision_before", kv.ModRevision).
		Msg("procure loaded current deployable state from etcd")
	if current.State != rmtypes.ShadowStateFree {
		log.Info().
			Str("env", env).
			Str("name", name).
			Str("state_before", string(current.State)).
			Msg("procure skipped because deployable is not FREE")
		return current, true, nil
	}

	updated := current
	updated.State = rmtypes.ShadowStateProcured
	updated.Owner = &models.Owner{
		WorkflowRunID: runID,
		WorkflowPlan:  plan,
		ProcuredAt:    time.Now().UTC(),
	}
	updated.Version = current.Version + 1
	updated.LastUpdatedAt = time.Now().UTC()

	updatedRaw, err := json.Marshal(updated)
	if err != nil {
		return models.ShadowDeployable{}, false, err
	}

	cas, err := compareAndSwap(
		ctx,
		s.client,
		key,
		kv.ModRevision,
		string(kv.Value),
		string(updatedRaw),
	)
	if err != nil {
		return models.ShadowDeployable{}, false, err
	}
	if !cas.Applied {
		log.Info().
			Str("env", env).
			Str("name", name).
			Str("run_id", runID).
			Msg("procure CAS not applied (conflict/stale value)")
		return current, true, nil
	}
	log.Info().
		Str("env", env).
		Str("name", name).
		Str("run_id", runID).
		Str("state_before", string(current.State)).
		Str("state_after", string(updated.State)).
		Int64("version_before", current.Version).
		Int64("version_after", updated.Version).
		Msg("procure CAS applied and etcd state updated")
	return updated, false, nil
}

func (s *EtcdShadowStateStore) Release(ctx context.Context, env, name, runID string) (models.ShadowDeployable, bool, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return models.ShadowDeployable{}, false, rmerrors.ErrUnsupportedEnv
	}
	key := shadowKey(env, name)

	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return models.ShadowDeployable{}, false, err
	}
	if len(resp.Kvs) == 0 {
		return models.ShadowDeployable{}, false, rmerrors.ErrNotFound
	}

	kv := resp.Kvs[0]
	var current models.ShadowDeployable
	if err := json.Unmarshal(kv.Value, &current); err != nil {
		return models.ShadowDeployable{}, false, err
	}
	if current.State != rmtypes.ShadowStateProcured {
		return current, true, nil
	}
	if current.Owner == nil || current.Owner.WorkflowRunID != runID {
		return current, false, rmerrors.ErrInvalidOwner
	}

	updated := current
	updated.State = rmtypes.ShadowStateFree
	updated.Owner = nil
	updated.MinPodCount = 0
	updated.Version = current.Version + 1
	updated.LastUpdatedAt = time.Now().UTC()

	updatedRaw, err := json.Marshal(updated)
	if err != nil {
		return models.ShadowDeployable{}, false, err
	}

	cas, err := compareAndSwap(
		ctx,
		s.client,
		key,
		kv.ModRevision,
		string(kv.Value),
		string(updatedRaw),
	)
	if err != nil {
		return models.ShadowDeployable{}, false, err
	}
	if !cas.Applied {
		return current, true, nil
	}
	return updated, false, nil
}

func (s *EtcdShadowStateStore) ChangeMinPodCount(ctx context.Context, env, name string, action rmtypes.Action, count int) (models.ShadowDeployable, error) {
	if !rmtypes.IsSupportedPoolEnv(env) {
		return models.ShadowDeployable{}, rmerrors.ErrUnsupportedEnv
	}
	key := shadowKey(env, name)

	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return models.ShadowDeployable{}, err
	}
	if len(resp.Kvs) == 0 {
		return models.ShadowDeployable{}, rmerrors.ErrNotFound
	}
	kv := resp.Kvs[0]
	var current models.ShadowDeployable
	if err := json.Unmarshal(kv.Value, &current); err != nil {
		return models.ShadowDeployable{}, err
	}

	updated := current
	switch action {
	case rmtypes.ActionIncrease:
		updated.MinPodCount += count
	case rmtypes.ActionDecrease:
		updated.MinPodCount -= count
		if updated.MinPodCount < 0 {
			updated.MinPodCount = 0
		}
	case rmtypes.ActionResetTo0:
		updated.MinPodCount = 0
	default:
		return models.ShadowDeployable{}, rmerrors.ErrInvalidAction
	}
	updated.Version = current.Version + 1
	updated.LastUpdatedAt = time.Now().UTC()

	updatedRaw, err := json.Marshal(updated)
	if err != nil {
		return models.ShadowDeployable{}, err
	}

	cas, err := compareAndSwap(ctx, s.client, key, kv.ModRevision, string(kv.Value), string(updatedRaw))
	if err != nil {
		return models.ShadowDeployable{}, err
	}
	if !cas.Applied {
		return models.ShadowDeployable{}, rmerrors.ErrCASConflict
	}
	return updated, nil
}

func shadowEnvPath(env string) string {
	base := strings.TrimSpace(os.Getenv("ETCD_APP_NAME"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("APP_NAME"))
	}
	if base == "" {
		base = "resource-manager"
	}
	return "/config/" + base + "/shadow-deployables/" + string(rmtypes.NormalizePoolEnv(env))
}

func shadowKey(env, name string) string {
	return shadowEnvPath(env) + "/" + strings.TrimSpace(name)
}
