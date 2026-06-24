package sdk

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
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
	conn net.Conn
}

// Dial opens a TCP connection (TCP_NODELAY) to a read server.
func Dial(addr string, timeout time.Duration) (*Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("mnemo dial %s: %w", addr, err)
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}
	return &Conn{conn: conn}, nil
}

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

// ConnPool keeps a bounded pool of connections per pod address.
type ConnPool struct {
	mu        sync.Mutex
	maxPerPod int
	dialTO    time.Duration
	pools     map[string]chan *Conn
	closed    bool
}

// NewConnPool creates a pool with maxPerPod buffered connections per pod.
func NewConnPool(maxPerPod int) *ConnPool {
	if maxPerPod <= 0 {
		maxPerPod = 4
	}
	return &ConnPool{
		maxPerPod: maxPerPod,
		dialTO:    5 * time.Second,
		pools:     make(map[string]chan *Conn),
	}
}

// Get returns a pooled connection for addr, dialing a new one if the pool is empty.
func (p *ConnPool) Get(addr string) (*Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("mnemo: connection pool closed")
	}
	ch, ok := p.pools[addr]
	if !ok {
		ch = make(chan *Conn, p.maxPerPod)
		p.pools[addr] = ch
	}
	p.mu.Unlock()

	select {
	case conn := <-ch:
		return conn, nil
	default:
		return Dial(addr, p.dialTO)
	}
}

// Put returns a connection to the pool, or closes it if the pool is full/unknown.
func (p *ConnPool) Put(addr string, conn *Conn) {
	p.mu.Lock()
	ch, ok := p.pools[addr]
	closed := p.closed
	p.mu.Unlock()

	if closed || !ok {
		_ = conn.Close()
		return
	}
	select {
	case ch <- conn:
	default:
		_ = conn.Close() // pool full
	}
}

// Prune closes and removes pools for any address not in the live set. Called
// after a DNS refresh so connections to scaled-down / no-longer-warm pods are
// released rather than lingering until their TTL.
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
	for addr, ch := range p.pools {
		if _, ok := liveSet[addr]; ok {
			continue
		}
		close(ch)
		for conn := range ch {
			_ = conn.Close()
		}
		delete(p.pools, addr)
	}
}

// Close closes all pooled connections and marks the pool closed.
func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	for addr, ch := range p.pools {
		close(ch)
		for conn := range ch {
			_ = conn.Close()
		}
		delete(p.pools, addr)
	}
}
