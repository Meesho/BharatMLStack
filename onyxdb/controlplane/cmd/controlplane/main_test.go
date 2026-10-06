package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

func init() { gin.SetMode(gin.TestMode) }

// Env vars read by run(); every test that drives run() sets all of them so the
// host environment can never leak in.
const (
	envAddr     = "ONYXDB_CP_ADDR"
	envEtcd     = "ONYXDB_ETCD_ENDPOINTS"
	envInterval = "ONYXDB_AUTO_PROMOTE_INTERVAL"
)

// ── envOrDefault ──────────────────────────────────────────────────────────────

func TestEnvOrDefault(t *testing.T) {
	const k = "ONYXDB_CP_TEST_ENV_OR_DEFAULT"
	tests := []struct {
		name  string
		value *string // nil = unset
		want  string
	}{
		{"unset returns default", nil, "fallback"},
		{"empty returns default", strPtr(""), "fallback"},
		{"set returns value", strPtr("explicit"), "explicit"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnsetEnv(t, k, tc.value)
			assert.Equal(t, tc.want, envOrDefault(k, "fallback"))
		})
	}
}

// ── autoPromoteInterval ───────────────────────────────────────────────────────

func TestAutoPromoteInterval(t *testing.T) {
	tests := []struct {
		name  string
		value *string // nil = unset
		want  time.Duration
	}{
		{"unset defaults to 5s", nil, 5 * time.Second},
		{"seconds override", strPtr("10s"), 10 * time.Second},
		{"minutes override", strPtr("1m"), time.Minute},
		{"zero disables", strPtr("0"), 0},
		{"unparseable falls back to 5s", strPtr("soon"), 5 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnsetEnv(t, envInterval, tc.value)
			assert.Equal(t, tc.want, autoPromoteInterval())
		})
	}
}

// ── run ───────────────────────────────────────────────────────────────────────

func TestRun_ReturnsErrorWhenEtcdClientCannotBeCreated(t *testing.T) {
	// "%zz" makes the gRPC dial target unparseable, so clientv3.New fails fast.
	t.Setenv(envAddr, freeAddr(t))
	t.Setenv(envEtcd, "127.0.0.1:%zz")
	t.Setenv(envInterval, "0")

	err := run()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connecting to etcd at [127.0.0.1:%zz]")
}

func TestRun_ReturnsErrorWhenListenAddressInUse(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	t.Setenv(envAddr, l.Addr().String())
	t.Setenv(envEtcd, startEmbeddedEtcd(t))
	t.Setenv(envInterval, "0")

	err = run()

	assert.ErrorIs(t, err, syscall.EADDRINUSE)
}

func TestRun_ServesUntilSIGTERMWithReconcilerDisabled(t *testing.T) {
	etcdURL := startEmbeddedEtcd(t)
	seedPromotableStore(t, etcdURL)
	addr := freeAddr(t)
	t.Setenv(envAddr, addr)
	t.Setenv(envEtcd, etcdURL)
	t.Setenv(envInterval, "0")

	errCh := startRun(t)
	// Ready (200) proves the server is bound to ONYXDB_CP_ADDR and talks to
	// the etcd named by ONYXDB_ETCD_ENDPOINTS.
	waitReady(t, addr, errCh)

	require.NoError(t, stopRun(t, errCh))
	// Reconciler disabled → the promotable version was never promoted.
	assert.Equal(t, "", activeVersionInEtcd(t, etcdURL))
}

func TestRun_AutoPromotesWhenReconcilerEnabled(t *testing.T) {
	etcdURL := startEmbeddedEtcd(t)
	seedPromotableStore(t, etcdURL)
	addr := freeAddr(t)
	t.Setenv(envAddr, addr)
	t.Setenv(envEtcd, etcdURL)
	t.Setenv(envInterval, "20ms")

	errCh := startRun(t)
	waitReady(t, addr, errCh)

	require.Eventually(t, func() bool {
		return activeVersionViaAPI(t, addr) == seededVersion
	}, 10*time.Second, 20*time.Millisecond, "reconciler should auto-promote the seeded version")

	require.NoError(t, stopRun(t, errCh))
}

// ── main (subprocess) ─────────────────────────────────────────────────────────

const (
	mainHelperEnv      = "ONYXDB_CP_TEST_RUN_MAIN"
	mainReturnedMarker = "ONYXDB_CP_TEST_MAIN_RETURNED"
)

// TestMainHelperProcess is not a real test. The TestMain_* tests re-exec the
// test binary with mainHelperEnv=1 so main() runs in a child process: its
// log.Fatal branch calls os.Exit(1), which would kill this test binary.
func TestMainHelperProcess(t *testing.T) {
	if os.Getenv(mainHelperEnv) != "1" {
		t.Skip("helper process for the TestMain_* subprocess tests")
	}
	main()
	fmt.Println(mainReturnedMarker)
}

func TestMain_ExitsWithStatus1WhenRunFails(t *testing.T) {
	cmd := mainCommand(t,
		envAddr+"="+freeAddr(t),
		envEtcd+"=127.0.0.1:%zz",
		envInterval+"=0",
	)

	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "output:\n%s", out)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, string(out), "connecting to etcd")
	assert.NotContains(t, string(out), mainReturnedMarker)
}

func TestMain_ExitsCleanlyOnSIGTERM(t *testing.T) {
	addr := freeAddr(t)
	cmd := mainCommand(t,
		envAddr+"="+addr,
		envEtcd+"="+startEmbeddedEtcd(t),
		envInterval+"=0",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Start())
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	t.Cleanup(func() { _ = cmd.Process.Kill() }) // no-op once the child has exited

	waitReady(t, addr, waitCh)
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))

	select {
	case err := <-waitCh:
		require.NoError(t, err, "output:\n%s", out.String())
	case <-time.After(15 * time.Second):
		t.Fatal("main did not exit after SIGTERM")
	}
	assert.Contains(t, out.String(), mainReturnedMarker)
}

