# mNemo Control Plane

The control plane manages the full version lifecycle for mNemo stores:
onboarding, publish, promote, rollback, and retire. It is backed by etcd and
exposes a REST API consumed by producers and operators.

## REST API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/tenants/:tenant/stores` | Create a store |
| `GET`  | `/api/v1/tenants/:tenant/stores/:store` | Get store state |
| `POST` | `/api/v1/tenants/:tenant/stores/:store/versions/:vId/publish` | Publish a version (status → READY) |
| `POST` | `/api/v1/tenants/:tenant/stores/:store/versions/:vId/promote` | Promote a version (status → ACTIVE) |
| `POST` | `/api/v1/tenants/:tenant/stores/:store/rollback` | Roll back to previous active version |
| `POST` | `/api/v1/tenants/:tenant/stores/:store/versions/:vId/retire` | Retire a version (status → RETIRING) |
| `GET`  | `/api/v1/tenants/:tenant/stores/:store/topology` | Get active version shard assignment |
| `GET`  | `/api/v1/health` | Health check |

## Configuration

Copy `env.example` to `.env` and adjust:

```bash
cp env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `MNEMO_CP_ADDR` | `:8080` | HTTP listen address |
| `MNEMO_ETCD_ENDPOINTS` | `localhost:2379` | Comma-separated etcd endpoints |

## Development

```bash
# Start local etcd
make -C .. dev-etcd

# Run tests with 100% coverage check
make -C .. coverage-check

# Build binary
go build ./cmd/controlplane
```

## Docker

Build from the `mnemo/` directory (Docker context must include the `schema/` sibling):

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f controlplane/Dockerfile \
  -t ghcr.io/meesho/mnemo-controlplane:v0.1.0 \
  .
```
