![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/numerix.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# Numerix

High-performance matrix operations service for BharatML Stack. Provides optimized mathematical computations and matrix operations for machine learning workloads with gRPC APIs.

## Overview

Numerix serves as the computational engine for BharatML Stack, offering:

- **Matrix Operations**: High-performance matrix computations and transformations
- **gRPC API**: Fast binary protocol for efficient data transfer
- **Multi-format Support**: String and byte-based matrix formats
- **Optimized Performance**: Built with Rust for maximum efficiency
- **Scalable Architecture**: Designed for distributed processing

## Features

- **High-Performance Computing**: Built with Rust for optimal performance
- **Matrix Operations**: Comprehensive matrix computation capabilities
- **Protocol Buffers**: Efficient serialization for fast data transfer
- **Multi-format Support**: String and byte-based matrix representations
- **Distributed Ready**: Designed for horizontal scaling
- **Docker Ready**: Containerized for easy deployment
- **Configuration Management**: etcd integration for distributed configuration

## API Endpoints

Numerix provides gRPC APIs for matrix operations:

### gRPC Service
- `Compute(NumerixRequestProto) returns (NumerixResponseProto)` - Matrix computation service

### HTTP Endpoints
- `/health` - Health check endpoint

## Development

### Prerequisites

- Rust 1.80 or later
- Protocol Buffers compiler (protoc)
- Docker (optional, for containerization)
- etcd (for configuration management)

### Getting Started

```bash
# Clone and navigate to the project
cd numerix

# Install dependencies
cargo fetch

# Set up environment variables
cp env.example .env
# Edit .env with your configuration

# Run tests
cargo test

# Build the application
cargo build --release

# Run the service
cargo run
```

### Configuration

Create a `.env` file or set environment variables:

```bash
# APPLICATION CONFIGURATION - Basic app settings
APPLICATION_PORT=8080
APP_ENV=prd
APP_LOG_LEVEL=ERROR
APP_NAME=numerix

# MATRIX OPERATIONS CONFIGURATION - Performance tuning
CHANNEL_BUFFER_SIZE=10000

# ETCD CONFIGURATION - Distributed configuration management
ETCD_SERVERS=http://127.0.0.1:2379

# MONITORING CONFIGURATION - Metrics and observability
METRIC_SAMPLING_RATE=1
TELEGRAF_UDP_HOST=127.0.0.1
TELEGRAF_UDP_PORT=8125
```

### Testing

```bash
# Run all tests
cargo test

# Run tests with output
cargo test -- --nocapture

# Run specific test package
cargo test --package numerix

# Run benchmarks
cargo test --release --benches
```

### Building

```bash
# Clone and navigate to the project
cd numerix

# Build for current platform
cargo build --release

# Build for production
cargo build --release --target x86_64-unknown-linux-gnu

# Cross-compile for different architectures
cargo build --release --target aarch64-unknown-linux-gnu
```

## Docker

### Building the Docker Image

```bash
# Clone and navigate to the project
cd numerix

# Build Docker image
docker build -t numerix -f Dockerfile .

# Run container with environment variables
docker run -p 8080:8080 \
  --env-file .env \
  numerix
```

### Multi-platform Build

```bash
# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 \
  -t numerix:latest \
  --push .
```

## Matrix Operations

Numerix supports various matrix operations and data formats:

### Supported Data Types

- **String Lists**: Text-based matrix representations
- **Byte Arrays**: Binary matrix data for optimal performance

### Example Usage

```rust
// Example gRPC client usage
let client = NumerixClient::connect("http://localhost:8080").await?;

let request = NumerixRequestProto {
    entity_score_data: Some(EntityScoreData {
        schema: vec!["feature1".to_string(), "feature2".to_string()],
        entity_scores: vec![/* matrix data */],
        compute_id: "computation_123".to_string(),
        data_type: "fp32".to_string(),
    }),
};

let response = client.compute(request).await?;
```

## API Documentation

### Health Check
```bash
curl http://localhost:8080/health
```

### Matrix Computation
```bash
# gRPC call example (using grpcurl, run from numerix/ directory)
grpcurl -plaintext \
  -import-path ./src/protos/proto \
  -proto numerix.proto \
  -d '{
    "entityScoreData": {
      "schema": ["feature1", "feature2"],
      "entityScores": [
        { "stringData": { "values": ["1.0", "2.0"] } }
      ],
      "computeId": "test_computation",
      "dataType": "fp32"
    }
  }' \
  localhost:8080 numerix.Numerix/Compute
```

### Metrics
```bash
curl http://localhost:8080/metrics
```

## Performance

Numerix is optimized for high-performance matrix operations:

- **Memory Efficient**: Minimal memory allocations during computation
- **Concurrent Processing**: Multi-threaded matrix operations
- **Zero-Copy**: Efficient data handling without unnecessary copying
- **Vectorized Operations**: SIMD optimizations where available

## Deployment

### Production with Docker

```bash
# Clone and navigate to the project
cd numerix

# Build production image
docker build -t numerix:latest -f Dockerfile .

# Run in production mode
docker run -d \
  --name numerix \
  -p 8080:8080 \
  --env-file .env \
  numerix:latest
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: numerix
spec:
  replicas: 3
  selector:
    matchLabels:
      app: numerix
  template:
    metadata:
      labels:
        app: numerix
    spec:
      containers:
      - name: numerix
        image: numerix:latest
        ports:
        - containerPort: 8080
        env:
        - name: APPLICATION_PORT
          value: "8080"
        - name: APP_ENV
          value: "production"
```

## Integration with BharatML Stack

Numerix integrates seamlessly with other BharatML Stack components:

1. **Online Feature Store**: Provides matrix operations for feature processing
2. **Horizon**: Backend API integration for computational services
3. **Python SDK**: Client libraries for easy integration

## Protocol Buffers

Numerix uses Protocol Buffers for efficient data serialization:

```protobuf
service Numerix {
    rpc Compute(NumerixRequestProto) returns (NumerixResponseProto);
}

message NumerixRequestProto {
    EntityScoreData entity_score_data = 1;
}

message EntityScoreData {
    repeated string schema = 1;
    repeated Score entity_scores = 2;
    string compute_id = 3;
    string data_type = 4;
}
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for your changes
5. Ensure all tests pass (`cargo test`)
6. Update documentation if needed
7. Submit a pull request

## Related Projects

- **[Online Feature Store](../online-feature-store)**: Core feature store service
- **[Horizon](../horizon)**: Backend API service
- **[Python SDK](../py-sdk)**: Python client libraries
- **[TruffleBox UI](../trufflebox-ui)**: Web interface

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>