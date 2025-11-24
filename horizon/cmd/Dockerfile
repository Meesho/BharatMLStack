# Stage 1: Build the Go binary
FROM golang:1.24-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /app

# Since the build context is ./horizon, helix-client is INSIDE it:
COPY helix-client/ ./helix-client/

# Copy go.mod and go.sum from the context root (./horizon)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire horizon source tree (everything inside ./horizon)
COPY . .

# Build the binary
RUN go build -o horizon ./cmd/horizon/main.go

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/horizon .

EXPOSE 8082
ENTRYPOINT ["/horizon"]
