package sdk

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Wire protocol constants — must match the Rust read server (LLD §4.1.1).
const (
	opSingle       = 0x01
	opBatch        = 0x02
	opStringSingle = 0x03
	opStringBatch  = 0x04
	keySize        = 12
)

// Conn is a single TCP connection to a OnyxDB read server.
type Conn struct {
	conn     net.Conn
	lastUsed time.Time // tracked for idle eviction

	// pending is the unread rest of a response whose request timed out after
	// it was fully written; broken means the stream is in an unknown state.
	// Both are reset by the next request.
	pending *response
	broken  bool
}

// Dial opens a TCP connection (TCP_NODELAY + keepalive) to a read server.
func Dial(addr string, timeout time.Duration) (*Conn, error) {
	return DialWithKeepalive(addr, timeout, 0, 0)
}

// DialWithKeepalive opens a TCP connection with configurable keepalive.
// keepaliveInterval=0 means use OS default (typically ~15s when enabled).
// keepaliveTimeout is currently not applied to the socket (any value, including
// 0, has no effect). It is kept so the signature and PoolConfig stay stable; dead
// peers are detected by the per-request deadline and idle eviction instead.
func DialWithKeepalive(addr string, dialTimeout, keepaliveInterval, keepaliveTimeout time.Duration) (*Conn, error) {
	return dialContext(context.Background(), addr, dialTimeout, keepaliveInterval)
}

// dialTCP is the connect itself, indirected so tests can stand in for a peer
// that never answers.
var dialTCP = func(ctx context.Context, d *net.Dialer, addr string) (net.Conn, error) {
	return d.DialContext(ctx, "tcp", addr)
}

// dialContext connects within the earlier of dialTimeout and ctx's deadline, so
// a request never waits on a connect longer than its own budget.
func dialContext(ctx context.Context, addr string, dialTimeout, keepaliveInterval time.Duration) (*Conn, error) {
	conn, err := dialTCP(ctx, &net.Dialer{Timeout: dialTimeout}, addr)
	if err != nil {
		return nil, fmt.Errorf("onyxdb dial %s: %w", addr, err)
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
		if keepaliveInterval > 0 {
			_ = tc.SetKeepAlivePeriod(keepaliveInterval)
		}
	}
	return &Conn{conn: conn, lastUsed: time.Now()}, nil
}

func (c *Conn) touch() { c.lastUsed = time.Now() }

// applyDeadline bounds the next request by ctx, and clears a deadline left by
// an earlier request when ctx has none: a reused connection must not inherit
// a deadline that has already passed.
func (c *Conn) applyDeadline(ctx context.Context) {
	deadline, _ := ctx.Deadline() // zero when ctx has none, which clears it
	_ = c.conn.SetWriteDeadline(deadline)
	_ = c.conn.SetReadDeadline(deadline)
}

// isTimeout reports whether err is a deadline expiring, as opposed to the
// peer closing or resetting the connection.
func isTimeout(err error) bool {
	var ne net.Error
	return errors.Is(err, os.ErrDeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout())
}

// roundTrip writes one request and reads its response. When the request's
// deadline fires after the whole request was written, the connection keeps
// its place in the unread response (c.pending) so ConnPool.Recycle can drain
// it and reuse the connection. Any other failure marks it broken.
func (c *Conn) roundTrip(ctx context.Context, req []byte, batch bool, name string) (*response, error) {
	c.pending, c.broken = nil, false
	c.applyDeadline(ctx)
	if n, err := c.conn.Write(req); err != nil {
		// A timeout before any byte left is harmless: the stream is still at a
		// request boundary. A partial write is not.
		c.broken = n > 0 || !isTimeout(err)
		return nil, fmt.Errorf("write %s request: %w", name, err)
	}
	r := newResponse(batch)
	if err := r.readFrom(c.conn); err != nil {
		if isTimeout(err) {
			c.pending = r.discardRest()
		} else {
			c.broken = true
		}
		return nil, r.wrap(name, err)
	}
	c.touch()
	return r, nil
}

