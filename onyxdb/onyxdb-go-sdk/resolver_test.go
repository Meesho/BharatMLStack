package sdk

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withLookup swaps the package lookupHost var for the duration of fn.
// Resolvers must be constructed inside fn so they capture the stub.
func withLookup(stub func(context.Context, string) ([]string, error), fn func()) {
	orig := lookupHost
	lookupHost = stub
	defer func() { lookupHost = orig }()
	fn()
}

func dnsCfg(port int) DNSConfig {
	return DNSConfig{Tenant: "recsys", Store: "catalog", Namespace: "onyxdb", DNSZone: "cluster.local", Port: port, Interval: time.Hour}
}

// TestDefaultLookupHost exercises the real net.DefaultResolver wrapper (the
// package default), which every other test stubs out.
func TestDefaultLookupHost_ResolvesLocalhost(t *testing.T) {
	addrs, err := lookupHost(context.Background(), "localhost")
	require.NoError(t, err)
	assert.NotEmpty(t, addrs)
}

// ── StaticResolver ────────────────────────────────────────────────────────────

func TestStaticResolver(t *testing.T) {
	s := NewStaticResolver(map[uint32][]string{0: {"a:1"}, 1: {"b:1", "c:1"}})
	assert.Equal(t, []string{"a:1"}, s.Resolve(0))
	assert.Equal(t, []string{"b:1", "c:1"}, s.Resolve(1))
	assert.Nil(t, s.Resolve(99))
}

// ── DNSResolver fqdn ──────────────────────────────────────────────────────────

func TestDNSResolver_FQDN(t *testing.T) {
	d := NewDNSResolver(dnsCfg(9091))
	assert.Equal(t, "recsys-catalog-shard-3.onyxdb.svc.cluster.local", d.fqdn(3))
}

// A store identity may legally contain underscores and uppercase, but a DNS
// label may not — the data-plane chart sanitizes the Service name, so the SDK
// must produce the same string. Regression: "ds"/"user_catalog_geohash_1_3"
// previously rendered an un-resolvable FQDN with underscores intact.
func TestDNSResolver_FQDN_SanitizesUnderscoresAndCase(t *testing.T) {
	for _, tc := range []struct {
		name          string
		tenant, store string
		shard         uint32
		want          string
	}{
		{
			name:   "underscored store matches the chart's shardName",
			tenant: "ds", store: "user_catalog_geohash_1_3", shard: 2,
			want: "ds-user-catalog-geohash-1-3-shard-2.prd-onyxdb-dataplane-ssd.svc.cluster.local",
		},
		{
			name:   "uppercase is lowered",
			tenant: "DS", store: "User_Catalog", shard: 0,
			want: "ds-user-catalog-shard-0.prd-onyxdb-dataplane-ssd.svc.cluster.local",
		},
		{
			name:   "clean names are unchanged",
			tenant: "recsys", store: "catalog", shard: 7,
			want: "recsys-catalog-shard-7.prd-onyxdb-dataplane-ssd.svc.cluster.local",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDNSResolver(DNSConfig{
				Tenant: tc.tenant, Store: tc.store,
				Namespace: "prd-onyxdb-dataplane-ssd", DNSZone: "cluster.local",
			})
			got := d.fqdn(tc.shard)
			assert.Equal(t, tc.want, got)
			assert.NotContains(t, got, "_", "'_' is illegal in an RFC-1123 DNS label")
		})
	}
}

// The shard label is truncated to 63 chars like the chart's `trunc 63`, with a
// trailing '-' trimmed so the label stays valid.
func TestDNSResolver_FQDN_TruncatesLongLabel(t *testing.T) {
	d := NewDNSResolver(DNSConfig{
		Tenant: "tenant", Store: strings.Repeat("x", 80),
		Namespace: "ns", DNSZone: "cluster.local",
	})
	label, _, _ := strings.Cut(d.fqdn(1), ".")
	assert.Len(t, label, 63)
	assert.False(t, strings.HasSuffix(label, "-"))
}

// Skip suppresses the lookup for shards the assignment map already covers —
// the reason the permanent NXDOMAIN warning no longer fires in K8s.
func TestDNSResolver_Refresh_SkipsCoveredShards(t *testing.T) {
	var mu sync.Mutex
	var looked []string
	withLookup(func(_ context.Context, host string) ([]string, error) {
		mu.Lock()
		looked = append(looked, host)
		mu.Unlock()
		return nil, errors.New("no such host")
	}, func() {
		cfg := dnsCfg(9091)
		// shard 0 is covered by the assignment; shard 1 is not.
		cfg.Skip = func(shardID uint32) bool { return shardID == 0 }
		d := NewDNSResolver(cfg)
		d.SetShardCount(2)
		require.NoError(t, d.Refresh(context.Background()))
	})

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"recsys-catalog-shard-1.onyxdb.svc.cluster.local"}, looked,
		"only the uncovered shard should be looked up")
}

