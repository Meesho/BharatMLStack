# mNemo: WORM Database Platform — Implementation Plan

> **Purpose:** This file is the single source of truth for implementing mNemo inside BharatMLStack.
> Read this file at the start of every session before writing any code.

---

## 0. Context & Design Decisions

### Problem
Multi-tenant WORM (Write-Once-Read-Many) database platform serving immutable datasets (10 TB+, ~10B rows) at >2M RPS with sub-2ms p99, atomic version flips, and sub-second rollback.

### Chosen Approach: Approach C + S1 Topology
- **Approach C** — Deployable-per-shard versioned store (Venice + kvrocks inspiration)
- **S1 Topology** — Each shard = independent K8s StatefulSet, KEDA-managed
- **Protocol** — Raw TCP binary (1.6× over gRPC at 2M RPS scale)
- **Storage** — RocksDB read-only mode, ZSTD compression (9.5× ratio), shared block cache
- **State** — etcd with CAS-guarded topology version
- **Sharding** — `crc32(entityKey|pk) % S` (IEEE polynomial, deterministic across Python/Go/Rust)

### Key References
- HLD: https://meesho.atlassian.net/wiki/spaces/EW/pages/5382701059
- LLD: https://meesho.atlassian.net/wiki/spaces/EW/pages/5387026444
- Local prototype: ~/dev/mnemo (baseline for Go/Rust code)

---

## 1. Repository Conventions (MUST follow)

| Convention | Rule |
|---|---|
| Module paths | `github.com/Meesho/BharatMLStack/mnemo/<component>` |
| Go version | `1.24.4` (matching horizon, online-feature-store) |
| Commits | Conventional Commits: `feat(mnemo):`, `fix(mnemo-sdk):`, `test(mnemo-cp):` |
| Branch naming | `feat/mNemo-<component>-description` |
| Code style (Go) | `gofmt`, `go vet`, `staticcheck` |
| Code style (Rust) | `cargo fmt`, `cargo clippy --deny warnings`, `cargo audit` |
| Code style (Python) | `black` (line-length 100), `ruff`, `pytest --cov-fail-under=100` |
| Docker | Multi-stage; Rust → `distroless/cc:nonroot`; Go → `distroless/static:nonroot` |
| Multi-arch | `linux/amd64,linux/arm64` via `docker buildx` |
| Images | `ghcr.io/$owner/mnemo-controlplane`, `ghcr.io/$owner/mnemo-readserver` |
| Helm charts | `helm-charts/mnemo-controlplane/`, `helm-charts/mnemo-readserver/` |
| Docs | `docs-src/docs/mnemo/v1.0.0/*.md` (Docusaurus, follows OFS pattern) |
| CI | `paths: ['mnemo/**']`, separate job per component |
| Release | `workflow_dispatch`, `release-mnemo-controlplane.yml`, `release-mnemo-readserver.yml` |
| Test coverage | **100% for all components** — CI enforces with `go tool cover` / `cargo tarpaulin` |
| `VERSION` file | At `mnemo/VERSION` (shared semver for the platform) |

---

## 2. Directory Structure (Final)

