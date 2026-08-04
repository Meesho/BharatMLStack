package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// EtcdClient is the narrow etcd surface the topology watcher needs.
// *clientv3.Client satisfies it; tests inject a mock.
type EtcdClient interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan
}

// TopologyWatcher keeps the Router's shard count, the DNS resolver, and the
// assignment resolver in sync with etcd.
//
// It watches two keys:
//  1. ActiveVersionPath — the control plane CAS-flips this on promote/rollback.
//     On each change (and once at startup) it reads the active version's
//     VersionMeta, pushes shard count + assignment, and triggers DNS refresh.
//  2. PodWatchPrefix — ephemeral pod registrations. A new warm pod appearing
//     triggers a re-read of the current version's assignment so the SDK picks
//     up scale-up events immediately (instead of waiting for DNS TTL).
//
// Pod IPs themselves come from K8s DNS (DNSResolver) or directly from the
// assignment map (AssignmentResolver) — etcd stays off the read hot path.
type TopologyWatcher struct {
	client       EtcdClient
	router       *Router
	dnsResolver  *DNSResolver
	assignRes    *AssignmentResolver
	pool         *ConnPool
	tenant       string
	store        string
	activeVID    string // last-known active version ID
	warmUpConns  int    // connections to pre-dial per new pod (0 = disabled)

	// Metric callbacks (nil-safe). Set via SetMetrics after construction.
	timing   func(string, time.Duration, []string)
	count    func(string, int64, []string)
	baseTags []string
}

// NewTopologyWatcher creates a watcher that updates the router + resolver from etcd.
func NewTopologyWatcher(client EtcdClient, router *Router, resolver *DNSResolver, tenant, store string) *TopologyWatcher {
	return &TopologyWatcher{
		client:      client,
		router:      router,
		dnsResolver: resolver,
		tenant:      tenant,
		store:       store,
	}
}

// SetAssignmentResolver wires the assignment-aware resolver for direct
// shard→addr routing (works on both K8s and VM). When set, the watcher pushes
// VersionMeta.Assignment on every version flip and pod registration change.
func (tw *TopologyWatcher) SetAssignmentResolver(ar *AssignmentResolver) {
	tw.assignRes = ar
}

// SetPoolForWarmUp wires the connection pool so newly-discovered pods get
// pre-dialed connections. n is the number of connections to warm per pod.
func (tw *TopologyWatcher) SetPoolForWarmUp(pool *ConnPool, n int) {
	tw.pool = pool
	tw.warmUpConns = n
}

// SetMetrics wires optional metric callbacks into the topology watcher.
func (tw *TopologyWatcher) SetMetrics(
	timing func(string, time.Duration, []string),
	count func(string, int64, []string),
	baseTags []string,
) {
	tw.timing = timing
	tw.count = count
	tw.baseTags = baseTags
}

func (tw *TopologyWatcher) twTags(extra ...string) []string {
	tags := make([]string, len(tw.baseTags)+len(extra))
	copy(tags, tw.baseTags)
	copy(tags[len(tw.baseTags):], extra)
	return tags
}

func (tw *TopologyWatcher) twEmitCount(name string, value int64, tags []string) {
	if tw.count != nil {
		tw.count(name, value, tags)
	}
}

func (tw *TopologyWatcher) twEmitTiming(name string, value time.Duration, tags []string) {
	if tw.timing != nil {
		tw.timing(name, value, tags)
	}
}

