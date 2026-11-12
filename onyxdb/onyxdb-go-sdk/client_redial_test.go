package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// B4 (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3): the
// redial knobs come from the store's etcd clientConfig document, read once at
// NewClient. They are decoded SDK-side (clientConfigExt) rather than through
// model.ClientConfig: a released SDK may build against a control plane module
// whose ClientConfig has none of them. The control plane's ClientConfig now
// carries them too (so its API keeps them), and the SDK keeps them out of its
// model.ClientConfig decode, so a wrong-typed one still costs only the redial
// keys.

func TestFetchClientConfig_DecodesTheRedialKeysFromTheSameDocument(t *testing.T) {
	m := newMockEtcd()
	m.put(model.ClientConfigPath("recsys", "catalog"), `{
		"requestTimeoutMs": 250, "maxConnsPerPod": 8,
		"dialRatePerPod": 0.5, "dialBackoffBaseMs": 50, "dialBackoffMaxMs": 2000,
		"unhealthyBackoffBaseMs": 500, "unhealthyBackoffMaxMs": 10000, "shardFailFast": false}`)

	cc, ext := fetchClientConfig(context.Background(), m, "recsys", "catalog")

	assert.Equal(t, &model.ClientConfig{RequestTimeoutMs: 250, MaxConnsPerPod: 8}, cc)
	assert.Equal(t, &clientConfigExt{
		DialRatePerPod: 0.5, DialBackoffBaseMs: 50, DialBackoffMaxMs: 2000,
		UnhealthyBackoffBaseMs: 500, UnhealthyBackoffMaxMs: 10000, ShardFailFast: boolPtr(false),
	}, ext)
}

// A redial key of the wrong type costs only the redial keys (their defaults
// apply); the rest of the document still applies.
func TestFetchClientConfig_ABadRedialKeyKeepsTheRestOfTheDocument(t *testing.T) {
	m := newMockEtcd()
	m.put(model.ClientConfigPath("recsys", "catalog"), `{"requestTimeoutMs":250,"shardFailFast":"no"}`)

	cc, ext := fetchClientConfig(context.Background(), m, "recsys", "catalog")

	assert.Equal(t, &model.ClientConfig{RequestTimeoutMs: 250}, cc)
	assert.Nil(t, ext)
}

// encoding/json matches keys case-insensitively, so a redial key written in
// another case still reaches its field, and must still be kept out of the
// model.ClientConfig decode.
func TestFetchClientConfig_ABadRedialKeyInAnotherCaseKeepsTheRestOfTheDocument(t *testing.T) {
	m := newMockEtcd()
	m.put(model.ClientConfigPath("recsys", "catalog"),
		`{"requestTimeoutMs":250,"ShardFailFast":"no","DIALRATEPERPOD":"fast"}`)

	cc, ext := fetchClientConfig(context.Background(), m, "recsys", "catalog")

	assert.Equal(t, &model.ClientConfig{RequestTimeoutMs: 250}, cc)
	assert.Nil(t, ext)
}

// The control plane's model.ClientConfig must carry every key clientConfigExt
// reads, under the same JSON name and Go type; otherwise its clientConfig PUT
// drops that key and its GET hides it.
//
// The SDK is normally released on its own and built against the last
// released control plane module; one that predates the redial keys carries
// none of them, and the SDK does not depend on it doing so. The test skips
// there and runs wherever the model has any of them (this monorepo).
func TestClientConfigExt_KeysMatchTheControlPlaneModel(t *testing.T) {
	modelFields := map[string]reflect.Type{}
	mt := reflect.TypeOf(model.ClientConfig{})
	for i := 0; i < mt.NumField(); i++ {
		name, _, _ := strings.Cut(mt.Field(i).Tag.Get("json"), ",")
		modelFields[name] = mt.Field(i).Type
	}

	et := reflect.TypeOf(clientConfigExt{})
	require.Positive(t, et.NumField())
	if !slices.ContainsFunc(redialKeys, func(k string) bool { _, ok := modelFields[k]; return ok }) {
		t.Skip("this control plane module predates the redial keys")
	}
	for i := 0; i < et.NumField(); i++ {
		name, _, _ := strings.Cut(et.Field(i).Tag.Get("json"), ",")
		typ, ok := modelFields[name]
		if assert.Truef(t, ok, "model.ClientConfig has no %q key", name) {
			assert.Equalf(t, et.Field(i).Type, typ, "type of %q", name)
		}
	}
}

