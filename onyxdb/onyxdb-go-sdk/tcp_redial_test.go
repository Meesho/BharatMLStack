package sdk

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B1 (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3): a pod
// that stalls used to get one SYN per request from every client at once (188K
// dials/min fleet-wide on 2026-11-10). Each client now dials a pod at most at
// DialRatePerPod, with a burst of MaxDialsPerPod, and backs off after a failed
// dial. With no idle connection, a request that may not dial fails at once
// with ErrDialBackoff instead of dialling or waiting.

var redialT0 = time.Date(2026, 11, 10, 9, 9, 0, 0, time.UTC)

// dialCounter stands in for the network: it counts dial attempts per address
// and either fails each at once (a pod resetting SYNs) or connects it to an
// in-memory peer.
type dialCounter struct {
	mu   sync.Mutex
	n    map[string]int
	fail atomic.Bool
}

func countDials(t *testing.T, fail bool) *dialCounter {
	t.Helper()
	d := &dialCounter{n: make(map[string]int)}
	d.fail.Store(fail)
	var peers []net.Conn
	orig := dialTCP
	dialTCP = func(_ context.Context, _ *net.Dialer, addr string) (net.Conn, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.n[addr]++
		if d.fail.Load() {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
		}
		client, server := net.Pipe()
		peers = append(peers, server)
		return client, nil
	}
	t.Cleanup(func() {
		dialTCP = orig
		d.mu.Lock()
		defer d.mu.Unlock()
		for _, c := range peers {
			c.Close()
		}
	})
	return d
}

func (d *dialCounter) count(addr string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.n[addr]
}

// redialPool builds a pool on clk whose backoff waits are exactly nominal (no
// jitter) and whose idle sweep never fires during a test.
func redialPool(cfg PoolConfig, clk *fakeClock) *ConnPool {
	cfg.IdleCheckInterval = time.Hour
	p := NewConnPoolWithConfig(cfg)
	p.now = clk.now
	p.float = func() float64 { return 0.5 }
	return p
}

// Under a dial-failure storm a pod never gets more than rate × elapsed + burst
// dials from one client, however many requests want it; with dial backoff on
// (the default) it gets far fewer.
func TestConnPool_DialRate_FailureStormStaysWithinRateTimesElapsedPlusBurst(t *testing.T) {
	cases := []struct {
		name    string
		backoff time.Duration
		check   func(t *testing.T, dials int)
	}{
		{"rate limit alone", -1, func(t *testing.T, dials int) {
			assert.GreaterOrEqual(t, dials, 2*10+4-1, "the bucket must still admit its rate")
		}},
		// Failed dials at 0, 0.1, 0.3, 0.7, 1.5, 3.1 and 6.3 s; the next is
		// due at 11.3 s (the wait caps at 5 s).
		{"with dial backoff", 0, func(t *testing.T, dials int) { assert.Equal(t, 7, dials) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := countDials(t, true)
			clk := &fakeClock{t: redialT0}
			p := redialPool(PoolConfig{MaxDialsPerPod: 4, DialRatePerPod: 2, DialBackoffBase: tc.backoff}, clk)
			defer p.Close()

			for i := 0; i <= 1000; i++ { // 10 s, 50 requests every 10 ms
				for j := 0; j < 50; j++ {
					_, err := p.GetContext(context.Background(), "pod")
					require.Error(t, err)
				}
				elapsed := clk.now().Sub(redialT0).Seconds()
				require.LessOrEqual(t, float64(d.count("pod")), 2*elapsed+4, "after %.2f s", elapsed)
				clk.advance(10 * time.Millisecond)
			}
			tc.check(t, d.count("pod"))
		})
	}
}

func TestConnPool_DialRate_ConcurrentStormStaysWithinTheBound(t *testing.T) {
	d := countDials(t, true)
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{MaxDialsPerPod: 4, DialRatePerPod: 2, DialBackoffBase: -1}, clk)
	defer p.Close()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = p.GetContext(context.Background(), "pod")
				}
			}
		}()
	}
	for i := 0; i < 300; i++ {
		clk.advance(10 * time.Millisecond)
		time.Sleep(100 * time.Microsecond)
		dials := d.count("pod") // read before the clock: every dial counted was admitted by then
		elapsed := clk.now().Sub(redialT0).Seconds()
		require.LessOrEqual(t, float64(dials), 2*elapsed+4, "after %.2f s", elapsed)
	}
	close(stop)
	wg.Wait()
	assert.Positive(t, d.count("pod"))
}

