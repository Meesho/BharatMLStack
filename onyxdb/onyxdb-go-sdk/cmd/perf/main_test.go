package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
	sdk "github.com/Meesho/BharatMLStack/onyxdb/onyxdb-go-sdk"
)

// ── fake read server (string-key opcodes 0x03 / 0x04) ───────────────────────

type fakeReadServer struct {
	addr      string
	data      map[string][]byte
	singles   atomic.Int64 // 0x03 requests served
	batches   atomic.Int64 // 0x04 requests served
	lastBatch atomic.Int64 // key count of the most recent 0x04 request
}

func startFakeReadServer(t *testing.T, data map[string][]byte) *fakeReadServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &fakeReadServer{addr: ln.Addr().String(), data: data}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.serve(conn)
			}()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		wg.Wait()
	})
	return s
}

func (s *fakeReadServer) reset() {
	s.singles.Store(0)
	s.batches.Store(0)
	s.lastBatch.Store(0)
}

func readStringKey(r io.Reader) ([]byte, error) {
	var l [2]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return nil, err
	}
	k := make([]byte, binary.BigEndian.Uint16(l[:]))
	_, err := io.ReadFull(r, k)
	return k, err
}

func appendValue(buf []byte, val []byte) []byte {
	if val == nil {
		return append(buf, 0)
	}
	buf = append(buf, 1)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(val)))
	return append(buf, val...)
}

func (s *fakeReadServer) serve(conn net.Conn) {
	defer conn.Close()
	for {
		var op [1]byte
		if _, err := io.ReadFull(conn, op[:]); err != nil {
			return
		}
		switch op[0] {
		case 0x03:
			k, err := readStringKey(conn)
			if err != nil {
				return
			}
			s.singles.Add(1)
			if _, err := conn.Write(appendValue(nil, s.data[string(k)])); err != nil {
				return
			}
		case 0x04:
			var nb [2]byte
			if _, err := io.ReadFull(conn, nb[:]); err != nil {
				return
			}
			n := int(binary.BigEndian.Uint16(nb[:]))
			resp := binary.BigEndian.AppendUint16(nil, uint16(n))
			for i := 0; i < n; i++ {
				k, err := readStringKey(conn)
				if err != nil {
					return
				}
				resp = appendValue(resp, s.data[string(k)])
			}
			s.batches.Add(1)
			s.lastBatch.Store(int64(n))
			if _, err := conn.Write(resp); err != nil {
				return
			}
		default:
			return
		}
	}
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

func directClient(t *testing.T, addr string) *sdk.Client {
	t.Helper()
	c := sdk.NewDirectClient(addr, 4)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// captureStdout runs fn with os.Stdout redirected and returns what it printed.
func captureStdout(t *testing.T, fn func()) (out string) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	var buf bytes.Buffer
	copied := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(copied)
	}()
	defer func() {
		os.Stdout = orig
		w.Close()
		<-copied
		r.Close()
		out = buf.String()
	}()
	fn()
	return ""
}

// ── pct ─────────────────────────────────────────────────────────────────────

func TestPct(t *testing.T) {
	tenMs := make([]time.Duration, 10)
	for i := range tenMs {
		tenMs[i] = time.Duration(i+1) * time.Millisecond
	}
	tests := []struct {
		name   string
		sorted []time.Duration
		p      float64
		want   time.Duration
	}{
		{"empty sample", nil, 50, 0},
		{"p0 is the minimum", tenMs, 0, time.Millisecond},
		{"p50 indexes floor(p*n)", tenMs, 50, 6 * time.Millisecond},
		{"p99.9 below the last index", tenMs, 99.9, 10 * time.Millisecond},
		{"p100 clamps to the maximum", tenMs, 100, 10 * time.Millisecond},
		{"rounded to the microsecond", []time.Duration{1500 * time.Nanosecond}, 50, 2 * time.Microsecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pct(tc.sorted, tc.p))
		})
	}
}

// ── loadOrGenKeys ───────────────────────────────────────────────────────────

func keyStrings(keys [][]byte) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	return out
}

func TestLoadOrGenKeys_Synthetic(t *testing.T) {
	assert.Equal(t, []string{"ent:0|0", "ent:1|1", "ent:2|2"}, keyStrings(loadOrGenKeys("", 3, "ent")))

	wrapped := loadOrGenKeys("", 4098, "ent")
	require.Len(t, wrapped, 4098)
	assert.Equal(t, "ent:4096|0", string(wrapped[4096]), "second pk component wraps at 4096")
	assert.Equal(t, "ent:4097|1", string(wrapped[4097]))

	assert.Empty(t, loadOrGenKeys("", 0, "ent"))
}