```
BharatMLStack/
├── mnemo/
│   ├── PLAN.md                          ← this file
│   ├── README.md
│   ├── VERSION                          ← v0.1.0
│   ├── go.work                          ← Go workspace (grows with each phase)
│   ├── go.work.sum
│   ├── Makefile                         ← build/test/lint/coverage/dev targets
│   │
│   ├── schema/                          ← [PHASE 1] Shared Go module — types + etcd keys
│   │   ├── go.mod
│   │   ├── types.go                     ← StoreConfig, VersionMeta, PodData
│   │   ├── types_test.go
│   │   ├── keys.go                      ← VersionStatus + 10 etcd key builders
│   │   └── keys_test.go
│   │
│   ├── controlplane/                    ← [PHASE 2] Go service — REST API + etcd state machine
│   │   ├── go.mod
│   │   ├── VERSION
│   │   ├── Dockerfile
│   │   ├── release.sh
│   │   ├── env.example
│   │   ├── README.md
│   │   ├── cmd/controlplane/main.go
│   │   └── internal/
│   │       ├── server/
│   │       │   ├── server.go            ← Gin router, graceful shutdown
│   │       │   ├── server_test.go
│   │       │   └── handlers/
│   │       │       ├── store.go         ← POST /stores, GET /stores/:store
│   │       │       ├── store_test.go
│   │       │       ├── version.go       ← publish, promote, rollback, retire
│   │       │       ├── version_test.go
│   │       │       ├── topology.go      ← GET /topology
│   │       │       └── topology_test.go
│   │       ├── etcdstate/
│   │       │   ├── client.go            ← EtcdStateClient interface + real impl
│   │       │   ├── client_test.go       ← mock-based unit tests
│   │       │   ├── store.go             ← CreateStore, GetStore, SetActiveVersion (CAS)
│   │       │   └── store_test.go
│   │       ├── coverage/
│   │       │   ├── checker.go           ← watches pod keys, computes coverage_ratio
│   │       │   └── checker_test.go
│   │       ├── placement/
│   │       │   ├── placement.go         ← shard → pod assignment
│   │       │   └── placement_test.go
│   │       └── sizing/
│   │           ├── sizing.go            ← {D,RPS,p99} → {S,R,V,pod-spec,cost}
│   │           └── sizing_test.go
│   │
│   ├── dataplane/
│   │   ├── dataloader/                  ← [PHASE 4] Go sidecar
│   │   │   ├── go.mod
│   │   │   ├── Dockerfile
│   │   │   ├── cmd/dataloader/main.go
│   │   │   └── internal/
│   │   │       ├── fetcher/
│   │   │       │   ├── fetcher.go       ← GCS multipart (8 parts/file, 10 goroutines/shard)
│   │   │       │   ├── fetcher_test.go  ← mock GCSClient interface
│   │   │       │   ├── manifest.go      ← _manifest.json parse + SHA256 verify
│   │   │       │   └── manifest_test.go
│   │   │       ├── ingest/
│   │   │       │   ├── client.go        ← IPC JSON-newline client (load/activate/drop)
│   │   │       │   └── client_test.go   ← net.Pipe() UDS mock
│   │   │       ├── watcher/
│   │   │       │   ├── watcher.go       ← etcd watch: version status transitions
│   │   │       │   └── watcher_test.go  ← mock watch channel
│   │   │       └── lifecycle/
│   │   │           ├── orchestrator.go  ← watch → fetch → load → activate → report
│   │   │           └── orchestrator_test.go
│   │   │
│   │   └── readserver/                  ← [PHASE 3] Rust service — TCP + RocksDB + ArcSwap
│   │       ├── Cargo.toml
│   │       ├── Cargo.lock
│   │       ├── Dockerfile
│   │       └── src/
│   │           ├── main.rs              ← tokio::select!(tcp + http + ipc)
│   │           ├── config.rs            ← clap CLI flags
│   │           ├── engine/
│   │           │   ├── mod.rs           ← Engine trait (pluggable)
│   │           │   ├── rocksdb.rs       ← RocksDB: read-only, shared LRU, ZSTD, bloom
│   │           │   └── rocksdb_test.rs
│   │           ├── tcp/
│   │           │   ├── mod.rs
│   │           │   ├── server.rs        ← TcpListener accept loop
│   │           │   ├── handler.rs       ← opcode dispatch, framing
│   │           │   ├── handler_test.rs
│   │           │   └── protocol.rs      ← encode/decode single(0x01) + batch(0x02)
│   │           │   └── protocol_test.rs ← roundtrip, N=0, N=10000, malformed
│   │           ├── ipc/
│   │           │   ├── mod.rs
│   │           │   ├── unix_socket.rs   ← UDS listener: load/activate/drop
│   │           │   └── unix_socket_test.rs
│   │           ├── version/
│   │           │   ├── mod.rs
│   │           │   ├── manager.rs       ← staging map + ArcSwap active view
│   │           │   └── manager_test.rs  ← concurrent load+activate correctness
│   │           ├── health.rs            ← /healthz?check=warm, /metrics
│   │           └── metrics.rs           ← Prometheus counter/histogram definitions
│   │
│   ├── sdk/                             ← [PHASE 5] Go client SDK
│   │   ├── go.mod
│   │   ├── client.go                    ← Client interface, NewClient, NewDirectClient
│   │   ├── client_test.go
│   │   ├── router.go                    ← crc32(key)%S → shard; PodFor(shard)
│   │   ├── router_test.go               ← cross-language golden values
│   │   ├── scatter_gather.go            ← GroupByShards, fan-out, merge in order
│   │   ├── scatter_gather_test.go
│   │   ├── tcp.go                       ← ConnPool: per-shard, DNS resolve, RR, TTL
│   │   ├── tcp_test.go                  ← net.Pipe() connections
│   │   ├── topology.go                  ← etcd watch: activeVersion changes
│   │   └── topology_test.go
│   │
│   ├── py-producer/                     ← [PHASE 6] Python package — PySpark SST pipeline
│   │   ├── pyproject.toml               ← black, ruff, pytest-cov, Python 3.10+
│   │   ├── README.md
│   │   ├── mnemo_producer/
│   │   │   ├── __init__.py
│   │   │   ├── shard.py                 ← crc32(entityKey|pk)%S (IEEE polynomial)
│   │   │   ├── sst_writer.py            ← RocksDB SstFileWriter: ZSTD L3, bloom, 16KB
│   │   │   ├── qc.py                    ← skew(30%), schema, null-ratio, SHA256
│   │   │   ├── manifest.py              ← _manifest.json builder
│   │   │   ├── uploader.py              ← GCS per-shard upload (service account auth)
│   │   │   ├── publisher.py             ← POST /versions/{vId}/publish
│   │   │   └── pipeline.py              ← PySpark: Source→Shard→SST→QC→Upload→Publish
│   │   └── tests/
│   │       ├── test_shard.py            ← CRC32 cross-language determinism (Go golden)
│   │       ├── test_qc.py               ← skew/null/schema edge cases
│   │       ├── test_manifest.py
│   │       └── test_pipeline.py         ← mock Spark + GCS + control plane
│   │
│   ├── tools/
│   │   ├── gen-sst/                     ← [PHASE 7] Rust: generate test SST files
│   │   │   ├── Cargo.toml
│   │   │   └── src/main.rs
│   │   └── smoke-test/                  ← [PHASE 7] Go: E2E gen-sst→readserver→SDK
│   │       ├── go.mod
│   │       └── main.go
│   │
│   └── quick-start/                     ← [PHASE 0] Local dev
│       ├── docker-compose.yml           ← etcd v3.5.12
│       ├── start.sh
│       ├── stop.sh
│       └── restart.sh
│
├── helm-charts/
│   ├── mnemo-controlplane/              ← [PHASE 8] Deployment (Go control plane)
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       └── service.yaml
│   └── mnemo-readserver/                ← [PHASE 8] Per-shard StatefulSet (Rust+Go sidecar)
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│           ├── statefulset.yaml         ← readserver + dataloader containers
│           ├── service-headless.yaml    ← headless for pod DNS
│           └── keda-scaled-object.yaml
│
├── .github/workflows/
│   ├── mnemo.yml                        ← [PHASE 0] CI: per-component jobs
│   ├── release-mnemo-controlplane.yml   ← [PHASE 0] Release workflow
│   └── release-mnemo-readserver.yml     ← [PHASE 0] Release workflow
│
└── docs-src/docs/mnemo/
    ├── _category_.json                  ← [PHASE 9] "mNemo WORM Database"
    └── v1.0.0/
        ├── index.md
        ├── architecture.md
        ├── producer.md
        ├── control-plane.md
        ├── data-plane.md
        ├── sdk.md
        ├── observability.md
        └── operations.md
```

