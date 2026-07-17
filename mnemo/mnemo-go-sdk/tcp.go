package sdk

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
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

// Conn is a single TCP connection to a mNemo read server.
type Conn struct {
	conn     net.Conn
	lastUsed time.Time // tracked for idle eviction
}

// Dial opens a TCP connection (TCP_NODELAY + keepalive) to a read server.
func Dial(addr string, timeout time.Duration) (*Conn, error) {
	return DialWithKeepalive(addr, timeout, 0, 0)
}

// DialWithKeepalive opens a TCP connection with configurable keepalive.
// keepaliveInterval=0 means use OS default (typically ~15s when enabled).
// keepaliveTimeout=0 means don't set it explicitly.
func DialWithKeepalive(addr string, dialTimeout, keepaliveInterval, keepaliveTimeout time.Duration) (*Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("mnemo dial %s: %w", addr, err)
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

func (c *Conn) applyDeadline(ctx context.Context) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(deadline)
		_ = c.conn.SetReadDeadline(deadline)
	}
}

// SingleLookup sends opcode 0x01 and returns the value, or ErrKeyNotFound.
//
// Request:  [1B op=0x01][12B key]
// Response: [1B found][4B len BE][value]
func (c *Conn) SingleLookup(ctx context.Context, key []byte) ([]byte, error) {
	var req [1 + keySize]byte
	req[0] = opSingle
	copy(req[1:], key)

	c.applyDeadline(ctx)

	if _, err := c.conn.Write(req[:]); err != nil {
		return nil, fmt.Errorf("write single request: %w", err)
	}

	var found [1]byte
	if _, err := io.ReadFull(c.conn, found[:]); err != nil {
		return nil, fmt.Errorf("read single found byte: %w", err)
	}
	if found[0] == 0 {
		return nil, ErrKeyNotFound
	}

	var lenBuf [4]byte
	if _, err := io.ReadFull(c.conn, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read single value length: %w", err)
	}
	val := make([]byte, binary.BigEndian.Uint32(lenBuf[:]))
	if _, err := io.ReadFull(c.conn, val); err != nil {
		return nil, fmt.Errorf("read single value: %w", err)
	}
	c.touch()
	return val, nil
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

	c.applyDeadline(ctx)

	if _, err := c.conn.Write(reqBuf); err != nil {
		return nil, fmt.Errorf("write batch request: %w", err)
	}

	var respHeader [2]byte
	if _, err := io.ReadFull(c.conn, respHeader[:]); err != nil {
		return nil, fmt.Errorf("read batch response header: %w", err)
	}
	respN := int(binary.BigEndian.Uint16(respHeader[:]))

	results := make([][]byte, respN)
	for i := 0; i < respN; i++ {
		var found [1]byte
		if _, err := io.ReadFull(c.conn, found[:]); err != nil {
			return nil, fmt.Errorf("read batch result %d found: %w", i, err)
		}
		if found[0] == 0 {
			continue // miss → nil
		}
		var lenBuf [4]byte
		if _, err := io.ReadFull(c.conn, lenBuf[:]); err != nil {
			return nil, fmt.Errorf("read batch result %d length: %w", i, err)
		}
		val := make([]byte, binary.BigEndian.Uint32(lenBuf[:]))
		if _, err := io.ReadFull(c.conn, val); err != nil {
			return nil, fmt.Errorf("read batch result %d value: %w", i, err)
		}
		results[i] = val
	}
	c.touch()
	return results, nil
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

	c.applyDeadline(ctx)

	if _, err := c.conn.Write(req); err != nil {
		return nil, fmt.Errorf("write string single request: %w", err)
	}

	var found [1]byte
	if _, err := io.ReadFull(c.conn, found[:]); err != nil {
		return nil, fmt.Errorf("read string single found byte: %w", err)
	}
	if found[0] == 0 {
		return nil, ErrKeyNotFound
	}

	var lenBuf [4]byte
	if _, err := io.ReadFull(c.conn, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read string single value length: %w", err)
	}
	val := make([]byte, binary.BigEndian.Uint32(lenBuf[:]))
	if _, err := io.ReadFull(c.conn, val); err != nil {
		return nil, fmt.Errorf("read string single value: %w", err)
	}
	c.touch()
	return val, nil
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

	c.applyDeadline(ctx)

	if _, err := c.conn.Write(reqBuf); err != nil {
		return nil, fmt.Errorf("write string batch request: %w", err)
	}

	var respHeader [2]byte
	if _, err := io.ReadFull(c.conn, respHeader[:]); err != nil {
		return nil, fmt.Errorf("read string batch response header: %w", err)
	}
	respN := int(binary.BigEndian.Uint16(respHeader[:]))

	results := make([][]byte, respN)
	for i := 0; i < respN; i++ {
		var found [1]byte
		if _, err := io.ReadFull(c.conn, found[:]); err != nil {
			return nil, fmt.Errorf("read string batch result %d found: %w", i, err)
		}
		if found[0] == 0 {
			continue // miss → nil
		}
		var lenBuf [4]byte
		if _, err := io.ReadFull(c.conn, lenBuf[:]); err != nil {
			return nil, fmt.Errorf("read string batch result %d length: %w", i, err)
		}
		val := make([]byte, binary.BigEndian.Uint32(lenBuf[:]))
		if _, err := io.ReadFull(c.conn, val); err != nil {
			return nil, fmt.Errorf("read string batch result %d value: %w", i, err)
		}
		results[i] = val
	}
	c.touch()
	return results, nil
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
	MinPerPod          int           // warm floor: pre-dialed idle connections (default 1)
	MaxPerPod          int           // pool ceiling (default 4)
	DialTimeout        time.Duration // TCP connect timeout (default 5s)
	IdleTimeout        time.Duration // evict connections idle longer than this (default 60s)
	IdleCheckInterval  time.Duration // sweep interval for idle eviction (default 10s)
	KeepAliveInterval  time.Duration // TCP keepalive probe interval (default 15s)
	KeepAliveTimeout   time.Duration // keepalive timeout (default 5s)
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
func (p *ConnPool) Get(addr string) (*Conn, error) {
	getStart := time.Now()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("mnemo: connection pool closed")
	}
	conns := p.pools[addr]
	if len(conns) > 0 {
		c := conns[len(conns)-1]
		p.pools[addr] = conns[:len(conns)-1]
		p.mu.Unlock()
		p.emitCount(MetricPoolGet, 1, p.poolTags("result:hit"))
		p.emitTiming(MetricPoolGetLatency, time.Since(getStart), p.baseTags)
		return c, nil
	}
	p.mu.Unlock()

	dialStart := time.Now()
	conn, err := DialWithKeepalive(addr, p.cfg.DialTimeout, p.cfg.KeepAliveInterval, p.cfg.KeepAliveTimeout)
	if err != nil {
		p.emitCount(MetricPoolGet, 1, p.poolTags("result:error"))
		p.emitTiming(MetricPoolGetLatency, time.Since(getStart), p.baseTags)
		return nil, err
	}
	dialElapsed := time.Since(dialStart)
	p.emitCount(MetricPoolGet, 1, p.poolTags("result:dial"))
	p.emitTiming(MetricPoolDialLatency, dialElapsed, p.baseTags)
	p.emitTiming(MetricPoolGetLatency, time.Since(getStart), p.baseTags)
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
	p.mu.Unlock()
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
}