func TestConnPool_DialBackoff_DoublesToTheCapAndResetsOnASuccessfulDial(t *testing.T) {
	d := countDials(t, true)
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{DialRatePerPod: -1, DialBackoffBase: 100 * time.Millisecond, DialBackoffMax: time.Second}, clk)
	defer p.Close()
	ctx := context.Background()

	for _, wait := range []time.Duration{
		100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond,
		time.Second, time.Second,
	} {
		before := d.count("pod")
		_, err := p.GetContext(ctx, "pod")
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrDialBackoff, "the window had passed: the request must dial")
		require.Equal(t, before+1, d.count("pod"))

		clk.advance(wait - time.Millisecond)
		_, err = p.GetContext(ctx, "pod")
		require.ErrorIs(t, err, ErrDialBackoff, "still inside the %v window", wait)
		require.Equal(t, before+1, d.count("pod"), "no dial inside the window")
		clk.advance(time.Millisecond)
	}

	d.fail.Store(false)
	conn, err := p.GetContext(ctx, "pod")
	require.NoError(t, err)
	defer conn.Close()

	// The success reset the schedule: the next failure waits the base again.
	d.fail.Store(true)
	_, err = p.GetContext(ctx, "pod")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrDialBackoff)
	clk.advance(99 * time.Millisecond)
	_, err = p.GetContext(ctx, "pod")
	require.ErrorIs(t, err, ErrDialBackoff)
	clk.advance(time.Millisecond)
	before := d.count("pod")
	_, _ = p.GetContext(ctx, "pod")
	assert.Equal(t, before+1, d.count("pod"), "after one base wait the pod is dialled again")
}

func TestConnPool_DialBackoff_IsJittered(t *testing.T) {
	cases := []struct {
		name string
		u    float64
		wait time.Duration
	}{
		{"lowest draw: 20% shorter", 0, 80 * time.Millisecond},
		{"middle draw: nominal", 0.5, 100 * time.Millisecond},
		{"highest draw: 20% longer", 0.99999, 120 * time.Millisecond},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := countDials(t, true)
			clk := &fakeClock{t: redialT0}
			p := redialPool(PoolConfig{DialRatePerPod: -1}, clk)
			p.float = func() float64 { return tc.u }
			defer p.Close()

			_, _ = p.GetContext(context.Background(), "pod")
			clk.advance(tc.wait - time.Millisecond)
			_, err := p.GetContext(context.Background(), "pod")
			require.ErrorIs(t, err, ErrDialBackoff)
			clk.advance(time.Millisecond)
			_, _ = p.GetContext(context.Background(), "pod")
			assert.Equal(t, 2, d.count("pod"))
		})
	}
}

// Dials that fail together (every slot dialling into one stall) are one
// failure, not one each: the pod waits the base, not 2^slots × the base.
func TestConnPool_DialBackoff_ConcurrentFailuresCountOnce(t *testing.T) {
	started, release := make(chan struct{}, 4), make(chan struct{})
	orig := dialTCP
	dialTCP = func(context.Context, *net.Dialer, string) (net.Conn, error) {
		started <- struct{}{}
		<-release
		return nil, errors.New("connection reset")
	}
	t.Cleanup(func() { dialTCP = orig })
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{MaxDialsPerPod: 4, DialRatePerPod: -1}, clk)
	defer p.Close()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = p.GetContext(context.Background(), "pod")
		}()
	}
	for i := 0; i < 4; i++ {
		<-started
	}
	close(release)
	wg.Wait()

	clk.advance(99 * time.Millisecond)
	_, err := p.GetContext(context.Background(), "pod")
	require.ErrorIs(t, err, ErrDialBackoff)
	clk.advance(time.Millisecond)
	go func() { _, _ = p.GetContext(context.Background(), "pod") }()
	select {
	case <-started: // dialled after one base wait
	case <-time.After(3 * time.Second):
		t.Fatal("four concurrent failures were counted as more than one")
	}
}