---

## 3. Data Model Reference (DO NOT CHANGE without LLD sign-off)

### etcd Key Space
```
/config/mnemo/tenants/{tenant}/stores/{store}/
    entityKey          → string
    shardCount         → int (S)
    activeVersion      → string (vId)
    rollbackVersion    → string (vId, V=2 only)
    topologyVersion    → int (monotonic, CAS-guarded)
    versions/{vId}     → VersionMeta JSON

/config/mnemo-cluster-manager/{tenant}/{store}/{podID}
    → PodData JSON (ephemeral, lease 5s + keepalive)
```

### TCP Wire Protocol (port 9091)
```
Single (opcode 0x01):
  REQ:  [1B op=0x01][8B key_a BE][4B key_b BE]      = 13 bytes
  RESP: [1B found][if found: 4B len BE][value bytes]

Batch (opcode 0x02):
  REQ:  [1B op=0x02][2B N BE][12B × N]               = 3 + 12N bytes
  RESP: [2B N BE][per-key: [1B found][4B len][value]]

Constants: KEY_SIZE=12, MAX_BATCH=10000, TCP_NODELAY=true
```

### IPC Protocol (Unix socket /tmp/mnemo-readserver.sock)
```json
→ {"cmd":"load",    "version":"20260528_001", "shards":{"0":"/data/store/20260528_001/shard_0"}}
→ {"cmd":"activate","version":"20260528_001"}
→ {"cmd":"drop",    "version":"20260527_003"}
← {"status":"ok"} | {"status":"error","error":"..."}
```

