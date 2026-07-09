// perf — load/latency benchmark for the mNemo read-server cluster over TCP.
//
// Uses the SDK's topology-aware client (NewClient): it discovers shards + pods
// from etcd (activeVersion + assignment), load-balances across all replicas of a
// shard, scatter-gathers BatchGet across shards, and pools TCP connections — so
// you only give it etcd + tenant/store, not per-pod addresses.
//
// Build a static linux/amd64 binary and scp it to a VM in the same VPC:
//   cd mnemo/mnemo-go-sdk
//   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOWORK=off go build -o /tmp/mnemo-perf ./cmd/perf
//   gcloud compute scp --tunnel-through-iap /tmp/mnemo-perf <vm>:~/mnemo-perf
//
// Run (single vs multi-get):
//   ./mnemo-perf -etcd 10.138.72.120:2379 -tenant ds -store catalog_geohash_e2e \
//     -keys keys.txt -mode both -batch 50 -concurrency 32 -duration 30s
//
// Prometheus metrics are exposed on -metrics-port (default 9103) while the
// benchmark runs:
//   curl http://<vm>:9103/metrics
//
// NOTE: requires the version to be PROMOTED (the SDK resolves pods from the
// control plane's assignment in etcd). If nothing is promoted, the client has no
// pods to route to and every op errors.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	sdk "github.com/Meesho/BharatMLStack/mnemo/mnemo-go-sdk"
)

// ── Prometheus metrics ──────────────────────────────────────────────────────

var (
	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mnemo_client_request_duration_seconds",
		Help:    "Client-side request latency",
		Buckets: []float64{0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"tenant", "store", "op", "status"})

	requestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mnemo_client_request_total",
		Help: "Client-side request count",
	}, []string{"tenant", "store", "op", "status"})

	batchKeys = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mnemo_client_batch_keys",
		Help:    "Keys per batch request",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
	}, []string{"tenant", "store"})

	poolGet = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mnemo_client_pool_get_total",
		Help: "Connection pool get operations",
	}, []string{"tenant", "store", "result"})

	poolDialLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mnemo_client_pool_dial_duration_seconds",
		Help:    "New TCP connection dial latency",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"tenant", "store"})

	poolOverflow = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mnemo_client_pool_overflow_total",
		Help: "Pool ceiling exceeded",
	}, []string{"tenant", "store"})

	poolIdleEvicted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mnemo_client_pool_idle_evicted_total",
		Help: "Idle connections evicted",
	}, []string{"tenant", "store"})

	topologyReloads = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mnemo_client_topology_reloads_total",
		Help: "Topology reload events",
	}, []string{"tenant", "store", "status"})
)

func init() {
	prometheus.MustRegister(requestDuration, requestTotal, batchKeys,
		poolGet, poolDialLatency, poolOverflow, poolIdleEvicted, topologyReloads)
}

// parseTags extracts key:value pairs from SDK tag slices into a label map.
func parseTags(tags []string) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if i := strings.IndexByte(t, ':'); i > 0 {
			m[t[:i]] = t[i+1:]
		}
	}
	return m
}

// promTiming bridges SDK Timing callbacks to Prometheus histograms.
func promTiming(name string, value time.Duration, tags []string) {
	m := parseTags(tags)
	sec := value.Seconds()
	switch name {
	case sdk.MetricRequestLatency:
		requestDuration.WithLabelValues(m["tenant"], m["store"], m["op"], m["status"]).Observe(sec)
	case sdk.MetricPoolDialLatency:
		poolDialLatency.WithLabelValues(m["tenant"], m["store"]).Observe(sec)
	}
}

