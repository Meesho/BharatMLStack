package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
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
	client      EtcdClient
	router      *Router
	dnsResolver *DNSResolver
	assignRes   *AssignmentResolver
	pool        *ConnPool
	tenant      string
	store       string
	activeVID   string // last-known active version ID
	warmUpConns int    // connections to pre-dial per new pod (0 = disabled)

	// Metric callbacks (nil-safe). Set via SetMetrics after construction.
	timing   func(string, time.Duration, []string)
	count    func(string, int64, []string)
	baseTags []string

	// Re-establishing an ended watch: the backoff bounds, the wait (false when
	// ctx ended first) and the clock. Tests replace them.
	restartMin time.Duration
	restartMax time.Duration
	sleep      func(ctx context.Context, d time.Duration) bool
	now        func() time.Time
}

// Backoff between attempts to re-establish an ended watch. The wait doubles
// per attempt up to the maximum, with jitter so a fleet of clients whose
// watches etcd cancelled together does not resync in step.
const (
	watchRestartMinBackoff = 500 * time.Millisecond
	watchRestartMaxBackoff = 30 * time.Second
)

// NewTopologyWatcher creates a watcher that updates the router + resolver from etcd.
func NewTopologyWatcher(client EtcdClient, router *Router, resolver *DNSResolver, tenant, store string) *TopologyWatcher {
	return &TopologyWatcher{
		client:      client,
		router:      router,
		dnsResolver: resolver,
		tenant:      tenant,
		store:       store,
		restartMin:  watchRestartMinBackoff,
		restartMax:  watchRestartMaxBackoff,
		sleep:       sleepCtx,
		now:         time.Now,
	}
}

