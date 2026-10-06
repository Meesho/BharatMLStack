package sdk

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── fake read server ──────────────────────────────────────────────────────────

// fakeServer answers the OnyxDB binary protocol from an in-memory key→value map.
func fakeServer(conn net.Conn, data map[string][]byte) {
	defer conn.Close()
	for {
		var op [1]byte
		if _, err := io.ReadFull(conn, op[:]); err != nil {
			return
		}
		switch op[0] {
		case opSingle:
			key := make([]byte, keySize)
			if _, err := io.ReadFull(conn, key); err != nil {
				return
			}
			writeValue(conn, data[string(key)])
		case opBatch:
			var nbuf [2]byte
			if _, err := io.ReadFull(conn, nbuf[:]); err != nil {
				return
			}
			n := int(binary.BigEndian.Uint16(nbuf[:]))
			keys := make([][]byte, n)
			for i := 0; i < n; i++ {
				k := make([]byte, keySize)
				if _, err := io.ReadFull(conn, k); err != nil {
					return
				}
				keys[i] = k
			}
			resp := make([]byte, 2)
			binary.BigEndian.PutUint16(resp, uint16(n))
			conn.Write(resp)
			for _, k := range keys {
				writeValueBytes(conn, data[string(k)])
			}
		case opStringSingle:
			var lenBuf [2]byte
			if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
				return
			}
			keyLen := int(binary.BigEndian.Uint16(lenBuf[:]))
			key := make([]byte, keyLen)
			if _, err := io.ReadFull(conn, key); err != nil {
				return
			}
			writeValue(conn, data[string(key)])
		case opStringBatch:
			var nbuf [2]byte
			if _, err := io.ReadFull(conn, nbuf[:]); err != nil {
				return
			}
			n := int(binary.BigEndian.Uint16(nbuf[:]))
			keys := make([][]byte, n)
			for i := 0; i < n; i++ {
				var klen [2]byte
				if _, err := io.ReadFull(conn, klen[:]); err != nil {
					return
				}
				k := make([]byte, binary.BigEndian.Uint16(klen[:]))
				if _, err := io.ReadFull(conn, k); err != nil {
					return
				}
				keys[i] = k
			}
			resp := make([]byte, 2)
			binary.BigEndian.PutUint16(resp, uint16(n))
			conn.Write(resp)
			for _, k := range keys {
				writeValueBytes(conn, data[string(k)])
			}
		default:
			return
		}
	}
}

func writeValue(conn net.Conn, val []byte) {
	if val == nil {
		conn.Write([]byte{0})
		return
	}
	buf := make([]byte, 1+4+len(val))
	buf[0] = 1
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(val)))
	copy(buf[5:], val)
	conn.Write(buf)
}

// writeValueBytes is the per-key body inside a batch response (no leading count).
func writeValueBytes(conn net.Conn, val []byte) {
	writeValue(conn, val)
}

// pipeConn wires a Conn to a fakeServer over net.Pipe.
func pipeConn(data map[string][]byte) *Conn {
	client, server := net.Pipe()
	go fakeServer(server, data)
	return &Conn{conn: client, lastUsed: time.Now()}
}

func key12(s string) []byte {
	k := make([]byte, keySize)
	copy(k, s)
	return k
}

// ── SingleLookup ────────────────────────────────────────────────────────────

func TestSingleLookup_Hit(t *testing.T) {
	k := key12("key1")
	c := pipeConn(map[string][]byte{string(k): []byte("value1")})
	defer c.Close()

	val, err := c.SingleLookup(context.Background(), k)
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), val)
}

func TestSingleLookup_Miss(t *testing.T) {
	c := pipeConn(map[string][]byte{})
	defer c.Close()

	_, err := c.SingleLookup(context.Background(), key12("nope"))
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSingleLookup_WithDeadline(t *testing.T) {
	k := key12("k")
	c := pipeConn(map[string][]byte{string(k): []byte("v")})
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	val, err := c.SingleLookup(ctx, k)
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), val)
}

