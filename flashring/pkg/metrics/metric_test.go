package metrics

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// withEnabled flips metricsEnabled for the test and restores it on teardown.
func withEnabled(t *testing.T, enabled bool) {
	t.Helper()
	prev := metricsEnabled
	metricsEnabled = enabled
	t.Cleanup(func() { metricsEnabled = prev })
}

// captureLog redirects the zerolog global logger to a buffer for the duration
// of the test. Returns the buffer so callers can assert on (or against) any
// log line emitted by the code under test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := log.Logger
	log.Logger = zerolog.New(buf)
	t.Cleanup(func() { log.Logger = prev })
	return buf
}

func TestLoadMetricsEnabled_Unset_ReturnsFalse(t *testing.T) {
	t.Setenv("FLASHRING_METRICS_ENABLED", "")

	if loadMetricsEnabled() {
		t.Fatal("loadMetricsEnabled with empty env = true, want false")
	}
}

func TestLoadMetricsEnabled_True_ReturnsTrue(t *testing.T) {
	t.Setenv("FLASHRING_METRICS_ENABLED", "true")

	if !loadMetricsEnabled() {
		t.Fatal("loadMetricsEnabled with 'true' = false, want true")
	}
}

func TestLoadMetricsEnabled_CaseInsensitiveTrue(t *testing.T) {
	t.Setenv("FLASHRING_METRICS_ENABLED", "TrUe")

	if !loadMetricsEnabled() {
		t.Fatal("loadMetricsEnabled with 'TrUe' = false, want true")
	}
}

func TestLoadMetricsEnabled_NumericOne_ReturnsTrue(t *testing.T) {
	t.Setenv("FLASHRING_METRICS_ENABLED", "1")

	if !loadMetricsEnabled() {
		t.Fatal("loadMetricsEnabled with '1' = false, want true")
	}
}

func TestLoadMetricsEnabled_OtherValue_ReturnsFalse(t *testing.T) {
	t.Setenv("FLASHRING_METRICS_ENABLED", "yes")

	if loadMetricsEnabled() {
		t.Fatal("loadMetricsEnabled with 'yes' = true, want false")
	}
}

