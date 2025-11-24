# Stage 1: Build the Go binary
FROM golang:1.24-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /app

# Copy helix-client (maintains repo structure)
COPY helix-client/ ./helix-client/

# Copy horizon directory (maintains repo structure so replace directive works)
COPY horizon/ ./horizon/

# Set working directory to horizon for the build
WORKDIR /app/horizon

# Download dependencies
RUN go mod download

# Build the binary to a different location to avoid conflict with /app/horizon/ directory
RUN go build -o /app/horizon-binary ./cmd/horizon/main.go

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/horizon-binary /horizon

EXPOSE 8082
ENTRYPOINT ["/horizon"]
