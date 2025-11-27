# Stage 1: Build the Go binary
FROM golang:1.24.4-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o inferflow ./cmd/inferflow/main.go

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/inferflow .

EXPOSE 8085

ENTRYPOINT ["/inferflow"]

