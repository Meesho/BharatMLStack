# Multi-stage build — context is onyxdb/controlplane (self-contained; model is
# an internal package, no external module replace).
# docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/meesho/onyxdb-controlplane:v0.1.0 .

# ── Build stage ───────────────────────────────────────────────────────────────
FROM mirror.gcr.io/library/golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build/controlplane

# Download dependencies first (better layer caching).
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /controlplane \
    ./cmd/controlplane

# ── Runtime stage — minimal, non-root ─────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /controlplane /controlplane

EXPOSE 8080

ENTRYPOINT ["/controlplane"]
