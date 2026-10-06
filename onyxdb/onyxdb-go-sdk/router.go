package sdk

import (
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
type Router struct {
	resolver ShardResolver

	mu         sync.RWMutex
	shardCount uint32
	unhealthy  map[string]struct{}  // locally-marked-down pods, cleared on refresh
	slowStart  map[string]SlowStart // addr → ramp; replaced whole by SetSlowStart
	rampPick   func()               // called when a pick lands on a ramping pod; nil-safe

	rr sync.Map // uint32 → *atomic.Uint32

	now   func() time.Time // indirected for tests
	float func() float64   // uniform in [0,1); indirected for tests
}

// NewRouter creates a router backed by the given resolver.
func NewRouter(resolver ShardResolver) *Router {
	return &Router{
		resolver:  resolver,
		unhealthy: make(map[string]struct{}),
		now:       time.Now,
		float:     rand.Float64,
	}
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
// current addrs, skipping pods locally marked unhealthy. If every pod is
// unhealthy it still returns one (best-effort). Returns ErrNoHealthyPod when the
// resolver has no addrs for the shard (topology not loaded / no warm pods).
//
// While one of the shard's healthy pods is in its slow start the pick is
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
	defer r.mu.RUnlock()
	if len(r.slowStart) > 0 && len(pods) > 1 {
		if pod, ok := r.weightedPick(pods); ok {
			return pod, nil
		}
	}
	for i := 0; i < len(pods); i++ {
		cand := pods[(start+i)%len(pods)]
		if _, bad := r.unhealthy[cand]; !bad {
			return cand, nil
		}
	}
	return pods[start%len(pods)], nil
}

// weightedPick draws one healthy pod with probability proportional to its
// weight. ok is false when no healthy pod is ramping (plain round-robin
// applies) or none is healthy (round-robin's best-effort applies). Caller
// holds r.mu.RLock. A shard holds a handful of pods, so the scans are short.
func (r *Router) weightedPick(pods []string) (pod string, ok bool) {
	now := r.now()
	var buf [8]float64
	ws := buf[:0]
	total, ramping := 0.0, false
	for _, p := range pods {
		w := 0.0
		if _, bad := r.unhealthy[p]; !bad {
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

// MarkUnhealthy flags a pod as locally unreachable so PodFor skips it until the
// next DNS refresh clears the set.
func (r *Router) MarkUnhealthy(pod string) {
	r.mu.Lock()
	r.unhealthy[pod] = struct{}{}
	r.mu.Unlock()
}

// ClearUnhealthy resets the local unhealthy set. Wired to run after each DNS
// refresh so a pod that recovered (and is still in DNS) is retried.
func (r *Router) ClearUnhealthy() {
	r.mu.Lock()
	r.unhealthy = make(map[string]struct{})
	r.mu.Unlock()
}