func TestSingleLookup_WriteError(t *testing.T) {
	client, server := net.Pipe()
	server.Close() // closed → write fails
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.SingleLookup(context.Background(), key12("k"))
	assert.Error(t, err)
}

func TestSingleLookup_ReadHeaderError(t *testing.T) {
	client, server := net.Pipe()
	// Server reads the request then closes without responding.
	go func() {
		buf := make([]byte, 1+keySize)
		io.ReadFull(server, buf)
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.SingleLookup(context.Background(), key12("k"))
	assert.Error(t, err)
}

func TestSingleLookup_ReadLengthError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+keySize)
		io.ReadFull(server, buf)
		server.Write([]byte{1}) // found=1 but no length follows
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.SingleLookup(context.Background(), key12("k"))
	assert.Error(t, err)
}

func TestSingleLookup_ReadValueError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+keySize)
		io.ReadFull(server, buf)
		hdr := []byte{1, 0, 0, 0, 10} // found=1, len=10, but no value
		server.Write(hdr)
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.SingleLookup(context.Background(), key12("k"))
	assert.Error(t, err)
}

// ── BatchLookup ───────────────────────────────────────────────────────────────

func TestBatchLookup_Empty(t *testing.T) {
	c := pipeConn(map[string][]byte{})
	defer c.Close()
	vals, err := c.BatchLookup(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestBatchLookup_HitAndMiss(t *testing.T) {
	k1, k2 := key12("k1"), key12("k2")
	c := pipeConn(map[string][]byte{string(k1): []byte("v1")})
	defer c.Close()

	vals, err := c.BatchLookup(context.Background(), [][]byte{k1, k2})
	require.NoError(t, err)
	require.Len(t, vals, 2)
	assert.Equal(t, []byte("v1"), vals[0])
	assert.Nil(t, vals[1])
}

func TestBatchLookup_WriteError(t *testing.T) {
	client, server := net.Pipe()
	server.Close()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.BatchLookup(context.Background(), [][]byte{key12("k")})
	assert.Error(t, err)
}

func TestBatchLookup_ReadHeaderError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+keySize)
		io.ReadFull(server, buf)
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.BatchLookup(context.Background(), [][]byte{key12("k")})
	assert.Error(t, err)
}

func TestBatchLookup_ReadFoundError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+keySize)
		io.ReadFull(server, buf)
		server.Write([]byte{0, 1}) // N=1, then close before per-key body
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.BatchLookup(context.Background(), [][]byte{key12("k")})
	assert.Error(t, err)
}

func TestBatchLookup_ReadLengthError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+keySize)
		io.ReadFull(server, buf)
		server.Write([]byte{0, 1, 1}) // N=1, found=1, no length
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.BatchLookup(context.Background(), [][]byte{key12("k")})
	assert.Error(t, err)
}

func TestBatchLookup_ReadValueError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+keySize)
		io.ReadFull(server, buf)
		server.Write([]byte{0, 1, 1, 0, 0, 0, 5}) // N=1, found=1, len=5, no value
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.BatchLookup(context.Background(), [][]byte{key12("k")})
	assert.Error(t, err)
}

// ── StringSingleLookup ──────────────────────────────────────────────────────

func TestStringSingleLookup_WriteError(t *testing.T) {
	client, server := net.Pipe()
	server.Close()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringSingleLookup(context.Background(), []byte("k"))
	assert.Error(t, err)
}

func TestStringSingleLookup_ReadHeaderError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+1) // op + keyLen + key
		io.ReadFull(server, buf)
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringSingleLookup(context.Background(), []byte("k"))
	assert.Error(t, err)
}

func TestStringSingleLookup_ReadLengthError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+1) // op + keyLen + key
		io.ReadFull(server, buf)
		server.Write([]byte{1}) // found=1 but no length
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringSingleLookup(context.Background(), []byte("k"))
	assert.Error(t, err)
}