// drain finishes reading the response a timed-out request left behind
// (c.pending, set and not broken: see ConnPool.Recycle), within timeout, and
// reports whether the connection is back at a request boundary.
func (c *Conn) drain(timeout time.Duration) bool {
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	err := c.pending.readFrom(c.conn)
	_ = c.conn.SetDeadline(time.Time{})
	if err != nil {
		c.broken = true
		return false
	}
	c.pending = nil
	c.touch()
	return true
}

// SingleLookup sends opcode 0x01 and returns the value, or ErrKeyNotFound.
//
// Request:  [1B op=0x01][12B key]
// Response: [1B found][4B len BE][value]
func (c *Conn) SingleLookup(ctx context.Context, key []byte) ([]byte, error) {
	var req [1 + keySize]byte
	req[0] = opSingle
	copy(req[1:], key)
	return c.single(ctx, req[:], "single")
}

// BatchLookup sends opcode 0x02 with N keys and returns N values in order
// (nil for misses).
//
// Request:  [1B op=0x02][2B N BE][12B × N]
// Response: [2B N BE][per-key: [1B found][4B len][value]]
func (c *Conn) BatchLookup(ctx context.Context, keys [][]byte) ([][]byte, error) {
	n := len(keys)
	if n == 0 {
		return nil, nil
	}

	reqBuf := make([]byte, 1+2+n*keySize)
	reqBuf[0] = opBatch
	binary.BigEndian.PutUint16(reqBuf[1:3], uint16(n))
	for i, k := range keys {
		off := 3 + i*keySize
		copy(reqBuf[off:off+keySize], k)
	}
	return c.batch(ctx, reqBuf, "batch")
}

// StringSingleLookup sends opcode 0x03 with a variable-length key and returns
// the value, or ErrKeyNotFound.
//
// Request:  [1B op=0x03][2B keyLen BE][key bytes]
// Response: [1B found][4B len BE][value]
func (c *Conn) StringSingleLookup(ctx context.Context, key []byte) ([]byte, error) {
	req := make([]byte, 1+2+len(key))
	req[0] = opStringSingle
	binary.BigEndian.PutUint16(req[1:3], uint16(len(key)))
	copy(req[3:], key)
	return c.single(ctx, req, "string single")
}

// StringBatchLookup sends opcode 0x04 with N variable-length keys and returns
// N values in order (nil for misses).
//
// Request:  [1B op=0x04][2B N BE][per-key: 2B keyLen BE + key bytes]
// Response: [2B N BE][per-key: [1B found][4B len][value]]
func (c *Conn) StringBatchLookup(ctx context.Context, keys [][]byte) ([][]byte, error) {
	n := len(keys)
	if n == 0 {
		return nil, nil
	}

	// Calculate total request size: 1 (op) + 2 (N) + sum(2 + len(k) for each key)
	reqSize := 3
	for _, k := range keys {
		reqSize += 2 + len(k)
	}
	reqBuf := make([]byte, reqSize)
	reqBuf[0] = opStringBatch
	binary.BigEndian.PutUint16(reqBuf[1:3], uint16(n))
	off := 3
	for _, k := range keys {
		binary.BigEndian.PutUint16(reqBuf[off:off+2], uint16(len(k)))
		copy(reqBuf[off+2:], k)
		off += 2 + len(k)
	}
	return c.batch(ctx, reqBuf, "string batch")
}

func (c *Conn) single(ctx context.Context, req []byte, name string) ([]byte, error) {
	r, err := c.roundTrip(ctx, req, false, name)
	if err != nil {
		return nil, err
	}
	if !r.found {
		return nil, ErrKeyNotFound
	}
	return r.value, nil
}

func (c *Conn) batch(ctx context.Context, req []byte, name string) ([][]byte, error) {
	r, err := c.roundTrip(ctx, req, true, name)
	if err != nil {
		return nil, err
	}
	return r.values, nil
}

// ── response parsing ─────────────────────────────────────────────────────────

type respStage uint8

const (
	stCount  respStage = iota // batch only: [2B N]
	stFound                   // [1B found]
	stLength                  // [4B len]
	stValue                   // value bytes
	stDone
)

