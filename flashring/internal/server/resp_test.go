package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeCache is the in-memory test double for the server.Cache interface.
// It records every Put/Get so tests can assert call counts and arguments.
type fakeCache struct {
	mu      sync.Mutex
	store   map[string]fakeEntry
	puts    int
	gets    int
	hits    int
	expired int
}

type fakeEntry struct {
	value     []byte
	expiresAt time.Time // zero = never expires
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: map[string]fakeEntry{}}
}

type fakeStats struct {
	puts, gets, hits, expired int
}

func (c *fakeCache) stats() fakeStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fakeStats{puts: c.puts, gets: c.gets, hits: c.hits, expired: c.expired}
}

func (c *fakeCache) Put(key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.puts++
	cp := make([]byte, len(value))
	copy(cp, value)
	e := fakeEntry{value: cp}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	c.store[key] = e
	return nil
}

func (c *fakeCache) Get(key string) ([]byte, bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets++
	e, ok := c.store[key]
	if !ok {
		return nil, false, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.expired++
		return nil, false, true
	}
	c.hits++
	return e.value, true, false
}

// startServer binds 127.0.0.1:0, runs server.Serve in a goroutine, and
// returns the accept-side address plus a cleanup that closes the listener.
func startServer(t *testing.T, cache Cache) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = Serve(ln, cache) }()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

// dial opens a TCP connection to the given address with a short read deadline
// applied to every Read so a hung server doesn't hang the test forever.
func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

