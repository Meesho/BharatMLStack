package sdk

import (
	"errors"
	"hash/crc32"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// FallbackResolver tries a primary resolver first (assignment-aware), falling
// back to a secondary (DNS) when the primary returns no addrs for a shard.
type FallbackResolver struct {
	primary   ShardResolver
	secondary ShardResolver
}

// NewFallbackResolver creates a resolver that tries primary, then secondary.
func NewFallbackResolver(primary, secondary ShardResolver) *FallbackResolver {
	return &FallbackResolver{primary: primary, secondary: secondary}
}

// Resolve returns addrs from the primary resolver, falling back to secondary
// when the primary returns nil/empty.
func (f *FallbackResolver) Resolve(shardID uint32) []string {
	addrs := f.primary.Resolve(shardID)
	if len(addrs) > 0 {
		return addrs
	}
	return f.secondary.Resolve(shardID)
}

// SlowStart is one pod's ramp after it started serving from an empty block
// cache (docs/design/onyxdb-pod-slow-start.md): its weight against a pod with
// a full share grows linearly from minSlowStartWeight to 1 over Window,
// starting at Since.
type SlowStart struct {
	Since  time.Time
	Window time.Duration
}

// minSlowStartWeight is the weight a ramp starts at, so a pod that has just
// joined gets a trickle of reads (its cache starts warming, and a pod that
// cannot be reached shows up at once) rather than none.
const minSlowStartWeight = 0.01

// weight is s's weight at now, in [minSlowStartWeight, 1].
func (s SlowStart) weight(now time.Time) float64 {
	if s.Window <= 0 {
		return 1
	}
	elapsed := now.Sub(s.Since)
	if elapsed >= s.Window {
		return 1
	}
	w := float64(elapsed) / float64(s.Window)
	if w < minSlowStartWeight {
		return minSlowStartWeight // also a Since in the future (clock skew)
	}
	return w
}

// Router maps keys → shards (crc32 % S) and shards → a warm pod, delegating
// pod discovery to a ShardResolver (DNS in production, static for tests).
//
// The hot path (ShardFor / PodFor) takes only a read lock; per-shard
// round-robin uses lock-free atomic counters. While one of a shard's pods is
// in its slow start, PodFor picks by weight instead (weightedPick).
//
// A pod that failed a request is kept out for a backoff window
// (MarkUnhealthy) until a request through it succeeds (MarkHealthy). With
// shard fail-fast on (the default), a pod past its window takes one probe at a
// time per shard before it gets traffic again, and a shard whose pods are all
// inside their windows fails at once with ErrShardUnavailable
// (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3 B2, B3).
type Router struct {
	resolver ShardResolver

	mu         sync.RWMutex
	shardCount uint32
	unhealthy  map[string]*podHealth // locally-marked-down pods (MarkUnhealthy)
	slowStart  map[string]SlowStart  // addr → ramp; replaced whole by SetSlowStart
	rampPick   func()                // called when a pick lands on a ramping pod; nil-safe

	// marked is len(unhealthy), kept under mu and read without it, so that
	// MarkHealthy, called on every successful request, takes no lock while
	// nothing is marked.
	marked atomic.Int32

	unhealthyBase time.Duration // first window after a failure (SetUnhealthyBackoff)
	unhealthyMax  time.Duration // cap on the window
	failFast      bool          // SetShardFailFast
	// probeLease is how long a probe counts as in flight without an outcome,
	// so a lost outcome cannot hold a shard shut. The client sets it to its
	// request timeout plus a second.
	probeLease time.Duration

	// Metric callback (nil-safe) and tags, set by SetMetrics.
	count    func(string, int64, []string)
	baseTags []string

	rr sync.Map // uint32 → *atomic.Uint32

	now   func() time.Time // indirected for tests; read under mu
	float func() float64   // uniform in [0,1); indirected for tests
}

// podHealth is one pod's unhealthy mark.
type podHealth struct {
	until    time.Time // no reads before this, except the shard's probe
	failures int       // failures in a row; the window doubles with each
	probeAt  time.Time // when the shard's probe went to this pod; zero = none
}

// Unhealthy-mark defaults (docs/design/onyxdb-readserver-admission-and-sdk-redial.md
// §3 B2).
const (
	defaultUnhealthyBackoffBase = time.Second
	defaultUnhealthyBackoffMax  = 30 * time.Second
	defaultProbeLease           = time.Second
)

// ErrShardUnavailable is returned by PodFor, and so by every lookup of the
// shard's keys, when every pod of the shard failed recently and is inside its
// backoff window, and the shard's one probe is not due or already out. Nothing
// is sent. Only with shard fail-fast on (Config.ShardFailFast, the default).
var ErrShardUnavailable = errors.New("onyxdb: every pod of the shard is backing off after failures")

// NewRouter creates a router backed by the given resolver, with the default
// unhealthy backoff (1 s doubling to 30 s) and shard fail-fast on.
func NewRouter(resolver ShardResolver) *Router {
	return &Router{
		resolver:      resolver,
		unhealthy:     make(map[string]*podHealth),
		unhealthyBase: defaultUnhealthyBackoffBase,
		unhealthyMax:  defaultUnhealthyBackoffMax,
		failFast:      true,
		probeLease:    defaultProbeLease,
		now:           time.Now,
		float:         rand.Float64,
	}
}

// SetUnhealthyBackoff sets the window a failed pod is kept out for: base after
// the first failure, doubling per failure in a row up to ceiling, ±20%
// jitter. Zero or negative means the default (1 s, 30 s); a ceiling below base
// is raised to it.
func (r *Router) SetUnhealthyBackoff(base, ceiling time.Duration) {
	if base <= 0 {
		base = defaultUnhealthyBackoffBase
	}
	if ceiling <= 0 {
		ceiling = defaultUnhealthyBackoffMax
	}
	r.mu.Lock()
	r.unhealthyBase, r.unhealthyMax = base, max(base, ceiling)
	r.mu.Unlock()
}

// SetShardFailFast turns shard fail-fast on (the default) or off. Off keeps
// the routing from before it: a shard whose pods are all marked still gets
// one of them, and a pod past its window takes traffic again without a probe.
func (r *Router) SetShardFailFast(on bool) {
	r.mu.Lock()
	r.failFast = on
	r.mu.Unlock()
}

// SetMetrics wires the count callback (nil-safe) for
// MetricRouterShardUnavailable and MetricRouterProbe, tagged with baseTags.
// Set once, before use.
func (r *Router) SetMetrics(count func(name string, value int64, tags []string), baseTags []string) {
	r.mu.Lock()
	r.count, r.baseTags = count, baseTags
	r.mu.Unlock()
}

// emitCount counts name with the base tags plus extra. Called without mu held.
func (r *Router) emitCount(name string, extra ...string) {
	r.mu.RLock()
	count, base := r.count, r.baseTags
	r.mu.RUnlock()
	if count == nil {
		return
	}
	tags := make([]string, 0, len(base)+len(extra))
	count(name, 1, append(append(tags, base...), extra...))
}

// SetSlowStart replaces the set of pods in their slow start; the topology
// watcher calls it on every reload. A nil or empty map means none: PodFor is
// plain round-robin. A ramp whose window has passed weighs 1, so a stale entry
// routes exactly like no entry.
func (r *Router) SetSlowStart(m map[string]SlowStart) {
	r.mu.Lock()
	r.slowStart = m
	r.mu.Unlock()
}

// SetRampPickHook registers f to run each time PodFor picks a pod that is
// still ramping (the client counts these as MetricSlowStartPick). Set once,
// before use.
func (r *Router) SetRampPickHook(f func()) {
	r.mu.Lock()
	r.rampPick = f
	r.mu.Unlock()
}

// ShardFor returns the shard ID for a key via CRC32 IEEE, matching the
// producer's crc32(entityKey|pk) % S. Returns 0 when shardCount is 0.
func (r *Router) ShardFor(key []byte) uint32 {
	r.mu.RLock()
	sc := r.shardCount
	r.mu.RUnlock()
	if sc == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(key) % sc
}

// ShardCount returns the current shard count.
func (r *Router) ShardCount() uint32 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.shardCount
}

