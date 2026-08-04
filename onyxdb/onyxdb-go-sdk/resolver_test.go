package sdk

import (
	"context"
	"errors"
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
