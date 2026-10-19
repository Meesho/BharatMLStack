package sdk

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
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
		server.Write([]byte{0, 1})         // N=1, then close before found byte
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
		io.ReadFull(server, buf[:1+2+2+1])        // op + N + keyLen + key
		server.Write([]byte{0, 1, 1, 0, 0, 0, 5}) // N=1, found=1, len=5, no value
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

// ── recycling a connection whose request failed ───────────────────────────────

// encodeValue is one value in the wire format: [1B found][4B len][value], or
// [0] for a miss.
func encodeValue(val []byte) []byte {
	if val == nil {
		return []byte{0}
	}
	buf := make([]byte, 1+4+len(val))
	buf[0] = 1
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(val)))
	copy(buf[5:], val)
	return buf
}

// encodeBatch is a batch response: [2B N] then N values.
func encodeBatch(vals ...[]byte) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(vals)))
	for _, v := range vals {
		buf = append(buf, encodeValue(v)...)
	}
	return buf
}

// readRequest consumes one request of any opcode from a server-side conn.
func readRequest(conn net.Conn) error {
	var op [1]byte
	if _, err := io.ReadFull(conn, op[:]); err != nil {
		return err
	}
	readStringKey := func() error {
		var klen [2]byte
		if _, err := io.ReadFull(conn, klen[:]); err != nil {
			return err
		}
		_, err := io.ReadFull(conn, make([]byte, binary.BigEndian.Uint16(klen[:])))
		return err
	}
	switch op[0] {
	case opSingle:
		_, err := io.ReadFull(conn, make([]byte, keySize))
		return err
	case opStringSingle:
		return readStringKey()
	case opBatch, opStringBatch:
		var nbuf [2]byte
		if _, err := io.ReadFull(conn, nbuf[:]); err != nil {
			return err
		}
		for i := 0; i < int(binary.BigEndian.Uint16(nbuf[:])); i++ {
			if op[0] == opBatch {
				if _, err := io.ReadFull(conn, make([]byte, keySize)); err != nil {
					return err
				}
			} else if err := readStringKey(); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("unknown opcode %#x", op[0])
}

// slowResponseConn returns a Conn whose server reads one request, writes
// resp[:cut], waits for release, writes the rest, and then serves every later
// request from data like fakeServer.
func slowResponseConn(t *testing.T, resp []byte, cut int, release <-chan struct{}, data map[string][]byte) *Conn {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})
	go func() {
		if readRequest(server) != nil {
			return
		}
		if _, err := server.Write(resp[:cut]); err != nil {
			return
		}
		<-release
		if _, err := server.Write(resp[cut:]); err != nil {
			return
		}
		fakeServer(server, data)
	}()
	return &Conn{conn: client, lastUsed: time.Now()}
}

// drainResults reports the result tag of every MetricPoolDrain the pool emits.
func drainResults(p *ConnPool) <-chan string {
	ch := make(chan string, 16)
	p.SetMetrics(nil, func(name string, _ int64, tags []string) {
		if name != MetricPoolDrain {
			return
		}
		for _, tag := range tags {
			if res, ok := strings.CutPrefix(tag, "result:"); ok {
				ch <- res
			}
		}
	}, nil)
	return ch
}

func awaitDrain(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case res := <-ch:
		return res
	case <-time.After(3 * time.Second):
		t.Fatal("no drain result")
		return ""
	}
}

func drainPool(drainTimeout time.Duration) *ConnPool {
	return NewConnPoolWithConfig(PoolConfig{
		MaxPerPod:         4,
		IdleCheckInterval: time.Hour,
		DrainTimeout:      drainTimeout,
	})
}

func shortCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	t.Cleanup(cancel)
	return ctx
}

// A request that hits its deadline with the response still on the wire leaves
// the connection reusable: Recycle drains the late response wherever the read
// stopped (before it, inside a length or a value, between entries) and pools
// the connection, and the next request on it reads its own response.
func TestConnPool_Recycle_TimedOutResponseIsDrainedAndTheConnReused(t *testing.T) {
	batch := encodeBatch([]byte("v-one"), nil, []byte("v-three"))
	withEmpty := encodeBatch([]byte{}, []byte("v"))
	single := encodeValue([]byte("value"))
	lookups := map[string]func(*Conn, context.Context) error{
		"string_batch": func(c *Conn, ctx context.Context) error {
			_, err := c.StringBatchLookup(ctx, [][]byte{[]byte("a"), []byte("b"), []byte("c")})
			return err
		},
		"batch": func(c *Conn, ctx context.Context) error {
			_, err := c.BatchLookup(ctx, [][]byte{key12("a"), key12("b"), key12("c")})
			return err
		},
		"string_single": func(c *Conn, ctx context.Context) error {
			_, err := c.StringSingleLookup(ctx, []byte("a"))
			return err
		},
		"single": func(c *Conn, ctx context.Context) error {
			_, err := c.SingleLookup(ctx, key12("a"))
			return err
		},
	}
	cases := []struct {
		name, op string
		resp     []byte
		cut      int
	}{
		{"batch: nothing yet", "string_batch", batch, 0},
		{"batch: inside the count", "string_batch", batch, 1},
		{"batch: after the count", "string_batch", batch, 2},
		{"batch: after a found byte", "string_batch", batch, 3},
		{"batch: inside a length", "string_batch", batch, 5},
		{"batch: inside a value", "string_batch", batch, 9},
		{"batch: after a miss", "string_batch", batch, 13},
		{"batch: one byte short", "string_batch", batch, len(batch) - 1},
		{"batch: before an empty value", "string_batch", withEmpty, 3},
		{"fixed-key batch: inside a value", "batch", batch, 9},
		{"single: nothing yet", "string_single", single, 0},
		{"single: inside the length", "string_single", single, 2},
		{"single: inside the value", "string_single", single, 7},
		{"fixed-key single: inside the value", "single", single, 7},
	}
	data := map[string][]byte{"next": []byte("next-value")}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := drainPool(2 * time.Second)
			defer p.Close()
			drained := drainResults(p)
			release := make(chan struct{})
			conn := slowResponseConn(t, tc.resp, tc.cut, release, data)

			err := lookups[tc.op](conn, shortCtx(t))
			require.ErrorIs(t, err, os.ErrDeadlineExceeded)
			p.Recycle("pod", conn)
			close(release)

			require.Equal(t, "reused", awaitDrain(t, drained))
			got, err := p.GetContext(context.Background(), "pod")
			require.NoError(t, err)
			assert.Same(t, conn, got)
			vals, err := got.StringBatchLookup(context.Background(), [][]byte{[]byte("next")})
			require.NoError(t, err)
			assert.Equal(t, [][]byte{[]byte("next-value")}, vals)
		})
	}
}