// fieldLen is the wire size of each fixed-size field.
var fieldLen = [...]int{stCount: 2, stFound: 1, stLength: 4}

// response is the parse state of one response. A read cut short by a deadline
// keeps its place, so drain can finish the same response and leave the stream
// at a request boundary instead of the connection being closed.
type response struct {
	batch   bool
	discard bool // draining: read the rest, keep nothing
	stage   respStage
	field   [4]byte // the count, found byte or length being read
	got     int     // bytes of the current field or value read so far
	entries int     // values in the response; 1 for a single lookup
	done    int     // values fully read
	valLen  int
	val     []byte // the value being read, unless discarding

	values [][]byte // batch result, nil for misses
	found  bool     // single result
	value  []byte   // single result
}

func newResponse(batch bool) *response {
	if batch {
		return &response{batch: true, stage: stCount}
	}
	return &response{stage: stFound, entries: 1}
}

// discardRest switches to draining: whatever is left is read and dropped.
func (r *response) discardRest() *response {
	r.discard = true
	r.val, r.values, r.value = nil, nil, nil
	return r
}

// readFrom reads until the response is complete or src fails, keeping its
// place on failure.
func (r *response) readFrom(src io.Reader) error {
	var scratch [4096]byte
	for r.stage != stDone {
		var (
			n   int
			err error
		)
		if r.stage == stValue {
			dst := scratch[:min(len(scratch), r.valLen-r.got)]
			if !r.discard {
				dst = r.val[r.got:]
			}
			n, err = src.Read(dst)
			r.got += n
			if r.got == r.valLen {
				r.valueDone()
			}
		} else {
			want := fieldLen[r.stage]
			n, err = src.Read(r.field[r.got:want])
			r.got += n
			if r.got == want {
				r.fieldDone()
			}
		}
		if err != nil && r.stage != stDone {
			return err
		}
	}
	return nil
}

func (r *response) fieldDone() {
	r.got = 0
	switch r.stage {
	case stCount:
		r.entries = int(binary.BigEndian.Uint16(r.field[:2]))
		if !r.discard {
			r.values = make([][]byte, r.entries)
		}
		r.stage = stFound
		if r.entries == 0 {
			r.stage = stDone
		}
	case stFound:
		if r.field[0] == 0 {
			r.entryDone() // miss
			return
		}
		r.stage = stLength
	case stLength:
		r.valLen = int(binary.BigEndian.Uint32(r.field[:4]))
		if !r.discard {
			r.val = make([]byte, r.valLen)
		}
		r.stage = stValue
		if r.valLen == 0 {
			r.valueDone()
		}
	}
}

func (r *response) valueDone() {
	if !r.discard {
		if r.batch {
			r.values[r.done] = r.val
		} else {
			r.found, r.value = true, r.val
		}
	}
	r.val, r.got = nil, 0
	r.entryDone()
}

func (r *response) entryDone() {
	r.done++
	r.stage = stFound
	if r.done == r.entries {
		r.stage = stDone
	}
}

// wrap names where the read stopped, in the wording the SDK has always used.
func (r *response) wrap(name string, err error) error {
	if !r.batch {
		switch r.stage {
		case stFound:
			return fmt.Errorf("read %s found byte: %w", name, err)
		case stLength:
			return fmt.Errorf("read %s value length: %w", name, err)
		default:
			return fmt.Errorf("read %s value: %w", name, err)
		}
	}
	switch r.stage {
	case stCount:
		return fmt.Errorf("read %s response header: %w", name, err)
	case stFound:
		return fmt.Errorf("read %s result %d found: %w", name, r.done, err)
	case stLength:
		return fmt.Errorf("read %s result %d length: %w", name, r.done, err)
	default:
		return fmt.Errorf("read %s result %d value: %w", name, r.done, err)
	}
}

// BuildStringKey constructs an OFS-compatible string key as UTF-8 bytes:
// "<entityLabel>:<pkValues[0]>|<pkValues[1]>|..."
func BuildStringKey(entityLabel string, pkValues ...int64) []byte {
	key := entityLabel + ":"
	for i, v := range pkValues {
		if i > 0 {
			key += "|"
		}
		key += fmt.Sprintf("%d", v)
	}
	return []byte(key)
}

