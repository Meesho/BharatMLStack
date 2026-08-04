package sdk

import "time"

// Metric name constants emitted by the SDK. Callers receive these via the
// Config.Timing / Config.Count callbacks and forward them to their metrics
// backend (typically Datadog StatsD).
const (
	// Client request metrics (emitted from Get/BatchGet/StringGet/StringBatchGet).
	// Tags: tenant, store, op, status
	MetricRequestLatency = "onyxdb.request.latency"
	MetricRequestCount   = "onyxdb.request.count"

	// Batch size observation (emitted once per BatchGet/StringBatchGet call).
	// Tags: tenant, store
	MetricBatchKeys = "onyxdb.batch.keys"

	// Connection pool metrics.
	// Tags: tenant, store, result (hit/dial/error)
	MetricPoolGet = "onyxdb.pool.get"
	// Tags: tenant, store
	MetricPoolDialLatency = "onyxdb.pool.dial.latency"
	MetricPoolIdleEvicted = "onyxdb.pool.idle_evicted"
	MetricPoolOverflow    = "onyxdb.pool.overflow"

	// Topology watcher metrics.
	// Tags: tenant, store, status (ok/error)
	MetricTopologyReload = "onyxdb.topology.reload"
)

// emitTiming is a nil-safe timing callback on Client.
func (c *Client) emitTiming(name string, value time.Duration, tags []string) {
	if c.config.Timing != nil {
		c.config.Timing(name, value, tags)
	}
}

// emitCount is a nil-safe count callback on Client.
func (c *Client) emitCount(name string, value int64, tags []string) {
	if c.config.Count != nil {
		c.config.Count(name, value, tags)
	}
}

// opTags returns base tags plus op and status.
func (c *Client) opTags(op, status string) []string {
	return []string{
		"tenant:" + c.config.Tenant,
		"store:" + c.config.Store,
		"op:" + op,
		"status:" + status,
	}
}

// baseTags returns tenant + store tags.
func (c *Client) baseTags() []string {
	return []string{
		"tenant:" + c.config.Tenant,
		"store:" + c.config.Store,
	}
}
