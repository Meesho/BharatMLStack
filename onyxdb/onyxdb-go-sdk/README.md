# OnyxDB Go Client SDK

The Go client for OnyxDB WORM stores. Exposes only point and batch reads — it
hides sharding, the binary TCP protocol, connection pooling, and version flips.

Two key flavours, four methods:

| Key type | Single | Batch | Opcodes |
|---|---|---|---|
| Variable-length UTF-8 (string) | `StringGet` | `StringBatchGet` | `0x03` / `0x04` |
| Fixed 12-byte wire key | `Get` | `BatchGet` | `0x01` / `0x02` |

Writes are out of band (SST → GCS → dataloader), so the SDK is read-only by
construction.

## Usage

```go
client, err := sdk.NewClient(sdk.Config{
    EtcdEndpoints: []string{"localhost:2379"},
    Tenant:        "recsys",
    Store:         "catalog",
})
if err != nil { /* ... */ }
defer client.Close()

// Single lookup — string key, e.g. "catalog__user_geohash_1_3:105959719|4236"
key := sdk.BuildStringKey("catalog__user_geohash_1_3", 105959719, 4236)
val, err := client.StringGet(ctx, key)
if errors.Is(err, sdk.ErrKeyNotFound) { /* miss */ }

// Batch (scatter-gather, results in input order)
results, err := client.StringBatchGet(ctx, keys)
for _, r := range results {
    // r.Key, r.Value (nil on miss), r.Err (per-key)
}
```

`Get` / `BatchGet` are the same calls over fixed 12-byte keys. `BuildStringKey`
emits `"<entityLabel>:<pk1>|<pk2>|..."`; callers that key per feature group
(OFS does) build the key themselves — the SDK never rewrites a key it is given.

The golden vectors in `key_test_vectors.json` are shared with the Python
producer and the Rust read server, so key bytes and shard assignment are
verified identical across all three.

For local development without etcd:

```go
client := sdk.NewDirectClient("127.0.0.1:9091", 4)
defer client.Close()
```

## How it works