// Backoff and the rate limit gate dials only: a pod with an idle connection
// still serves from it.
func TestConnPool_DialBackoff_AnIdleConnectionIsStillServed(t *testing.T) {
	countDials(t, true)
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{}, clk)
	defer p.Close()
	_, err := p.GetContext(context.Background(), "pod")
	require.Error(t, err)
	_, err = p.GetContext(context.Background(), "pod")
	require.ErrorIs(t, err, ErrDialBackoff)

	idle := idleConn(t, time.Now())
	p.Put("pod", idle)
	got, err := p.GetContext(context.Background(), "pod")
	require.NoError(t, err)
	assert.Same(t, idle, got)
}

// discardBroken hands conn back as one the peer closed, the way a request
// whose connection broke does: the pool closes it, and it no longer counts as
// a connection that may come back.
func discardBroken(p *ConnPool, addr string, conn *Conn) {
	conn.broken = true
	p.Recycle(addr, conn)
}

// Out of tokens with no idle connection and none out or being dialled: the
// request fails at once, whatever its deadline, without dialling, and the
// refusal is told apart from a backoff in the metric and the error.
func TestConnPool_DialRate_OutOfTokensWithNothingToWaitForFailsAtOnce(t *testing.T) {
	d := countDials(t, false)
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{MaxDialsPerPod: 2, DialRatePerPod: 1}, clk)
	defer p.Close()
	mc := &metricCollector{}
	p.SetMetrics(mc.timing, mc.count, []string{"tenant:t", "store:s"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 2; i++ { // the burst; the peer closes both
		conn, err := p.GetContext(ctx, "pod")
		require.NoError(t, err)
		discardBroken(p, "pod", conn)
	}
	start := time.Now()
	_, err := p.GetContext(ctx, "pod")
	require.ErrorIs(t, err, ErrDialBackoff)
	assert.ErrorIs(t, err, errDialRateLimited)
	assert.Less(t, time.Since(start), 100*time.Millisecond, "a refused dial must not wait")
	assert.Equal(t, 2, d.count("pod"))
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolGet, "result:dial_ratelimited"))
	assert.Equal(t, []string{"tenant:t", "store:s", "result:dial_ratelimited"}, mc.findAll(MetricPoolGet)[2].Tags)

	clk.advance(time.Second) // one token back
	conn, err := p.GetContext(ctx, "pod")
	require.NoError(t, err)
	discardBroken(p, "pod", conn)
	assert.Equal(t, 3, d.count("pod"))

	// A failed dial: the next refusal is a backoff, counted as such.
	d.fail.Store(true)
	clk.advance(time.Second)
	_, err = p.GetContext(ctx, "pod")
	require.Error(t, err)
	_, err = p.GetContext(ctx, "pod")
	require.ErrorIs(t, err, ErrDialBackoff)
	assert.ErrorIs(t, err, errDialBackingOff)
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolGet, "result:dial_backoff"))
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolGet, "result:dial_ratelimited"))
}

// Out of tokens while connections to the pod are out: a burst of concurrency
// is served by the connections there are. The request waits for one to come
// back, up to its deadline, and never dials past the rate.
func TestConnPool_DialRate_OutOfTokensWaitsForAConnectionThatIsOut(t *testing.T) {
	d := countDials(t, false)
	p := redialPool(PoolConfig{MaxDialsPerPod: 2, DialRatePerPod: 1}, &fakeClock{t: redialT0})
	defer p.Close()
	mc := &metricCollector{}
	p.SetMetrics(mc.timing, mc.count, nil)
	a, err := p.GetContext(context.Background(), "pod")
	require.NoError(t, err)
	b, err := p.GetContext(context.Background(), "pod")
	require.NoError(t, err)

	got := make(chan *Conn, 1)
	go func() {
		conn, err := p.GetContext(context.Background(), "pod")
		assert.NoError(t, err)
		got <- conn
	}()
	select {
	case <-got:
		t.Fatal("the request did not wait for a connection")
	case <-time.After(50 * time.Millisecond):
	}
	p.Put("pod", b)
	select {
	case conn := <-got:
		assert.Same(t, b, conn)
	case <-time.After(3 * time.Second):
		t.Fatal("the waiter did not take the returned connection")
	}
	assert.Equal(t, 2, d.count("pod"), "no dial past the rate")

	short, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = p.GetContext(short, "pod")
	require.ErrorIs(t, err, context.DeadlineExceeded, "a connection is out: wait, up to the deadline")
	assert.Equal(t, int64(1), countTotal(mc, MetricPoolGet, "result:dial_wait_expired"))

	// The last connections out break: the waiter stops waiting and fails at
	// once, since nothing is left to come back.
	waiting := make(chan error, 1)
	go func() { _, err := p.GetContext(context.Background(), "pod"); waiting <- err }()
	time.Sleep(20 * time.Millisecond)
	discardBroken(p, "pod", a)
	discardBroken(p, "pod", b)
	select {
	case err := <-waiting:
		assert.ErrorIs(t, err, errDialRateLimited)
	case <-time.After(3 * time.Second):
		t.Fatal("the waiter was not released when the connections it waited for broke")
	}
	assert.Equal(t, 2, d.count("pod"))
}