// sleepCtx waits d, or until ctx ends; it reports whether the full wait ran.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// jitter returns a random wait in [d/2, d]; d is never negative.
func jitter(d time.Duration) time.Duration {
	half := d / 2
	return half + rand.N(d-half+1)
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
//
// A watch that ends while ctx is live is re-established. etcd cancels a watch
// that resumes from a compacted revision — the usual end of a watch whose
// connection dropped — and clientv3 then closes its channel. Without a new
// watch the client would route on its last assignment for its lifetime. Each
// attempt re-reads the topology first and watches from the revision of that
// read, so nothing written in between is missed. Attempts back off
// exponentially with jitter and are counted in MetricTopologyWatchRestart. A
// failed read is retried before any watch is opened: a watch started blind
// would skip whatever changed in the gap.
func (tw *TopologyWatcher) Run(ctx context.Context) error {
	rev, err := tw.resync(ctx)
	if err != nil {
		log.Warn().Err(err).Str("tenant", tw.tenant).Str("store", tw.store).
			Msg("topology: initial reload failed, waiting for activeVersion")
	}
	backoff := tw.restartMin
	for {
		started := tw.now()
		reason := tw.watch(ctx, rev)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if tw.now().Sub(started) >= tw.restartMax {
			backoff = tw.restartMin // the watch was healthy for a while: start over
		}
		tw.twEmitCount(MetricTopologyWatchRestart, 1, tw.twTags("reason:"+reason))
		log.Warn().Str("tenant", tw.tenant).Str("store", tw.store).Str("reason", reason).
			Msg("topology: etcd watch ended — resyncing and watching again")
		for {
			if !tw.sleep(ctx, jitter(backoff)) {
				return ctx.Err()
			}
			backoff = min(2*backoff, tw.restartMax)
			if rev, err = tw.resync(ctx); err == nil || !errors.Is(err, errActiveVersionRead) {
				break
			}
			log.Warn().Err(err).Str("tenant", tw.tenant).Str("store", tw.store).
				Msg("topology: resync read failed — retrying before watching")
		}
		if err != nil {
			log.Error().Err(err).Str("tenant", tw.tenant).Str("store", tw.store).
				Msg("topology: resync after a watch restart failed")
		}
	}
}

// watch runs the activeVersion and pod-prefix watches from rev+1 (from now
// when rev is 0) until either ends or ctx is done, and returns why it ended:
// "closed", "canceled" or "compacted"; "" when ctx ended it. Ending one watch
// cancels the other.
func (tw *TopologyWatcher) watch(ctx context.Context, rev int64) string {
	wctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var from []clientv3.OpOption
	if rev > 0 {
		from = append(from, clientv3.WithRev(rev+1))
	}
	activeCh := tw.client.Watch(wctx, model.ActiveVersionPath(tw.tenant, tw.store), from...)
	podCh := tw.client.Watch(wctx, model.PodWatchPrefix(tw.tenant, tw.store),
		append([]clientv3.OpOption{clientv3.WithPrefix()}, from...)...)

	for {
		select {
		case resp, ok := <-activeCh:
			if !ok {
				return "closed"
			}
			if reason := watchEndReason(resp); reason != "" {
				return reason
			}
			tw.handleActiveVersionWatch(ctx, resp)
		case resp, ok := <-podCh:
			if !ok {
				return "closed"
			}
			if reason := watchEndReason(resp); reason != "" {
				return reason
			}
			tw.handlePodWatch(ctx, resp)
		case <-ctx.Done():
			return ""
		}
	}
}

// watchEndReason says why etcd ended a watch with resp, or "" when resp is an
// ordinary one. A compaction cancel carries the compacted revision.
func watchEndReason(resp clientv3.WatchResponse) string {
	switch {
	case resp.CompactRevision > 0:
		return "compacted"
	case resp.Canceled:
		return "canceled"
	}
	return ""
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

// errActiveVersionRead marks a failed read of the activeVersion key: the
// resync learned no revision to watch from.
var errActiveVersionRead = errors.New("read activeVersion")

// resync reads the current activeVersion and rebuilds the router from its
// meta. It returns the etcd revision of that read, so the watches that follow
// start right after it (0 when the response carries no header: they then
// start from now). An error wrapping errActiveVersionRead means the read
// itself failed; any other error is a failed reload of the version it read.
func (tw *TopologyWatcher) resync(ctx context.Context) (int64, error) {
	resp, err := tw.client.Get(ctx, model.ActiveVersionPath(tw.tenant, tw.store))
	if err != nil {
		return 0, fmt.Errorf("get activeVersion: %w: %w", errActiveVersionRead, err)
	}
	var rev int64
	if resp.Header != nil {
		rev = resp.Header.Revision
	}
	if len(resp.Kvs) == 0 || len(resp.Kvs[0].Value) == 0 {
		return rev, nil // no active version yet
	}
	return rev, tw.reloadVersion(ctx, string(resp.Kvs[0].Value))
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

	// Slow start comes from the pods' own registrations, not the version meta:
	// a pod key is what the watch fires on, so its ramp is current when this
	// reload runs, while the meta is rewritten up to a control-plane tick later.
	tw.reloadSlowStart(ctx)

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

// reloadSlowStart reads every pod registration of the store and hands the
// router the pods still inside their slow-start window
// (docs/design/onyxdb-pod-slow-start.md). A failed read keeps the router's
// last set: at worst a ramp runs on that set until the next reload.
func (tw *TopologyWatcher) reloadSlowStart(ctx context.Context) {
	resp, err := tw.client.Get(ctx, model.PodWatchPrefix(tw.tenant, tw.store), clientv3.WithPrefix())
	if err != nil {
		log.Warn().Err(err).Str("tenant", tw.tenant).Str("store", tw.store).
			Msg("topology: pod read for slow start failed, keeping the last ramp set")
		return
	}
	now := time.Now()
	var ramps map[string]SlowStart
	for _, kv := range resp.Kvs {
		var pd podRegistration
		if json.Unmarshal(kv.Value, &pd) != nil || pd.ServingSince <= 0 || pd.SlowStartSec <= 0 {
			continue
		}
		s := SlowStart{
			Since:  time.UnixMilli(pd.ServingSince),
			Window: time.Duration(pd.SlowStartSec) * time.Second,
		}
		if !now.Before(s.Since.Add(s.Window)) {
			continue // ramp already over
		}
		if ramps == nil {
			ramps = make(map[string]SlowStart)
		}
		ramps[pd.addr()] = s
	}
	tw.router.SetSlowStart(ramps)
}

// podRegistration is the part of a pod's etcd registration (the dataloader
// writes it as controlplane/model.PodData) that slow start needs. It is decoded
// here rather than through model.PodData because the SDK is released on its own
// while the control plane is only deployed: the published SDK builds against
// the last released model, which predates these fields. The JSON keys must
// match model.PodData's tags.
type podRegistration struct {
	PodIP        string `json:"podIP"`
	Port         int    `json:"port,omitempty"`
	ServingSince int64  `json:"servingSince,omitempty"` // unix ms; 0 = no ramp
	SlowStartSec int    `json:"slowStartSec,omitempty"`
}

// defaultReadServerPort is the read server's port when a registration omits
// Port. Keep in sync with controlplane/model.DefaultReadServerPort.
const defaultReadServerPort = 9091

// addr is the address placement puts in the shard assignment for this pod, so
// a ramp is keyed exactly like the assignment entry it weights. Keep in sync
// with controlplane/model PodData.Addr.
func (p podRegistration) addr() string {
	port := p.Port
	if port == 0 {
		port = defaultReadServerPort
	}
	return fmt.Sprintf("%s:%d", p.PodIP, port)
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
