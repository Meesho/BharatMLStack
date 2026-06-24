package sdk

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── fake read server ──────────────────────────────────────────────────────────

// fakeServer answers the mNemo binary protocol from an in-memory key→value map.
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
	return &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
	_, err := c.BatchLookup(context.Background(), [][]byte{key12("k")})
	assert.Error(t, err)
}

// ── StringSingleLookup ──────────────────────────────────────────────────────

func TestStringSingleLookup_WriteError(t *testing.T) {
	client, server := net.Pipe()
	server.Close()
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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
	c := &Conn{conn: client}
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

func TestConnPool_PutUnknownAddrClosesConn(t *testing.T) {
	client, _ := net.Pipe()
	p := NewConnPool(2)
	defer p.Close()
	// Put to an address the pool has no channel for → connection is closed.
	p.Put("never-Get-this", &Conn{conn: client})
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
	assert.Equal(t, 4, p.maxPerPod)
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