### REST API (Control Plane, port 8080)
```
POST   /api/v1/tenants/:tenant/stores
GET    /api/v1/tenants/:tenant/stores/:store
POST   /api/v1/tenants/:tenant/stores/:store/versions/:vId/publish
POST   /api/v1/tenants/:tenant/stores/:store/versions/:vId/promote
POST   /api/v1/tenants/:tenant/stores/:store/rollback
POST   /api/v1/tenants/:tenant/stores/:store/versions/:vId/retire
GET    /api/v1/tenants/:tenant/stores/:store/topology
GET    /api/v1/health
```

### Sharding Formula (cross-language deterministic)
```
shard = crc32_ieee(entityKey + "|" + pk) % S

Python: binascii.crc32(key.encode()) & 0xFFFFFFFF
Go:     hash/crc32.ChecksumIEEE(key)
Rust:   crc32fast::hash(key)
```

### GCS Layout
```
gs://mnemo-data/{tenant}/{store}/{vId}/
    shard_{i}/part-{n}.sst
    _manifest.json  → {date, run, shardCount, shards:[{id,files:[{name,rows,bytes,sha256}]}]}
```

---

## 4. Implementation Phases

### PHASE 0 — Scaffolding ✅ IN PROGRESS
**Goal:** Repo structure, CI/CD, local dev, build tooling.

| File | Status |
|---|---|
| `mnemo/VERSION` | ✅ created |
| `mnemo/go.work` | ✅ created (schema only; grow per phase) |
| `mnemo/PLAN.md` | ✅ this file |
| `mnemo/Makefile` | ⬜ |
| `mnemo/README.md` | ⬜ |
| `mnemo/quick-start/docker-compose.yml` | ⬜ |
| `mnemo/quick-start/start.sh` | ⬜ |
| `mnemo/quick-start/stop.sh` | ⬜ |
| `mnemo/quick-start/restart.sh` | ⬜ |
| `.github/workflows/mnemo.yml` | ⬜ |
| `.github/workflows/release-mnemo-controlplane.yml` | ⬜ |
| `.github/workflows/release-mnemo-readserver.yml` | ⬜ |

### PHASE 1 — Schema Module ✅ COMPLETE
**Goal:** Shared types + etcd key builders, 100% test coverage.
**Module:** `github.com/Meesho/BharatMLStack/mnemo/schema`

| File | Status |
|---|---|
| `mnemo/schema/go.mod` | ✅ |
| `mnemo/schema/types.go` | ✅ |
| `mnemo/schema/types_test.go` | ✅ |
| `mnemo/schema/keys.go` | ✅ |
| `mnemo/schema/keys_test.go` | ✅ (all 10 functions covered) |

**Coverage:** `go test -coverprofile=coverage.out ./...` → `total: 100.0%`

### PHASE 2 — Control Plane
**Goal:** Full REST API with CAS-guarded etcd state machine, 100% test coverage.
**Module:** `github.com/Meesho/BharatMLStack/mnemo/controlplane`
**Dependencies:** schema, gin, etcd client v3, testify/mock

Key testing decisions:
- Define `EtcdStateClient` interface → handlers use mock in tests, real impl in cmd/
- `testcontainers-go` for integration tests with real etcd
- `sizing.go` is pure functions → table-driven unit tests

**go.work change:** add `./controlplane`

### PHASE 3 — Read Server (Rust)
**Goal:** TCP binary protocol + RocksDB + ArcSwap, 100% Rust coverage.
**Crate:** `mnemo-readserver`
**Coverage tool:** `cargo tarpaulin --threshold 100`

Key testing decisions:
- `Engine` trait with `MockEngine` for TCP handler tests
- `tempdir()` backed RocksDB for engine tests
- `Arc<Mutex<Vec<Command>>>` capture for IPC tests
- 100 concurrent readers during ArcSwap activate test

### PHASE 4 — Data Loader (Go)
**Goal:** GCS multipart download + lifecycle orchestration, 100% test coverage.
**Module:** `github.com/Meesho/BharatMLStack/mnemo/dataplane/dataloader`
**Dependencies:** schema, GCS client, etcd client v3, testify/mock