// A request that finds every dial slot busy and waits for one is still held to
// the rate when its turn comes: with no token left it waits for a connection
// instead of dialling.
func TestConnPool_DialRate_AWaiterWithoutATokenWaitsForAConnection(t *testing.T) {
	g := gateDials(t)
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{MaxPerPod: 8, MaxDialsPerPod: 1, DialRatePerPod: 1}, clk)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type result struct {
		conn *Conn
		err  error
	}
	get := func(out chan<- result) {
		conn, err := p.GetContext(ctx, "pod")
		out <- result{conn, err}
	}

	first := make(chan result, 1)
	go get(first) // takes the only slot and the only token
	<-g.started
	clk.advance(time.Second) // one token back: two waiters see it and park on the slot
	waiters := make(chan result, 2)
	go get(waiters)
	go get(waiters)
	select {
	case r := <-waiters:
		t.Fatalf("a request did not wait for the slot: %v", r.err)
	case <-time.After(50 * time.Millisecond):
	}

	close(g.release)
	r := <-first
	require.NoError(t, r.err)
	r = <-waiters // the first waiter's turn spends the token
	require.NoError(t, r.err)
	select {
	case r := <-waiters:
		t.Fatalf("the second waiter dialled or failed instead of waiting: %v", r.err)
	case <-time.After(50 * time.Millisecond):
	}
	p.Put("pod", r.conn)
	r2 := <-waiters
	require.NoError(t, r2.err)
	assert.Same(t, r.conn, r2.conn, "it took the returned connection")
	_, total := g.counts()
	assert.Equal(t, 2, total)
}

func TestConnPool_DialLimitsOff(t *testing.T) {
	t.Run("negative rate: no rate limit", func(t *testing.T) {
		d := countDials(t, false)
		p := redialPool(PoolConfig{DialRatePerPod: -1}, &fakeClock{t: redialT0})
		defer p.Close()
		for i := 0; i < 20; i++ {
			conn, err := p.GetContext(context.Background(), "pod")
			require.NoError(t, err)
			defer conn.Close()
		}
		assert.Equal(t, 20, d.count("pod"))
	})
	t.Run("negative backoff base: every request dials", func(t *testing.T) {
		d := countDials(t, true)
		p := redialPool(PoolConfig{DialRatePerPod: -1, DialBackoffBase: -1}, &fakeClock{t: redialT0})
		defer p.Close()
		for i := 0; i < 20; i++ {
			_, err := p.GetContext(context.Background(), "pod")
			require.NotErrorIs(t, err, ErrDialBackoff)
		}
		assert.Equal(t, 20, d.count("pod"))
	})
}

// Only the pod's failure to answer starts a backoff: a dial the caller
// cancelled, or one never started because the request's deadline had already
// passed, says nothing about the pod. A dial cut off by the request deadline
// (a pod dropping SYNs) does.
func TestConnPool_DialBackoff_OnlyThePodsFailuresCount(t *testing.T) {
	t.Run("cancelled by the caller", func(t *testing.T) {
		g := gateDials(t)
		p := redialPool(PoolConfig{}, &fakeClock{t: redialT0})
		defer p.Close()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, err := p.GetContext(ctx, "pod"); done <- err }()
		<-g.started
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)

		go func() { _, _ = p.GetContext(context.Background(), "pod") }()
		select {
		case <-g.started:
		case <-time.After(3 * time.Second):
			t.Fatal("a cancelled dial put the pod in backoff")
		}
	})
	t.Run("request deadline already passed", func(t *testing.T) {
		d := countDials(t, true)
		p := redialPool(PoolConfig{}, &fakeClock{t: redialT0})
		defer p.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		// A free slot and the done ctx are both ready, and select picks either:
		// both ways the request gives up without dialling.
		for i := 0; i < 64; i++ {
			_, err := p.GetContext(ctx, "pod")
			require.ErrorIs(t, err, context.Canceled)
		}
		assert.Equal(t, 0, d.count("pod"), "a request already over its deadline must not dial")

		_, err := p.GetContext(context.Background(), "pod")
		require.NotErrorIs(t, err, ErrDialBackoff)
		assert.Equal(t, 1, d.count("pod"))
	})
	t.Run("cut off by the request deadline", func(t *testing.T) {
		blackholeDial(t)
		p := redialPool(PoolConfig{}, &fakeClock{t: redialT0})
		defer p.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := p.GetContext(ctx, "pod")
		require.ErrorIs(t, err, context.DeadlineExceeded)

		_, err = p.GetContext(context.Background(), "pod")
		assert.ErrorIs(t, err, ErrDialBackoff)
	})
}

