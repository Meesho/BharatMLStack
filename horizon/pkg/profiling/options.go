package profiling

// PprofHandler selects one pprof endpoint to mount.
type PprofHandler int

const (
	PprofHeap PprofHandler = iota
	PprofAllocs
	PprofGoroutine
	PprofThreadcreate
	PprofBlock
	PprofMutex
	PprofProfile
	PprofTrace
	PprofCmdline
	PprofSymbol
)

// PprofAll is every handler above. Snapshot handlers are cheap; Profile and
// Trace only cost while a request is in flight.
var PprofAll = []PprofHandler{
	PprofHeap, PprofAllocs, PprofGoroutine, PprofThreadcreate,
	PprofBlock, PprofMutex, PprofProfile, PprofTrace, PprofCmdline, PprofSymbol,
}

// RuntimeMetric selects a family of Go runtime metrics to export.
type RuntimeMetric int

const (
	RuntimeMetricMemory RuntimeMetric = iota
	RuntimeMetricGC
	RuntimeMetricScheduler
	RuntimeMetricAll
)

// RuntimeMetricAllSet is the full set, matching the go-runtime-metrics dashboard.
var RuntimeMetricAllSet = []RuntimeMetric{RuntimeMetricAll}

type config struct {
	pprofHandlers      []PprofHandler
	runtimeMetrics     []RuntimeMetric
	continuousProfiler bool
}

// Option configures InitWithOptions.
type Option func(*config)

// WithPprof mounts the given pprof handlers on the metrics server. Called with
// no arguments it mounts the snapshot-safe set (everything except Profile and
// Trace, which hold a request open for their duration).
func WithPprof(handlers ...PprofHandler) Option {
	return func(c *config) {
		if len(handlers) == 0 {
			handlers = []PprofHandler{
				PprofHeap, PprofAllocs, PprofGoroutine,
				PprofThreadcreate, PprofBlock, PprofMutex,
			}
		}
		c.pprofHandlers = append(c.pprofHandlers, handlers...)
	}
}

// WithRuntimeMetrics exports the given Go runtime metric families to Prometheus.
func WithRuntimeMetrics(metrics ...RuntimeMetric) Option {
	return func(c *config) { c.runtimeMetrics = append(c.runtimeMetrics, metrics...) }
}

// WithContinuousProfiler starts the Google Cloud Profiler agent. Requires
// APP_NAME and CICD_VERSION_ID; a failure is logged and does not fail Init.
func WithContinuousProfiler() Option {
	return func(c *config) { c.continuousProfiler = true }
}
