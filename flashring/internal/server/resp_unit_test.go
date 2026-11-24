package server

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// These tests exercise the unit-level parse/encode helpers and the
// connection-level error paths that the TCP integration tests in resp_test.go
// can't reach (malformed framing fed straight into the bufio.Reader, listener
// errors, etc.).

func TestServeRESP_PortInUse_ReturnsListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	// Same address must be busy → ServeRESP fails on net.Listen.
	cache := newFakeCache()
	if err := ServeRESP(addr, cache); err == nil {
		t.Fatal("ServeRESP returned nil err for busy address")
	}
}

func TestServeRESP_BindsAndServes(t *testing.T) {
	cache := newFakeCache()
	errCh := make(chan error, 1)

	// Bind on :0 by launching a goroutine that races a Close — verifies the
	// happy path through net.Listen + Serve before the listener is torn down.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	// Run ServeRESP on the now-free port. Wait briefly, then dial; close from
	// the dial side ends the test.
	go func() { errCh <- ServeRESP(addr, cache) }()

	deadline := time.Now().Add(2 * time.Second)
	var conn net.Conn
	for time.Now().Before(deadline) {
		// If ServeRESP exits early (bind race, listen failure, etc.),
		// surface the error instead of looping silently to the deadline.
		select {
		case serveErr := <-errCh:
			t.Fatalf("ServeRESP exited before becoming reachable: %v", serveErr)
		default:
		}
		conn, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if conn == nil {
		t.Fatalf("never reachable: %v", err)
	}
	_, _ = conn.Write([]byte("*2\r\n$3\r\nGET\r\n$1\r\nk\r\n"))
	buf := make([]byte, 8)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := conn.Read(buf)
	if n == 0 || buf[0] != '$' {
		t.Fatalf("unexpected response: %q", buf[:n])
	}
	conn.Close()
	// ServeRESP loops forever on Accept; the only way it exits early is on
	// failure, so an error after the round-trip is also worth surfacing.
	select {
	case serveErr := <-errCh:
		t.Fatalf("ServeRESP exited unexpectedly: %v", serveErr)
	default:
	}
}

func TestServe_AcceptPermanentError_Returns(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cache := newFakeCache()
	errCh := make(chan error, 1)
	go func() { errCh <- Serve(ln, cache) }()

	// Closing the listener produces a non-temporary error → Serve returns.
	ln.Close()

	select {
	case e := <-errCh:
		if e == nil {
			t.Fatal("Serve returned nil error after listener close")
		}
		if !strings.Contains(e.Error(), "closed") && !errors.Is(e, net.ErrClosed) {
			t.Fatalf("Serve err = %v, want closed-listener error", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s after listener close")
	}
}

func TestServe_AcceptTemporaryError_BackoffContinues(t *testing.T) {
	// Use a fake listener whose Accept returns a temporary error twice then
	// a permanent error — exercises the temporary-error sleep+continue branch
	// before falling through to the return.
	fl := &flakyListener{tempLeft: 2}
	cache := newFakeCache()

	start := time.Now()
	err := Serve(fl, cache)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Serve returned nil after permanent failure")
	}
	if elapsed < 100*time.Millisecond {
		t.Fatalf("temporary-error backoff too short: %v (want >= 100ms)", elapsed)
	}
}

type flakyListener struct {
	tempLeft int
}

func (f *flakyListener) Accept() (net.Conn, error) {
	if f.tempLeft > 0 {
		f.tempLeft--
		return nil, tempErr{}
	}
	return nil, errors.New("permanent")
}
func (f *flakyListener) Close() error   { return nil }
func (f *flakyListener) Addr() net.Addr { return dummyAddr{} }

type tempErr struct{}

func (tempErr) Error() string   { return "temporary" }
func (tempErr) Timeout() bool   { return false }
func (tempErr) Temporary() bool { return true }

type dummyAddr struct{}

func (dummyAddr) Network() string { return "tcp" }
func (dummyAddr) String() string  { return "127.0.0.1:0" }

// fakeConn is a fully in-process net.Conn driven by two bytes.Buffer
// halves — input fed to the server, output captured by the test.
type fakeConn struct {
	in     *bytes.Buffer
	out    *bytes.Buffer
	closed bool
}

func newFakeConn(input string) *fakeConn {
	return &fakeConn{in: bytes.NewBufferString(input), out: &bytes.Buffer{}}
}

func (c *fakeConn) Read(b []byte) (int, error) {
	if c.in.Len() == 0 {
		return 0, io.EOF
	}
	return c.in.Read(b)
}
func (c *fakeConn) Write(b []byte) (int, error) { return c.out.Write(b) }
func (c *fakeConn) Close() error                 { c.closed = true; return nil }
func (c *fakeConn) LocalAddr() net.Addr          { return dummyAddr{} }
func (c *fakeConn) RemoteAddr() net.Addr         { return dummyAddr{} }
func (c *fakeConn) SetDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func TestHandleConn_EmptyArray_LoopContinues(t *testing.T) {
	cache := newFakeCache()
	// `*0\r\n` is a zero-element array; readRESPArray returns nil/nil/nil and
	// handleConn continues. Then EOF closes the loop.
	conn := newFakeConn("*0\r\n")
	handleConn(conn, cache)

	if !conn.closed {
		t.Fatal("handleConn did not close conn")
	}
	if cache.puts != 0 || cache.gets != 0 {
		t.Fatalf("counters touched: puts=%d gets=%d", cache.puts, cache.gets)
	}
}

func TestHandleConn_NonArrayPrefix_ReturnsImmediately(t *testing.T) {
	cache := newFakeCache()
	// '+' is the simple-string marker — not '*'. readRESPArray returns
	// io.ErrUnexpectedEOF → handleConn returns silently.
	conn := newFakeConn("+PING\r\n")
	handleConn(conn, cache)

	if !conn.closed {
		t.Fatal("handleConn did not close conn on parse error")
	}
}

func TestHandleConn_ImmediateEOF_Returns(t *testing.T) {
	cache := newFakeCache()
	conn := newFakeConn("")
	handleConn(conn, cache)

	if !conn.closed {
		t.Fatal("handleConn did not close on empty input")
	}
}

func TestReadRESPArray_NegativeArrayCount(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("*-1\r\n"))
	cmd, args, err := readRESPArray(r)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if cmd != nil || args != nil {
		t.Fatalf("cmd/args non-nil for negative count: %v %v", cmd, args)
	}
}

func TestReadRESPArray_MissingArrayLength(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("*abc\r\n"))
	if _, _, err := readRESPArray(r); err == nil {
		t.Fatal("err = nil for non-numeric array length")
	}
}