// Close closes the underlying TCP connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// ── PoolConfig ───────────────────────────────────────────────────────────────

// PoolConfig holds tuning parameters for the connection pool.
type PoolConfig struct {
	MinPerPod         int           // warm floor: pre-dialed idle connections (default 1)
	MaxPerPod         int           // pool ceiling (default 4)
	DialTimeout       time.Duration // TCP connect timeout (default 5s)
	IdleTimeout       time.Duration // evict connections idle longer than this (default 60s)
	IdleCheckInterval time.Duration // sweep interval for idle eviction (default 10s)
	KeepAliveInterval time.Duration // TCP keepalive probe interval (default 15s)
	KeepAliveTimeout  time.Duration // accepted but not applied to the socket (see DialWithKeepalive); default 5s
	// DrainTimeout bounds how long a connection whose request timed out may
	// take to deliver the late response before it is closed instead of being
	// reused (default 1s; negative closes every timed-out connection).
	DrainTimeout time.Duration
	// MaxDialsPerPod bounds concurrent dials to one pod; a request that finds
	// the pool empty with every dial slot busy waits for a returned connection
	// or a free slot, up to its own deadline (default 4).
	MaxDialsPerPod int
}

func (pc *PoolConfig) applyDefaults() {
	if pc.MinPerPod <= 0 {
		pc.MinPerPod = 1
	}
	if pc.MaxPerPod <= 0 {
		pc.MaxPerPod = 4
	}
	if pc.MaxPerPod < pc.MinPerPod {
		pc.MaxPerPod = pc.MinPerPod
	}
	if pc.DialTimeout <= 0 {
		pc.DialTimeout = 5 * time.Second
	}
	if pc.IdleTimeout <= 0 {
		pc.IdleTimeout = 60 * time.Second
	}
	if pc.IdleCheckInterval <= 0 {
		pc.IdleCheckInterval = 10 * time.Second
	}
	if pc.KeepAliveInterval <= 0 {
		pc.KeepAliveInterval = 15 * time.Second
	}
	if pc.KeepAliveTimeout <= 0 {
		pc.KeepAliveTimeout = 5 * time.Second
	}
	if pc.DrainTimeout == 0 {
		pc.DrainTimeout = time.Second
	}
	if pc.MaxDialsPerPod <= 0 {
		pc.MaxDialsPerPod = 4
	}
}

// ── ConnPool ─────────────────────────────────────────────────────────────────

// ConnPool keeps a bounded pool of connections per pod address with idle
// eviction and configurable keepalive.
type ConnPool struct {
	mu     sync.Mutex
	cfg    PoolConfig
	dialTO time.Duration // shortcut to cfg.DialTimeout
	pools  map[string][]*Conn
	closed bool
	cancel context.CancelFunc

	// dials holds one slot per dial in flight to a pod, MaxDialsPerPod deep.
	// wake is closed when a connection comes back to a pod's pool, waking
	// every request waiting for one.
	dials map[string]chan struct{}
	wake  map[string]chan struct{}

	// Metric callbacks (nil-safe). Set via SetMetrics after construction.
	timing   func(string, time.Duration, []string)
	count    func(string, int64, []string)
	baseTags []string
}

// NewConnPool creates a pool with the legacy maxPerPod-only API.
func NewConnPool(maxPerPod int) *ConnPool {
	cfg := PoolConfig{MaxPerPod: maxPerPod}
	cfg.applyDefaults()
	return newConnPoolInternal(cfg)
}

// NewConnPoolWithConfig creates a pool with full configuration.
func NewConnPoolWithConfig(cfg PoolConfig) *ConnPool {
	cfg.applyDefaults()
	return newConnPoolInternal(cfg)
}