// A nil Skip preserves the previous behaviour: every shard is resolved.
func TestDNSResolver_Refresh_NilSkipResolvesEveryShard(t *testing.T) {
	var mu sync.Mutex
	var count int
	withLookup(func(_ context.Context, _ string) ([]string, error) {
		mu.Lock()
		count++
		mu.Unlock()
		return []string{"10.0.0.1"}, nil
	}, func() {
		d := NewDNSResolver(dnsCfg(9091))
		d.SetShardCount(3)
		require.NoError(t, d.Refresh(context.Background()))
	})

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 3, count)
}

func TestNewDNSResolver_DefaultInterval(t *testing.T) {
	d := NewDNSResolver(DNSConfig{Interval: 0})
	assert.Equal(t, 30*time.Second, d.cfg.Interval)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestDNSResolver_Refresh_Success(t *testing.T) {
	withLookup(func(_ context.Context, host string) ([]string, error) {
		// shard-0 → two pods, shard-1 → one pod
		if host == "recsys-catalog-shard-0.onyxdb.svc.cluster.local" {
			return []string{"10.0.0.1", "10.0.0.2"}, nil
		}
		return []string{"10.0.1.1"}, nil
	}, func() {
		d := NewDNSResolver(dnsCfg(9091))
		d.SetShardCount(2)
		require.NoError(t, d.Refresh(context.Background()))

		assert.Equal(t, []string{"10.0.0.1:9091", "10.0.0.2:9091"}, d.Resolve(0))
		assert.Equal(t, []string{"10.0.1.1:9091"}, d.Resolve(1))
	})
}

func TestDNSResolver_Refresh_LookupErrorKeepsLastKnown(t *testing.T) {
	calls := 0
	withLookup(func(_ context.Context, _ string) ([]string, error) {
		calls++
		if calls == 1 {
			return []string{"10.0.0.1"}, nil // first refresh succeeds
		}
		return nil, errors.New("dns timeout") // second fails
	}, func() {
		d := NewDNSResolver(dnsCfg(9091))
		d.SetShardCount(1)

		require.NoError(t, d.Refresh(context.Background()))
		assert.Equal(t, []string{"10.0.0.1:9091"}, d.Resolve(0))

		// Second refresh fails → keep last-known.
		require.NoError(t, d.Refresh(context.Background()))
		assert.Equal(t, []string{"10.0.0.1:9091"}, d.Resolve(0))
	})
}

func TestDNSResolver_AllAddrs_Deduplicated(t *testing.T) {
	withLookup(func(_ context.Context, _ string) ([]string, error) {
		return []string{"10.0.0.1"}, nil // both shards resolve to same IP
	}, func() {
		d := NewDNSResolver(dnsCfg(9091))
		d.SetShardCount(2)
		require.NoError(t, d.Refresh(context.Background()))
		assert.Equal(t, []string{"10.0.0.1:9091"}, d.AllAddrs())
	})
}

func TestDNSResolver_OnRefreshCallbackFires(t *testing.T) {
	withLookup(func(_ context.Context, _ string) ([]string, error) {
		return []string{"10.0.0.1"}, nil
	}, func() {
		d := NewDNSResolver(dnsCfg(9091))
		d.SetShardCount(1)
		var fired int
		d.OnRefresh(func() { fired++ })
		require.NoError(t, d.Refresh(context.Background()))
		assert.Equal(t, 1, fired)
	})
}

func TestDNSResolver_Refresh_ZeroShards(t *testing.T) {
	d := NewDNSResolver(dnsCfg(9091)) // shardCount 0
	require.NoError(t, d.Refresh(context.Background()))
	assert.Empty(t, d.AllAddrs())
}

// ── Run (background ticker) ───────────────────────────────────────────────────

func TestDNSResolver_Run_RefreshesOnTick(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	withLookup(func(_ context.Context, _ string) ([]string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return []string{"10.0.0.1"}, nil
	}, func() {
		cfg := dnsCfg(9091)
		cfg.Interval = 10 * time.Millisecond
		d := NewDNSResolver(cfg)
		d.SetShardCount(1)

		ctx, cancel := context.WithCancel(context.Background())
		go d.Run(ctx)

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return calls >= 2
		}, time.Second, 5*time.Millisecond)
		cancel()

		assert.Equal(t, []string{"10.0.0.1:9091"}, d.Resolve(0))
	})
}

func TestDNSResolver_Run_StopsOnContextCancel(t *testing.T) {
	cfg := dnsCfg(9091)
	cfg.Interval = time.Hour
	d := NewDNSResolver(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { d.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}
