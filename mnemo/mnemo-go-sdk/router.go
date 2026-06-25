package sdk

import (
	"hash/crc32"
	"sync"
	"sync/atomic"
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

// Router maps keys → shards (crc32 % S) and shards → a warm pod, delegating
// pod discovery to a ShardResolver (DNS in production, static for tests).
//
// The hot path (ShardFor / PodFor) takes only a read lock; per-shard
// round-robin uses lock-free atomic counters.
type Router struct {
	resolver ShardResolver

	mu         sync.RWMutex
	shardCount uint32
	unhealthy  map[string]struct{} // locally-marked-down pods, cleared on refresh

	rr sync.Map // uint32 → *atomic.Uint32
}

// NewRouter creates a router backed by the given resolver.
func NewRouter(resolver ShardResolver) *Router {
	return &Router{
		resolver:  resolver,
		unhealthy: make(map[string]struct{}),
	}
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
func (r *Router) PodFor(shardID uint32) (string, error) {
	pods := r.resolver.Resolve(shardID)
	if len(pods) == 0 {
		return "", ErrNoHealthyPod
	}
	start := int(r.counter(shardID).Add(1))

	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := 0; i < len(pods); i++ {
		cand := pods[(start+i)%len(pods)]
		if _, bad := r.unhealthy[cand]; !bad {
			return cand, nil
		}
	}
	return pods[start%len(pods)], nil
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
