package sdk

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── prune grace ──────────────────────────────────────────────────────────────
//
// A pod that drops out of the assignment keeps its pooled connections for
// PoolConfig.PruneGrace. A pod a promote listed a reconciler tick late comes
// back within it and reuses them, instead of every client redialling it at
// the same moment.

func TestPoolConfig_PruneGraceDefault(t *testing.T) {
	var pc PoolConfig
	pc.applyDefaults()
	assert.Equal(t, 15*time.Second, pc.PruneGrace)

	pc = PoolConfig{PruneGrace: -1}
	pc.applyDefaults()
	assert.Equal(t, time.Duration(-1), pc.PruneGrace, "negative keeps closing a dropped pod's connections at once")
}

func TestConnPool_Prune_HoldsADroppedPodForTheGrace(t *testing.T) {
	p := gracePool(time.Hour)
	defer p.Close()
	c := idleConn(t, time.Now())
	p.Put("10.0.0.1:9091", c)

	p.Prune([]string{"10.0.0.2:9091"})

	assert.Equal(t, 1, podCount(p, "10.0.0.1:9091"), "a dropped pod keeps its connections during the grace")
	assert.False(t, isClosed(c))
}

func TestConnPool_Prune_APodBackWithinTheGraceKeepsItsOpenConnections(t *testing.T) {
	addr, accepted := acceptingListener(t)
	mc := &metricCollector{}
	p := gracePool(time.Hour)
	p.SetMetrics(mc.timing, mc.count, nil)
	defer p.Close()

	open := dialConn(t, addr)
	<-accepted
	closedByPeer := dialConn(t, addr)
	require.NoError(t, (<-accepted).Close())
	require.Eventually(t, func() bool { return !closedByPeer.peerOpen() }, 2*time.Second, 5*time.Millisecond)
	p.Put(addr, open)
	p.Put(addr, closedByPeer)

	p.Prune(nil)            // dropped
	p.Prune([]string{addr}) // back within the grace

	assert.Equal(t, 1, podCount(p, addr), "only the connection the peer still holds is kept")
	assert.False(t, tcpClosed(open))
	assert.True(t, tcpClosed(closedByPeer))
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolPruneGrace, "result:reused"))
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolPruneGrace, "result:dead"))
	assert.Equal(t, int64(0), countTotal(mc, MetricPoolPruneGrace, "result:expired"))
}

func TestConnPool_AHeldPodIsClosedOnceTheGracePasses(t *testing.T) {
	mc := &metricCollector{}
	p := NewConnPoolWithConfig(PoolConfig{
		MaxPerPod:         4,
		PruneGrace:        20 * time.Millisecond,
		IdleTimeout:       time.Hour,
		IdleCheckInterval: 5 * time.Millisecond, // the sweep closes it, no further Prune needed
	})
	p.SetMetrics(mc.timing, mc.count, nil)
	defer p.Close()
	c := idleConn(t, time.Now())
	p.Put("10.0.0.1:9091", c)

	p.Prune(nil)

	require.Eventually(t, func() bool { return !hasPool(p, "10.0.0.1:9091") }, 2*time.Second, 5*time.Millisecond)
	assert.True(t, isClosed(c))
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolPruneGrace, "result:expired"))
}

func TestConnPool_Prune_NegativeGraceClosesAtOnce(t *testing.T) {
	p := gracePool(-1)
	defer p.Close()
	c := idleConn(t, time.Now())
	p.Put("10.0.0.1:9091", c)

	p.Prune(nil)

	assert.False(t, hasPool(p, "10.0.0.1:9091"))
	assert.True(t, isClosed(c))
}

func TestConnPool_Prune_TheGraceRunsFromTheFirstDrop(t *testing.T) {
	p := gracePool(50 * time.Millisecond)
	defer p.Close()
	c := idleConn(t, time.Now())
	p.Put("10.0.0.1:9091", c)

	p.Prune(nil)
	time.Sleep(30 * time.Millisecond)
	p.Prune(nil) // still dropped: the grace keeps running from the first drop
	time.Sleep(30 * time.Millisecond)
	p.Prune(nil)

	assert.False(t, hasPool(p, "10.0.0.1:9091"))
	assert.True(t, isClosed(c))
}

