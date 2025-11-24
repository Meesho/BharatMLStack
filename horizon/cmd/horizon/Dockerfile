# Stage 1: Build the Go binary
FROM golang:1.24-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /app

# Copy helix-client module first (needed for local replace directive)
COPY helix-client/go.mod helix-client/go.sum helix-client/
COPY helix-client/pkg helix-client/pkg/

# Copy horizon module files
COPY horizon/go.mod horizon/go.sum horizon/
WORKDIR /app/horizon
RUN go mod download

# Copy horizon source code
COPY horizon/ .
RUN go build -o horizon ./cmd/horizon/main.go

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/horizon .

EXPOSE 8082

ENTRYPOINT ["/horizon"]