// SetShardCount updates the shard count (called by the topology watcher).
func (r *Router) SetShardCount(n uint32) {
	r.mu.Lock()
	r.shardCount = n
	r.mu.Unlock()
}

func (r *Router) counter(shardID uint32) *atomic.Uint32 {
	v, _ := r.rr.LoadOrStore(shardID, &atomic.Uint32{})
	return v.(*atomic.Uint32)
}

// PodFor returns a warm pod for the shard using round-robin over the resolver's
// current addrs, skipping pods inside their unhealthy window. Returns
// ErrNoHealthyPod when the resolver has no addrs for the shard (topology not
// loaded / no warm pods).
//
// With shard fail-fast on (the default): a pod past its window gets the
// shard's probe, one request at a time per shard, before it takes traffic
// again; the probe goes to the pod whose window ended first, ahead of the
// healthy pods. A shard with no pod to route to and no probe due returns
// ErrShardUnavailable. With it off, a pod past its window routes like any
// other, and a shard whose pods are all inside their windows still gets one
// of them (best effort).
//
// While one of the shard's routable pods is in its slow start the pick is
// weighted instead: a ramping pod by its SlowStart weight, every other pod 1.
// Weights only matter relative to each other, so a shard whose pods are all
// ramping splits evenly among them.
func (r *Router) PodFor(shardID uint32) (string, error) {
	pods := r.resolver.Resolve(shardID)
	if len(pods) == 0 {
		return "", ErrNoHealthyPod
	}
	start := int(r.counter(shardID).Add(1))

	r.mu.RLock()
	var now time.Time // read only when a pod is marked or ramping: the common case pays no clock read
	if len(r.unhealthy) > 0 || len(r.slowStart) > 0 {
		now = r.now()
	}
	if !r.probeDueLocked(pods, now) {
		if len(r.slowStart) > 0 && len(pods) > 1 {
			if pod, ok := r.weightedPick(pods, now); ok {
				r.mu.RUnlock()
				return pod, nil
			}
		}
		for i := 0; i < len(pods); i++ {
			if cand := pods[(start+i)%len(pods)]; r.routableLocked(cand, now) {
				r.mu.RUnlock()
				return cand, nil
			}
		}
		if !r.failFast {
			r.mu.RUnlock()
			return pods[start%len(pods)], nil // every pod inside its window: best effort
		}
	}
	r.mu.RUnlock()
	return r.probeOrFail(pods, start)
}