// promCount bridges SDK Count callbacks to Prometheus counters/histograms.
func promCount(name string, value int64, tags []string) {
	m := parseTags(tags)
	switch name {
	case sdk.MetricRequestCount:
		requestTotal.WithLabelValues(m["tenant"], m["store"], m["op"], m["status"]).Add(float64(value))
	case sdk.MetricBatchKeys:
		batchKeys.WithLabelValues(m["tenant"], m["store"]).Observe(float64(value))
	case sdk.MetricPoolGet:
		poolGet.WithLabelValues(m["tenant"], m["store"], m["result"]).Add(float64(value))
	case sdk.MetricPoolOverflow:
		poolOverflow.WithLabelValues(m["tenant"], m["store"]).Add(float64(value))
	case sdk.MetricPoolIdleEvicted:
		poolIdleEvicted.WithLabelValues(m["tenant"], m["store"]).Add(float64(value))
	case sdk.MetricTopologyReload:
		topologyReloads.WithLabelValues(m["tenant"], m["store"], m["status"]).Add(float64(value))
	}
}

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	etcd := flag.String("etcd", "10.138.72.120:2379", "comma-separated etcd endpoints for topology discovery")
	tenant := flag.String("tenant", "ds", "tenant")
	store := flag.String("store", "catalog_geohash_e2e", "store")
	keysFile := flag.String("keys", "", "file of real keys (one per line). If empty, synthetic keys are generated")
	genN := flag.Int("gen", 100000, "synthetic key count when -keys is empty")
	entity := flag.String("entity", "catalog__user_geohash_1_3:derived_fp32", "entity prefix for synthetic keys")
	mode := flag.String("mode", "both", "single | batch | both")
	batch := flag.Int("batch", 50, "keys per multi-get")
	concurrency := flag.Int("concurrency", 32, "parallel load-generator workers (offered load — NOT a client config)")
	duration := flag.Duration("duration", 30*time.Second, "measured run duration per mode")
	warmup := flag.Duration("warmup", 5*time.Second, "warmup per mode (also lets topology resolve; not recorded)")
	metricsPort := flag.Int("metrics-port", 9103, "Prometheus /metrics port (0 to disable)")
	minConns := flag.Int("min-conns", 0, "min idle connections per pod (0 = use control plane value or default 1)")
	maxConns := flag.Int("max-conns", 0, "max connections per pod (0 = use control plane value or default 4)")
	flag.Parse()

	// Start Prometheus metrics server.
	if *metricsPort > 0 {
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())
			addr := fmt.Sprintf(":%d", *metricsPort)
			fmt.Printf("Prometheus metrics at http://0.0.0.0%s/metrics\n", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				fmt.Fprintf(os.Stderr, "metrics server: %v\n", err)
			}
		}()
	}

	sdkCfg := sdk.Config{
		EtcdEndpoints: strings.Split(*etcd, ","),
		Tenant:        *tenant,
		Store:         *store,
		Timing:        promTiming,
		Count:         promCount,
	}
	if *minConns > 0 || *maxConns > 0 {
		sdkCfg.Pool = &sdk.PoolConfig{}
		if *minConns > 0 {
			sdkCfg.Pool.MinPerPod = *minConns
		}
		if *maxConns > 0 {
			sdkCfg.Pool.MaxPerPod = *maxConns
		}
	}
	client, err := sdk.NewClient(sdkCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: NewClient: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	keys := loadOrGenKeys(*keysFile, *genN, *entity)
	if len(keys) == 0 {
		fmt.Fprintln(os.Stderr, "FATAL: no keys")
		os.Exit(1)
	}

	fmt.Printf("mNemo perf — etcd=%s %s/%s keys=%d concurrency=%d batch=%d duration=%s\n",
		*etcd, *tenant, *store, len(keys), *concurrency, *batch, *duration)
	fmt.Println("(pool/timeouts from control-plane ClientConfig; shards/replicas from the promoted assignment)\n")

	cfg := runCfg{
		client: client, keys: keys, concurrency: *concurrency,
		duration: *duration, warmup: *warmup, batch: *batch,
	}
	if *mode == "single" || *mode == "both" {
		report("SINGLE GET", run(cfg, false))
	}
	if *mode == "batch" || *mode == "both" {
		report(fmt.Sprintf("BATCH GET (multi-get, batch=%d)", *batch), run(cfg, true))
	}
}

type runCfg struct {
	client      *sdk.Client
	keys        [][]byte
	concurrency int
	duration    time.Duration
	warmup      time.Duration
	batch       int
}

type result struct {
	lats    []time.Duration
	ops     int64
	keys    int64
	errs    int64
	elapsed time.Duration
	start   time.Time
	end     time.Time
}

func run(cfg runCfg, isBatch bool) result {
	if cfg.warmup > 0 {
		rampUp(cfg, isBatch, cfg.warmup)
	}
	return drive(cfg, isBatch, cfg.duration, true)
}

// rampUp gradually increases concurrency from 1 to cfg.concurrency over the
// warmup duration. Each step runs for an equal slice of time, doubling the
// worker count: 1 → 2 → 4 → 8 → ... → concurrency. This warms the RocksDB
// block cache, TCP connection pool, and OS page cache before the measured run.
func rampUp(cfg runCfg, isBatch bool, dur time.Duration) {
	// Build ramp steps: 1, 2, 4, 8, ..., concurrency
	var steps []int
	for c := 1; c < cfg.concurrency; c *= 2 {
		steps = append(steps, c)
	}
	steps = append(steps, cfg.concurrency)

	stepDur := dur / time.Duration(len(steps))
	if stepDur < 1*time.Second {
		stepDur = 1 * time.Second
	}

	for _, c := range steps {
		stepCfg := cfg
		stepCfg.concurrency = c
		fmt.Printf("  [warmup] concurrency=%d for %s\n", c, stepDur)
		drive(stepCfg, isBatch, stepDur, false)
	}
	fmt.Println("  [warmup] done — starting measured run")
}