func TestStringSingleLookup_ReadValueError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 1+2+1) // op + keyLen + key
		io.ReadFull(server, buf)
		server.Write([]byte{1, 0, 0, 0, 10}) // found=1, len=10, no value
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringSingleLookup(context.Background(), []byte("k"))
	assert.Error(t, err)
}

func TestStringSingleLookup_Hit(t *testing.T) {
	k := []byte("catalog__user_geohash_1_3:105959719|4236")
	c := pipeConn(map[string][]byte{string(k): []byte("value1")})
	defer c.Close()

	val, err := c.StringSingleLookup(context.Background(), k)
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), val)
}

func TestStringSingleLookup_Miss(t *testing.T) {
	c := pipeConn(map[string][]byte{})
	defer c.Close()

	_, err := c.StringSingleLookup(context.Background(), []byte("no:such|key"))
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

// ── StringBatchLookup ───────────────────────────────────────────────────────

func TestStringBatchLookup_WriteError(t *testing.T) {
	client, server := net.Pipe()
	server.Close()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
	assert.Error(t, err)
}

func TestStringBatchLookup_ReadHeaderError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 256)
		io.ReadFull(server, buf[:1+2+2+1]) // op + N + keyLen + key
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
	assert.Error(t, err)
}

func TestStringBatchLookup_ReadFoundError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 256)
		io.ReadFull(server, buf[:1+2+2+1]) // op + N + keyLen + key
		server.Write([]byte{0, 1})          // N=1, then close before found byte
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
	assert.Error(t, err)
}

func TestStringBatchLookup_ReadLengthError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 256)
		io.ReadFull(server, buf[:1+2+2+1]) // op + N + keyLen + key
		server.Write([]byte{0, 1, 1})      // N=1, found=1, no length
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
	assert.Error(t, err)
}

func TestStringBatchLookup_ReadValueError(t *testing.T) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 256)
		io.ReadFull(server, buf[:1+2+2+1])           // op + N + keyLen + key
		server.Write([]byte{0, 1, 1, 0, 0, 0, 5})   // N=1, found=1, len=5, no value
		server.Close()
	}()
	c := &Conn{conn: client, lastUsed: time.Now()}
	_, err := c.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
	assert.Error(t, err)
}

func TestStringBatchLookup_Empty(t *testing.T) {
	c := pipeConn(map[string][]byte{})
	defer c.Close()
	vals, err := c.StringBatchLookup(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestStringBatchLookup_HitAndMiss(t *testing.T) {
	k1 := []byte("catalog__user_geohash_1_3:136588307|4205")
	k2 := []byte("catalog__user_geohash_1_3:999999999|1")
	c := pipeConn(map[string][]byte{string(k1): []byte("v1")})
	defer c.Close()

	vals, err := c.StringBatchLookup(context.Background(), [][]byte{k1, k2})
	require.NoError(t, err)
	require.Len(t, vals, 2)
	assert.Equal(t, []byte("v1"), vals[0])
	assert.Nil(t, vals[1])
}

// ── BuildStringKey ──────────────────────────────────────────────────────────

func TestBuildStringKey(t *testing.T) {
	key := BuildStringKey("catalog__user_geohash_1_3", 105959719, 4236)
	assert.Equal(t, "catalog__user_geohash_1_3:105959719|4236", string(key))
}

func TestBuildStringKey_ZeroValues(t *testing.T) {
	key := BuildStringKey("catalog__user_geohash_1_3", 0, 0)
	assert.Equal(t, "catalog__user_geohash_1_3:0|0", string(key))
}

// ── Dial + ConnPool ──────────────────────────────────────────────────────────

func TestDial_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	c, err := Dial(ln.Addr().String(), time.Second)
	require.NoError(t, err)
	assert.NoError(t, c.Close())
}

func TestDial_Failure(t *testing.T) {
	// Reserve a port then close it so the dial is refused.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	_, err := Dial(addr, 200*time.Millisecond)
	assert.Error(t, err)
}

func TestConnPool_GetDialsNew(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	p := NewConnPool(2)
	defer p.Close()
	conn, err := p.Get(ln.Addr().String())
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestConnPool_PutAndReuse(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn
		}
	}()

	p := NewConnPool(2)
	defer p.Close()
	addr := ln.Addr().String()

	conn, err := p.Get(addr)
	require.NoError(t, err)
	p.Put(addr, conn)

	// Next Get should return the pooled connection (same pointer).
	conn2, err := p.Get(addr)
	require.NoError(t, err)
	assert.Same(t, conn, conn2)
}

