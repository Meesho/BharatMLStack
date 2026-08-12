package profiling_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/profiling"
	"github.com/spf13/viper"
)

// The point of this test is compatibility, not coverage: the platform's
// Grafana dashboards select on metric names emitted by the Prometheus Go
// collector and on the service/env labels. If either drifts, the panels go
// blank without anything failing, so assert both against a live endpoint.
func TestMetricsEndpointServesRuntimeMetricsAndPprof(t *testing.T) {
	viper.Set("APP_NAME", "skye-test")
	viper.Set("APP_ENV", "stg")

	metric.Init()
	if metric.Mux() == nil || metric.Registerer() == nil {
		t.Fatal("metric.Init did not start the metrics server")
	}

	if err := profiling.InitWithOptions(
		profiling.WithPprof(profiling.PprofAll...),
		profiling.WithRuntimeMetrics(profiling.RuntimeMetricAll),
	); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}

	base := "http://" + strings.Replace(metric.Addr(), "[::]", "127.0.0.1", 1)
	body := get(t, base+"/metrics")

	// Names the go-runtime-metrics dashboard reads.
	for _, want := range []string{
		"go_gc_heap_allocs_bytes_total",
		"go_sched_latencies_seconds",
		"go_memory_classes_total_bytes",
		"go_cpu_classes_total_cpu_seconds_total",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing runtime metric %q", want)
		}
	}
	// Every series must carry both identifying labels or the dashboard's
	// service/env template variables match nothing.
	for _, want := range []string{`service="skye-test"`, `env="stg"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing label %s on the exposed series", want)
		}
	}

	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "go_gc_heap_allocs_bytes_total") || strings.HasPrefix(l, "go_sched_latencies_seconds_count") {
			t.Logf("sample series: %s", l)
		}
	}

	// pprof must be on the same port as /metrics, not a second one.
	for _, path := range []string{"/debug/pprof/heap", "/debug/pprof/goroutine", "/debug/pprof/cmdline"} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, resp.StatusCode)
		}
	}
}

func get(t *testing.T, url string) string {
	t.Helper()
	var last error
	for i := 0; i < 20; i++ {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			return string(b)
		}
		last = err
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("GET %s never succeeded: %v", url, last)
	return ""
}