func TestReadRESPArray_EmptyReader(t *testing.T) {
	r := bufio.NewReader(strings.NewReader(""))
	if _, _, err := readRESPArray(r); err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadRESPArray_FirstBulkReadFails(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("*1\r\n"))
	if _, _, err := readRESPArray(r); err == nil {
		t.Fatal("err = nil when first bulk truncated")
	}
}

func TestReadRESPArray_NthBulkReadFails(t *testing.T) {
	// "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n" — claims 3 elements but only 2 supplied.
	r := bufio.NewReader(strings.NewReader("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n"))
	if _, _, err := readRESPArray(r); err == nil {
		t.Fatal("err = nil when last bulk missing")
	}
}

func TestReadBulkString_BadMarker(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("*3\r\n"))
	if _, err := readBulkString(r); err != io.ErrUnexpectedEOF {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadBulkString_NegativeLength_ReturnsNil(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("$-1\r\n"))
	got, err := readBulkString(r)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil for $-1", got)
	}
}

func TestReadBulkString_TruncatedPayload(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("$5\r\nhi"))
	if _, err := readBulkString(r); err == nil {
		t.Fatal("err = nil for truncated payload")
	}
}

func TestReadBulkString_BadCRLFTerminator(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("$2\r\nhixx"))
	if _, err := readBulkString(r); err == nil {
		t.Fatal("err = nil for bad CRLF terminator")
	}
}

func TestReadBulkString_EOFOnMarker(t *testing.T) {
	r := bufio.NewReader(strings.NewReader(""))
	if _, err := readBulkString(r); err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadBulkString_EOFAfterMarker_MissingLength(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("$"))
	if _, err := readBulkString(r); err == nil {
		t.Fatal("err = nil when length missing")
	}
}

func TestReadIntCRLF_MissingLF(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("3\rx"))
	if _, err := readIntCRLF(r); err == nil {
		t.Fatal("err = nil when LF missing after CR")
	}
}

func TestReadIntCRLF_MissingCR(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("3"))
	if _, err := readIntCRLF(r); err == nil {
		t.Fatal("err = nil when CR absent")
	}
}

func TestReadIntCRLF_EOFAfterCR(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("3\r"))
	if _, err := readIntCRLF(r); err == nil {
		t.Fatal("err = nil at EOF after CR")
	}
}

func TestReadIntCRLF_HappyPath(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("42\r\n"))
	got, err := readIntCRLF(r)
	if err != nil || got != 42 {
		t.Fatalf("readIntCRLF = (%d, %v), want (42, nil)", got, err)
	}
}