func TestConnPool_PutUnknownAddrAcceptsConn(t *testing.T) {
	client, _ := net.Pipe()
	p := NewConnPool(2)
	defer p.Close()
	// Put to a new address — pool creates an entry for it.
	p.Put("never-Get-this", &Conn{conn: client, lastUsed: time.Now()})
	p.mu.Lock()
	_, exists := p.pools["never-Get-this"]
	p.mu.Unlock()
	assert.True(t, exists)
}

func TestConnPool_PutFullClosesConn(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn
		}
	}()

	p := NewConnPool(1)
	defer p.Close()
	addr := ln.Addr().String()

	// Prime the pool channel by Get→Put once.
	c0, _ := p.Get(addr)
	p.Put(addr, c0) // pool now has 1 (full)

	// Put a second connection → pool full → it gets closed.
	c1, _ := p.Get(addr) // takes the pooled one
	c2, _ := Dial(addr, time.Second)
	p.Put(addr, c1)
	p.Put(addr, c2) // full → closed
}

func TestConnPool_GetAfterClose(t *testing.T) {
	p := NewConnPool(2)
	p.Close()
	_, err := p.Get("127.0.0.1:1")
	assert.Error(t, err)
}

func TestConnPool_PutAfterClose(t *testing.T) {
	client, _ := net.Pipe()
	p := NewConnPool(2)
	p.Close()
	p.Put("addr", &Conn{conn: client}) // closed pool → conn closed, no panic
}

func TestConnPool_DoubleClose(t *testing.T) {
	p := NewConnPool(2)
	p.Close()
	p.Close() // idempotent
}

func TestNewConnPool_DefaultsOnNonPositive(t *testing.T) {
	p := NewConnPool(0)
	defer p.Close()
	assert.Equal(t, 4, p.cfg.MaxPerPod)
}

func TestConnPool_PruneClosesDeadPods(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn
		}
	}()
	addr := ln.Addr().String()

	p := NewConnPool(2)
	defer p.Close()

	// Establish a pool for addr.
	c, err := p.Get(addr)
	require.NoError(t, err)
	p.Put(addr, c)

	// Prune with a live set that excludes addr → its pool is closed/removed.
	p.Prune([]string{"some-other:9091"})
	p.mu.Lock()
	_, exists := p.pools[addr]
	p.mu.Unlock()
	assert.False(t, exists, "pruned pool should be removed")
}

func TestConnPool_PruneKeepsLivePods(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn
		}
	}()
	addr := ln.Addr().String()

	p := NewConnPool(2)
	defer p.Close()
	c, _ := p.Get(addr)
	p.Put(addr, c)

	p.Prune([]string{addr}) // addr is live → kept
	p.mu.Lock()
	_, exists := p.pools[addr]
	p.mu.Unlock()
	assert.True(t, exists, "live pool should be kept")
}

func TestConnPool_PruneAfterClose_NoOp(t *testing.T) {
	p := NewConnPool(2)
	p.Close()
	p.Prune([]string{"x:1"}) // closed pool → no panic
}

// ── PoolConfig + NewConnPoolWithConfig ──────────────────────────────────────

func TestPoolConfig_Defaults(t *testing.T) {
	pc := PoolConfig{}
	pc.applyDefaults()
	assert.Equal(t, 1, pc.MinPerPod)
	assert.Equal(t, 4, pc.MaxPerPod)
	assert.Equal(t, 5*time.Second, pc.DialTimeout)
	assert.Equal(t, 60*time.Second, pc.IdleTimeout)
	assert.Equal(t, 10*time.Second, pc.IdleCheckInterval)
	assert.Equal(t, 15*time.Second, pc.KeepAliveInterval)
	assert.Equal(t, 5*time.Second, pc.KeepAliveTimeout)
}

