package sdk

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A connect must never outlive the request that needs it: a peer that drops
// SYNs (a full accept queue, a vanished pod IP) used to hold a request for the
// kernel's ~1 s SYN retransmit or the 5 s DialTimeout, far past the 100 ms
// request deadline.

// blackholeDial makes every dial behave like one to a peer that never answers:
// it returns only when the dialer gives up, on its Timeout or its ctx.
func blackholeDial(t *testing.T) {
	t.Helper()
	orig := dialTCP
	dialTCP = func(ctx context.Context, d *net.Dialer, addr string) (net.Conn, error) {
		if d.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d.Timeout)
			defer cancel()
		}
		<-ctx.Done()
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: ctx.Err()}
	}
	t.Cleanup(func() { dialTCP = orig })
}

func TestConnPool_GetContextDialStopsAtTheRequestDeadline(t *testing.T) {
	blackholeDial(t)
	p := NewConnPool(2) // DialTimeout defaults to 5 s
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := p.GetContext(ctx, "10.0.0.1:9091")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), time.Second, "the dial waited past the request deadline")
}

func TestConnPool_GetWithoutContextKeepsTheDialTimeout(t *testing.T) {
	blackholeDial(t)
	p := NewConnPoolWithConfig(PoolConfig{DialTimeout: 50 * time.Millisecond})
	defer p.Close()

	start := time.Now()
	_, err := p.Get("10.0.0.1:9091")

	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
}

func TestConnPool_GetContextDialsAndReusesAsGetDoes(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()
	p := NewConnPool(2)
	defer p.Close()

	conn, err := p.GetContext(context.Background(), ln.Addr().String())
	require.NoError(t, err)
	p.Put(ln.Addr().String(), conn)
	again, err := p.GetContext(context.Background(), ln.Addr().String())
	require.NoError(t, err)
	assert.Same(t, conn, again, "a pooled connection is reused, not redialled")
}

// The request paths hand their deadline to the dial: every public read fails at
// about the request timeout (100 ms by default), not the 5 s DialTimeout.
func TestClient_ReadsDoNotWaitOnADialPastTheRequestTimeout(t *testing.T) {
	blackholeDial(t)
	c := NewDirectClient("10.0.0.1:9091", 2)
	defer c.Close()
	key := make([]byte, keySize)

	reads := map[string]func() error{
		"Get":            func() error { _, err := c.Get(context.Background(), key); return err },
		"BatchGet":       func() error { _, err := c.BatchGet(context.Background(), [][]byte{key}); return err },
		"StringGet":      func() error { _, err := c.StringGet(context.Background(), []byte("k")); return err },
		"StringBatchGet": func() error { _, err := c.StringBatchGet(context.Background(), [][]byte{[]byte("k")}); return err },
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			err := read()
			require.Error(t, err)
			assert.True(t, errors.Is(err, context.DeadlineExceeded), "err = %v", err)
			assert.Less(t, time.Since(start), time.Second, "%s waited on the dial past its 100 ms timeout", name)
		})
	}
}

// ── bounded dials per pod ────────────────────────────────────────────────────
//
// Each timed-out request used to close its connection, and every request that
// then found the pool empty dialled: a short stall on one pod became
// thousands of dials, most of them dropped by its full accept queue.

// gatedDialer stands in for the network: it counts dials in flight, reports
// each one on started, and connects only once release is closed.
type gatedDialer struct {
	mu                     sync.Mutex
	inFlight, max, total   int
	started, release, done chan struct{}
}

