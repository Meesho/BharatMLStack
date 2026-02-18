# Build
FROM golang:1.24-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=1 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

RUN apt-get update && apt-get install -y --no-install-recommends \
    librdkafka-dev gcc g++ && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o skye-serving ./cmd/serving

# Runtime — needs libc for CGO/librdkafka
FROM gcr.io/distroless/base:nonroot

WORKDIR /

COPY --from=builder /app/skye-serving .
COPY --from=builder /usr/lib/*-linux-gnu/librdkafka* /usr/lib/

EXPOSE 9090

ENTRYPOINT ["/skye-serving"]