func newConnPoolInternal(cfg PoolConfig) *ConnPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &ConnPool{
		cfg:    cfg,
		dialTO: cfg.DialTimeout,
		pools:  make(map[string][]*Conn),
		cancel: cancel,
		dials:  make(map[string]chan struct{}),
		wake:   make(map[string]chan struct{}),
	}
	go p.idleEvictor(ctx)
	return p
}

// SetMetrics wires optional metric callbacks into the pool.
func (p *ConnPool) SetMetrics(
	timing func(string, time.Duration, []string),
	count func(string, int64, []string),
	baseTags []string,
) {
	p.timing = timing
	p.count = count
	p.baseTags = baseTags
}

func (p *ConnPool) poolTags(extra ...string) []string {
	tags := make([]string, len(p.baseTags)+len(extra))
	copy(tags, p.baseTags)
	copy(tags[len(p.baseTags):], extra)
	return tags
}

func (p *ConnPool) emitTiming(name string, value time.Duration, tags []string) {
	if p.timing != nil {
		p.timing(name, value, tags)
	}
}

func (p *ConnPool) emitCount(name string, value int64, tags []string) {
	if p.count != nil {
		p.count(name, value, tags)
	}
}

// idleEvictor periodically scans all pools and closes connections that have
// been idle longer than cfg.IdleTimeout, while keeping at least MinPerPod
// connections alive per pod.
func (p *ConnPool) idleEvictor(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.IdleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.evictIdle()
		}
	}
}

func (p *ConnPool) evictIdle() {
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	for addr, conns := range p.pools {
		kept := make([]*Conn, 0, len(conns))
		for _, c := range conns {
			if now.Sub(c.lastUsed) > p.cfg.IdleTimeout && len(kept) >= p.cfg.MinPerPod {
				_ = c.Close()
				log.Debug().Str("addr", addr).Msg("pool: evicted idle connection")
				p.emitCount(MetricPoolIdleEvicted, 1, p.baseTags)
			} else {
				kept = append(kept, c)
			}
		}
		if len(kept) == 0 {
			delete(p.pools, addr)
		} else {
			p.pools[addr] = kept
		}
	}
}

// Get returns a pooled connection for addr, dialing a new one if the pool is empty.
// The dial is bounded by the pool's DialTimeout only; request paths use
// GetContext so a dial never outlives the request.
func (p *ConnPool) Get(addr string) (*Conn, error) {
	return p.GetContext(context.Background(), addr)
}

// GetContext is Get with the dial bounded by ctx as well: a new connection must
// connect within the earlier of DialTimeout and ctx's deadline. A dial into a
// peer that drops SYNs (a full accept queue, a vanished pod IP) then fails at
// the request's deadline instead of after the kernel's ~1 s SYN retransmit or
// the 5 s DialTimeout.
func (p *ConnPool) GetContext(ctx context.Context, addr string) (*Conn, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, fmt.Errorf("onyxdb: connection pool closed")
		}
		if c := p.popLocked(addr); c != nil {
			p.mu.Unlock()
			p.emitCount(MetricPoolGet, 1, p.poolTags("result:hit"))
			return c, nil
		}
		slots := p.dials[addr]
		if slots == nil {
			slots = make(chan struct{}, p.cfg.MaxDialsPerPod)
			p.dials[addr] = slots
		}
		wake := p.wake[addr]
		if wake == nil {
			wake = make(chan struct{})
			p.wake[addr] = wake
		}
		p.mu.Unlock()

		// Every slot busy: wait for a returned connection or a free slot rather
		// than dial past the bound. A pod that has stopped answering otherwise
		// gets one SYN per request until its accept queue overflows.
		select {
		case slots <- struct{}{}:
		case <-wake:
			continue
		case <-ctx.Done():
			p.emitCount(MetricPoolGet, 1, p.poolTags("result:dial_wait_expired"))
			return nil, fmt.Errorf("onyxdb: no connection to %s before the request deadline, %d dials already in flight: %w",
				addr, cap(slots), ctx.Err())
		}

		// A connection may have come back while this request waited.
		p.mu.Lock()
		c := p.popLocked(addr)
		p.mu.Unlock()
		if c != nil {
			<-slots
			p.emitCount(MetricPoolGet, 1, p.poolTags("result:hit"))
			return c, nil
		}
		conn, err := p.dial(ctx, addr)
		<-slots
		return conn, err
	}
}