func TestFetchClientConfig_NoDocumentNoRedialKeys(t *testing.T) {
	m := newMockEtcd()
	m.setGetErr(model.ClientConfigPath("recsys", "catalog"), errors.New("etcd down"))
	cc, ext := fetchClientConfig(context.Background(), m, "recsys", "catalog")
	assert.Nil(t, cc)
	assert.Nil(t, ext)

	m = newMockEtcd()
	m.put(model.ClientConfigPath("recsys", "catalog"), "{not json")
	cc, ext = fetchClientConfig(context.Background(), m, "recsys", "catalog")
	assert.Nil(t, cc)
	assert.Nil(t, ext)
}

// The etcd key names are a contract with whoever writes the document (and with
// the control plane once model.ClientConfig carries them): pin them so a
// rename cannot slip through.
func TestClientConfigExt_EtcdKeysArePinned(t *testing.T) {
	b, err := json.Marshal(clientConfigExt{
		DialRatePerPod: 1.5, DialBackoffBaseMs: 1, DialBackoffMaxMs: 2,
		UnhealthyBackoffBaseMs: 3, UnhealthyBackoffMaxMs: 4, ShardFailFast: boolPtr(true),
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"dialRatePerPod": 1.5,
		"dialBackoffBaseMs": 1,
		"dialBackoffMaxMs": 2,
		"unhealthyBackoffBaseMs": 3,
		"unhealthyBackoffMaxMs": 4,
		"shardFailFast": true
	}`, string(b))

	b, err = json.Marshal(clientConfigExt{})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(b), "unset keys stay absent, so absent means default")
}

func TestConfigBuildPoolConfig_RedialKnobs(t *testing.T) {
	tests := []struct {
		name                   string
		cfg                    Config
		wantRate               float64
		wantDialBase, wantDial time.Duration
		wantUnhBase, wantUnh   time.Duration
		wantFailFast           *bool
	}{
		{
			name:     "no redial keys: defaults",
			cfg:      Config{},
			wantRate: 2, wantDialBase: 100 * time.Millisecond, wantDial: 5 * time.Second,
		},
		{
			name: "every redial key set",
			cfg: Config{ext: &clientConfigExt{
				DialRatePerPod: 0.5, DialBackoffBaseMs: 50, DialBackoffMaxMs: 2000,
				UnhealthyBackoffBaseMs: 500, UnhealthyBackoffMaxMs: 10000, ShardFailFast: boolPtr(false),
			}},
			wantRate: 0.5, wantDialBase: 50 * time.Millisecond, wantDial: 2 * time.Second,
			wantUnhBase: 500 * time.Millisecond, wantUnh: 10 * time.Second, wantFailFast: boolPtr(false),
		},
		{
			name:     "negative dial keys turn the dial limits off; non-positive unhealthy keys are ignored",
			cfg:      Config{ext: &clientConfigExt{DialRatePerPod: -1, DialBackoffBaseMs: -1, UnhealthyBackoffBaseMs: -5, UnhealthyBackoffMaxMs: -5}},
			wantRate: -1, wantDialBase: -time.Millisecond, wantDial: 5 * time.Second,
		},
		{
			name: "the caller's router settings stand where the document is silent",
			cfg: Config{
				UnhealthyBackoffBase: 2 * time.Second, UnhealthyBackoffMax: time.Minute, ShardFailFast: boolPtr(false),
				ext: &clientConfigExt{},
			},
			wantRate: 2, wantDialBase: 100 * time.Millisecond, wantDial: 5 * time.Second,
			wantUnhBase: 2 * time.Second, wantUnh: time.Minute, wantFailFast: boolPtr(false),
		},
		{
			name: "the document wins over the caller, as every clientConfig key does",
			cfg: Config{
				UnhealthyBackoffBase: 2 * time.Second, ShardFailFast: boolPtr(true),
				ext: &clientConfigExt{UnhealthyBackoffBaseMs: 300, ShardFailFast: boolPtr(false)},
			},
			wantRate: 2, wantDialBase: 100 * time.Millisecond, wantDial: 5 * time.Second,
			wantUnhBase: 300 * time.Millisecond, wantFailFast: boolPtr(false),
		},
		{
			name: "an explicit Pool turns every clientConfig overlay off",
			cfg: Config{
				Pool: &PoolConfig{DialRatePerPod: 5},
				ext:  &clientConfigExt{DialRatePerPod: 0.5, UnhealthyBackoffBaseMs: 300, ShardFailFast: boolPtr(false)},
			},
			wantRate: 5, wantDialBase: 100 * time.Millisecond, wantDial: 5 * time.Second,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg.applyDefaults()
			pc := tc.cfg.buildPoolConfig()

			assert.Equal(t, tc.wantRate, pc.DialRatePerPod)
			assert.Equal(t, tc.wantDialBase, pc.DialBackoffBase)
			assert.Equal(t, tc.wantDial, pc.DialBackoffMax)
			assert.Equal(t, tc.wantUnhBase, tc.cfg.UnhealthyBackoffBase)
			assert.Equal(t, tc.wantUnh, tc.cfg.UnhealthyBackoffMax)
			assert.Equal(t, tc.wantFailFast, tc.cfg.ShardFailFast)
		})
	}
}

func TestNewClientWithEtcd_RedialKnobs(t *testing.T) {
	tests := []struct {
		name         string
		doc          string // "" = absent
		wantRate     float64
		wantDialBase time.Duration
		wantUnhBase  time.Duration
		wantUnhMax   time.Duration
		wantFailFast bool
		wantLease    time.Duration
	}{
		{
			name:     "absent: defaults, fail-fast on",
			wantRate: 2, wantDialBase: 100 * time.Millisecond,
			wantUnhBase: time.Second, wantUnhMax: 30 * time.Second, wantFailFast: true,
			wantLease: 1100 * time.Millisecond,
		},
		{
			name:     "from the store's clientConfig",
			doc:      `{"requestTimeoutMs":250,"dialRatePerPod":0.5,"dialBackoffBaseMs":50,"unhealthyBackoffBaseMs":500,"unhealthyBackoffMaxMs":10000,"shardFailFast":false}`,
			wantRate: 0.5, wantDialBase: 50 * time.Millisecond,
			wantUnhBase: 500 * time.Millisecond, wantUnhMax: 10 * time.Second, wantFailFast: false,
			wantLease: 1250 * time.Millisecond,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockEtcd()
			if tc.doc != "" {
				m.put(model.ClientConfigPath("recsys", "catalog"), tc.doc)
			}
			withLookup(failLookup, func() {
				c, err := newClientWithEtcd(Config{Tenant: "recsys", Store: "catalog"}, m, nil)
				require.NoError(t, err)
				defer c.Close()

				assert.Equal(t, tc.wantRate, c.pool.cfg.DialRatePerPod)
				assert.Equal(t, tc.wantDialBase, c.pool.cfg.DialBackoffBase)
				c.router.mu.RLock()
				defer c.router.mu.RUnlock()
				assert.Equal(t, tc.wantUnhBase, c.router.unhealthyBase)
				assert.Equal(t, tc.wantUnhMax, c.router.unhealthyMax)
				assert.Equal(t, tc.wantFailFast, c.router.failFast)
				assert.Equal(t, tc.wantLease, c.router.probeLease)
			})
		})
	}
}

func TestNewClientWithEtcd_CountsShardUnavailable(t *testing.T) {
	m := newMockEtcd()
	putActiveV1(m, metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.10:9091"}}))
	mc := &metricCollector{}
	withLookup(failLookup, func() {
		c, err := newClientWithEtcd(Config{
			Tenant: "recsys", Store: "catalog", Count: mc.count, DNSRefreshInterval: time.Hour,
			ClientConfig: &model.ClientConfig{WarmUpOnTopologyChange: boolPtr(false)},
		}, m, nil)
		require.NoError(t, err)
		defer c.Close()
		require.Eventually(t, func() bool { return c.router.ShardCount() == 1 }, 3*time.Second, 5*time.Millisecond)

		c.router.MarkUnhealthy("10.0.1.10:9091")
		_, err = c.StringGet(context.Background(), []byte("e:1"))
		require.ErrorIs(t, err, ErrShardUnavailable)
		got := mc.findAll(MetricRouterShardUnavailable)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"tenant:recsys", "store:catalog"}, got[0].Tags)
	})
}

// fakeRouterClock puts c's router on a settable clock with nominal waits.
func fakeRouterClock(c *Client) *fakeClock {
	clk := &fakeClock{t: redialT0}
	c.router.mu.Lock()
	c.router.now = clk.now
	c.router.float = func() float64 { return 0.5 }
	c.router.mu.Unlock()
	return clk
}

// Every read of a shard whose replicas are all inside their windows fails at
// once, without a dial, and the batch calls fail just that shard's keys.
func TestClient_ShardUnavailableFailsEveryReadWithoutDialling(t *testing.T) {
	d := countDials(t, true)
	reads := map[string]func(c *Client) error{
		"Get":       func(c *Client) error { _, err := c.Get(context.Background(), key12("k")); return err },
		"StringGet": func(c *Client) error { _, err := c.StringGet(context.Background(), []byte("e:1")); return err },
		"BatchGet": func(c *Client) error {
			res, _ := c.BatchGet(context.Background(), [][]byte{key12("k")})
			return res[0].Err
		},
		"StringBatchGet": func(c *Client) error {
			res, _ := c.StringBatchGet(context.Background(), [][]byte{[]byte("e:1")})
			return res[0].Err
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			c := staticClient(1, map[uint32][]string{0: {"10.0.1.10:9091", "10.0.1.11:9091"}})
			defer c.Close()
			fakeRouterClock(c)
			c.router.MarkUnhealthy("10.0.1.10:9091")
			c.router.MarkUnhealthy("10.0.1.11:9091")

			assert.ErrorIs(t, read(c), ErrShardUnavailable)
			assert.Zero(t, d.count("10.0.1.10:9091")+d.count("10.0.1.11:9091"))
		})
	}
}

// A successful read (a miss included) is the probe's success: the pod's mark
// and its backoff end.
func TestClient_ASuccessfulReadEndsThePodsBackoff(t *testing.T) {
	addr := startFakeServer(t, map[string][]byte{string(key12("k")): []byte("v"), "e:1": []byte("v")})
	reads := map[string]func(c *Client) error{
		"Get hit":  func(c *Client) error { _, err := c.Get(context.Background(), key12("k")); return err },
		"Get miss": func(c *Client) error { _, err := c.Get(context.Background(), key12("absent")); return ignoreMiss(err) },
		"StringGet hit": func(c *Client) error {
			_, err := c.StringGet(context.Background(), []byte("e:1"))
			return err
		},
		"StringGet miss": func(c *Client) error {
			_, err := c.StringGet(context.Background(), []byte("e:2"))
			return ignoreMiss(err)
		},
		"BatchGet": func(c *Client) error { _, err := c.BatchGet(context.Background(), [][]byte{key12("k")}); return err },
		"StringBatchGet": func(c *Client) error {
			_, err := c.StringBatchGet(context.Background(), [][]byte{[]byte("e:1")})
			return err
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			c := staticClient(1, map[uint32][]string{0: {addr}})
			defer c.Close()
			clk := fakeRouterClock(c)
			mc := &metricCollector{}
			c.router.SetMetrics(mc.count, nil)
			c.router.MarkUnhealthy(addr)
			clk.advance(time.Second)

			require.NoError(t, read(c))
			assert.False(t, isUnhealthy(c.router, addr))
			assert.Equal(t, int64(1), countTotal(mc, MetricRouterProbe, "result:ok"))
		})
	}
}

func ignoreMiss(err error) error {
	if errors.Is(err, ErrKeyNotFound) {
		return nil
	}
	return err
}

// This client's own dial rate refusing a dial says nothing about the pod: it
// is not marked unhealthy. A pod that refuses dials because its dials keep
// failing is: a probe refused that way counts as failed.
func TestClient_DialRefusalsAndUnhealthyMarks(t *testing.T) {
	t.Run("own dial rate: not marked", func(t *testing.T) {
		addr := startFakeServer(t, map[string][]byte{"e:1": []byte("v")})
		c := staticClient(1, map[uint32][]string{0: {addr}})
		defer c.Close()
		c.pool.Close()
		c.pool = redialPool(PoolConfig{MaxDialsPerPod: 1, DialRatePerPod: 0.001}, &fakeClock{t: redialT0})
		_, err := c.StringGet(context.Background(), []byte("e:1")) // spends the only token
		require.NoError(t, err)
		gone, err := c.pool.GetContext(context.Background(), addr) // the idle one, which the peer then closes
		require.NoError(t, err)
		discardBroken(c.pool, addr, gone)

		_, err = c.StringGet(context.Background(), []byte("e:1"))
		require.ErrorIs(t, err, errDialRateLimited)
		assert.False(t, isUnhealthy(c.router, addr))
	})
	t.Run("own dial rate during a probe: the probe is freed, the window kept", func(t *testing.T) {
		addr := startFakeServer(t, map[string][]byte{"e:1": []byte("v")})
		c := staticClient(1, map[uint32][]string{0: {addr}})
		defer c.Close()
		c.pool.Close()
		c.pool = redialPool(PoolConfig{MaxDialsPerPod: 1, DialRatePerPod: 0.001}, &fakeClock{t: redialT0})
		clk := fakeRouterClock(c)
		gone, err := c.pool.GetContext(context.Background(), addr) // spends the only token
		require.NoError(t, err)
		discardBroken(c.pool, addr, gone)
		c.router.MarkUnhealthy(addr)
		clk.advance(time.Second)

		_, err = c.StringGet(context.Background(), []byte("e:1")) // the probe, refused by the rate
		require.ErrorIs(t, err, errDialRateLimited)
		_, err = c.StringGet(context.Background(), []byte("e:1"))
		require.ErrorIs(t, err, errDialRateLimited, "routed as the next probe, not failed as unavailable")
		c.router.mu.RLock()
		defer c.router.mu.RUnlock()
		assert.Equal(t, 1, c.router.unhealthy[addr].failures)
	})
	t.Run("pod in dial backoff: a refused probe failed", func(t *testing.T) {
		countDials(t, true)
		c := staticClient(1, map[uint32][]string{0: {"10.0.1.10:9091"}})
		defer c.Close()
		c.pool.Close()
		c.pool = redialPool(PoolConfig{DialBackoffBase: time.Minute}, &fakeClock{t: redialT0})
		clk := fakeRouterClock(c)
		mc := &metricCollector{}
		c.router.SetMetrics(mc.count, nil)

		_, err := c.StringGet(context.Background(), []byte("e:1")) // the dial fails: marked
		require.Error(t, err)
		require.Equal(t, time.Second, window(c.router, "10.0.1.10:9091"))
		clk.advance(time.Second)

		_, err = c.StringGet(context.Background(), []byte("e:1")) // the probe meets the dial backoff
		require.ErrorIs(t, err, errDialBackingOff)
		assert.Equal(t, 2*time.Second, window(c.router, "10.0.1.10:9091"))
		assert.Equal(t, int64(1), countTotal(mc, MetricRouterProbe, "result:fail"))
	})
}
