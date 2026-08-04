//go:build integration

package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// ── embedded etcd helper ──────────────────────────────────────────────────────

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func startEmbeddedEtcd(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "etcd-sdk-test-*")
	require.NoError(t, err)

	cu, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", freePort(t)))
	pu, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", freePort(t)))

	cfg := embed.NewConfig()
	cfg.Dir = dir
	cfg.LogLevel = "error"
	cfg.ListenClientUrls = []url.URL{*cu}
	cfg.AdvertiseClientUrls = []url.URL{*cu}
	cfg.ListenPeerUrls = []url.URL{*pu}
	cfg.AdvertisePeerUrls = []url.URL{*pu}
	cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, pu.String())

	e, err := embed.StartEtcd(cfg)
	require.NoError(t, err)
	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		e.Server.Stop()
		t.Fatal("embedded etcd not ready")
	}
	return cu.String(), func() {
		e.Server.Stop()
		_ = os.RemoveAll(dir)
	}
}

// ── Config defaults ───────────────────────────────────────────────────────────

func TestConfig_Defaults(t *testing.T) {
	c := Config{}
	c.applyDefaults()
	assert.Equal(t, 4, c.ConnsPerPod)
	assert.Equal(t, 100, c.TimeoutMs)
}

// ── NewClient ─────────────────────────────────────────────────────────────────

func TestNewClient_NoEndpoints(t *testing.T) {
	_, err := NewClient(Config{Tenant: "t", Store: "s"})
	assert.ErrorIs(t, err, ErrNoEndpoints)
}

func TestNewClient_EtcdDialError(t *testing.T) {
	orig := newEtcdClient
	defer func() { newEtcdClient = orig }()
	newEtcdClient = func(_ []string) (*clientv3.Client, error) {
		return nil, errors.New("dial boom")
	}
	_, err := NewClient(Config{EtcdEndpoints: []string{"x"}, Tenant: "t", Store: "s"})
	assert.ErrorContains(t, err, "dial boom")
}

func TestNewClient_EmbeddedEtcd_ResolvesViaDNSAndGets(t *testing.T) {
	etcdURL, teardown := startEmbeddedEtcd(t)
	defer teardown()

	// Fake read server; split its addr so DNS resolves host and Config carries port.
	k := key12("hello")
	addr := startFakeServer(t, map[string][]byte{string(k): []byte("world")})
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)

	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{etcdURL}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer cli.Close()
	ctx := context.Background()
	meta, _ := json.Marshal(model.VersionMeta{ShardCount: 1, Status: model.StatusActive})
	_, err = cli.Put(ctx, model.VersionPrefix("recsys", "catalog", "v1"), string(meta))
	require.NoError(t, err)
	_, err = cli.Put(ctx, model.ActiveVersionPath("recsys", "catalog"), "v1")
	require.NoError(t, err)

	// Stub DNS so the shard's headless Service resolves to the fake server's host.
	withLookup(okLookup(host), func() {
		client, err := NewClient(Config{
			EtcdEndpoints:      []string{etcdURL},
			Tenant:             "recsys",
			Store:              "catalog",
			Namespace:          "onyxdb",
			Port:               port,
			DNSRefreshInterval: time.Hour, // initial resolve via watcher; no ticker noise
		})
		require.NoError(t, err)
		defer client.Close()

		require.Eventually(t, func() bool {
			return client.router.ShardCount() == 1
		}, 3*time.Second, 10*time.Millisecond)

		val, err := client.Get(ctx, k)
		require.NoError(t, err)
		assert.Equal(t, []byte("world"), val)
	})
}

// ── NewDirectClient + Get ─────────────────────────────────────────────────────

func TestNewDirectClient_GetHitAndMiss(t *testing.T) {
	k := key12("k")
	addr := startFakeServer(t, map[string][]byte{string(k): []byte("v")})
	c := NewDirectClient(addr, 0) // 0 → default conns
	defer c.Close()

	val, err := c.Get(context.Background(), k)
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), val)

	_, err = c.Get(context.Background(), key12("missing"))
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestGet_NoHealthyPod(t *testing.T) {
	c := staticClient(1, nil) // resolver returns no pods for shard 0
	defer c.Close()
	_, err := c.Get(context.Background(), key12("k"))
	assert.ErrorIs(t, err, ErrNoHealthyPod)
}

func TestGet_PoolDialError(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	deadAddr := ln.Addr().String()
	ln.Close()

	c := NewDirectClient(deadAddr, 4)
	defer c.Close()
	_, err := c.Get(context.Background(), key12("k"))
	assert.Error(t, err)
}

func TestGet_BrokenConnMarksUnhealthy(t *testing.T) {
	// Server closes immediately → SingleLookup errors (not ErrKeyNotFound).
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

	c := NewDirectClient(ln.Addr().String(), 4)
	defer c.Close()
	_, err := c.Get(context.Background(), key12("k"))
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrKeyNotFound)
}

// ── Close ─────────────────────────────────────────────────────────────────────

func TestClose_DirectClient(t *testing.T) {
	c := NewDirectClient("127.0.0.1:1", 4)
	assert.NoError(t, c.Close())
}