func TestLoadOrGenKeys_FromFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"trims whitespace and skips blank lines", "  a:1|2  \n\n b \n\t\nc", []string{"a:1|2", "b", "c"}},
		{"empty file yields no keys", "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "keys.txt")
			require.NoError(t, os.WriteFile(path, []byte(tc.content), 0o600))
			got := loadOrGenKeys(path, 99, "ignored")
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tc.want, keyStrings(got))
		})
	}
}

// ── doSingle / doBatch ──────────────────────────────────────────────────────

func TestDoSingle(t *testing.T) {
	srv := startFakeReadServer(t, map[string][]byte{"e:1": []byte("v")})
	tests := []struct {
		name     string
		addr     string
		key      string
		wantErrc int
	}{
		{"hit", srv.addr, "e:1", 0},
		{"miss is a valid response", srv.addr, "e:2", 0},
		{"transport error counts", refusedAddr(t), "e:1", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := runCfg{client: directClient(t, tc.addr), keys: [][]byte{[]byte(tc.key)}}
			lat, errc := doSingle(cfg, rand.New(rand.NewSource(1)))
			assert.Equal(t, tc.wantErrc, errc)
			assert.Greater(t, lat, time.Duration(0))
		})
	}
}

func TestDoBatch(t *testing.T) {
	srv := startFakeReadServer(t, map[string][]byte{"e:1": []byte("v")})
	tests := []struct {
		name     string
		addr     string
		wantErrc int
	}{
		{"ok", srv.addr, 0},
		{"transport error counts", refusedAddr(t), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv.reset()
			cfg := runCfg{client: directClient(t, tc.addr), keys: [][]byte{[]byte("e:1"), []byte("e:2")}, batch: 3}
			lat, nkeys, errc := doBatch(cfg, rand.New(rand.NewSource(1)))
			assert.Equal(t, 3, nkeys)
			assert.Equal(t, tc.wantErrc, errc)
			assert.Greater(t, lat, time.Duration(0))
			if tc.wantErrc == 0 {
				assert.Equal(t, int64(1), srv.batches.Load())
				assert.Equal(t, int64(3), srv.lastBatch.Load(), "one multi-get carrying all batch keys")
			}
		})
	}
}

// ── drive / run ─────────────────────────────────────────────────────────────

func TestDrive_SingleRecorded(t *testing.T) {
	srv := startFakeReadServer(t, map[string][]byte{"e:1": []byte("v")})
	cfg := runCfg{client: directClient(t, srv.addr), keys: [][]byte{[]byte("e:1")}, concurrency: 2}

	r := drive(cfg, false, 50*time.Millisecond, true)

	require.Positive(t, r.ops)
	assert.Equal(t, r.ops, r.keys, "one key per single get")
	assert.Equal(t, int64(0), r.errs)
	assert.Len(t, r.lats, int(r.ops))
	assert.Equal(t, r.ops, srv.singles.Load())
	assert.Equal(t, int64(0), srv.batches.Load())
	assert.GreaterOrEqual(t, r.elapsed, 50*time.Millisecond)
}

func TestDrive_BatchRecorded(t *testing.T) {
	srv := startFakeReadServer(t, map[string][]byte{"e:1": []byte("v")})
	cfg := runCfg{client: directClient(t, srv.addr), keys: [][]byte{[]byte("e:1")}, concurrency: 2, batch: 4}

	r := drive(cfg, true, 50*time.Millisecond, true)

	require.Positive(t, r.ops)
	assert.Equal(t, 4*r.ops, r.keys)
	assert.Equal(t, int64(0), r.errs)
	assert.Len(t, r.lats, int(r.ops))
	assert.Equal(t, r.ops, srv.batches.Load())
	assert.Equal(t, int64(0), srv.singles.Load())
}

func TestDrive_UnrecordedKeepsCountsButNoLatencies(t *testing.T) {
	srv := startFakeReadServer(t, map[string][]byte{})
	cfg := runCfg{client: directClient(t, srv.addr), keys: [][]byte{[]byte("e:1")}, concurrency: 1}

	r := drive(cfg, false, 30*time.Millisecond, false)

	require.Positive(t, r.ops)
	assert.Empty(t, r.lats)
}