// routableLocked reports whether PodFor may route an ordinary request to pod:
// it is not marked, or, with fail-fast off, its window has passed. Caller
// holds r.mu.
func (r *Router) routableLocked(pod string, now time.Time) bool {
	h := r.unhealthy[pod]
	return h == nil || (!r.failFast && !now.Before(h.until))
}

// probingLocked reports whether h's pod has a probe in flight. Caller holds
// r.mu.
func (r *Router) probingLocked(h *podHealth, now time.Time) bool {
	return !h.probeAt.IsZero() && now.Sub(h.probeAt) < r.probeLease
}

// probeDueLocked reports whether the shard of pods owes a probe: fail-fast is
// on, one of its pods is past its window, and none is being probed. Caller
// holds r.mu. A shard holds a handful of pods, so the scan is short.
func (r *Router) probeDueLocked(pods []string, now time.Time) bool {
	if !r.failFast || len(r.unhealthy) == 0 {
		return false
	}
	due := false
	for _, p := range pods {
		h := r.unhealthy[p]
		if h == nil {
			continue
		}
		if r.probingLocked(h, now) {
			return false
		}
		if !now.Before(h.until) {
			due = true
		}
	}
	return due
}

// probeOrFail runs under the write lock when PodFor's read-locked pass found a
// probe due or nothing to route to: it sends the probe to the pod whose window
// ended first, else routes to a pod that became routable in between, else
// fails the shard with ErrShardUnavailable.
func (r *Router) probeOrFail(pods []string, start int) (string, error) {
	r.mu.Lock()
	now := r.now()
	if r.probeDueLocked(pods, now) {
		var probe string
		var first *podHealth
		for _, p := range pods {
			if h := r.unhealthy[p]; h != nil && !now.Before(h.until) && (first == nil || h.until.Before(first.until)) {
				probe, first = p, h
			}
		}
		first.probeAt = now
		r.mu.Unlock()
		return probe, nil
	}
	for i := 0; i < len(pods); i++ {
		if cand := pods[(start+i)%len(pods)]; r.routableLocked(cand, now) {
			r.mu.Unlock()
			return cand, nil
		}
	}
	r.mu.Unlock()
	r.emitCount(MetricRouterShardUnavailable)
	return "", ErrShardUnavailable
}