// ── helpers ───────────────────────────────────────────────────────────────────

const (
	seededTenant  = "t"
	seededStore   = "s"
	seededVersion = "20260101_001"
)

func strPtr(s string) *string { return &s }

// setOrUnsetEnv sets key to *value, or unsets it when value is nil. Either way
// the original value is restored when the test ends.
func setOrUnsetEnv(t *testing.T, key string, value *string) {
	t.Helper()
	t.Setenv(key, "")
	if value == nil {
		require.NoError(t, os.Unsetenv(key))
		return
	}
	t.Setenv(key, *value)
}

// freeAddr returns a 127.0.0.1 address whose port was just free.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// startEmbeddedEtcd starts an in-process etcd on free loopback ports and
// returns its client URL. It is stopped when the test ends.
func startEmbeddedEtcd(t *testing.T) string {
	t.Helper()
	cu, _ := url.Parse("http://" + freeAddr(t))
	pu, _ := url.Parse("http://" + freeAddr(t))

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
		t.Fatal("embedded etcd did not start in time")
	}
	return cu.String()
}

// seedPromotableStore creates a 1-shard auto-promote store with one READY
// version that its only pod already has warm, i.e. the reconciler's next tick
// promotes seededVersion.
func seedPromotableStore(t *testing.T, etcdURL string) {
	t.Helper()
	ctx := context.Background()
	sc, err := etcdstate.NewEtcdStateClient([]string{etcdURL})
	require.NoError(t, err)
	defer sc.Close()

	require.NoError(t, sc.CreateStore(ctx, model.StoreConfig{
		Tenant: seededTenant, Store: seededStore, EntityKey: "id", ShardCount: 1,
	}))
	require.NoError(t, sc.PutDataflow(ctx, seededTenant, seededStore, model.DataflowConfig{AutoPromote: true}))
	require.NoError(t, sc.PublishVersion(ctx, seededTenant, seededStore, seededVersion,
		model.VersionMeta{ShardCount: 1, Status: model.StatusReady}))

	pod, err := json.Marshal(model.PodData{PodIP: "10.0.0.1", WarmVersions: []string{seededVersion}})
	require.NoError(t, err)
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{etcdURL}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer cli.Close()
	_, err = cli.Put(ctx, model.PodDataPath(seededTenant, seededStore, "t-s-shard-0-0"), string(pod))
	require.NoError(t, err)
}

func activeVersionInEtcd(t *testing.T, etcdURL string) string {
	t.Helper()
	sc, err := etcdstate.NewEtcdStateClient([]string{etcdURL})
	require.NoError(t, err)
	defer sc.Close()
	st, err := sc.GetStore(context.Background(), seededTenant, seededStore)
	require.NoError(t, err)
	return st.ActiveVersion
}

var httpClient = &http.Client{Timeout: 2 * time.Second}

func activeVersionViaAPI(t *testing.T, addr string) string {
	t.Helper()
	resp, err := httpClient.Get(fmt.Sprintf("http://%s/api/v1/tenants/%s/stores/%s", addr, seededTenant, seededStore))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var st etcdstate.StoreState
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&st))
	return st.ActiveVersion
}

// startRun runs run() in a goroutine and returns the channel its result is
// delivered on.
func startRun(t *testing.T) <-chan error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- run() }()
	return errCh
}

// waitReady polls GET /api/v1/ready on addr until it answers 200. It fails
// fast if the process/goroutine reports an exit on exited first.
func waitReady(t *testing.T, addr string, exited <-chan error) {
	t.Helper()
	readyURL := "http://" + addr + "/api/v1/ready"
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			t.Fatalf("control plane exited before becoming ready: %v", err)
		default:
		}
		if resp, err := httpClient.Get(readyURL); err == nil {
			code := resp.StatusCode
			resp.Body.Close()
			if code == http.StatusOK {
				return
			}
		}
		select {
		case err := <-exited:
			t.Fatalf("control plane exited before becoming ready: %v", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatalf("control plane at %s never became ready", addr)
}

// stopRun delivers SIGTERM to this test process — run()'s signal.NotifyContext
// is registered before the server starts listening, so once waitReady has
// returned the signal is captured (it cancels run's context instead of killing
// the test binary) — and returns run()'s result.
func stopRun(t *testing.T, errCh <-chan error) error {
	t.Helper()
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
	select {
	case err := <-errCh:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("run did not return after SIGTERM")
		return errors.New("unreachable")
	}
}

// mainCommand builds a command that re-execs this test binary so that only
// TestMainHelperProcess runs, and it calls main() with the given env.
// Under `go test -cover` the child writes its coverage counters into the
// parent's coverage directory, so main()'s lines count toward the profile.
func mainCommand(t *testing.T, env ...string) *exec.Cmd {
	t.Helper()
	args := []string{"-test.run=^TestMainHelperProcess$"}
	childEnv := append(os.Environ(), mainHelperEnv+"=1")
	if dir := parentCoverDir(); dir != "" {
		args = append(args, "-test.gocoverdir="+dir)
		childEnv = append(childEnv, "GOCOVERDIR="+dir)
	}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(childEnv, env...)
	return cmd
}

// parentCoverDir is the directory `go test -cover` collects coverage data
// from, or "" when coverage is off.
func parentCoverDir() string {
	if testing.CoverMode() == "" {
		return ""
	}
	if f := flag.Lookup("test.gocoverdir"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}
	return os.Getenv("GOCOVERDIR")
}