func TestPoolConfig_MaxClampedToMin(t *testing.T) {
	pc := PoolConfig{MinPerPod: 8, MaxPerPod: 2}
	pc.applyDefaults()
	assert.Equal(t, 8, pc.MaxPerPod) // clamped up to MinPerPod
}

// The background sweeper closes connections idle past IdleTimeout but never
// takes a pod below MinPerPod. A zero MinPerPod means "unset" and is defaulted
// to 1 by applyDefaults, so one connection always survives the sweep.
func TestNewConnPoolWithConfig_IdleEviction(t *testing.T) {
	addr := holdingListener(t)
	mc := &metricCollector{}
	p := NewConnPoolWithConfig(PoolConfig{
		MaxPerPod:         4,
		MinPerPod:         1,
		IdleTimeout:       50 * time.Millisecond,
		IdleCheckInterval: 10 * time.Millisecond,
		DialTimeout:       time.Second,
	})
	p.SetMetrics(mc.timing, mc.count, []string{"tenant:t", "store:s"})
	defer p.Close()

	conns := make([]*Conn, 3)
	for i := range conns {
		c, err := p.Get(addr)
		require.NoError(t, err)
		conns[i] = c
	}
	for _, c := range conns {
		p.Put(addr, c)
	}

	require.Eventually(t, func() bool {
		return len(mc.findAll(MetricPoolIdleEvicted)) == 2
	}, 3*time.Second, 5*time.Millisecond, "the two connections above the floor should be evicted")

	p.mu.Lock()
	remaining := append([]*Conn(nil), p.pools[addr]...)
	p.mu.Unlock()
	require.Len(t, remaining, 1, "MinPerPod=1 keeps one idle connection")
	assert.Same(t, conns[0], remaining[0])
	for _, evicted := range conns[1:] {
		_, err := evicted.conn.Write([]byte{0})
		assert.ErrorIs(t, err, net.ErrClosed, "evicted connection must be closed")
	}
}

// holdingListener accepts TCP connections and keeps them open until the test
// ends, so pooled connections stay usable.
func holdingListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var held []net.Conn
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			held = append(held, conn)
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		<-done
		for _, c := range held {
			c.Close()
		}
	})
	return ln.Addr().String()
}

// refusedAddr returns a loopback address with nothing listening on it.
func refusedAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

// idleConn is a pooled-style Conn over net.Pipe whose last use was at lastUsed.
func idleConn(t *testing.T, lastUsed time.Time) *Conn {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})
	return &Conn{conn: client, lastUsed: lastUsed}
}

// isClosed reports whether a pipe-backed Conn has been closed, without
// blocking: an open pipe with a past write deadline fails with a deadline
// error instead of io.ErrClosedPipe.
func isClosed(c *Conn) bool {
	_ = c.conn.SetWriteDeadline(time.Now().Add(-time.Second))
	_, err := c.conn.Write([]byte{0})
	return errors.Is(err, io.ErrClosedPipe)
}

// quietPool builds a pool whose background sweeper never fires during a test,
// so evictIdle can be driven directly.
func quietPool(minPerPod int) *ConnPool {
	return NewConnPoolWithConfig(PoolConfig{
		MinPerPod:         minPerPod,
		MaxPerPod:         4,
		IdleTimeout:       time.Minute,
		IdleCheckInterval: time.Hour,
	})
}