// popLocked takes the most recently returned connection for addr, or nil.
// Caller holds p.mu.
func (p *ConnPool) popLocked(addr string) *Conn {
	conns := p.pools[addr]
	if len(conns) == 0 {
		return nil
	}
	c := conns[len(conns)-1]
	p.pools[addr] = conns[:len(conns)-1]
	return c
}

func (p *ConnPool) dial(ctx context.Context, addr string) (*Conn, error) {
	start := time.Now()
	conn, err := dialContext(ctx, addr, p.cfg.DialTimeout, p.cfg.KeepAliveInterval)
	if err != nil {
		p.emitCount(MetricPoolGet, 1, p.poolTags("result:error"))
		return nil, err
	}
	p.emitCount(MetricPoolGet, 1, p.poolTags("result:dial"))
	p.emitTiming(MetricPoolDialLatency, time.Since(start), p.baseTags)
	return conn, nil
}

// Put returns a connection to the pool, or closes it if the pool is full/unknown.
func (p *ConnPool) Put(addr string, conn *Conn) {
	conn.touch()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = conn.Close()
		return
	}
	conns := p.pools[addr]
	if len(conns) >= p.cfg.MaxPerPod {
		p.mu.Unlock()
		_ = conn.Close()
		p.emitCount(MetricPoolOverflow, 1, p.baseTags)
		return
	}
	p.pools[addr] = append(conns, conn)
	if wake := p.wake[addr]; wake != nil {
		close(wake)
		delete(p.wake, addr)
	}
	p.mu.Unlock()
}

// Recycle takes back a connection whose lookup failed (any error other than
// ErrKeyNotFound). Callers use it instead of Close.
//
// A request that only timed out leaves its response on the wire; closing the
// connection for that turned every slow moment on a pod into a reconnect per
// request. Recycle instead drains the late response in the background, within
// DrainTimeout, and pools the connection again; one whose stream is in an
// unknown state (peer closed, partial write), or that cannot be drained in
// time, is closed.
func (p *ConnPool) Recycle(addr string, conn *Conn) {
	if conn.pending == nil && !conn.broken {
		p.Put(addr, conn) // nothing was sent: still at a request boundary
		return
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if conn.broken || p.cfg.DrainTimeout < 0 || closed {
		_ = conn.Close()
		return
	}
	go func() {
		if conn.drain(p.cfg.DrainTimeout) {
			p.Put(addr, conn)
			p.emitCount(MetricPoolDrain, 1, p.poolTags("result:reused"))
			return
		}
		_ = conn.Close()
		p.emitCount(MetricPoolDrain, 1, p.poolTags("result:closed"))
	}()
}

// Prune closes and removes pools for any address not in the live set. Called
// after a topology change so connections to scaled-down / no-longer-warm pods
// are released rather than lingering.
func (p *ConnPool) Prune(live []string) {
	liveSet := make(map[string]struct{}, len(live))
	for _, a := range live {
		liveSet[a] = struct{}{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	for addr, conns := range p.pools {
		if _, ok := liveSet[addr]; ok {
			continue
		}
		for _, c := range conns {
			_ = c.Close()
		}
		delete(p.pools, addr)
	}
	for addr := range p.dials {
		if _, ok := liveSet[addr]; !ok {
			delete(p.dials, addr) // a dial still in flight releases into its own slot set
		}
	}
	for addr, wake := range p.wake {
		if _, ok := liveSet[addr]; !ok {
			close(wake)
			delete(p.wake, addr)
		}
	}
}

// Close closes all pooled connections, stops the idle evictor, and marks the
// pool closed.
func (p *ConnPool) Close() {
	p.cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	for addr, conns := range p.pools {
		for _, c := range conns {
			_ = c.Close()
		}
		delete(p.pools, addr)
	}
	for addr, wake := range p.wake {
		close(wake) // waiters retry and see the pool closed
		delete(p.wake, addr)
	}
}
