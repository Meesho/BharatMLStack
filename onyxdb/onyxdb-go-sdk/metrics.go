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
	// Tags: tenant, store, result (hit/dial/error/dial_wait_expired, and
	// dial_backoff / dial_ratelimited for a dial refused at once because the
	// pod's last dial failed / this client's dial rate to it is used up)
	MetricPoolGet = "onyxdb.pool.get"
	// Tags: tenant, store
	MetricPoolDialLatency = "onyxdb.pool.dial.latency"
	MetricPoolIdleEvicted = "onyxdb.pool.idle_evicted"
	MetricPoolOverflow    = "onyxdb.pool.overflow"
	// A timed-out connection's late response was drained and the connection
	// went back to the pool (result:reused), or it was closed (result:closed).
	// Tags: tenant, store, result
	MetricPoolDrain = "onyxdb.pool.drain"
	// A pod that drops out of the assignment keeps its pooled connections for
	// PoolConfig.PruneGrace. Counts connections: kept because the pod came back
	// within it (result:reused), closed then because the peer had closed them
	// (result:dead), or closed when the grace ran out (result:expired).
	// Tags: tenant, store, result
	MetricPoolPruneGrace = "onyxdb.pool.prune_grace"

	// Topology watcher metrics.
	// Tags: tenant, store, status (ok/error)
	MetricTopologyReload = "onyxdb.topology.reload"
	// An etcd watch ended while the client was live and was re-established.
	// reason: closed (the channel closed), canceled (etcd cancelled the watch),
	// compacted (etcd cancelled it because the revision it resumed from had
	// been compacted — the usual cause after a reconnect).
	// Tags: tenant, store, reason
	MetricTopologyWatchRestart = "onyxdb.topology.watch_restart"

	// Picks that landed on a pod still in its slow start (one per shard
	// request routed there): the ramp as the client sees it.
	// Tags: tenant, store
	MetricSlowStartPick = "onyxdb.router.slow_start_pick"

	// Shard fail-fast (docs/design/onyxdb-readserver-admission-and-sdk-redial.md
	// §3 B3). A shard request failed with ErrShardUnavailable because every pod
	// of the shard is inside its unhealthy window. No pod or shard tag.
	// Tags: tenant, store
	MetricRouterShardUnavailable = "onyxdb.router.shard_unavailable"
	// The outcome of a probe: the one request a shard sends to a pod whose
	// window has passed, to find out whether it is back (result:ok heals it,
	// result:fail doubles its window).
	// Tags: tenant, store, result (ok/fail)
	MetricRouterProbe = "onyxdb.router.probe"
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