func TestDrive_ErrorsCounted(t *testing.T) {
	cfg := runCfg{client: directClient(t, refusedAddr(t)), keys: [][]byte{[]byte("e:1")}, concurrency: 1}

	r := drive(cfg, false, 30*time.Millisecond, true)

	require.Positive(t, r.ops)
	assert.Equal(t, r.ops, r.errs)
}

func TestDrive_NoWork(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		dur         time.Duration
	}{
		{"zero duration: deadline already passed", 2, 0},
		{"zero workers", 0, 30 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := startFakeReadServer(t, map[string][]byte{})
			cfg := runCfg{client: directClient(t, srv.addr), keys: [][]byte{[]byte("e:1")}, concurrency: tc.concurrency}

			r := drive(cfg, false, tc.dur, true)

			assert.Equal(t, int64(0), r.ops)
			assert.Equal(t, int64(0), r.keys)
			assert.Empty(t, r.lats)
			assert.Equal(t, int64(0), srv.singles.Load())
		})
	}
}

func TestRun_WarmupTrafficIsNotReported(t *testing.T) {
	tests := []struct {
		name   string
		warmup time.Duration
	}{
		{"no warmup: every request is measured", 0},
		{"warmup: its requests are excluded", 200 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := startFakeReadServer(t, map[string][]byte{"e:1": []byte("v")})
			cfg := runCfg{
				client: directClient(t, srv.addr), keys: [][]byte{[]byte("e:1")},
				concurrency: 1, warmup: tc.warmup, duration: 50 * time.Millisecond,
			}

			r := run(cfg, false)

			require.Positive(t, r.ops)
			assert.Len(t, r.lats, int(r.ops))
			if tc.warmup == 0 {
				assert.Equal(t, r.ops, srv.singles.Load())
			} else {
				assert.Greater(t, srv.singles.Load(), r.ops)
			}
		})
	}
}

// ── report ──────────────────────────────────────────────────────────────────

func TestReport(t *testing.T) {
	const body = "== SINGLE GET ==\n" +
		"  elapsed      1s\n" +
		"  throughput   3 ops/sec   6 keys/sec\n" +
		"  latency      p50=2ms  p75=3ms  p95=3ms  p99=3ms  p99.9=3ms  max=3ms\n"
	tests := []struct {
		name string
		errs int64
		want string
	}{
		{"no errors", 0, body + "\n"},
		{"errors are called out", 2, body + "  errors       2 failed ops (check connectivity / promotion)\n\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lats := []time.Duration{3 * time.Millisecond, time.Millisecond, 2 * time.Millisecond}
			r := result{lats: lats, ops: 3, keys: 6, errs: tc.errs, elapsed: time.Second}

			out := captureStdout(t, func() { report("SINGLE GET", r) })

			assert.Equal(t, tc.want, out)
			assert.Equal(t, []time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}, lats,
				"latencies are sorted in place")
		})
	}
}

// ── main (in-process, against embedded etcd + fake read server) ─────────────

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

// startEmbeddedEtcd runs a single-node etcd on loopback and returns its client
// endpoint as host:port.
func startEmbeddedEtcd(t *testing.T) string {
	t.Helper()
	cu, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", freePort(t)))
	require.NoError(t, err)
	pu, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", freePort(t)))
	require.NoError(t, err)

	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	cfg.LogLevel = "error"
	cfg.ListenClientUrls = []url.URL{*cu}
	cfg.AdvertiseClientUrls = []url.URL{*cu}
	cfg.ListenPeerUrls = []url.URL{*pu}
	cfg.AdvertisePeerUrls = []url.URL{*pu}
	cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, pu.String())

	e, err := embed.StartEtcd(cfg)
	require.NoError(t, err)
	t.Cleanup(e.Close)
	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		t.Fatal("embedded etcd not ready")
	}
	return cu.Host
}

// seedTopology promotes a one-shard version whose only pod is podAddr.
func seedTopology(t *testing.T, endpoint, tenant, store, podAddr string) {
	t.Helper()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{endpoint}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta, err := json.Marshal(model.VersionMeta{
		ShardCount: 1, Status: model.StatusActive,
		Assignment: map[string][]string{"0": {podAddr}},
	})
	require.NoError(t, err)
	cc, err := json.Marshal(model.ClientConfig{RequestTimeoutMs: 2000})
	require.NoError(t, err)
	for k, v := range map[string]string{
		model.ClientConfigPath(tenant, store):    string(cc),
		model.VersionPrefix(tenant, store, "v1"): string(meta),
		model.ActiveVersionPath(tenant, store):   "v1",
	} {
		_, err := cli.Put(ctx, k, v)
		require.NoError(t, err)
	}
}

