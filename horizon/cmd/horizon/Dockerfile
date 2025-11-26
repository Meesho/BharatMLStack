# Stage 1: Build the Go binary
FROM golang:1.22-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o horizon ./cmd/horizon/main.go

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/horizon .

EXPOSE 8082

ENTRYPOINT ["/horizon"]
//sdkhfksjdhfks
