# Resource Manager

Resource Manager provides service-oriented APIs for infrastructure operations such as deployable lifecycle management, model load operations, job triggers, and restart flows.

The API server is intentionally synchronous at the request boundary and asynchronous for readiness/state convergence.

## High-Level Architecture

At a high level:

1. Workflow/Client sends an infra intent to API server.
2. API server validates request and executes immediate action (Kubernetes operation, CAS state update in etcd, etc.).
3. API server emits a watch intent to queue/store for async watcher processing.
4. Watcher/consumer path tracks desired condition and triggers callback when complete.

## Request Lifecycle (API Server)

For each mutating API:

1. Request validation + strict payload decode.
2. Idempotency check (`X-Idempotency-Key`).
3. Execute operation:
   - shadow deployable state transitions via etcd CAS
   - k8s-facing operations via executor interface (mock/real adapter)
4. Persist watch intent metadata.
5. Publish watch intent (mock queue adapter today).
6. Return quickly (`200` / `202`) without waiting for readiness.

## etcd Data Model

Shadow deployables are stored as:

`/shadow-deployables/{env}/{deployable_name}`

```json
{
  "name": "int-predator-g2-std-16",
  "node_pool": "g2-std-16",
  "dns": "predator-g2-std-16.meesho.int",
  "state": "FREE | PROCURED",
  "owner": {
    "workflow_run_id": "er5tdgsh",
    "workflow_plan": "pre-scaleup-load-test",
    "procured_at": "2026-01-10T14:20:00Z"
  },
  "last_updated_at": "2026-01-10T14:20:00Z",
  "version": 23
}
```

Secondary key used for in-use index:

`/shadow-deployables-index/{env}/in-use/{deployable_name}`

## CAS Semantics

### Procure

- Preconditions:
  - current record exists
  - `state == FREE`
  - compare-and-swap condition holds on current etcd value/revision
- Transition:
  - `state = PROCURED`
  - set `owner` fields
  - `version = version + 1`
  - set in-use index to `true`

### Release

- Preconditions:
  - current record exists
  - `state == PROCURED`
  - `owner.workflow_run_id == caller`
  - compare-and-swap condition holds on current etcd value/revision
- Transition:
  - `state = FREE`
  - `owner = null`
  - `min_pod_count = 0`
  - `version = version + 1`
  - set in-use index to `false`

## API Surface

- `GET /api/1.0/{env}/shadow-deployables`
- `POST /api/1.0/{env}/shadow-deployables` (`PROCURE` / `RELEASE`)
- `POST /api/1.0/{env}/shadow-deployables/min-pod-count`
- `POST /api/1.0/{env}/deployables`
- `POST /api/1.0/{env}/deployables/restart`
- `POST /api/1.0/{env}/models/load`
- `POST /api/1.0/{env}/jobs`

Backward-compatible alias is supported for typo variant:

- `/api/1.0/{env}/shadow-deploybles`

## Module Structure

- `cmd/api-server`: entrypoint and bootstrap
- `internal/api`: HTTP handlers and contracts
- `internal/application`: orchestration services
- `internal/adapters`: etcd/kubernetes/redis adapters
- `internal/data/models`: shared data models
- `internal/types`: enums and action types
- `internal/errors`: shared business errors
- `internal/ports`: interfaces/ports
- `pkg/*`: shared initialization/util packages (`config`, `logger`, `metric`, `etcd`)

## Runtime Modes

Controlled via env:

- `USE_MOCK_ADAPTERS=true`: in-memory adapters (local/dev)
- `USE_MOCK_ADAPTERS=false`: etcd-backed state/idempotency/operation stores

See `.env.example` for full variable list.

