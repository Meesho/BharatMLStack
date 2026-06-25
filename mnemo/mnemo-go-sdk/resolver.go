package sdk

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ShardResolver maps a shard ID to the pod addresses (ip:port) serving it.
// The hot path (PodFor) calls Resolve, which must be cheap (cache read).
type ShardResolver interface {
	Resolve(shardID uint32) []string
}

// lookupHost is indirected so tests can stub DNS. A DNSResolver captures the
// current value at construction so its background refresh goroutine never races
// a test that swaps this var.
var lookupHost = func(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// ── StaticResolver ────────────────────────────────────────────────────────────

// StaticResolver returns a fixed assignment. Used by NewDirectClient and tests.
type StaticResolver struct {
	addrs map[uint32][]string
}

// NewStaticResolver wraps a fixed shard→addrs map.
func NewStaticResolver(addrs map[uint32][]string) *StaticResolver {
	return &StaticResolver{addrs: addrs}
}

// Resolve returns the fixed addrs for a shard.
func (s *StaticResolver) Resolve(shardID uint32) []string {
	return s.addrs[shardID]
}

// ── AssignmentResolver ────────────────────────────────────────────────────────

// AssignmentResolver uses the control plane's shard→[]addr assignment map
// directly. Works for both K8s and VM deployments — no DNS needed. The
// topology watcher pushes new assignments atomically via SwapAssignment.
type AssignmentResolver struct {
	mu    sync.RWMutex
	addrs map[uint32][]string
}

// NewAssignmentResolver creates an empty assignment resolver.
func NewAssignmentResolver() *AssignmentResolver {
	return &AssignmentResolver{addrs: make(map[uint32][]string)}
}

// Resolve returns the cached pod addrs for a shard — no IO on the hot path.
func (a *AssignmentResolver) Resolve(shardID uint32) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.addrs[shardID]
}

// SwapAssignment atomically replaces the full assignment map. The input is
// the VersionMeta.Assignment from etcd (shard-ID-string → []addr). Returns
// the set of newly-added pod addresses (for connection warm-up).
func (a *AssignmentResolver) SwapAssignment(assignment map[string][]string) []string {
	next := make(map[uint32][]string, len(assignment))
	for sidStr, addrs := range assignment {
		sid, err := strconv.ParseUint(sidStr, 10, 32)
		if err != nil {
			continue
		}
		next[uint32(sid)] = addrs
	}

	// Compute newly-added addrs (present in next but not in old).
	a.mu.RLock()
	oldSet := make(map[string]struct{})
	for _, addrs := range a.addrs {
		for _, addr := range addrs {
			oldSet[addr] = struct{}{}
		}
	}
	a.mu.RUnlock()

	var newAddrs []string
	seen := make(map[string]struct{})
	for _, addrs := range next {
		for _, addr := range addrs {
			if _, ok := oldSet[addr]; ok {
				continue
			}
			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}
			newAddrs = append(newAddrs, addr)
		}
	}

	a.mu.Lock()
	a.addrs = next
	a.mu.Unlock()

	return newAddrs
}

// AllAddrs returns the deduplicated union of all current addrs.
func (a *AssignmentResolver) AllAddrs() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	seen := make(map[string]struct{})
	var out []string
	for _, addrs := range a.addrs {
		for _, addr := range addrs {
			if _, ok := seen[addr]; !ok {
				seen[addr] = struct{}{}
				out = append(out, addr)
			}
		}
	}
	return out
}

// ── DNSResolver ───────────────────────────────────────────────────────────────

// DNSConfig configures a DNSResolver.
type DNSConfig struct {
	Tenant    string
	Store     string
	Namespace string        // K8s namespace
	DNSZone   string        // e.g. "cluster.local"
	Port      int           // read server TCP port (9091)
	Interval  time.Duration // background refresh cadence (default 30s)
}

// DNSResolver resolves each shard's headless Service to its ready pod IPs.
//
// Per-shard FQDN: {tenant}-{store}-shard-{N}.{namespace}.svc.{dnsZone}
//
// CoreDNS returns one A record per *ready* endpoint behind the headless
// Service, so only warm pods (whose readiness probe /healthz?check=warm passes)
// ever appear here. Resolution runs on a background ticker plus an on-demand
// Refresh triggered by the topology watcher on each version flip.
type DNSResolver struct {
	cfg    DNSConfig
	lookup func(context.Context, string) ([]string, error)

	mu         sync.RWMutex
	shardCount uint32
	cache      map[uint32][]string

	onRefresh func()
}

// NewDNSResolver builds a DNSResolver. Interval defaults to 30s.
func NewDNSResolver(cfg DNSConfig) *DNSResolver {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	return &DNSResolver{
		cfg:    cfg,
		lookup: lookupHost, // capture now; tests override the var before construction
		cache:  make(map[uint32][]string),
	}
}

// fqdn returns the headless Service DNS name for a shard.
func (d *DNSResolver) fqdn(shardID uint32) string {
	return fmt.Sprintf("%s-%s-shard-%d.%s.svc.%s",
		d.cfg.Tenant, d.cfg.Store, shardID, d.cfg.Namespace, d.cfg.DNSZone)
}

// Resolve returns the cached pod addrs for a shard — no DNS call on the hot path.
func (d *DNSResolver) Resolve(shardID uint32) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cache[shardID]
}

// SetShardCount sets how many shards (0..n-1) the refresh loop resolves.
func (d *DNSResolver) SetShardCount(n uint32) {
	d.mu.Lock()
	d.shardCount = n
	d.mu.Unlock()
}

// AllAddrs returns the deduplicated union of all currently-resolved addrs.
func (d *DNSResolver) AllAddrs() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	seen := make(map[string]struct{})
	var out []string
	for _, addrs := range d.cache {
		for _, a := range addrs {
			if _, ok := seen[a]; !ok {
				seen[a] = struct{}{}
				out = append(out, a)
			}
		}
	}
	return out
}

// OnRefresh registers a callback invoked after each Refresh completes.
func (d *DNSResolver) OnRefresh(fn func()) { d.onRefresh = fn }

// Refresh re-resolves every shard's headless Service and swaps the cache.
// A per-shard lookup error keeps that shard's last-known addrs — a transient
// DNS blip must not blank out a live shard.
func (d *DNSResolver) Refresh(ctx context.Context) error {
	d.mu.RLock()
	sc := d.shardCount
	d.mu.RUnlock()

	next := make(map[uint32][]string, sc)
	for shard := uint32(0); shard < sc; shard++ {
		fqdn := d.fqdn(shard)
		ips, err := d.lookup(ctx, fqdn)
		if err != nil {
			log.Warn().Err(err).Str("fqdn", fqdn).Msg("dns: lookup failed, keeping last-known addrs")
			next[shard] = d.Resolve(shard)
			continue
		}
		addrs := make([]string, len(ips))
		for i, ip := range ips {
			addrs[i] = net.JoinHostPort(ip, strconv.Itoa(d.cfg.Port))
		}
		next[shard] = addrs
	}

	d.mu.Lock()
	d.cache = next
	d.mu.Unlock()

	if d.onRefresh != nil {
		d.onRefresh()
	}
	return nil
}

// Run refreshes on a ticker until ctx is cancelled.
func (d *DNSResolver) Run(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = d.Refresh(ctx)
		}
	}
}