// Prune forgets the dial state of a pod that left the assignment, unless it is
// still backing off: a stalled pod that drops out and comes straight back
// must not get a fresh burst.
func TestConnPool_Prune_KeepsTheDialStateOfAPodStillBackingOff(t *testing.T) {
	d := countDials(t, true)
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{}, clk)
	defer p.Close()
	_, _ = p.GetContext(context.Background(), "stalled") // in backoff now
	d.fail.Store(false)
	conn, err := p.GetContext(context.Background(), "fine") // used a token
	require.NoError(t, err)
	defer conn.Close()

	p.Prune(nil)
	assert.True(t, hasGate(p, "stalled"))
	assert.False(t, hasGate(p, "fine"))
	_, err = p.GetContext(context.Background(), "stalled")
	require.ErrorIs(t, err, ErrDialBackoff)

	clk.advance(time.Second)
	p.Prune(nil)
	assert.False(t, hasGate(p, "stalled"))
	p.Prune([]string{"stalled"}) // a live pod's state is never dropped
	conn, err = p.GetContext(context.Background(), "stalled")
	require.NoError(t, err)
	defer conn.Close()
	p.Prune([]string{"stalled"})
	assert.True(t, hasGate(p, "stalled"))
}

func hasGate(p *ConnPool, addr string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.gates[addr]
	return ok
}

func TestPoolConfig_RedialDefaults(t *testing.T) {
	pc := PoolConfig{}
	pc.applyDefaults()
	assert.Equal(t, 2.0, pc.DialRatePerPod)
	assert.Equal(t, 100*time.Millisecond, pc.DialBackoffBase)
	assert.Equal(t, 5*time.Second, pc.DialBackoffMax)

	off := PoolConfig{DialRatePerPod: -1, DialBackoffBase: -1}
	off.applyDefaults()
	assert.Equal(t, -1.0, off.DialRatePerPod, "negative turns the rate limit off")
	assert.Equal(t, time.Duration(-1), off.DialBackoffBase, "negative turns dial backoff off")

	clamped := PoolConfig{DialBackoffBase: 2 * time.Second, DialBackoffMax: time.Second}
	clamped.applyDefaults()
	assert.Equal(t, 2*time.Second, clamped.DialBackoffMax, "the cap is never below the base")
}

// ── warm-up ──────────────────────────────────────────────────────────────────
//
// Warm-up pre-dials MinPerPod connections to each pod newly in the assignment,
// once per change, off the request path. It is exempt from the rate limit, so
// a client with a large minConnsPerPod does not take seconds per pod to warm,
// but it honours dial backoff and feeds it.

func TestWarmUp_IsNotThrottledByTheDialRate(t *testing.T) {
	addr := holdingListener(t)
	pool := NewConnPoolWithConfig(PoolConfig{
		MaxPerPod: 8, MaxDialsPerPod: 1, DialRatePerPod: 0.001, IdleCheckInterval: time.Hour,
	})
	defer pool.Close()
	used, err := pool.GetContext(context.Background(), addr) // the request path spends the only token
	require.NoError(t, err)
	defer used.Close()
	tw := NewTopologyWatcher(newMockEtcd(), nil, nil, "recsys", "catalog")
	tw.SetPoolForWarmUp(pool, 6)

	start := time.Now()
	tw.warmUp([]string{addr})

	assert.Equal(t, 6, podCount(pool, addr), "warm-up dials every connection it owes")
	assert.Less(t, time.Since(start), 2*time.Second)
	for i := 0; i < 6; i++ {
		conn, err := pool.GetContext(context.Background(), addr)
		require.NoError(t, err)
		defer conn.Close()
	}
	short, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = pool.GetContext(short, addr)
	assert.ErrorIs(t, err, context.DeadlineExceeded,
		"requests stay held to the rate: no dial, only a wait for a connection to come back")
}

