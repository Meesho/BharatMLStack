package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// EtcdClient is the narrow etcd surface the topology watcher needs.
// *clientv3.Client satisfies it; tests inject a mock.
type EtcdClient interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan
}

// TopologyWatcher keeps the Router's shard count and the DNS resolver in sync
// with etcd.
//
// It watches a single key — ActiveVersionPath — which the control plane
// CAS-flips on promote/rollback. On every change (and once at startup) it reads
// the active version's VersionMeta for the shard count, pushes it to the router
// and resolver, and triggers a DNS re-resolve so the new warm pod set is picked
// up faster than the resolver's periodic TTL. Pod IPs themselves come from K8s
// DNS, not etcd — so etcd stays off the read hot path entirely.
type TopologyWatcher struct {
	client   EtcdClient
	router   *Router
	resolver *DNSResolver
	tenant   string
	store    string
}

// NewTopologyWatcher creates a watcher that updates the router + resolver from etcd.
func NewTopologyWatcher(client EtcdClient, router *Router, resolver *DNSResolver, tenant, store string) *TopologyWatcher {
	return &TopologyWatcher{client: client, router: router, resolver: resolver, tenant: tenant, store: store}
}

// Run does an initial reload, then watches ActiveVersionPath and reloads the
// assignment on each change. Blocks until ctx is cancelled or the watch closes.
func (tw *TopologyWatcher) Run(ctx context.Context) error {
	if err := tw.reload(ctx); err != nil {
		// Non-fatal: the store may not be promoted yet. The watch below will
		// pick it up once the control plane sets activeVersion.
		log.Warn().Err(err).Str("tenant", tw.tenant).Str("store", tw.store).
			Msg("topology: initial reload failed, waiting for activeVersion")
	}

	key := model.ActiveVersionPath(tw.tenant, tw.store)
	ch := tw.client.Watch(ctx, key)

	for {
		select {
		case resp, ok := <-ch:
			if !ok {
				return ctx.Err()
			}
			tw.handleWatch(ctx, resp)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (tw *TopologyWatcher) handleWatch(ctx context.Context, resp clientv3.WatchResponse) {
	for _, ev := range resp.Events {
		if ev.Type != mvccpb.PUT {
			continue
		}
		version := string(ev.Kv.Value)
		if version == "" {
			continue
		}
		if err := tw.reloadVersion(ctx, version); err != nil {
			log.Error().Err(err).Str("version", version).Msg("topology: reload on activeVersion change failed")
		}
	}
}

// reload reads the current activeVersion and rebuilds the router from its meta.
func (tw *TopologyWatcher) reload(ctx context.Context) error {
	resp, err := tw.client.Get(ctx, model.ActiveVersionPath(tw.tenant, tw.store))
	if err != nil {
		return fmt.Errorf("get activeVersion: %w", err)
	}
	if len(resp.Kvs) == 0 || len(resp.Kvs[0].Value) == 0 {
		return nil // no active version yet
	}
	return tw.reloadVersion(ctx, string(resp.Kvs[0].Value))
}

// reloadVersion reads VersionMeta for version, pushes the shard count to the
// router + resolver, and triggers a DNS re-resolve of the (now-warm) pod set.
func (tw *TopologyWatcher) reloadVersion(ctx context.Context, version string) error {
	resp, err := tw.client.Get(ctx, model.VersionPrefix(tw.tenant, tw.store, version))
	if err != nil {
		return fmt.Errorf("get version meta: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return fmt.Errorf("version meta %q not found", version)
	}

	var meta model.VersionMeta
	if err := json.Unmarshal(resp.Kvs[0].Value, &meta); err != nil {
		return fmt.Errorf("parse version meta %q: %w", version, err)
	}

	sc := uint32(meta.ShardCount)
	tw.router.SetShardCount(sc)
	tw.resolver.SetShardCount(sc)
	log.Info().
		Str("version", version).
		Int("shardCount", meta.ShardCount).
		Msg("topology: version active, re-resolving DNS")
	return tw.resolver.Refresh(ctx)
}
