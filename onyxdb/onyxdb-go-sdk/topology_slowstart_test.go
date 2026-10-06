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

func putPod(t *testing.T, m *mockEtcd, podID string, pd model.PodData) {
	t.Helper()
	b, err := json.Marshal(pd)
	require.NoError(t, err)
	m.put(model.PodDataPath("recsys", "catalog", podID), string(b))
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
		putPod(t, m, "recsys-catalog-shard-0-old", model.PodData{PodIP: "10.0.1.10", WarmVersions: []string{"v1"}})
		putPod(t, m, "recsys-catalog-shard-0-new", model.PodData{
			PodIP: "10.0.1.20", WarmVersions: []string{"v1"}, ServingSince: ramping.UnixMilli(), SlowStartSec: 300,
		})
		putPod(t, m, "recsys-catalog-shard-0-port", model.PodData{
			PodIP: "10.0.1.30", Port: 9300, WarmVersions: []string{"v1"}, ServingSince: ramping.UnixMilli(), SlowStartSec: 300,
		})
		putPod(t, m, "recsys-catalog-shard-0-done", model.PodData{
			PodIP: "10.0.1.40", WarmVersions: []string{"v1"}, ServingSince: now.Add(-time.Hour).UnixMilli(), SlowStartSec: 300,
		})
		putPod(t, m, "recsys-catalog-shard-0-nowindow", model.PodData{
			PodIP: "10.0.1.50", WarmVersions: []string{"v1"}, ServingSince: ramping.UnixMilli(),
		})
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
		putPod(t, m, "recsys-catalog-shard-0-new", model.PodData{
			PodIP: "10.0.1.20", ServingSince: time.Now().UnixMilli(), SlowStartSec: 300,
		})
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
		putPod(t, m, "recsys-catalog-shard-0-old", model.PodData{PodIP: "10.0.1.10", WarmVersions: []string{"v1"}})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go tw.Run(ctx)
		require.Eventually(t, func() bool { pod, err := r.PodFor(0); return err == nil && pod == "10.0.1.10:9091" },
			time.Second, 5*time.Millisecond)

		// The new pod goes warm: the control plane adds it to the assignment
		// and its registration carries the ramp it just started.
		m.put(model.VersionPrefix("recsys", "catalog", "v1"),
			metaWithAssignmentJSON(t, 1, map[string][]string{"0": {"10.0.1.10:9091", "10.0.1.20:9091"}}))
		putPod(t, m, "recsys-catalog-shard-0-new", model.PodData{
			PodIP: "10.0.1.20", WarmVersions: []string{"v1"}, ServingSince: time.Now().UnixMilli(), SlowStartSec: 300,
		})
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
