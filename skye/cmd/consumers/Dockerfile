# Build
FROM golang:1.24-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o skye-consumers ./cmd/consumers

# Runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/skye-consumers .

EXPOSE 8080

ENTRYPOINT ["/skye-consumers"]
