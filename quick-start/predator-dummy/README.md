# Predator Dummy gRPC Server

A dummy gRPC inference server that implements the Triton Inference Server gRPC API for testing purposes.

A dummy gRPC inference server that implements the Triton Inference Server gRPC API for testing purposes.

## Features

- Implements the `GRPCInferenceService` interface from Triton Inference Server
- Generates deterministic model scores (0-1) based on input feature values
- Uses SHA256 hashing to ensure same inputs produce same outputs
- Different inputs produce different outputs
- Returns scores for multiple catalogs

## How It Works

1. Receives `ModelInferRequest` with input tensors containing feature values
2. Hashes all input tensor contents using SHA256
3. Generates deterministic scores (0-1) for each catalog based on the hash
4. Returns `ModelInferResponse` with the generated scores

## Usage

The server is automatically started as part of the quick-start setup when selected.

### Manual Start

```bash
cd quick-start
docker-compose up -d predator
```

### Testing

The server listens on port 8001 (gRPC). You can test it using the predator client from helix-client.

## Configuration

- **Port**: 8001 (configurable via `PORT` environment variable)
- **Default number of catalogs**: 10

## Implementation Details

- Uses SHA256 to hash input features
- Generates scores in range [0, 1) for each catalog
- Same input always produces same output (deterministic)
- Different inputs produce different outputs