func TestWarmUp_SkipsAPodInDialBackoffAndFeedsIt(t *testing.T) {
	d := countDials(t, true)
	clk := &fakeClock{t: redialT0}
	pool := redialPool(PoolConfig{MaxPerPod: 4}, clk)
	defer pool.Close()
	tw := NewTopologyWatcher(newMockEtcd(), nil, nil, "recsys", "catalog")
	tw.SetPoolForWarmUp(pool, 2)

	tw.warmUp([]string{"pod"}) // fails once, stops for this pod
	assert.Equal(t, 1, d.count("pod"))
	_, err := pool.GetContext(context.Background(), "pod")
	require.ErrorIs(t, err, ErrDialBackoff, "a failed warm-up dial backs the pod off")

	tw.warmUp([]string{"pod"})
	assert.Equal(t, 1, d.count("pod"), "warm-up must not dial a pod in backoff")

	d.fail.Store(false)
	clk.advance(time.Second)
	tw.warmUp([]string{"pod"})
	assert.Equal(t, 2, podCount(pool, "pod"))
	assert.Equal(t, 3, d.count("pod"))
}

// ── a request parked on a busy dial slot ─────────────────────────────────────
//
// With a token to spend but every slot busy, a request waits for a slot. When
// its turn comes it looks again, because things changed while it waited.

// parkOnTheSlot starts a dial that holds the pod's only slot until it is
// released (started reports it), refills the bucket, and parks a second
// request behind it; that request's outcome arrives on the returned channels.
func parkOnTheSlot(t *testing.T, started <-chan struct{}, ctx context.Context) (*ConnPool, <-chan error, <-chan *Conn) {
	t.Helper()
	clk := &fakeClock{t: redialT0}
	p := redialPool(PoolConfig{MaxPerPod: 8, MaxDialsPerPod: 1, DialRatePerPod: 1}, clk)
	t.Cleanup(p.Close)
	go func() { _, _ = p.GetContext(context.Background(), "pod") }() // the slot holder
	<-started
	clk.advance(time.Second) // a token for the parked request
	errs, conns := make(chan error, 1), make(chan *Conn, 1)
	go func() {
		conn, err := p.GetContext(ctx, "pod")
		errs <- err
		conns <- conn
	}()
	select {
	case err := <-errs:
		t.Fatalf("the request did not park on the slot: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	return p, errs, conns
}

func TestConnPool_ParkedOnASlot_GivesUpAtItsDeadline(t *testing.T) {
	g := gateDials(t)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, errs, _ := parkOnTheSlot(t, g.started, ctx)

	assert.ErrorIs(t, <-errs, context.DeadlineExceeded)
	_, total := g.counts()
	assert.Equal(t, 1, total)
}

func TestConnPool_ParkedOnASlot_TakesAConnectionThatArrivedMeanwhile(t *testing.T) {
	g := gateDials(t)
	p, errs, conns := parkOnTheSlot(t, g.started, context.Background())
	idle := idleConn(t, time.Now())
	p.mu.Lock()
	p.pools["pod"] = append(p.pools["pod"], idle) // there, without waking the request
	p.mu.Unlock()

	close(g.release) // the slot frees: the parked request's turn
	require.NoError(t, <-errs)
	assert.Same(t, idle, <-conns)
	_, total := g.counts()
	assert.Equal(t, 1, total, "it took the connection instead of dialling")
}

// Requests parked behind a dial that fails do not dial in turn: the pod is
// backing off, so they fail at once.
func TestConnPool_ParkedOnASlot_FailsAtOnceWhenTheDialAheadFails(t *testing.T) {
	started, release := make(chan struct{}, 4), make(chan struct{})
	var dials atomic.Int32
	orig := dialTCP
	dialTCP = func(context.Context, *net.Dialer, string) (net.Conn, error) {
		dials.Add(1)
		started <- struct{}{}
		<-release
		return nil, errors.New("connection reset")
	}
	t.Cleanup(func() { dialTCP = orig })
	_, errs, _ := parkOnTheSlot(t, started, context.Background())

	close(release)
	assert.ErrorIs(t, <-errs, errDialBackingOff)
	assert.Equal(t, int32(1), dials.Load())
}
