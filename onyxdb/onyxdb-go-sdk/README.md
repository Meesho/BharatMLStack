# OnyxDB Go Client SDK

The Go client for OnyxDB WORM stores. Exposes only `Get` / `BatchGet` — it hides
sharding, the binary TCP protocol, connection pooling, and version flips.

## Usage

```go
client, err := sdk.NewClient(sdk.Config{
    EtcdEndpoints: []string{"localhost:2379"},
    Tenant:        "recsys",
    Store:         "catalog",
})
if err != nil { /* ... */ }
defer client.Close()

// Single lookup
val, err := client.Get(ctx, key)        // key is the 12-byte wire key
if errors.Is(err, sdk.ErrKeyNotFound) { /* miss */ }

// Batch (scatter-gather, results in input order)
results, err := client.BatchGet(ctx, keys)
for _, r := range results {
    // r.Key, r.Value (nil on miss), r.Err (per-key)
}
```

For local development without etcd:

```go
client := sdk.NewDirectClient("127.0.0.1:9091", 4)
defer client.Close()
```

## How it works

| Concern | Mechanism |
|---|---|
| Shard routing | `crc32(key) % shardCount` (IEEE — matches producer & read server) |
| Pod discovery | **Kubernetes headless-service DNS**. Each shard resolves `{tenant}-{store}-shard-{N}.{namespace}.svc.{dnsZone}` → ready pod IPs. CoreDNS only returns pods passing the read server's `/healthz?check=warm` readiness probe, so the warm-ring filter lives in K8s |
| Topology | etcd watcher on `activeVersion`; on each promote/rollback it reads `shardCount` and triggers a DNS re-resolve. **etcd carries no pod IPs and is off the read hot path** |
| DNS refresh | background ticker (`DNSRefreshInterval`, default 30s) re-resolves all shards; a transient lookup failure keeps last-known addrs |
| Pod selection | round-robin across resolved pods; a broken pod is marked unhealthy locally and skipped until the next refresh clears the mark |
| Pool pruning | after each refresh, connection pools for pods that dropped out of DNS (scaled-down / no-longer-warm) are closed |
| BatchGet | group keys by shard → parallel one-batch-per-shard fan-out → merge in input order; per-shard partial failure |

## Config

| Field | Default | Meaning |
|---|---|---|
| `EtcdEndpoints` | — (required) | etcd endpoints for the `activeVersion` watch |
| `Tenant` / `Store` | — | which store to serve |
| `Namespace` | `default` | K8s namespace of the shard headless services |
| `DNSZone` | `cluster.local` | cluster DNS zone |
| `Port` | 9091 | read server TCP port |
| `DNSRefreshInterval` | 30s | background DNS re-resolve cadence |
| `ConnsPerPod` | 4 | pooled TCP connections per pod |
| `TimeoutMs` | 100 | per-request deadline |

## Kubernetes requirements

For DNS discovery to work, each shard needs a **headless Service** (Phase 8 Helm):

- `clusterIP: None`, selector matching that shard's StatefulSet pods, named `{tenant}-{store}-shard-{N}`.
- Read server **readiness probe** = `GET /healthz?check=warm` on port 9100 — a pod enters the Service's endpoints (and thus DNS) only when it has an active warm version.
- `publishNotReadyAddresses: false` (default) so not-warm pods never appear in DNS.

## Test

```bash
go test -race -coverprofile=coverage.out ./...   # 100% coverage
```