Key testing decisions:
- `GCSClient` interface → mock for fetcher tests
- `net.Pipe()` for IPC client tests (no real UDS needed)
- Mock etcd watch channel for watcher tests
- All 5 orchestrator states covered (idle→downloading→loaded→activated→reported)

**go.work change:** add `./dataplane/dataloader`

### PHASE 5 — Client SDK (Go)
**Goal:** CRC32 routing + scatter-gather + connection pool, 100% test coverage.
**Module:** `github.com/Meesho/BharatMLStack/mnemo/sdk`
**Dependencies:** schema, etcd client v3, testify/mock

Key testing decisions:
- Golden CRC32 values from Python + Rust for cross-language determinism test
- `net.Listener` fake server for TCP pool tests
- Scatter-gather: verify original order preserved, test partial shard failure
- topology.go: mock etcd watch, assert route rebuild on activeVersion change

**go.work change:** add `./sdk`

### PHASE 6 — Python Producer
**Goal:** PySpark SST pipeline, 100% pytest coverage.
**Package:** `mnemo_producer`
**GCS Auth:** Service account JSON secret (K8s Secret mounted as env var)
**Env:** Works both standalone and within Databricks/PySpark

Key testing decisions:
- `test_shard.py`: 10K keys, compare against Go CRC32 golden values file
- `test_qc.py`: skew ratio >30% fails, <30% passes; null-ratio, schema checks
- `test_pipeline.py`: mock Spark session + GCS client + control plane HTTP

### PHASE 7 — Tools
**Goal:** Test data generator + E2E smoke test.

- `tools/gen-sst/`: Rust CLI, generates `N` shards × `M` keys/shard → SST files + golden CSV
- `tools/smoke-test/`: Go binary: load golden SST → start readserver → connect via SDK → verify all keys

**go.work change:** add `./tools/smoke-test`

### PHASE 8 — Helm Charts
**Goal:** Production K8s deployment.

`helm-charts/mnemo-controlplane/` — Deployment + ClusterIP Service
`helm-charts/mnemo-readserver/` — StatefulSet (readserver + dataloader containers) + Headless Service + KEDA ScaledObject

**Critical values:**
```yaml
# mnemo-readserver/values.yaml
tenant: ""
store: ""
shardCount: 10
replicationFactor: 2
versionsOnDisk: 2
readserver:
  image: ghcr.io/meesho/mnemo-readserver
  tag: v0.1.0
  blockCacheBytes: 8589934592    # 8 GB
  bloomBits: 10
dataloader:
  image: ghcr.io/meesho/mnemo-dataloader
  tag: v0.1.0
  gcsBucket: mnemo-data
  gcsServiceAccountSecret: mnemo-gcs-sa
keda:
  rpsThreshold: 50000
  cpuThreshold: 70
```

### PHASE 9 — Documentation
**Goal:** Full Docusaurus docs under `docs-src/docs/mnemo/v1.0.0/`.

Follows OFS pattern (`_category_.json` + versioned subdirectory with `index.md`).

| Doc file | Content |
|---|---|
| `index.md` | Overview, problem statement, architecture diagram |
| `architecture.md` | Components, data flow, design decisions |
| `producer.md` | PySpark pipeline, sharding formula, QC gate |
| `control-plane.md` | REST API reference, etcd state machine |
| `data-plane.md` | TCP wire protocol, RocksDB config, IPC commands, disk layout |
| `sdk.md` | Go SDK install, Get/BatchGet usage, config reference |
| `observability.md` | All 25 Prometheus metrics, SLOs, Grafana dashboard spec, alerts |
| `operations.md` | Version lifecycle runbook, rollback procedure, capacity sizing |

---

## 5. Test Coverage Strategy

### Go (all modules)
```bash
go test -race -coverprofile=coverage.out ./...
TOTAL=$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $3}')
[ "$TOTAL" = "100.0%" ] || exit 1
```
- Every exported function has a test
- Every error branch is exercised via mock injection
- Interface-first design: all external dependencies (etcd, GCS, K8s) injectable

### Rust (readserver)
```bash
cargo tarpaulin --out Xml --timeout 120
# CI: fail if line coverage < 100%
```
- `#[cfg(test)]` modules in each source file
- `MockEngine` trait implementation for TCP handler isolation
- Tempdir-backed RocksDB for engine tests (no mock, test the real thing)
- Concurrent correctness: spawn 100 reader tasks during ArcSwap activate

