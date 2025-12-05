FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY predator-dummy/go.mod predator-dummy/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY predator-dummy/cmd/ ./cmd/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o predator-server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates netcat-openbsd

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/predator-server .

EXPOSE 8001

CMD ["./predator-server"]

