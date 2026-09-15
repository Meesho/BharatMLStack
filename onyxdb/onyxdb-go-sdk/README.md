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
| Topology | etcd watcher on **two** keys: `activeVersion` (promote/rollback) and the pod-registration prefix (scale-up / warm reports). Each event re-reads `VersionMeta` and atomically swaps `shardCount` + assignment. **etcd is off the request hot path** — only the watcher touches it, on topology events |
| Pod discovery fallback | Kubernetes headless-service DNS, per shard: `{tenant}-{store}-shard-{N}.{namespace}.svc.{dnsZone}`. Used **only** for a shard the assignment map does not cover; the refresh loop skips the lookup entirely for covered shards, since a store addressed by assignment may have no Service at all |
| DNS refresh | background ticker (`DNSRefreshInterval`, default 30s); a transient lookup failure keeps last-known addrs. Tenant/store are sanitized (`lower`, `_`→`-`, trunc 63) to match the name the data-plane chart renders — see the constraint in [ADR-0007](../docs/adr/0007-headless-service-dns-pod-discovery.md) |
| Pod selection | round-robin (lock-free atomic counter per shard); a broken pod is marked unhealthy locally and skipped until the next refresh clears the mark |
| Connection warm-up | pods newly added to the assignment get `MinConnsPerPod` connections pre-dialed in the background (disable via `warmUpOnTopologyChange: false` in `clientConfig`) |
| Pool pruning | after each topology change and each DNS refresh, pools for pods no longer in the assignment (scaled-down / no-longer-warm) are closed |
| Idle eviction | a background sweep closes connections idle beyond `IdleTimeout`, keeping at least `MinPerPod` per pod |
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

### etcd keys

All under `/config/mnemo/tenants/{tenant}/stores/{store}` except pod
registrations, which are ephemeral and lease-bound:

| Key | Access |
|---|---|
| `…/activeVersion` | watched; the control plane CAS-flips it on promote/rollback |
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
| `onyxdb.pool.get` | `tenant`, `store`, `result` (`hit`/`dial`/`error`) |
| `onyxdb.pool.dial.latency`, `onyxdb.pool.idle_evicted`, `onyxdb.pool.overflow` | `tenant`, `store` |
| `onyxdb.topology.reload` | `tenant`, `store`, `status` (`ok`/`error`) |

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
