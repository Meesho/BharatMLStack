# Stage 1: Build the Go binary
FROM golang:1.24.4-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=1 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

# Install build dependencies for cgo
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        gcc libc6-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build from your main package
RUN go build -o interaction-store-server ./cmd/server/main.go

# Final image using distroless (only works if all assets are statically linked)
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /

COPY --from=builder /app/interaction-store-server .

EXPOSE 8080
ENTRYPOINT ["/interaction-store-server"]
