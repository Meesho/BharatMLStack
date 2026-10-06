package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// The topology watcher reads slow-start ramps from the pods' own etcd
// registrations (docs/design/onyxdb-pod-slow-start.md) on every reload.

// loaderRegistration is a pod registration as the dataloader writes it
// (controlplane/model.PodData's JSON), spelled out key by key. The SDK decodes
// it with its own podRegistration so that it builds against the released
// model, which predates these fields: these literal keys are the wire contract.
// Zero values are left out, as the model's omitempty tags do.
func loaderRegistration(t *testing.T, podIP string, port int, servingSince int64, slowStartSec int) string {
	t.Helper()
	reg := map[string]any{"nodeIP": "10.0.0.9", "podIP": podIP, "servingVersion": "v1", "warmVersions": []string{"v1"}}
	if port != 0 {
		reg["port"] = port
	}
	if servingSince != 0 {
		reg["servingSince"] = servingSince
	}
	if slowStartSec != 0 {
		reg["slowStartSec"] = slowStartSec
	}
	b, err := json.Marshal(reg)
	require.NoError(t, err)
	return string(b)
}

func putPod(t *testing.T, m *mockEtcd, podID, registration string) {
	t.Helper()
	m.put(model.PodDataPath("recsys", "catalog", podID), registration)
}

func slowStartWatcher(t *testing.T, assignment map[string][]string) (*mockEtcd, *TopologyWatcher, *Router) {
	t.Helper()
	m := newMockEtcd()
	m.put(model.ActiveVersionPath("recsys", "catalog"), "v1")
	m.put(model.VersionPrefix("recsys", "catalog", "v1"), metaWithAssignmentJSON(t, 1, assignment))
	tw, r, _ := newWatcherWith(m)
	return m, tw, r
}

func TestReloadSlowStart_ReadsRampsFromPodRegistrations(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m, tw, r := slowStartWatcher(t, map[string][]string{"0": {"10.0.1.10:9091", "10.0.1.20:9091", "10.0.1.30:9300"}})
		now := time.Now()
		ramping := now.Add(-60 * time.Second)
		putPod(t, m, "recsys-catalog-shard-0-old", loaderRegistration(t, "10.0.1.10", 0, 0, 0))
		putPod(t, m, "recsys-catalog-shard-0-new", loaderRegistration(t, "10.0.1.20", 0, ramping.UnixMilli(), 300))
		putPod(t, m, "recsys-catalog-shard-0-port", loaderRegistration(t, "10.0.1.30", 9300, ramping.UnixMilli(), 300))
		putPod(t, m, "recsys-catalog-shard-0-done", loaderRegistration(t, "10.0.1.40", 0, now.Add(-time.Hour).UnixMilli(), 300))
		putPod(t, m, "recsys-catalog-shard-0-nowindow", loaderRegistration(t, "10.0.1.50", 0, ramping.UnixMilli(), 0))
		m.put(model.PodDataPath("recsys", "catalog", "recsys-catalog-shard-0-corrupt"), "{not json")

		require.NoError(t, tw.reload(context.Background()))

		r.mu.RLock()
		defer r.mu.RUnlock()
		want := SlowStart{Since: time.UnixMilli(ramping.UnixMilli()), Window: 300 * time.Second}
		assert.Equal(t, map[string]SlowStart{"10.0.1.20:9091": want, "10.0.1.30:9300": want}, r.slowStart,
			"only registrations inside their window, keyed like the assignment entry")
	})
}

func TestReloadSlowStart_PodReadFailureKeepsTheLastSet(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m, tw, r := slowStartWatcher(t, map[string][]string{"0": {"10.0.1.10:9091", "10.0.1.20:9091"}})
		putPod(t, m, "recsys-catalog-shard-0-new", loaderRegistration(t, "10.0.1.20", 0, time.Now().UnixMilli(), 300))
		require.NoError(t, tw.reload(context.Background()))
		r.mu.RLock()
		before := r.slowStart
		r.mu.RUnlock()
		require.Len(t, before, 1)

		m.setGetErr(model.PodWatchPrefix("recsys", "catalog"), errors.New("etcd unavailable"))
		require.NoError(t, tw.reload(context.Background()), "a failed pod read must not fail the reload")

		r.mu.RLock()
		defer r.mu.RUnlock()
		assert.Equal(t, before, r.slowStart)
	})
}

// End to end through Run: a new pod's registration with a ramp, arriving on
// the pod watch, makes PodFor hand it only a trickle at first.
func TestRun_NewPodRegistrationRampsItsShare(t *testing.T) {
	withLookup(okLookup("10.0.0.1"), func() {
		m, tw, r := slowStartWatcher(t, map[string][]string{"0": {"10.0.1.10:9091"}})
		putPod(t, m, "recsys-catalog-shard-0-old", loaderRegistration(t, "10.0.1.10", 0, 0, 0))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)
		require.Eventually(t, func() bool { pod, err := r.PodFor(0); return err == nil && pod == "10.0.1.10:9091" },
			time.Second, 5*time.Millisecond)

		// The new pod goes warm: the control plane adds it to the assignment
		// and its registration carries the ramp it just started.
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.10:9091", "10.0.1.20:9091"}}))
		putPod(t, m, "recsys-catalog-shard-0-new", loaderRegistration(t, "10.0.1.20", 0, time.Now().UnixMilli(), 300))
		m.podCh() <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: mvccpb.PUT,
			Kv: &mvccpb.KeyValue{Key: []byte(model.PodDataPath("recsys", "catalog", "recsys-catalog-shard-0-new"))}}}}

		require.Eventually(t, func() bool {
			r.mu.RLock()
			defer r.mu.RUnlock()
			return len(r.slowStart) == 1
		}, time.Second, 5*time.Millisecond)
		toNew := 0
		for i := 0; i < 10_000; i++ {
			if pod, _ := r.PodFor(0); pod == "10.0.1.20:9091" {
				toNew++
			}
		}
		// Seconds into a 300 s window the weight is at its 1% floor: ~1% of
		// reads, against 50% under plain round-robin.
		assert.Positive(t, toNew, "the ramping pod still gets a trickle")
		assert.Less(t, toNew, 300, "the ramping pod got %d of 10000 reads", toNew)
	})
}

// The decode the router relies on, against literal loader JSON: the keys, the
// omitted zero values, and the address placement routes to.
func TestPodRegistration_DecodesTheLoaderJSON(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		want     podRegistration
		wantAddr string
	}{
		{"ramping, default port",
			`{"nodeIP":"10.0.0.9","podIP":"10.0.1.20","servingVersion":"v1","warmVersions":["v1"],"servingSince":1791281000123,"slowStartSec":300}`,
			podRegistration{PodIP: "10.0.1.20", ServingSince: 1791281000123, SlowStartSec: 300}, "10.0.1.20:9091"},
		{"explicit port",
			`{"nodeIP":"10.0.0.9","podIP":"10.0.1.30","port":9300,"servingVersion":"v1","warmVersions":["v1"],"servingSince":5,"slowStartSec":60}`,
			podRegistration{PodIP: "10.0.1.30", Port: 9300, ServingSince: 5, SlowStartSec: 60}, "10.0.1.30:9300"},
		{"written by a loader that predates slow start",
			`{"nodeIP":"10.0.0.9","podIP":"10.0.1.10","servingVersion":"v1","warmVersions":["v1"]}`,
			podRegistration{PodIP: "10.0.1.10"}, "10.0.1.10:9091"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got podRegistration
			require.NoError(t, json.Unmarshal([]byte(tc.json), &got))
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantAddr, got.addr())
		})
	}
}