// weightedPick draws one routable pod with probability proportional to its
// weight. ok is false when no routable pod is ramping (plain round-robin
// applies) or none is routable (round-robin's fallback applies). Caller holds
// r.mu.RLock. A shard holds a handful of pods, so the scans are short.
func (r *Router) weightedPick(pods []string, now time.Time) (pod string, ok bool) {
	var buf [8]float64
	ws := buf[:0]
	total, ramping := 0.0, false
	for _, p := range pods {
		w := 0.0
		if r.routableLocked(p, now) {
			w = 1
			if s, in := r.slowStart[p]; in {
				if w = s.weight(now); w < 1 {
					ramping = true
				}
			}
		}
		ws = append(ws, w)
		total += w
	}
	if !ramping || total == 0 {
		return "", false
	}
	pick := len(ws) - 1
	u := r.float() * total
	for i, w := range ws {
		if u < w {
			pick = i
			break
		}
		u -= w
	}
	for ws[pick] == 0 { // u landed past the last pod by float rounding: take the last healthy one
		pick--
	}
	if ws[pick] < 1 && r.rampPick != nil {
		r.rampPick()
	}
	return pods[pick], true
}

// MarkUnhealthy records a failed request through pod: PodFor keeps it out for
// a backoff window, 1 s after the first failure in a row and doubling per
// failure up to 30 s by default (SetUnhealthyBackoff), ±20% jitter. Failures
// inside the current window are the same episode (requests that were in
// flight when the pod stalled) and do not stretch it. A failure of the
// shard's probe to pod counts as the probe's outcome.
func (r *Router) MarkUnhealthy(pod string) {
	r.mu.Lock()
	now := r.now()
	h := r.unhealthy[pod]
	if h == nil {
		h = &podHealth{}
		r.unhealthy[pod] = h
		r.marked.Store(int32(len(r.unhealthy)))
	}
	probed := !h.probeAt.IsZero()
	h.probeAt = time.Time{}
	if !now.Before(h.until) {
		h.failures++
		h.until = now.Add(backoffDelay(r.unhealthyBase, r.unhealthyMax, h.failures, r.float()))
	}
	r.mu.Unlock()
	if probed {
		r.emitCount(MetricRouterProbe, "result:fail")
	}
}

// MarkHealthy records a successful request through pod (a miss included): its
// mark and backoff end. Called on every success, so it takes no lock while no
// pod is marked.
func (r *Router) MarkHealthy(pod string) {
	if r.marked.Load() == 0 {
		return
	}
	r.mu.Lock()
	h := r.unhealthy[pod]
	if h != nil {
		delete(r.unhealthy, pod)
		r.marked.Store(int32(len(r.unhealthy)))
	}
	r.mu.Unlock()
	if h != nil && !h.probeAt.IsZero() {
		r.emitCount(MetricRouterProbe, "result:ok")
	}
}

// endProbe ends a probe to pod that never reached it (this client's own dial
// rate refused the connection): no verdict, the window stays, and the shard
// may probe again at once.
func (r *Router) endProbe(pod string) {
	if r.marked.Load() == 0 {
		return
	}
	r.mu.Lock()
	if h := r.unhealthy[pod]; h != nil {
		h.probeAt = time.Time{}
	}
	r.mu.Unlock()
}

// ClearUnhealthy drops the marks whose window has passed and the marks of pods
// no longer routed to (in no shard's resolved addrs), so the set cannot grow
// with pod churn. It keeps active windows and a pod with a probe in flight:
// it runs on every DNS refresh, which in production means every topology reload and
// every pod-key re-put (~every 30 s / pod count), and must not cut a backoff
// short.
func (r *Router) ClearUnhealthy() {
	if r.marked.Load() == 0 {
		return
	}
	routed := make(map[string]struct{})
	for shard, n := uint32(0), r.ShardCount(); shard < n; shard++ {
		for _, p := range r.resolver.Resolve(shard) {
			routed[p] = struct{}{}
		}
	}
	r.mu.Lock()
	now := r.now()
	for pod, h := range r.unhealthy {
		if _, ok := routed[pod]; ok && (now.Before(h.until) || r.probingLocked(h, now)) {
			continue
		}
		delete(r.unhealthy, pod)
	}
	r.marked.Store(int32(len(r.unhealthy)))
	r.mu.Unlock()
}