func drive(cfg runCfg, isBatch bool, dur time.Duration, record bool) result {
	deadline := time.Now().Add(dur)
	var ops, keysN, errs int64
	perWorkerLats := make([][]time.Duration, cfg.concurrency)

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < cfg.concurrency; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(w)))
			var lats []time.Duration
			for time.Now().Before(deadline) {
				var lat time.Duration
				var k, e int
				if isBatch {
					lat, k, e = doBatch(cfg, rng)
				} else {
					lat, e = doSingle(cfg, rng)
					k = 1
				}
				atomic.AddInt64(&ops, 1)
				atomic.AddInt64(&keysN, int64(k))
				atomic.AddInt64(&errs, int64(e))
				if record {
					lats = append(lats, lat)
				}
			}
			perWorkerLats[w] = lats
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)

	var all []time.Duration
	for _, l := range perWorkerLats {
		all = append(all, l...)
	}
	return result{lats: all, ops: ops, keys: keysN, errs: errs, elapsed: elapsed, start: start, end: time.Now()}
}

// doSingle — SDK routes the key to its shard's pod (replica load-balanced).
// Uses a plain context so the SDK's own per-request timeout (from the control
// plane's requestTimeoutMs) is the sole governor — that's what we want to measure.
func doSingle(cfg runCfg, rng *rand.Rand) (lat time.Duration, errc int) {
	key := cfg.keys[rng.Intn(len(cfg.keys))]
	t0 := time.Now()
	_, err := cfg.client.StringGet(context.Background(), key)
	lat = time.Since(t0)
	if err != nil && err != sdk.ErrKeyNotFound { // a miss is a valid, timed response
		errc = 1
	}
	return
}

// doBatch — SDK scatter-gathers the batch across shards + replicas in one call.
// The recorded latency is the wall time for the whole multi-get.
func doBatch(cfg runCfg, rng *rand.Rand) (lat time.Duration, nkeys, errc int) {
	ks := make([][]byte, cfg.batch)
	for i := range ks {
		ks[i] = cfg.keys[rng.Intn(len(cfg.keys))]
	}
	nkeys = len(ks)
	t0 := time.Now()
	_, err := cfg.client.StringBatchGet(context.Background(), ks)
	lat = time.Since(t0)
	if err != nil {
		errc = 1
	}
	return
}

func report(title string, r result) {
	sort.Slice(r.lats, func(i, j int) bool { return r.lats[i] < r.lats[j] })
	opsPerSec := float64(r.ops) / r.elapsed.Seconds()
	keysPerSec := float64(r.keys) / r.elapsed.Seconds()
	fmt.Printf("== %s ==\n", title)
	fmt.Printf("  start        %s\n", r.start.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  end          %s\n", r.end.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  elapsed      %s\n", r.elapsed.Round(time.Millisecond))
	fmt.Printf("  throughput   %.0f ops/sec   %.0f keys/sec\n", opsPerSec, keysPerSec)
	fmt.Printf("  latency      p50=%s  p75=%s  p95=%s  p99=%s  p99.9=%s  max=%s\n",
		pct(r.lats, 50), pct(r.lats, 75), pct(r.lats, 95), pct(r.lats, 99), pct(r.lats, 99.9), pct(r.lats, 100))
	if r.errs > 0 {
		fmt.Printf("  errors       %d failed ops (check connectivity / promotion)\n", r.errs)
	}
	fmt.Println()
}

func pct(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p / 100 * float64(len(sorted)))
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i].Round(time.Microsecond)
}

func loadOrGenKeys(path string, n int, entity string) [][]byte {
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: open keys %s: %v\n", path, err)
			os.Exit(1)
		}
		defer f.Close()
		var keys [][]byte
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		for sc.Scan() {
			if line := strings.TrimSpace(sc.Text()); line != "" {
				keys = append(keys, []byte(line))
			}
		}
		return keys
	}
	keys := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = []byte(fmt.Sprintf("%s:%d|%d", entity, i, i%4096))
	}
	return keys
}