// roundTrip writes payload and returns whatever the server emits before either
// `expectedBytes` are read, or `total` arrives, or 1s elapses.
func roundTrip(t *testing.T, conn net.Conn, payload string, expectedBytes int) string {
	t.Helper()
	if _, err := conn.Write([]byte(payload)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	buf := make([]byte, expectedBytes)
	n, err := io.ReadFull(conn, buf)
	if err != nil && !(err == io.ErrUnexpectedEOF && n > 0) {
		t.Fatalf("read: %v (got %d bytes: %q)", err, n, buf[:n])
	}
	return string(buf[:n])
}

func TestRESP_SetGet_RoundTrip(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()

	conn := dial(t, addr)
	defer conn.Close()

	got := roundTrip(t, conn, "*3\r\n$3\r\nSET\r\n$5\r\nuser1\r\n$5\r\nhello\r\n", len("+OK\r\n"))
	if got != "+OK\r\n" {
		t.Fatalf("SET: want +OK\\r\\n, got %q", got)
	}

	got = roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$5\r\nuser1\r\n", len("$5\r\nhello\r\n"))
	if got != "$5\r\nhello\r\n" {
		t.Fatalf("GET: want $5\\r\\nhello\\r\\n, got %q", got)
	}

	s := cache.stats()
	if s.puts != 1 || s.gets != 1 || s.hits != 1 {
		t.Errorf("counters: puts=%d gets=%d hits=%d, want 1/1/1", s.puts, s.gets, s.hits)
	}
}

func TestRESP_SetWithEX_TTL(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()

	conn := dial(t, addr)
	defer conn.Close()

	// SET tmp v1 EX 5 — value should be live within window, and the cache
	// should record a non-zero TTL passed through.
	roundTrip(t, conn, "*5\r\n$3\r\nSET\r\n$3\r\ntmp\r\n$2\r\nv1\r\n$2\r\nEX\r\n$1\r\n5\r\n", len("+OK\r\n"))

	got := roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$3\r\ntmp\r\n", len("$2\r\nv1\r\n"))
	if got != "$2\r\nv1\r\n" {
		t.Fatalf("GET within TTL: want $2\\r\\nv1\\r\\n, got %q", got)
	}

	// Confirm the entry has a non-zero expiry.
	cache.mu.Lock()
	e, ok := cache.store["tmp"]
	cache.mu.Unlock()
	if !ok {
		t.Fatal("tmp not in cache")
	}
	if e.expiresAt.IsZero() {
		t.Error("expiresAt should be non-zero when EX is set")
	}
}

func TestRESP_Get_UnknownKey_ReturnsNil(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()

	conn := dial(t, addr)
	defer conn.Close()

	got := roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$5\r\nghost\r\n", len("$-1\r\n"))
	if got != "$-1\r\n" {
		t.Fatalf("GET ghost: want $-1\\r\\n, got %q", got)
	}
	s := cache.stats()
	if s.hits != 0 || s.gets != 1 {
		t.Errorf("counters: gets=%d hits=%d, want 1/0", s.gets, s.hits)
	}
}

func TestRESP_Set_Overwrite_ReturnsNewValue(t *testing.T) {
	cache := newFakeCache()
	_ = cache.Put("k1", []byte("first"), 0)
	cache.puts = 0 // pretest state — counters reset for clarity

	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	roundTrip(t, conn, "*3\r\n$3\r\nSET\r\n$2\r\nk1\r\n$6\r\nsecond\r\n", len("+OK\r\n"))
	got := roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$2\r\nk1\r\n", len("$6\r\nsecond\r\n"))
	if got != "$6\r\nsecond\r\n" {
		t.Fatalf("after overwrite, GET k1: want $6\\r\\nsecond\\r\\n, got %q", got)
	}
}

func TestRESP_Set_PXOption_Rejected(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	got := roundTrip(t, conn, "*5\r\n$3\r\nSET\r\n$2\r\nk1\r\n$1\r\nv\r\n$2\r\nPX\r\n$4\r\n1000\r\n", len("-ERR only EX option is supported\r\n"))
	if got != "-ERR only EX option is supported\r\n" {
		t.Fatalf("PX: want -ERR only EX..., got %q", got)
	}

	// Confirm not stored.
	got = roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$2\r\nk1\r\n", len("$-1\r\n"))
	if got != "$-1\r\n" {
		t.Fatalf("after PX error, GET k1: want $-1, got %q", got)
	}
}

func TestRESP_Set_NonNumericEX_Rejected(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	got := roundTrip(t, conn, "*5\r\n$3\r\nSET\r\n$2\r\nk1\r\n$1\r\nv\r\n$2\r\nEX\r\n$3\r\nabc\r\n", len("-ERR invalid expire seconds\r\n"))
	if got != "-ERR invalid expire seconds\r\n" {
		t.Fatalf("EX abc: want -ERR invalid expire seconds, got %q", got)
	}
}

func TestRESP_Get_WrongArity(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	// GET with no key.
	got := roundTrip(t, conn, "*1\r\n$3\r\nGET\r\n", len("-ERR wrong number of arguments for 'get'\r\n"))
	if got != "-ERR wrong number of arguments for 'get'\r\n" {
		t.Fatalf("zero-arg GET: want -ERR wrong number..., got %q", got)
	}

	// Connection should still be alive.
	got = roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$1\r\nk\r\n", len("$-1\r\n"))
	if got != "$-1\r\n" {
		t.Fatalf("conn dead after error: got %q", got)
	}
}

func TestRESP_UnknownCommand(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	got := roundTrip(t, conn, "*2\r\n$3\r\nDEL\r\n$2\r\nk1\r\n", len("-ERR unknown command\r\n"))
	if got != "-ERR unknown command\r\n" {
		t.Fatalf("DEL: want -ERR unknown command, got %q", got)
	}
}

func TestRESP_Pipelined_SetGet(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	want := "+OK\r\n$1\r\nv\r\n"
	got := roundTrip(t, conn, "*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n*2\r\n$3\r\nGET\r\n$1\r\nk\r\n", len(want))
	if got != want {
		t.Fatalf("pipelined SET+GET: want %q, got %q", want, got)
	}
}

func TestRESP_LowercaseAndMixedCase(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	roundTrip(t, conn, "*3\r\n$3\r\nset\r\n$7\r\ncasekey\r\n$7\r\ncaseval\r\n", len("+OK\r\n"))
	got := roundTrip(t, conn, "*2\r\n$3\r\nGet\r\n$7\r\ncasekey\r\n", len("$7\r\ncaseval\r\n"))
	if got != "$7\r\ncaseval\r\n" {
		t.Fatalf("mixed case Get: got %q", got)
	}
}

func TestRESP_EmptyValue_Distinguishable(t *testing.T) {
	// SET k "" → GET k must return $0\r\n\r\n (present, empty), NOT $-1\r\n.
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	roundTrip(t, conn, "*3\r\n$3\r\nSET\r\n$6\r\nemptyk\r\n$0\r\n\r\n", len("+OK\r\n"))
	got := roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$6\r\nemptyk\r\n", len("$0\r\n\r\n"))
	if got != "$0\r\n\r\n" {
		t.Fatalf("empty value GET: want $0\\r\\n\\r\\n, got %q", got)
	}

	// Contrast: a never-set key must still return nil.
	got = roundTrip(t, conn, "*2\r\n$3\r\nGET\r\n$5\r\nnoset\r\n", len("$-1\r\n"))
	if got != "$-1\r\n" {
		t.Fatalf("nil vs empty: noset returned %q", got)
	}
}

func TestRESP_BinaryValue_RoundTrip(t *testing.T) {
	// Length-prefixed bulk-string framing must preserve embedded \r\n + 0x00.
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	binary := []byte{0x00, 0xFF, 'h', 'i', '\r', '\n', 't', 'h', 'e', 'r', 'e', 0x80}
	setCmd := append([]byte("*3\r\n$3\r\nSET\r\n$6\r\nbinkey\r\n$12\r\n"), binary...)
	setCmd = append(setCmd, '\r', '\n')

	if _, err := conn.Write(setCmd); err != nil {
		t.Fatalf("write SET: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("deadline: %v", err)
	}
	okBuf := make([]byte, len("+OK\r\n"))
	if _, err := io.ReadFull(conn, okBuf); err != nil || string(okBuf) != "+OK\r\n" {
		t.Fatalf("SET ack: got %q, err=%v", okBuf, err)
	}

	if _, err := conn.Write([]byte("*2\r\n$3\r\nGET\r\n$6\r\nbinkey\r\n")); err != nil {
		t.Fatalf("write GET: %v", err)
	}
	expected := append([]byte("$12\r\n"), binary...)
	expected = append(expected, '\r', '\n')
	got := make([]byte, len(expected))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read GET: %v", err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatalf("binary round-trip:\n got: %v\nwant: %v", got, expected)
	}
}

func TestRESP_LargeValue_AcrossBufferBoundary(t *testing.T) {
	// 256 KiB value forces multiple flushes through the 64 KiB bufio writer.
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()
	conn := dial(t, addr)
	defer conn.Close()

	const N = 256 * 1024
	val := make([]byte, N)
	for i := range val {
		val[i] = byte(i % 256)
	}

	// SET big <256KiB>
	header := []byte("*3\r\n$3\r\nSET\r\n$3\r\nbig\r\n$262144\r\n")
	if _, err := conn.Write(header); err != nil {
		t.Fatalf("write SET header: %v", err)
	}
	if _, err := conn.Write(val); err != nil {
		t.Fatalf("write SET value: %v", err)
	}
	if _, err := conn.Write([]byte("\r\n")); err != nil {
		t.Fatalf("write SET trailer: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("deadline: %v", err)
	}
	okBuf := make([]byte, len("+OK\r\n"))
	if _, err := io.ReadFull(conn, okBuf); err != nil || string(okBuf) != "+OK\r\n" {
		t.Fatalf("SET large ack: got %q, err=%v", okBuf, err)
	}

	// GET big
	if _, err := conn.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nbig\r\n")); err != nil {
		t.Fatalf("write GET: %v", err)
	}
	respHeader := make([]byte, len("$262144\r\n"))
	if _, err := io.ReadFull(conn, respHeader); err != nil {
		t.Fatalf("read GET header: %v", err)
	}
	if string(respHeader) != "$262144\r\n" {
		t.Fatalf("header: want $262144\\r\\n, got %q", respHeader)
	}
	gotVal := make([]byte, N)
	if _, err := io.ReadFull(conn, gotVal); err != nil {
		t.Fatalf("read GET payload: %v", err)
	}
	trailer := make([]byte, 2)
	if _, err := io.ReadFull(conn, trailer); err != nil {
		t.Fatalf("read GET trailer: %v", err)
	}
	if !bytes.Equal(gotVal, val) {
		t.Errorf("256KiB payload mismatch (lengths %d vs %d)", len(gotVal), len(val))
	}
	if string(trailer) != "\r\n" {
		t.Errorf("trailer: want \\r\\n, got %q", trailer)
	}
}

func TestRESP_NonRESP_ConnectionDropped_ListenerSurvives(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()

	// Bad client sends raw HTTP — server must drop the conn but stay listening.
	bad := dial(t, addr)
	if _, err := bad.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\n\r\n")); err != nil {
		t.Fatalf("bad write: %v", err)
	}
	// Server should close from its side.
	if err := bad.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("deadline: %v", err)
	}
	one := make([]byte, 1)
	n, err := bad.Read(one)
	switch {
	case err == nil:
		t.Fatalf("bad-conn read returned %d byte(s) with nil err — server did not close the connection", n)
	case err == io.EOF, strings.Contains(err.Error(), "closed"):
		// expected: server dropped the malformed connection
	default:
		var ne net.Error
		if (errors.As(err, &ne) && ne.Timeout()) || strings.Contains(err.Error(), "timeout") {
			t.Fatalf("bad-conn read timed out (%v) — server did not close the connection within the deadline", err)
		}
		t.Fatalf("bad-conn read: want io.EOF or closed-conn error, got %v", err)
	}
	bad.Close()

	// Healthy client on a fresh connection still works.
	good := dial(t, addr)
	defer good.Close()
	want := "+OK\r\n$1\r\nv\r\n"
	got := roundTrip(t, good, "*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n*2\r\n$3\r\nGET\r\n$1\r\nk\r\n", len(want))
	if got != want {
		t.Fatalf("after bad client, healthy SET+GET: got %q", got)
	}
}