// Run does an initial reload, then watches ActiveVersionPath and the pod
// registration prefix, reloading on each change. Blocks until ctx is cancelled.
func (tw *TopologyWatcher) Run(ctx context.Context) error {
	if err := tw.reload(ctx); err != nil {
		log.Warn().Err(err).Str("tenant", tw.tenant).Str("store", tw.store).
			Msg("topology: initial reload failed, waiting for activeVersion")
	}

	activeKey := model.ActiveVersionPath(tw.tenant, tw.store)
	activeCh := tw.client.Watch(ctx, activeKey)

	podPrefix := model.PodWatchPrefix(tw.tenant, tw.store)
	podCh := tw.client.Watch(ctx, podPrefix, clientv3.WithPrefix())

	for {
		select {
		case resp, ok := <-activeCh:
			if !ok {
				return ctx.Err()
			}
			tw.handleActiveVersionWatch(ctx, resp)
		case resp, ok := <-podCh:
			if !ok {
				return ctx.Err()
			}
			tw.handlePodWatch(ctx, resp)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (tw *TopologyWatcher) handleActiveVersionWatch(ctx context.Context, resp clientv3.WatchResponse) {
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

func (tw *TopologyWatcher) handlePodWatch(ctx context.Context, resp clientv3.WatchResponse) {
	if tw.activeVID == "" {
		return // no active version yet; nothing to re-derive
	}
	// A pod registration changed (new pod, pod gone, version warmup reported).
	// Re-read the active version's assignment to pick up the new pod set.
	if err := tw.reloadVersion(ctx, tw.activeVID); err != nil {
		log.Warn().Err(err).Str("version", tw.activeVID).Msg("topology: pod watch triggered reload failed")
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

// reloadVersion reads VersionMeta for version, pushes the shard count +
// assignment to the router + resolvers, and triggers DNS re-resolve + warm-up.
func (tw *TopologyWatcher) reloadVersion(ctx context.Context, version string) error {
	start := time.Now()

	resp, err := tw.client.Get(ctx, model.VersionPrefix(tw.tenant, tw.store, version))
	if err != nil {
		tw.twEmitCount(MetricTopologyReload, 1, tw.twTags("status:error"))
		return fmt.Errorf("get version meta: %w", err)
	}
	if len(resp.Kvs) == 0 {
		tw.twEmitCount(MetricTopologyReload, 1, tw.twTags("status:error"))
		return fmt.Errorf("version meta %q not found", version)
	}

	var meta model.VersionMeta
	if err := json.Unmarshal(resp.Kvs[0].Value, &meta); err != nil {
		tw.twEmitCount(MetricTopologyReload, 1, tw.twTags("status:error"))
		return fmt.Errorf("parse version meta %q: %w", version, err)
	}

	sc := uint32(meta.ShardCount)
	tw.router.SetShardCount(sc)
	tw.dnsResolver.SetShardCount(sc)
	tw.activeVID = version

	// Push assignment map to the assignment resolver (if wired).
	var newAddrs []string
	if tw.assignRes != nil && meta.Assignment != nil {
		newAddrs = tw.assignRes.SwapAssignment(meta.Assignment)
	}

	log.Info().
		Str("version", version).
		Int("shardCount", meta.ShardCount).
		Int("newPods", len(newAddrs)).
		Msg("topology: version active, updating routing")

	// Trigger DNS re-resolve for K8s deployments.
	_ = tw.dnsResolver.Refresh(ctx)

	// Prune connection pools for pods no longer in the assignment.
	if tw.pool != nil && tw.assignRes != nil {
		tw.pool.Prune(tw.assignRes.AllAddrs())
	}

	// Warm up connections to newly-discovered pods.
	if tw.pool != nil && tw.warmUpConns > 0 && len(newAddrs) > 0 {
		go tw.warmUp(newAddrs)
	}

	tw.twEmitCount(MetricTopologyReload, 1, tw.twTags("status:ok"))
	tw.twEmitTiming(MetricTopologyReload, time.Since(start), tw.twTags("status:ok"))
	return nil
}

// warmUp pre-dials connections to new pods in the background.
func (tw *TopologyWatcher) warmUp(addrs []string) {
	for _, addr := range addrs {
		for i := 0; i < tw.warmUpConns; i++ {
			conn, err := Dial(addr, tw.pool.dialTO)
			if err != nil {
				log.Warn().Err(err).Str("addr", addr).Msg("topology: warm-up dial failed")
				break // skip remaining dials to this pod
			}
			tw.pool.Put(addr, conn)
		}
	}
}