func TestConnPool_Prune_StillReleasesRequestsWaitingOnADroppedPod(t *testing.T) {
	addr := holdingListener(t)
	p := NewConnPoolWithConfig(PoolConfig{
		MaxPerPod: 4, MaxDialsPerPod: 1, PruneGrace: time.Hour,
		IdleTimeout: time.Hour, IdleCheckInterval: time.Hour,
	})
	defer p.Close()
	p.mu.Lock()
	p.pools[addr] = []*Conn{idleConn(t, time.Now())}
	p.dials[addr] = make(chan struct{}, 1)
	wake := make(chan struct{})
	p.wake[addr] = wake
	p.mu.Unlock()

	p.Prune(nil)

	select {
	case <-wake:
	default:
		t.Fatal("a request waiting on a dropped pod must be released at the drop, not when the grace ends")
	}
	assert.True(t, hasPool(p, addr), "the pod's idle connections are still held")
}

func TestConnPeerOpen(t *testing.T) {
	addr, accepted := acceptingListener(t)

	t.Run("idle and open", func(t *testing.T) {
		c := dialConn(t, addr)
		<-accepted
		assert.True(t, c.peerOpen())
	})
	t.Run("the peer closed it", func(t *testing.T) {
		c := dialConn(t, addr)
		require.NoError(t, (<-accepted).Close())
		assert.Eventually(t, func() bool { return !c.peerOpen() }, 2*time.Second, 5*time.Millisecond)
	})
	t.Run("bytes nobody asked for are unread and leave it unusable", func(t *testing.T) {
		c := dialConn(t, addr)
		_, err := (<-accepted).Write([]byte{0x01})
		require.NoError(t, err)
		assert.Eventually(t, func() bool { return !c.peerOpen() }, 2*time.Second, 5*time.Millisecond)
	})
	t.Run("closed on this side", func(t *testing.T) {
		c := dialConn(t, addr)
		<-accepted
		require.NoError(t, c.Close())
		assert.False(t, c.peerOpen())
	})
	t.Run("not a socket", func(t *testing.T) {
		assert.True(t, idleConn(t, time.Now()).peerOpen())
	})
}

func TestWarmUp_TopsUpAPodThatKeptConnections(t *testing.T) {
	addr, accepted := acceptingListener(t)
	pool := gracePool(time.Hour)
	defer pool.Close()
	pool.Put(addr, dialConn(t, addr))
	<-accepted
	tw := NewTopologyWatcher(newMockEtcd(), nil, nil, "recsys", "catalog")
	tw.SetPoolForWarmUp(pool, 3)

	tw.warmUp([]string{addr})

	assert.Equal(t, 3, podCount(pool, addr))
	for i := 0; i < 2; i++ {
		<-accepted
	}
	select {
	case <-accepted:
		t.Fatal("warm-up dialled past the shortfall")
	case <-time.After(50 * time.Millisecond):
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

// gracePool builds a pool with the given prune grace whose sweeper never fires
// during a test.
func gracePool(grace time.Duration) *ConnPool {
	return NewConnPoolWithConfig(PoolConfig{
		MaxPerPod:         4,
		PruneGrace:        grace,
		IdleTimeout:       time.Hour,
		IdleCheckInterval: time.Hour,
	})
}

// acceptingListener accepts TCP connections and hands the server side of each
// to the test, which may close it or write to it; all are closed at cleanup.
func acceptingListener(t *testing.T) (string, <-chan net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	accepted := make(chan net.Conn, 16)
	var seen []net.Conn
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			seen = append(seen, conn)
			accepted <- conn
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		<-done
		for _, c := range seen {
			c.Close()
		}
	})
	return ln.Addr().String(), accepted
}

// dialConn opens a TCP Conn to addr, closed at cleanup.
func dialConn(t *testing.T, addr string) *Conn {
	t.Helper()
	c, err := Dial(addr, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	return c
}

// tcpClosed reports whether a TCP-backed Conn has been closed on this side.
func tcpClosed(c *Conn) bool {
	return errors.Is(c.conn.SetReadDeadline(time.Now()), net.ErrClosed)
}

// countTotal sums the count emissions of name that carry tag.
func countTotal(mc *metricCollector, name, tag string) int64 {
	var n int64
	for _, r := range mc.findAll(name) {
		if v, ok := r.Value.(int64); ok && hasTag(r.Tags, tag) {
			n += v
		}
	}
	return n
}