func TestRESP_ConcurrentConnections_NoCrosstalk(t *testing.T) {
	cache := newFakeCache()
	addr, stop := startServer(t, cache)
	defer stop()

	const numClients = 16
	const opsPer = 25
	var wg sync.WaitGroup
	errCh := make(chan error, numClients)
	wg.Add(numClients)
	for i := 0; i < numClients; i++ {
		i := i
		go func() {
			defer wg.Done()
			conn := dial(t, addr)
			defer conn.Close()
			for n := 0; n < opsPer; n++ {
				k := []byte(itoa(i) + "_" + itoa(n))
				v := []byte("v_" + itoa(i) + "_" + itoa(n))
				setCmd := buildSet(k, v)
				if _, err := conn.Write(setCmd); err != nil {
					errCh <- err
					return
				}
				if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
					errCh <- err
					return
				}
				okBuf := make([]byte, 5)
				if _, err := io.ReadFull(conn, okBuf); err != nil || string(okBuf) != "+OK\r\n" {
					errCh <- err
					return
				}
				if _, err := conn.Write(buildGet(k)); err != nil {
					errCh <- err
					return
				}
				header := readBulkHeader(conn)
				if header != "$"+itoa(len(v))+"\r\n" {
					errCh <- fmt.Errorf("header mismatch: %q", header)
					return
				}
				body := make([]byte, len(v)+2) // value + CRLF
				if _, err := io.ReadFull(conn, body); err != nil {
					errCh <- err
					return
				}
				if !bytes.Equal(body[:len(v)], v) {
					errCh <- fmt.Errorf("value mismatch: %q", body[:len(v)])
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("client error: %v", err)
		}
	}
	if s := cache.stats(); s.puts != numClients*opsPer {
		t.Errorf("total puts: want %d, got %d", numClients*opsPer, s.puts)
	}
}

// --- small RESP encoding helpers used by concurrent test ---

func buildSet(key, value []byte) []byte {
	var b bytes.Buffer
	b.WriteString("*3\r\n$3\r\nSET\r\n$")
	b.WriteString(itoa(len(key)))
	b.WriteString("\r\n")
	b.Write(key)
	b.WriteString("\r\n$")
	b.WriteString(itoa(len(value)))
	b.WriteString("\r\n")
	b.Write(value)
	b.WriteString("\r\n")
	return b.Bytes()
}

func buildGet(key []byte) []byte {
	var b bytes.Buffer
	b.WriteString("*2\r\n$3\r\nGET\r\n$")
	b.WriteString(itoa(len(key)))
	b.WriteString("\r\n")
	b.Write(key)
	b.WriteString("\r\n")
	return b.Bytes()
}

// readBulkHeader reads `$N\r\n` from conn. Used by the concurrent test only.
func readBulkHeader(conn net.Conn) string {
	var sb strings.Builder
	one := make([]byte, 1)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := io.ReadFull(conn, one); err != nil {
			return sb.String()
		}
		sb.WriteByte(one[0])
		if one[0] == '\n' {
			return sb.String()
		}
	}
}

// itoa avoids strconv.Itoa to keep the dependency surface narrow; it's only
// used for non-negative ints in the helpers above.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