// Only an explicit "false" (any case) or "0" turns the predictor group off.
func TestLoadPredictorMetricsEnabled(t *testing.T) {
	for _, tc := range []struct {
		name, env string
		want      bool
	}{
		{"unset defaults on", "", true},
		{"false", "false", false},
		{"false in mixed case", "FaLsE", false},
		{"zero", "0", false},
		{"any other value stays on", "maybe", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FLASHRING_METRICS_PREDICTOR_ENABLED", tc.env)
			if got := loadPredictorMetricsEnabled(); got != tc.want {
				t.Fatalf("loadPredictorMetricsEnabled() with %q = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestPredictorMetricsEnabled_ReflectsGlobalState(t *testing.T) {
	prev := predictorMetricsEnabled
	t.Cleanup(func() { predictorMetricsEnabled = prev })

	predictorMetricsEnabled = true
	if !PredictorMetricsEnabled() {
		t.Fatal("PredictorMetricsEnabled() = false when predictorMetricsEnabled=true")
	}
	predictorMetricsEnabled = false
	if PredictorMetricsEnabled() {
		t.Fatal("PredictorMetricsEnabled() = true when predictorMetricsEnabled=false")
	}
}

func TestEnabled_ReflectsGlobalState(t *testing.T) {
	withEnabled(t, true)
	if !Enabled() {
		t.Fatal("Enabled() = false when metricsEnabled=true")
	}
	withEnabled(t, false)
	if Enabled() {
		t.Fatal("Enabled() = true when metricsEnabled=false")
	}
}

// withNilClient temporarily nils out statsDClient so the underlying datadog
// statsd methods return ErrNoClient — that's the only deterministic way to
// drive the `if err != nil` warn branch in Timing/Count/Gauge.
func withNilClient(t *testing.T) {
	t.Helper()
	prev := statsDClient
	statsDClient = nil
	t.Cleanup(func() { statsDClient = prev })
}

// emitter wraps the four metric-emission entry points so a single
// table-driven test can drive each through identical disabled / enabled /
// client-error scenarios.
type emitter struct {
	name string
	call func()
}

func emitters() []emitter {
	return []emitter{
		{"Timing", func() { Timing("flashring_get_latency", 1*time.Millisecond, []string{"shard_idx:0"}) }},
		{"Count", func() { Count("flashring_gets", 3, []string{"shard_idx:0"}) }},
		{"Incr", func() { Incr("flashring_hits", []string{"shard_idx:0"}) }},
		{"Gauge", func() { Gauge("flashring_hitrate", 0.95, []string{"shard_idx:0"}) }},
	}
}

// TestEmitters_Disabled_AreNoOps asserts the early-return contract: even
// with a nil statsd client (which would error on dispatch if reached), each
// call must complete and emit no log line. That proves the function
// short-circuited at the `if !metricsEnabled` guard.
func TestEmitters_Disabled_AreNoOps(t *testing.T) {
	for _, e := range emitters() {
		t.Run(e.name, func(t *testing.T) {
			withEnabled(t, false)
			withNilClient(t)
			buf := captureLog(t)

			e.call()

			if buf.Len() != 0 {
				t.Fatalf("Disabled %s logged %q, want no log", e.name, buf.String())
			}
		})
	}
}

// withHealthyClient installs a freshly-built UDP statsd client targeting
// localhost:8125 so the test exercises the actual dispatch path instead of
// inheriting whatever ambient state the package globals happen to hold.
// Required because a previous test in the same package could have nil'd
// out statsDClient (see withNilClient).
func withHealthyClient(t *testing.T) {
	t.Helper()
	prev := statsDClient
	statsDClient = getDefaultClient()
	t.Cleanup(func() { statsDClient = prev })
}

// TestEmitters_Enabled_HealthyClient_NoLog drives the happy dispatch path:
// statsd's UDP client returns nil from Timing/Count/Gauge so the function
// body never enters the warn branch and the log stream stays empty.
func TestEmitters_Enabled_HealthyClient_NoLog(t *testing.T) {
	for _, e := range emitters() {
		t.Run(e.name, func(t *testing.T) {
			withEnabled(t, true)
			withHealthyClient(t)
			buf := captureLog(t)

			e.call()

			if buf.Len() != 0 {
				t.Fatalf("Enabled %s with healthy client logged %q, want no log", e.name, buf.String())
			}
		})
	}
}

// TestEmitters_Enabled_NilClient_PreservesState characterizes the err != nil
// branch. Note: in metric.go the warn statement is `log.Warn().AnErr(...)`
// without a terminating `.Msg(...)`, so zerolog never finalizes the event —
// the warn branch is observationally silent (refactor signal in
// docs/unit-test-coverage-report.md). The contract we assert is that the
// function completes without panic and leaves the global metric state
// exactly as the test set it up. Incr is excluded because it delegates to
// Count and the err is already exercised there.
func TestEmitters_Enabled_NilClient_PreservesState(t *testing.T) {
	cases := []emitter{
		{"Timing", func() { Timing("flashring_get_latency", 1*time.Millisecond, nil) }},
		{"Count", func() { Count("flashring_gets", 1, nil) }},
		{"Gauge", func() { Gauge("flashring_hitrate", 0.5, nil) }},
	}
	for _, e := range cases {
		t.Run(e.name, func(t *testing.T) {
			withEnabled(t, true)
			withNilClient(t)
			buf := captureLog(t)

			e.call()

			if !Enabled() {
				t.Fatalf("%s flipped Enabled() during error branch", e.name)
			}
			if statsDClient != nil {
				t.Fatalf("%s repopulated statsDClient during error branch", e.name)
			}
			if buf.Len() != 0 {
				t.Fatalf("%s wrote to log unexpectedly: %q", e.name, buf.String())
			}
		})
	}
}

func TestGetDefaultClient_ReturnsNonNilClient(t *testing.T) {
	c := getDefaultClient()
	if c == nil {
		t.Fatal("getDefaultClient() returned nil")
	}
}

func TestGetGlobalTags_WithEnvAndService(t *testing.T) {
	viper.Reset()
	viper.Set("APP_ENV", "test")
	viper.Set("APP_NAME", "flashring")
	t.Cleanup(viper.Reset)

	got := getGlobalTags()
	want := []string{"env:test", "service:flashring"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("getGlobalTags = %v, want %v", got, want)
	}
}

func TestGetGlobalTags_MissingEnvAndService_StillReturnsTwoTags(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	got := getGlobalTags()
	want := []string{"env:", "service:"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("getGlobalTags = %v, want %v", got, want)
	}
}

func TestBuildShardTags_AndGetShardTag_ReturnsPerShardTag(t *testing.T) {
	prev := shardTags
	t.Cleanup(func() { shardTags = prev })

	BuildShardTags(3)

	for i := uint32(0); i < 3; i++ {
		got := GetShardTag(i)
		if len(got) != 1 {
			t.Fatalf("GetShardTag(%d) len = %d, want 1", i, len(got))
		}
		want := "shard_idx:" + itoa(int(i))
		if got[0] != want {
			t.Fatalf("GetShardTag(%d) = %q, want %q", i, got[0], want)
		}
	}
}

func TestBuildShardTags_Zero_GivesEmptyShardTags(t *testing.T) {
	prev := shardTags
	t.Cleanup(func() { shardTags = prev })

	BuildShardTags(0)

	if len(shardTags) != 0 {
		t.Fatalf("shardTags len = %d, want 0", len(shardTags))
	}
}

// TestInit covers both branches: the once.Do body on first call and the
// `initialized` short-circuit on subsequent calls. Because `once`,
// `initialized`, and `samplingRate` are package-globals, the test resets
// them to a known baseline before running so it is independent of
// execution order with the rest of the package's tests.
//
// The sync.Once value-assignment below is flagged by go vet's `copylocks`
// check but is safe here: this test is the sole writer of `once` and is
// strictly sequential, so the lock state can't be torn. Equivalent
// production code would be wrong; test-reset code is the documented
// exception.
//
//nolint:copylocks // intentional once-reset; see comment above
func TestInit_FirstCall_RunsAndSubsequentCallsShortCircuit(t *testing.T) {
	prevInitialized := initialized
	prevSamplingRate := samplingRate
	t.Cleanup(func() {
		initialized = prevInitialized
		samplingRate = prevSamplingRate
		once = sync.Once{}
	})

	// Force a clean baseline so Init()'s body actually runs.
	initialized = false
	samplingRate = 0
	once = sync.Once{}

	viper.Reset()
	viper.Set("APP_ENV", "test")
	viper.Set("APP_NAME", "flashring")
	viper.Set("APP_METRIC_SAMPLING_RATE", 0.25)
	t.Cleanup(viper.Reset)

	Init()
	if !initialized {
		t.Fatal("initialized = false after first Init()")
	}
	if samplingRate != 0.25 {
		t.Fatalf("samplingRate = %f, want 0.25", samplingRate)
	}

	// Second call must return early via the `if initialized` branch.
	Init()
	if !initialized {
		t.Fatal("initialized flipped back to false after second Init()")
	}
}

// itoa avoids importing strconv just for one helper; behaviour-identical for
// the small non-negative values the suite uses.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// Appending to a shard's tag slice used to write into the shared array and
// replace the next shard's tag (the old flush counter did exactly that with
// its memtable_id tag).
func TestGetShardTag_AppendDoesNotOverwriteNextShard(t *testing.T) {
	prev := shardTags
	t.Cleanup(func() { shardTags = prev })

	BuildShardTags(2)
	_ = append(GetShardTag(0), "extra:1")
	if got, want := GetShardTag(1)[0], "shard_idx:1"; got != want {
		t.Fatalf("GetShardTag(1) = %q after appending to shard 0's tags, want %q", got, want)
	}
}
