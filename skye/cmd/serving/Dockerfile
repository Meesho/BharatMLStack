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
RUN go build -o skye-serving ./cmd/serving

# Runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/skye-serving .

EXPOSE 9090

ENTRYPOINT ["/skye-serving"]