// runPerfMain invokes main() with the given flags and returns its stdout.
func runPerfMain(t *testing.T, args ...string) string {
	t.Helper()
	origArgs, origFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = origArgs, origFlags })
	os.Args = append([]string{"perf"}, args...)
	flag.CommandLine = flag.NewFlagSet("perf", flag.ContinueOnError)
	return captureStdout(t, main)
}

func TestPerfMain_ModeSelectsWorkload(t *testing.T) {
	srv := startFakeReadServer(t, map[string][]byte{"ent:0|0": []byte("v")})
	endpoint := startEmbeddedEtcd(t)
	seedTopology(t, endpoint, "recsys", "catalog", srv.addr)

	tests := []struct {
		mode        string
		wantSingles bool
		wantBatches bool
	}{
		{"single", true, false},
		{"batch", false, true},
		{"both", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			srv.reset()

			out := runPerfMain(t,
				"-etcd", endpoint, "-tenant", "recsys", "-store", "catalog",
				"-gen", "50", "-entity", "ent", "-mode", tc.mode, "-batch", "5",
				"-concurrency", "2", "-duration", "100ms", "-warmup", "300ms")

			assert.Contains(t, out, fmt.Sprintf(
				"OnyxDB perf — etcd=%s recsys/catalog keys=50 concurrency=2 batch=5 duration=100ms\n", endpoint))
			assert.Equal(t, tc.wantSingles, strings.Contains(out, "== SINGLE GET ==\n"))
			assert.Equal(t, tc.wantBatches, strings.Contains(out, "== BATCH GET (multi-get, batch=5) ==\n"))
			assert.Equal(t, tc.wantSingles, srv.singles.Load() > 0, "single gets sent")
			assert.Equal(t, tc.wantBatches, srv.batches.Load() > 0, "multi-gets sent")
		})
	}
}

// ── fatal exits (os.Exit) via a re-executed child process ───────────────────

const helperArgsEnv = "PERF_TEST_MAIN_ARGS"

// TestMainExitHelper is not a real test. runMainExpectExit re-executes the test
// binary with only this test selected; it then runs main() with the arguments
// from the environment so os.Exit paths can be observed. Under `go test -cover`
// the child writes its coverage counters to the parent's GOCOVERDIR on exit, so
// those paths are counted in the parent's profile.
func TestMainExitHelper(t *testing.T) {
	raw, ok := os.LookupEnv(helperArgsEnv)
	if !ok {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(raw), &args))
	os.Args = append([]string{"perf"}, args...)
	flag.CommandLine = flag.NewFlagSet("perf", flag.ContinueOnError)
	main()
}

// runMainExpectExit runs main() with args in a child process and returns its
// exit code and stderr. GOCOVERDIR collects counters from an os.Exit death,
// -test.gocoverdir from a normal exit.
func runMainExpectExit(t *testing.T, args ...string) (int, string) {
	t.Helper()
	enc, err := json.Marshal(args)
	require.NoError(t, err)
	childArgs := []string{"-test.run=^TestMainExitHelper$", "-test.timeout=60s"}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), helperArgsEnv+"="+string(enc))
	if f := flag.Lookup("test.gocoverdir"); f != nil && f.Value.String() != "" {
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+f.Value.String())
		childArgs = append(childArgs, "-test.gocoverdir="+f.Value.String())
	}
	cmd.Args = append(cmd.Args, childArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stderr.String()
	}
	require.NoError(t, err, "run child")
	return 0, stderr.String()
}

func TestPerfMain_FatalExits(t *testing.T) {
	endpoint := startEmbeddedEtcd(t)
	emptyKeys := filepath.Join(t.TempDir(), "empty.txt")
	require.NoError(t, os.WriteFile(emptyKeys, nil, 0o600))
	missingKeys := filepath.Join(t.TempDir(), "missing.txt")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"etcd endpoint gRPC cannot parse", []string{"-etcd", "%zz"}, "FATAL: NewClient: "},
		{"keys file empty", []string{"-etcd", endpoint, "-keys", emptyKeys}, "FATAL: no keys\n"},
		{"keys file unreadable", []string{"-etcd", endpoint, "-keys", missingKeys}, "FATAL: open keys " + missingKeys + ": "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stderr := runMainExpectExit(t, tc.args...)
			assert.Equal(t, 1, code, "stderr:\n%s", stderr)
			assert.Contains(t, stderr, tc.want)
		})
	}
}