### Python (py-producer)
```bash
pytest tests/ --cov=mnemo_producer --cov-report=term-missing --cov-fail-under=100
```
- `unittest.mock.patch` for GCS client, HTTP requests
- Mock Spark session for pipeline tests
- CRC32 golden file generated by Go test (committed to repo)

---

## 6. Observability Reference

### Prometheus Metrics (all 25)

**Read Server (Rust):**
- `mnemo_lookup_duration_seconds{op=single|batch}` — Histogram
- `mnemo_lookup_total{op,status=hit|miss|error}` — Counter
- `mnemo_batch_size` — Histogram
- `mnemo_active_connections` — Gauge
- `mnemo_version_swaps_total` — Counter
- `mnemo_open_versions` — Gauge
- `mnemo_warm_shards` — Gauge
- `mnemo_block_cache_hit_ratio` — Gauge

**Data Loader (Go):**
- `mnemo_loader_download_bytes_total` — Counter
- `mnemo_loader_download_duration_seconds` — Histogram
- `mnemo_loader_download_throughput_bytes` — Gauge
- `mnemo_loader_ingest_duration_seconds` — Histogram
- `mnemo_loader_warm_duration_seconds` — Histogram
- `mnemo_loader_checksum_failures_total` — Counter

**Control Plane (Go):**
- `mnemo_cp_version_promotes_total` — Counter
- `mnemo_cp_version_rollbacks_total` — Counter
- `mnemo_cp_coverage_ratio{tenant,store,version}` — Gauge
- `mnemo_cp_topology_version` — Gauge
- `mnemo_cp_cas_conflicts_total` — Counter
- `mnemo_cp_registered_pods{tenant,store,shard}` — Gauge

**Client SDK (Go):**
- `mnemo_sdk_request_duration_seconds{op,store}` — Histogram
- `mnemo_sdk_requests_total{op,status}` — Counter
- `mnemo_sdk_fanout_size` — Histogram
- `mnemo_sdk_topology_rebuilds_total` — Counter
- `mnemo_sdk_connection_pool_size{pod}` — Gauge

### SLOs
| SLO | Target |
|---|---|
| Read p99 | < 2 ms |
| Read p999 | < 5 ms |
| Availability | 99.95% |
| Miss-rate | ~0% (after warm) |
| Rollback | < 1 s (V=2) |
| Warm coverage | 100% before promote |

---

## 7. Current State

| Component | Status | Notes |
|---|---|---|
| `mnemo/VERSION` | ✅ done | v0.1.0 |
| `mnemo/go.work` | ✅ done | schema only for now |
| `mnemo/PLAN.md` | ✅ done | this file |
| `mnemo/schema/` | ✅ done | all 5 files, 100% coverage |
| `mnemo/Makefile` | ⬜ pending | Phase 0 |
| `mnemo/README.md` | ⬜ pending | Phase 0 |
| `mnemo/quick-start/` | ⬜ pending | Phase 0 |
| CI workflows | ⬜ pending | Phase 0 |
| `mnemo/controlplane/` | ⬜ pending | Phase 2 |
| `mnemo/dataplane/readserver/` | ⬜ pending | Phase 3 |
| `mnemo/dataplane/dataloader/` | ⬜ pending | Phase 4 |
| `mnemo/sdk/` | ⬜ pending | Phase 5 |
| `mnemo/py-producer/` | ⬜ pending | Phase 6 |
| `mnemo/tools/` | ⬜ pending | Phase 7 |
| Helm charts | ⬜ pending | Phase 8 |
| Docs | ⬜ pending | Phase 9 |

---

## 8. How to Resume After Interruption

1. Read this file top-to-bottom
2. Check Section 7 "Current State" for what's done
3. Find the first ⬜ phase, look at its file table
4. Check which files exist: `ls mnemo/<component>/`
5. Continue from the first missing file in that phase
6. Update Section 7 as each file is completed
7. Run `cd mnemo/schema && go test -v ./...` to confirm Phase 1 is still green

---

## 9. go.work Evolution

Add modules to `mnemo/go.work` as each phase completes:

```
# After Phase 1 (schema) — already done
use ./schema

# After Phase 2 (controlplane)
use ./controlplane

# After Phase 4 (dataloader)
use ./dataplane/dataloader

# After Phase 5 (sdk)
use ./sdk

# After Phase 7 (smoke-test)
use ./tools/smoke-test
```