func TestConnPool_EvictIdle_ClosesStaleConnsAboveFloor(t *testing.T) {
	p := quietPool(1)
	defer p.Close()
	mc := &metricCollector{}
	p.SetMetrics(mc.timing, mc.count, []string{"tenant:t", "store:s"})

	stale := time.Now().Add(-2 * time.Minute)
	floor := idleConn(t, stale)      // stale, but the pod is still under MinPerPod → kept
	extra := idleConn(t, stale)      // stale and above the floor → evicted
	fresh := idleConn(t, time.Now()) // not idle long enough → kept
	p.mu.Lock()
	p.pools["pod:1"] = []*Conn{floor, extra, fresh}
	p.mu.Unlock()

	p.evictIdle()

	p.mu.Lock()
	kept := p.pools["pod:1"]
	p.mu.Unlock()
	assert.Equal(t, []*Conn{floor, fresh}, kept)
	assert.True(t, isClosed(extra))
	assert.False(t, isClosed(floor))
	assert.False(t, isClosed(fresh))

	evicted := mc.findAll(MetricPoolIdleEvicted)
	require.Len(t, evicted, 1)
	assert.Equal(t, int64(1), evicted[0].Value)
	assert.Equal(t, []string{"tenant:t", "store:s"}, evicted[0].Tags)
}

func TestConnPool_EvictIdle_DropsPodWithNoIdleConns(t *testing.T) {
	p := quietPool(1)
	defer p.Close()
	c := idleConn(t, time.Now())
	p.Put("pod:1", c)
	got, err := p.Get("pod:1") // checks the only conn out, leaving an empty slot
	require.NoError(t, err)
	require.Same(t, c, got)

	p.evictIdle()

	p.mu.Lock()
	_, exists := p.pools["pod:1"]
	p.mu.Unlock()
	assert.False(t, exists, "an empty per-pod slot should be deleted")
}

func TestConnPool_EvictIdle_ClosedPoolIsNoOp(t *testing.T) {
	p := quietPool(0)
	p.Close()
	stale := idleConn(t, time.Now().Add(-time.Hour))
	p.mu.Lock()
	p.pools["pod:1"] = []*Conn{stale}
	p.mu.Unlock()

	p.evictIdle()

	p.mu.Lock()
	kept := p.pools["pod:1"]
	p.mu.Unlock()
	assert.Equal(t, []*Conn{stale}, kept)
	assert.False(t, isClosed(stale))
}

// The result slice is sized by the count in the server's response header, not
// by the number of keys requested.
func TestBatchLookups_ZeroCountResponse_ReturnsNoResults(t *testing.T) {
	tests := []struct {
		name   string
		reqLen int
		lookup func(c *Conn) ([][]byte, error)
	}{
		{"BatchLookup", 1 + 2 + keySize, func(c *Conn) ([][]byte, error) {
			return c.BatchLookup(context.Background(), [][]byte{key12("k")})
		}},
		{"StringBatchLookup", 1 + 2 + 2 + 1, func(c *Conn) ([][]byte, error) {
			return c.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			go func() {
				defer server.Close()
				buf := make([]byte, tc.reqLen)
				if _, err := io.ReadFull(server, buf); err != nil {
					return
				}
				server.Write([]byte{0, 0}) // N=0
			}()
			vals, err := tc.lookup(&Conn{conn: client, lastUsed: time.Now()})
			require.NoError(t, err)
			assert.Equal(t, [][]byte{}, vals)
		})
	}
}

func TestNewConnPoolWithConfig_MinPerPodPreservesIdleConns(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn
		}
	}()

	p := NewConnPoolWithConfig(PoolConfig{
		MaxPerPod:         4,
		MinPerPod:         1,
		IdleTimeout:       50 * time.Millisecond,
		IdleCheckInterval: 20 * time.Millisecond,
		DialTimeout:       time.Second,
	})
	defer p.Close()

	addr := ln.Addr().String()
	c, err := p.Get(addr)
	require.NoError(t, err)
	p.Put(addr, c)

	// Wait for idle eviction sweep — MinPerPod=1 should keep 1 conn alive.
	time.Sleep(200 * time.Millisecond)
	p.mu.Lock()
	remaining := len(p.pools[addr])
	p.mu.Unlock()
	assert.Equal(t, 1, remaining, "MinPerPod=1 should preserve one connection")
}

func TestDialWithKeepalive_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	c, err := DialWithKeepalive(ln.Addr().String(), time.Second, 15*time.Second, 5*time.Second)
	require.NoError(t, err)
	assert.NoError(t, c.Close())
}