| Concern | Mechanism |
|---|---|
| Shard routing | `crc32(key) % shardCount` (IEEE — matches producer & read server) |
| Pod discovery | **Control-plane assignment map** ([ADR-0007](../docs/adr/0007-headless-service-dns-pod-discovery.md)). The control plane derives `shard → ["podIP:port", ...]` from pod registrations that report the active version warm, and publishes it as `VersionMeta.assignment`. The SDK serves it from an in-memory snapshot — works identically on K8s and VM, and does not depend on any K8s object name |
| Topology | etcd watcher on **three** keys: `activeVersion` (promote/rollback), the pod-registration prefix (scale-up / warm reports) and `topologyVersion` (bumped with every assignment write, including the control plane's refresh that adds a pod warmed after promote). Each event re-reads `VersionMeta` and atomically swaps `shardCount` + assignment. **etcd is off the request hot path** — only the watcher touches it, on topology events |
| Watch restart | etcd cancels a watch that resumes from a compacted revision, and clientv3 then closes its channel. The watcher re-establishes all three watches: it re-reads the active version and its meta, then watches from the revision of that read, so nothing written in between is missed. Attempts back off from 500 ms, doubling to 30 s, with jitter; a watch that stayed up 30 s starts the backoff over |
| Pod discovery fallback | Kubernetes headless-service DNS, per shard: `{tenant}-{store}-shard-{N}.{namespace}.svc.{dnsZone}`. Used **only** for a shard the assignment map does not cover; the refresh loop skips the lookup entirely for covered shards, since a store addressed by assignment may have no Service at all |
| DNS refresh | background ticker (`DNSRefreshInterval`, default 30s); a transient lookup failure keeps last-known addrs. Tenant/store are sanitized (`lower`, `_`→`-`, trunc 63) to match the name the data-plane chart renders — see the constraint in [ADR-0007](../docs/adr/0007-headless-service-dns-pod-discovery.md) |
| Pod selection | round-robin (lock-free atomic counter per shard). A pod that failed a request is marked unhealthy locally and skipped for a backoff window: 1s after the first failure in a row, doubling to 30s, ±20% jitter (`Config.UnhealthyBackoffBase` / `UnhealthyBackoffMax`). The first request through it that succeeds (a miss counts) clears the mark. A refresh drops only expired marks and marks of pods no longer assigned; active windows survive it |
| Connection warm-up | pods newly added to the assignment get `MinConnsPerPod` connections pre-dialed in the background (disable via `warmUpOnTopologyChange: false` in `clientConfig`) |
| Pool pruning | after each topology change and each DNS refresh, pools for pods no longer in the assignment (scaled-down / no-longer-warm) are closed |
| Idle eviction | a background sweep closes connections idle beyond `IdleTimeout`, keeping at least `MinPerPod` per pod |
| Timed-out connections | a request that hits its deadline after it was fully written leaves its response on the wire; the connection is not closed. `ConnPool.Recycle` reads the rest of that response in the background, within `PoolConfig.DrainTimeout` (default 1s; negative disables), and returns the connection to the pool. EOF, a reset, a partial write or a drain that runs out of time still closes it |
| Bounded dials | at most `PoolConfig.MaxDialsPerPod` (default 4) dials in flight per pod. A request that finds the pool empty with every slot busy waits for a returned connection or a free slot, up to its own deadline, instead of dialling again |
| Prune grace | a pod that leaves the assignment keeps its pooled connections for `PoolConfig.PruneGrace` (default 15s; negative closes them at once). A pod that comes back within it reuses the connections its peer still holds, and warm-up dials only the shortfall, instead of every client redialling it at once; the idle sweep closes them when the grace runs out |
| Dial rate limit | at most `PoolConfig.DialRatePerPod` dials per second per pod from one client (default 2, burst `MaxDialsPerPod`; negative turns it off). A request that finds no idle connection and no token never dials: it waits for a connection that is out or being dialled, up to its deadline, or, if there is none, fails at once with `ErrDialBackoff` (`onyxdb.pool.get{result:dial_ratelimited}`). Background warm-up is exempt from the rate, not from dial backoff |
| Dial backoff | after a failed dial, no dial to that pod for `PoolConfig.DialBackoffBase` (default 100ms; negative turns it off), doubling per failure in a row up to `DialBackoffMax` (5s), ±20% jitter; the first successful dial resets it. Meanwhile a request that finds no idle connection fails at once with `ErrDialBackoff` (`result:dial_backoff`). Together with the rate limit this stops every client redialling a stalled pod at once |
| Shard fail-fast | `Config.ShardFailFast` (default on). While every pod of a shard is inside its unhealthy window, the shard's keys fail at once with `ErrShardUnavailable` and nothing is sent, instead of waiting out the timeout on a stalled pod. A pod past its window first takes one probe (one ordinary request; at most one per shard per client at a time): success heals it, failure doubles its window. Off keeps the old best-effort routing |
| Batch reads | group keys by shard → parallel one-batch-per-shard fan-out → merge in input order. Partial failure is per-shard: a failed shard sets `Result.Err` on only its own keys; the top-level error is the first shard error |

## Config

Caller-supplied `sdk.Config`:

| Field | Default | Meaning |
|---|---|---|
| `EtcdEndpoints` | — (required) | etcd endpoints for topology discovery; `ErrNoEndpoints` if empty |
| `Tenant` / `Store` | — | which store to serve |
| `Namespace` | `default` | K8s namespace, for the DNS fallback only |
| `DNSZone` | `cluster.local` | cluster DNS zone, for the DNS fallback only |
| `Port` | 9091 | read server TCP port |
| `DNSRefreshInterval` | 30s | background re-resolve / pool-prune cadence |
| `ConnsPerPod` | 4 | pool ceiling per pod — legacy alias for `PoolConfig.MaxPerPod` |
| `TimeoutMs` | 100 | per-request deadline — see the note below |
| `Pool` | nil | full `PoolConfig`; when set, `ConnsPerPod` / `TimeoutMs` are ignored |
| `ClientConfig` | nil | supply explicitly to skip the control-plane fetch |
| `UnhealthyBackoffBase` / `UnhealthyBackoffMax` | 1s / 30s | how long a pod that failed a request is skipped, doubling per failure in a row |
| `ShardFailFast` | nil (= true) | `*bool`; `false` turns shard fail-fast off |
| `Timing` / `Count` | nil | metric callbacks (nil-safe) — see [Metrics](#metrics) |

> **`TimeoutMs` bounds the whole call, not one round trip.** For
> `BatchGet` / `StringBatchGet` it bounds the entire scatter-gather, so size it
> for the largest batch you will send, not for a single shard's latency.

### Pool tunables come from the control plane

At init the SDK best-effort reads the store's `clientConfig` from etcd and
overlays it onto the pool config — a store's tunables are an operator concern,
not a per-caller one. A missing key means all defaults; a fetch failure is not
fatal.

| `clientConfig` field | Effect |
|---|---|
| `requestTimeoutMs` | overrides `Config.TimeoutMs` |
| `connectTimeoutMs` | `PoolConfig.DialTimeout` (default 5s) |
| `minConnsPerPod` / `maxConnsPerPod` | pool floor (default 1) / ceiling (default 4) |
| `keepAliveIntervalMs` / `keepAliveTimeoutMs` | TCP keepalive (defaults 15s / 5s) |
| `idleTimeoutMs` / `idleCheckIntervalMs` | idle eviction (defaults 60s / 10s) |
| `dnsRefreshIntervalMs` | overrides `Config.DNSRefreshInterval` |
| `warmUpOnTopologyChange` | `false` disables connection warm-up |
| `dialRatePerPod` | `PoolConfig.DialRatePerPod` (default 2; negative turns the rate limit off) |
| `dialBackoffBaseMs` / `dialBackoffMaxMs` | `PoolConfig.DialBackoffBase` / `DialBackoffMax` (defaults 100 / 5000; a negative base turns dial backoff off) |
| `unhealthyBackoffBaseMs` / `unhealthyBackoffMaxMs` | `Config.UnhealthyBackoffBase` / `UnhealthyBackoffMax` (defaults 1000 / 30000) |
| `shardFailFast` | `Config.ShardFailFast` (default true when absent) |

The last four rows are decoded by the SDK itself from the same document
(`client.go` `clientConfigExt`), so the SDK still builds against the released
control plane module. That module's `ClientConfig` does not carry them yet, so
its clientConfig API drops them: write them to etcd directly. A key of the
wrong type drops only these keys, with one warning; `Config.Pool` set turns them
off along with every other overlay. A control plane module whose `ClientConfig`
does carry them is handled too: the SDK removes them before its
`model.ClientConfig` decode (`withoutRedialKeys`), so a wrong-typed one never
costs the rest of the document.

### etcd keys

All under `/config/mnemo/tenants/{tenant}/stores/{store}` except pod
registrations, which are ephemeral and lease-bound:

| Key | Access |
|---|---|
| `…/activeVersion` | watched; the control plane CAS-flips it on promote/rollback |
| `…/topologyVersion` | watched; the control plane bumps it in the same transaction as every assignment write (promote, rollback, refresh). A bump re-reads `activeVersion` and its assignment |
| `…/versions/{versionID}` | read on each event → `shardCount` + `assignment` |
| `…/clientConfig` | read once at init |
| `/config/mnemo-cluster-manager/{tenant}/{store}/` | watched with prefix; a registration change re-reads the active version's assignment |

## Metrics

`Config.Timing` and `Config.Count` are optional callbacks — the SDK never opens
a socket of its own. Leave them nil and all SDK-level observability is silently
dropped, including pool-dial pressure and topology-reload failures.

| Metric | Tags |
|---|---|
| `onyxdb.request.latency` / `onyxdb.request.count` | `tenant`, `store`, `op` (`single`/`batch`/`string_single`/`string_batch`), `status` (`hit`/`miss`/`error`/`ok`) |
| `onyxdb.batch.keys` | `tenant`, `store` |
| `onyxdb.pool.get` | `tenant`, `store`, `result` (`hit`/`dial`/`error`/`dial_wait_expired`/`dial_backoff`/`dial_ratelimited`): `dial_backoff` and `dial_ratelimited` are checkouts refused at once because the pod's last dial failed or this client's dial rate to it is used up |
| `onyxdb.pool.drain` | `tenant`, `store`, `result` (`reused`/`closed`): outcome of draining a timed-out connection |
| `onyxdb.pool.prune_grace` | `tenant`, `store`, `result` (`reused`/`dead`/`expired`): connections of a pod that left the assignment, kept on its return, closed then because the peer had closed them, or closed when the grace ran out |
| `onyxdb.pool.dial.latency`, `onyxdb.pool.idle_evicted`, `onyxdb.pool.overflow` | `tenant`, `store` |
| `onyxdb.topology.reload` | `tenant`, `store`, `status` (`ok`/`error`) |
| `onyxdb.topology.watch_restart` | `tenant`, `store`, `reason` (`closed`/`canceled`/`compacted`): an etcd watch re-established after it ended |
| `onyxdb.router.shard_unavailable` | `tenant`, `store`: a shard request failed with `ErrShardUnavailable` (no pod or shard tag) |
| `onyxdb.router.probe` | `tenant`, `store`, `result` (`ok`/`fail`): the outcome of each probe to a pod past its unhealthy window |

## Deployment requirements

**The assignment map carries pod IPs, so the caller must have a network route
to the data plane's pod network.** This is an operational precondition the SDK
cannot work around: a caller in a different cluster without pod-CIDR
routability can reach the pods by neither path — DNS cannot resolve another
cluster's `svc.cluster.local` either. See [ADR-0007](../docs/adr/0007-headless-service-dns-pod-discovery.md) *Current Assessment*.

For the assignment path:

- Pods must register in etcd with their pod IP + port and report the active
  version warm, or the control plane leaves the shard uncovered and reads
  return `ErrNoHealthyPod`.
- A version must be **promoted**. With nothing active the SDK has no pods to
  route to and every op errors.

For the DNS fallback only — each shard needs a headless Service:

- `clusterIP: None`, selector matching that shard's pods, named
  `{tenant}-{store}-shard-{N}` (sanitized: lowercase, `_`→`-`, truncated to 63).
- Read server readiness probe `GET /healthz?check=warm` on port 9100, so a pod
  enters DNS only once it has an active warm version.
- `publishNotReadyAddresses: false` (default) so not-warm pods never appear.

## Test

```bash
GOWORK=off go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
```