// A late response that does not finish within DrainTimeout closes the
// connection rather than pooling it.
func TestConnPool_Recycle_DrainPastTheTimeoutClosesTheConn(t *testing.T) {
	p := drainPool(30 * time.Millisecond)
	defer p.Close()
	drained := drainResults(p)
	never := make(chan struct{})
	conn := slowResponseConn(t, encodeBatch([]byte("v")), 3, never, nil)

	_, err := conn.StringBatchLookup(shortCtx(t), [][]byte{[]byte("a")})
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	p.Recycle("pod", conn)

	assert.Equal(t, "closed", awaitDrain(t, drained))
	assert.True(t, isClosed(conn))
	assert.Empty(t, p.pools["pod"])
}

// Errors that leave the stream in an unknown state close the connection at
// once, without a drain: the peer closed it, the request was only partly
// written, draining is disabled, or the pool is closed.
func TestConnPool_Recycle_UnrecoverableFailuresCloseImmediately(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) (*ConnPool, *Conn, error)
	}{
		{"peer closed mid-response", func(t *testing.T) (*ConnPool, *Conn, error) {
			client, server := net.Pipe()
			t.Cleanup(func() { client.Close() })
			go func() {
				_ = readRequest(server)
				_, _ = server.Write(encodeBatch([]byte("value"))[:4])
				server.Close()
			}()
			conn := &Conn{conn: client}
			_, err := conn.StringBatchLookup(context.Background(), [][]byte{[]byte("a")})
			return drainPool(time.Second), conn, err
		}},
		{"request only partly written", func(t *testing.T) (*ConnPool, *Conn, error) {
			client, server := net.Pipe()
			t.Cleanup(func() {
				client.Close()
				server.Close()
			})
			go func() { _, _ = io.ReadFull(server, make([]byte, 1)) }()
			conn := &Conn{conn: client}
			_, err := conn.StringBatchLookup(shortCtx(t), [][]byte{[]byte("a-key")})
			return drainPool(time.Second), conn, err
		}},
		{"draining disabled", func(t *testing.T) (*ConnPool, *Conn, error) {
			conn := slowResponseConn(t, encodeBatch([]byte("v")), 0, make(chan struct{}), nil)
			_, err := conn.StringBatchLookup(shortCtx(t), [][]byte{[]byte("a")})
			return drainPool(-1), conn, err
		}},
		{"pool closed", func(t *testing.T) (*ConnPool, *Conn, error) {
			conn := slowResponseConn(t, encodeBatch([]byte("v")), 0, make(chan struct{}), nil)
			_, err := conn.StringBatchLookup(shortCtx(t), [][]byte{[]byte("a")})
			p := drainPool(time.Second)
			p.Close()
			return p, conn, err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, conn, err := tc.setup(t)
			defer p.Close()
			drained := drainResults(p)
			require.Error(t, err)

			p.Recycle("pod", conn)

			assert.True(t, isClosed(conn))
			assert.Empty(t, p.pools["pod"])
			assert.Empty(t, drained, "no drain was attempted")
		})
	}
}

// A request whose deadline had already passed sends nothing, so its
// connection is clean and goes straight back to the pool.
func TestConnPool_Recycle_NothingSentPoolsTheConnAtOnce(t *testing.T) {
	p := drainPool(time.Second)
	defer p.Close()
	conn := pipeConn(map[string][]byte{"k": []byte("v")})
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := conn.StringBatchLookup(expired, [][]byte{[]byte("k")})
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	p.Recycle("pod", conn)

	require.Len(t, p.pools["pod"], 1)
	assert.Same(t, conn, p.pools["pod"][0])
	vals, err := conn.StringBatchLookup(context.Background(), [][]byte{[]byte("k")})
	require.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("v")}, vals)
}

func TestPoolConfig_DrainAndDialDefaults(t *testing.T) {
	pc := PoolConfig{}
	pc.applyDefaults()
	assert.Equal(t, time.Second, pc.DrainTimeout)
	assert.Equal(t, 4, pc.MaxDialsPerPod)

	disabled := PoolConfig{DrainTimeout: -1}
	disabled.applyDefaults()
	assert.Equal(t, time.Duration(-1), disabled.DrainTimeout, "negative keeps draining off")
}