func TestExpectCRLF_HappyPath(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\r\n"))
	if err := expectCRLF(r); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestExpectCRLF_BadFirstByte(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("XY"))
	if err := expectCRLF(r); err != io.ErrUnexpectedEOF {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestExpectCRLF_EOFOnFirstByte(t *testing.T) {
	r := bufio.NewReader(strings.NewReader(""))
	if err := expectCRLF(r); err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestExpectCRLF_EOFOnSecondByte(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\r"))
	if err := expectCRLF(r); err == nil {
		t.Fatal("err = nil for EOF on second byte")
	}
}

func TestParseUint_HappyAndError(t *testing.T) {
	if got, err := parseUint([]byte("12345")); err != nil || got != 12345 {
		t.Fatalf("parseUint(12345) = (%d, %v), want (12345, nil)", got, err)
	}
	if _, err := parseUint([]byte("xx")); err == nil {
		t.Fatal("parseUint(xx) err = nil")
	}
}

func TestB2s_EmptyAndNonEmpty(t *testing.T) {
	if b2s(nil) != "" {
		t.Fatalf("b2s(nil) = %q, want empty", b2s(nil))
	}
	if b2s([]byte("hi")) != "hi" {
		t.Fatalf("b2s(hi) = %q, want hi", b2s([]byte("hi")))
	}
}

func TestWriteHelpers_FormatBytes(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*bufio.Writer)
		want string
	}{
		{"simple", func(w *bufio.Writer) { writeSimpleString(w, "OK") }, "+OK\r\n"},
		{"error", func(w *bufio.Writer) { writeError(w, "boom") }, "-ERR boom\r\n"},
		{"bulk", func(w *bufio.Writer) { writeBulk(w, []byte("hi")) }, "$2\r\nhi\r\n"},
		{"nil-bulk", func(w *bufio.Writer) { writeBulkNil(w) }, "$-1\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			tc.fn(w)
			w.Flush()
			if got := buf.String(); got != tc.want {
				t.Errorf("write = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandleConn_SetMissingArgs_WritesError(t *testing.T) {
	cache := newFakeCache()
	conn := newFakeConn("*1\r\n$3\r\nSET\r\n")
	handleConn(conn, cache)

	if !strings.Contains(conn.out.String(), "-ERR wrong number of arguments for 'set'") {
		t.Fatalf("response = %q, missing SET arity error", conn.out.String())
	}
}

// failingWriteConn returns one large RESP payload on Read and fails every
// Write. Drives the `w.Flush() != nil` early-return branches inside
// handleConn (SET with bad EX option, SET with bad seconds, end-of-loop
// flush). The bufio.Writer is large enough to buffer the response, so the
// write failure surfaces only at Flush time.
type failingWriteConn struct {
	in     *bytes.Buffer
	closed bool
}

func (c *failingWriteConn) Read(b []byte) (int, error) {
	if c.in.Len() == 0 {
		return 0, io.EOF
	}
	return c.in.Read(b)
}
func (c *failingWriteConn) Write(b []byte) (int, error)    { return 0, io.ErrClosedPipe }
func (c *failingWriteConn) Close() error                   { c.closed = true; return nil }
func (c *failingWriteConn) LocalAddr() net.Addr            { return dummyAddr{} }
func (c *failingWriteConn) RemoteAddr() net.Addr           { return dummyAddr{} }
func (c *failingWriteConn) SetDeadline(time.Time) error    { return nil }
func (c *failingWriteConn) SetReadDeadline(time.Time) error  { return nil }
func (c *failingWriteConn) SetWriteDeadline(time.Time) error { return nil }

func TestHandleConn_SetPXOption_FlushFails_Returns(t *testing.T) {
	cache := newFakeCache()
	conn := &failingWriteConn{in: bytes.NewBufferString(
		"*5\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n$2\r\nPX\r\n$3\r\n100\r\n",
	)}
	handleConn(conn, cache)

	if !conn.closed {
		t.Fatal("handleConn did not close after flush failure")
	}
	if cache.puts != 0 {
		t.Fatalf("Put called on PX error path: puts=%d", cache.puts)
	}
}

func TestHandleConn_SetBadEXSeconds_FlushFails_Returns(t *testing.T) {
	cache := newFakeCache()
	conn := &failingWriteConn{in: bytes.NewBufferString(
		"*5\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n$2\r\nEX\r\n$2\r\nxx\r\n",
	)}
	handleConn(conn, cache)

	if !conn.closed {
		t.Fatal("handleConn did not close after EX-parse flush failure")
	}
	if cache.puts != 0 {
		t.Fatal("Put called even though EX seconds failed parse")
	}
}

func TestHandleConn_OKResponseFlushFails_Returns(t *testing.T) {
	cache := newFakeCache()
	conn := &failingWriteConn{in: bytes.NewBufferString(
		"*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n",
	)}
	handleConn(conn, cache)

	if !conn.closed {
		t.Fatal("handleConn did not close after OK flush failure")
	}
	// Put still runs before the flush; the loop returns when Flush fails.
	if cache.puts != 1 {
		t.Fatalf("puts = %d, want 1 (set executed before failed flush)", cache.puts)
	}
}

func TestHandleConn_GetExpiredKey_ReturnsNilBulk(t *testing.T) {
	cache := newFakeCache()
	// Pre-load with an already-expired entry.
	cache.store["exp"] = fakeEntry{
		value:     []byte("v"),
		expiresAt: time.Now().Add(-time.Hour),
	}
	conn := newFakeConn("*2\r\n$3\r\nGET\r\n$3\r\nexp\r\n")
	handleConn(conn, cache)

	if !strings.Contains(conn.out.String(), "$-1\r\n") {
		t.Fatalf("response = %q, want $-1 for expired key", conn.out.String())
	}
	if cache.expired != 1 {
		t.Fatalf("expired counter = %d, want 1", cache.expired)
	}
}