func gateDials(t *testing.T) *gatedDialer {
	t.Helper()
	g := &gatedDialer{
		started: make(chan struct{}, 64),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
	orig := dialTCP
	dialTCP = func(ctx context.Context, _ *net.Dialer, _ string) (net.Conn, error) {
		g.mu.Lock()
		g.inFlight++
		g.total++
		g.max = max(g.max, g.inFlight)
		g.mu.Unlock()
		defer func() {
			g.mu.Lock()
			g.inFlight--
			g.mu.Unlock()
		}()
		g.started <- struct{}{}
		select {
		case <-g.release:
			client, server := net.Pipe()
			go func() { <-g.done; server.Close() }()
			return client, nil
		case <-ctx.Done():
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: ctx.Err()}
		}
	}
	t.Cleanup(func() {
		dialTCP = orig
		close(g.done)
	})
	return g
}

func (g *gatedDialer) counts() (maxInFlight, total int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.max, g.total
}

func boundedPool(maxDials int) *ConnPool {
	return NewConnPoolWithConfig(PoolConfig{
		MaxPerPod:         32,
		MaxDialsPerPod:    maxDials,
		IdleCheckInterval: time.Hour,
	})
}

// Ten requests that find the pool empty never put more than MaxDialsPerPod
// dials in flight at once, and all of them still get a connection.
func TestConnPool_GetContext_DialsInFlightAreBoundedPerPod(t *testing.T) {
	g := gateDials(t)
	p := boundedPool(2)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := p.GetContext(ctx, "pod")
			if err == nil {
				p.Put("pod", conn)
			}
			errs <- err
		}()
	}
	<-g.started
	<-g.started
	select {
	case <-g.started:
		t.Fatal("a third dial started while two were in flight")
	case <-time.After(100 * time.Millisecond):
	}
	close(g.release)
	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
	maxInFlight, _ := g.counts()
	assert.Equal(t, 2, maxInFlight)
}

// A request waiting for a dial slot takes a connection that comes back to the
// pool instead of dialling another.
func TestConnPool_GetContext_AWaiterTakesAReturnedConnection(t *testing.T) {
	g := gateDials(t)
	p := boundedPool(1)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _, _ = p.GetContext(ctx, "pod") }() // holds the only dial slot
	<-g.started
	got := make(chan *Conn, 1)
	go func() {
		conn, err := p.GetContext(ctx, "pod")
		if err == nil {
			got <- conn
		}
	}()
	select {
	case <-g.started:
		t.Fatal("the waiter dialled past the bound")
	case <-time.After(100 * time.Millisecond): // the waiter is parked
	}

	returned := idleConn(t, time.Now())
	p.Put("pod", returned)

	select {
	case conn := <-got:
		assert.Same(t, returned, conn)
	case <-time.After(3 * time.Second):
		t.Fatal("the waiter did not take the returned connection")
	}
	_, total := g.counts()
	assert.Equal(t, 1, total, "the waiter dialled")
}

// A request that cannot get a dial slot before its deadline gives up with the
// deadline error and never dials.
func TestConnPool_GetContext_AWaiterGivesUpAtItsDeadlineWithoutDialling(t *testing.T) {
	g := gateDials(t)
	p := boundedPool(1)
	defer p.Close()
	long, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _, _ = p.GetContext(long, "pod") }()
	<-g.started

	short, cancelShort := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelShort()
	_, err := p.GetContext(short, "pod")

	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, total := g.counts()
	assert.Equal(t, 1, total)
}

// Pruning a pod releases the requests waiting on its dial bound: they retry
// at once instead of sitting out their deadline behind a pod that left the
// topology.
func TestConnPool_Prune_ReleasesRequestsWaitingOnADeadPod(t *testing.T) {
	g := gateDials(t)
	p := boundedPool(1)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _, _ = p.GetContext(ctx, "gone") }() // holds the only slot
	<-g.started
	go func() { _, _ = p.GetContext(ctx, "gone") }()
	select {
	case <-g.started:
		t.Fatal("the second request dialled past the bound")
	case <-time.After(100 * time.Millisecond): // parked
	}

	p.Prune(nil)

	select {
	case <-g.started: // woken, it no longer queues behind the dead pod's slot
	case <-time.After(3 * time.Second):
		t.Fatal("the waiter stayed parked after its pod was pruned")
	}
}
